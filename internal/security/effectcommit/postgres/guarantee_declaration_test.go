package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"tei/internal/security/effectcommit"
)

// Provisional delegated decision D10: only a Provider built by
// NewWithTrustedOwnerRoutine declares atomic-with-commit; every other
// construction path declares nothing and is refused by RequireGuarantee. The
// declaration is a claim, not proof; the real PostgreSQL tests that back it are
// listed in README.md. These tests need no database.

func TestTrustedOwnerProviderDeclaresProvisionalAtomicWithCommit(t *testing.T) {
	p, err := NewWithTrustedOwnerRoutine(&pgxpool.Pool{}, "tei_schema")
	if err != nil {
		t.Fatal(err)
	}
	g := effectcommit.DeclaredGuarantee(p)
	if g.Level != effectcommit.GuaranteeAtomicWithCommit || g.Reason != ReasonProvisionalD10 {
		t.Fatalf("declared %+v", g)
	}
	if err := effectcommit.RequireGuarantee(p, effectcommit.GuaranteeAtomicWithCommit); err != nil {
		t.Fatalf("RequireGuarantee(trusted-owner, AtomicWithCommit) = %v", err)
	}
	if err := effectcommit.RequireGuarantee(p, effectcommit.GuaranteeNotAtomic); !errors.Is(err, effectcommit.ErrGuaranteeNotMet) {
		t.Fatalf("RequireGuarantee(trusted-owner, NotAtomic) = %v, want ErrGuaranteeNotMet", err)
	}
	if got := ReasonProvisionalD10; got != "provisional D10: trusted-owner routine path; re-verify against real deployment" {
		t.Fatalf("reason changed: %q", got)
	}
}

func TestOtherPostgresProvidersDeclareNothingAndAreRefused(t *testing.T) {
	direct, err := New(&pgxpool.Pool{}, "tei_schema")
	if err != nil {
		t.Fatal(err)
	}
	for name, p := range map[string]effectcommit.Provider{
		"New (direct objects UPDATE path)": direct,
		"zero value":                       Provider{},
		"hand-built literal":               Provider{Pool: &pgxpool.Pool{}, Schema: "tei_schema"},
	} {
		if g := effectcommit.DeclaredGuarantee(p); g.Level != effectcommit.GuaranteeUndeclared {
			t.Fatalf("%s declared %q; only the trusted-owner routine path may declare (provisional D10)", name, g.Level)
		}
		if err := effectcommit.RequireGuarantee(p, effectcommit.GuaranteeAtomicWithCommit); !errors.Is(err, effectcommit.ErrGuaranteeUndeclared) {
			t.Fatalf("%s: RequireGuarantee = %v, want ErrGuaranteeUndeclared", name, err)
		}
		if _, err := effectcommit.NewRequiredRunner(p, effectcommit.GuaranteeAtomicWithCommit, effectcommit.AuditConfig{}); !errors.Is(err, effectcommit.ErrGuaranteeUndeclared) {
			t.Fatalf("%s: NewRequiredRunner = %v, want ErrGuaranteeUndeclared", name, err)
		}
	}
}

// A wrapper that holds the Provider in a NAMED field hides the declaration
// (fail closed). A wrapper that EMBEDS the concrete Provider would inherit it
// (see the warning on effectcommit.GuaranteeDeclarer); wrappers must not.
type namedFieldWrapper struct{ inner Provider }

func (w namedFieldWrapper) Begin(ctx context.Context) (effectcommit.Transaction, error) {
	return w.inner.Begin(ctx)
}

func TestWrapperDeclarationDoesNotLeakThroughNamedField(t *testing.T) {
	p, err := NewWithTrustedOwnerRoutine(&pgxpool.Pool{}, "tei_schema")
	if err != nil {
		t.Fatal(err)
	}
	if g := effectcommit.DeclaredGuarantee(namedFieldWrapper{inner: p}); g.Level != effectcommit.GuaranteeUndeclared {
		t.Fatalf("named-field wrapper inherited %q", g.Level)
	}
}

// embeddingWrapper EMBEDS the concrete Provider, so Guarantee() is promoted and
// the wrapper is reported as atomic-with-commit even if it adds non-atomic
// behaviour. This pins that known property (see effectcommit.GuaranteeDeclarer):
// it is a composition-review matter, listed in README.md, not something this
// package can prevent.
type embeddingWrapper struct{ Provider }

func TestEmbeddingWrapperInheritsDeclarationKnownLimitation(t *testing.T) {
	p, err := NewWithTrustedOwnerRoutine(&pgxpool.Pool{}, "tei_schema")
	if err != nil {
		t.Fatal(err)
	}
	if g := effectcommit.DeclaredGuarantee(embeddingWrapper{p}); g.Level != effectcommit.GuaranteeAtomicWithCommit {
		t.Fatalf("embedding wrapper = %q; the promoted declaration is the documented limitation", g.Level)
	}
	direct, err := New(&pgxpool.Pool{}, "tei_schema")
	if err != nil {
		t.Fatal(err)
	}
	if g := effectcommit.DeclaredGuarantee(embeddingWrapper{direct}); g.Level != effectcommit.GuaranteeUndeclared {
		t.Fatalf("embedding the direct-path Provider declared %q", g.Level)
	}
}
