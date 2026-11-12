package main

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

func sample() input {
	return input{
		Schema: inputSchema, Scope: "pdp-public", BaseRevision: "R0", CandidateRevision: "R1",
		DeclaredScopeClosed: true,
		Nodes:               []node{{"pdp.banner", "content"}, {"pdp.layout", "presentation"}, {"pdp.offer-rule", "business"}, {"pdp.consent", "consent"}},
		Required:            []string{"pdp.banner", "pdp.layout", "pdp.offer-rule", "pdp.consent"},
		Changed:             []string{"pdp.banner"},
	}
}

func TestContentOnlyIsScopedAndNotDeploymentPermission(t *testing.T) {
	v, err := inspect(sample())
	if err != nil {
		t.Fatal(err)
	}
	if v.Coverage != "declared-closed-only" || v.Disposition != "no-known-critical-impact-within-declared-scope" || len(v.Lanes) != 1 || v.Lanes[0] != "C0" {
		t.Fatalf("wrong bounded claim: %#v", v)
	}
	if len(v.AuthorityLimits) < 3 || !strings.Contains(strings.Join(v.AuthorityLimits, " "), "no-release-grant") {
		t.Fatalf("missing authority limit: %#v", v)
	}
}

func TestContentLabelCanReachBusinessAndConsent(t *testing.T) {
	v := sample()
	v.Edges = []edge{{"pdp.banner", "pdp.offer-rule"}, {"pdp.offer-rule", "pdp.consent"}}
	r, err := inspect(v)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(r.Lanes, ",") != "C0,C3" || r.Disposition != "known-interaction-business-or-release-impact" {
		t.Fatalf("business escalation lost: %#v", r)
	}
}

func TestUnobservedNativeOrMissingNodesCannotCertifyClosure(t *testing.T) {
	v := sample()
	v.Unobserved = []string{"external-cms-rule"}
	v.Edges = []edge{{"pdp.banner", "dynamic-import"}}
	r, err := inspect(v)
	if err != nil {
		t.Fatal(err)
	}
	if r.Coverage != "partial-or-unverified" || r.Disposition != "incomplete-scope-needs-review" || strings.Join(r.Unresolved, ",") != "dynamic-import,external-cms-rule" {
		t.Fatalf("false completeness: %#v", r)
	}
}

func TestCyclesDoNotLoopAndDuplicateIdentityRejects(t *testing.T) {
	v := sample()
	v.Edges = []edge{{"pdp.banner", "pdp.layout"}, {"pdp.layout", "pdp.banner"}}
	r, err := inspect(v)
	if err != nil || strings.Join(r.Impacted, ",") != "pdp.banner,pdp.layout" || strings.Join(r.Lanes, ",") != "C0,C1" {
		t.Fatalf("cycle: %#v err %v", r, err)
	}
	v.Nodes = append(v.Nodes, node{ID: "pdp.banner", Layer: "business"})
	if _, err := inspect(v); err == nil {
		t.Fatal("duplicate ID silently shadowed")
	}
}

func TestUnknownChangeAndOpenScopeAreNotSafe(t *testing.T) {
	v := sample()
	v.Changed = []string{"unmapped-native-route"}
	v.DeclaredScopeClosed = false
	r, err := inspect(v)
	if err != nil || r.Disposition != "incomplete-scope-needs-review" || len(r.Unresolved) != 1 {
		t.Fatalf("unsafe unknown: %#v %v", r, err)
	}
}

func TestStrictInputAndStableReport(t *testing.T) {
	v := sample()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := run(bytes.NewReader(b), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"schema": "`+reportSchema+`"`) {
		t.Fatal(out.String())
	}
	if err := run(strings.NewReader(string(b)+` {}`), &out); err == nil {
		t.Fatal("trailing JSON accepted")
	}
	if err := run(strings.NewReader(`{"schema":"`+inputSchema+`","unexpected":1}`), &out); err == nil {
		t.Fatal("unrecognized input was silently discarded")
	}
}

func TestRequiredUnknownLayerDoesNotBecomeClosedCoverage(t *testing.T) {
	v := sample()
	v.Nodes[2].Layer = "unknown"
	r, err := inspect(v)
	if err != nil {
		t.Fatal(err)
	}
	if r.Coverage != "partial-or-unverified" || strings.Join(r.Unresolved, ",") != "pdp.offer-rule" {
		t.Fatalf("unknown classification was counted as covered: %#v", r)
	}
}

func TestFixedFixtureMatchesIndependentExpected(t *testing.T) {
	b, err := os.ReadFile("example-input.json")
	if err != nil {
		t.Fatal(err)
	}
	expected, err := os.ReadFile("example-expected.json")
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := run(bytes.NewReader(b), &out); err != nil {
		t.Fatal(err)
	}
	var got, want any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(expected, &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("fixture != independent expected\nactual: %s\nexpected: %s", out.String(), string(expected))
	}
}

func TestUnrelatedUnknownDoesNotInvalidateQuestion(t *testing.T) {
	v := sample()
	// These nodes and the malformed edge are outside the explicitly
	// required universe and outside all paths from the changed subject.
	v.Nodes = append(v.Nodes,
		node{ID: "unrelated.native", Layer: "unknown"},
		node{ID: "unrelated.page", Layer: "presentation"},
	)
	v.Edges = []edge{{From: "external.unseen", To: "unrelated.native"}}
	r, err := inspect(v)
	if err != nil {
		t.Fatal(err)
	}
	if r.Coverage != "declared-closed-only" ||
		r.Disposition != "no-known-critical-impact-within-declared-scope" ||
		len(r.Unresolved) != 0 || strings.Join(r.Impacted, ",") != "pdp.banner" {
		t.Fatalf("unrelated unknown over-invalidated the requested change: %+v", r)
	}
	// The bounded report still cannot grant release or world-wide safety.
	if !strings.Contains(strings.Join(r.AuthorityLimits, ","), "unverified-claims") {
		t.Fatalf("missing closure claim limit: %+v", r)
	}
}

func TestRequiredUnknownEndpointPreventsFalseNoImpact(t *testing.T) {
	for _, edges := range [][]edge{
		{{From: "pdp.offer-rule", To: "unresolved.price"}},
		{{From: "unresolved.price", To: "pdp.offer-rule"}},
	} {
		v := sample()
		v.Edges = edges
		r, err := inspect(v)
		if err != nil {
			t.Fatal(err)
		}
		if r.Coverage != "partial-or-unverified" ||
			r.Disposition != "incomplete-scope-needs-review" ||
			strings.Join(r.Unresolved, ",") != "unresolved.price" {
			t.Fatalf("required unresolved dependency was omitted: %+v", r)
		}
	}
}

func TestClosedClaimWithoutRequiredUniverseIsNotComplete(t *testing.T) {
	v := sample()
	v.Required = nil
	r, err := inspect(v)
	if err != nil {
		t.Fatal(err)
	}
	if r.Coverage != "partial-or-unverified" ||
		r.Disposition != "incomplete-scope-needs-review" {
		t.Fatalf("empty required universe certified closure: %+v", r)
	}
	v.Required = []string{""}
	if _, err := inspect(v); err == nil {
		t.Fatal("empty required ID accepted")
	}
}

func TestKnownBusinessImpactRetainsUnknownCoverage(t *testing.T) {
	v := sample()
	v.Edges = []edge{
		{From: "pdp.banner", To: "pdp.offer-rule"},
		{From: "pdp.offer-rule", To: "dynamic.price-rule"},
	}
	r, err := inspect(v)
	if err != nil {
		t.Fatal(err)
	}
	if r.Coverage != "partial-or-unverified" ||
		r.Disposition != "known-interaction-business-or-release-impact" ||
		strings.Join(r.Lanes, ",") != "C0,C3" ||
		strings.Join(r.Unresolved, ",") != "dynamic.price-rule" {
		t.Fatalf("known impact and unknown closure were conflated: %+v", r)
	}
}
