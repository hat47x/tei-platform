package main

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

const (
	actorContextR3GPrecedenceUnresolved     = "unresolved"
	actorContextR3GPrecedencePatternMinimum = "pattern-minimum"
	actorContextR3GPrecedenceStepException  = "step-exception"

	actorContextR3GOutcomeConflictUnresolved = "conflict-unresolved"
)

type actorContextR3GPatternMinimumProbe struct {
	ApplicationPattern   string
	Status               string
	Source               string
	SupportingCapability string
}

type actorContextR3GBusinessVariationProbe struct {
	Procedure       string
	Step            string
	Status          string
	Source          string
	ExceptionIntent bool
}

type actorContextR3GResolutionProbe struct {
	Status   string
	Conflict bool
	Sources  []string
}

func TestActorContextR3GRepeatedBusinessRequirementsDoNotCreatePatternAuthority(t *testing.T) {
	uses := []actorContextR3EPatternUseProbe{
		{
			Procedure:            "order-entry.order-handling",
			Step:                 "BP-02",
			ApplicationPattern:   "order_entry.customer_reference",
			RequirementStatus:    actorContextR3BAuthorizationRequired,
			RequirementSource:    "business-step:order-entry.order-handling/BP-02",
			SupportingCapability: "security.authorize",
		},
		{
			Procedure:            "customer-inquiry.customer-confirmation",
			Step:                 "CI-01",
			ApplicationPattern:   "order_entry.customer_reference",
			RequirementStatus:    actorContextR3BAuthorizationRequired,
			RequirementSource:    "business-step:customer-inquiry.customer-confirmation/CI-01",
			SupportingCapability: "security.authorize",
		},
	}

	explicitRequired := actorContextR3EExplicitUsesByStatus(
		uses,
		"order_entry.customer_reference",
		actorContextR3BAuthorizationRequired,
	)
	if len(explicitRequired) != 2 {
		t.Fatalf("test setup changed: expected two independently explicit Business requirements, got %v", explicitRequired)
	}

	for _, use := range uses {
		if use.RequirementSource == "application-pattern:order_entry.customer_reference" {
			t.Fatalf("repeated Business requirements were silently promoted to Pattern authority: %+v", use)
		}
	}
}

func TestActorContextR3GMatchingMinimumAndBusinessRequirementPreserveProvenance(t *testing.T) {
	pattern := actorContextR3GPatternMinimumProbe{
		ApplicationPattern:   "order_entry.customer_reference",
		Status:               actorContextR3BAuthorizationRequired,
		Source:               "application-pattern:order_entry.customer_reference",
		SupportingCapability: "security.authorize",
	}
	step := actorContextR3GBusinessVariationProbe{
		Procedure: "order-entry.order-handling",
		Step:      "BP-02",
		Status:    actorContextR3BAuthorizationRequired,
		Source:    "business-step:order-entry.order-handling/BP-02",
	}

	got, err := actorContextR3GResolveRequirement(pattern, step, actorContextR3GPrecedenceUnresolved)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != actorContextR3BAuthorizationRequired || got.Conflict {
		t.Fatalf("matching Pattern/Business requirements should not conflict: %+v", got)
	}

	wantSources := []string{
		"application-pattern:order_entry.customer_reference",
		"business-step:order-entry.order-handling/BP-02",
	}
	if !reflect.DeepEqual(got.Sources, wantSources) {
		t.Fatalf("matching requirements collapsed distinct provenance: got=%v want=%v", got.Sources, wantSources)
	}
}

func TestActorContextR3GBusinessAbsenceDoesNotBecomeNotRequired(t *testing.T) {
	pattern := actorContextR3GPatternMinimumProbe{
		ApplicationPattern:   "order_entry.customer_reference",
		Status:               actorContextR3BAuthorizationRequired,
		Source:               "application-pattern:order_entry.customer_reference",
		SupportingCapability: "security.authorize",
	}
	step := actorContextR3GBusinessVariationProbe{
		Procedure: "customer-inquiry.customer-confirmation",
		Step:      "CI-01",
		Status:    actorContextR3BAuthorizationUnresolved,
	}

	got, err := actorContextR3GResolveRequirement(pattern, step, actorContextR3GPrecedenceUnresolved)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != actorContextR3BAuthorizationRequired || got.Conflict {
		t.Fatalf("unresolved Business authoring negated Pattern minimum candidate: %+v", got)
	}
	if !reflect.DeepEqual(got.Sources, []string{"application-pattern:order_entry.customer_reference"}) {
		t.Fatalf("unresolved Business Step fabricated provenance: %+v", got)
	}
}

func TestActorContextR3GSpecificScopeDoesNotImplyPrecedence(t *testing.T) {
	pattern := actorContextR3GPatternMinimumProbe{
		ApplicationPattern:   "order_entry.customer_reference",
		Status:               actorContextR3BAuthorizationRequired,
		Source:               "application-pattern:order_entry.customer_reference",
		SupportingCapability: "security.authorize",
	}
	step := actorContextR3GBusinessVariationProbe{
		Procedure: "customer-inquiry.customer-confirmation",
		Step:      "CI-01",
		Status:    actorContextR3BAuthorizationNotRequired,
		Source:    "business-step:customer-inquiry.customer-confirmation/CI-01",
	}

	got, err := actorContextR3GResolveRequirement(pattern, step, actorContextR3GPrecedenceUnresolved)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Conflict || got.Status != actorContextR3GOutcomeConflictUnresolved {
		t.Fatalf("more-specific Business scope silently became precedence: %+v", got)
	}
}

func TestActorContextR3GExceptionIntentStillNeedsPrecedenceAuthority(t *testing.T) {
	pattern := actorContextR3GPatternMinimumProbe{
		ApplicationPattern:   "order_entry.customer_reference",
		Status:               actorContextR3BAuthorizationRequired,
		Source:               "application-pattern:order_entry.customer_reference",
		SupportingCapability: "security.authorize",
	}
	step := actorContextR3GBusinessVariationProbe{
		Procedure:       "customer-inquiry.customer-confirmation",
		Step:            "CI-01",
		Status:          actorContextR3BAuthorizationNotRequired,
		Source:          "business-step:customer-inquiry.customer-confirmation/CI-01",
		ExceptionIntent: true,
	}

	unresolved, err := actorContextR3GResolveRequirement(pattern, step, actorContextR3GPrecedenceUnresolved)
	if err != nil {
		t.Fatal(err)
	}
	if !unresolved.Conflict || unresolved.Status != actorContextR3GOutcomeConflictUnresolved {
		t.Fatalf("explicit exception intent was treated as accepted precedence: %+v", unresolved)
	}

	patternWins, err := actorContextR3GResolveRequirement(pattern, step, actorContextR3GPrecedencePatternMinimum)
	if err != nil {
		t.Fatal(err)
	}
	stepWins, err := actorContextR3GResolveRequirement(pattern, step, actorContextR3GPrecedenceStepException)
	if err != nil {
		t.Fatal(err)
	}
	if patternWins.Status != actorContextR3BAuthorizationRequired || patternWins.Conflict {
		t.Fatalf("test-only Pattern precedence candidate was not distinct: %+v", patternWins)
	}
	if stepWins.Status != actorContextR3BAuthorizationNotRequired || stepWins.Conflict {
		t.Fatalf("test-only Step exception candidate was not distinct: %+v", stepWins)
	}
	if !reflect.DeepEqual(patternWins.Sources, stepWins.Sources) {
		t.Fatalf("precedence comparison erased provenance: pattern=%v step=%v", patternWins.Sources, stepWins.Sources)
	}
}

func actorContextR3GResolveRequirement(
	pattern actorContextR3GPatternMinimumProbe,
	step actorContextR3GBusinessVariationProbe,
	precedence string,
) (actorContextR3GResolutionProbe, error) {
	if precedence != actorContextR3GPrecedenceUnresolved &&
		precedence != actorContextR3GPrecedencePatternMinimum &&
		precedence != actorContextR3GPrecedenceStepException {
		return actorContextR3GResolutionProbe{}, fmt.Errorf("unsupported R3G precedence candidate %q", precedence)
	}
	if pattern.Status != "" {
		if pattern.ApplicationPattern == "" || pattern.Source == "" {
			return actorContextR3GResolutionProbe{}, fmt.Errorf("Pattern minimum candidate requires Pattern identity and provenance")
		}
		if pattern.Status != actorContextR3BAuthorizationRequired {
			return actorContextR3GResolutionProbe{}, fmt.Errorf("R3G only pressures required Pattern minimum, got %q", pattern.Status)
		}
		if pattern.SupportingCapability != "security.authorize" {
			return actorContextR3GResolutionProbe{}, fmt.Errorf("Pattern minimum candidate must name security.authorize")
		}
	}
	if step.Procedure == "" || step.Step == "" {
		return actorContextR3GResolutionProbe{}, fmt.Errorf("Business variation requires Procedure and Step")
	}
	if step.Status != actorContextR3BAuthorizationUnresolved &&
		step.Status != actorContextR3BAuthorizationRequired &&
		step.Status != actorContextR3BAuthorizationNotRequired {
		return actorContextR3GResolutionProbe{}, fmt.Errorf("unsupported Business variation status %q", step.Status)
	}
	if step.Status != actorContextR3BAuthorizationUnresolved && step.Source == "" {
		return actorContextR3GResolutionProbe{}, fmt.Errorf("explicit Business variation requires provenance")
	}

	var sources []string
	if pattern.Source != "" {
		sources = append(sources, pattern.Source)
	}
	if step.Source != "" {
		sources = append(sources, step.Source)
	}
	sort.Strings(sources)

	if pattern.Status == "" {
		return actorContextR3GResolutionProbe{Status: step.Status, Sources: sources}, nil
	}
	switch step.Status {
	case actorContextR3BAuthorizationUnresolved:
		return actorContextR3GResolutionProbe{Status: actorContextR3BAuthorizationRequired, Sources: sources}, nil
	case actorContextR3BAuthorizationRequired:
		return actorContextR3GResolutionProbe{Status: actorContextR3BAuthorizationRequired, Sources: sources}, nil
	case actorContextR3BAuthorizationNotRequired:
		if !step.ExceptionIntent || precedence == actorContextR3GPrecedenceUnresolved {
			return actorContextR3GResolutionProbe{
				Status:   actorContextR3GOutcomeConflictUnresolved,
				Conflict: true,
				Sources:  sources,
			}, nil
		}
		switch precedence {
		case actorContextR3GPrecedencePatternMinimum:
			return actorContextR3GResolutionProbe{Status: actorContextR3BAuthorizationRequired, Sources: sources}, nil
		case actorContextR3GPrecedenceStepException:
			return actorContextR3GResolutionProbe{Status: actorContextR3BAuthorizationNotRequired, Sources: sources}, nil
		}
	}
	return actorContextR3GResolutionProbe{}, fmt.Errorf("unhandled R3G resolution state")
}
