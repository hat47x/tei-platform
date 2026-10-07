package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReviewProjectionOrdersA1ToA8AndMakesMissingCoverageExplicit(t *testing.T) {
	dir := scratchWorkspace(t)
	before := treeDigest(t, dir)
	candidate := unitPriceCandidate(revisionOfDir(t, dir))

	ev, err := evaluate(dir, mustJSON(t, candidate))
	if err != nil {
		t.Fatal(err)
	}
	if !ev.preview.Acceptable {
		t.Fatalf("preview is not acceptable: %+v", ev.preview.Stages)
	}

	review := buildReviewProjection(ev)
	if review.Schema != reviewSchema {
		t.Fatalf("schema=%q", review.Schema)
	}
	if review.CandidateFingerprint == "" || review.EvidenceFingerprint == "" || review.ReviewFingerprint == "" {
		t.Fatalf("review identities are incomplete: %+v", review)
	}
	if got := fingerprintReviewProjection(review); got != review.ReviewFingerprint {
		t.Fatalf("review fingerprint is not reproducible: have=%s got=%s", review.ReviewFingerprint, got)
	}
	if len(review.Sections) != 8 {
		t.Fatalf("sections=%d want 8", len(review.Sections))
	}
	if review.Coverage.Available != 3 ||
		review.Coverage.Partial != 5 ||
		review.Coverage.NotEvaluated != 0 ||
		len(review.Coverage.Incomplete) != 5 {
		t.Fatalf("coverage=%+v", review.Coverage)
	}
	for _, want := range []string{"A2", "A3", "A5", "A6", "A7"} {
		found := false
		for _, gap := range review.Coverage.Incomplete {
			if gap.Code == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("coverage incomplete lacks %s: %+v", want, review.Coverage.Incomplete)
		}
	}
	for i, section := range review.Sections {
		want := "A" + string(rune('1'+i))
		if section.Code != want {
			t.Fatalf("section[%d]=%s want %s", i, section.Code, want)
		}
	}

	a1 := reviewSection(t, review, "A1")
	if a1.Coverage != reviewCoverageAvailable ||
		a1.Summary != "semantic 1件 / application 0件 / environment 0件" ||
		!sectionContains(a1, "added validation:sales.order_line.unit_price_positive") ||
		!containsString(a1.Sources, "preview.semanticDiff") {
		t.Fatalf("A1=%+v", a1)
	}

	a2 := reviewSection(t, review, "A2")
	if a2.Coverage != reviewCoveragePartial || !strings.Contains(a2.Summary, "impact ") {
		t.Fatalf("A2=%+v", a2)
	}

	a5 := reviewSection(t, review, "A5")
	if a5.Coverage != reviewCoveragePartial ||
		a5.Summary != "runtime authority grant 0件 / effective authority impact 未評価" ||
		!sectionContains(a5, "runtime authority grant: 0件") {
		t.Fatalf("A5=%+v", a5)
	}

	a6 := reviewSection(t, review, "A6")
	if a6.Coverage != reviewCoveragePartial ||
		a6.Summary != "native code authoring 0件 / residual native dependency impact 未評価" ||
		!sectionContains(a6, "customGoBindings collection: unchanged") {
		t.Fatalf("A6=%+v", a6)
	}

	a7 := reviewSection(t, review, "A7")
	if a7.Coverage != reviewCoveragePartial ||
		!sectionContains(a7, "candidateFingerprint: "+review.CandidateFingerprint) ||
		!sectionContains(a7, "source edit: semantic/sales/validation.tei") ||
		!containsString(a7.Sources, "preview.evidenceFingerprint") {
		t.Fatalf("A7=%+v", a7)
	}

	a8 := reviewSection(t, review, "A8")
	if a8.Coverage != reviewCoverageAvailable || !sectionContains(a8, "変更subject数 1 / 上限 8") {
		t.Fatalf("A8=%+v", a8)
	}

	assertTreeUnchanged(t, before, treeDigest(t, dir))
}

func TestReviewProjectionCoversSecondApplicationComposition(t *testing.T) {
	dir := scratchWorkspaceBeforeSecondApplication(t)
	before := treeDigest(t, dir)
	ev, err := evaluate(dir, mustJSON(t, secondApplicationCandidate(revisionOfDir(t, dir))))
	if err != nil {
		t.Fatal(err)
	}
	if !ev.preview.Acceptable {
		t.Fatalf("preview is not acceptable: %+v", ev.preview.Stages)
	}

	review := buildReviewProjection(ev)
	a1 := reviewSection(t, review, "A1")
	if len(a1.Items) != 6 || a1.Summary != "semantic 4件 / application 2件 / environment 0件" {
		t.Fatalf("A1=%+v", a1)
	}
	for _, want := range []string{
		"added namespace:repair_intake",
		"added data:repair_intake.intake",
		"added application:repair_intake",
		"added application-pattern:repair_intake.customer_reference",
	} {
		if !sectionContains(a1, want) {
			t.Fatalf("A1 lacks %q: %+v", want, a1.Items)
		}
	}

	a4 := reviewSection(t, review, "A4")
	for _, want := range []string{
		"ENVIRONMENT_BINDING_NOT_AUTHORED",
		"DRIVER_SCENARIO_NOT_RUN",
		"TARGET_PROJECTION_NOT_EVALUATED",
	} {
		if !sectionContains(a4, want) {
			t.Fatalf("A4 lacks %s: %+v", want, a4.Items)
		}
	}

	a5 := reviewSection(t, review, "A5")
	if a5.Coverage != reviewCoveragePartial ||
		!sectionContains(a5, "Application追加: 1件") ||
		!sectionContains(a5, "ENVIRONMENT_BINDING_NOT_AUTHORED") {
		t.Fatalf("A5=%+v", a5)
	}

	a7 := reviewSection(t, review, "A7")
	for _, path := range []string{
		s4SemanticPath,
		s4ApplicationPath,
		s4PatternPath,
	} {
		if !sectionContains(a7, "source edit: "+path) {
			t.Fatalf("A7 lacks source %s: %+v", path, a7.Items)
		}
	}

	a8 := reviewSection(t, review, "A8")
	if a8.Summary != "6 / 8" || !sectionContains(a8, "変更subject数 6 / 上限 8") {
		t.Fatalf("A8=%+v", a8)
	}
	assertTreeUnchanged(t, before, treeDigest(t, dir))
}

func TestReviewFingerprintBindsExactProjectionWithoutChangingSemanticIdentity(t *testing.T) {
	dir := scratchWorkspace(t)
	ev, err := evaluate(dir, mustJSON(t, unitPriceCandidate(revisionOfDir(t, dir))))
	if err != nil {
		t.Fatal(err)
	}
	review := buildReviewProjection(ev)
	if review.ReviewFingerprint == "" {
		t.Fatal("review fingerprint is empty")
	}

	changed := review
	changed.Sections = append([]ReviewSection(nil), review.Sections...)
	changed.Sections[0] = review.Sections[0]
	changed.Sections[0].Summary += "（表示変更）"
	changedFingerprint := fingerprintReviewProjection(changed)
	if changedFingerprint == review.ReviewFingerprint {
		t.Fatalf("presentation change did not alter review fingerprint: %s", review.ReviewFingerprint)
	}
	if changed.CandidateFingerprint != review.CandidateFingerprint ||
		changed.EvidenceFingerprint != review.EvidenceFingerprint {
		t.Fatalf(
			"presentation-only change altered semantic/evidence identities: candidate=%s/%s evidence=%s/%s",
			review.CandidateFingerprint,
			changed.CandidateFingerprint,
			review.EvidenceFingerprint,
			changed.EvidenceFingerprint,
		)
	}
}

func TestAcceptCanBindTheExactReviewProjection(t *testing.T) {
	dir := scratchWorkspace(t)
	data := mustJSON(t, unitPriceCandidate(revisionOfDir(t, dir)))
	ev, err := evaluate(dir, data)
	if err != nil {
		t.Fatal(err)
	}
	review := buildReviewProjection(ev)

	decision, err := accept(dir, data, acceptOptions{
		ExpectedReviewFingerprint: review.ReviewFingerprint,
	})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Decision != "accepted" || !decision.SourceChanged {
		t.Fatalf("decision=%+v", decision)
	}
	if decision.ReviewBinding == nil ||
		decision.ReviewBinding.Fingerprint != review.ReviewFingerprint ||
		decision.ReviewBinding.Projection == nil ||
		decision.ReviewBinding.Projection.ReviewFingerprint != review.ReviewFingerprint ||
		fingerprintReviewProjection(*decision.ReviewBinding.Projection) != review.ReviewFingerprint {
		t.Fatalf("review binding=%+v want %s", decision.ReviewBinding, review.ReviewFingerprint)
	}
	if decision.ReviewBinding.Coverage.Available != review.Coverage.Available ||
		decision.ReviewBinding.Coverage.Partial != review.Coverage.Partial ||
		decision.ReviewBinding.Coverage.NotEvaluated != review.Coverage.NotEvaluated ||
		len(decision.ReviewBinding.Coverage.Incomplete) != len(review.Coverage.Incomplete) {
		t.Fatalf("review coverage drifted: decision=%+v review=%+v", decision.ReviewBinding.Coverage, review.Coverage)
	}
}

func TestAcceptRefusesWhenTheReviewedProjectionChanged(t *testing.T) {
	dir := scratchWorkspace(t)
	before := treeDigest(t, dir)
	candidate := unitPriceCandidate(revisionOfDir(t, dir))
	originalData := mustJSON(t, candidate)
	originalEvaluation, err := evaluate(dir, originalData)
	if err != nil {
		t.Fatal(err)
	}
	originalReview := buildReviewProjection(originalEvaluation)

	changed := candidate
	changed.Intent += "（Review後に説明を変更）"
	changedData := mustJSON(t, changed)
	changedEvaluation, err := evaluate(dir, changedData)
	if err != nil {
		t.Fatal(err)
	}
	changedReview := buildReviewProjection(changedEvaluation)
	if changedReview.CandidateFingerprint != originalReview.CandidateFingerprint ||
		changedReview.EvidenceFingerprint != originalReview.EvidenceFingerprint {
		t.Fatalf(
			"presentation-only change altered semantic/evidence identity: candidate=%s/%s evidence=%s/%s",
			originalReview.CandidateFingerprint,
			changedReview.CandidateFingerprint,
			originalReview.EvidenceFingerprint,
			changedReview.EvidenceFingerprint,
		)
	}
	if changedReview.ReviewFingerprint == originalReview.ReviewFingerprint {
		t.Fatalf("changed presentation kept review fingerprint %s", originalReview.ReviewFingerprint)
	}

	decision, err := accept(dir, changedData, acceptOptions{
		ExpectedReviewFingerprint: originalReview.ReviewFingerprint,
	})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Decision != "refused" || decision.SourceChanged {
		t.Fatalf("decision=%+v", decision)
	}
	if !decisionRefusalHasCode(decision, "REVIEW_FINGERPRINT_MISMATCH") {
		t.Fatalf("review mismatch refusal is absent: %+v", decision.Refusals)
	}
	if decision.ReviewBinding != nil {
		t.Fatalf("mismatched Review was recorded as bound: %+v", decision.ReviewBinding)
	}
	assertTreeUnchanged(t, before, treeDigest(t, dir))
}

func TestReviewSeparatesEnvironmentBindingFromRuntimeAuthority(t *testing.T) {
	dir := scratchWorkspaceBeforeEnvironmentBinding(t)
	before := treeDigest(t, dir)
	ev, err := evaluate(dir, mustJSON(t, environmentBindingCandidate(revisionOfDir(t, dir))))
	if err != nil {
		t.Fatal(err)
	}
	if !ev.preview.Acceptable {
		t.Fatalf("preview is not acceptable: %+v", ev.preview.Stages)
	}

	review := buildReviewProjection(ev)
	a5 := reviewSection(t, review, "A5")
	if a5.Coverage != reviewCoveragePartial ||
		a5.Summary != "runtime authority grant 0件 / effective authority impact 未評価" ||
		!sectionContains(a5, "Environment binding追加: 2件") ||
		!sectionContains(a5, "RUNTIME_AUTHORITY_NOT_GRANTED") {
		t.Fatalf("A5=%+v", a5)
	}
	if !strings.Contains(a5.Note, "実効権限が変わらないことは別") {
		t.Fatalf("A5 did not preserve the effective-authority gap: %+v", a5)
	}

	a6 := reviewSection(t, review, "A6")
	if a6.Coverage != reviewCoveragePartial ||
		!sectionContains(a6, "candidateがauthorするnative code: 0件") ||
		!strings.Contains(a6.Note, "既存custom implementation") {
		t.Fatalf("A6=%+v", a6)
	}

	assertTreeUnchanged(t, before, treeDigest(t, dir))
}

func TestReviewCLIIsReadOnlyAndCanEmitTextOrJSON(t *testing.T) {
	for _, format := range []string{"text", "json"} {
		t.Run(format, func(t *testing.T) {
			dir := scratchWorkspace(t)
			before := treeDigest(t, dir)
			candidatePath := filepath.Join(t.TempDir(), "candidate.json")
			if err := os.WriteFile(candidatePath, mustJSON(t, unitPriceCandidate(revisionOfDir(t, dir))), 0o644); err != nil {
				t.Fatal(err)
			}

			var stdout, stderr bytes.Buffer
			code := runCLI(
				[]string{"review", "-workspace", dir, "-candidate", candidatePath, "-format", format},
				&stdout,
				&stderr,
			)
			if code != 0 {
				t.Fatalf("code=%d stderr=%s output=%s", code, stderr.String(), stdout.String())
			}

			switch format {
			case "text":
				for _, want := range []string{
					"# Shared Acceptance Review",
					"## A1 意味差分 [表示可能]",
					"## A5 権限の変更 [一部]",
					"## A6 native codeへ残る箇所 [一部]",
					"## A8 承認単位の大きさ [表示可能]",
					"candidate fingerprint:",
					"evidence fingerprint:",
					"review fingerprint:",
					"coverage: 表示可能 3 / 一部 5 / 未評価 0",
					"要約: semantic 1件 / application 0件 / environment 0件",
					"要約: runtime authority grant 0件 / effective authority impact 未評価",
					"根拠: preview.semanticDiff",
				} {
					if !strings.Contains(stdout.String(), want) {
						t.Fatalf("text review lacks %q:\n%s", want, stdout.String())
					}
				}
			case "json":
				var review ReviewProjection
				if err := json.Unmarshal(stdout.Bytes(), &review); err != nil {
					t.Fatal(err)
				}
				if review.Schema != reviewSchema || len(review.Sections) != 8 || review.ReviewFingerprint == "" {
					t.Fatalf("review=%+v", review)
				}
				if got := fingerprintReviewProjection(review); got != review.ReviewFingerprint {
					t.Fatalf("JSON review fingerprint is not reproducible: have=%s got=%s", review.ReviewFingerprint, got)
				}
			}

			assertTreeUnchanged(t, before, treeDigest(t, dir))
		})
	}
}

func TestReviewFingerprintGuardFlowsThroughCLI(t *testing.T) {
	dir := scratchWorkspaceBeforeSecondApplication(t)
	candidatePath := filepath.Join(t.TempDir(), "candidate.json")
	if err := os.WriteFile(candidatePath, mustJSON(t, unitPriceCandidate(revisionOfDir(t, dir))), 0o644); err != nil {
		t.Fatal(err)
	}

	var reviewOut, reviewErr bytes.Buffer
	if code := runCLI(
		[]string{"review", "-workspace", dir, "-candidate", candidatePath, "-format", "json"},
		&reviewOut,
		&reviewErr,
	); code != 0 {
		t.Fatalf("review code=%d stderr=%s output=%s", code, reviewErr.String(), reviewOut.String())
	}
	var review ReviewProjection
	if err := json.Unmarshal(reviewOut.Bytes(), &review); err != nil {
		t.Fatal(err)
	}
	if review.ReviewFingerprint == "" {
		t.Fatal("review CLI returned no reviewFingerprint")
	}

	var acceptOut, acceptErr bytes.Buffer
	if code := runCLI(
		[]string{
			"accept",
			"-workspace", dir,
			"-candidate", candidatePath,
			"-expected-review-fingerprint", review.ReviewFingerprint,
		},
		&acceptOut,
		&acceptErr,
	); code != 0 {
		t.Fatalf("accept code=%d stderr=%s output=%s", code, acceptErr.String(), acceptOut.String())
	}
	var decision Decision
	if err := json.Unmarshal(acceptOut.Bytes(), &decision); err != nil {
		t.Fatal(err)
	}
	if decision.Decision != "accepted" ||
		decision.ReviewBinding == nil ||
		decision.ReviewBinding.Fingerprint != review.ReviewFingerprint {
		t.Fatalf("decision=%+v review=%+v", decision, review)
	}

	index := decisionsOf(t, dir)
	if len(index.Records) != 1 {
		t.Fatalf("records=%+v", index.Records)
	}
	record := index.Records[0]
	binding := record.Decision.ReviewBinding
	if binding == nil ||
		binding.Fingerprint != review.ReviewFingerprint ||
		binding.Projection == nil ||
		binding.Projection.ReviewFingerprint != review.ReviewFingerprint ||
		fingerprintReviewProjection(*binding.Projection) != review.ReviewFingerprint ||
		len(binding.Coverage.Incomplete) != len(review.Coverage.Incomplete) {
		t.Fatalf("persisted review binding=%+v review=%+v", binding, review.Coverage)
	}
	if record.ReviewBindingStatus != "verified" ||
		record.ReviewCoverageStatus != "incomplete" ||
		len(record.ReviewIncomplete) != len(review.Coverage.Incomplete) ||
		len(record.ReviewDiagnostics) != 0 {
		t.Fatalf(
			"review index status=%q coverage=%q incomplete=%+v diagnostics=%+v",
			record.ReviewBindingStatus,
			record.ReviewCoverageStatus,
			record.ReviewIncomplete,
			record.ReviewDiagnostics,
		)
	}
}

func TestReviewShowsUnresolvedFailureWithoutInventingLaterCoverage(t *testing.T) {
	dir := scratchWorkspace(t)
	candidate := unitPriceCandidate(revisionOfDir(t, dir))
	candidate.Delta[0].Attribute = "sales.order_line.no_such_attribute"

	ev, err := evaluate(dir, mustJSON(t, candidate))
	if err != nil {
		t.Fatal(err)
	}
	if ev.preview.Acceptable {
		t.Fatal("invalid candidate became acceptable")
	}

	review := buildReviewProjection(ev)
	if review.PreviewAcceptable {
		t.Fatal("review hid Preview rejection")
	}
	if review.Coverage.NotEvaluated == 0 || len(review.Coverage.Incomplete) == 0 {
		t.Fatalf("rejected review hid incomplete coverage: %+v", review.Coverage)
	}
	if reviewSection(t, review, "A1").Coverage != reviewCoverageNotEvaluated {
		t.Fatalf("A1 should be unavailable after resolve failure: %+v", reviewSection(t, review, "A1"))
	}
	if reviewSection(t, review, "A7").Coverage != reviewCoveragePartial {
		t.Fatalf("A7 should retain normalized candidate identity: %+v", reviewSection(t, review, "A7"))
	}
	if !sectionContains(reviewSection(t, review, "A3"), "ATTRIBUTE_NOT_FOUND") {
		t.Fatalf("A3 does not expose the failing diagnostic: %+v", reviewSection(t, review, "A3"))
	}
}

func reviewSection(t *testing.T, review ReviewProjection, code string) ReviewSection {
	t.Helper()
	for _, section := range review.Sections {
		if section.Code == code {
			return section
		}
	}
	t.Fatalf("review section %s not found: %+v", code, review.Sections)
	return ReviewSection{}
}

func decisionRefusalHasCode(decision Decision, want string) bool {
	for _, refusal := range decision.Refusals {
		if refusal.Code == want {
			return true
		}
	}
	return false
}

func sectionContains(section ReviewSection, want string) bool {
	for _, item := range section.Items {
		if strings.Contains(item, want) {
			return true
		}
	}
	return strings.Contains(section.Note, want)
}
