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
	if review.CandidateFingerprint == "" || review.EvidenceFingerprint == "" {
		t.Fatalf("review identities are incomplete: %+v", review)
	}
	if len(review.Sections) != 8 {
		t.Fatalf("sections=%d want 8", len(review.Sections))
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
	if a5.Coverage != reviewCoverageNotEvaluated {
		t.Fatalf("A5 coverage=%s", a5.Coverage)
	}

	a6 := reviewSection(t, review, "A6")
	if a6.Coverage != reviewCoverageNotEvaluated || !sectionContains(a6, "customGoBindings collection: unchanged") {
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
	if a5.Coverage != reviewCoverageNotEvaluated || !sectionContains(a5, "ENVIRONMENT_BINDING_NOT_AUTHORED") {
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
					"## A5 権限の変更 [未評価]",
					"## A6 native codeへ残る箇所 [未評価]",
					"## A8 承認単位の大きさ [表示可能]",
					"candidate fingerprint:",
					"evidence fingerprint:",
					"要約: semantic 1件 / application 0件 / environment 0件",
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
				if review.Schema != reviewSchema || len(review.Sections) != 8 {
					t.Fatalf("review=%+v", review)
				}
			}

			assertTreeUnchanged(t, before, treeDigest(t, dir))
		})
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
	if review.Acceptable {
		t.Fatal("review hid Preview rejection")
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

func sectionContains(section ReviewSection, want string) bool {
	for _, item := range section.Items {
		if strings.Contains(item, want) {
			return true
		}
	}
	return strings.Contains(section.Note, want)
}
