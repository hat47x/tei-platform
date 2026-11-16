package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/reacandidate"
)

// The two saved attempts have equal structural source positions but distinct
// external edge IDs. The second one deliberately reverses provider edge order.
func parallelREAAttemptFixture(t *testing.T, evidenceID string, edgeIDs ...string) []byte {
	t.Helper()
	var document map[string]any
	if err := json.Unmarshal(reaIntakeFixture(t), &document); err != nil {
		t.Fatal(err)
	}
	data := document["data"].(map[string]any)
	data["evidence_id"] = evidenceID
	graph := data["normalized_result"].(map[string]any)["graph"].(map[string]any)
	edge := graph["edges"].([]any)[0].(map[string]any)
	graph["edges"] = make([]any, 0, len(edgeIDs))
	for _, id := range edgeIDs {
		cloned := make(map[string]any, len(edge))
		for key, value := range edge {
			cloned[key] = value
		}
		cloned["edge_id"] = id
		graph["edges"] = append(graph["edges"].([]any), cloned)
	}
	raw, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func runREACandidatesCLI(t *testing.T, path, output string) (int, reaIntakeResponse) {
	t.Helper()
	args := []string{"rea-candidates", "-evidence", path}
	if output != "" {
		args = append(args, "-output", output)
	}
	var response strings.Builder
	code, err := runThreeElementCommand(args, &response)
	if err != nil {
		t.Fatal(err)
	}
	return code, decodeReaIntake(t, response.String())
}

func TestREACandidatesCLIToTypedComparisonKeepsParallelRelationsUnpaired(t *testing.T) {
	dir := t.TempDir()
	beforeInput := filepath.Join(dir, "old-evidence.json")
	afterInput := filepath.Join(dir, "new-evidence.json")
	beforeSet := filepath.Join(dir, "old-candidates.json")
	afterSet := filepath.Join(dir, "new-candidates.json")
	oldBytes := parallelREAAttemptFixture(t, "ev-old", "old-1", "old-2")
	newBytes := parallelREAAttemptFixture(t, "ev-new", "new-2", "new-1")
	if err := os.WriteFile(beforeInput, oldBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(afterInput, newBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		evidencePath string
		outputPath   string
		raw          []byte
	}{
		{beforeInput, beforeSet, oldBytes},
		{afterInput, afterSet, newBytes},
	} {
		code, response := runREACandidatesCLI(t, tc.evidencePath, tc.outputPath)
		if code != 0 || !response.OK ||
			response.SchemaVersion != "tei.cli.three-element.rea-candidates/v0" {
			t.Fatalf("rea-candidates response: code=%d response=%+v", code, response)
		}
		result := response.Result.(map[string]any)
		expectedRawDigest := fmt.Sprintf("sha256:%x", sha256.Sum256(tc.raw))
		if result["rawEvidenceFingerprint"] != expectedRawDigest ||
			result["candidateOutputWritten"] != true {
			t.Fatalf("raw evidence or output was misreported: %+v", result)
		}
		packet := result["reviewPacket"].(map[string]any)
		setWire := result["candidateSet"].(map[string]any)
		if packet["acceptedMeaning"] != false || packet["ownerDecision"] != "pending" ||
			packet["effect"] != "read-only" ||
			packet["candidateSetFingerprint"] != setWire["fingerprint"] {
			t.Fatalf("source-backed candidate incorrectly accepted meaning: %+v", packet)
		}
		set, err := readREAComparisonCandidateSet(tc.outputPath)
		if err != nil {
			t.Fatalf("typed output is not consumable by existing comparator: %v", err)
		}
		if set.Attempt.EvidenceID == "" || len(set.Candidates) != 4 ||
			set.Effect != reacandidate.EffectReadOnly {
			t.Fatalf("typed candidate output lost observations/edges: %+v", set)
		}
		if set.Fingerprint == expectedRawDigest {
			t.Fatalf("raw byte digest was incorrectly used as candidate identity: %s", set.Fingerprint)
		}
	}
	code, response := runComparisonCLI(t, beforeSet, afterSet)
	if code != 0 || !response.OK {
		t.Fatalf("typed CLI comparison failed: exit=%d response=%+v", code, response)
	}
	packet := response.Result.(map[string]any)["reviewPacket"].(map[string]any)
	if packet["acceptedMeaning"] != false || packet["ownerDecision"] != "pending" {
		t.Fatalf("comparison acquired business authority: %+v", packet)
	}
	entries := packet["comparison"].(map[string]any)["entries"].([]any)
	oldCount, newCount := 0, 0
	for _, raw := range entries {
		entry := raw.(map[string]any)
		if entry["status"] != reacandidate.ComparisonAmbiguousStructural {
			continue
		}
		_, beforePresent := entry["beforeSource"]
		_, afterPresent := entry["afterSource"]
		_, beforeAttempt := entry["beforeAttemptId"]
		_, afterAttempt := entry["afterAttemptId"]
		switch {
		case beforePresent && beforeAttempt && !afterPresent && !afterAttempt:
			oldCount++
		case afterPresent && afterAttempt && !beforePresent && !beforeAttempt:
			newCount++
		default:
			t.Fatalf("unproven relation acquired a counterpart: %+v", entry)
		}
	}
	if oldCount != 2 || newCount != 2 {
		t.Fatalf("parallel relations were lost or paired: before=%d after=%d", oldCount, newCount)
	}
	summary := response.Result.(map[string]any)["summary"].([]any)
	found := false
	for _, line := range summary {
		if line == "comparison: ambiguous-structural-match=4" {
			found = true
		}
	}
	if !found {
		t.Fatalf("comparison hid ambiguous relations: %+v", summary)
	}
}

// A targeted CLI question is an exact producer-edge selection over the
// already-generated review; an "imports" edge cannot become an accepted
// Function→Data Read/Write merely because the caller asks that question.
func TestREACandidatesCLISelectsOneRelationWithoutInventingFunctionRead(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "analysis.json")
	output := filepath.Join(dir, "candidates.json")
	raw := parallelREAAttemptFixture(t, "ev-one-question", "edge-first", "edge-second")
	if err := os.WriteFile(input, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	code, err := runThreeElementCommand([]string{
		"rea-candidates", "-evidence", input, "-relation-id", "edge-second", "-output", output,
	}, &out)
	if err != nil {
		t.Fatal(err)
	}
	response := decodeReaIntake(t, out.String())
	if code != 0 || !response.OK {
		t.Fatalf("question CLI failed: code=%d response=%+v", code, response)
	}
	result := response.Result.(map[string]any)
	question, ok := result["relationQuestion"].(map[string]any)
	if !ok {
		t.Fatalf("explicit relation question missing: %+v", result)
	}
	relation := question["relation"].(map[string]any)
	facts := relation["facts"].(map[string]any)
	direction := facts["direction"].(map[string]any)
	if facts["externalOccurrenceId"] != "edge-second" ||
		facts["observedKind"] != "imports" ||
		direction["fromExternalId"] != "node-order" || direction["toExternalId"] != "node-line" ||
		len(question["fromObservations"].([]any)) != 1 ||
		len(question["toObservations"].([]any)) != 1 {
		t.Fatalf("question lost observed edge occurrence/endpoint: %+v", question)
	}
	if question["functionDataClaim"] != "not-established-by-external-structure" ||
		question["acceptedMeaning"] != false || question["ownerDecision"] != "pending" ||
		question["effect"] != "read-only" {
		t.Fatalf("question promoted imports into Function read: %+v", question)
	}
	packet := result["reviewPacket"].(map[string]any)
	coverage := packet["threeElementCoverage"].(map[string]any)
	for _, item := range coverage["elements"].([]any) {
		element := item.(map[string]any)
		if element["element"] == "function" && element["status"] != "unsupported" {
			t.Fatalf("function gained meaning from static graph: %+v", element)
		}
	}
	set, err := readREAComparisonCandidateSet(output)
	if err != nil {
		t.Fatalf("existing saved CandidateSet contract rejected: %v", err)
	}
	if len(set.Candidates) != 4 {
		t.Fatalf("question selection incorrectly replaced the full CandidateSet: %+v", set.Candidates)
	}
	if _, exists := map[string]any(result)["rawEvidenceFingerprint"]; !exists {
		t.Fatal("one-question result omitted the original raw-byte digest")
	}
}

func TestREACandidatesCLIRefusesUnobservedRelationBeforeWritingOutput(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "analysis.json")
	output := filepath.Join(dir, "must-not-create.json")
	if err := os.WriteFile(input, reaIntakeFixture(t), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, relationID := range []string{"imports", "missing-edge-id", "node-order"} {
		var out strings.Builder
		code, err := runThreeElementCommand([]string{
			"rea-candidates", "-evidence", input, "-relation-id", relationID, "-output", output,
		}, &out)
		if err != nil {
			t.Fatal(err)
		}
		response := decodeReaIntake(t, out.String())
		if code != 2 || response.OK || response.Error == nil ||
			response.Error.Code != "RELATION_SELECTION_UNAVAILABLE" {
			t.Fatalf("%q invented an available relation: code=%d response=%+v", relationID, code, response)
		}
		if _, err := os.Stat(output); !os.IsNotExist(err) {
			t.Fatalf("%q created output for unavailable relation: %v", relationID, err)
		}
	}
}

func TestREACandidatesCLIRefusesInvalidEvidenceWithoutOutput(t *testing.T) {
	dir := t.TempDir()
	evidencePath := filepath.Join(dir, "invalid.json")
	for _, tc := range []struct {
		name string
		raw  []byte
		code string
	}{
		{"bad-json", []byte("{not json"), "EVIDENCE_INVALID_JSON"},
		{"failure-envelope", []byte(`{"ok":false,"data":{}}`), "EVIDENCE_REPORTED_FAILURE"},
		{"missing-status", []byte(`{"data":{}}`), "EVIDENCE_STATUS_MISSING"},
		{"null", []byte("null"), "EVIDENCE_STATUS_MISSING"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			outPath := filepath.Join(dir, tc.name+"-candidates.json")
			if err := os.WriteFile(evidencePath, tc.raw, 0o600); err != nil {
				t.Fatal(err)
			}
			code, response := runREACandidatesCLI(t, evidencePath, outPath)
			if code != 2 || response.OK || response.Result != nil ||
				response.Error == nil || response.Error.Code != tc.code {
				t.Fatalf("invalid evidence accepted: exit=%d response=%+v", code, response)
			}
			if _, err := os.Stat(outPath); !os.IsNotExist(err) {
				t.Fatalf("invalid evidence created output: %v", err)
			}
		})
	}
	t.Run("missing-evidence-file", func(t *testing.T) {
		code, response := runREACandidatesCLI(t, filepath.Join(dir, "missing.json"), "")
		if code != 2 || response.Error == nil || response.Error.Code != "EVIDENCE_READ_FAILED" {
			t.Fatalf("missing source accepted: exit=%d response=%+v", code, response)
		}
	})
}

func TestREACandidatesCLIDoesNotOverwriteSourceOrPreviousOutput(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "analysis.json")
	raw := reaIntakeFixture(t)
	if err := os.WriteFile(input, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	code, response := runREACandidatesCLI(t, input, input)
	if code != 2 || response.OK || response.Error == nil ||
		response.Error.Code != "CANDIDATE_OUTPUT_WRITE_FAILED" {
		t.Fatalf("source overwrite was not refused: exit=%d response=%+v", code, response)
	}
	existing, err := os.ReadFile(input)
	if err != nil || !bytes.Equal(existing, raw) {
		t.Fatal("raw evidence bytes were modified by the projection command")
	}
	output := filepath.Join(dir, "candidates.json")
	if err := os.WriteFile(output, []byte("existing-owner-artifact"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, response = runREACandidatesCLI(t, input, output)
	if code != 2 || response.Error == nil ||
		response.Error.Code != "CANDIDATE_OUTPUT_WRITE_FAILED" {
		t.Fatalf("previous output was overwritten: exit=%d response=%+v", code, response)
	}
	current, err := os.ReadFile(output)
	if err != nil || string(current) != "existing-owner-artifact" {
		t.Fatal("previous output was modified")
	}
	t.Run("no-output-requested", func(t *testing.T) {
		code, response := runREACandidatesCLI(t, input, "")
		if code != 0 || !response.OK {
			t.Fatalf("read-only CLI refused input: exit=%d response=%+v", code, response)
		}
		if response.Result.(map[string]any)["candidateOutputWritten"] != false {
			t.Fatal("CLI claimed it wrote an unrequested candidate file")
		}
	})
}
