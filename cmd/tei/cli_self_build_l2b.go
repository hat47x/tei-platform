package main

import "fmt"

// This file computes SB-C0's L2-b claim from the observations an attempt made.
//
// Until now every field that could have carried the claim wrote `l2bClaimed:
// false` as a constant: a single observation is not the claim, and saying so
// prevented any one of them from being read as the whole. That is still right for
// each individual field. What was missing is the statement that the claim holds
// when all of them hold together.
//
// So the per-observation flags keep saying false — an artifact binding alone
// still does not make L2-b — and this record answers the combined question
// separately, naming every condition an attempt did not meet.
//
// §4.8 split L2 into L2-a (the entry handed over the declaration) and L2-b (the
// declaration was interpreted into the plan that produced the artifact). This is
// the L2-b predicate.

const l2bClaimBoundary = "this attempt read the declaration, interpreted it into a plan itself, matched the builder's declaration bytes to the ones it held, saw the builder report the same plan, reproduced the artifact from that plan, and found the file it started to have the bytes of the builder the plan declares; that is L2-b for this attempt on the host that ran it, with these binaries. Observed on linux/amd64 and windows/amd64 (§4.25); not proven for any other host. Builder identity here means the bytes of the file named by -child, read before each start; it is not an observation of the process image or of anything that file launches. It is not acceptance, not Runtime Authority, not a statement about the catalog's correctness, not a trust judgement about the supply, and not L2-c or L3"

// selfBuildL2BClaim is the combined claim, with the conditions it failed.
type selfBuildL2BClaim struct {
	Claimed bool `json:"claimed"`
	// Unmet lists every condition the attempt did not meet, so a reader learns
	// what to change instead of only that the claim is false.
	Unmet    []string `json:"unmet,omitempty"`
	Boundary string   `json:"boundary"`
}

// assessL2B evaluates the predicate over what this attempt recorded.
//
// Each condition is stated as a closed question about a field, so a future change
// to one observation cannot silently satisfy the claim.
func assessL2B(report selfBuildOuterEvidence) selfBuildL2BClaim {
	claim := selfBuildL2BClaim{Boundary: l2bClaimBoundary}
	unmet := func(format string, args ...any) {
		claim.Unmet = append(claim.Unmet, fmt.Sprintf(format, args...))
	}

	// The entry must have interpreted the declaration itself. A plan produced by
	// a resolver child is L2-a plus a hand-off, not this entry interpreting it.
	switch {
	case report.PlanReceipt == nil:
		unmet("no plan was derived for this attempt")
	case report.PlanReceipt.DerivedBy != derivedByEntryInProcess:
		unmet("the plan was derived by %q, not by this entry", report.PlanReceipt.DerivedBy)
	case report.PlanReceipt.State != intentResolutionResolved:
		unmet("the plan derivation did not resolve: %s", report.PlanReceipt.FailureCause)
	}

	// The declaration bytes the builder read must be the ones this entry held.
	switch {
	case report.IntentBinding == nil:
		unmet("the attempt recorded no declaration binding")
	case !report.IntentBinding.CheckPerformed:
		unmet("the declaration byte check did not run")
	case !report.IntentBinding.CheckMatched:
		unmet("the builder read different declaration bytes than the entry held")
	}

	// The builder's reported plan must be the plan the entry derived.
	if report.PlanCrossCheck == nil || report.PlanCrossCheck.State != planCrossCheckMatched {
		state := "unperformed"
		if report.PlanCrossCheck != nil {
			state = report.PlanCrossCheck.State
		}
		unmet("the builder's plan was not matched against the entry's (%s)", state)
	}

	// The plan must be bound to the artifact, and the artifact must be
	// reproducible from that plan. The binding alone ties two values; only the
	// reproduction shows the plan produces those bytes.
	if report.PlanArtifactBinding == nil || report.PlanArtifactBinding.State != planArtifactBound {
		state := "unperformed"
		if report.PlanArtifactBinding != nil {
			state = report.PlanArtifactBinding.State
		}
		unmet("the plan was not bound to the artifact (%s)", state)
	}
	// The binding's own flag stays false as a statement about that observation;
	// this predicate is where the combination is allowed to mean more. Reading
	// the flag from the binding here would make the claim depend on a field whose
	// whole purpose is to say "not yet".
	if report.ArtifactReproducibility == nil || report.ArtifactReproducibility.State != reproducibilityReproduced {
		state := "unperformed"
		if report.ArtifactReproducibility != nil {
			state = report.ArtifactReproducibility.State
		}
		unmet("the artifact was not reproduced from the plan (%s)", state)
	}

	// §4.42: the plan declares a builder by digest. The claim "this plan produced
	// this artifact" is only as strong as the link between that declaration and
	// the file the entry started. Reproduction compares the started file with
	// itself across two runs; this compares it with the plan. Without it, a
	// wrapper or proxy named as -child satisfies every other condition while the
	// declared builder is never shown to have been the file that ran.
	if report.BuilderBinding == nil || report.BuilderBinding.State != builderBindingBound {
		state := "unperformed"
		if report.BuilderBinding != nil {
			state = report.BuilderBinding.State
		}
		unmet("the file this entry started was not shown to be the builder the plan declares (%s)", state)
	}

	claim.Claimed = len(claim.Unmet) == 0
	return claim
}
