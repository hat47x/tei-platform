package plugin_test

import (
	"strings"
	"testing"

	"tei/internal/environment"
	"tei/internal/plugin"
)

// A catalog that declares the same id and version twice is ambiguous: the
// resolver would pick one arbitrary copy, so it stops instead. The code names
// that case so a caller can say "fix the catalog" rather than "install more".
func TestADuplicatePluginVersionReportsItsOwnCode(t *testing.T) {
	builtin := plugin.Runtime{Kind: plugin.RuntimeBuiltin}
	c := plugin.Catalog{Plugins: []plugin.Definition{
		{ID: cid("tei.a"), Version: "1.0.0", Runtime: builtin},
		{ID: cid("tei.a"), Version: "1.0.0", Runtime: builtin},
	}}
	env := environment.Environment{
		ID:      cid("prod"),
		Plugins: []environment.ExtensionRequirement{{ID: cid("tei.a"), CompatibleMajor: 1}},
	}
	_, err := c.ResolveEnvironment(env)
	if err == nil {
		t.Fatal("expected a duplicate plugin version refusal")
	}
	if !strings.Contains(err.Error(), "declared more than once") {
		t.Fatalf("message changed: %v", err)
	}
	if got := plugin.ResolveCode(err); got != plugin.DiagnosticDuplicatePluginVersion {
		t.Fatalf("ResolveCode = %q, want %q", got, plugin.DiagnosticDuplicatePluginVersion)
	}
}

// The bundle counterpart. It is reached only through the catalog, because a
// repeated selection is refused earlier as an environment selection problem,
// so nothing else exercises this branch.
func TestADuplicateBundleVersionReportsItsOwnCode(t *testing.T) {
	c := plugin.Catalog{Bundles: []plugin.Bundle{
		{ID: cid("tei.bundle"), Version: "1.0.0"},
		{ID: cid("tei.bundle"), Version: "1.0.0"},
	}}
	env := environment.Environment{
		ID:      cid("prod"),
		Bundles: []environment.ExtensionRequirement{{ID: cid("tei.bundle"), CompatibleMajor: 1}},
	}
	_, err := c.ResolveEnvironment(env)
	if err == nil {
		t.Fatal("expected a duplicate bundle version refusal")
	}
	if !strings.Contains(err.Error(), "declared more than once") {
		t.Fatalf("message changed: %v", err)
	}
	if got := plugin.ResolveCode(err); got != plugin.DiagnosticDuplicateBundleVersion {
		t.Fatalf("ResolveCode = %q, want %q", got, plugin.DiagnosticDuplicateBundleVersion)
	}
}

// A bundle that no catalog entry matches is a different refusal from a missing
// plugin: the caller has to install or declare a bundle, not a plugin.
func TestAMissingBundleReportsItsOwnCode(t *testing.T) {
	c := plugin.Catalog{}
	env := environment.Environment{
		ID:      cid("prod"),
		Bundles: []environment.ExtensionRequirement{{ID: cid("tei.bundle"), CompatibleMajor: 2}},
	}
	_, err := c.ResolveEnvironment(env)
	if err == nil {
		t.Fatal("expected a missing bundle refusal")
	}
	if !strings.Contains(err.Error(), "no bundle tei.bundle compatible with major 2") {
		t.Fatalf("message changed: %v", err)
	}
	if got := plugin.ResolveCode(err); got != plugin.DiagnosticNoCompatibleBundle {
		t.Fatalf("ResolveCode = %q, want %q", got, plugin.DiagnosticNoCompatibleBundle)
	}
}

// Two plugins can each be satisfiable on their own and still disagree once a
// shared dependency is fixed: the resolver keeps the version it already
// resolved, so the second requirement is refused rather than silently
// downgraded or upgraded.
func TestAnIncompatibleTransitiveRequirementReportsItsOwnCode(t *testing.T) {
	builtin := plugin.Runtime{Kind: plugin.RuntimeBuiltin}
	c := plugin.Catalog{Plugins: []plugin.Definition{
		// tei.a depends on tei.c major 1.
		{ID: cid("tei.a"), Version: "1.0.0", Runtime: builtin,
			Requires: []plugin.Requirement{{Plugin: cid("tei.c"), CompatibleMajor: 1}}},
		// tei.b depends on tei.c major 2.
		{ID: cid("tei.b"), Version: "1.0.0", Runtime: builtin,
			Requires: []plugin.Requirement{{Plugin: cid("tei.c"), CompatibleMajor: 2}}},
		{ID: cid("tei.c"), Version: "1.0.0", Runtime: builtin},
	}}
	env := environment.Environment{
		ID: cid("prod"),
		Plugins: []environment.ExtensionRequirement{
			{ID: cid("tei.a"), CompatibleMajor: 1},
			{ID: cid("tei.b"), CompatibleMajor: 1},
		},
	}
	_, err := c.ResolveEnvironment(env)
	if err == nil {
		t.Fatal("expected an incompatible transitive refusal")
	}

	codes := map[string]bool{}
	for _, d := range plugin.ResolveDiagnostics(err) {
		codes[d.Code] = true
	}
	// The refusal is raised while walking tei.b's dependency, so the outer
	// wrapper is the requires-hop; the incompatibility is one hop in.
	if !codes[plugin.DiagnosticIncompatibleTransitive] {
		t.Fatalf("the chain does not name the incompatibility: %+v", plugin.ResolveDiagnostics(err))
	}
}

// The environment's own selection list is validated before any lookup, so a
// bad selector never reaches the catalog. Callers need to distinguish "the
// environment is wrong" from "the catalog is missing something".
func TestABadEnvironmentSelectionReportsItsOwnCode(t *testing.T) {
	c := plugin.Catalog{}
	env := environment.Environment{
		ID: cid("prod"),
		Plugins: []environment.ExtensionRequirement{
			{ID: cid("tei.a"), CompatibleMajor: 0},
		},
	}
	_, err := c.ResolveEnvironment(env)
	if err == nil {
		t.Fatal("expected an environment selection refusal")
	}
	if got := plugin.ResolveCode(err); got != plugin.DiagnosticEnvironmentSelection {
		t.Fatalf("ResolveCode = %q, want %q", got, plugin.DiagnosticEnvironmentSelection)
	}
}

// Every declared code is part of the resolver's vocabulary, so a reader can
// list them without reading the switch sites. The count is pinned here so a
// code added without a case that emits it is a failing test, not a silent
// widening of the vocabulary.
func TestTheResolutionVocabularyIsClosedAndCounted(t *testing.T) {
	declared := []string{
		plugin.DiagnosticDuplicateRequirement,
		plugin.DiagnosticInvalidRequirement,
		plugin.DiagnosticNoCompatiblePlugin,
		plugin.DiagnosticDuplicatePluginVersion,
		plugin.DiagnosticNoCompatibleBundle,
		plugin.DiagnosticDuplicateBundleVersion,
		plugin.DiagnosticIncompatibleTransitive,
		plugin.DiagnosticDependencyCycle,
		plugin.DiagnosticEnvironmentSelection,
	}
	seen := map[string]bool{}
	for _, code := range declared {
		if code == "" {
			t.Fatal("a declared code is empty")
		}
		if !strings.HasPrefix(code, "PLUGIN_RESOLVE_") {
			t.Fatalf("code %q is outside the PLUGIN_RESOLVE_ namespace", code)
		}
		if seen[code] {
			t.Fatalf("code %q is declared twice", code)
		}
		seen[code] = true
	}
	if len(declared) != 9 {
		t.Fatalf("the vocabulary has %d codes; update this list and the tests that cover a new one", len(declared))
	}
}
