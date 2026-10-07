package implementation_test

import (
	"context"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	"tei/internal/plugin/implementation"
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
			Kind: plugin.CapabilityDeploymentExecutor,
			ID:   core.MustCanonicalID("runtime.native"),
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

func TestNormalizeSortsOperationsWithoutMutatingSource(t *testing.T) {
	source := implementationSource(`{"outcome":"accepted"}`)
	source.Operations = append([]implementation.Operation{
		{Name: "z", Payload: []byte(`{"value":2}`)},
		{Name: "a", Payload: []byte(`{"value":1}`)},
	}, source.Operations...)
	before := source.Operations[0].Name

	normalized, err := implementation.Normalize(source)
	if err != nil {
		t.Fatal(err)
	}
	if source.Operations[0].Name != before {
		t.Fatal("normalization mutated authoritative source ordering")
	}
	if normalized.Operations[0].Name != "a" || normalized.Operations[len(normalized.Operations)-1].Name != "z" {
		t.Fatalf("normalized operations: %#v", normalized.Operations)
	}
}

func TestStaticResponseBuiltinUsesExistingRuntimeABI(t *testing.T) {
	source := implementationSource(`{"attemptId":"tei-native","outcome":"accepted"}`)
	definition := implementationDefinition()
	builtin, err := implementation.NewBuiltin(source, definition)
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
	source := implementationSource(`{"outcome":"accepted"}`)
	builtin, err := implementation.NewBuiltin(source, implementationDefinition())
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
	source := implementationSource(`{"outcome":"accepted"}`)
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
