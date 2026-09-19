package main

import (
	"reflect"
	"sort"
	"testing"
)

type actorContextR3EPatternUseProbe struct {
	Procedure            string
	Step                 string
	ApplicationPattern   string
	RequirementStatus    string
	RequirementSource    string
	SupportingCapability string
}

func TestActorContextR3EPatternReuseDoesNotFillAnotherBusinessContextsUnresolvedRequirement(t *testing.T) {
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
			RequirementStatus:    actorContextR3BAuthorizationUnresolved,
			RequirementSource:    "",
			SupportingCapability: "",
		},
	}

	patternAffected := actorContextR3EUsesForPattern(uses, "order_entry.customer_reference")
	explicitRequired := actorContextR3EExplicitUsesByStatus(
		uses,
		"order_entry.customer_reference",
		actorContextR3BAuthorizationRequired,
	)

	if !reflect.DeepEqual(patternAffected, []string{
		"customer-inquiry.customer-confirmation/CI-01",
		"order-entry.order-handling/BP-02",
	}) {
		t.Fatalf("test setup no longer exposes two Business contexts sharing one Pattern: %v", patternAffected)
	}
	if !reflect.DeepEqual(explicitRequired, []string{"order-entry.order-handling/BP-02"}) {
		t.Fatalf("one explicit Business requirement leaked into another unresolved context: %v", explicitRequired)
	}
	if uses[1].RequirementStatus != actorContextR3BAuthorizationUnresolved {
		t.Fatalf("second Business context must remain unresolved, not inherited required/not-required: %+v", uses[1])
	}
}

func TestActorContextR3EMatchingIndependentRequirementsCreateDuplicationPressureOnly(t *testing.T) {
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
		t.Fatalf("expected two independently authored required contexts, got %v", explicitRequired)
	}
	if uses[0].RequirementSource == uses[1].RequirementSource {
		t.Fatalf("matching status must retain independent Business provenance: %+v", uses)
	}
	if uses[0].SupportingCapability != uses[1].SupportingCapability ||
		uses[0].SupportingCapability != "security.authorize" {
		t.Fatalf("test setup changed: expected the same supporting capability, got %+v", uses)
	}

	// Two independent Business authorities agreeing creates duplication pressure.
	// It does not mutate either source into a Pattern-level authority.
	for _, use := range uses {
		if use.RequirementSource == "application-pattern:order_entry.customer_reference" {
			t.Fatalf("independent Business authority was silently rewritten as Pattern authority: %+v", use)
		}
	}
}

func TestActorContextR3EUnresolvedIsNeitherRequiredNorNotRequired(t *testing.T) {
	use := actorContextR3EPatternUseProbe{
		Procedure:          "customer-inquiry.customer-confirmation",
		Step:               "CI-01",
		ApplicationPattern: "order_entry.customer_reference",
		RequirementStatus:  actorContextR3BAuthorizationUnresolved,
	}

	if use.RequirementStatus == actorContextR3BAuthorizationRequired {
		t.Fatalf("unresolved requirement was treated as inherited required: %+v", use)
	}
	if use.RequirementStatus == actorContextR3BAuthorizationNotRequired {
		t.Fatalf("unresolved requirement was treated as explicit not-required: %+v", use)
	}
}

func actorContextR3EUsesForPattern(
	uses []actorContextR3EPatternUseProbe,
	pattern string,
) []string {
	var affected []string
	for _, use := range uses {
		if use.ApplicationPattern != pattern {
			continue
		}
		affected = append(affected, use.Procedure+"/"+use.Step)
	}
	sort.Strings(affected)
	return affected
}

func actorContextR3EExplicitUsesByStatus(
	uses []actorContextR3EPatternUseProbe,
	pattern string,
	status string,
) []string {
	var affected []string
	for _, use := range uses {
		if use.ApplicationPattern != pattern || use.RequirementStatus != status {
			continue
		}
		if use.RequirementSource == "" {
			continue
		}
		affected = append(affected, use.Procedure+"/"+use.Step)
	}
	sort.Strings(affected)
	return affected
}
