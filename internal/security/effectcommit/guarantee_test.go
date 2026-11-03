package effectcommit

import (
	"context"
	"errors"
	"testing"
)

type declaring struct {
	Provider
	g Guarantee
}

func (d declaring) Guarantee() Guarantee { return d.g }

// remoteEffectProvider models a remote SaaS/queue effect that cannot couple
// authorization with the effect. It counts Begin calls to prove it is never run.
type remoteEffectProvider struct{ begins int }

func (p *remoteEffectProvider) Begin(context.Context) (Transaction, error) {
	p.begins++
	return nil, errors.New("must not be reached")
}
func (p *remoteEffectProvider) Guarantee() Guarantee {
	return Guarantee{Level: GuaranteeNotAtomic, Reason: "remote-effect: no shared commit with authority"}
}

func TestRemoteEffectNotAtomicIsRefusedAndNeverExecutes(t *testing.T) {
	p := &remoteEffectProvider{}
	runner, err := NewRequiredRunner(p, GuaranteeAtomicWithCommit, AuditConfig{})
	if !errors.Is(err, ErrGuaranteeNotMet) || runner != (Runner{}) {
		t.Fatalf("not refused: %+v %v", runner, err)
	}
	if err := RequireGuarantee(p, GuaranteeAtomicWithCommit); !errors.Is(err, ErrGuaranteeNotMet) {
		t.Fatal(err)
	}
	// The refused result is the zero Runner: Execute cannot reach the Provider.
	req := fixtureRequest()
	if _, err := runExec(runner, req); !errors.Is(err, ErrMissingProvider) {
		t.Fatal(err)
	}
	if p.begins != 0 {
		t.Fatalf("refused Provider reached Begin %d times", p.begins)
	}
	if err := RequireGuarantee(p, GuaranteeNotAtomic); err != nil {
		t.Fatal(err)
	}
}

func TestRequireGuaranteeFailClosed(t *testing.T) {
	plain := newMemoryProvider()
	if g := DeclaredGuarantee(plain); g.Level != GuaranteeUndeclared {
		t.Fatal(g)
	}
	if err := RequireGuarantee(plain, GuaranteeAtomicWithCommit); !errors.Is(err, ErrGuaranteeUndeclared) {
		t.Fatal(err)
	}
	bogus := declaring{plain, Guarantee{Level: "super-atomic"}}
	if err := RequireGuarantee(bogus, GuaranteeAtomicWithCommit); !errors.Is(err, ErrGuaranteeUndeclared) {
		t.Fatal(err)
	}
	if err := RequireGuarantee(nil, GuaranteeAtomicWithCommit); !errors.Is(err, ErrMissingProvider) {
		t.Fatal(err)
	}
	if err := RequireGuarantee(plain, GuaranteeUndeclared); !errors.Is(err, ErrInvalidRequest) {
		t.Fatal(err)
	}
}

func TestAtomicDeclarationAccepted(t *testing.T) {
	p := declaring{newMemoryProvider(), Guarantee{Level: GuaranteeAtomicWithCommit, Reason: "synthetic same-lock memory store"}}
	runner, err := NewRequiredRunner(p, GuaranteeAtomicWithCommit, AuditConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if res, err := runExec(runner, fixtureRequest()); err != nil || res.Status != StatusCommitted {
		t.Fatal(res, err)
	}
}

func TestUndeclaredProviderStillWorksWithoutRequire(t *testing.T) {
	if res, err := runExec(Runner{Provider: newMemoryProvider()}, fixtureRequest()); err != nil || res.Status != StatusCommitted {
		t.Fatal(res, err)
	}
}
