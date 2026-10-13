//go:build tei_sqlite_reference && cgo

package effectfence

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
)

func epochScope() Scope {
	return Scope{Application: "orders", Capability: "save", Subject: "editor", Action: "write", Resource: "record-1"}
}

func seedAuthorityDependencies(t *testing.T, f *sqliteFixture, scope Scope) {
	t.Helper()
	if err := f.SetAuthority(scope, 1, true); err != nil {
		t.Fatal(err)
	}
	for _, dim := range authorityDimensions {
		if err := f.SetAuthorityDependency(scope, dim, 1, true); err != nil {
			t.Fatalf("%s: %v", dim, err)
		}
	}
}

func TestSQLiteDependencyEpochChangedBeforeCommit(t *testing.T) {
	cases := []struct {
		dimension string
		allowed   bool
	}{
		{"subject-link", false},
		{"membership", false},
		{"policy", true},
	}
	for _, tt := range cases {
		t.Run(tt.dimension, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "epochs.sqlite")
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
			s := epochScope()
			seedAuthorityDependencies(t, a, s)
			a.RequireAuthorityDependencies()
			r := Request{Scope: s, OperationID: "epoch-op", Value: "must-not-commit"}
			p, result, err := a.PrepareForTest(context.Background(), r)
			if err != nil || result.Disposition != Prepared {
				t.Fatalf("prepare: %+v %v", result, err)
			}
			if err := b.SetAuthorityDependency(s, tt.dimension, 2, tt.allowed); err != nil {
				t.Fatal(err)
			}
			result, err = a.CommitForTest(context.Background(), p)
			if err != nil || result.Disposition != Denied || result.Reason != "authority-dependency-changed" || result.Receipt != (Receipt{}) {
				t.Fatalf("stale authority dependency committed: %+v %v", result, err)
			}
			if value, version := b.Observe(s); value != "" || version != 0 {
				t.Fatalf("write survived revocation: %q %d", value, version)
			}
			if _, found := b.Lookup(s, r.OperationID); found {
				t.Fatal("receipt created for rejected commit")
			}
		})
	}
}

func TestSQLiteDependencyFenceRequiresEveryDimension(t *testing.T) {
	f, err := openSQLiteFixture(filepath.Join(t.TempDir(), "missing.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	s := epochScope()
	if err := f.SetAuthority(s, 1, true); err != nil {
		t.Fatal(err)
	}
	for _, dim := range []string{"subject-link", "membership"} {
		if err := f.SetAuthorityDependency(s, dim, 1, true); err != nil {
			t.Fatal(err)
		}
	}
	f.RequireAuthorityDependencies()
	req := Request{Scope: s, OperationID: "no-policy", Value: "must-not-commit"}
	_, result, err := f.PrepareForTest(context.Background(), req)
	if err != nil || result.Disposition != Denied || result.Reason != "authority-dependency-unavailable" {
		t.Fatalf("missing epoch admitted: %+v %v", result, err)
	}
	if err := f.SetAuthorityDependency(s, "policy", 1, false); err != nil {
		t.Fatal(err)
	}
	_, result, err = f.PrepareForTest(context.Background(), req)
	if err != nil || result.Disposition != Denied {
		t.Fatalf("revoked policy admitted: %+v %v", result, err)
	}
	if err := f.SetAuthorityDependency(s, "policy", 2, true); err != nil {
		t.Fatal(err)
	}
	p, result, err := f.PrepareForTest(context.Background(), req)
	if err != nil || result.Disposition != Prepared {
		t.Fatalf("restored policy rejected: %+v %v", result, err)
	}
	result, err = f.CommitForTest(context.Background(), p)
	if err != nil || result.Disposition != Committed {
		t.Fatalf("valid epoch fence failed: %+v %v", result, err)
	}
}

func TestSQLiteDependencyVersionsMustAdvance(t *testing.T) {
	f, err := openSQLiteFixture(filepath.Join(t.TempDir(), "monotone.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	s := epochScope()
	if err := f.SetAuthorityDependency(s, "subject-link", 1, true); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		dimension string
		version   uint64
	}{
		{"subject-link", 0},
		{"subject-link", 1},
		{"unknown", 2},
	} {
		if err := f.SetAuthorityDependency(s, tt.dimension, tt.version, false); err == nil {
			t.Fatalf("unsafe revision accepted: %q %d", tt.dimension, tt.version)
		}
	}
	if err := f.SetAuthorityDependency(s, "subject-link", 2, false); err != nil {
		t.Fatal(err)
	}
}

func TestSQLiteDependencyRevocationHidesPendingReplay(t *testing.T) {
	path := filepath.Join(t.TempDir(), "replay.sqlite")
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
	s := epochScope()
	seedAuthorityDependencies(t, a, s)
	a.RequireAuthorityDependencies()
	b.RequireAuthorityDependencies()
	r := Request{Scope: s, OperationID: "replay-epoch", Value: "once"}
	p1, prepared, err := a.PrepareForTest(context.Background(), r)
	if err != nil || prepared.Disposition != Prepared {
		t.Fatalf("prepare first: %+v %v", prepared, err)
	}
	p2, prepared, err := a.PrepareForTest(context.Background(), r)
	if err != nil || prepared.Disposition != Prepared {
		t.Fatalf("prepare second: %+v %v", prepared, err)
	}
	result, err := a.CommitForTest(context.Background(), p1)
	if err != nil || result.Disposition != Committed {
		t.Fatalf("initial commit: %+v %v", result, err)
	}
	if err := b.SetAuthorityDependency(s, "subject-link", 2, false); err != nil {
		t.Fatal(err)
	}
	result, err = a.CommitForTest(context.Background(), p2)
	if err != nil || result.Disposition != Denied || result.Receipt != (Receipt{}) {
		t.Fatalf("revoked replay disclosed receipt: %+v %v", result, err)
	}
	if _, again, err := a.PrepareForTest(context.Background(), r); err != nil || again.Disposition != Denied || again.Receipt != (Receipt{}) {
		t.Fatalf("revoked prepare disclosed receipt: %+v %v", again, err)
	}
	if _, ok := a.Lookup(s, r.OperationID); ok {
		t.Fatal("revoked receipt lookup succeeded")
	}
	if val, ver := b.Observe(s); val != "once" || ver != 1 {
		t.Fatalf("revocation changed already committed data: %q %d", val, ver)
	}
}

func TestSQLiteConcurrentEpochChangeLinearizesWithCommit(t *testing.T) {
	for i := 0; i < 10; i++ {
		path := filepath.Join(t.TempDir(), "concurrent.sqlite")
		a, err := openSQLiteFixture(path)
		if err != nil {
			t.Fatal(err)
		}
		b, err := openSQLiteFixture(path)
		if err != nil {
			a.Close()
			t.Fatal(err)
		}
		s := epochScope()
		seedAuthorityDependencies(t, a, s)
		a.RequireAuthorityDependencies()
		r := Request{Scope: s, OperationID: "linearize", Value: "new-value"}
		p, result, err := a.PrepareForTest(context.Background(), r)
		if err != nil || result.Disposition != Prepared {
			a.Close()
			b.Close()
			t.Fatalf("prepare: %+v %v", result, err)
		}
		begin := make(chan struct{})
		var wg sync.WaitGroup
		var commitResult Result
		var commitErr error
		var revokeErr error
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-begin
			commitResult, commitErr = a.CommitForTest(context.Background(), p)
		}()
		go func() {
			defer wg.Done()
			<-begin
			revokeErr = b.SetAuthorityDependency(s, "subject-link", 2, false)
		}()
		close(begin)
		wg.Wait()
		if commitErr != nil || revokeErr != nil {
			a.Close()
			b.Close()
			t.Fatalf("parallel transaction failed: commit=%v revoke=%v", commitErr, revokeErr)
		}
		_, version := b.Observe(s)
		switch commitResult.Disposition {
		case Committed:
			if version != 1 {
				t.Fatalf("missing authorized commit: %d", version)
			}
		case Denied:
			if version != 0 || commitResult.Reason != "authority-dependency-changed" {
				t.Fatalf("revoked dependency committed: %+v version=%d", commitResult, version)
			}
		default:
			t.Fatalf("unexpected concurrent result: %+v", commitResult)
		}
		a.Close()
		b.Close()
	}
}
