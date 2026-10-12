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
			RawDiagnosticHint: "compile failed",
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
			RawDiagnosticHint: "builder did not start",
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
			RawDiagnosticHint: "compiler diagnostic captured",
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
			RawDiagnosticHint: "compile failed", Stderr: []byte("diagnostic A"),
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
			RawDiagnosticHint: "compile failed", Stderr: []byte("diagnostic B"),
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
			RawDiagnosticHint: "first compile failure", Stderr: []byte("first"),
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
			RawDiagnosticHint: "second link failure", Stderr: []byte("second"),
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
			RawDiagnosticHint: "different plan", Stderr: []byte("different"),
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
			RawDiagnosticHint: "compile failed", Stderr: []byte("diagnostic"),
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
			RawDiagnosticHint: "compile failed", Stderr: []byte("diagnostic"),
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
			RawDiagnosticHint: "original", Stderr: []byte("diagnostic"),
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
			RawDiagnosticHint: "first recorded attempt", Stderr: []byte("first"),
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
			RawDiagnosticHint: "duplicate ordinal", Stderr: []byte("duplicate"),
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
			RawDiagnosticHint: "earlier ordinal", Stderr: []byte("earlier"),
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

func TestFailedAttemptRejectsSuccessfulOrInvalidExitStatus(t *testing.T) {
	plan := referenceBuildPlan()
	for _, exitCode := range []int{0, -1} {
		_, _, err := newFailedAttempt(plan, 1, fakeFailure{
			Phase: attemptPhaseCompile, FailureClass: "compile-error",
			ExitCode: &exitCode, RawDiagnosticHint: "simulated failure",
		}, 8)
		if err == nil {
			t.Fatalf("failed attempt accepted invalid exit status %d", exitCode)
		}
	}
}

func TestSignalledBuildFailureAndExitStatusAreNotConflated(t *testing.T) {
	plan := referenceBuildPlan()
	zero, minusOne, minusTwo, positive := 0, -1, -2, 2
	for _, tt := range []struct {
		name string
		code *int
		signal string
		wantError bool
	}{
		{name: "signal-only", signal: "SIGTERM"},
		{name: "signal-exit-minus-one", code: &minusOne, signal: "SIGTERM"},
		{name: "ordinary-failure", code: &positive},
		{name: "successful-exit", code: &zero, wantError: true},
		{name: "zero-with-signal", code: &zero, signal: "SIGTERM", wantError: true},
		{name: "negative-without-signal", code: &minusOne, wantError: true},
		{name: "unsupported-negative-code", code: &minusTwo, signal: "SIGTERM", wantError: true},
		{name: "status-and-signal-conflict", code: &positive, signal: "SIGTERM", wantError: true},
		{name: "no-termination-status", wantError: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := newFailedAttempt(plan, 1, fakeFailure{
				Phase: attemptPhaseCompile, FailureClass: "compile-error",
				ExitCode: tt.code, Signal: tt.signal, RawDiagnosticHint: "classified failure",
			}, 16)
			if (err != nil) != tt.wantError {
				t.Fatalf("newFailedAttempt error = %v, want error = %t", err, tt.wantError)
			}
		})
	}
}

func TestUntrustedDiagnosticHintIsNotPublicEvidence(t *testing.T) {
	plan := referenceBuildPlan()
	exitCode := 2
	marker := "token=do-not-publish path=/private/workspace/user.go"
	unsafe, _, err := newFailedAttempt(plan, 1, fakeFailure{
		Phase: attemptPhaseCompile, FailureClass: "compile-error",
		ExitCode: &exitCode, RawDiagnosticHint: marker,
	}, 16)
	if err != nil {
		t.Fatal(err)
	}
	safe, _, err := newFailedAttempt(plan, 1, fakeFailure{
		Phase: attemptPhaseCompile, FailureClass: "compile-error",
		ExitCode: &exitCode, RawDiagnosticHint: "compile failed",
	}, 16)
	if err != nil {
		t.Fatal(err)
	}
	if unsafe.DiagnosticSummary != "build failed during compile: compile-error" {
		t.Fatalf("untrusted diagnostic hint changed public summary: %q", unsafe.DiagnosticSummary)
	}
	if unsafe.EvidenceFingerprint != safe.EvidenceFingerprint {
		t.Fatal("untrusted diagnostic hint changed public Evidence identity")
	}
	raw, err := json.Marshal(unsafe)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), marker) || strings.Contains(string(raw), "/private/") {
		t.Fatal("untrusted raw diagnostic hint leaked into public Evidence")
	}
}

func TestPublicFailureClassificationRejectsUnsafeLabelsAndSummary(t *testing.T) {
	plan := referenceBuildPlan()
	exitCode := 2
	for _, unsafeClass := range []string{"path=/private", "token=abc", "../internal", "UpperCase", strings.Repeat("a", 65)} {
		if _, _, err := newFailedAttempt(plan, 1, fakeFailure{
			Phase: attemptPhaseCompile, FailureClass: unsafeClass,
			ExitCode: &exitCode, RawDiagnosticHint: "ignored",
		}, 16); err == nil {
			t.Fatalf("unsafe failure class was accepted: %q", unsafeClass)
		}
	}
	attempt, _, err := newFailedAttempt(plan, 1, fakeFailure{
		Phase: attemptPhaseCompile, FailureClass: "compile-error",
		ExitCode: &exitCode, RawDiagnosticHint: "ignored",
	}, 16)
	if err != nil {
		t.Fatal(err)
	}
	attempt.DiagnosticSummary = "token=forged-value"
	attempt.EvidenceFingerprint, err = buildAttemptEvidenceFingerprint(attempt)
	if err != nil {
		t.Fatal(err)
	}
	if err := attempt.ValidateAgainst(plan); err == nil {
		t.Fatal("forged diagnostic summary passed validation after recomputing its fingerprint")
	}
}

func TestSignalMetadataRejectsRawDiagnosticContent(t *testing.T) {
	plan := referenceBuildPlan()
	for _, signal := range []string{
		"token=unsafe", "path=/private", " SIGTERM ", "signal 15",
		strings.Repeat("A", 33),
	} {
		_, _, err := newFailedAttempt(plan, 1, fakeFailure{
			Phase: attemptPhaseBuilderRun, FailureClass: "builder-error",
			Signal: signal, RawDiagnosticHint: "ignored",
		}, 8)
		if err == nil {
			t.Fatalf("unstructured signal value was accepted: %q", signal)
		}
	}
}
