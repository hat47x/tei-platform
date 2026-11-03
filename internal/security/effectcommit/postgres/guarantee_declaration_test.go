package postgres

import (
	"errors"
	"testing"

	"tei/internal/security/effectcommit"
)

// The postgres Provider deliberately declares NO atomicity guarantee. Whether
// the authority check and the effect are one atomic unit depends on pieces the
// Go code cannot verify (the lock_authority function, database ownership and
// EXECUTE grants, every authority mutation path). Declaring it would be a
// decision for the owner or a conformance attestation, not something this
// package asserts about itself. This test pins the current behaviour: a
// composition that requires AtomicWithCommit refuses this Provider.
func TestPostgresProviderDeclaresNoGuaranteeAndIsRefused(t *testing.T) {
	// The zero value satisfies effectcommit.Provider (Begin has a value receiver)
	// and needs no database to be inspected for a declaration.
	var p effectcommit.Provider = Provider{}
	if g := effectcommit.DeclaredGuarantee(p); g.Level != effectcommit.GuaranteeUndeclared {
		t.Fatalf("the postgres Provider declared %q; declaring atomicity needs an explicit owner decision", g.Level)
	}
	if err := effectcommit.RequireGuarantee(p, effectcommit.GuaranteeAtomicWithCommit); !errors.Is(err, effectcommit.ErrGuaranteeUndeclared) {
		t.Fatalf("RequireGuarantee(postgres, AtomicWithCommit) = %v, want ErrGuaranteeUndeclared", err)
	}
}
