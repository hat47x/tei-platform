package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestStudioBridgeShoDocumentUsesTheRealConsumerCommand is the end-to-end
// Studio half of the integration: the bridge invokes the REAL `bin/sho-doc.cjs`
// from a SHO checkout, pointed at by SHO_DOC_COMMAND, and relays what it says.
//
// The test is opt-in because it needs a SHO checkout with its dependencies
// installed. Ordinary CI runs the stubbed relay tests instead; this one proves
// the wiring against the real component boundary.
func TestStudioBridgeShoDocumentUsesTheRealConsumerCommand(t *testing.T) {
	shoRoot := os.Getenv("STUDIO_SHO_ROOT")
	if shoRoot == "" {
		t.Skip("STUDIO_SHO_ROOT is not set")
	}
	command := filepath.Join(shoRoot, "bin", "sho-doc.cjs")
	if _, err := os.Stat(command); err != nil {
		t.Fatalf("SHO document command is missing: %v", err)
	}
	// The bridge reads the command from the environment, exactly as the real
	// Studio build does. The test sets it rather than assuming it is inherited.
	t.Setenv("SHO_DOC_COMMAND", command)
	dir := t.TempDir()

	// One draft, written independently of both products: a minimal valid
	// envelope with a single explanation occurrence.
	draftPath := filepath.Join(dir, "draft.json")
	draft := map[string]any{
		"schemaVersion": "tei.document-draft/v0",
		"effect":        "read-only-projection",
		"draftId":       "sha256:" + repeat("a", 64),
		"documentKind":  "element-explanation",
		"title":         "studio integration",
		"model":         map[string]string{"modelRevision": "sha256:" + repeat("d", 64)},
		"subjects": []map[string]any{{
			"subject": map[string]string{"kind": "semantic-data", "id": "sales.order"},
			"sources": []map[string]string{{
				"artifactPath": "semantic/sales/order.tei",
				"fingerprint":  "sha256:" + repeat("e", 64),
			}},
		}},
		"occurrences": []map[string]any{{
			"occurrenceId": "occ-explanation-1",
			"kind":         "explanation",
			"subjectRefs":  []map[string]string{{"kind": "semantic-data", "id": "sales.order"}},
			"generationConditions": map[string]any{
				"generator": "tei.internal.documentdraft", "generatorVersion": "v0",
				"rule":         "element-summary-v0",
				"inputDigests": map[string]string{"modelRevision": "sha256:" + repeat("d", 64)},
			},
			"content":       map[string]string{"text": "sales.order は semantic-data として宣言されたデータ要素です。"},
			"contentDigest": "sha256:" + repeat("1", 64),
		}},
		"authorityLimits": []string{"consumer-save-is-not-tei-model-ingestion"},
	}
	encoded, err := json.Marshal(draft)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(draftPath, encoded, 0600); err != nil {
		t.Fatal(err)
	}
	documentPath := filepath.Join(dir, "document.md")

	bridge := &StudioBridge{}
	opened := bridge.OpenShoDocument(draftPath, documentPath)
	if !opened.OK {
		t.Fatalf("open failed: %+v", opened.Failure)
	}
	var openReceipt struct {
		Kind   string `json:"kind"`
		Status string `json:"status"`
		Claims struct {
			TeiModelIngested       bool `json:"teiModelIngested"`
			AcceptedMeaningUpdated bool `json:"acceptedMeaningUpdated"`
		} `json:"claims"`
	}
	if err := json.Unmarshal(opened.Result, &openReceipt); err != nil {
		t.Fatalf("result is not the consumer's receipt: %v", err)
	}
	if openReceipt.Kind != "sho-draft-open" || openReceipt.Status != "created" {
		t.Fatalf("receipt = %+v", openReceipt)
	}
	if openReceipt.Claims.TeiModelIngested || openReceipt.Claims.AcceptedMeaningUpdated {
		t.Fatal("the consumer claimed model ingestion or meaning update")
	}

	// Resume reads the same document back through the same boundary.
	resumed := bridge.ResumeShoDocument(documentPath)
	if !resumed.OK {
		t.Fatalf("resume failed: %+v", resumed.Failure)
	}
	var resumeReceipt struct {
		Kind        string `json:"kind"`
		Status      string `json:"status"`
		Occurrences []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"occurrences"`
	}
	if err := json.Unmarshal(resumed.Result, &resumeReceipt); err != nil {
		t.Fatalf("result is not the consumer's receipt: %v", err)
	}
	if resumeReceipt.Kind != "sho-draft-resume" || resumeReceipt.Status != "read" {
		t.Fatalf("receipt = %+v", resumeReceipt)
	}
	if len(resumeReceipt.Occurrences) != 1 ||
		resumeReceipt.Occurrences[0].Status != "intact" {
		t.Fatalf("occurrences = %+v", resumeReceipt.Occurrences)
	}

	// A command that is not there is a stable failure, not an empty document.
	// Node reports a missing script itself, so the code is the child's non-zero
	// exit rather than Go's spawn failure; either is a refusal, never a result.
	t.Setenv("SHO_DOC_COMMAND", filepath.Join(dir, "absent.cjs"))
	missing := (&StudioBridge{}).ResumeShoDocument(documentPath)
	if missing.OK || missing.Failure == nil {
		t.Fatalf("response = %+v failure=%+v", missing, missing.Failure)
	}
	if missing.Failure.Code != ShoSpawnFailed && missing.Failure.Code != ShoNonZeroExit {
		t.Fatalf("code = %q", missing.Failure.Code)
	}
	if missing.Result != nil {
		t.Fatalf("a failure carried a document state: %s", missing.Result)
	}
}

func repeat(value string, count int) string {
	out := ""
	for i := 0; i < count; i++ {
		out += value
	}
	return out
}
