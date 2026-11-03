package orderruledesign

import (
	"strings"
	"testing"
)

func sampleChain(t *testing.T) *Chain {
	t.Helper()
	c := NewChain()
	un := &Unresolved{Total: 3, SourceObservation: 1, Mapping: 1, SemanticCandidate: 1}
	base := Carried{SourceDigest: "sha256:s", ContractDigest: "sha256:c", ReceiptFingerprint: "sha256:r", AccountingFingerprint: "sha256:a",
		ProvenanceState: StateBound, Unresolved: un, EvidenceFingerprint: "sha256:e"}
	for stage := 1; stage <= 6; stage++ {
		switch stage {
		case 2:
			base.BaselineRevision, base.TargetsDigest = "sha256:b0", "sha256:t"
		case 3:
			base.CandidateDigest, base.CandidateFingerprint, base.AttachmentDigest = "sha256:cd", "sha256:cf", "sha256:at"
		case 4:
			base.MainCommit, base.TaskHead, base.TaskTree, base.IntegrationHead, base.IntegrationTree = "m", "h", "tr", "h", "it"
		case 5:
			base.AssuranceDigest, base.TrialRevision, base.ImpactRangeDigest = "sha256:as", "sha256:b1", "sha256:ir"
		case 6:
			base.ReviewFingerprint, base.PacketDigest, base.ReasonDigest, base.MessageDigest, base.SignatureDigest = "sha256:rf", "sha256:pk", "sha256:rs", "sha256:ms", "sha256:sg"
		}
		if err := c.Append(stage, base, map[string]any{"n": stage}); err != nil {
			t.Fatal(err)
		}
	}
	return c
}

func TestThroughCompletesOnlyWhenEveryIdentityIsOneValue(t *testing.T) {
	c := sampleChain(t)
	if th := VerifyThrough(*c); !th.Complete {
		t.Fatalf("a consistent chain is not complete: %+v", th)
	}
	forged := forge(*c, 6, func(x *Carried) { x.CandidateFingerprint = "sha256:other" })
	if !standaloneOK(forged) {
		t.Fatal("a relinked chain must pass each record's own check")
	}
	if th := VerifyThrough(forged); th.Complete {
		t.Fatal("a replaced identity went unnoticed")
	}
	reduced := forge(*c, 3, func(x *Carried) { x.Unresolved = &Unresolved{} })
	if VerifyThrough(reduced).Complete {
		t.Fatal("a reduced unresolved count went unnoticed")
	}
	cut := c.Clone()
	cut.Records[2].Carried.ProvenanceState = StateNoTarget
	if VerifyThrough(cut).Complete {
		t.Fatal("an edit without relinking went unnoticed")
	}
}

func TestMatrixRefusesNoFindingOnlyAndMissingScope(t *testing.T) {
	var rows []CheckerRow
	for _, n := range requiredCheckers {
		rows = append(rows, CheckerRow{Checker: n, Class: ClassEvaluated, Verdict: "x", Basis: "b"})
	}
	if p := VerifyMatrix(rows); len(p) != 0 {
		t.Fatalf("a full list is refused: %v", p)
	}
	for i := range rows {
		rows[i].Verdict = "no-finding"
	}
	if p := VerifyMatrix(rows); !anyHas(p, "only collects") {
		t.Fatalf("a no-finding-only list passed: %v", p)
	}
	rows[0].Class, rows[1].Class = ClassOutOfScope, ClassInsufficient
	if SoundnessBasis(rows) != len(rows)-2 {
		t.Fatal("an out-of-scope row counted as a basis")
	}
	if p := VerifyMatrix(rows[1:]); !anyHas(p, "not listed") {
		t.Fatalf("a list without a checker passed: %v", p)
	}
}

func TestImpactRangeRefusesOverclaims(t *testing.T) {
	r := BuildImpactRange("b0", "cf", "b1", assuranceOut{Action: "recheck"},
		[]Derivation{{"screen", executionTargets[0].Ref, "cf"}, {"server", executionTargets[1].Ref, "cf"}}, inFlightCases("b0", mainDispositions),
		[]ReuseCandidate{{"e", "reuse-candidate", false, false}})
	if p := VerifyImpactRange(r, inFlightCases("b0", nil)); len(p) != 0 {
		t.Fatalf("a sound range is refused: %v", p)
	}
	if got := r.Targets[2]; got.Derived || !got.Recheck || !strings.Contains(got.Reason, "not-derived") {
		t.Fatalf("the underived target is not in the recheck range: %+v", got)
	}
	r.Reuse[0].Pass = true
	if p := VerifyImpactRange(r, nil); len(p) == 0 {
		t.Fatal("a reuse candidate read as a pass was accepted")
	}
}

func TestSourceVariantsGiveTheThreeStates(t *testing.T) {
	for _, tc := range []struct {
		src  string
		want int
	}{{sourceBound, 1}, {sourceTableOnly, 1}, {sourceNoTarget, 0}} {
		ords, err := tableOrdinals([]byte(tc.src), TargetTable)
		if err != nil || len(ords) != tc.want {
			t.Fatalf("ordinals %v err %v, want %d", ords, err, tc.want)
		}
	}
	if !ioReaches([]byte(sourceBound), TargetTable) || ioReaches([]byte(sourceTableOnly), TargetTable) {
		t.Fatal("the io path check does not tell the two materials apart")
	}
}
