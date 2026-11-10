package modelref_test

import (
	"encoding/json"
	"strings"
	"testing"

	"tei/internal/modelref"
	"tei/internal/subject"
)

// The trace result has two fingerprints with different scopes. The inner one,
// next to nextOffset, used to carry the same name as the Workspace-level
// continuation token, so a reader took it for the token and got
// STALE_SNAPSHOT (#3625). It is now graphFingerprint, and the old name stays as
// a deprecated alias for one v0 cycle.
func TestTraceResultNamesTheGraphFingerprintApartFromTheDeprecatedAlias(t *testing.T) {
	graph := modelref.Graph{
		Schema: "tei.modelref.v0",
		Nodes: []modelref.Node{
			{Key: modelref.Key{Scope: "s", Subject: subject.Subject{Kind: "semantic-data", ID: "sales.order"}}},
			{Key: modelref.Key{Scope: "s", Subject: subject.Subject{Kind: "semantic-data", ID: "customer.customer"}}},
		},
		Edges: []modelref.Edge{{
			From:     modelref.Key{Scope: "s", Subject: subject.Subject{Kind: "semantic-data", ID: "sales.order"}},
			To:       modelref.Ref{Scope: "s", Subject: subject.Subject{Kind: "semantic-data", ID: "customer.customer"}},
			Relation: "targets",
		}},
	}
	rules := []modelref.Rule{{Relation: "targets", FromKind: "semantic-data", ToKind: "semantic-data"}}
	key := modelref.Key{Scope: "s", Subject: subject.Subject{Kind: "semantic-data", ID: "sales.order"}}
	result, err := modelref.TraceDirect(graph, rules, key, modelref.TraceOptions{Mode: "full", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if result.GraphFingerprint == "" {
		t.Fatal("graphFingerprint is empty")
	}
	if result.SnapshotFingerprint != result.GraphFingerprint {
		t.Fatalf("deprecated alias differs from graphFingerprint: %q vs %q",
			result.SnapshotFingerprint, result.GraphFingerprint)
	}

	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"graphFingerprint", "snapshotFingerprint"} {
		if _, ok := document[name]; !ok {
			t.Fatalf("%s is missing from the v0 document: %s", name, raw)
		}
	}
	var inner struct {
		GraphFingerprint    string `json:"graphFingerprint"`
		SnapshotFingerprint string `json:"snapshotFingerprint"`
	}
	if err := json.Unmarshal(raw, &inner); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(inner.GraphFingerprint, "sha256:") {
		t.Fatalf("graphFingerprint is not a fingerprint: %q", inner.GraphFingerprint)
	}
}
