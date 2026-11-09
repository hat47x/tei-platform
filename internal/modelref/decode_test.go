package modelref_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"tei/internal/modelref"
)

func TestDecodeGraphPreservesScopedAndVersionedReferences(t *testing.T) {
	g := fixture()
	g.Nodes[0].Key.Scope = "orders-2026"
	g.Nodes[0].PhysicalName = "order_line_v2"
	g.Nodes[0].Evidence = []modelref.Evidence{{
		Authority: "source-observation", Context: "legacy-2026",
		SourceToken: "sha256:observed-token", ArtifactFingerprint: "sha256:original-asset",
	}}
	g.Edges[0].To.Scope = g.Nodes[0].Key.Scope
	g.Edges[0].Evidence = []modelref.Evidence{{
		Authority: "derived", Context: "projection-v0",
		ArtifactFingerprint: "sha256:source", RuleFingerprint: "sha256:rules",
	}}
	serialized, err := json.Marshal(g)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := modelref.DecodeGraph(serialized)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(g, restored) {
		t.Fatalf("scope, version or provenance lost: %#v", restored)
	}
	if diagnostics := modelref.Validate(restored, rules()); len(diagnostics) != 0 {
		t.Fatalf("roundtripped graph invalid: %+v", diagnostics)
	}
}

func TestDecodeGraphRejectsUnknownAndRepeatedMembers(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
		want string
	}{
		{"unknown graph field", `{"schema":"tei.modelref.v0","nodes":[],"edges":[],"futureMeaning":true}`, "unknown field"},
		{"unknown identity field", `{"schema":"tei.modelref.v0","nodes":[{"key":{"scope":"s","subject":{"kind":"data","id":"x","revision":"2"}}}],"edges":[]}`, "unknown field"},
		{"unknown evidence field", `{"schema":"tei.modelref.v0","nodes":[{"key":{"scope":"s","subject":{"kind":"data","id":"x"}},"evidence":[{"authority":"source-observation","sourceToken":"token","artifactFingerprint":"sha256:source","approved":true}]}],"edges":[]}`, "unknown field"},
		{"repeated schema", `{"schema":"tei.modelref.v0","schema":"tei.modelref.v1","nodes":[],"edges":[]}`, "repeated"},
		{"repeated scope", `{"schema":"tei.modelref.v0","nodes":[{"key":{"scope":"orders","scope":"other","subject":{"kind":"data","id":"x"}}}],"edges":[]}`, "repeated"},
		{"case-alias scope", `{"schema":"tei.modelref.v0","nodes":[{"key":{"scope":"orders","Scope":"other","subject":{"kind":"data","id":"x"}}}],"edges":[]}`, "repeated"},
		{"repeated provenance", `{"schema":"tei.modelref.v0","nodes":[{"key":{"scope":"orders","subject":{"kind":"data","id":"x"}},"evidence":[{"authority":"proposal","authority":"accepted"}]}],"edges":[]}`, "repeated"},
		{"unsupported schema", `{"schema":"tei.modelref.v1","nodes":[],"edges":[]}`, "unsupported schema"},
		{"missing schema", `{"nodes":[],"edges":[]}`, "unsupported schema"},
		{"extra JSON value", `{"schema":"tei.modelref.v0","nodes":[],"edges":[]} {"schema":"tei.modelref.v0"}`, "trailing"},
		{"malformed JSON", `{"schema":"tei.modelref.v0","nodes":[}`, "invalid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := modelref.DecodeGraph([]byte(tc.raw))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected diagnostic containing %q, got %v", tc.want, err)
			}
		})
	}
}

func TestDecodeGraphDoesNotConferMeaning(t *testing.T) {
	graph, err := modelref.DecodeGraph([]byte(`{"schema":"tei.modelref.v0","nodes":[{"key":{"scope":"s","subject":{"kind":"data","id":"x"}}}],"edges":[{"from":{"scope":"s","subject":{"kind":"data","id":"x"}},"to":{"scope":"s","subject":{"kind":"data","id":"missing"}},"relation":"depends-on"}]}`))
	if err != nil {
		t.Fatalf("well-formed wire representation rejected: %v", err)
	}
	if !hasCode(modelref.Validate(graph, rules()), "missing-reference") {
		t.Fatal("decoding a graph incorrectly marked a dangling reference valid")
	}
}
