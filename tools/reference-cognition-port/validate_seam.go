package cognitionport

// ValidateProposalForTest exposes the port's own proposal check to a reference
// comparison harness.
//
// The comparison tool must judge a provider's output by the same rule the port
// applies when it accepts a proposal into a receipt. Re-implementing that check
// in the harness would let the two drift, and a harness that is more lenient
// than the port would report a format-violation rate the port does not share.
//
// This is a reference tool's seam, not a product API: production composes the
// check inside ProposeWithInvoker, where no caller can skip it.
func ValidateProposalForTest(proposal Proposal, request Request) error {
	return validateProposal(proposal, request)
}
