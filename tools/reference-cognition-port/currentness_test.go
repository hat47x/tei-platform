package cognitionport_test

import (
	"testing"

	port "tei/tools/reference-cognition-port"
)

func choiceRequest(subject, revision, statement string, options []port.ChoiceOption, criteria ...string) port.Request {
	return port.Request{
		Schema:  port.RequestSchema,
		ID:      "req-1",
		Kind:    "choice",
		Profile: port.ProfileOffline,
		Context: port.Context{
			WorkspaceRevision: revision,
			Observations:      []port.Observation{{ID: subject, Statement: statement}},
			Constraints:       criteria,
			Choice:            &port.ChoiceQuestion{ObservationID: subject, Options: options},
		},
	}
}

func billingOptions() []port.ChoiceOption {
	return []port.ChoiceOption{
		{ID: "billing", Description: "請求・支払い"},
		{ID: "contract", Description: "契約内容・契約変更"},
	}
}

// receiptFor builds a receipt the way the port does, so the test exercises the
// same binding the runtime writes rather than a hand-made struct.
func receiptFor(t *testing.T, request port.Request) port.Receipt {
	t.Helper()
	basis, err := port.BasisDigest(request)
	if err != nil {
		t.Fatal(err)
	}
	return port.Receipt{
		Schema:    port.ReceiptSchema,
		RequestID: request.ID,
		Kind:      request.Kind,
		Profile:   request.Profile,
		Binding:   port.BindingOf(request, basis),
		State:     port.StateProposed,
		Proposal: &port.Proposal{
			Schema: port.ProposalSchema,
			State:  port.StateProposed,
			Candidates: []port.Candidate{{
				ID:      "choice-1",
				Kind:    "choice",
				Subject: request.Context.Choice.ObservationID,
				Payload: map[string]any{"optionId": "billing"},
				Basis:   []string{"observation:" + request.Context.Choice.ObservationID},
			}},
		},
	}
}

func TestRequireCurrentDistinguishesTheReasons(t *testing.T) {
	criteria := []string{"主な目的を分類する"}
	produced := choiceRequest("inquiry", "rev-1", "請求書の宛名が違う", billingOptions(), criteria...)
	receipt := receiptFor(t, produced)
	basis, err := port.BasisDigest(produced)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("unchanged binding is current but still not acceptable", func(t *testing.T) {
		result := port.RequireCurrent(receipt, produced, basis)
		if result.State != port.RecheckCurrent {
			t.Fatalf("state = %s, want %s", result.State, port.RecheckCurrent)
		}
		if result.Acceptable || result.GrantsAuthority {
			t.Fatal("a current candidate must not be reported as accepted or authoritative")
		}
	})

	t.Run("moved revision is stale", func(t *testing.T) {
		// A -> B: the observation text changed in the same subject.
		next := choiceRequest("inquiry", "rev-2", "請求書の宛名が違う", billingOptions(), criteria...)
		if result := port.RequireCurrent(receipt, next, basis); result.State != port.RecheckStaleRevision {
			t.Fatalf("state = %s, want %s", result.State, port.RecheckStaleRevision)
		}
		// A -> B -> A: the revision marker returned to its old value, so the
		// revision alone cannot recover the stale candidate; only a re-proposal
		// against the current value may be shown.
		if result := port.RequireCurrent(receipt, produced, basis); result.State != port.RecheckCurrent {
			t.Fatal("the same revision must re-check as current; the caller must still re-propose for a fresh candidate")
		}
	})

	t.Run("revised criteria are stale even at the same revision", func(t *testing.T) {
		revised := choiceRequest("inquiry", "rev-1", "請求書の宛名が違う", billingOptions(), "契約分類の基準を改訂")
		nowBasis, err := port.BasisDigest(revised)
		if err != nil {
			t.Fatal(err)
		}
		if result := port.RequireCurrent(receipt, revised, nowBasis); result.State != port.RecheckStaleBasis {
			t.Fatalf("state = %s, want %s", result.State, port.RecheckStaleBasis)
		}
	})

	t.Run("a removed option is stale basis", func(t *testing.T) {
		// The option set shrinks; the old candidate cites an observation that
		// still exists, but the basis digest moved.
		narrowed := choiceRequest("inquiry", "rev-1", "請求書の宛名が違う",
			[]port.ChoiceOption{{ID: "contract", Description: "契約内容・契約変更"}, {ID: "other", Description: "その他"}}, criteria...)
		nowBasis, err := port.BasisDigest(narrowed)
		if err != nil {
			t.Fatal(err)
		}
		if result := port.RequireCurrent(receipt, narrowed, nowBasis); result.State != port.RecheckStaleBasis {
			t.Fatalf("state = %s, want %s", result.State, port.RecheckStaleBasis)
		}
	})

	t.Run("another subject is never current", func(t *testing.T) {
		other := choiceRequest("inquiry-2", "rev-1", "別の問い合わせ", billingOptions(), criteria...)
		otherBasis, err := port.BasisDigest(other)
		if err != nil {
			t.Fatal(err)
		}
		if result := port.RequireCurrent(receipt, other, otherBasis); result.State != port.RecheckForeignSubject {
			t.Fatalf("state = %s, want %s", result.State, port.RecheckForeignSubject)
		}
	})

	t.Run("no proposal is reported as such, not as current", func(t *testing.T) {
		empty := receipt
		empty.Proposal = nil
		if result := port.RequireCurrent(empty, produced, basis); result.State != port.RecheckNoProposal {
			t.Fatalf("state = %s, want %s", result.State, port.RecheckNoProposal)
		}
	})
}

// The basis digest must be sensitive to the option set and the criteria, and
// insensitive to the observation text: a changed statement is a revision change,
// not a basis change, and reporting both as "stale" would hide which one moved.
func TestBasisDigestCoversOptionsAndCriteriaOnly(t *testing.T) {
	base := choiceRequest("inquiry", "rev-1", "請求書の宛名が違う", billingOptions(), "主な目的を分類する")
	baseDigest, err := port.BasisDigest(base)
	if err != nil {
		t.Fatal(err)
	}

	reworded := choiceRequest("inquiry", "rev-1", "請求金額が想定と違う", billingOptions(), "主な目的を分類する")
	rewordedDigest, err := port.BasisDigest(reworded)
	if err != nil {
		t.Fatal(err)
	}
	if baseDigest != rewordedDigest {
		t.Fatal("a changed observation statement must not change the basis digest")
	}

	addedCriterion := choiceRequest("inquiry", "rev-1", "請求書の宛名が違う", billingOptions(), "主な目的を分類する", "曖昧ならunknown")
	addedDigest, err := port.BasisDigest(addedCriterion)
	if err != nil {
		t.Fatal(err)
	}
	if baseDigest == addedDigest {
		t.Fatal("an added criterion must change the basis digest")
	}
}
