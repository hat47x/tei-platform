package effectfence

import (
	"context"
	"strings"
	"testing"
)

func TestAtomicConformanceMemoryStore(t *testing.T) {
	results := CheckAtomicConformance(context.Background(), func() (ConformanceFixture, error) {
		return NewMemoryStore(), nil
	})
	if len(results) != 6 {
		t.Fatalf("expected six checks, got %+v", results)
	}
	for _, result := range results {
		if !result.Passed {
			t.Errorf("%s: %s", result.Name, result.Detail)
		}
	}
}

// 認可版の変更を無視する不正なAdapterが、機械的に検出されることを確認する。
type staleAuthorityFixture struct{ *MemoryStore }

func (f *staleAuthorityFixture) SetAuthority(s Scope, version uint64, allowed bool) error {
	if version > 1 {
		return nil // 故意に版変更と失効を無視
	}
	return f.MemoryStore.SetAuthority(s, version, allowed)
}

func TestAtomicConformanceRejectsStaleAuthority(t *testing.T) {
	results := CheckAtomicConformance(context.Background(), func() (ConformanceFixture, error) {
		return &staleAuthorityFixture{MemoryStore: NewMemoryStore()}, nil
	})
	fails := map[string]bool{}
	for _, result := range results {
		if !result.Passed {
			fails[result.Name] = true
		}
	}
	if !fails["revoked-before-commit"] || !fails["epoch-changed-before-commit"] {
		t.Fatalf("did not detect stale authority: %+v", results)
	}
	if fails["commit-observe-replay"] {
		t.Fatalf("unrelated normal case failed: %+v", results)
	}
}

func TestAtomicConformanceRequiresIsolatedFixture(t *testing.T) {
	shared := NewMemoryStore()
	if err := shared.AdvanceResource(Scope{Application: "conformance-app", Capability: "save", Subject: "editor", Action: "write", Resource: "record-1"}, "汚染"); err != nil {
		t.Fatal(err)
	}
	results := CheckAtomicConformance(context.Background(), func() (ConformanceFixture, error) {
		return shared, nil
	})
	for _, r := range results {
		if r.Passed || !strings.Contains(r.Detail, "隔離された空") {
			t.Fatalf("dirty fixture accepted: %+v", r)
		}
	}
}

func TestAtomicConformanceMissingFixtureFailsClosed(t *testing.T) {
	results := CheckAtomicConformance(context.Background(), nil)
	for _, r := range results {
		if r.Passed || r.Detail == "" {
			t.Fatalf("missing fixture accepted: %+v", r)
		}
	}
}

func TestScopeValidationAllFieldsAndSequenceOverflow(t *testing.T) {
	m := NewMemoryStore()
	fields := []func(*Scope){
		func(s *Scope) { s.Application = "a\x00b" },
		func(s *Scope) { s.Capability = "c\x00d" },
		func(s *Scope) { s.Subject = "a\nb" },
		func(s *Scope) { s.Action = " a\r\n" },
		func(s *Scope) { s.Resource = "  " },
	}
	for i, change := range fields {
		s := scope
		change(&s)
		if err := m.SetAuthority(s, 1, true); err == nil {
			t.Errorf("invalid scope %d accepted", i)
		}
	}
	if err := m.SetAuthority(scope, 1, true); err != nil {
		t.Fatal(err)
	}
	r := request("op\ninvalid")
	if _, outcome, err := m.Prepare(context.Background(), r); err != nil || outcome.Disposition != Denied {
		t.Fatalf("invalid operation accepted: %+v %v", outcome, err)
	}
	m.sequence = ^uint64(0)
	if _, outcome, err := m.Prepare(context.Background(), request("op-10")); err == nil || outcome.Disposition != "" {
		t.Fatalf("permit sequence wrapped: %+v %v", outcome, err)
	}
}
