package operationdriftcheck

import (
	"testing"

	"tei/internal/operations"
)

// surface builds a minimal surface. Only the three fields this package reads
// are set; the rest stay empty so a test failure points at the comparison.
func surface(world, revision string) operations.OperationSurface {
	return operations.OperationSurface{
		SchemaVersion:    operations.OperationSurfaceSchemaVersion,
		Effect:           "read-only",
		Application:      "ledger",
		WorldFingerprint: world,
		ModelRevision:    revision,
	}
}

func TestAMatchingExpectationIsCurrent(t *testing.T) {
	report := Check(surface("sha256:aaa", "sha256:bbb"), Expectation{
		WorldFingerprint: "sha256:aaa",
		ModelRevision:    "sha256:bbb",
	})
	if report.Status != StatusCurrent {
		t.Fatalf("status = %q, want %q (%+v)", report.Status, StatusCurrent, report)
	}
	if len(report.Findings) != 2 {
		t.Fatalf("findings = %+v", report.Findings)
	}
	for _, finding := range report.Findings {
		if !finding.Match {
			t.Fatalf("finding reported a mismatch: %+v", finding)
		}
	}
	if report.Precondition() != "" {
		t.Fatalf("a current surface produced a stop line: %q", report.Precondition())
	}
}

func TestAChangedWorldIsStaleAndStops(t *testing.T) {
	report := Check(surface("sha256:new", "sha256:bbb"), Expectation{
		WorldFingerprint: "sha256:old",
		ModelRevision:    "sha256:bbb",
	})
	if report.Status != StatusStale {
		t.Fatalf("status = %q, want %q", report.Status, StatusStale)
	}
	if report.Diagnostic != StaleOperationSurface {
		t.Fatalf("diagnostic = %q", report.Diagnostic)
	}
	if report.Findings[0].Reason == "" {
		t.Fatal("a mismatch must explain itself")
	}
	if report.Precondition() == "" {
		t.Fatal("a stale surface must produce a stop line")
	}
}

func TestAChangedSourceSetIsStale(t *testing.T) {
	report := Check(surface("sha256:aaa", "sha256:new"), Expectation{
		WorldFingerprint: "sha256:aaa",
		ModelRevision:    "sha256:old",
	})
	if report.Status != StatusStale || report.Findings[1].Value != "modelRevision" {
		t.Fatalf("report = %+v", report)
	}
}

func TestNoExpectationIsUncheckedAndNotAPass(t *testing.T) {
	report := Check(surface("sha256:aaa", "sha256:bbb"), Expectation{})
	if report.Status != StatusUnchecked {
		t.Fatalf("status = %q, want %q", report.Status, StatusUnchecked)
	}
	if len(report.Findings) != 0 {
		t.Fatalf("unchecked must compare nothing, got %+v", report.Findings)
	}
	// An unchecked surface is not current and must not read as one.
	if report.Status == StatusCurrent {
		t.Fatal("an unchecked surface was reported as current")
	}
}

func TestOnlyTheSuppliedValueIsCompared(t *testing.T) {
	// The caller checked the World and left the revision unset. A stale
	// revision must not be invented into a mismatch.
	report := Check(surface("sha256:aaa", "sha256:new"), Expectation{WorldFingerprint: "sha256:aaa"})
	if report.Status != StatusCurrent {
		t.Fatalf("status = %q, want current; an unset value must not be compared", report.Status)
	}
	if len(report.Findings) != 1 || report.Findings[0].Value != "worldFingerprint" {
		t.Fatalf("findings = %+v", report.Findings)
	}
}

func TestBasisIsNotACounterAndTheReportSaysSo(t *testing.T) {
	// Two surfaces that differ only in Basis are indistinguishable to this
	// check: Basis is per persistence mapping, not the whole meaning.
	a := surface("sha256:aaa", "sha256:bbb")
	b := surface("sha256:aaa", "sha256:bbb")
	a.Operations = []operations.Operation{{ID: "x", Kind: "list", Basis: "sha256:one"}}
	b.Operations = []operations.Operation{{ID: "x", Kind: "list", Basis: "sha256:two"}}
	reportA := Check(a, Expectation{WorldFingerprint: "sha256:aaa", ModelRevision: "sha256:bbb"})
	reportB := Check(b, Expectation{WorldFingerprint: "sha256:aaa", ModelRevision: "sha256:bbb"})
	if reportA.Status != StatusCurrent || reportB.Status != StatusCurrent {
		t.Fatalf("basis alone changed the verdict: %q / %q", reportA.Status, reportB.Status)
	}
	joined := reportA.Notes[0] + reportA.Notes[1] + reportA.Notes[2] + reportA.Notes[3]
	if joined == "" {
		t.Fatal("the report must state what it does not establish")
	}
}

func TestReportStatesAuthorityAndIsDeterministic(t *testing.T) {
	a := Check(surface("sha256:aaa", "sha256:bbb"), Expectation{WorldFingerprint: "sha256:aaa"})
	b := Check(surface("sha256:aaa", "sha256:bbb"), Expectation{WorldFingerprint: "sha256:aaa"})
	if a.RuleFingerprint == "" || a.RuleFingerprint != b.RuleFingerprint {
		t.Fatalf("rule fingerprint = %q / %q", a.RuleFingerprint, b.RuleFingerprint)
	}
	if a.Authority != Authority || a.Schema != Schema {
		t.Fatalf("authority = %q schema = %q", a.Authority, a.Schema)
	}
	if len(a.Findings) != len(b.Findings) || a.Findings[0] != b.Findings[0] {
		t.Fatal("the comparison is not repeatable")
	}
}
