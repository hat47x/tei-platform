package knowledgecontext

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/index"
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
			BusinessIdentityCandidates: []DataDesignBusinessIdentityCandidate{
				{
					Fingerprint: "sha256:candidate",
					Relation:    "sales.order.lines",
					Components: []DataDesignDeterminantTerm{
						{Kind: "relation-source-occurrence", Data: "sales.order"},
						{Kind: "attribute", Data: "sales.order_line", Attribute: "sales.order_line.line_number"},
					},
					TargetOccurrence: "sales.order_line",
					BasisConstraint:  "sales.order.lines.line_number_unique",
					Scope:            "per-relation-source-occurrence",
					Nature:           "candidate-from-accepted-scoped-uniqueness",
					EvidenceNeeded:   []string{"relation-business-meaning"},
				},
			},
			AuthorityLimits: []string{"data-design-source-limit"},
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
		"selective-disclosure-is-not-access-control-or-redaction",
		"disclosure-fingerprint-is-cache-identity-not-semantic-proof",
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

	got.DataDesign.AuthorityLimits[0] = "mutated"
	got.DataDesign.BusinessIdentityCandidates[0].Components[0].Kind = "mutated"
	got.DataDesign.BusinessIdentityCandidates[0].EvidenceNeeded[0] = "mutated"
	if ctx.DataDesign.AuthorityLimits[0] != "data-design-source-limit" ||
		ctx.DataDesign.BusinessIdentityCandidates[0].Components[0].Kind != "relation-source-occurrence" ||
		ctx.DataDesign.BusinessIdentityCandidates[0].EvidenceNeeded[0] != "relation-business-meaning" {
		t.Fatalf("disclosure Data Design snapshot aliases source Context: source=%+v disclosure=%+v", ctx.DataDesign, got.DataDesign)
	}
}

func TestBuildDisclosureDetachesApplicationPatternDefinition(t *testing.T) {
	ctx := Context{
		Subject: subject.Subject{Kind: "application-pattern", ID: "order_entry.customer_reference"},
		ApplicationPattern: &index.ApplicationPatternIndexEntry{
			Application:   "order_entry",
			Pattern:       "order_entry.customer_reference",
			Kind:          "reference",
			Relation:      "sales.order.customer",
			Contract:      "master_data.customer",
			ContractQuery: "find_by_id",
			ContractQueryInputs: []index.ApplicationPatternContractQueryInputEntry{
				{
					TargetAttribute: core.CanonicalID("customer.customer.customer_id"),
					Input:           core.LocalName("id"),
				},
			},
		},
		AuthorityLimits: []string{"derived-context-is-not-canonical-authority"},
	}

	got, err := BuildDisclosure(ctx, []string{"definition"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ApplicationPattern == nil || len(got.ApplicationPattern.ContractQueryInputs) != 1 {
		t.Fatalf("Application Pattern definition is missing: %+v", got.ApplicationPattern)
	}
	got.ApplicationPattern.ContractQueryInputs[0].Input = "mutated"
	if ctx.ApplicationPattern.ContractQueryInputs[0].Input != "id" {
		t.Fatalf("disclosure Application Pattern snapshot aliases source Context: source=%+v disclosure=%+v", ctx.ApplicationPattern, got.ApplicationPattern)
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
