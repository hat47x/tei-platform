package knowledgecontext

import (
	"strings"
	"testing"

	"tei/internal/subject"
)

func TestBuildDisclosureSeparatesSelectedDisclosedAndOmittedKnowledge(t *testing.T) {
	ctx := Context{
		Subject: subject.Subject{Kind: "semantic-data", ID: "sales.order_line"},
		DataDesign: &DataDesignContext{
			ProjectionRevision:     dataDesignProjectionRevision,
			Fingerprint:            "sha256:data-design",
			ResolutionStatus:       "resolved",
			EvidenceCoverageStatus: "complete",
			Data:                   "sales.order_line",
		},
		Outgoing: []Edge{
			{
				From:     subject.Subject{Kind: "semantic-data", ID: "sales.order_line"},
				Relation: "example",
				To:       subject.Subject{Kind: "semantic-data", ID: "sales.order"},
			},
		},
		Unresolved:      []string{"business-identity-still-unresolved"},
		AuthorityLimits: []string{"derived-context-is-not-canonical-authority"},
	}

	got, err := BuildDisclosure(ctx, []string{"data-design"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ProjectionRevision != knowledgeDisclosureProjectionRevision ||
		got.Status != "complete" ||
		got.Subject != ctx.Subject ||
		got.DataDesign == nil ||
		got.DataDesign.Data != "sales.order_line" ||
		len(got.DisclosedSections) != 1 ||
		got.DisclosedSections[0] != "data-design" ||
		len(got.OmittedAvailableSections) != 1 ||
		got.OmittedAvailableSections[0] != "dependencies" ||
		len(got.UnavailableSections) != 0 {
		t.Fatalf("disclosure=%+v", got)
	}
	if len(got.Outgoing) != 0 || len(got.Incoming) != 0 {
		t.Fatalf("omitted dependencies leaked into disclosure: outgoing=%+v incoming=%+v", got.Outgoing, got.Incoming)
	}
	if len(got.Unresolved) != 1 ||
		got.Unresolved[0] != "business-identity-still-unresolved" {
		t.Fatalf("mandatory unresolved state was omitted: %+v", got.Unresolved)
	}
	for _, limit := range []string{
		"derived-context-is-not-canonical-authority",
		"selective-disclosure-is-derived-not-authority",
		"selection-does-not-change-qualification-or-canonical-meaning",
		"omitted-knowledge-is-not-negative-evidence",
		"unavailable-requested-section-is-not-empty-meaning",
	} {
		if !containsDisclosureString(got.AuthorityLimits, limit) {
			t.Fatalf("authority limit %q is missing: %+v", limit, got.AuthorityLimits)
		}
	}
	if !strings.HasPrefix(got.Fingerprint, "sha256:") {
		t.Fatalf("disclosure fingerprint=%q", got.Fingerprint)
	}
	if len(ctx.AuthorityLimits) != 1 {
		t.Fatalf("source Context was mutated: %+v", ctx.AuthorityLimits)
	}
}

func TestBuildDisclosureReportsUnavailableRequestedKnowledge(t *testing.T) {
	ctx := Context{
		Subject: subject.Subject{Kind: "semantic-data", ID: "sales.order_line"},
		DataDesign: &DataDesignContext{
			ProjectionRevision:     dataDesignProjectionRevision,
			Fingerprint:            "sha256:data-design",
			ResolutionStatus:       "resolved",
			EvidenceCoverageStatus: "complete",
			Data:                   "sales.order_line",
		},
		Outgoing: []Edge{
			{
				From:     subject.Subject{Kind: "semantic-data", ID: "sales.order_line"},
				Relation: "example",
				To:       subject.Subject{Kind: "semantic-data", ID: "sales.order"},
			},
		},
	}

	got, err := BuildDisclosure(ctx, []string{" source, data-design ", "source"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "partial" ||
		len(got.SelectedSections) != 2 ||
		got.SelectedSections[0] != "data-design" ||
		got.SelectedSections[1] != "source" ||
		len(got.DisclosedSections) != 1 ||
		got.DisclosedSections[0] != "data-design" ||
		len(got.UnavailableSections) != 1 ||
		got.UnavailableSections[0] != "source" ||
		len(got.OmittedAvailableSections) != 1 ||
		got.OmittedAvailableSections[0] != "dependencies" {
		t.Fatalf("partial disclosure=%+v", got)
	}
	if len(got.Artifacts) != 0 || len(got.References) != 0 {
		t.Fatalf("unavailable source section leaked empty payload as disclosed knowledge: %+v", got)
	}
}

func TestDisclosureFingerprintIsIndependentOfSelectionOrder(t *testing.T) {
	ctx := Context{
		Subject: subject.Subject{Kind: "semantic-data", ID: "sales.order_line"},
		DataDesign: &DataDesignContext{
			ProjectionRevision:     dataDesignProjectionRevision,
			Fingerprint:            "sha256:data-design",
			ResolutionStatus:       "resolved",
			EvidenceCoverageStatus: "complete",
			Data:                   "sales.order_line",
		},
		Outgoing: []Edge{
			{
				From:     subject.Subject{Kind: "semantic-data", ID: "sales.order_line"},
				Relation: "example",
				To:       subject.Subject{Kind: "semantic-data", ID: "sales.order"},
			},
		},
	}

	first, err := BuildDisclosure(ctx, []string{"data-design,dependencies"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := BuildDisclosure(ctx, []string{"dependencies", "data-design"})
	if err != nil {
		t.Fatal(err)
	}
	if first.Fingerprint != second.Fingerprint {
		t.Fatalf("selection order changed disclosure identity: first=%s second=%s", first.Fingerprint, second.Fingerprint)
	}
}

func TestBuildDisclosureRejectsUnknownOrEmptySelection(t *testing.T) {
	ctx := Context{Subject: subject.Subject{Kind: "semantic-data", ID: "sales.order_line"}}
	if _, err := BuildDisclosure(ctx, []string{"data-design,secret"}); err == nil ||
		!strings.Contains(err.Error(), "unknown knowledge disclosure section") {
		t.Fatalf("unknown disclosure section was not rejected: %v", err)
	}
	if _, err := BuildDisclosure(ctx, []string{" , "}); err == nil ||
		!strings.Contains(err.Error(), "requires at least one selected section") {
		t.Fatalf("empty disclosure selection was not rejected: %v", err)
	}
}

func containsDisclosureString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
