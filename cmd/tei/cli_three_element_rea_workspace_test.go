package main

// TestThreeElementReaIntakeBindsAgainstARealWorkspace exercises the CLI
// binding path against an actual workspace rather than a fake port. The
// fixture workspace declares one semantic-source file, and the REA attempt
// points at that same file, so the sha256 of the file's CURRENT bytes is what
// an independent read produces.
//
// This test proves the read-only comparison and its four outcomes. It does not
// prove that REA read those bytes: the attempt is a hand-written fixture, and
// its provider descriptor is a label, not a witness.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const reaRealWorkspaceManifest = `# tei.workspace.yaml
workspace: rea-intake-probe
formatVersion: 1
sources:
  - semantic
`

// reaRealWorkspaceSource is the analyzed artifact. Its bytes are fixed by this
// file, so the test can compute the digest an independent read must produce.
const reaRealWorkspaceSource = `namespace probe

data Order {
    attribute status type std.text
}
`

func writeReaRealWorkspace(t *testing.T) (root, sourcePath, digest string) {
	t.Helper()
	root = t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "semantic"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(reaRealWorkspaceManifest), 0o644); err != nil {
		t.Fatal(err)
	}
	relative := "semantic/probe.tei"
	sourcePath = filepath.Join(root, "semantic", "probe.tei")
	if err := os.WriteFile(sourcePath, []byte(reaRealWorkspaceSource), 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(reaRealWorkspaceSource))
	return root, relative, hex.EncodeToString(sum[:])
}

// reaEvidenceForSource builds an attempt whose observation points at one
// workspace-relative path with a chosen digest, plus one directed relation.
//
// The span is the exact source range of the declaration the fixture workspace
// contains (`semantic-data:probe.order`, line 3 columns 6-11). The Source
// Index records the identifier token's range, not the whole declaration, so a
// wider span would never fall inside a declared range and the coincidence
// check would not be exercised.
func reaEvidenceForSource(t *testing.T, source, digest string, mutate func(map[string]any)) []byte {
	t.Helper()
	observation := func(id string) map[string]any {
		return map[string]any{
			"observation_id": id,
			"evidence": map[string]any{
				"state":     "observed",
				"authority": "derived",
				"artifact":  map[string]any{"sha256": digest},
				"location": map[string]any{"available": true, "value": map[string]any{
					"kind":   "source-range",
					"source": source,
					"start":  map[string]any{"line": 3, "column": 6},
					"end":    map[string]any{"line": 3, "column": 11},
				}},
				"coverage": map[string]any{"status": "complete", "exhausted": true},
			},
		}
	}
	document := map[string]any{
		"ok": true,
		"data": map[string]any{
			"operation":   "analyze_javascript_application",
			"evidence_id": "ev-real-workspace",
			"provider":    map[string]any{"id": "rea-javascript-application", "version": "6.8.0"},
			"subject":     map[string]any{"digest": map[string]any{"sha256": digest}},
			"normalized_result": map[string]any{
				"root_artifact_sha256": digest,
				"graph": map[string]any{
					"schema":   "JavaScriptApplicationGraph",
					"coverage": map[string]any{"status": "complete", "exhausted": true},
					"nodes": []any{
						map[string]any{"node_id": "node-probe", "kind": "module",
							"observations": []any{observation("obs-probe")}},
					},
					"edges": []any{
						map[string]any{
							"edge_id":        "edge-probe-self",
							"source_node_id": "node-probe",
							"target_node_id": "node-probe",
							"relation":       "declares",
							"evidence": map[string]any{
								"state":    "observed",
								"artifact": map[string]any{"sha256": digest},
								"location": map[string]any{"available": true, "value": map[string]any{
									"kind": "source-range", "source": source,
									"start": map[string]any{"line": 1, "column": 1},
									"end":   map[string]any{"line": 3, "column": 1},
								}},
							},
						},
					},
				},
				"semantic_graph": map[string]any{
					"unknowns": []any{},
					"coverage": map[string]any{"status": "complete", "exhausted": true},
				},
			},
		},
		"meta": map[string]any{"command": "analyze"},
	}
	if mutate != nil {
		sub := document["data"].(map[string]any)
		graph := sub["normalized_result"].(map[string]any)["graph"].(map[string]any)
		first := graph["nodes"].([]any)[0].(map[string]any)
		firstObservation := first["observations"].([]any)[0].(map[string]any)
		mutate(firstObservation)
	}
	raw, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func runReaIntakeWithWorkspace(t *testing.T, workspaceRoot string, evidence []byte) reaIntakeResponse {
	t.Helper()
	root := t.TempDir()
	evidencePath := filepath.Join(root, "analysis.json")
	if err := os.WriteFile(evidencePath, evidence, 0o644); err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	code, err := runThreeElementCommand([]string{"rea-intake", "-evidence", evidencePath, "-workspace", workspaceRoot}, &out)
	if err != nil {
		t.Fatalf("command error: %v", err)
	}
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, out.String())
	}
	return decodeReaIntake(t, out.String())
}

func bindingForExternal(t *testing.T, response reaIntakeResponse, externalIdentity string) map[string]any {
	t.Helper()
	packet := response.Result.(map[string]any)
	candidates := packet["candidates"].([]any)
	bindings := packet["bindings"].([]any)
	var candidateID string
	for _, raw := range candidates {
		candidate := raw.(map[string]any)
		facts := candidate["facts"].(map[string]any)
		if facts["externalIdentity"] == externalIdentity &&
			candidate["kind"] == "data-element" {
			candidateID = candidate["candidateId"].(string)
		}
	}
	if candidateID == "" {
		t.Fatalf("no data candidate for %q:\n%s", externalIdentity, mustJSON(t, packet))
	}
	for _, raw := range bindings {
		binding := raw.(map[string]any)
		if binding["candidateId"] == candidateID {
			return binding
		}
	}
	t.Fatalf("candidate %s has no binding", candidateID)
	return nil
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// The analyzed bytes and the workspace's current bytes agree, so the binding
// is independent and the candidate becomes resolved ONLY because the caller
// supplied no mapping — it stays unresolved, and the digest match is reported
// as a current-byte fact, not an acceptance.
func TestThreeElementReaIntakeBindsIndependentlyWhenBytesMatch(t *testing.T) {
	workspaceRoot, source, digest := writeReaRealWorkspace(t)
	evidence := reaEvidenceForSource(t, source, digest, nil)
	response := runReaIntakeWithWorkspace(t, workspaceRoot, evidence)
	packet := response.Result.(map[string]any)
	if packet["bindingAttempted"] != true {
		t.Fatal("binding was not reported")
	}
	if packet["acceptedMeaning"] != false || packet["ownerDecision"] != "pending" {
		t.Fatalf("binding flipped an acceptance: %v %v", packet["acceptedMeaning"], packet["ownerDecision"])
	}
	binding := bindingForExternal(t, response, "node-probe")
	if binding["sourceBinding"] != "independently-bound" {
		t.Fatalf("source binding = %v", binding["sourceBinding"])
	}
	if binding["resolution"] != "unresolved" {
		t.Fatalf("digest match alone resolved the candidate: %v", binding["resolution"])
	}
	coincident, ok := binding["coincidentDeclarations"].([]any)
	if !ok || len(coincident) == 0 {
		t.Fatalf("the declared range was not reported as coincident: %v", binding["coincidentDeclarations"])
	}
	for _, raw := range coincidenceNotes(t, binding) {
		if raw == "coincident-declaration-is-structural-not-acceptance" {
			return
		}
	}
	t.Fatalf("the coincidence is not marked structural: %v", binding["sourceBindingNotes"])
}

func coincidenceNotes(t *testing.T, binding map[string]any) []string {
	t.Helper()
	raw, ok := binding["sourceBindingNotes"].([]any)
	if !ok {
		return nil
	}
	notes := make([]string, 0, len(raw))
	for _, value := range raw {
		notes = append(notes, value.(string))
	}
	return notes
}

// A changed digit in the analyzed artifact is reported as a digest conflict.
// It is neither an acceptance nor a failure of the candidate.
func TestThreeElementReaIntakeReportsDigestConflictForChangedBytes(t *testing.T) {
	workspaceRoot, source, digest := writeReaRealWorkspace(t)
	other := strings.Repeat("f", 64)
	if other == digest {
		t.Fatal("fixture digest collision")
	}
	response := runReaIntakeWithWorkspace(t, workspaceRoot, reaEvidenceForSource(t, source, other, nil))
	binding := bindingForExternal(t, response, "node-probe")
	if binding["sourceBinding"] != "source-digest-conflict" {
		t.Fatalf("source binding = %v", binding["sourceBinding"])
	}
	if binding["resolution"] != "unresolved" {
		t.Fatalf("conflict resolved the candidate: %v", binding["resolution"])
	}
	packet := response.Result.(map[string]any)
	unresolved := packet["unresolved"].([]any)
	found := false
	for _, raw := range unresolved {
		if raw.(string) == "source-digest-conflict-requires-fresh-analysis" {
			found = true
		}
	}
	if !found {
		t.Fatalf("conflict is not aggregated as unresolved: %v", unresolved)
	}
}

// An analyzed path that the workspace does not contain is reported as absent,
// not as a failure and not as an empty success.
func TestThreeElementReaIntakeReportsAbsentArtifact(t *testing.T) {
	workspaceRoot, _, digest := writeReaRealWorkspace(t)
	response := runReaIntakeWithWorkspace(t, workspaceRoot, reaEvidenceForSource(t, "semantic/absent.tei", digest, nil))
	binding := bindingForExternal(t, response, "node-probe")
	if binding["sourceBinding"] != "not-present-in-read-artifact-set" {
		t.Fatalf("source binding = %v", binding["sourceBinding"])
	}
	if binding["resolution"] != "unresolved" {
		t.Fatalf("absence resolved the candidate: %v", binding["resolution"])
	}
}

// A provider-reported invalid path (escaping the workspace) is refused at
// decode time, so it can never be compared against a workspace at all.
func TestThreeElementReaIntakeRefusesEscapingSourcePath(t *testing.T) {
	workspaceRoot, _, digest := writeReaRealWorkspace(t)
	evidence := reaEvidenceForSource(t, "semantic/probe.tei", digest, func(observation map[string]any) {
		value := observation["evidence"].(map[string]any)["location"].(map[string]any)["value"].(map[string]any)
		value["source"] = "../outside.tei"
	})
	root := t.TempDir()
	evidencePath := filepath.Join(root, "analysis.json")
	if err := os.WriteFile(evidencePath, evidence, 0o644); err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	code, err := runThreeElementCommand([]string{"rea-intake", "-evidence", evidencePath, "-workspace", workspaceRoot}, &out)
	if err != nil {
		t.Fatalf("command error: %v", err)
	}
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, out.String())
	}
	binding := bindingForExternal(t, decodeReaIntake(t, out.String()), "node-probe")
	if binding["sourceBinding"] == "independently-bound" {
		t.Fatalf("an escaping path was bound: %v", binding)
	}
}

// A real workspace whose bytes match still does not resolve anything without
// an Owner mapping; with one, the existing InspectSubject decides.
func TestThreeElementReaIntakeOwnerMappingUsesExistingInspect(t *testing.T) {
	workspaceRoot, source, digest := writeReaRealWorkspace(t)
	evidence := reaEvidenceForSource(t, source, digest, nil)
	response := runReaIntakeWithWorkspace(t, workspaceRoot, evidence)
	binding := bindingForExternal(t, response, "node-probe")
	candidateID := binding["candidateId"].(string)

	root := t.TempDir()
	evidencePath := filepath.Join(root, "analysis.json")
	mappingPath := filepath.Join(root, "mapping.json")
	if err := os.WriteFile(evidencePath, evidence, 0o644); err != nil {
		t.Fatal(err)
	}
	mapping := map[string]any{candidateID: map[string]any{"kind": "semantic-data", "id": "probe.Order"}}
	raw, err := json.Marshal(mapping)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mappingPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	code, err := runThreeElementCommand([]string{
		"rea-intake", "-evidence", evidencePath, "-workspace", workspaceRoot, "-mapping", mappingPath,
	}, &out)
	if err != nil {
		t.Fatalf("command error: %v", err)
	}
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, out.String())
	}
	packet := decodeReaIntake(t, out.String()).Result.(map[string]any)
	if packet["acceptedMeaning"] != false || packet["ownerDecision"] != "pending" {
		t.Fatalf("mapped run claims an acceptance: %v %v", packet["acceptedMeaning"], packet["ownerDecision"])
	}
	for _, raw := range packet["bindings"].([]any) {
		mapped := raw.(map[string]any)
		if mapped["candidateId"] != candidateID {
			continue
		}
		subject, ok := mapped["mappedSubject"].(map[string]any)
		if !ok || subject["id"] != "probe.Order" {
			t.Fatalf("mapping was not carried: %v", mapped["mappedSubject"])
		}
		if mapped["inspectStatus"] == "not-attempted" {
			t.Fatalf("the existing InspectSubject was not consulted: %v", mapped)
		}
		return
	}
	t.Fatalf("mapped candidate %s is missing from the bindings", candidateID)
}
