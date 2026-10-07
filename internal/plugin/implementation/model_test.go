package implementation_test

import (
	"context"
	"testing"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/permission"
	"tei/internal/plugin"
	"tei/internal/plugin/implementation"
	pluginbuiltin "tei/internal/plugin/implementation/builtin"
	pruntime "tei/internal/plugin/runtime"
)

func implementationSource(payload string) implementation.Source {
	return implementation.Source{
		Plugin:         plugin.Identity{ID: core.MustCanonicalID("tei.native.reference"), Version: "1.0.0"},
		Profile:        implementation.StaticResponseProfileV1,
		CapabilityKind: plugin.CapabilityDeploymentExecutor,
		CapabilityID:   core.MustCanonicalID("runtime.native"),
		Operations: []implementation.Operation{{
			Name:    "execute",
			Payload: []byte(payload),
		}},
	}
}

func implementationDefinition() plugin.Definition {
	return plugin.Definition{
		ID:      core.MustCanonicalID("tei.native.reference"),
		Version: "1.0.0",
		Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
		Provides: []plugin.Capability{{
			Kind:    plugin.CapabilityDeploymentExecutor,
			ID:      core.MustCanonicalID("runtime.native"),
			Actions: []deployment.ActionKind{deployment.ActionApplicationStart},
		}},
	}
}

func TestMaterializeCanonicalizesJSON(t *testing.T) {
	a := implementationSource(`{"attemptId":"a","outcome":"accepted"}`)
	b := implementationSource(`{ "outcome" : "accepted", "attemptId" : "a" }`)

	aBytes, aDigest, err := implementation.Materialize(a)
	if err != nil {
		t.Fatal(err)
	}
	bBytes, bDigest, err := implementation.Materialize(b)
	if err != nil {
		t.Fatal(err)
	}
	if string(aBytes) != string(bBytes) || aDigest != bDigest {
		t.Fatalf("equivalent JSON produced different realization:\n%s\n%s\n%s != %s", aBytes, bBytes, aDigest, bDigest)
	}
}

func TestNormalizeDoesNotMutateSource(t *testing.T) {
	source := implementationSource(`{ "outcome" : "accepted", "attemptId" : "a" }`)
	before := append([]byte(nil), source.Operations[0].Payload...)

	normalized, err := implementation.Normalize(source)
	if err != nil {
		t.Fatal(err)
	}
	if string(source.Operations[0].Payload) != string(before) {
		t.Fatal("normalization mutated authoritative source payload")
	}
	if string(normalized.Operations[0].Payload) != `{"attemptId":"a","outcome":"accepted"}` {
		t.Fatalf("normalized payload: %s", normalized.Operations[0].Payload)
	}
}

func TestStaticResponseBuiltinUsesExistingRuntimeABI(t *testing.T) {
	source := implementationSource(`{"attemptId":"tei-native","outcome":"accepted"}`)
	definition := implementationDefinition()
	builtin, err := pluginbuiltin.New(source, definition)
	if err != nil {
		t.Fatal(err)
	}
	response, err := builtin.Invoke(context.Background(), pruntime.Request{
		Plugin:         source.Plugin,
		CapabilityKind: source.CapabilityKind,
		CapabilityID:   source.CapabilityID,
		Operation:      "execute",
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Status != pruntime.ResponseOK {
		t.Fatalf("status: %s", response.Status)
	}
	if string(response.Payload) != `{"attemptId":"tei-native","outcome":"accepted"}` {
		t.Fatalf("payload: %s", response.Payload)
	}
}

func TestStaticResponseBuiltinRejectsUnsupportedInvocation(t *testing.T) {
	source := implementationSource(`{"attemptId":"a","outcome":"accepted"}`)
	builtin, err := pluginbuiltin.New(source, implementationDefinition())
	if err != nil {
		t.Fatal(err)
	}
	tests := []pruntime.Request{
		{
			Plugin:         source.Plugin,
			CapabilityKind: source.CapabilityKind,
			CapabilityID:   source.CapabilityID,
			Operation:      "unknown",
		},
		{
			Plugin:         source.Plugin,
			CapabilityKind: source.CapabilityKind,
			CapabilityID:   core.MustCanonicalID("runtime.other"),
			Operation:      "execute",
		},
	}
	for _, request := range tests {
		if _, err := builtin.Invoke(context.Background(), request); err == nil {
			t.Fatalf("expected invocation to fail: %#v", request)
		}
	}
}

func TestValidateAgainstDefinitionRejectsRuntimeAndCapabilityMismatch(t *testing.T) {
	source := implementationSource(`{"attemptId":"a","outcome":"accepted"}`)
	definition := implementationDefinition()

	process := definition
	process.Runtime.Kind = plugin.RuntimeProcess
	if err := implementation.ValidateAgainstDefinition(source, process); err == nil {
		t.Fatal("expected process runtime to fail")
	}

	wrongCapability := definition
	wrongCapability.Provides = []plugin.Capability{{
		Kind: plugin.CapabilityDeploymentExecutor,
		ID:   core.MustCanonicalID("runtime.other"),
	}}
	if err := implementation.ValidateAgainstDefinition(source, wrongCapability); err == nil {
		t.Fatal("expected capability mismatch to fail")
	}
}


func TestStaticResponseProfileRejectsScopeExpansion(t *testing.T) {
	t.Run("other capability kind", func(t *testing.T) {
		source := implementationSource(`{"attemptId":"a","outcome":"accepted"}`)
		source.CapabilityKind = plugin.CapabilityObservationProvider
		if err := source.Validate(); err == nil {
			t.Fatal("expected non-deployment capability to fail")
		}
	})
	t.Run("additional operation", func(t *testing.T) {
		source := implementationSource(`{"attemptId":"a","outcome":"accepted"}`)
		source.Operations = append(source.Operations, implementation.Operation{
			Name:    "other",
			Payload: []byte(`{"ok":true}`),
		})
		if err := source.Validate(); err == nil {
			t.Fatal("expected additional operation to fail")
		}
	})
	t.Run("renamed operation", func(t *testing.T) {
		source := implementationSource(`{"attemptId":"a","outcome":"accepted"}`)
		source.Operations[0].Name = "observe"
		if err := source.Validate(); err == nil {
			t.Fatal("expected non-execute operation to fail")
		}
	})
	t.Run("additional capability", func(t *testing.T) {
		source := implementationSource(`{"attemptId":"a","outcome":"accepted"}`)
		definition := implementationDefinition()
		definition.Provides = append(definition.Provides, plugin.Capability{
			Kind: plugin.CapabilityObservationProvider,
			ID:   core.MustCanonicalID("observation.other"),
		})
		if err := implementation.ValidateAgainstDefinition(source, definition); err == nil {
			t.Fatal("expected partial Plugin realization to fail")
		}
	})
	t.Run("multiple deployment actions", func(t *testing.T) {
		source := implementationSource(`{"attemptId":"a","outcome":"accepted"}`)
		definition := implementationDefinition()
		definition.Provides[0].Actions = append(
			definition.Provides[0].Actions,
			deployment.ActionApplicationRetire,
		)
		if err := implementation.ValidateAgainstDefinition(source, definition); err == nil {
			t.Fatal("expected multi-action capability to fail")
		}
	})
	t.Run("capability permission", func(t *testing.T) {
		source := implementationSource(`{"attemptId":"a","outcome":"accepted"}`)
		definition := implementationDefinition()
		definition.Provides[0].Permissions = []permission.Spec{{
			Kind:   permission.KindFilesystem,
			Access: permission.AccessRead,
			Scope:  "workspace-source",
		}}
		if err := implementation.ValidateAgainstDefinition(source, definition); err == nil {
			t.Fatal("expected permission-bearing capability to fail")
		}
	})
	t.Run("plugin dependency", func(t *testing.T) {
		source := implementationSource(`{"attemptId":"a","outcome":"accepted"}`)
		definition := implementationDefinition()
		definition.Requires = []plugin.Requirement{{
			Plugin:          core.MustCanonicalID("tei.other"),
			CompatibleMajor: 1,
		}}
		if err := implementation.ValidateAgainstDefinition(source, definition); err == nil {
			t.Fatal("expected Plugin dependency to fail")
		}
	})
	t.Run("external dependency", func(t *testing.T) {
		source := implementationSource(`{"attemptId":"a","outcome":"accepted"}`)
		definition := implementationDefinition()
		definition.External = []plugin.ExternalRequirement{{
			Scope: "runtime",
			Kind:  "service",
			Name:  "external",
		}}
		if err := implementation.ValidateAgainstDefinition(source, definition); err == nil {
			t.Fatal("expected external dependency to fail")
		}
	})
}


func TestStaticResponseProfileRejectsHostOwnedOrInvalidResponseFields(t *testing.T) {
	tests := []struct {
		name    string
		payload string
	}{
		{"action id", `{"attemptId":"a","actionId":"forged","outcome":"accepted"}`},
		{"idempotency key", `{"attemptId":"a","idempotencyKey":"forged","outcome":"accepted"}`},
		{"executor id", `{"attemptId":"a","executorId":"forged","outcome":"accepted"}`},
		{"plugin provenance", `{"attemptId":"a","outcome":"accepted","pluginInvocation":{"pluginId":"tei.x"}}`},
		{"external ref", `{"attemptId":"a","outcome":"accepted","externalRef":"forged"}`},
		{"observed at", `{"attemptId":"a","outcome":"accepted","observedAt":"2026-01-01T00:00:00Z"}`},
		{"unknown field", `{"attemptId":"a","outcome":"accepted","unknown":true}`},
		{"duplicate key", `{"attemptId":"a","attemptId":"b","outcome":"accepted"}`},
		{"missing attempt", `{"outcome":"accepted"}`},
		{"invalid outcome", `{"attemptId":"a","outcome":"maybe"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := implementationSource(tt.payload)
			if err := source.Validate(); err == nil {
				t.Fatal("expected invalid deployment executor response to fail")
			}
		})
	}
}


func TestStaticResponseProfileRejectsExcessiveJSONNesting(t *testing.T) {
	payload := `{"attemptId":"a","outcome":"accepted","message":`
	for i := 0; i < 130; i++ {
		payload += "["
	}
	payload += `"x"`
	for i := 0; i < 130; i++ {
		payload += "]"
	}
	payload += "}"
	source := implementationSource(payload)
	if err := source.Validate(); err == nil {
		t.Fatal("expected excessive JSON nesting to fail")
	}
}
