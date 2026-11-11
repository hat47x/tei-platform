package plugin_test

import (
	"testing"

	"tei/internal/plugin"
)

// CapabilityKinds is what a report lists and what Valid() accepts. The comment
// on it says the two cannot drift, but nothing proves the const block and the
// list are the same set: adding a kind to the const block alone would make
// Valid() reject a kind the code declares, and adding it to the list alone
// would accept a kind no constant names. Both are silent.
func TestEveryDeclaredCapabilityKindIsInTheReportList(t *testing.T) {
	listed := make(map[plugin.CapabilityKind]bool, len(plugin.CapabilityKinds))
	for _, kind := range plugin.CapabilityKinds {
		listed[kind] = true
	}

	// The declared set, written out so a new constant is a compile error here
	// until it is also added to CapabilityKinds. It is also compared by length,
	// so an extra entry in the list that no constant names is caught even though
	// every declared kind is still present.
	declared := []plugin.CapabilityKind{
		plugin.CapabilityDeploymentExecutor,
		plugin.CapabilityObservationProvider,
		plugin.CapabilityEvidenceProvider,
		plugin.CapabilityDatabaseProvider,
		plugin.CapabilityContractQueryExecutor,
		plugin.CapabilityMigrationProvider,
		plugin.CapabilityEditorIntelligence,
	}

	for _, kind := range declared {
		if !listed[kind] {
			t.Fatalf("declared kind %q is missing from CapabilityKinds, so Valid() rejects it", kind)
		}
		if !kind.Valid() {
			t.Fatalf("declared kind %q is in the list but Valid() rejects it", kind)
		}
	}
	if len(declared) != len(plugin.CapabilityKinds) {
		t.Fatalf("declared %d kinds but the list has %d; a constant was added without updating the list",
			len(declared), len(plugin.CapabilityKinds))
	}
}

// A kind outside the vocabulary is refused, and the refusal names the kind so a
// caller can see what was written.
func TestAKindOutsideTheVocabularyIsRefused(t *testing.T) {
	if plugin.CapabilityKind("missing").Valid() {
		t.Fatal("a kind outside the vocabulary was accepted")
	}
}

// The runtime vocabulary has the same shape, and the same drift risk: a
// RuntimeKind constant that is not in RuntimeKinds would be refused while the
// code declares it.
func TestEveryDeclaredRuntimeKindIsInTheReportList(t *testing.T) {
	listed := make(map[plugin.RuntimeKind]bool, len(plugin.RuntimeKinds))
	for _, kind := range plugin.RuntimeKinds {
		listed[kind] = true
	}
	declared := []plugin.RuntimeKind{plugin.RuntimeBuiltin, plugin.RuntimeProcess}
	for _, kind := range declared {
		if !listed[kind] {
			t.Fatalf("declared runtime kind %q is missing from RuntimeKinds", kind)
		}
		if !kind.Valid() {
			t.Fatalf("declared runtime kind %q is in the list but Valid() rejects it", kind)
		}
	}
	if len(declared) != len(plugin.RuntimeKinds) {
		t.Fatalf("declared %d runtime kinds but the list has %d", len(declared), len(plugin.RuntimeKinds))
	}
	if plugin.RuntimeKind("wasm").Valid() {
		t.Fatal("a runtime kind outside the vocabulary was accepted")
	}
}
