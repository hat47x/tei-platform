//go:build tei_sqlite_reference && cgo

package effectfence

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
)

func TestSQLiteActualTransactionalConformance(t *testing.T) {
	seq := 0
	results := CheckAtomicConformance(context.Background(), func() (ConformanceFixture, error) {
		seq++
		f, err := openSQLiteFixture(filepath.Join(t.TempDir(), fmt.Sprintf("fence%d.sqlite", seq)))
		if err == nil {
			t.Cleanup(f.Close)
		}
		return f, err
	})
	for _, r := range results {
		if !r.Passed {
			t.Errorf("%s: %s", r.Name, r.Detail)
		}
	}
}
func TestSQLiteRevocationAcrossConnectionsAndRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fence.sqlite")
	a, err := openSQLiteFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	b, err := openSQLiteFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	s := Scope{Application: "orders", Capability: "save", Subject: "editor", Action: "write", Resource: "order-1"}
	if err := a.SetAuthority(s, 1, true); err != nil {
		t.Fatal(err)
	}
	req := Request{Scope: s, OperationID: "op-1", Value: "first"}
	tok, res, err := a.PrepareForTest(context.Background(), req)
	if err != nil || res.Disposition != Prepared {
		t.Fatalf("prepare: %+v %v", res, err)
	}
	if err := b.SetAuthority(s, 2, false); err != nil {
		t.Fatal(err)
	}
	result, err := a.CommitForTest(context.Background(), tok)
	if err != nil || result.Disposition != Denied {
		t.Fatalf("revocation crossed transaction boundary: %+v %v", result, err)
	}
	if _, version := b.Observe(s); version != 0 {
		t.Fatal("revocation failed to block effect")
	}
	if err := b.SetAuthority(s, 3, true); err != nil {
		t.Fatal(err)
	}
	tok, res, err = a.PrepareForTest(context.Background(), req)
	if err != nil || res.Disposition != Prepared {
		t.Fatalf("reprepare: %+v %v", res, err)
	}
	result, err = a.CommitForTest(context.Background(), tok)
	if err != nil || result.Disposition != Committed {
		t.Fatalf("commit: %+v %v", result, err)
	}
	a.Close()
	restarted, err := openSQLiteFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	_, again, err := restarted.PrepareForTest(context.Background(), req)
	if err != nil || again.Disposition != Replayed || again.Receipt != result.Receipt {
		t.Fatalf("durable replay: %+v %v", again, err)
	}
	if _, version := restarted.Observe(s); version != 1 {
		t.Fatalf("effect duplicated after restart: %d", version)
	}
}

// 二つの独立接続が同時に同一操作を確定しても、保存は一度だけ成立する。
func TestSQLiteParallelConnectionsSingleEffect(t *testing.T) {
	path := filepath.Join(t.TempDir(), "parallel.sqlite")
	a, err := openSQLiteFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := openSQLiteFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	s := Scope{Application: "orders", Capability: "save", Subject: "editor", Action: "write", Resource: "order-2"}
	if err := a.SetAuthority(s, 1, true); err != nil {
		t.Fatal(err)
	}
	req := Request{Scope: s, OperationID: "same-op", Value: "once"}
	p1, r, err := a.PrepareForTest(context.Background(), req)
	if err != nil || r.Disposition != Prepared {
		t.Fatalf("prepare a: %+v %v", r, err)
	}
	p2, r, err := b.PrepareForTest(context.Background(), req)
	if err != nil || r.Disposition != Prepared {
		t.Fatalf("prepare b: %+v %v", r, err)
	}
	start := make(chan struct{})
	results := make(chan Result, 2)
	errs := make(chan error, 2)
	go func() {
		<-start
		result, err := a.CommitForTest(context.Background(), p1)
		results <- result
		errs <- err
	}()
	go func() {
		<-start
		result, err := b.CommitForTest(context.Background(), p2)
		results <- result
		errs <- err
	}()
	close(start)
	first, second := <-results, <-results
	if err := <-errs; err != nil {
		t.Fatal(err)
	}
	if err := <-errs; err != nil {
		t.Fatal(err)
	}
	if !((first.Disposition == Committed && second.Disposition == Replayed) || (first.Disposition == Replayed && second.Disposition == Committed)) {
		t.Fatalf("not serialized: %+v %+v", first, second)
	}
	if value, version := a.Observe(s); value != "once" || version != 1 {
		t.Fatalf("duplicate effect: %q %d", value, version)
	}
}

// 確定記録の INSERT が失敗した場合、その直前の保存更新も取り消す。
func TestSQLiteRollbackOnReceiptWriteFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rollback.sqlite")
	f, err := openSQLiteFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	s := Scope{Application: "orders", Capability: "save", Subject: "editor", Action: "write", Resource: "order-3"}
	if err := f.SetAuthority(s, 1, true); err != nil {
		t.Fatal(err)
	}
	if err := f.exec("CREATE TRIGGER reject_receipt BEFORE INSERT ON committed BEGIN SELECT RAISE(ABORT, 'injected failure'); END"); err != nil {
		t.Fatal(err)
	}
	req := Request{Scope: s, OperationID: "receipt-error", Value: "should-not-save"}
	p, r, err := f.PrepareForTest(context.Background(), req)
	if err != nil || r.Disposition != Prepared {
		t.Fatalf("prepare: %+v %v", r, err)
	}
	result, err := f.CommitForTest(context.Background(), p)
	if err == nil || result.Disposition == Committed {
		t.Fatalf("false success: %+v %v", result, err)
	}
	if value, version := f.Observe(s); value != "" || version != 0 {
		t.Fatalf("partial effect: %q %d", value, version)
	}
	if _, ok := f.Lookup(s, req.OperationID); ok {
		t.Fatal("receipt persisted despite rollback")
	}
	if err := f.exec("DROP TRIGGER reject_receipt"); err != nil {
		t.Fatal(err)
	}
	p, r, err = f.PrepareForTest(context.Background(), req)
	if err != nil || r.Disposition != Prepared {
		t.Fatalf("retry prepare: %+v %v", r, err)
	}
	result, err = f.CommitForTest(context.Background(), p)
	if err != nil || result.Disposition != Committed {
		t.Fatalf("retry commit: %+v %v", result, err)
	}
}

// 同じ連番を持つ別接続のPermitが他の書込みを確定させないことを確認する。
func TestSQLitePermitBoundToIssuingConnection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "permit-scope.sqlite")
	a, err := openSQLiteFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := openSQLiteFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	s := Scope{Application: "orders", Capability: "save", Subject: "editor", Action: "write", Resource: "order-4"}
	if err := a.SetAuthority(s, 1, true); err != nil {
		t.Fatal(err)
	}
	first := Request{Scope: s, OperationID: "first-connection", Value: "wrong"}
	second := Request{Scope: s, OperationID: "second-connection", Value: "right"}
	fromA, outcome, err := a.PrepareForTest(context.Background(), first)
	if err != nil || outcome.Disposition != Prepared {
		t.Fatalf("prepare A: %+v %v", outcome, err)
	}
	fromB, outcome, err := b.PrepareForTest(context.Background(), second)
	if err != nil || outcome.Disposition != Prepared {
		t.Fatalf("prepare B: %+v %v", outcome, err)
	}
	result, err := b.CommitForTest(context.Background(), fromA)
	if err != nil || result.Disposition != Denied || result.Receipt != (Receipt{}) {
		t.Fatalf("foreign permit used: %+v %v", result, err)
	}
	if _, version := a.Observe(s); version != 0 {
		t.Fatal("foreign permit produced effect")
	}
	result, err = b.CommitForTest(context.Background(), fromB)
	if err != nil || result.Disposition != Committed {
		t.Fatalf("proper permit refused: %+v %v", result, err)
	}
	if value, version := a.Observe(s); value != "right" || version != 1 {
		t.Fatalf("unexpected outcome: %q %d", value, version)
	}
}

// 認可の保存先が参照不能になった場合、拒否と断定せず障害を返し、更新しない。
func TestSQLiteAuthorizationStoreFailureDoesNotCommit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "authority-fault.sqlite")
	f, err := openSQLiteFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	s := Scope{Application: "orders", Capability: "save", Subject: "editor", Action: "write", Resource: "order-5"}
	if err := f.SetAuthority(s, 1, true); err != nil {
		t.Fatal(err)
	}
	req := Request{Scope: s, OperationID: "authority-fault-op", Value: "should-not-save"}
	p, outcome, err := f.PrepareForTest(context.Background(), req)
	if err != nil || outcome.Disposition != Prepared {
		t.Fatalf("prepare: %+v %v", outcome, err)
	}
	if err := f.exec("DROP TABLE authority"); err != nil {
		t.Fatal(err)
	}
	result, err := f.CommitForTest(context.Background(), p)
	if err == nil || result.Disposition != "" {
		t.Fatalf("authority store outage wrongly classified as business denial or success: %+v %v", result, err)
	}
	if value, version := f.Observe(s); value != "" || version != 0 {
		t.Fatalf("effect despite authority failure: %q %d", value, version)
	}
	if err := f.exec("CREATE TABLE authority (scope TEXT PRIMARY KEY, version INTEGER NOT NULL, allowed INTEGER NOT NULL)"); err != nil {
		t.Fatal(err)
	}
	if err := f.SetAuthority(s, 2, true); err != nil {
		t.Fatal(err)
	}
	p, outcome, err = f.PrepareForTest(context.Background(), req)
	if err != nil || outcome.Disposition != Prepared {
		t.Fatalf("retry prepare: %+v %v", outcome, err)
	}
	result, err = f.CommitForTest(context.Background(), p)
	if err != nil || result.Disposition != Committed {
		t.Fatalf("retry after authority recovery: %+v %v", result, err)
	}
}
