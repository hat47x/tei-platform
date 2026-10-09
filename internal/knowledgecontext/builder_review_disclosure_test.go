package knowledgecontext

import (
	"reflect"
	"strings"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/subject"
)

func TestBuildBuilderReviewDisclosureMatchesDirectCurrentDisclosure(t *testing.T) {
	ctx, sources := builderReviewHandoffApplicationFixture(
		"sha256:pattern",
		"sha256:dependency",
	)
	requirements := QualificationRequirements{
		ExpectedKind: "application-pattern",
		Application:  "order_entry",
		RequiredDependencies: []DependencyRequirement{
			{
				Direction: "outgoing",
				Relation:  "contract-query",
				Kind:      "database-contract-query",
				ID:        "master_data.customer/find_by_id",
			},
		},
	}
	handoff, err := BuildBuilderReviewHandoff(ctx, sources, requirements)
	if err != nil {
		t.Fatal(err)
	}
	if handoff.Status != "qualified-for-disclosure" {
		t.Fatalf("fixture did not qualify for disclosure: %+v", handoff)
	}

	got, err := BuildBuilderReviewDisclosure(ctx, sources, handoff)
	if err != nil {
		t.Fatal(err)
	}
	want, err := BuildDisclosure(ctx, handoff.SelectedSections)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("qualified disclosure changed existing Disclosure semantics:\n got=%+v\nwant=%+v", got, want)
	}
	if got.Status != "complete" ||
		len(got.UnavailableSections) != 0 ||
		len(got.OmittedAvailableSections) != 0 {
		t.Fatalf("review-selected disclosure is not complete over current available Knowledge: %+v", got)
	}
}

func TestBuildBuilderReviewDisclosureRejectsBlockedHandoff(t *testing.T) {
	ctx := Context{
		Subject:    subject.Subject{Kind: "semantic-data", ID: "missing.example"},
		Unresolved: []string{"source-reference-unavailable"},
	}
	sources := sourceindex.Index{}
	handoff, err := BuildBuilderReviewHandoff(
		ctx,
		sources,
		QualificationRequirements{ExpectedKind: "semantic-data"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if handoff.Status != "blocked-unresolved" {
		t.Fatalf("fixture handoff status=%q", handoff.Status)
	}

	_, err = BuildBuilderReviewDisclosure(ctx, sources, handoff)
	if err == nil || !strings.Contains(err.Error(), "blocked by handoff status") {
		t.Fatalf("blocked handoff disclosed Knowledge: %v", err)
	}
}

func TestBuildBuilderReviewDisclosureRejectsStaleHandoff(t *testing.T) {
	oldCtx, oldSources := builderReviewHandoffApplicationFixture(
		"sha256:pattern-old",
		"sha256:dependency",
	)
	handoff, err := BuildBuilderReviewHandoff(
		oldCtx,
		oldSources,
		QualificationRequirements{
			ExpectedKind: "application-pattern",
			Application:  "order_entry",
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	currentCtx, currentSources := builderReviewHandoffApplicationFixture(
		"sha256:pattern-current",
		"sha256:dependency",
	)
	_, err = BuildBuilderReviewDisclosure(currentCtx, currentSources, handoff)
	if err == nil || !strings.Contains(err.Error(), "rejects stale handoff") {
		t.Fatalf("stale handoff disclosed current Knowledge: %v", err)
	}
}

func TestBuildBuilderReviewDisclosureRejectsModifiedHandoff(t *testing.T) {
	ctx, sources := builderReviewHandoffApplicationFixture(
		"sha256:pattern",
		"sha256:dependency",
	)
	handoff, err := BuildBuilderReviewHandoff(
		ctx,
		sources,
		QualificationRequirements{
			ExpectedKind: "application-pattern",
			Application:  "order_entry",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	handoff.SelectedSections = append([]string(nil), handoff.SelectedSections...)
	handoff.SelectedSections[0] = "modified"

	_, err = BuildBuilderReviewDisclosure(ctx, sources, handoff)
	if err == nil || !strings.Contains(err.Error(), "modified Builder Review Handoff") {
		t.Fatalf("modified handoff disclosed Knowledge: %v", err)
	}
}

func TestBuildBuilderReviewDisclosureRejectsSubjectMismatch(t *testing.T) {
	ctx, sources := builderReviewHandoffApplicationFixture(
		"sha256:pattern",
		"sha256:dependency",
	)
	handoff, err := BuildBuilderReviewHandoff(
		ctx,
		sources,
		QualificationRequirements{
			ExpectedKind: "application-pattern",
			Application:  "order_entry",
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	other := ctx
	other.Subject = subject.Subject{
		Kind: "application-pattern",
		ID:   "order_entry.other",
	}
	_, err = BuildBuilderReviewDisclosure(other, sources, handoff)
	if err == nil || !strings.Contains(err.Error(), "subject mismatch") {
		t.Fatalf("handoff was reused for a different subject: %v", err)
	}
}
