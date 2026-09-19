package main

import "testing"

const (
	secondApplicationGateCBaselineRevision = "e5231839a5dcc62cce1181a2731fcb7c70d16736"
	secondApplicationGateCTargetRevision   = "0ba87a4e0a96f70a198b03ff7652fb00e3826abe"
	secondApplicationGateCSemanticSHA      = "78c4c3b1031a70eb62683233fb5f1794cf7f71fd8b420ffa741d5ebe3f649d6b"
	secondApplicationGateCVisualSHA        = "61858f3d89891381684b5f8a363fbbfa0471ae313286cf239e2903aa800e2180"
	secondApplicationGateCReadinessSHA     = "b796e38db9a249ef98f2cb600eb354afc70f9adf57d2c83d47116f9968a0ff6c"
)

func TestSecondApplicationActualGateCEntryIsEvidenceCompleteWithoutAuthoringDecision(t *testing.T) {
	input := actualSecondApplicationGateCEntryInput()
	got := qualifyGateCSecondApplicationEntry(input)

	if got.State != "entry-evidence-complete" {
		t.Fatalf("actual Gate C entry did not open S0: %+v", got)
	}
	if len(got.Missing) != 0 {
		t.Fatalf("actual Gate C entry unexpectedly has missing inputs: %+v", got.Missing)
	}
	if input.HandoverBaselineID != "B" ||
		input.BaselineRepositoryRevision != secondApplicationGateCBaselineRevision ||
		input.TargetBuildRevision != secondApplicationGateCTargetRevision {
		t.Fatalf("actual B/R/T identity drifted: %+v", input)
	}
	if input.SemanticEvidenceRef != "sha256:"+secondApplicationGateCSemanticSHA+"#semantic-conformance" ||
		input.FunctionalEvidenceRef != "sha256:"+secondApplicationGateCSemanticSHA+"#functional-evidence" ||
		input.VisualEvidenceRef != "sha256:"+secondApplicationGateCVisualSHA+"#visual-fidelity" {
		t.Fatalf("independent Evidence references drifted: %+v", input)
	}

	for _, want := range []string{
		"target-capability-candidates-recorded",
		"runtime-binding-observations-recorded",
		"execution-artifact-abi-host-recorded",
		"web-specific-do-not-canonicalize-recorded",
		"carry-forward-unresolved-recorded",
	} {
		if !containsGateCEntry(input.TargetDifferences, want) {
			t.Fatalf("actual Gate C entry lost Target Difference category %q: %+v", want, input.TargetDifferences)
		}
	}
	for _, want := range []string{
		"actual-external-command-observation-not-acquired",
		"browse-environment-instance-binding-not-exposed-by-current-driver-api",
		"general-target-neutral-presentation-taxonomy-not-yet-defined",
		"target-capability-taxonomy-remains-candidate-until-second-target",
	} {
		if !containsGateCEntry(input.Unresolved, want) {
			t.Fatalf("actual Gate C entry lost carry-forward unresolved %q: %+v", want, input.Unresolved)
		}
	}
}

func TestSecondApplicationActualGateCEntryDoesNotUseReadinessHashAsSemanticOrVisualEvidence(t *testing.T) {
	input := actualSecondApplicationGateCEntryInput()
	for _, ref := range []string{
		input.SemanticEvidenceRef,
		input.FunctionalEvidenceRef,
		input.VisualEvidenceRef,
	} {
		if ref == "sha256:"+secondApplicationGateCReadinessSHA {
			t.Fatalf("final readiness artifact replaced an independent Evidence class: %q", ref)
		}
	}
}

func actualSecondApplicationGateCEntryInput() gateCSecondApplicationEntryInput {
	return gateCSecondApplicationEntryInput{
		GateCStatus:                "ready",
		HandoverBaselineID:         "B",
		BaselineRepositoryRevision: secondApplicationGateCBaselineRevision,
		TargetBuildRevision:        secondApplicationGateCTargetRevision,
		SemanticEvidenceRef:        "sha256:" + secondApplicationGateCSemanticSHA + "#semantic-conformance",
		FunctionalEvidenceRef:      "sha256:" + secondApplicationGateCSemanticSHA + "#functional-evidence",
		VisualEvidenceRef:          "sha256:" + secondApplicationGateCVisualSHA + "#visual-fidelity",
		TargetDifferences: []string{
			"target-capability-candidates-recorded",
			"runtime-binding-observations-recorded",
			"execution-artifact-abi-host-recorded",
			"web-specific-do-not-canonicalize-recorded",
			"carry-forward-unresolved-recorded",
		},
		Unresolved: []string{
			"actual-external-command-observation-not-acquired",
			"browse-environment-instance-binding-not-exposed-by-current-driver-api",
			"general-target-neutral-presentation-taxonomy-not-yet-defined",
			"target-capability-taxonomy-remains-candidate-until-second-target",
		},
	}
}
