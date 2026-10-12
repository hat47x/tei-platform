package main

import (
	"encoding/json"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
)

func TestFailedAttemptBindsExactBuildPlanAndBuilder(t *testing.T) {
	plan := referenceBuildPlan()
	exitCode := 2
	attempt, _, err := newFailedAttempt(
		plan,
		1,
		fakeFailure{
			Phase:             attemptPhaseCompile,
			FailureClass:      "compile-type-error",
			ExitCode:          &exitCode,
			DiagnosticSummary: "compile failed",
			Stdout:            []byte("progress"),
			Stderr:            []byte("type mismatch"),
		},
		16,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := attempt.ValidateAgainst(plan); err != nil {
		t.Fatal(err)
	}

	changedBuilder := plan
	changedBuilder.Builder.ArtifactDigest = "sha256:" + strings.Repeat("c", 64)
	if err := attempt.ValidateAgainst(changedBuilder); err == nil {
		t.Fatal("Build Attempt Evidence accepted a different exact Builder")
	}

	changedTarget := plan
	changedTarget.Target = "host/other"
	if err := attempt.ValidateAgainst(changedTarget); err == nil {
		t.Fatal("Build Attempt Evidence accepted a different Target")
	}
}

func TestAttemptEvidenceRequiresResolvedValidBuildPlan(t *testing.T) {
	plan := referenceBuildPlan()
	plan.Builder.ArtifactDigest = ""
	exitCode := 1
	if _, _, err := newFailedAttempt(
		plan,
		1,
		fakeFailure{
			Phase:             attemptPhaseBuilderRun,
			FailureClass:      "builder-start-failure",
			ExitCode:          &exitCode,
			DiagnosticSummary: "builder did not start",
		},
		16,
	); err == nil {
		t.Fatal("Build Attempt Evidence was created before an exact valid BuildPlan existed")
	}
}

func TestDiagnosticCaptureIsBoundedAndRawSecretIsNotInline(t *testing.T) {
	plan := referenceBuildPlan()
	exitCode := 2
	secret := "token=super-secret-value"
	raw := []byte(strings.Repeat("x", 40) + secret + strings.Repeat("y", 40))
	attempt, retained, err := newFailedAttempt(
		plan,
		1,
		fakeFailure{
			Phase:             attemptPhaseCompile,
			FailureClass:      "compile-error",
			ExitCode:          &exitCode,
			DiagnosticSummary: "compiler diagnostic captured",
			Stderr:            raw,
		},
		24,
	)
	if err != nil {
		t.Fatal(err)
	}
	if attempt.Stderr.Bytes != int64(len(raw)) ||
		attempt.Stderr.RetainedBytes != 24 ||
		!attempt.Stderr.Truncated {
		t.Fatalf("stderr capture: %#v", attempt.Stderr)
	}
	if len(retained.Stderr) != 24 {
		t.Fatalf("retained stderr bytes=%d want=24", len(retained.Stderr))
	}
	encoded, err := json.Marshal(attempt)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), secret) {
		t.Fatal("raw diagnostic secret leaked into Build Attempt Evidence JSON")
	}
	if attempt.Stderr.Digest != digestForTest(raw) {
		t.Fatalf("stderr digest=%s want=%s", attempt.Stderr.Digest, digestForTest(raw))
	}
}

func TestRawDiagnosticChangeDoesNotChangeBuildPlanIdentity(t *testing.T) {
	plan := referenceBuildPlan()
	planFingerprint, err := plugin.BuildPlanFingerprint(plan)
	if err != nil {
		t.Fatal(err)
	}
	exitCode := 2
	first, _, err := newFailedAttempt(
		plan, 1,
		fakeFailure{
			Phase: attemptPhaseCompile, FailureClass: "compile-error", ExitCode: &exitCode,
			DiagnosticSummary: "compile failed", Stderr: []byte("diagnostic A"),
		},
		16,
	)
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := newFailedAttempt(
		plan, 1,
		fakeFailure{
			Phase: attemptPhaseCompile, FailureClass: "compile-error", ExitCode: &exitCode,
			DiagnosticSummary: "compile failed", Stderr: []byte("diagnostic B"),
		},
		16,
	)
	if err != nil {
		t.Fatal(err)
	}
	if first.BuildPlanFingerprint != planFingerprint || second.BuildPlanFingerprint != planFingerprint {
		t.Fatal("raw diagnostic changed BuildPlan identity")
	}
	if first.EvidenceFingerprint == second.EvidenceFingerprint {
		t.Fatal("different raw diagnostics collapsed into one Attempt Evidence identity")
	}
}

func TestRetryHistoryPreservesFailuresAfterSuccess(t *testing.T) {
	plan := referenceBuildPlan()
	exitCode := 2
	first, _, err := newFailedAttempt(
		plan, 1,
		fakeFailure{
			Phase: attemptPhaseCompile, FailureClass: "compile-error", ExitCode: &exitCode,
			DiagnosticSummary: "first compile failure", Stderr: []byte("first"),
		},
		8,
	)
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := newFailedAttempt(
		plan, 2,
		fakeFailure{
			Phase: attemptPhaseLink, FailureClass: "link-error", ExitCode: &exitCode,
			DiagnosticSummary: "second link failure", Stderr: []byte("second"),
		},
		8,
	)
	if err != nil {
		t.Fatal(err)
	}
	history, err := newAttemptHistory(plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := history.AppendFailure(plan, first); err != nil {
		t.Fatal(err)
	}
	if err := history.AppendFailure(plan, second); err != nil {
		t.Fatal(err)
	}
	builder := &fakeSuccessfulBuilder{}
	success, err := builder.Run(plan, 3, []byte("verified fake-builder output"))
	if err != nil {
		t.Fatal(err)
	}
	if err := history.AppendSuccess(plan, success); err != nil {
		t.Fatal(err)
	}
	if err := history.Validate(); err != nil {
		t.Fatal(err)
	}
	if len(history.FailedAttempts) != 2 {
		t.Fatalf("successful retry erased failure history: %#v", history)
	}
}

func TestAttemptHistoryRejectsFailureFromDifferentPlan(t *testing.T) {
	firstPlan := referenceBuildPlan()
	secondPlan := referenceBuildPlan()
	secondPlan.Sources = append([]plugin.BuildSource(nil), secondPlan.Sources...)
	secondPlan.Sources[0].Digest = "sha256:" + strings.Repeat("f", 64)

	exitCode := 2
	attempt, _, err := newFailedAttempt(
		secondPlan, 1,
		fakeFailure{
			Phase: attemptPhaseCompile, FailureClass: "compile-error", ExitCode: &exitCode,
			DiagnosticSummary: "different plan", Stderr: []byte("different"),
		},
		8,
	)
	if err != nil {
		t.Fatal(err)
	}
	history, err := newAttemptHistory(firstPlan)
	if err != nil {
		t.Fatal(err)
	}
	if err := history.AppendFailure(firstPlan, attempt); err == nil {
		t.Fatal("Attempt History mixed failures from different BuildPlans")
	}
}

func TestGraphBlockedNodeIsNotBuildFailure(t *testing.T) {
	failed := buildGraphAttemptState{
		NodeID:                     "generator",
		State:                      nodeAttemptFailed,
		AttemptEvidenceFingerprint: strings.Repeat("a", 64),
	}
	if err := failed.Validate(); err != nil {
		t.Fatal(err)
	}
	blocked := buildGraphAttemptState{
		NodeID:    "consumer",
		State:     nodeAttemptBlocked,
		BlockedBy: []string{"generator"},
	}
	if err := blocked.Validate(); err != nil {
		t.Fatal(err)
	}
	blocked.AttemptEvidenceFingerprint = strings.Repeat("b", 64)
	if err := blocked.Validate(); err == nil {
		t.Fatal("blocked node claimed Build Attempt Evidence as if it had executed and failed")
	}
}

func TestAttemptEvidenceFingerprintCanonicalizesBuilderDigestHexCase(t *testing.T) {
	plan := referenceBuildPlan()
	exitCode := 2
	attempt, _, err := newFailedAttempt(
		plan, 1,
		fakeFailure{
			Phase: attemptPhaseCompile, FailureClass: "compile-error", ExitCode: &exitCode,
			DiagnosticSummary: "compile failed", Stderr: []byte("diagnostic"),
		},
		8,
	)
	if err != nil {
		t.Fatal(err)
	}
	upper := attempt
	upper.Builder.ArtifactDigest = uppercaseDigest(attempt.Builder.ArtifactDigest)
	upper.BuildPlanFingerprint = strings.ToUpper(attempt.BuildPlanFingerprint)
	upper.Stdout.Digest = uppercaseDigest(attempt.Stdout.Digest)
	upper.Stderr.Digest = uppercaseDigest(attempt.Stderr.Digest)

	lowerFP, err := buildAttemptEvidenceFingerprint(attempt)
	if err != nil {
		t.Fatal(err)
	}
	upperFP, err := buildAttemptEvidenceFingerprint(upper)
	if err != nil {
		t.Fatal(err)
	}
	if lowerFP != upperFP {
		t.Fatal("valid SHA-256 hex case changed Build Attempt Evidence identity")
	}
}

func TestReferenceEvidenceDoesNotInlineSensitiveDiagnostic(t *testing.T) {
	evidence, err := buildReferenceEvidence()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(evidence)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "secret-never-inline") ||
		strings.Contains(string(raw), "/workspace/private/source.go") {
		t.Fatal("Reference Evidence leaked raw sensitive diagnostic content")
	}
	if evidence.Attempt.Outcome != attemptOutcomeFailed {
		t.Fatalf("attempt outcome=%s", evidence.Attempt.Outcome)
	}
	if len(evidence.History.FailedAttempts) != 1 ||
		len(evidence.History.SuccessfulAttempts) != 0 {
		t.Fatalf("failed fake build must not claim a successful execution: %#v", evidence.History)
	}
}

func TestAttemptAuthorityLimits(t *testing.T) {
	evidence, err := buildReferenceEvidence()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range attemptAuthorityLimits {
		found := false
		for _, got := range evidence.Attempt.AuthorityLimits {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("authority limit %q missing: %#v", want, evidence.Attempt.AuthorityLimits)
		}
	}
}

func TestBuilderIdentityMismatchOnLogicalFieldsIsRejected(t *testing.T) {
	plan := referenceBuildPlan()
	exitCode := 2
	attempt, _, err := newFailedAttempt(
		plan, 1,
		fakeFailure{
			Phase: attemptPhaseCompile, FailureClass: "compile-error", ExitCode: &exitCode,
			DiagnosticSummary: "compile failed", Stderr: []byte("diagnostic"),
		},
		8,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*plugin.BuildPlan){
		func(plan *plugin.BuildPlan) { plan.Builder.ID = core.MustCanonicalID("tei.builder.other") },
		func(plan *plugin.BuildPlan) { plan.Builder.Version = "2.0.0" },
		func(plan *plugin.BuildPlan) { plan.Builder.ContractVersion = "tei.reference.other/v1" },
	} {
		changed := plan
		mutate(&changed)
		if err := attempt.ValidateAgainst(changed); err == nil {
			t.Fatal("Build Attempt Evidence accepted a different logical Builder identity")
		}
	}
}

func digestForTest(value []byte) string {
	return digestBytes(value)
}

func uppercaseDigest(value string) string {
	return "sha256:" + strings.ToUpper(strings.TrimPrefix(value, "sha256:"))
}

func TestAttemptHistoryRejectsTamperedFailureEvidence(t *testing.T) {
	plan := referenceBuildPlan()
	exitCode := 2
	attempt, _, err := newFailedAttempt(
		plan, 1,
		fakeFailure{
			Phase: attemptPhaseCompile, FailureClass: "compile-error", ExitCode: &exitCode,
			DiagnosticSummary: "original", Stderr: []byte("diagnostic"),
		},
		8,
	)
	if err != nil {
		t.Fatal(err)
	}
	history, err := newAttemptHistory(plan)
	if err != nil {
		t.Fatal(err)
	}
	tampered := attempt
	tampered.DiagnosticSummary = "tampered after fingerprint"
	if err := history.AppendFailure(plan, tampered); err == nil {
		t.Fatal("Attempt History accepted Evidence whose fingerprint no longer matches its content")
	}
}

func TestAttemptHistoryRejectsSuccessFromDifferentPlan(t *testing.T) {
	plan := referenceBuildPlan()
	other := referenceBuildPlan()
	other.Sources = append([]plugin.BuildSource(nil), other.Sources...)
	other.Sources[0].Digest = "sha256:" + strings.Repeat("f", 64)
	builder := &fakeSuccessfulBuilder{}
	success, err := builder.Run(other, 1, []byte("different output"))
	if err != nil {
		t.Fatal(err)
	}
	history, err := newAttemptHistory(plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := history.AppendSuccess(plan, success); err == nil {
		t.Fatal("Attempt History accepted an execution from a different BuildPlan")
	}
	history.SuccessfulAttempts = append(history.SuccessfulAttempts, success)
	if err := history.Validate(); err == nil {
		t.Fatal("Attempt History validation accepted an execution from a different BuildPlan")
	}
}

func TestAttemptHistoryRejectsDuplicateOrOutOfOrderOrdinal(t *testing.T) {
	plan := referenceBuildPlan()
	exitCode := 2
	first, _, err := newFailedAttempt(
		plan, 2,
		fakeFailure{
			Phase: attemptPhaseCompile, FailureClass: "compile-error", ExitCode: &exitCode,
			DiagnosticSummary: "first recorded attempt", Stderr: []byte("first"),
		},
		8,
	)
	if err != nil {
		t.Fatal(err)
	}
	history, err := newAttemptHistory(plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := history.AppendFailure(plan, first); err != nil {
		t.Fatal(err)
	}

	duplicateOrdinal, _, err := newFailedAttempt(
		plan, 2,
		fakeFailure{
			Phase: attemptPhaseLink, FailureClass: "link-error", ExitCode: &exitCode,
			DiagnosticSummary: "duplicate ordinal", Stderr: []byte("duplicate"),
		},
		8,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := history.AppendFailure(plan, duplicateOrdinal); err == nil {
		t.Fatal("Attempt History accepted a duplicate attempt ordinal")
	}

	earlierOrdinal, _, err := newFailedAttempt(
		plan, 1,
		fakeFailure{
			Phase: attemptPhaseBuilderRun, FailureClass: "builder-error", ExitCode: &exitCode,
			DiagnosticSummary: "earlier ordinal", Stderr: []byte("earlier"),
		},
		8,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := history.AppendFailure(plan, earlierOrdinal); err == nil {
		t.Fatal("Attempt History accepted an out-of-order attempt ordinal")
	}
}

func TestAttemptHistoryValidationRejectsDuplicateEvidenceFingerprint(t *testing.T) {
	history := buildAttemptHistory{
		Schema:               attemptHistorySchema,
		BuildPlanFingerprint: strings.Repeat("a", 64),
		FailedAttempts: []failedAttemptReference{
			{AttemptOrdinal: 1, EvidenceFingerprint: strings.Repeat("b", 64)},
			{AttemptOrdinal: 2, EvidenceFingerprint: strings.Repeat("b", 64)},
		},
	}
	if err := history.Validate(); err == nil {
		t.Fatal("Attempt History accepted a duplicated failure Evidence fingerprint")
	}
}
