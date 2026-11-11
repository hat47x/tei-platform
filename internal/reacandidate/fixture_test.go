package reacandidate

import (
	"encoding/json"
	"strings"
	"testing"
)

// Fixed digests. They are test values, not real artifact digests.
const (
	digestA = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	digestB = "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"
)

// nodeFixture is one REA graph node in the test builder.
type nodeFixture struct {
	id           string
	kind         string
	observations []observationFixture
}

type observationFixture struct {
	id        string
	state     string
	authority string
	artifact  string
	locKind   string // "" | artifact-path | source-range | bad-kind | escape-path
	locPath   string
	locSource string
	startLine int
	startCol  int
	endLine   int
	endCol    int
	coverage  string
}

type edgeFixture struct {
	id        string
	from      string
	to        string
	rel       string
	state     string
	artifact  string
	locPath   string
	locSource string
	startLine int
	startCol  int
	endLine   int
	endCol    int
}

type unknownFixture struct {
	id     string
	family string
	reason string
}

type evidenceFixture struct {
	evidenceID      string
	operation       string
	providerID      string
	providerVersion string
	subjectDigest   string
	rootDigest      string
	omitRootDigest  bool
	nodes           []nodeFixture
	edges           []edgeFixture
	unknowns        []unknownFixture
	limitations     []string
	graphExhausted  *bool
	semExhausted    *bool
	extraTopLevel   map[string]any
}

func boolPtr(value bool) *bool { return &value }

// defaultEvidence is the minimal complete attempt: two observed nodes, one
// observation each, and one directed relation between them.
func defaultEvidence() evidenceFixture {
	observation := func(id, artifact, source string, line int) observationFixture {
		return observationFixture{
			id: id, state: "observed", authority: "derived",
			artifact: artifact, locKind: "source-range", locSource: source,
			startLine: line, startCol: 1, endLine: line + 4, endCol: 1,
			coverage: "complete",
		}
	}
	return evidenceFixture{
		evidenceID:      "ev-minimal-1",
		operation:       "analyze_javascript_application",
		providerID:      "rea-javascript-application",
		providerVersion: "6.8.0",
		subjectDigest:   digestA,
		rootDigest:      digestA,
		nodes: []nodeFixture{
			{id: "node-order", kind: "module", observations: []observationFixture{observation("obs-order", digestA, "src/order.js", 1)}},
			{id: "node-line", kind: "module", observations: []observationFixture{observation("obs-line", digestA, "src/line.js", 20)}},
		},
		edges: []edgeFixture{{
			id: "edge-order-line", from: "node-order", to: "node-line", rel: "imports",
			state: "observed", artifact: digestA, locPath: "src/order.js", locSource: "src/order.js",
			startLine: 3, startCol: 1, endLine: 3, endCol: 20,
		}},
		graphExhausted: boolPtr(true),
		semExhausted:   boolPtr(true),
	}
}

func (f evidenceFixture) build(t *testing.T) []byte {
	t.Helper()
	graph := map[string]any{
		"schema": "JavaScriptApplicationGraph",
		"nodes":  []any{},
		"edges":  []any{},
	}
	if f.graphExhausted != nil {
		graph["coverage"] = map[string]any{"status": "complete", "exhausted": *f.graphExhausted}
	}
	for _, node := range f.nodes {
		observations := []any{}
		for _, observation := range node.observations {
			evidence := map[string]any{
				"state":     observation.state,
				"authority": observation.authority,
			}
			if observation.artifact != "" {
				evidence["artifact"] = map[string]any{"sha256": observation.artifact}
			}
			evidence["location"] = locationOf(observation.locKind, observation.locPath, observation.locSource,
				observation.startLine, observation.startCol, observation.endLine, observation.endCol)
			if observation.coverage != "" {
				evidence["coverage"] = map[string]any{"status": observation.coverage, "exhausted": true}
			}
			observations = append(observations, map[string]any{
				"observation_id": observation.id,
				"evidence":       evidence,
			})
		}
		graph["nodes"] = append(graph["nodes"].([]any), map[string]any{
			"node_id":      node.id,
			"kind":         node.kind,
			"observations": observations,
		})
	}
	for _, edge := range f.edges {
		evidence := map[string]any{"state": edge.state}
		if edge.artifact != "" {
			evidence["artifact"] = map[string]any{"sha256": edge.artifact}
		}
		locKind := "source-range"
		if edge.locPath != "" && edge.locSource == "" {
			locKind = "artifact-path"
		}
		evidence["location"] = locationOf(locKind, edge.locPath, edge.locSource,
			edge.startLine, edge.startCol, edge.endLine, edge.endCol)
		graph["edges"] = append(graph["edges"].([]any), map[string]any{
			"edge_id":        edge.id,
			"source_node_id": edge.from,
			"target_node_id": edge.to,
			"relation":       edge.rel,
			"evidence":       evidence,
		})
	}
	normalized := map[string]any{"graph": graph}
	if !f.omitRootDigest && f.rootDigest != "" {
		normalized["root_artifact_sha256"] = f.rootDigest
	}
	unknowns := []any{}
	for _, unknown := range f.unknowns {
		unknowns = append(unknowns, map[string]any{
			"unknown_id": unknown.id,
			"family":     unknown.family,
			"reason":     unknown.reason,
		})
	}
	semantic := map[string]any{"unknowns": unknowns}
	if f.semExhausted != nil {
		semantic["coverage"] = map[string]any{"status": "complete", "exhausted": *f.semExhausted}
	}
	normalized["semantic_graph"] = semantic

	data := map[string]any{
		"operation":         f.operation,
		"evidence_id":       f.evidenceID,
		"provider":          map[string]any{"id": f.providerID, "version": f.providerVersion},
		"subject":           map[string]any{"digest": map[string]any{"sha256": f.subjectDigest}},
		"normalized_result": normalized,
	}
	if len(f.limitations) > 0 {
		data["limitations"] = f.limitations
	}
	document := map[string]any{"ok": true, "data": data, "meta": map[string]any{"command": "analyze"}}
	for key, value := range f.extraTopLevel {
		document[key] = value
	}
	raw, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	return raw
}

func locationOf(kind, pathValue, source string, startLine, startCol, endLine, endCol int) map[string]any {
	location := map[string]any{"available": true}
	switch kind {
	case "":
		location["available"] = false
	case "artifact-path":
		location["value"] = map[string]any{"kind": "artifact-path", "path": pathValue}
	case "source-range":
		value := map[string]any{"kind": "source-range", "source": source}
		if startLine > 0 {
			value["start"] = map[string]any{"line": startLine, "column": startCol}
			value["end"] = map[string]any{"line": endLine, "column": endCol}
		}
		location["value"] = value
	case "bad-kind":
		location["value"] = map[string]any{"kind": "mystery-location"}
	case "escape-path":
		location["value"] = map[string]any{"kind": "artifact-path", "path": "../outside.js"}
	default:
		location["value"] = map[string]any{"kind": kind}
	}
	return location
}

func mustDecode(t *testing.T, fixture evidenceFixture) AttemptEvidence {
	t.Helper()
	evidence, err := DecodeAttemptEvidence(fixture.build(t))
	if err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	return evidence
}

func mustGenerate(t *testing.T, fixture evidenceFixture) CandidateSet {
	t.Helper()
	set, err := GenerateCandidates(mustDecode(t, fixture))
	if err != nil {
		t.Fatalf("generate candidates: %v", err)
	}
	return set
}

func candidatesOfKind(set CandidateSet, kind string) []StructureCandidate {
	var out []StructureCandidate
	for _, candidate := range set.Candidates {
		if candidate.Kind == kind {
			out = append(out, candidate)
		}
	}
	return out
}

func candidateByID(set CandidateSet, id string) (StructureCandidate, bool) {
	for _, candidate := range set.Candidates {
		if candidate.CandidateID == id {
			return candidate, true
		}
	}
	return StructureCandidate{}, false
}

func hasNote(candidate StructureCandidate, note string) bool {
	for _, value := range candidate.Notes {
		if value == note {
			return true
		}
	}
	return false
}

func hasNotePrefix(set CandidateSet, prefix string) bool {
	for _, note := range set.Notes {
		if strings.HasPrefix(note, prefix) {
			return true
		}
	}
	return false
}

func hasString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func jsonUnmarshal(raw []byte, target any) error {
	return json.Unmarshal(raw, target)
}

func jsonMarshal(value any) ([]byte, error) {
	return json.Marshal(value)
}
