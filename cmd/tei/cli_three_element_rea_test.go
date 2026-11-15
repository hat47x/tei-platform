package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// reaIntakeFixture builds one minimal REA attempt document. The digests are
// test values; no real REA run produced this file.
func reaIntakeFixture(t *testing.T) []byte {
	t.Helper()
	const digest = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	observation := func(id, source string, line int) map[string]any {
		return map[string]any{
			"observation_id": id,
			"evidence": map[string]any{
				"state":     "observed",
				"authority": "derived",
				"artifact":  map[string]any{"sha256": digest},
				"location": map[string]any{
					"available": true,
					"value": map[string]any{
						"kind":   "source-range",
						"source": source,
						"start":  map[string]any{"line": line, "column": 1},
						"end":    map[string]any{"line": line + 4, "column": 1},
					},
				},
				"coverage": map[string]any{"status": "complete", "exhausted": true},
			},
		}
	}
	document := map[string]any{
		"ok": true,
		"data": map[string]any{
			"operation":   "analyze_javascript_application",
			"evidence_id": "ev-cli-1",
			"provider":    map[string]any{"id": "rea-javascript-application", "version": "6.8.0"},
			"subject":     map[string]any{"digest": map[string]any{"sha256": digest}},
			"normalized_result": map[string]any{
				"root_artifact_sha256": digest,
				"graph": map[string]any{
					"schema":   "JavaScriptApplicationGraph",
					"coverage": map[string]any{"status": "complete", "exhausted": true},
					"nodes": []any{
						map[string]any{"node_id": "node-order", "kind": "module",
							"observations": []any{observation("obs-order", "src/order.js", 1)}},
						map[string]any{"node_id": "node-line", "kind": "module",
							"observations": []any{observation("obs-line", "src/line.js", 20)}},
					},
					"edges": []any{
						map[string]any{
							"edge_id":        "edge-order-line",
							"source_node_id": "node-order",
							"target_node_id": "node-line",
							"relation":       "imports",
							"evidence": map[string]any{
								"state":    "observed",
								"artifact": map[string]any{"sha256": digest},
								"location": map[string]any{"available": true, "value": map[string]any{
									"kind": "artifact-path", "path": "src/order.js",
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
	raw, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func decodeReaIntake(t *testing.T, output string) reaIntakeResponse {
	t.Helper()
	var response reaIntakeResponse
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("invalid JSON envelope: %v\n%s", err, output)
	}
	return response
}

// The first implementation target through the real CLI entry: one Observation
// and one directed Relation reach the review packet, and nothing is accepted.
func TestThreeElementReaIntakeMinimalAttempt(t *testing.T) {
	root := t.TempDir()
	evidencePath := filepath.Join(root, "analysis.json")
	if err := os.WriteFile(evidencePath, reaIntakeFixture(t), 0o644); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	code, err := runThreeElementCommand([]string{"rea-intake", "-evidence", evidencePath}, &out)
	if err != nil {
		t.Fatalf("command error: %v", err)
	}
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, out.String())
	}
	response := decodeReaIntake(t, out.String())
	if response.SchemaVersion != "tei.cli.three-element.rea-intake/v0" || !response.OK {
		t.Fatalf("envelope = %+v", response)
	}
	packet, ok := response.Result.(map[string]any)
	if !ok {
		t.Fatalf("result is not an object: %T", response.Result)
	}
	if packet["acceptedMeaning"] != false {
		t.Fatalf("acceptedMeaning = %v", packet["acceptedMeaning"])
	}
	if packet["ownerDecision"] != "pending" {
		t.Fatalf("ownerDecision = %v", packet["ownerDecision"])
	}
	if packet["effect"] != "read-only" {
		t.Fatalf("effect = %v", packet["effect"])
	}
	if packet["qualification"] != "complete" {
		t.Fatalf("qualification = %v", packet["qualification"])
	}
	candidates, ok := packet["candidates"].([]any)
	if !ok || len(candidates) != 3 {
		t.Fatalf("candidates = %v", packet["candidates"])
	}
	carrier, ok := packet["carrier"].(map[string]any)
	if !ok {
		t.Fatal("carrier audit missing")
	}
	if missing, ok := carrier["missing"].([]any); !ok || len(missing) == 0 {
		t.Fatal("carrier audit reports no gaps; the minimal attempt has several")
	}
	if packet["bindingAttempted"] != false || packet["adoptionPreviewed"] != false {
		t.Fatalf("unrequested sections were attempted: %v %v", packet["bindingAttempted"], packet["adoptionPreviewed"])
	}
}

// A missing -evidence is an argument error, not an empty success.
func TestThreeElementReaIntakeRequiresEvidence(t *testing.T) {
	var out strings.Builder
	code, err := runThreeElementCommand([]string{"rea-intake"}, &out)
	if err != nil {
		t.Fatalf("command error: %v", err)
	}
	if code != 2 {
		t.Fatalf("expected exit 2, got %d: %s", code, out.String())
	}
	response := decodeReaIntake(t, out.String())
	if response.OK || response.Error == nil || response.Error.Code != "INVALID_ARGUMENT" {
		t.Fatalf("failure envelope = %+v", response)
	}
}

// A reported REA failure is refused with the evidence code, not read as a
// partial success.
func TestThreeElementReaIntakeRejectsReportedFailure(t *testing.T) {
	root := t.TempDir()
	evidencePath := filepath.Join(root, "failed.json")
	if err := os.WriteFile(evidencePath, []byte(`{"ok":false,"data":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	code, err := runThreeElementCommand([]string{"rea-intake", "-evidence", evidencePath}, &out)
	if err != nil {
		t.Fatalf("command error: %v", err)
	}
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
	response := decodeReaIntake(t, out.String())
	if response.Error == nil || response.Error.Code != "EVIDENCE_REPORTED_FAILURE" {
		t.Fatalf("failure = %+v", response.Error)
	}
}

func TestThreeElementReaIntakeRejectsMalformedEvidence(t *testing.T) {
	root := t.TempDir()
	evidencePath := filepath.Join(root, "bad.json")
	if err := os.WriteFile(evidencePath, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	code, _ := runThreeElementCommand([]string{"rea-intake", "-evidence", evidencePath}, &out)
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
	response := decodeReaIntake(t, out.String())
	if response.Error == nil || response.Error.Code != "EVIDENCE_INVALID_JSON" {
		t.Fatalf("failure = %+v", response.Error)
	}
}

func TestThreeElementReaIntakeRejectsPositionalArguments(t *testing.T) {
	root := t.TempDir()
	evidencePath := filepath.Join(root, "analysis.json")
	if err := os.WriteFile(evidencePath, reaIntakeFixture(t), 0o644); err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	code, _ := runThreeElementCommand([]string{"rea-intake", "-evidence", evidencePath, "stray"}, &out)
	if code != 2 {
		t.Fatalf("expected exit 2, got %d: %s", code, out.String())
	}
	response := decodeReaIntake(t, out.String())
	if response.Error == nil || response.Error.Code != "INVALID_ARGUMENT" ||
		!strings.Contains(response.Error.Message, "positional") {
		t.Fatalf("failure = %+v", response.Error)
	}
}

// A mapping without a workspace would silently resolve nothing, so it is
// refused instead of looking applied.
func TestThreeElementReaIntakeRejectsMappingWithoutWorkspace(t *testing.T) {
	root := t.TempDir()
	evidencePath := filepath.Join(root, "analysis.json")
	mappingPath := filepath.Join(root, "mapping.json")
	if err := os.WriteFile(evidencePath, reaIntakeFixture(t), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mappingPath, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	code, _ := runThreeElementCommand([]string{"rea-intake", "-evidence", evidencePath, "-mapping", mappingPath}, &out)
	if code != 2 {
		t.Fatalf("expected exit 2, got %d: %s", code, out.String())
	}
	response := decodeReaIntake(t, out.String())
	if response.Error == nil || !strings.Contains(response.Error.Message, "requires -workspace") {
		t.Fatalf("failure = %+v", response.Error)
	}
}

// The adoption preview is requested by -model and still accepts nothing.
func TestThreeElementReaIntakePreviewsAdoptionWithoutAccepting(t *testing.T) {
	root := t.TempDir()
	evidencePath := filepath.Join(root, "analysis.json")
	modelPath := filepath.Join(root, "model.json")
	if err := os.WriteFile(evidencePath, reaIntakeFixture(t), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(modelPath, fixtureBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	code, err := runThreeElementCommand([]string{"rea-intake", "-evidence", evidencePath, "-model", modelPath}, &out)
	if err != nil {
		t.Fatalf("command error: %v", err)
	}
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, out.String())
	}
	response := decodeReaIntake(t, out.String())
	packet := response.Result.(map[string]any)
	if packet["adoptionPreviewed"] != true {
		t.Fatal("adoption preview was not reported")
	}
	adoption, ok := packet["adoption"].(map[string]any)
	if !ok {
		t.Fatal("adoption section missing")
	}
	for _, raw := range adoption["records"].([]any) {
		record := raw.(map[string]any)
		if record["accepted"] != false || record["owner"] != "" {
			t.Fatalf("record claims an acceptance: %+v", record)
		}
		if record["sourceRevision"] == nil {
			t.Fatalf("record lost the source revision: %+v", record)
		}
	}
}

// -output writes the packet and the envelope still reports it; the file is the
// review artefact, not an accepted model.
func TestThreeElementReaIntakeWritesReviewPacket(t *testing.T) {
	root := t.TempDir()
	evidencePath := filepath.Join(root, "analysis.json")
	outputPath := filepath.Join(root, "review.json")
	if err := os.WriteFile(evidencePath, reaIntakeFixture(t), 0o644); err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	code, err := runThreeElementCommand([]string{"rea-intake", "-evidence", evidencePath, "-output", outputPath}, &out)
	if err != nil {
		t.Fatalf("command error: %v", err)
	}
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, out.String())
	}
	raw, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	var written map[string]any
	if err := json.Unmarshal(raw, &written); err != nil {
		t.Fatalf("written file is not JSON: %v", err)
	}
	if written["acceptedMeaning"] != false || written["ownerDecision"] != "pending" {
		t.Fatalf("written packet claims an acceptance: %v %v", written["acceptedMeaning"], written["ownerDecision"])
	}
	expectedDigest := fmt.Sprintf("sha256:%x", sha256.Sum256(reaIntakeFixture(t)))
	if written["candidateSetFingerprint"] != expectedDigest {
		t.Fatalf("fingerprint = %v, want raw input digest %q", written["candidateSetFingerprint"], expectedDigest)
	}
	if written["fingerprintBasis"] != "raw-evidence-bytes-not-candidate-identity" {
		t.Fatalf("missing raw-byte fingerprint qualification: %v", written["fingerprintBasis"])
	}
	if len(expectedDigest) != 71 {
		t.Fatalf("bad digest length: %q", expectedDigest)
	}
	unresolved, ok := written["unresolved"].([]any)
	if !ok || len(unresolved) == 0 {
		t.Fatal("written packet reports nothing unresolved")
	}
}

// -before used to issue "unchanged" for every synthetic candidate regardless
// of the contents of the previous packet. A CLI consumer must not receive a
// success or false equality while the source-backed comparison is unbound.
func TestThreeElementReaIntakeBeforeRefusesFabricatedUnchanged(t *testing.T) {
	root := t.TempDir()
	evidencePath := filepath.Join(root, "analysis.json")
	previousPath := filepath.Join(root, "previous.json")
	outputPath := filepath.Join(root, "would-write.json")
	if err := os.WriteFile(evidencePath, reaIntakeFixture(t), 0o644); err != nil {
		t.Fatal(err)
	}
	var first strings.Builder
	if code, err := runThreeElementCommand([]string{"rea-intake", "-evidence", evidencePath}, &first); err != nil || code != 0 {
		t.Fatalf("first run: code=%d err=%v out=%s", code, err, first.String())
	}
	response := decodeReaIntake(t, first.String())
	original, err := json.Marshal(response.Result)
	if err != nil {
		t.Fatal(err)
	}
	var changed map[string]any
	if err := json.Unmarshal(original, &changed); err != nil {
		t.Fatal(err)
	}
	changed["candidateSetFingerprint"] = "sha256:other-source"
	changed["candidates"] = []any{map[string]any{"candidateId": "unrelated", "status": "pending"}}
	changedBytes, err := json.Marshal(changed)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name     string
		previous []byte
	}{
		{name: "identical-previous-review", previous: original},
		{name: "changed-source-and-candidates", previous: changedBytes},
		{name: "unqualified-object", previous: []byte("{}")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.WriteFile(previousPath, tc.previous, 0o644); err != nil {
				t.Fatal(err)
			}
			var out strings.Builder
			code, err := runThreeElementCommand([]string{"rea-intake", "-evidence", evidencePath, "-before", previousPath, "-output", outputPath}, &out)
			if err != nil || code != 2 {
				t.Fatalf("expected fail-closed exit 2: code=%d err=%v out=%s", code, err, out.String())
			}
			response := decodeReaIntake(t, out.String())
			if response.OK || response.Result != nil || response.Error == nil || response.Error.Code != "REA_COMPARISON_UNAVAILABLE" {
				t.Fatalf("fabricated equality was not refused: %+v", response)
			}
			if _, err := os.Stat(outputPath); !os.IsNotExist(err) {
				t.Fatalf("refused comparison wrote an output packet: %v", err)
			}
		})
	}
}

func TestThreeElementReaIntakeBeforeMalformedOrMissing(t *testing.T) {
	root := t.TempDir()
	evidencePath := filepath.Join(root, "analysis.json")
	previousPath := filepath.Join(root, "previous.json")
	if err := os.WriteFile(evidencePath, reaIntakeFixture(t), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		prior []byte
		write bool
	}{
		{name: "missing", write: false},
		{name: "malformed", prior: []byte("{invalid"), write: true},
		{name: "null", prior: []byte("null"), write: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_ = os.Remove(previousPath)
			if tc.write {
				if err := os.WriteFile(previousPath, tc.prior, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			var out strings.Builder
			code, err := runThreeElementCommand([]string{"rea-intake", "-evidence", evidencePath, "-before", previousPath}, &out)
			if err != nil || code != 2 {
				t.Fatalf("invalid before did not fail: code=%d err=%v out=%s", code, err, out.String())
			}
			resp := decodeReaIntake(t, out.String())
			if resp.OK || resp.Error == nil || resp.Error.Code != "BEFORE_SET_READ_FAILED" {
				t.Fatalf("invalid previous packet was not rejected: %+v", resp)
			}
		})
	}
}

func TestThreeElementReaIntakeHelpListsItsOwnFlagsOnly(t *testing.T) {
	var out strings.Builder
	code, err := runThreeElementCommand([]string{"rea-intake", "-h"}, &out)
	if err != nil {
		t.Fatalf("command error: %v", err)
	}
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	text := out.String()
	for _, want := range []string{"-evidence", "-workspace", "-model", "-mapping", "-before", "-output"} {
		if !strings.Contains(text, want) {
			t.Fatalf("help does not document %s:\n%s", want, text)
		}
	}
}
