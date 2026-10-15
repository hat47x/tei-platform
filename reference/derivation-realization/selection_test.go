package derivationrealization

import (
	"errors"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/plugin"
	"tei/internal/semantic/derivationref"
	languagereference "tei/reference/dsl-language-plugin"
)

const pinnedArtifact = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func fixture(t *testing.T) (HostSelection, Invocation, GoRealization) {
	t.Helper()
	id := core.MustCanonicalID("project.line_amount")
	def := plugin.Definition{
		ID: id, Version: "1.0.0",
		Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
		Artifact: plugin.Artifact{Digest: pinnedArtifact},
	}
	env := environment.Environment{
		ID: core.MustCanonicalID("project.test"),
		Plugins: []environment.ExtensionRequirement{{ID: id, CompatibleMajor: 1}},
	}
	catalog := plugin.Catalog{Plugins: []plugin.Definition{def}}
	lock, err := plugin.BuildLock(catalog, []environment.Environment{env})
	if err != nil {
		t.Fatal(err)
	}
	operator := languagereference.OperatorContract{
		ID: "multiply", Version: "1.0.0", Fingerprint: strings.Repeat("b", 64),
		Effect: "pure", InputType: "decimal,decimal", OutputType: "decimal-or-null",
	}
	invocation := Invocation{
		Rule: derivationref.Definition{
			ID: "line-amount", Inputs: []string{"line.quantity", "line.unit-price"},
			Operator: "multiply", Pure: true,
		},
		OperatorPin: operator, PluginPin: plugin.Identity{ID: id, Version: "1.0.0"},
		ArtifactDigest: pinnedArtifact, Target: "go/reference",
	}
	realization := GoRealization{
		Plugin: invocation.PluginPin, Operator: operator, Target: "go/reference",
		Run: GoMultiply,
	}
	return HostSelection{
		Catalog: catalog, Environment: env, Lock: lock,
		OperatorCatalog: []languagereference.OperatorContract{operator},
	}, invocation, realization
}

func TestSelectedGoCalculationPreservesD1OracleAndInputStates(t *testing.T) {
	host, in, implementation := fixture(t)
	cases := []struct {
		name string
		a, b derivationref.Input
		state derivationref.InputState
		value string
		wantErr error
	}{
		{"2x300", derivationref.Input{State: derivationref.Present, Decimal: "2"}, derivationref.Input{State: derivationref.Present, Decimal: "300"}, derivationref.Present, "600", nil},
		{"zero", derivationref.Input{State: derivationref.Present, Decimal: "0"}, derivationref.Input{State: derivationref.Present, Decimal: "300"}, derivationref.Present, "0", nil},
		{"exact decimal", derivationref.Input{State: derivationref.Present, Decimal: "0.2"}, derivationref.Input{State: derivationref.Present, Decimal: "0.5"}, derivationref.Present, "0.1", nil},
		{"NULL propagated", derivationref.Input{State: derivationref.Null}, derivationref.Input{State: derivationref.Present, Decimal: "300"}, derivationref.Null, "", nil},
		{"missing is not NULL", derivationref.Input{State: derivationref.Missing}, derivationref.Input{State: derivationref.Present, Decimal: "300"}, "", "", derivationref.ErrInput},
		{"absent relation", derivationref.Input{State: derivationref.AbsentRelation}, derivationref.Input{State: derivationref.Present, Decimal: "300"}, "", "", derivationref.ErrInput},
		{"multiple relation", derivationref.Input{State: derivationref.MultipleRelations}, derivationref.Input{State: derivationref.Present, Decimal: "300"}, "", "", derivationref.ErrInput},
		{"invalid decimal", derivationref.Input{State: derivationref.Present, Decimal: "not-a-number"}, derivationref.Input{State: derivationref.Present, Decimal: "300"}, "", "", derivationref.ErrOperator},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := host.Evaluate(in, implementation, tc.a, tc.b)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error %v, expected %v", err, tc.wantErr)
			}
			if tc.wantErr == nil && (got.State != tc.state || got.Decimal != tc.value) {
				t.Fatalf("got %#v, want state %q decimal %q", got, tc.state, tc.value)
			}
		})
	}
}

func TestRuleIdentitySurvivesImplementationSelection(t *testing.T) {
	host, in, implementation := fixture(t)
	before := in.Rule.ID
	// The existing plugin resolver may see a newer compatible release,
	// but the Host's earlier environment lock still selects exactly v1.0.0.
	newer := host.Catalog.Plugins[0]
	newer.Version = "1.1.0"
	host.Catalog.Plugins = append(host.Catalog.Plugins, newer)
	result, err := host.Evaluate(in, implementation, derivationref.Input{State: derivationref.Present, Decimal: "2"}, derivationref.Input{State: derivationref.Present, Decimal: "300"})
	if err != nil || result.Decimal != "600" || in.Rule.ID != before {
		t.Fatalf("locked implementation changed rule meaning or identity: %#v err=%v", result, err)
	}
}

func TestSelectionRejectsMismatchBeforeCallingImplementation(t *testing.T) {
	cases := []struct {
		name string
		edit func(*HostSelection, *Invocation, *GoRealization)
	}{
		{"target unsupported", func(_ *HostSelection, i *Invocation, _ *GoRealization) { i.Target = "sql/postgres" }},
		{"unregistered version", func(_ *HostSelection, i *Invocation, _ *GoRealization) { i.PluginPin.Version = "1.2.0" }},
		{"unlocked plugin", func(_ *HostSelection, i *Invocation, _ *GoRealization) { i.PluginPin.ID = core.MustCanonicalID("project.not_selected") }},
		{"artifact mismatch", func(_ *HostSelection, i *Invocation, _ *GoRealization) { i.ArtifactDigest = "sha256:" + strings.Repeat("c", 64) }},
		{"definition mutated after locking", func(h *HostSelection, _ *Invocation, _ *GoRealization) { h.Catalog.Plugins[0].Artifact.Digest = "sha256:" + strings.Repeat("c", 64) }},
		{"environment not locked", func(h *HostSelection, _ *Invocation, _ *GoRealization) { h.Environment.ID = core.MustCanonicalID("project.other") }},
		{"missing pinned plugin", func(h *HostSelection, _ *Invocation, _ *GoRealization) { h.Catalog.Plugins = nil }},
		{"duplicate exact plugin", func(h *HostSelection, _ *Invocation, _ *GoRealization) { h.Catalog.Plugins = append(h.Catalog.Plugins, h.Catalog.Plugins[0]) }},
		{"impure rule", func(_ *HostSelection, i *Invocation, _ *GoRealization) { i.Rule.Pure = false }},
		{"unknown rule operator", func(_ *HostSelection, i *Invocation, _ *GoRealization) { i.Rule.Operator = "unknown" }},
		{"duplicate rule input", func(_ *HostSelection, i *Invocation, _ *GoRealization) { i.Rule.Inputs[1] = i.Rule.Inputs[0] }},
		{"unsupported nullable contract", func(_ *HostSelection, i *Invocation, _ *GoRealization) { i.OperatorPin.OutputType = "decimal-nonnull" }},
		{"operator effect escalation", func(_ *HostSelection, i *Invocation, _ *GoRealization) { i.OperatorPin.Effect = "writes" }},
		{"operator missing from catalog", func(h *HostSelection, _ *Invocation, _ *GoRealization) { h.OperatorCatalog = nil }},
		{"operator ambiguous", func(h *HostSelection, _ *Invocation, _ *GoRealization) { h.OperatorCatalog = append(h.OperatorCatalog, h.OperatorCatalog[0]) }},
		{"operator revision differs", func(_ *HostSelection, i *Invocation, _ *GoRealization) { i.OperatorPin.Version = "1.0.1" }},
		{"realization mismatched plugin", func(_ *HostSelection, _ *Invocation, r *GoRealization) { r.Plugin.Version = "2.0.0" }},
		{"realization mismatched operator", func(_ *HostSelection, _ *Invocation, r *GoRealization) { r.Operator.Fingerprint = strings.Repeat("c", 64) }},
		{"realization mismatched target", func(_ *HostSelection, _ *Invocation, r *GoRealization) { r.Target = "sql/postgres" }},
		{"realization unavailable", func(_ *HostSelection, _ *Invocation, r *GoRealization) { r.Run = nil }},
		{"process runtime is not this builtin realization", func(h *HostSelection, _ *Invocation, _ *GoRealization) { h.Catalog.Plugins[0].Runtime = plugin.Runtime{Kind: plugin.RuntimeProcess, Command: "unknown"} }},
		{"pinned artifact absent", func(_ *HostSelection, i *Invocation, _ *GoRealization) { i.ArtifactDigest = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			host, invocation, implementation := fixture(t)
			called := false
			implementation.Run = func(a, b derivationref.Input) (derivationref.Input, error) {
				called = true
				return GoMultiply(a, b)
			}
			tc.edit(&host, &invocation, &implementation)
			_, err := host.Evaluate(invocation, implementation,
				derivationref.Input{State: derivationref.Present, Decimal: "2"},
				derivationref.Input{State: derivationref.Present, Decimal: "300"})
			if !errors.Is(err, ErrUnproven) || called {
				t.Fatalf("unproven realization was invoked or silently accepted: called=%v err=%v", called, err)
			}
		})
	}
}

func TestNoImplicitFallbackToAvailableGoImplementation(t *testing.T) {
	host, invocation, implementation := fixture(t)
	other := implementation
	other.Plugin.Version = "1.1.0"
	// Even though a working Go implementation is available at v1, requesting
	// the wrong one is not a reason to run v1 by "best effort".
	invocation.PluginPin = other.Plugin
	_, err := host.Evaluate(invocation, implementation,
		derivationref.Input{State: derivationref.Present, Decimal: "2"},
		derivationref.Input{State: derivationref.Present, Decimal: "300"})
	if !errors.Is(err, ErrUnproven) {
		t.Fatalf("unlocked version silently fell back to v1: %v", err)
	}
}
