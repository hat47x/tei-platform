package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// One read-only CLI review must keep same-endpoint, same-label external edge
// occurrences distinguishable, including the in-memory adoption preview.
// This is a synthetic *application graph* test, not a semantic-graph mapping,
// Source write, Owner decision, or independent product Consumer.
func TestREAIntakeParallelRelationsStayUnacceptedInPreview(t *testing.T) {
	var document map[string]any
	if err := json.Unmarshal(reaIntakeFixture(t), &document); err != nil {
		t.Fatal(err)
	}
	data := document["data"].(map[string]any)
	normalized := data["normalized_result"].(map[string]any)
	graph := normalized["graph"].(map[string]any)
	edges := graph["edges"].([]any)
	first := edges[0].(map[string]any)
	second := make(map[string]any, len(first))
	for key, value := range first {
		second[key] = value
	}
	second["edge_id"] = "edge-order-line-parallel"
	graph["edges"] = append(edges, second)

	root := t.TempDir()
	modelPath := filepath.Join(root, "model.json")
	if err := os.WriteFile(modelPath, fixtureBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	run := func() (int, map[string]any) {
		t.Helper()
		raw, err := json.Marshal(document)
		if err != nil {
			t.Fatal(err)
		}
		evidencePath := filepath.Join(root, "analysis.json")
		if err := os.WriteFile(evidencePath, raw, 0o600); err != nil {
			t.Fatal(err)
		}
		var out strings.Builder
		exit, err := runThreeElementCommand([]string{"rea-intake", "-evidence", evidencePath, "-model", modelPath}, &out)
		if err != nil {
			t.Fatalf("CLI handler: %v", err)
		}
		var response map[string]any
		if err := json.Unmarshal([]byte(out.String()), &response); err != nil {
			t.Fatalf("CLI response not JSON: %v: %s", err, out.String())
		}
		return exit, response
	}
	exit, response := run()
	if exit != 0 || response["ok"] != true {
		t.Fatalf("parallel attempt was not reviewable: exit=%d response=%+v", exit, response)
	}
	packet := response["result"].(map[string]any)
	if packet["acceptedMeaning"] != false || packet["ownerDecision"] != "pending" ||
		packet["effect"] != "read-only" || packet["adoptionPreviewed"] != true {
		t.Fatalf("candidate review acquired unwarranted authority: %+v", packet)
	}

	relations := map[string]string{}
	for _, rawCandidate := range packet["candidates"].([]any) {
		candidate := rawCandidate.(map[string]any)
		if candidate["kind"] != "cross-reference" {
			continue
		}
		facts := candidate["facts"].(map[string]any)
		externalID := facts["externalIdentity"].(string)
		if externalID != "edge-order-line" && externalID != "edge-order-line-parallel" {
			t.Fatalf("unexpected external relation identity %q", externalID)
		}
		if facts["externalOccurrenceId"] != externalID {
			t.Fatalf("external occurrence changed: %+v", facts)
		}
		if _, ok := relations[externalID]; ok {
			t.Fatalf("duplicate candidate for external edge %q", externalID)
		}
		relations[externalID] = candidate["candidateId"].(string)
		notes := candidate["notes"].([]any)
		has := func(want string) bool {
			for _, value := range notes {
				if value == want {
					return true
				}
			}
			return false
		}
		if !has("parallel-relation") || !has("same-endpoints-same-label") {
			t.Fatalf("parallel ambiguity was erased: edge=%q notes=%v", externalID, notes)
		}
		inference := candidate["inference"].(map[string]any)
		if inference["targetElement"] != "cross-reference" {
			t.Fatalf("external relation silently became accepted meaning: %+v", inference)
		}
		if candidate["sourceRevision"] == nil {
			t.Fatalf("source backing disappeared: edge=%q", externalID)
		}
	}
	if len(relations) != 2 || relations["edge-order-line"] == relations["edge-order-line-parallel"] {
		t.Fatalf("separate edge occurrences collapsed: %+v", relations)
	}

	// The preview uses candidate IDs for *provisional* CrossReference IDs.
	// The model and its original sources remain unmodified and unaccepted.
	adoption := packet["adoption"].(map[string]any)
	after := adoption["after"].(map[string]any)
	previewIDs := map[string]bool{}
	for _, rawRef := range after["crossReferences"].([]any) {
		ref := rawRef.(map[string]any)
		previewIDs[ref["id"].(string)] = true
	}
	for edgeID, candidateID := range relations {
		if !previewIDs[candidateID] {
			t.Fatalf("preview omitted edge %s (candidate %s)", edgeID, candidateID)
		}
	}
	for _, rawRecord := range adoption["records"].([]any) {
		record := rawRecord.(map[string]any)
		if record["accepted"] == true || (record["owner"] != nil && record["owner"] != "") {
			t.Fatalf("preview assigned owner acceptance: %+v", record)
		}
	}
	carrier := packet["carrier"].(map[string]any)
	gapFound := false
	for _, rawGap := range carrier["missing"].([]any) {
		gap := rawGap.(map[string]any)
		if gap["fact"] == "external-edge-identity" {
			gapFound = true
			if gap["present"] != float64(2) {
				t.Fatalf("carrier gap lost both external edge occurrences: %+v", gap)
			}
		}
	}
	if !gapFound {
		t.Fatal("read-only carrier no longer reports the missing external-edge-ID destination")
	}

	// An ambiguous producer input is not resolved by selecting either occurrence.
	second["edge_id"] = first["edge_id"]
	exit, refused := run()
	if exit != 2 || refused["ok"] == true {
		t.Fatalf("duplicate producer edge ID was silently accepted: exit=%d result=%v", exit, refused)
	}
	errorInfo := refused["error"].(map[string]any)
	if errorInfo["code"] != "EDGE_IDENTITY_DUPLICATE" {
		t.Fatalf("wrong duplicate-ID refusal: %+v", errorInfo)
	}
}
