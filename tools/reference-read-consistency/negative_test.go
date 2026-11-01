package readconsistency

import "testing"

func d11Base() Input {
	return Input{
		RequiredLevel:     LevelSameTransactionSnapshot,
		ProviderGuarantee: LevelSameTransactionSnapshot,
		Operation:         OpSubtract,
		Reads:             reads("120", "R0", "100", "R0"),
	}
}

// The same input set yields a confirmed value under a strong requirement and an
// explicitly non-atomic result only when the case accepts best-effort.
func TestSameInputSetDependsOnRequiredLevel(t *testing.T) {
	strong := d11Base()
	strong.Reads = reads("120", "R0", "90", "R1")
	strongReport := Evaluate(strong)
	if strongReport.Accepted || strongReport.Code != CodeMixedSnapshot || strongReport.Mixed != "30" {
		t.Fatalf("strong report=%+v want MIXED_SNAPSHOT 30 not accepted", strongReport)
	}

	weak := Input{
		RequiredLevel:    LevelBestEffort,
		AcceptBestEffort: true,
		Operation:        OpSubtract,
		Reads:            reads("120", "R0", "90", "R1"),
	}
	weakReport := Evaluate(weak)
	if !weakReport.Accepted || weakReport.Code != CodeNonAtomicRead || weakReport.Result != "30" {
		t.Fatalf("weak report=%+v want NON_ATOMIC_READ 30", weakReport)
	}
}

// Absent relation, missing input, NULL, mixed revision, and snapshot loss are
// separate diagnostics; none is folded into another.
func TestFailureDiagnosticsAreDistinct(t *testing.T) {
	seen := map[string]bool{}
	add := func(in Input) {
		code := Evaluate(in).Code
		if seen[code] {
			t.Fatalf("diagnostics are not distinct: %q repeated", code)
		}
		seen[code] = true
	}

	lost := d11Base()
	lost.SnapshotTokenLost = true
	add(lost)

	missing := d11Base()
	missing.Reads[1].State = ReadMissing
	add(missing)

	absent := d11Base()
	absent.Reads[1].State = ReadAbsentRelation
	add(absent)

	nullRead := d11Base()
	nullRead.Reads[1].State = ReadNull
	add(nullRead)

	mixed := d11Base()
	mixed.Reads[1].Revision = "R1"
	add(mixed)

	want := []string{CodeInputUnavailable, CodeInputMissing, CodeAbsentRelation, CodeNullInput, CodeMixedSnapshot}
	if len(seen) != len(want) {
		t.Fatalf("seen=%v want %v", seen, want)
	}
	for _, code := range want {
		if !seen[code] {
			t.Fatalf("missing diagnostic %q in %v", code, seen)
		}
	}
}

// A lost snapshot and an unreadable input are both INPUT_UNAVAILABLE, but a
// normal NULL and a missing input stay distinct from them.
func TestSnapshotLossIsNotANormalNull(t *testing.T) {
	lost := d11Base()
	lost.SnapshotTokenLost = true
	if report := Evaluate(lost); report.Code != CodeInputUnavailable || report.Accepted {
		t.Fatalf("lost snapshot report=%+v", report)
	}

	nullRead := d11Base()
	nullRead.Reads[1].State = ReadNull
	if report := Evaluate(nullRead); report.Code != CodeNullInput {
		t.Fatalf("null report=%+v", report)
	}
}
