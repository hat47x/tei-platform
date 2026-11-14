package main

import (
	"encoding/json"
	"strings"
	"testing"

	"tei/internal/plugin"
)

type errText string

func (e errText) Error() string { return string(e) }

// The CLI reported every resolver failure as one code plus prose. The
// resolver's own reason is now carried alongside, so a caller can tell a cycle
// from a missing plugin without reading the message.
func TestPluginResolveFailureCarriesTheResolverReason(t *testing.T) {
	cause := error(&plugin.ResolveDiagnostic{
		Code:    plugin.DiagnosticDependencyCycle,
		Subject: "tei.a",
		Message: "plugin dependency cycle at tei.a",
	})

	failure := pluginResolveFailure(resultCodePluginResolveFailed, cause)
	if failure.Code != resultCodePluginResolveFailed {
		t.Fatalf("code = %q; the CLI code must not change", failure.Code)
	}
	if failure.ResolveReason != plugin.DiagnosticDependencyCycle {
		t.Fatalf("resolveReason = %q", failure.ResolveReason)
	}
	// The message is unchanged: readers that only match text keep working.
	if failure.Message != "plugin dependency cycle at tei.a" {
		t.Fatalf("message = %q", failure.Message)
	}
}

// An error the resolver did not classify yields no reason. A caller must read
// that as "unknown", not as a specific refusal.
func TestAnOrdinaryErrorCarriesNoResolveReason(t *testing.T) {
	failure := pluginResolveFailure(resultCodePluginResolveFailed, errText("something else"))
	if failure.ResolveReason != "" {
		t.Fatalf("resolveReason = %q", failure.ResolveReason)
	}
}

// The field is omitted from JSON when empty, so a reader expecting exactly code
// and message sees the same object as before.
func TestTheReasonIsOmittedFromJSONWhenAbsent(t *testing.T) {
	raw, err := json.Marshal(pluginResolveFailure(resultCodePluginResolveFailed, errText("something else")))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "resolveReason") {
		t.Fatalf("empty reason leaked into JSON: %s", raw)
	}

	withReason := pluginResolveFailure(resultCodePluginResolveFailed, error(&plugin.ResolveDiagnostic{
		Code: plugin.DiagnosticNoCompatiblePlugin, Message: "no plugin tei.a compatible with major 1",
	}))
	raw, err = json.Marshal(withReason)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"resolveReason":"PLUGIN_RESOLVE_NO_COMPATIBLE_PLUGIN"`) {
		t.Fatalf("reason missing from JSON: %s", raw)
	}
}
