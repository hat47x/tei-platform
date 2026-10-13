package composition

import (
	"errors"
	"strings"
	"testing"

	"tei/core/effect"
	"tei/core/generation"
)

func TestGenerationCoreAliasesPreserveTypesAndErrors(t *testing.T) {
	var old Binding = generation.Binding{ContractID: "service", ContractVersion: "1", ProviderID: "provider", ArtifactDigest: "sha256:" + strings.Repeat("a", 64), Generation: 1}
	var canonical generation.Binding = old
	if !canonical.Valid() {
		t.Fatal("binding should be structurally valid")
	}
	var originalScope *Scope = effect.NewScope()
	var canonicalScope *effect.Scope = originalScope
	r, err := NewRouter(old, canonicalScope)
	if err != nil {
		t.Fatal(err)
	}
	var canonicalRouter *generation.Router = r
	var originalRouter *Router = canonicalRouter
	lease, err := originalRouter.Acquire(func(Binding) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	var canonicalLease *generation.Lease = lease
	canonicalLease.Release()
	if !errors.Is(ErrUnbound, generation.ErrUnbound) || !errors.Is(ErrGuardRequired, generation.ErrGuardRequired) || !errors.Is(ErrStaleGeneration, generation.ErrStaleGeneration) {
		t.Fatal("error identities changed")
	}
	if err := originalScope.Close(); err != nil {
		t.Fatal(err)
	}
}
