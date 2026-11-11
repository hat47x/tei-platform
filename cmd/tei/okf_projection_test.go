package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

// TestOKFBundleProjectionIsCandidateOnly covers the -project flag: the projection is
// a proposal list, never an acceptance, and its absence is the default.
func TestOKFBundleProjectionIsCandidateOnly(t *testing.T) {
	var baseline bytes.Buffer
	if code, err := okfBundleJSON(&baseline, okfFixturePath, "", "", "", "", 0, 0, 0, false, 0); err != nil || code != 0 {
		t.Fatalf("baseline: code=%d err=%v", code, err)
	}
	envelope, _ := decodeOKFEnvelope(t, baseline)
	result := envelope["result"].(map[string]any)
	if _, present := result["projection"]; present {
		t.Fatal("projection must be absent without -project")
	}

	var projected bytes.Buffer
	if code, err := okfBundleJSON(&projected, okfFixturePath, "", "", "", "", 0, 0, 0, true, 0); err != nil || code != 0 {
		t.Fatalf("projected: code=%d err=%v", code, err)
	}
	projectedEnvelope, _ := decodeOKFEnvelope(t, projected)
	projectionRaw, present := projectedEnvelope["result"].(map[string]any)["projection"]
	if !present {
		t.Fatal("projection must be present with -project")
	}
	projection := projectionRaw.(map[string]any)
	if projection["admission"] != "candidate" {
		t.Fatalf("admission = %v, want candidate", projection["admission"])
	}
	if projection["profile"] != "read-only-candidates" {
		t.Fatalf("profile = %v", projection["profile"])
	}
	candidates := projection["candidates"].([]any)
	if len(candidates) == 0 {
		t.Fatal("the fixture must produce candidates")
	}
	for _, raw := range candidates {
		candidate := raw.(map[string]any)
		if candidate["admission"] != "candidate" {
			t.Fatalf("candidate %v admission = %v", candidate["proposedId"], candidate["admission"])
		}
		if candidate["source"].(map[string]any)["artifactDigest"] == "" {
			t.Fatalf("candidate %v has no digest", candidate["proposedId"])
		}
	}
	if len(projection["losses"].([]any)) == 0 {
		t.Fatal("the loss manifest must not be empty")
	}
	if len(projection["notProposed"].([]any)) == 0 {
		t.Fatal("the projection must state what it does not propose")
	}
}

// TestOKFBundleProjectionKeepsFormatVerdict covers the separation: adding a projection
// does not change the format verdict, and the reader's notEvaluated list survives.
func TestOKFBundleProjectionKeepsFormatVerdict(t *testing.T) {
	var out bytes.Buffer
	if code, err := okfBundleJSON(&out, okfFixturePath, "", "", "", "", 0, 0, 0, true, 0); err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	envelope, _ := decodeOKFEnvelope(t, out)
	result := envelope["result"].(map[string]any)
	if result["format"].(map[string]any)["conformant"] != true {
		t.Fatal("the fixture stays conformant with -project")
	}
	notEvaluated := result["notEvaluated"].(map[string]any)["items"].([]any)
	if len(notEvaluated) == 0 {
		t.Fatal("notEvaluated must survive the projection")
	}
	// The projection must not carry an acceptance anywhere in its JSON.
	var raw map[string]any
	body, err := json.Marshal(result["projection"])
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatal(err)
	}
	if _, present := raw["accepted"]; present {
		t.Fatal("the projection must not carry an acceptance field")
	}
}

// TestOKFBundleProjectionIsAdvertised covers discovery: -project appears in the
// advertised flag set so a caller can find it without reading prose.
func TestOKFBundleProjectionIsAdvertised(t *testing.T) {
	record := catalogRecord(t, "okf-bundle")
	for _, flag := range advertisedFlags(record) {
		if flag == "project" {
			return
		}
	}
	t.Fatalf("advertised flags %v lack project", advertisedFlags(record))
}

// TestOKFBundleProjectionWithConceptMode covers the combination: a single-concept
// read may also carry a projection, and the projection still describes the whole
// bundle it was derived from.
func TestOKFBundleProjectionWithConceptMode(t *testing.T) {
	var out bytes.Buffer
	if code, err := okfBundleJSON(&out, okfFixturePath, "data/order", "", "", "", 0, 0, 0, true, 0); err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	envelope, _ := decodeOKFEnvelope(t, out)
	if envelope["mode"] != resultModeConcept {
		t.Fatalf("mode = %v", envelope["mode"])
	}
	result := envelope["result"].(map[string]any)
	projection := result["projection"].(map[string]any)
	candidates := projection["candidates"].([]any)
	if len(candidates) != 6 {
		t.Fatalf("candidates = %d, want the whole bundle's six", len(candidates))
	}
	revision := projection["bundleRevision"].(map[string]any)["value"]
	if revision != result["bundleRevision"].(map[string]any)["value"] {
		t.Fatalf("the projection revision %v differs from the read revision", revision)
	}
}
