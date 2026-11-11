package reacandidate

import (
	"strings"
	"testing"
)

// The three-element model has Data, Function and Business. An external static
// analyzer supports a data-shaped candidate; naming a Function or a Business
// element from it would be inference the attempt does not support. The table
// must say so per element instead of leaving a reviewer to guess.
func TestThreeElementCoverageSupportsOnlyData(t *testing.T) {
	set := mustGenerate(t, defaultEvidence())
	coverage := EvaluateThreeElementCoverage(set)
	if coverage.SchemaVersion != "tei.rea.three-element-coverage/v0" || coverage.Effect != EffectReadOnly {
		t.Fatalf("envelope = %q %q", coverage.SchemaVersion, coverage.Effect)
	}
	if len(coverage.Elements) != 3 {
		t.Fatalf("elements = %d, want 3", len(coverage.Elements))
	}
	byElement := map[string]ElementCoverage{}
	for _, element := range coverage.Elements {
		byElement[element.Element] = element
	}
	data := byElement["data"]
	if data.Status != ElementSupported {
		t.Fatalf("data = %+v", data)
	}
	if data.Candidates != len(set.Candidates) {
		t.Fatalf("data candidates = %d, want %d", data.Candidates, len(set.Candidates))
	}
	if !hasString(data.SupportedBy, "source-backed-node-observations") ||
		!hasString(data.SupportedBy, "directed-external-relations") {
		t.Fatalf("data support = %v", data.SupportedBy)
	}
	for _, name := range []string{"function", "business"} {
		element := byElement[name]
		if element.Status != ElementUnsupported {
			t.Fatalf("%s = %+v", name, element)
		}
		if element.Reason == "" {
			t.Fatalf("%s has no reason", name)
		}
		if len(element.WouldNeed) == 0 {
			t.Fatalf("%s has no wouldNeed", name)
		}
	}
	if !strings.Contains(byElement["function"].Reason, "imports") {
		t.Fatalf("function reason does not name the labels considered: %q", byElement["function"].Reason)
	}
	if !strings.Contains(byElement["business"].Reason, "Owner") {
		t.Fatalf("business reason does not point at the Owner: %q", byElement["business"].Reason)
	}
	if !hasString(coverage.AuthorityLimits, "unsupported-element-is-not-an-absence-of-that-element-in-the-system") {
		t.Fatalf("authority limits = %v", coverage.AuthorityLimits)
	}
}

// An attempt with no observation and no relation supports nothing, and the
// Data row says why instead of claiming support with zero candidates.
func TestThreeElementCoverageEmptyAttemptSupportsNothing(t *testing.T) {
	fixture := defaultEvidence()
	fixture.nodes = nil
	fixture.edges = nil
	set := mustGenerate(t, fixture)
	coverage := EvaluateThreeElementCoverage(set)
	byElement := map[string]ElementCoverage{}
	for _, element := range coverage.Elements {
		byElement[element.Element] = element
	}
	data := byElement["data"]
	if data.Status != ElementUnsupported {
		t.Fatalf("data = %+v", data)
	}
	if data.Reason == "" {
		t.Fatal("data has no reason for the empty attempt")
	}
	if data.Candidates != 0 {
		t.Fatalf("data candidates = %d", data.Candidates)
	}
}

// The packet carries the coverage table and the summary states it, so a
// reviewer reading only the summary still sees that Function and Business were
// not produced.
func TestReviewPacketCarriesThreeElementCoverage(t *testing.T) {
	set := mustGenerate(t, defaultEvidence())
	packet := BuildReviewPacket(set, nil, nil, nil)
	if len(packet.ThreeElement.Elements) != 3 {
		t.Fatalf("packet coverage = %+v", packet.ThreeElement)
	}
	joined := strings.Join(packet.Summary(), "\n")
	for _, want := range []string{
		"three-element: data=supported",
		"three-element: function=unsupported",
		"three-element: business=unsupported",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("summary is missing %q:\n%s", want, joined)
		}
	}
	if packet.AcceptedMeaning {
		t.Fatal("coverage flipped an acceptance")
	}
}

// REAL REA names relation labels such as contains/loads/exposes. Those are
// static relationships, so they may appear in the Function reason as
// "considered", but they must never turn the Function row into supported.
func TestThreeElementCoverageRelationLabelsDoNotSupportFunction(t *testing.T) {
	fixture := defaultEvidence()
	fixture.edges = []edgeFixture{
		{id: "e1", from: "node-order", to: "node-line", rel: "contains", state: "observed", artifact: digestA, locPath: "src/order.js"},
		{id: "e2", from: "node-line", to: "node-order", rel: "exposes", state: "inferred", artifact: digestA, locPath: "src/line.js"},
	}
	set := mustGenerate(t, fixture)
	coverage := EvaluateThreeElementCoverage(set)
	for _, element := range coverage.Elements {
		if element.Element != "function" {
			continue
		}
		if element.Status != ElementUnsupported {
			t.Fatalf("function became %q from relation labels", element.Status)
		}
		reason := element.Reason
		if !strings.Contains(reason, "contains") || !strings.Contains(reason, "exposes") {
			t.Fatalf("reason = %q", reason)
		}
		return
	}
	t.Fatal("function row missing")
}
