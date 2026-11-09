package effectfence

import (
	"context"
	"sync"
	"testing"
)

var scope = Scope{Application: "app", Capability: "save", Subject: "editor", Action: "write", Resource: "record-1"}

func setup(t *testing.T) *MemoryStore {
	t.Helper()
	m := NewMemoryStore()
	if err := m.SetAuthority(scope, 1, true); err != nil {
		t.Fatal(err)
	}
	return m
}

func request(operation string) Request {
	return Request{Scope: scope, OperationID: operation, ExpectedResourceVersion: 0, Value: "更新後"}
}

func mustPrepare(t *testing.T, m *MemoryStore, req Request) Permit {
	t.Helper()
	p, r, err := m.Prepare(context.Background(), req)
	if err != nil || r.Disposition != Prepared {
		t.Fatalf("prepare: %+v %v", r, err)
	}
	return p
}

func TestCommitAndReplayLostResponse(t *testing.T) {
	m := setup(t)
	req := request("op-1")
	p := mustPrepare(t, m, req)
	r, err := m.Commit(context.Background(), p)
	if err != nil || r.Disposition != Committed || r.Receipt.ResourceVersion != 1 {
		t.Fatalf("commit: %+v %v", r, err)
	}
	if v, version := m.Observe(scope); v != "更新後" || version != 1 {
		t.Fatalf("observe %q %d", v, version)
	}
	if receipt, ok := m.Lookup(scope, req.OperationID); !ok || receipt != r.Receipt {
		t.Fatalf("lookup: %+v %v", receipt, ok)
	}
	_, again, err := m.Prepare(context.Background(), req)
	if err != nil || again.Disposition != Replayed || again.Receipt != r.Receipt {
		t.Fatalf("retry: %+v %v", again, err)
	}
	if v, version := m.Observe(scope); v != "更新後" || version != 1 {
		t.Fatalf("replayed produced effect: %q %d", v, version)
	}
	if _, ok := m.Lookup(Scope{Application: "app", Capability: "save", Subject: "viewer", Action: "write", Resource: "record-1"}, req.OperationID); ok {
		t.Fatal("unauthorized status disclosure")
	}
	other := req
	other.Value = "衝突"
	_, conflict, err := m.Prepare(context.Background(), other)
	if err != nil || conflict.Disposition != Denied || conflict.Reason != "operation-id-conflict" {
		t.Fatalf("collision: %+v %v", conflict, err)
	}
}

func TestRevocationBetweenPrepareAndCommit(t *testing.T) {
	m := setup(t)
	p := mustPrepare(t, m, request("op-2"))
	// この境界が、認可成功後・書込み直前の競合を再現する。
	if err := m.SetAuthority(scope, 2, false); err != nil {
		t.Fatal(err)
	}
	r, err := m.Commit(context.Background(), p)
	if err != nil || r.Disposition != Denied || r.Reason != "authority-changed" {
		t.Fatalf("revoked: %+v %v", r, err)
	}
	if value, version := m.Observe(scope); value != "" || version != 0 {
		t.Fatalf("effect after revoke: %q %d", value, version)
	}
	if _, ok := m.Lookup(scope, "op-2"); ok {
		t.Fatal("denied operation stored as committed")
	}
}

func TestResourceCASAndScope(t *testing.T) {
	m := setup(t)
	p := mustPrepare(t, m, request("op-3"))
	if err := m.AdvanceResource(scope, "別の更新"); err != nil {
		t.Fatal(err)
	}
	r, err := m.Commit(context.Background(), p)
	if err != nil || r.Reason != "resource-version-changed" {
		t.Fatalf("resource CAS: %+v %v", r, err)
	}
	if value, version := m.Observe(scope); value != "別の更新" || version != 1 {
		t.Fatalf("stale overwrote: %q %d", value, version)
	}
	wrong := request("op-4")
	wrong.Scope.Subject = "viewer"
	_, r, err = m.Prepare(context.Background(), wrong)
	if err != nil || r.Reason != "authority-unavailable" {
		t.Fatalf("wrong subject: %+v %v", r, err)
	}
	wrong = request("op-5")
	wrong.Scope.Resource = "record-2"
	_, r, err = m.Prepare(context.Background(), wrong)
	if err != nil || r.Reason != "authority-unavailable" {
		t.Fatalf("wrong resource: %+v %v", r, err)
	}
	wrong = request("op-6")
	wrong.Scope.Action = "delete"
	_, r, err = m.Prepare(context.Background(), wrong)
	if err != nil || r.Reason != "authority-unavailable" {
		t.Fatalf("wrong action: %+v %v", r, err)
	}
}

func TestUnknownPermitAndMonotoneEpoch(t *testing.T) {
	m := setup(t)
	if r, err := m.Commit(context.Background(), Permit{}); err != nil || r.Reason != "unknown-permit" {
		t.Fatalf("zero permit: %+v %v", r, err)
	}
	if err := m.SetAuthority(scope, 1, false); err == nil {
		t.Fatal("epoch reuse allowed")
	}
	if err := m.SetAuthority(scope, 0, false); err == nil {
		t.Fatal("zero epoch allowed")
	}
	bad := request("op-7")
	bad.Scope.Application = "a\x00b"
	_, r, err := m.Prepare(context.Background(), bad)
	if err != nil || r.Reason != "invalid-request" {
		t.Fatalf("separator injection: %+v %v", r, err)
	}
	bad = request("op-8")
	bad.OperationID = ""
	_, r, err = m.Prepare(context.Background(), bad)
	if err != nil || r.Reason != "invalid-request" {
		t.Fatalf("missing id: %+v %v", r, err)
	}
}

func TestCanceledBeforeCommitDoesNotChangeState(t *testing.T) {
	m := setup(t)
	p := mustPrepare(t, m, request("op-9"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := m.Commit(ctx, p); err == nil {
		t.Fatal("canceled context succeeded")
	}
	if _, version := m.Observe(scope); version != 0 {
		t.Fatal("canceled operation changed state")
	}
	r, err := m.Commit(context.Background(), p)
	if err != nil || r.Disposition != Committed {
		t.Fatalf("retry after cancel: %+v %v", r, err)
	}
}

func TestConcurrentPermitOnlyOneEffect(t *testing.T) {
	m := setup(t)
	p1 := mustPrepare(t, m, request("same-op"))
	p2 := mustPrepare(t, m, request("same-op"))
	var wg sync.WaitGroup
	results := make(chan Result, 2)
	errs := make(chan error, 2)
	for _, p := range []Permit{p1, p2} {
		wg.Add(1)
		go func(p Permit) {
			defer wg.Done()
			r, err := m.Commit(context.Background(), p)
			results <- r
			errs <- err
		}(p)
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	committed, replayed := 0, 0
	for r := range results {
		if r.Disposition == Committed {
			committed++
		}
		if r.Disposition == Replayed {
			replayed++
		}
	}
	if committed != 1 || replayed != 1 {
		t.Fatalf("outcomes committed=%d replayed=%d", committed, replayed)
	}
	if _, version := m.Observe(scope); version != 1 {
		t.Fatalf("double effect: %d", version)
	}
}

func TestParallelRevocationAndCommitIsSerialized(t *testing.T) {
	for i := 0; i < 100; i++ {
		m := setup(t)
		p := mustPrepare(t, m, request("op-parallel"))
		start := make(chan struct{})
		results := make(chan Result, 1)
		errs := make(chan error, 2)
		go func() { <-start; r, err := m.Commit(context.Background(), p); results <- r; errs <- err }()
		go func() { <-start; errs <- m.SetAuthority(scope, 2, false) }()
		close(start)
		r := <-results
		for j := 0; j < 2; j++ {
			if err := <-errs; err != nil {
				t.Fatal(err)
			}
		}
		_, v := m.Observe(scope)
		if (r.Disposition == Committed && v != 1) || (r.Disposition == Denied && v != 0) {
			t.Fatalf("inconsistent outcome: %+v resource=%d", r, v)
		}
	}
}

// 確定済みの応答を別のPermitから再送する場合も、失効後には開示しない。
func TestRevocationBeforePendingReplay(t *testing.T) {
	m := setup(t)
	r := request("pending-replay")
	p1 := mustPrepare(t, m, r)
	p2 := mustPrepare(t, m, r)
	first, err := m.Commit(context.Background(), p1)
	if err != nil || first.Disposition != Committed {
		t.Fatalf("initial commit: %+v %v", first, err)
	}
	if err := m.SetAuthority(scope, 2, false); err != nil {
		t.Fatal(err)
	}
	second, err := m.Commit(context.Background(), p2)
	if err != nil || second.Disposition != Denied || second.Reason != "authority-changed" || second.Receipt != (Receipt{}) {
		t.Fatalf("revoked replay leaked receipt: %+v %v", second, err)
	}
	if _, retry, err := m.Prepare(context.Background(), r); err != nil || retry.Disposition != Denied || retry.Receipt != (Receipt{}) {
		t.Fatalf("revoked prepare replay: %+v %v", retry, err)
	}
	if _, ok := m.Lookup(scope, r.OperationID); ok {
		t.Fatal("revoked lookup leaked receipt")
	}
	if value, version := m.Observe(scope); value != r.Value || version != 1 {
		t.Fatalf("past committed data unexpectedly mutated: %q %d", value, version)
	}
}

// 別の保存先インスタンスで発行されたPermitは、同じ連番でも実行しない。
func TestPermitCannotCrossStoreBoundary(t *testing.T) {
	first := setup(t)
	second := setup(t)
	original := mustPrepare(t, first, request("first-op"))
	other := mustPrepare(t, second, request("second-op"))
	if original.id != other.id {
		t.Fatal("この試験では連番が衝突する必要があります")
	}
	result, err := second.Commit(context.Background(), original)
	if err != nil || result.Disposition != Denied || result.Receipt != (Receipt{}) {
		t.Fatalf("別の保存先がPermitを受け付けました: %+v %v", result, err)
	}
	if _, version := second.Observe(scope); version != 0 {
		t.Fatal("別の保存先に書込みが発生しました")
	}
	result, err = second.Commit(context.Background(), other)
	if err != nil || result.Disposition != Committed {
		t.Fatalf("正しい保存先のPermitまで破棄されました: %+v %v", result, err)
	}
}
