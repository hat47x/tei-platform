package validation

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/plugin"
)

// The plugin vocabulary batch (#4094). Each diagnostic must name the values the
// validator accepts, so an author can fix the declaration without reading the
// source.
func TestPluginVocabularyDiagnosticsNameTheirValues(t *testing.T) {
	cases := []struct {
		name   string
		world  ir.World
		code   string
		values []string
	}{
		{
			name: "runtime kind",
			world: ir.World{Plugins: []plugin.Definition{{
				ID: "probe.runtime", Version: "1.0.0",
				Runtime: plugin.Runtime{Kind: plugin.RuntimeKind("gibberish")},
			}}},
			code:   "INVALID_PLUGIN_RUNTIME",
			values: []string{"builtin", "process"},
		},
		{
			name: "sandbox policy",
			world: ir.World{Plugins: []plugin.Definition{{
				ID: "probe.sandbox", Version: "1.0.0",
				Runtime: plugin.Runtime{Kind: plugin.RuntimeProcess, Command: "probe",
					Sandbox: plugin.SandboxPolicy("gibberish")},
			}}},
			code:   "INVALID_PLUGIN_SANDBOX_POLICY",
			values: []string{"none", "preferred", "required"},
		},
		{
			name: "capability kind",
			world: ir.World{Plugins: []plugin.Definition{{
				ID: "probe.capability", Version: "1.0.0",
				Runtime:  plugin.Runtime{Kind: plugin.RuntimeBuiltin},
				Provides: []plugin.Capability{{ID: core.CanonicalID("probe.thing"), Kind: plugin.CapabilityKind("gibberish")}},
			}}},
			code:   "INVALID_PLUGIN_CAPABILITY_KIND",
			values: []string{"deployment-executor", "observation-provider", "evidence-provider", "database-provider", "contract-query-executor", "migration-provider", "editor-intelligence"},
		},
		{
			name: "external scope",
			world: ir.World{Plugins: []plugin.Definition{{
				ID: "probe.scope", Version: "1.0.0",
				Runtime:  plugin.Runtime{Kind: plugin.RuntimeBuiltin},
				External: []plugin.ExternalRequirement{{Scope: "gibberish", Kind: "tool", Name: "thing"}},
			}}},
			code:   "INVALID_PLUGIN_EXTERNAL_SCOPE",
			values: []string{"editor", "build", "test", "runtime", "migration"},
		},
		{
			name: "external kind",
			world: ir.World{Plugins: []plugin.Definition{{
				ID: "probe.extkind", Version: "1.0.0",
				Runtime:  plugin.Runtime{Kind: plugin.RuntimeBuiltin},
				External: []plugin.ExternalRequirement{{Scope: "editor", Kind: "gibberish", Name: "thing"}},
			}}},
			code:   "INVALID_PLUGIN_EXTERNAL_KIND",
			values: []string{"toolchain", "runtime", "language-package", "tool"},
		},
	}
	for _, c := range cases {
		message := diagnosticMessage(t, c.world, c.code)
		for _, value := range c.values {
			if !strings.Contains(message, value) {
				t.Errorf("%s: message does not name %q: %s", c.name, value, message)
			}
		}
		if !strings.Contains(message, "valid values are") {
			t.Errorf("%s: message does not introduce the list: %s", c.name, message)
		}
	}
}

// The empty sandbox policy is accepted, so the message must not imply that only
// the listed values work.
func TestSandboxDiagnosticSaysThePolicyMayBeOmitted(t *testing.T) {
	message := diagnosticMessage(t, ir.World{Plugins: []plugin.Definition{{
		ID: "probe.sandbox2", Version: "1.0.0",
		Runtime: plugin.Runtime{Kind: plugin.RuntimeProcess, Command: "probe",
			Sandbox: plugin.SandboxPolicy("gibberish")},
	}}}, "INVALID_PLUGIN_SANDBOX_POLICY")
	if !strings.Contains(message, "omit") {
		t.Errorf("message does not say the policy may be omitted, although the empty policy is accepted: %s", message)
	}
	if strings.Contains(message, `""`) {
		t.Errorf("message lists the empty value as a choice: %s", message)
	}
}
