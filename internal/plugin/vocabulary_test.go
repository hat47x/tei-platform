package plugin

import "testing"

// Each list is quoted by a validation diagnostic, so the list and the predicate
// that decides acceptance must be one answer (#4094).
func TestPluginVocabularyListsMatchTheirPredicates(t *testing.T) {
	capabilityKinds := []CapabilityKind{
		CapabilityDeploymentExecutor, CapabilityObservationProvider, CapabilityEvidenceProvider,
		CapabilityDatabaseProvider, CapabilityContractQueryExecutor, CapabilityMigrationProvider,
		CapabilityEditorIntelligence,
	}
	if len(capabilityKinds) != len(CapabilityKinds) {
		t.Fatalf("CapabilityKinds has %d entries but %d are declared", len(CapabilityKinds), len(capabilityKinds))
	}
	for _, v := range CapabilityKinds {
		if !v.Valid() {
			t.Errorf("CapabilityKinds lists %q but Valid() rejects it", v)
		}
	}
	for _, bad := range []CapabilityKind{"", "Deployment-Executor", "database"} {
		if bad.Valid() {
			t.Errorf("Valid() accepted capability kind %q", bad)
		}
	}

	runtimeKinds := []RuntimeKind{RuntimeBuiltin, RuntimeProcess}
	if len(runtimeKinds) != len(RuntimeKinds) {
		t.Fatalf("RuntimeKinds has %d entries but %d are declared", len(RuntimeKinds), len(runtimeKinds))
	}
	for _, v := range RuntimeKinds {
		if !v.Valid() {
			t.Errorf("RuntimeKinds lists %q but Valid() rejects it", v)
		}
	}
	for _, bad := range []RuntimeKind{"", "Process", "wasm"} {
		if bad.Valid() {
			t.Errorf("Valid() accepted runtime kind %q", bad)
		}
	}

	sandboxPolicies := []SandboxPolicy{SandboxNone, SandboxPreferred, SandboxRequired}
	if len(sandboxPolicies) != len(SandboxPolicies) {
		t.Fatalf("SandboxPolicies has %d entries but %d are declared", len(SandboxPolicies), len(sandboxPolicies))
	}
	for _, v := range SandboxPolicies {
		if !v.Valid() {
			t.Errorf("SandboxPolicies lists %q but Valid() rejects it", v)
		}
	}
	// The empty policy means "not declared" and is accepted, but it must not be
	// listed: a diagnostic should not tell an author to write an empty value.
	if !SandboxPolicy("").Valid() {
		t.Error("the empty sandbox policy must stay valid")
	}
	for _, v := range SandboxPolicies {
		if v == "" {
			t.Error("the empty sandbox policy must not appear in SandboxPolicies")
		}
	}
	for _, bad := range []SandboxPolicy{"Required", "strict"} {
		if bad.Valid() {
			t.Errorf("Valid() accepted sandbox policy %q", bad)
		}
	}
}

func TestExternalVocabularyListsMatchTheirPredicates(t *testing.T) {
	for _, scope := range ExternalScopes {
		if !(ExternalRequirement{Scope: scope}).ValidScope() {
			t.Errorf("ExternalScopes lists %q but ValidScope() rejects it", scope)
		}
	}
	for _, bad := range []string{"", "Editor", "deploy"} {
		if (ExternalRequirement{Scope: bad}).ValidScope() {
			t.Errorf("ValidScope() accepted %q", bad)
		}
	}
	for _, kind := range ExternalKinds {
		if !(ExternalRequirement{Kind: kind}).ValidKind() {
			t.Errorf("ExternalKinds lists %q but ValidKind() rejects it", kind)
		}
	}
	for _, bad := range []string{"", "Tool", "library"} {
		if (ExternalRequirement{Kind: bad}).ValidKind() {
			t.Errorf("ValidKind() accepted %q", bad)
		}
	}
}
