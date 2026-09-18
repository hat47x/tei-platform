package main

import (
	"path/filepath"
	"reflect"
	"testing"
)

const (
	actorContextR3LGateInsufficientEvidence       = "insufficient-evidence"
	actorContextR3LGateSingleQualifyingInteraction = "single-qualifying-interaction"
	actorContextR3LGateMultipleNondiscriminating   = "multiple-nondiscriminating-interactions"
	actorContextR3LGateQualified                   = "action-locus-pressure-qualified"
)

type actorContextR3LInteractionEvidenceProbe struct {
	BusinessAddress        string
	RequirementStatus      string
	Resource               string
	ResourceSource         string
	Action                 string
	ActionSource           string
	RepositoryBacked       bool
	ProviderIndependent    bool
	PatternKind            string
	ReuseShape             string
	BusinessSpecificAction bool
}

type actorContextR3LActionLocusGateProbe struct {
	Classification      string
	QualifyingAddresses []string
	BlockingReasons     []string
	AuthorityLimits     []string
}

func TestActorContextR3LCurrentRepositoryDoesNotQualifyActionLocusSelection(t *testing.T) {
	root := repoPath()
	primary, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	followUp, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-follow-up-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}

	bp06Requirement, err := deriveActorContextR3BAuthorizationRequirementProbe(primary, "BP-06")
	if err != nil {
		t.Fatal(err)
	}
	bp00Requirement, err := deriveActorContextR3BAuthorizationRequirementProbe(primary, "BP-00")
	if err != nil {
		t.Fatal(err)
	}
	fu00Requirement, err := deriveActorContextR3BAuthorizationRequirementProbe(followUp, "FU-00")
	if err != nil {
		t.Fatal(err)
	}

	got := actorContextR3LEvaluateActionLocusGate([]actorContextR3LInteractionEvidenceProbe{
		{
			BusinessAddress:     "order-entry.order-handling/BP-06",
			RequirementStatus:   bp06Requirement.Status,
			Resource:            "order_entry.submit_order",
			ResourceSource:      "reference-pressure-overlay:BP-06",
			Action:              "",
			ActionSource:        "",
			RepositoryBacked:    true,
			ProviderIndependent: false,
			PatternKind:         "command",
			ReuseShape:          "single-use",
		},
		{
			BusinessAddress:     "order-entry.order-handling/BP-00",
			RequirementStatus:   bp00Requirement.Status,
			RepositoryBacked:    true,
			ProviderIndependent: false,
			PatternKind:         "browse",
			ReuseShape:          "cross-procedure-reuse",
		},
		{
			BusinessAddress:     "order-entry.follow-up/FU-00",
			RequirementStatus:   fu00Requirement.Status,
			RepositoryBacked:    true,
			ProviderIndependent: false,
			PatternKind:         "browse",
			ReuseShape:          "cross-procedure-reuse",
		},
	})

	if got.Classification != actorContextR3LGateInsufficientEvidence {
		t.Fatalf("current Repository unexpectedly qualified action locus comparison: %+v", got)
	}
	if len(got.QualifyingAddresses) != 0 {
		t.Fatalf("current nonqualifying interactions were counted as action-authority records: %+v", got)
	}
}

func TestActorContextR3LOneQualifyingInteractionDoesNotSelectReusableLocus(t *testing.T) {
	got := actorContextR3LEvaluateActionLocusGate([]actorContextR3LInteractionEvidenceProbe{
		actorContextR3LQualifyingFixture(
			"order-entry.order-handling/BP-06",
			"execute",
			"command",
			"single-use",
			false,
		),
	})

	if got.Classification != actorContextR3LGateSingleQualifyingInteraction {
		t.Fatalf("one qualifying interaction should not open reusable locus comparison: %+v", got)
	}
}

func TestActorContextR3LRepeatedEquivalentInteractionsStillDoNotQualifyLocusComparison(t *testing.T) {
	got := actorContextR3LEvaluateActionLocusGate([]actorContextR3LInteractionEvidenceProbe{
		actorContextR3LQualifyingFixture(
			"example.procedure/EX-01",
			"execute",
			"command",
			"single-use",
			false,
		),
		actorContextR3LQualifyingFixture(
			"example.procedure/EX-02",
			"execute",
			"command",
			"single-use",
			false,
		),
	})

	if got.Classification != actorContextR3LGateMultipleNondiscriminating {
		t.Fatalf("duplicated equivalent interactions were treated as locus-discriminating Evidence: %+v", got)
	}
}

func TestActorContextR3LTwoQualifyingInteractionsNeedDiscriminatingVariation(t *testing.T) {
	got := actorContextR3LEvaluateActionLocusGate([]actorContextR3LInteractionEvidenceProbe{
		actorContextR3LQualifyingFixture(
			"example.procedure/EX-01",
			"execute",
			"command",
			"single-use",
			false,
		),
		actorContextR3LQualifyingFixture(
			"example.follow-up/EX-02",
			"browse",
			"browse",
			"cross-procedure-reuse",
			true,
		),
	})

	if got.Classification != actorContextR3LGateQualified {
		t.Fatalf("two qualifying interactions with discriminating variation did not open locus comparison pressure: %+v", got)
	}

	typ := reflect.TypeOf(got)
	for _, forbidden := range []string{
		"Winner",
		"PreferredLocus",
		"Confidence",
		"SelectedLocus",
		"Authorized",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R3L Evidence Gate introduced forbidden selection/runtime shortcut field %q", forbidden)
		}
	}
}

func TestActorContextR3LPolicyDependentActionDoesNotQualify(t *testing.T) {
	record := actorContextR3LQualifyingFixture(
		"example.procedure/EX-01",
		"execute",
		"command",
		"single-use",
		false,
	)
	record.ProviderIndependent = false
	record.ActionSource = "legacy-policy:execute/example.resource"

	got := actorContextR3LEvaluateActionLocusGate([]actorContextR3LInteractionEvidenceProbe{record})
	if got.Classification != actorContextR3LGateInsufficientEvidence {
		t.Fatalf("policy-dependent action was treated as provider-independent authority: %+v", got)
	}
	if !actorContextR0Contains(got.BlockingReasons, "provider-independent-action-authority-unresolved") {
		t.Fatalf("R3L lost provider-independence blocker: %+v", got.BlockingReasons)
	}
}

func actorContextR3LQualifyingFixture(
	address string,
	action string,
	patternKind string,
	reuseShape string,
	businessSpecificAction bool,
) actorContextR3LInteractionEvidenceProbe {
	return actorContextR3LInteractionEvidenceProbe{
		BusinessAddress:        address,
		RequirementStatus:      actorContextR3BAuthorizationRequired,
		Resource:               "example.resource",
		ResourceSource:         "business-application:protected-resource",
		Action:                 action,
		ActionSource:           "business-application:authorization-action",
		RepositoryBacked:       true,
		ProviderIndependent:    true,
		PatternKind:            patternKind,
		ReuseShape:             reuseShape,
		BusinessSpecificAction: businessSpecificAction,
	}
}

func actorContextR3LEvaluateActionLocusGate(
	records []actorContextR3LInteractionEvidenceProbe,
) actorContextR3LActionLocusGateProbe {
	result := actorContextR3LActionLocusGateProbe{
		AuthorityLimits: []string{
			"policy-does-not-backfill-action-authority",
			"pattern-kind-does-not-backfill-action-authority",
			"one-interaction-does-not-establish-reusable-locus",
			"repetition-without-variation-does-not-qualify-locus-comparison",
			"qualified-locus-pressure-does-not-select-winner",
		},
	}

	var qualifying []actorContextR3LInteractionEvidenceProbe
	for _, record := range records {
		recordQualifies := true
		if record.RequirementStatus != actorContextR3BAuthorizationRequired {
			recordQualifies = false
			result.BlockingReasons = append(result.BlockingReasons, "required-authorization-interaction-missing")
		}
		if record.Resource == "" || record.ResourceSource == "" {
			recordQualifies = false
			result.BlockingReasons = append(result.BlockingReasons, "protected-resource-authority-unresolved")
		}
		if record.Action == "" || record.ActionSource == "" {
			recordQualifies = false
			result.BlockingReasons = append(result.BlockingReasons, "authorization-action-authority-unresolved")
		}
		if !record.RepositoryBacked {
			recordQualifies = false
			result.BlockingReasons = append(result.BlockingReasons, "repository-backed-interaction-unresolved")
		}
		if !record.ProviderIndependent {
			recordQualifies = false
			result.BlockingReasons = append(result.BlockingReasons, "provider-independent-action-authority-unresolved")
		}
		if recordQualifies {
			qualifying = append(qualifying, record)
			result.QualifyingAddresses = append(result.QualifyingAddresses, record.BusinessAddress)
		}
	}

	result.QualifyingAddresses = actorContextR0SortedUnique(result.QualifyingAddresses)
	result.BlockingReasons = actorContextR0SortedUnique(result.BlockingReasons)

	switch len(qualifying) {
	case 0:
		result.Classification = actorContextR3LGateInsufficientEvidence
		return result
	case 1:
		result.Classification = actorContextR3LGateSingleQualifyingInteraction
		return result
	}

	base := qualifying[0]
	discriminating := false
	for _, record := range qualifying[1:] {
		if record.Action != base.Action ||
			record.PatternKind != base.PatternKind ||
			record.ReuseShape != base.ReuseShape ||
			record.BusinessSpecificAction != base.BusinessSpecificAction {
			discriminating = true
			break
		}
	}
	if !discriminating {
		result.Classification = actorContextR3LGateMultipleNondiscriminating
		return result
	}

	result.Classification = actorContextR3LGateQualified
	return result
}
