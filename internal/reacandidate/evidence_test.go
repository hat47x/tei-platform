package reacandidate

import (
	"strings"
	"testing"
)

func TestDecodeMinimalAttemptCarriesExternalIdentities(t *testing.T) {
	evidence := mustDecode(t, defaultEvidence())
	if evidence.EvidenceID != "ev-minimal-1" {
		t.Fatalf("evidence id = %q", evidence.EvidenceID)
	}
	if evidence.Provider.ID != "rea-javascript-application" || evidence.Provider.Version != "6.8.0" {
		t.Fatalf("provider descriptor = %+v", evidence.Provider)
	}
	if evidence.Subject.DigestSHA256 != digestA {
		t.Fatalf("subject digest = %q", evidence.Subject.DigestSHA256)
	}
	if !evidence.RootDigestOK {
		t.Fatal("root digest agreement not recorded")
	}
	if len(evidence.Normalized.Graph.Nodes) != 2 || len(evidence.Normalized.Graph.Edges) != 1 {
		t.Fatalf("graph size = %d nodes %d edges", len(evidence.Normalized.Graph.Nodes), len(evidence.Normalized.Graph.Edges))
	}
	edge := evidence.Normalized.Graph.Edges[0]
	if edge.SourceNodeID != "node-order" || edge.TargetNodeID != "node-line" || edge.EdgeID != "edge-order-line" {
		t.Fatalf("edge identity/direction lost: %+v", edge)
	}
	node := evidence.Normalized.Graph.Nodes[0]
	observation := node.Observations[0]
	if observation.Evidence.Location == nil || observation.Evidence.Location.Value == nil ||
		observation.Evidence.Location.Value.Source != "src/order.js" {
		t.Fatalf("source location lost: %+v", observation.Evidence.Location)
	}
	if observation.Evidence.Artifact.SHA256 != digestA {
		t.Fatalf("artifact digest lost: %+v", observation.Evidence.Artifact)
	}
}

func TestDecodeRejectsReportedFailureAndMalformedDocuments(t *testing.T) {
	cases := []struct {
		name     string
		raw      string
		wantCode string
	}{
		{"not-json", `{`, "EVIDENCE_INVALID_JSON"},
		{"no-ok", `{"data":{}}`, "EVIDENCE_STATUS_MISSING"},
		{"reported-failure", `{"ok":false,"data":{}}`, "EVIDENCE_REPORTED_FAILURE"},
		{"no-data", `{"ok":true}`, "EVIDENCE_DATA_MISSING"},
		{"no-operation", `{"ok":true,"data":{"evidence_id":"e"}}`, "OPERATION_MISSING"},
		{"no-evidence-id", `{"ok":true,"data":{"operation":"x"}}`, "EVIDENCE_ID_MISSING"},
		{"no-provider", `{"ok":true,"data":{"operation":"x","evidence_id":"e","provider":{"id":"p"}}}`, "PROVIDER_DESCRIPTOR_INCOMPLETE"},
		{"bad-subject-digest", `{"ok":true,"data":{"operation":"x","evidence_id":"e","provider":{"id":"p","version":"1"},"subject":{"digest":{"sha256":"abc"}}}}`, "SUBJECT_DIGEST_INVALID"},
		{"no-graph", `{"ok":true,"data":{"operation":"x","evidence_id":"e","provider":{"id":"p","version":"1"},"subject":{"digest":{"sha256":"` + digestA + `"}}}}`, "GRAPH_MISSING"},
		{"unsupported-schema", `{"ok":true,"data":{"operation":"x","evidence_id":"e","provider":{"id":"p","version":"1"},"subject":{"digest":{"sha256":"` + digestA + `"}},"normalized_result":{"graph":{"schema":"OtherGraph"}}}}`, "GRAPH_SCHEMA_UNSUPPORTED"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecodeAttemptEvidence([]byte(tc.raw))
			if err == nil {
				t.Fatalf("document accepted: %s", tc.raw)
			}
			evidenceErr, ok := err.(*EvidenceError)
			if !ok {
				t.Fatalf("error type = %T", err)
			}
			if evidenceErr.Code != tc.wantCode {
				t.Fatalf("code = %q want %q (%v)", evidenceErr.Code, tc.wantCode, err)
			}
		})
	}
}

func TestDecodeRejectsDigestConflict(t *testing.T) {
	fixture := defaultEvidence()
	fixture.rootDigest = digestB
	_, err := DecodeAttemptEvidence(fixture.build(t))
	if err == nil {
		t.Fatal("subject/root digest conflict accepted")
	}
	evidenceErr, ok := err.(*EvidenceError)
	if !ok || evidenceErr.Code != "DIGEST_CONFLICT" {
		t.Fatalf("error = %v", err)
	}
}

func TestDecodeRejectsDuplicateIdentities(t *testing.T) {
	duplicateNode := defaultEvidence()
	duplicateNode.nodes = append(duplicateNode.nodes, duplicateNode.nodes[0])
	if _, err := DecodeAttemptEvidence(duplicateNode.build(t)); err == nil {
		t.Fatal("duplicate node_id accepted")
	} else if code := err.(*EvidenceError).Code; code != "NODE_IDENTITY_DUPLICATE" {
		t.Fatalf("code = %q", code)
	}

	duplicateObservation := defaultEvidence()
	duplicateObservation.nodes[1].observations[0].id = duplicateObservation.nodes[0].observations[0].id
	if _, err := DecodeAttemptEvidence(duplicateObservation.build(t)); err == nil {
		t.Fatal("duplicate observation_id accepted")
	} else if code := err.(*EvidenceError).Code; code != "OBSERVATION_IDENTITY_DUPLICATE" {
		t.Fatalf("code = %q", code)
	}

	duplicateEdge := defaultEvidence()
	duplicateEdge.edges = append(duplicateEdge.edges, duplicateEdge.edges[0])
	if _, err := DecodeAttemptEvidence(duplicateEdge.build(t)); err == nil {
		t.Fatal("duplicate edge_id accepted")
	} else if code := err.(*EvidenceError).Code; code != "EDGE_IDENTITY_DUPLICATE" {
		t.Fatalf("code = %q", code)
	}

	duplicateUnknown := defaultEvidence()
	duplicateUnknown.unknowns = []unknownFixture{{id: "u1"}, {id: "u1"}}
	if _, err := DecodeAttemptEvidence(duplicateUnknown.build(t)); err == nil {
		t.Fatal("duplicate unknown_id accepted")
	} else if code := err.(*EvidenceError).Code; code != "UNKNOWN_IDENTITY_DUPLICATE" {
		t.Fatalf("code = %q", code)
	}
}

func TestDecodeRejectsReservedControlMarkerInsideRecords(t *testing.T) {
	fixture := defaultEvidence()
	document := map[string]any{}
	if err := jsonUnmarshal(fixture.build(t), &document); err != nil {
		t.Fatal(err)
	}
	data := document["data"].(map[string]any)
	graph := data["normalized_result"].(map[string]any)["graph"].(map[string]any)
	node := graph["nodes"].([]any)[0].(map[string]any)
	node["$rea"] = "jsonl/v1"
	raw, err := jsonMarshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeAttemptEvidence(raw); err == nil {
		t.Fatal("reserved control marker accepted inside a node")
	} else if code := err.(*EvidenceError).Code; code != "RESERVED_CONTROL_MARKER" {
		t.Fatalf("code = %q", code)
	}
}

func TestDecodeRejectsMissingOccurrenceEvidenceAndEndpoint(t *testing.T) {
	fixture := defaultEvidence()
	document := map[string]any{}
	if err := jsonUnmarshal(fixture.build(t), &document); err != nil {
		t.Fatal(err)
	}
	data := document["data"].(map[string]any)
	graph := data["normalized_result"].(map[string]any)["graph"].(map[string]any)
	observation := graph["nodes"].([]any)[0].(map[string]any)["observations"].([]any)[0].(map[string]any)
	delete(observation, "evidence")
	raw, err := jsonMarshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeAttemptEvidence(raw); err == nil {
		t.Fatal("observation without evidence accepted")
	} else if code := err.(*EvidenceError).Code; code != "OCCURRENCE_EVIDENCE_MISSING" {
		t.Fatalf("code = %q", code)
	}

	edgeFixture := defaultEvidence()
	edgeFixture.edges[0].from = ""
	if _, err := DecodeAttemptEvidence(edgeFixture.build(t)); err == nil {
		t.Fatal("edge without a source endpoint accepted")
	} else if code := err.(*EvidenceError).Code; code != "EDGE_ENDPOINT_MISSING" {
		t.Fatalf("code = %q", code)
	}
}

func TestDecodeToleratesUnknownExternalFields(t *testing.T) {
	fixture := defaultEvidence()
	fixture.extraTopLevel = map[string]any{"unexpected": map[string]any{"deep": []any{1, 2, 3}}}
	evidence := mustDecode(t, fixture)
	if evidence.EvidenceID == "" {
		t.Fatal("unknown top-level fields broke decoding")
	}
}

func TestDecodeRecordsUnavailableAndInvalidLocations(t *testing.T) {
	fixture := defaultEvidence()
	fixture.nodes[0].observations[0].locKind = ""
	fixture.nodes[1].observations[0].locKind = "bad-kind"
	set := mustGenerate(t, fixture)
	for _, candidate := range candidatesOfKind(set, CandidateKindDataElement) {
		if candidate.SourceRevision.Binding != SourceBindingUnavailable {
			t.Fatalf("candidate %s binding = %q, want unavailable", candidate.CandidateID, candidate.SourceRevision.Binding)
		}
		if !hasString(candidate.Unresolved, "source-location") {
			t.Fatalf("candidate %s does not record the unresolved source location: %v", candidate.CandidateID, candidate.Unresolved)
		}
	}
	if set.Qualification != "partial" {
		t.Fatalf("qualification = %q, want partial", set.Qualification)
	}
}

func TestDecodeRecordsInvalidPathAndDigestWithoutTrustingThem(t *testing.T) {
	fixture := defaultEvidence()
	fixture.nodes[0].observations[0].locKind = "escape-path"
	fixture.nodes[1].observations[0].artifact = "not-a-digest"
	set := mustGenerate(t, fixture)
	for _, candidate := range candidatesOfKind(set, CandidateKindDataElement) {
		if candidate.SourceRevision.Binding == SourceBindingProviderReported {
			t.Fatalf("candidate %s kept an unusable source as bound", candidate.CandidateID)
		}
		if candidate.SourceRevision.ArtifactPath != "" && strings.Contains(candidate.SourceRevision.ArtifactPath, "..") {
			t.Fatalf("escaping path retained: %q", candidate.SourceRevision.ArtifactPath)
		}
	}
}
