package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

func parseWorld(t *testing.T, content string) World {
	t.Helper()
	var v World
	if err := json.Unmarshal([]byte(content), &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func fixture(t *testing.T) (World, Facts) {
	t.Helper()
	w := parseWorld(t, `{"semantics":[{"data":[{"id":"sales.order"},{"id":"stock.reservation"}]}],"applications":[{"patterns":[{"id":"sales.submit"},{"id":"stock.reserve"},{"id":"sales.list"}]}]}`)
	f := Facts{
		SchemaVersion:  factsSchema,
		Owners:         []Owner{{Data: "sales.order", Subsystem: "sales", Source: "owner:sales.order"}, {Data: "stock.reservation", Subsystem: "stock", Source: "owner:stock.reservation"}},
		FunctionIO:     []FunctionIO{{Pattern: "sales.submit", Subsystem: "sales", Data: "sales.order", Operation: "U", Source: "function:sales.submit"}, {Pattern: "stock.reserve", Subsystem: "stock", Data: "stock.reservation", Operation: "U", Source: "function:stock.reserve"}, {Pattern: "sales.list", Subsystem: "sales", Data: "stock.reservation", Operation: "R", Source: "function:sales.list"}},
		ProcedureSteps: []ProcedureStep{{ID: "confirm-by-phone", Subsystem: "sales", Realization: "manual", Source: "procedure:confirm-by-phone"}, {ID: "submit", Subsystem: "sales", Realization: "system", Pattern: "sales.submit", Source: "procedure:submit"}, {ID: "notify", Subsystem: "sales", Realization: "external", Source: "procedure:notify"}},
	}
	return w, f
}

func codeCount(r Report, code string) int {
	count := 0
	for _, d := range r.Diagnostics {
		if d.Code == code {
			count++
		}
	}
	return count
}

func TestOwnerSideDelegationAndCrossSubsystemReadsAllowed(t *testing.T) {
	w, f := fixture(t)
	r := Inspect(w, f)
	if r.Status != "no-findings-in-declared-scope" || len(r.Diagnostics) != 0 {
		t.Fatalf("unexpected: %+v", r)
	}
	if r.AuthorityLimit != authorityLimit {
		t.Fatal("report must limit its authority")
	}
}

func TestCrossSubsystemWriteRequiresReview(t *testing.T) {
	w, f := fixture(t)
	f.FunctionIO = append(f.FunctionIO, FunctionIO{Pattern: "sales.submit", Subsystem: "sales", Data: "stock.reservation", Operation: "D", Source: "function:unsafe"})
	r := Inspect(w, f)
	if r.Status != "review-needed" || codeCount(r, "CROSS_SUBSYSTEM_WRITE_INTENT") != 1 {
		t.Fatalf("unexpected: %+v", r)
	}
}

func TestUnknownDataAndAmbiguousOwnerStayUnresolved(t *testing.T) {
	w, f := fixture(t)
	f.Owners = append(f.Owners, f.Owners[0])
	f.FunctionIO = append(f.FunctionIO, FunctionIO{Pattern: "sales.submit", Subsystem: "sales", Data: "unknown.data", Operation: "U", Source: "function:unknown"})
	r := Inspect(w, f)
	if r.Status != "unresolved" || codeCount(r, "AMBIGUOUS_DATA_OWNER") != 1 || codeCount(r, "IO_DATA_UNRESOLVED") != 1 {
		t.Fatalf("unexpected: %+v", r)
	}
}

func TestNoPatternInferenceOrManualFabrication(t *testing.T) {
	w, f := fixture(t)
	f.ProcedureSteps = append(f.ProcedureSteps, ProcedureStep{ID: "submit", Subsystem: "sales", Realization: "manual", Pattern: "sales.submit", Source: "procedure:bad"})
	f.FunctionIO = append(f.FunctionIO, FunctionIO{Pattern: "made.up", Subsystem: "sales", Data: "sales.order", Operation: "C", Source: "function:bad"})
	r := Inspect(w, f)
	for _, code := range []string{"AMBIGUOUS_PROCEDURE_STEP", "MANUAL_PATTERN_CONFLICT", "IO_PATTERN_UNRESOLVED"} {
		if codeCount(r, code) != 1 {
			t.Fatalf("missing %s: %+v", code, r)
		}
	}
}

func TestUnrecognizedOperationCannotProducePartialAuthority(t *testing.T) {
	w, f := fixture(t)
	f.FunctionIO[0].Operation = "CU"
	r := Inspect(w, f)
	if r.Status != "unresolved" || codeCount(r, "UNKNOWN_OPERATION") != 1 {
		t.Fatalf("unexpected: %+v", r)
	}
}

func TestEmptyAndDuplicateWorldDataNotAccepted(t *testing.T) {
	w, f := fixture(t)
	w.Semantics[0].Data = append(w.Semantics[0].Data, w.Semantics[0].Data[0])
	r := Inspect(w, f)
	if codeCount(r, "AMBIGUOUS_CANONICAL_DATA") != 1 || r.Status != "unresolved" {
		t.Fatalf("unexpected: %+v", r)
	}
}

func TestSchemaMismatchCannotBeReadAsClean(t *testing.T) {
	w, f := fixture(t)
	f.SchemaVersion = "old"
	r := Inspect(w, f)
	if codeCount(r, "UNSUPPORTED_FACTS_SCHEMA") != 1 || r.Status != "unresolved" {
		t.Fatalf("unexpected: %+v", r)
	}
}

func TestOutputStableAcrossFactOrder(t *testing.T) {
	w, f := fixture(t)
	f.FunctionIO = append(f.FunctionIO,
		FunctionIO{Pattern: "sales.submit", Subsystem: "sales", Data: "stock.reservation", Operation: "D", Source: "bad2"},
		FunctionIO{Pattern: "sales.list", Subsystem: "sales", Data: "stock.reservation", Operation: "C", Source: "bad1"})
	a := Inspect(w, f)
	f.FunctionIO[3], f.FunctionIO[4] = f.FunctionIO[4], f.FunctionIO[3]
	b := Inspect(w, f)
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("order instability: %v vs %v", a, b)
	}
}

func TestEmptyCandidateScopeCannotReportClean(t *testing.T) {
	w, _ := fixture(t)
	r := Inspect(w, Facts{SchemaVersion: factsSchema})
	if r.Status != "unresolved" || codeCount(r, "EMPTY_CANDIDATE_SCOPE") != 1 {
		t.Fatalf("unexpected: %+v", r)
	}
	if r.Coverage.WorldData != 2 || r.Coverage.OwnerFacts != 0 {
		t.Fatalf("incorrect coverage: %+v", r.Coverage)
	}
}

func TestUnresolvedPatternDoesNotInventCrossSubsystemWriteFinding(t *testing.T) {
	w, f := fixture(t)
	f.FunctionIO = append(f.FunctionIO, FunctionIO{Pattern: "unknown.submit", Subsystem: "sales", Data: "stock.reservation", Operation: "U", Source: "candidate:unknown"})
	r := Inspect(w, f)
	if r.Status != "unresolved" || codeCount(r, "IO_PATTERN_UNRESOLVED") != 1 || codeCount(r, "CROSS_SUBSYSTEM_WRITE_INTENT") != 0 {
		t.Fatalf("bad: %+v", r)
	}
	w.Applications[0].Patterns = append(w.Applications[0].Patterns, w.Applications[0].Patterns[0])
	f.FunctionIO[len(f.FunctionIO)-1].Pattern = "sales.submit"
	r = Inspect(w, f)
	if r.Status != "unresolved" || codeCount(r, "AMBIGUOUS_APPLICATION_PATTERN") != 1 || codeCount(r, "IO_PATTERN_UNRESOLVED") != 2 || codeCount(r, "CROSS_SUBSYSTEM_WRITE_INTENT") != 0 {
		t.Fatalf("bad: %+v", r)
	}
}
func TestExactInputDigestIsContentBoundNotAcceptance(t *testing.T) {
	a := snapshotDigest([]byte(`{"a":1}`))
	b := snapshotDigest([]byte(`{"a":2}`))
	if a != "sha256:015abd7f5cc57a2dd94b7590f04ad8084273905ee33ec5cebeae62276a97f862" || a == b {
		t.Fatal(a, b)
	}
}
func TestSnapshotBindingRejectsMissingOrStaleInputs(t *testing.T) {
	raw := []byte(`{"semantics":[{"data":[{"id":"sales.order"},{"id":"stock.reservation"}]}],"applications":[{"patterns":[{"id":"sales.submit"},{"id":"stock.reserve"},{"id":"sales.list"}]}]}`)
	w, f := fixture(t)
	f.WorldJSONFingerprint = snapshotDigest(raw)
	factsBytes, _ := json.Marshal(f)
	good := InspectSnapshot(w, f, raw, factsBytes)
	if good.Status != "no-findings-in-declared-scope" || good.InputFingerprints.WorldJSON != f.WorldJSONFingerprint || good.InputFingerprints.FactsJSON != snapshotDigest(factsBytes) {
		t.Fatalf("bound: %+v", good)
	}
	bad := InspectSnapshot(w, f, append(append([]byte(nil), raw...), byte(' ')), factsBytes)
	if bad.Status != "unresolved" || codeCount(bad, "WORLD_SNAPSHOT_STALE") != 1 {
		t.Fatalf("stale: %+v", bad)
	}
	f.WorldJSONFingerprint = ""
	unbound := InspectSnapshot(w, f, raw, factsBytes)
	if unbound.Status != "unresolved" || codeCount(unbound, "WORLD_SNAPSHOT_UNBOUND") != 1 {
		t.Fatalf("unbound: %+v", unbound)
	}
}
