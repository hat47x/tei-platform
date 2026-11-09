package validation_test

import (
	"errors"
	"testing"

	"tei/internal/validation"
)

func testProposals() []validation.Diagnostic {
	return []validation.Diagnostic{
		{Code: "SEMANTIC_ATTRIBUTE_ROLE_UNSUPPORTED", Severity: validation.Warning, Message: "role is not used"},
		{Code: "PERIOD_CLOSED_REJECT_UNKNOWN", Severity: validation.Error, Message: "closed-rejects names an unknown kind"},
	}
}

// TestProposeNeverAccepts checks that producing a proposal is not a decision:
// every proposal starts unanswered even when the diagnostic is an error.
func TestProposeNeverAccepts(t *testing.T) {
	proposals := validation.Propose(testProposals(), "rev-1")
	if len(proposals) != 2 {
		t.Fatalf("proposals=%+v", proposals)
	}
	for _, proposal := range proposals {
		if proposal.Accepted {
			t.Fatalf("proposal was accepted without a human decision: %+v", proposal)
		}
		if proposal.Revision != "rev-1" {
			t.Fatalf("revision=%q", proposal.Revision)
		}
	}
}

// TestAcceptRequiresTheAcceptedRevision checks that a proposal read from another
// revision cannot be accepted against the current accepted source.
func TestAcceptRequiresTheAcceptedRevision(t *testing.T) {
	proposals := validation.Propose(testProposals(), "rev-1")
	proposal := &proposals[0]
	if err := validation.Accept(proposal, "rev-1"); err != nil {
		t.Fatalf("same revision: %v", err)
	}
	if !proposal.Accepted {
		t.Fatal("proposal was not marked accepted")
	}

	other := &validation.Propose(testProposals(), "rev-1")[0]
	err := validation.Accept(other, "rev-2")
	var mismatch *validation.RevisionMismatchError
	if !errors.As(err, &mismatch) {
		t.Fatalf("changed revision err=%v", err)
	}
	if other.Accepted {
		t.Fatal("a changed revision accepted the proposal")
	}
}

// TestAcceptRefusesMissingRevision checks that neither side may omit the
// revision, so a proposal cannot be accepted by default.
func TestAcceptRefusesMissingRevision(t *testing.T) {
	proposal := &validation.Propose(testProposals(), "")[0]
	if err := validation.Accept(proposal, "rev-1"); err == nil {
		t.Fatal("proposal without a revision was accepted")
	}
	proposal = &validation.Propose(testProposals(), "rev-1")[0]
	if err := validation.Accept(proposal, ""); err == nil {
		t.Fatal("acceptance without a revision was allowed")
	}
	if proposal.Accepted {
		t.Fatal("missing revision marked the proposal accepted")
	}
}

// TestAcceptRefusesRepeatedAcceptance checks that repeating a call cannot widen
// or restate a decision that a person already made.
func TestAcceptRefusesRepeatedAcceptance(t *testing.T) {
	proposal := &validation.Propose(testProposals(), "rev-1")[0]
	if err := validation.Accept(proposal, "rev-1"); err != nil {
		t.Fatalf("first accept: %v", err)
	}
	if err := validation.Accept(proposal, "rev-1"); err == nil {
		t.Fatal("repeated acceptance was allowed")
	}
}
