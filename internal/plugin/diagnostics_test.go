package plugin_test

import (
	"errors"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/plugin"
)

func TestResolveDiagnosticsCarriesACodeAndTheSameMessage(t *testing.T) {
	c := plugin.Catalog{}
	env := environment.Environment{
		ID:      cid("prod"),
		Plugins: []environment.ExtensionRequirement{{ID: cid("tei.a"), CompatibleMajor: 1}},
	}
	_, err := c.ResolveEnvironment(env)
	if err == nil {
		t.Fatal("expected a resolution refusal")
	}

	// The message is unchanged: a reader that never learned about codes still
	// sees the text it saw before.
	if !strings.Contains(err.Error(), "no plugin tei.a compatible with major 1") {
		t.Fatalf("message changed: %v", err)
	}

	// And the same refusal now also carries a code and a subject.
	diagnostics := plugin.ResolveDiagnostics(err)
	if len(diagnostics) == 0 {
		t.Fatal("no structured diagnostic was attached")
	}
	outermost := diagnostics[0]
	if outermost.Code != plugin.DiagnosticNoCompatiblePlugin {
		t.Fatalf("code = %q", outermost.Code)
	}
	if outermost.Subject != cid("tei.a") {
		t.Fatalf("subject = %q", outermost.Subject)
	}
	// A missing plugin is refused while walking the queue, so its code is
	// outermost. A refusal inside the environment's own selection list is
	// wrapped instead, which is why the chain, not one code, is the contract.
	if plugin.ResolveCode(err) != plugin.DiagnosticNoCompatiblePlugin {
		t.Fatalf("ResolveCode = %q", plugin.ResolveCode(err))
	}
}

// A cycle is reachable only through the resolver, and it is the case most
// likely to be reported as a generic failure today.
func TestACycleReportsTheCycleCode(t *testing.T) {
	c := plugin.Catalog{Plugins: []plugin.Definition{
		{ID: cid("tei.a"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}, Requires: []plugin.Requirement{{Plugin: cid("tei.b"), CompatibleMajor: 1}}},
		{ID: cid("tei.b"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}, Requires: []plugin.Requirement{{Plugin: cid("tei.a"), CompatibleMajor: 1}}},
	}}
	env := environment.Environment{ID: cid("prod"), Plugins: []environment.ExtensionRequirement{{ID: cid("tei.a"), CompatibleMajor: 1}}}
	_, err := c.ResolveEnvironment(env)
	if err == nil {
		t.Fatal("expected a dependency cycle")
	}

	codes := map[string]bool{}
	for _, d := range plugin.ResolveDiagnostics(err) {
		codes[d.Code] = true
	}
	if !codes[plugin.DiagnosticDependencyCycle] {
		t.Fatalf("the chain does not name the cycle: %+v", plugin.ResolveDiagnostics(err))
	}
}

// An error that carries no diagnostic yields no code. A caller must treat that
// as "unknown reason", not as success.
func TestAnOrdinaryErrorHasNoResolveCode(t *testing.T) {
	if got := plugin.ResolveCode(errors.New("something else")); got != "" {
		t.Fatalf("ResolveCode = %q", got)
	}
	if diagnostics := plugin.ResolveDiagnostics(errors.New("something else")); len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %+v", diagnostics)
	}
	if got := plugin.ResolveCode(nil); got != "" {
		t.Fatalf("ResolveCode(nil) = %q", got)
	}
}

// The duplicate-requirement refusal is decided before any lookup, so it must
// report itself rather than the generic "no compatible plugin".
func TestADuplicateRequirementReportsItsOwnCode(t *testing.T) {
	c := plugin.Catalog{}
	env := environment.Environment{
		ID: cid("prod"),
		Plugins: []environment.ExtensionRequirement{
			{ID: cid("tei.a"), CompatibleMajor: 1},
			{ID: cid("tei.a"), CompatibleMajor: 1},
		},
	}
	_, err := c.ResolveEnvironment(env)
	if err == nil {
		t.Fatal("expected a duplicate refusal")
	}
	if !strings.Contains(err.Error(), "appears more than once") {
		t.Fatalf("message changed: %v", err)
	}
	// The outermost code names the stage that stopped (the environment's own
	// selection list); the specific reason is one hop in.
	codes := map[string]bool{}
	for _, d := range plugin.ResolveDiagnostics(err) {
		codes[d.Code] = true
	}
	if !codes[plugin.DiagnosticDuplicateRequirement] {
		t.Fatalf("the chain does not name the duplicate: %+v", plugin.ResolveDiagnostics(err))
	}
}

// An invalid requirement value is a different refusal from a missing plugin,
// and callers need to tell them apart.
func TestAnInvalidRequirementReportsItsOwnCode(t *testing.T) {
	c := plugin.Catalog{}
	env := environment.Environment{
		ID:      cid("prod"),
		Plugins: []environment.ExtensionRequirement{{ID: cid("tei.a"), CompatibleMajor: 0}},
	}
	_, err := c.ResolveEnvironment(env)
	if err == nil {
		t.Fatal("expected an invalid requirement refusal")
	}
	codes := map[string]bool{}
	for _, d := range plugin.ResolveDiagnostics(err) {
		codes[d.Code] = true
	}
	if !codes[plugin.DiagnosticInvalidRequirement] {
		t.Fatalf("the chain does not name the invalid requirement: %+v", plugin.ResolveDiagnostics(err))
	}
}

var _ = core.CanonicalID("")
