package syntax_test

import (
	"testing"

	"tei/internal/plugin"
	"tei/internal/plugin/implementation"
	"tei/internal/syntax"
)

func TestParsePluginImplementationStaticResponse(t *testing.T) {
	src := `
plugin-implementation tei.native.runtime.reference 1.0.0 {
    profile static-response-v1
    capability deployment-executor runtime.native_reference
    operation execute json {
        {
            "attemptId": "tei-native-reference",
            "outcome": "accepted"
        }
    }
}
`
	got, err := syntax.NewParser(src).ParsePluginImplementation()
	if err != nil {
		t.Fatal(err)
	}
	if got.Plugin.ID != "tei.native.runtime.reference" || got.Plugin.Version != "1.0.0" {
		t.Fatalf("plugin: %#v", got.Plugin)
	}
	if got.Profile != implementation.StaticResponseProfileV1 {
		t.Fatalf("profile: %s", got.Profile)
	}
	if got.CapabilityKind != plugin.CapabilityDeploymentExecutor || got.CapabilityID != "runtime.native_reference" {
		t.Fatalf("capability: %s %s", got.CapabilityKind, got.CapabilityID)
	}
	if len(got.Operations) != 1 || got.Operations[0].Name != "execute" {
		t.Fatalf("operations: %#v", got.Operations)
	}
}

func TestParsePluginImplementationRejectsDuplicateProfile(t *testing.T) {
	src := `
plugin-implementation tei.native.runtime.reference 1.0.0 {
    profile static-response-v1
    profile static-response-v1
    capability deployment-executor runtime.native_reference
    operation execute json { {"outcome":"accepted"} }
}
`
	if _, err := syntax.NewParser(src).ParsePluginImplementation(); err == nil {
		t.Fatal("expected duplicate profile to fail")
	}
}

func TestParsePluginImplementationRejectsUnknownProfile(t *testing.T) {
	src := `
plugin-implementation tei.native.runtime.reference 1.0.0 {
    profile general-purpose
    capability deployment-executor runtime.native_reference
    operation execute json { {"outcome":"accepted"} }
}
`
	if _, err := syntax.NewParser(src).ParsePluginImplementation(); err == nil {
		t.Fatal("expected unsupported profile to fail")
	}
}
