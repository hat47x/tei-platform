package facetmappingcheck

import (
	"encoding/json"
	"testing"
)

// TestContractIsMachineReadable pins the self-description: a consumer, including
// a generating model, must be able to learn the vocabulary and the verdict
// precedence from the report itself rather than from prose in a README.
func TestContractIsMachineReadable(t *testing.T) {
	desc := ContractDescription()
	if len(desc.Columns) != 12 {
		t.Fatalf("the contract has %d columns, want 12", len(desc.Columns))
	}
	for _, want := range []string{"source", "target", "identity", "order", "multiset",
		"grain", "presence", "coverage", "quantity", "time", "loss", "operation"} {
		found := false
		for _, col := range desc.Columns {
			if col == want {
				found = true
			}
		}
		if !found {
			t.Errorf("contract is missing column %q", want)
		}
	}
	if len(desc.Verdicts) != 5 {
		t.Fatalf("the contract lists %d verdicts, want the fixed 5", len(desc.Verdicts))
	}
	if len(desc.Losses) == 0 || len(desc.Inverses) == 0 {
		t.Fatal("contract must list the loss and inverse vocabularies")
	}
	if len(desc.VerdictRules) != len(desc.Verdicts) {
		t.Errorf("every verdict needs one precedence rule; got %d rules for %d verdicts",
			len(desc.VerdictRules), len(desc.Verdicts))
	}
	if len(desc.NotClaims) == 0 {
		t.Error("a contract without explicit non-claims invites overreach")
	}

	// The report must carry the contract, so one JSON document is enough.
	out := Check(CaseFile{SchemaVersion: InputSchema, Cases: []Case{{ID: "empty"}}}, Options{})
	if out.Contract == nil {
		t.Fatal("the report does not embed the contract")
	}
	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("report is not serialisable: %v", err)
	}
	var round Output
	if err := json.Unmarshal(raw, &round); err != nil {
		t.Fatalf("report does not round-trip: %v", err)
	}
	if round.Contract == nil || len(round.Contract.Columns) != 12 {
		t.Fatal("contract did not survive serialisation")
	}
}

// TestLossVocabularyMatchesChecks keeps the published vocabulary and the
// implementation from drifting apart: every loss the contract advertises must be
// producible by at least one rule, and every non-invertible entry must be in the
// advertised list.
func TestLossVocabularyMatchesChecks(t *testing.T) {
	desc := ContractDescription()
	known := map[string]bool{}
	for _, l := range desc.Losses {
		known[l] = true
	}
	for loss := range nonInvertibleLosses {
		if !known[loss] {
			t.Errorf("non-invertible loss %q is not in the published vocabulary", loss)
		}
	}
	for _, loss := range []string{LossOrder, LossGrain, LossPresenceStates, LossCoverage,
		LossSourceRevision, LossContextIdentity, LossQuantityScale, LossTimeAxisRole,
		LossSelectionIdentity, LossInjectivity, LossFanout} {
		if !known[loss] {
			t.Errorf("loss %q is produced by a rule but missing from the vocabulary", loss)
		}
	}
}
