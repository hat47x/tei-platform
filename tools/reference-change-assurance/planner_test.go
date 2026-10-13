package main

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestExampleRecommendations(t *testing.T) {
	b, err := os.ReadFile("example.json")
	if err != nil {
		t.Fatal(err)
	}
	var f Fixture
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatal(err)
	}
	report, err := evaluate(f)
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]string{
		"display-only-rename/semantic-rule":         "reuse-candidate",
		"display-only-rename/ui-display":            "recheck",
		"threshold-change/approval-boundary":        "recheck",
		"threshold-change/independent-order-intake": "reuse-candidate",
		"provider-replaced/business-meaning":        "reuse-candidate",
		"provider-replaced/provider-conformance":    "recheck",
		"opaque-native-effect/effect-conformance":   "unresolved",
		"runtime-authority-revoked/static-shape":    "reuse-candidate",
		"unknown-aspect/rule":                       "unresolved",
		"target-drift/target-e2e":                   "recheck",
		"self-derived-oracle/fixture-expectation":   "recheck",
	}
	if len(report.Decisions) != len(expected) {
		t.Fatalf("got %d decisions, want %d", len(report.Decisions), len(expected))
	}
	for _, d := range report.Decisions {
		key := d.CaseID + "/" + d.ClaimID
		if want, ok := expected[key]; !ok || d.Action != want || d.ExecutionAllowed || len(d.ReasonCodes) == 0 {
			t.Errorf("invalid decision for %s: %#v", key, d)
		}
		if d.Action == "reuse-candidate" && d.ReceiptRef == "" {
			t.Errorf("candidate without receipt: %s", key)
		}
	}
	var auth Decision
	for _, d := range report.Decisions {
		if d.CaseID == "runtime-authority-revoked" {
			auth = d
		}
	}
	if !reflect.DeepEqual(auth.RuntimeChecks, []string{"authority.current", "effect.fence"}) {
		t.Errorf("runtime checks lost: %#v", auth)
	}
}

func TestStrictJSONAndDeterminism(t *testing.T) {
	original, err := os.ReadFile("example.json")
	if err != nil {
		t.Fatal(err)
	}
	var first bytes.Buffer
	if err := run(bytes.NewReader(original), &first); err != nil {
		t.Fatal(err)
	}
	var second bytes.Buffer
	if err := run(bytes.NewReader(original), &second); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatal("not deterministic")
	}
	for _, raw := range []string{
		`{"cases":[],"extra":true}`,
		`{"cases":[]} true`,
		`{"cases":[]}`,
		`{"cases":[{"id":"x","purpose":"simulate","claims":[]}]}`,
	} {
		if err := run(strings.NewReader(raw), &bytes.Buffer{}); err == nil {
			t.Errorf("accepted invalid json: %s", raw)
		}
	}
}

func TestUnknownChangeAspectNeverSilentlyIgnored(t *testing.T) {
	c := Claim{ID: "x", Coverage: "complete", Dependencies: []string{"subject:x"}, CoveredAspects: []string{"display", "meaning"}, ImpactAspects: []string{"meaning"}, CurrentContext: map[string]string{"meaning": "v1"}, Receipts: []Receipt{{ID: "receipt", Outcome: "pass", OracleOrigin: "independent", Context: map[string]string{"meaning": "v1"}}}}
	got := assess("case", "simulate", c, []Change{{Subject: "subject:x", Aspects: []string{"effect"}}})
	if got.Action != "unresolved" {
		t.Errorf("unexpected decision: %#v", got)
	}
	// A declared unrelated display change, with a complete dependency closure,
	// is the only case where this synthetic receipt can be a reuse candidate.
	got = assess("case", "simulate", c, []Change{{Subject: "subject:x", Aspects: []string{"display"}}})
	if got.Action != "reuse-candidate" {
		t.Errorf("unexpected decision: %#v", got)
	}
}

func TestRejectMalformedClaimOrUntrustedOracle(t *testing.T) {
	c := Claim{ID: "x", Coverage: "complete", Dependencies: []string{"subject:x"}, CoveredAspects: []string{"meaning"}, ImpactAspects: []string{"meaning"}, CurrentContext: map[string]string{"meaning": "v1"}, Receipts: []Receipt{{ID: "r", Outcome: "pass", OracleOrigin: "unknown", Context: map[string]string{"meaning": "v1"}}}}
	f := Fixture{Cases: []Case{{ID: "case", Purpose: "simulate", Claims: []Claim{c}}}}
	r, err := evaluate(f)
	if err != nil || r.Decisions[0].Action != "recheck" {
		t.Errorf("untrusted oracle accepted: %v, %#v", err, r)
	}
	f.Cases[0].Claims[0].ImpactAspects = []string{"not-covered"}
	if _, err = evaluate(f); err == nil {
		t.Fatal("invalid impact aspects accepted")
	}
}
