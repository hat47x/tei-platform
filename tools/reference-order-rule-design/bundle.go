package orderruledesign

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"tei/tools/internal/opensshwitness"
)

const bundleSchema = "tei.reference.order-rule-design.submission-bundle/v0"

// SubmissionBundle is the one value this sample hands to an acceptance
// path: the candidate version, its provenance and the signed decision. It
// is a type of its own so that the acceptance path behind it can be
// replaced. Today it is tools/reference-shared-acceptance, which the
// sample calls for preview and review only. When the preceding sample
// `order-workflow` reaches main, a Submitter that hands this bundle to its
// common acceptance replaces sharedAcceptanceSubmitter. The bundle does not
// change; the points to map are listed in the README.
type SubmissionBundle struct {
	Schema               string          `json:"schema"`
	Target               string          `json:"target"`
	CandidateJSON        json.RawMessage `json:"candidate"`
	CandidateDigest      string          `json:"candidateDigest"`
	CandidateFingerprint string          `json:"candidateFingerprint"`
	BaselineRevision     string          `json:"baselineRevision"`
	Provenance           Attachment      `json:"provenance"`
	ReviewFingerprint    string          `json:"reviewFingerprint"`
	PacketDigest         string          `json:"packetDigest"`
	Decision             SignedDecision  `json:"decision"`
	Acceptance           AcceptanceState `json:"acceptance"`
}

// AcceptanceState says what was done with the bundle. Performed is false
// in this sample for every path: nothing is accepted or rejected.
type AcceptanceState struct {
	Performed bool   `json:"performed"`
	Status    string `json:"status"`
	Reason    string `json:"reason"`
}

// SignedDecision is a decision signed with OpenSSH. Decision "accept" here
// is the decider's sign-off on submitting the candidate. It does not write
// any source and does not stand for an acceptance record.
type SignedDecision struct {
	Payload    opensshwitness.Payload  `json:"payload"`
	Principal  string                  `json:"principal"`
	Namespace  string                  `json:"namespace"`
	Signature  []byte                  `json:"signature"`
	Evidence   opensshwitness.Evidence `json:"evidence"`
	Reason     Reason                  `json:"reason"`
	ReasonText string                  `json:"reasonText"`
}

// Reason is what the signature's reason digest covers.
type Reason struct {
	ProvenanceState     string     `json:"provenanceState"`
	Unresolved          Unresolved `json:"unresolved"`
	SourceDigest        string     `json:"sourceDigest"`
	ContractDigest      string     `json:"contractDigest"`
	EvidenceFingerprint string     `json:"evidenceFingerprint"`
	AssuranceDigest     string     `json:"assuranceDigest"`
	ImpactRangeDigest   string     `json:"impactRangeDigest"`
	MatrixDigest        string     `json:"matrixDigest"`
	InFlight            int        `json:"inFlightCases"`
	Undecided           int        `json:"undecidedCases"`
}

// SubmissionReceipt is what a Submitter reports.
type SubmissionReceipt struct {
	Submitter           string   `json:"submitter"`
	Mode                string   `json:"mode"`
	Accepted            bool     `json:"accepted"`
	RecomputedCandidate string   `json:"recomputedCandidateFingerprint"`
	RecomputedReview    string   `json:"recomputedReviewFingerprint"`
	RecomputedBase      string   `json:"recomputedBaseRevision"`
	Problems            []string `json:"problems"`
	DecisionsUnchanged  bool     `json:"decisionLedgerUnchanged"`
}

// Submitter hands a bundle to an acceptance path and reports what the path
// recomputed. It never accepts.
type Submitter interface {
	Name() string
	Submit(ctx context.Context, b SubmissionBundle) (SubmissionReceipt, error)
}

// sharedAcceptanceSubmitter submits to tools/reference-shared-acceptance on
// a disposable copy of the baseline, with preview and review only.
type sharedAcceptanceSubmitter struct {
	s        *Sample
	Baseline string
	Work     string
}

func (x sharedAcceptanceSubmitter) Name() string { return "tools/reference-shared-acceptance" }

func (x sharedAcceptanceSubmitter) Submit(ctx context.Context, b SubmissionBundle) (SubmissionReceipt, error) {
	rc := SubmissionReceipt{Submitter: x.Name(), Mode: "preview-and-review-only", Accepted: false}
	scratch := filepath.Join(x.Work, "submit-scratch")
	_ = os.RemoveAll(scratch)
	if err := copyTree(x.Baseline, scratch); err != nil {
		return rc, err
	}
	before, err := treeDigest(filepath.Join(scratch, ".tei"))
	if err != nil {
		return rc, err
	}
	cand := filepath.Join(x.Work, "submit-candidate.json")
	if err := writeFile(cand, b.CandidateJSON); err != nil {
		return rc, err
	}
	rev, err := x.s.revisionOf(ctx, scratch)
	if err != nil {
		return rc, err
	}
	rc.RecomputedBase = rev
	prev, err := x.s.Tools.mustRun(ctx, x.s.Tools.SharedAcceptance, x.s.Repo, "preview", "-workspace", scratch, "-candidate", cand)
	if err != nil {
		return rc, err
	}
	rv, err := x.s.Tools.mustRun(ctx, x.s.Tools.SharedAcceptance, x.s.Repo, "review", "-workspace", scratch, "-candidate", cand, "-format", "json")
	if err != nil {
		return rc, err
	}
	var p struct {
		CandidateFingerprint string `json:"candidateFingerprint"`
	}
	var r struct {
		ReviewFingerprint string `json:"reviewFingerprint"`
	}
	_ = json.Unmarshal(prev, &p)
	_ = json.Unmarshal(rv, &r)
	rc.RecomputedCandidate, rc.RecomputedReview = p.CandidateFingerprint, r.ReviewFingerprint
	after, _ := treeDigest(filepath.Join(scratch, ".tei"))
	rc.DecisionsUnchanged = before == after
	pl := b.Decision.Payload
	if rc.RecomputedCandidate != pl.CandidateFingerprint || rc.RecomputedCandidate != b.CandidateFingerprint {
		rc.Problems = append(rc.Problems, "candidate fingerprint recomputed by the acceptance path differs from the signed value")
	}
	if rc.RecomputedReview != pl.ReviewFingerprint || rc.RecomputedReview != b.ReviewFingerprint {
		rc.Problems = append(rc.Problems, "review fingerprint recomputed by the acceptance path differs from the signed value")
	}
	if rc.RecomputedBase != pl.BaseModelRevision || rc.RecomputedBase != b.BaselineRevision {
		rc.Problems = append(rc.Problems, "base revision recomputed by the acceptance path differs from the signed value")
	}
	if !rc.DecisionsUnchanged {
		rc.Problems = append(rc.Problems, "the submission wrote to the decision ledger")
	}
	return rc, nil
}

// notConnectedSubmitter stands for order-workflow, which is not on main.
type notConnectedSubmitter struct{}

func (notConnectedSubmitter) Name() string { return "order-workflow common acceptance" }

func (notConnectedSubmitter) Submit(context.Context, SubmissionBundle) (SubmissionReceipt, error) {
	return SubmissionReceipt{Submitter: "order-workflow", Mode: "not-connected", Accepted: false,
		Problems: []string{"order-workflow is on a branch that is not merged into main; acceptance is not connected"}}, nil
}

// VerifyBundle checks that a bundle says one thing: the payload, the
// candidate, the provenance and the reason agree, and nothing is accepted.
func VerifyBundle(b SubmissionBundle) []string {
	var out []string
	add := func(f string, a ...any) { out = append(out, fmt.Sprintf(f, a...)) }
	if b.Schema != bundleSchema {
		add("schema %q", b.Schema)
	}
	if digestBytes(b.CandidateJSON) != b.CandidateDigest {
		add("candidate bytes do not match the candidate digest")
	}
	pl := b.Decision.Payload
	if pl.CandidateFingerprint != b.CandidateFingerprint {
		add("signed candidate fingerprint differs from the bundle")
	}
	if pl.ReviewFingerprint != b.ReviewFingerprint {
		add("signed review fingerprint differs from the bundle")
	}
	if pl.BaseModelRevision != b.BaselineRevision {
		add("signed base revision differs from the bundle")
	}
	if pl.DecisionReasonDigest != mustDigest(b.Decision.Reason) {
		add("signed reason digest does not match the stated reason")
	}
	r := b.Decision.Reason
	if r.ProvenanceState != b.Provenance.ProvenanceState || r.Unresolved != b.Provenance.Unresolved ||
		r.SourceDigest != b.Provenance.SourceDigest || r.EvidenceFingerprint != b.Provenance.EvidenceFingerprint {
		add("the signed reason does not carry the provenance state and unresolved count of the candidate attachment")
	}
	if b.Acceptance.Performed {
		add("the bundle claims an acceptance was performed")
	}
	return out
}
