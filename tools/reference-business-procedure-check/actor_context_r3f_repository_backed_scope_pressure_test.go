package main

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
)

type actorContextR3FRequirementScopeCandidateProbe struct {
	Scope              string
	Procedure          string
	Step               string
	ApplicationPattern string
}

func TestActorContextR3FExistingReferencesExposeCrossProcedurePatternReuse(t *testing.T) {
	root := repoPath()
	primary, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	followUp, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-follow-up-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	corpus := []fixture{primary, followUp}

	patternScoped, err := actorContextR3FAffectedAddresses(corpus, actorContextR3FRequirementScopeCandidateProbe{
		Scope:              actorContextR3DScopeApplicationPattern,
		ApplicationPattern: "order_entry.recent_orders",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(patternScoped, []string{
		"order-entry.follow-up/FU-00",
		"order-entry.order-handling/BP-00",
	}) {
		t.Fatalf("existing Reference no longer exposes cross-Procedure recent_orders reuse: %v", patternScoped)
	}

	stepScoped, err := actorContextR3FAffectedAddresses(corpus, actorContextR3FRequirementScopeCandidateProbe{
		Scope:     actorContextR3DScopeBusinessStep,
		Procedure: "order-entry.order-handling",
		Step:      "BP-00",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(stepScoped, []string{"order-entry.order-handling/BP-00"}) {
		t.Fatalf("Business Step scope leaked across Procedure context: %v", stepScoped)
	}
	if reflect.DeepEqual(stepScoped, patternScoped) {
		t.Fatalf("repository-backed Step and Pattern scopes collapsed: step=%v pattern=%v", stepScoped, patternScoped)
	}

	primaryStep, err := actorContextR3FFindStep(primary, "BP-00")
	if err != nil {
		t.Fatal(err)
	}
	followUpStep, err := actorContextR3FFindStep(followUp, "FU-00")
	if err != nil {
		t.Fatal(err)
	}
	if primaryStep.BusinessAction == followUpStep.BusinessAction {
		t.Fatalf("same Pattern reuse no longer carries distinct Business Action wording: primary=%q follow-up=%q",
			primaryStep.BusinessAction, followUpStep.BusinessAction)
	}
}

func TestActorContextR3FSharedPolicyTargetDoesNotAuthorRequirementOrScope(t *testing.T) {
	root := repoPath()
	primary, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	followUp, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-follow-up-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	policy, err := readLegacyAuthorizationFixtureProbe(filepath.Join(root, "reference", "legacy-auth", "authorization.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !legacyAuthorizationAllowsTarget(policy, "browse", "order_entry.recent_orders") {
		t.Fatal("test setup changed: current Legacy Authorization policy lacks browse/allow for order_entry.recent_orders")
	}

	primaryRequirement, err := deriveActorContextR3BAuthorizationRequirementProbe(primary, "BP-00")
	if err != nil {
		t.Fatal(err)
	}
	followUpRequirement, err := deriveActorContextR3BAuthorizationRequirementProbe(followUp, "FU-00")
	if err != nil {
		t.Fatal(err)
	}
	for address, requirement := range map[string]actorContextR3BAuthorizationRequirementProbe{
		"order-entry.order-handling/BP-00": primaryRequirement,
		"order-entry.follow-up/FU-00":      followUpRequirement,
	} {
		if requirement.Status != actorContextR3BAuthorizationUnresolved {
			t.Fatalf("Legacy policy presence authored Authorization applicability for %s: %+v", address, requirement)
		}
		if requirement.RequirementSource != "" || requirement.SupportingCapability != "" {
			t.Fatalf("Legacy policy presence invented Business requirement provenance for %s: %+v", address, requirement)
		}
	}
}

func TestActorContextR3FExistingReuseDoesNotSatisfyIndependentSecondCorpusEvidence(t *testing.T) {
	root := repoPath()
	primary, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	followUp, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-follow-up-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}

	if primary.Application != "order_entry" || followUp.Application != "order_entry" {
		t.Fatalf("test setup changed: R3F is intended to remain within the existing order_entry first-domain Reference: primary=%q follow-up=%q",
			primary.Application, followUp.Application)
	}
	if primary.Procedure.ID == followUp.Procedure.ID {
		t.Fatalf("test setup changed: expected distinct Procedure contexts")
	}
}

func actorContextR3FAffectedAddresses(
	corpus []fixture,
	candidate actorContextR3FRequirementScopeCandidateProbe,
) ([]string, error) {
	var affected []string
	switch candidate.Scope {
	case actorContextR3DScopeBusinessStep:
		if candidate.Procedure == "" || candidate.Step == "" || candidate.ApplicationPattern != "" {
			return nil, fmt.Errorf("Business Step scoped R3F candidate must name Procedure + Step only")
		}
		for _, procedure := range corpus {
			if procedure.Procedure.ID != candidate.Procedure {
				continue
			}
			for _, item := range procedure.Steps {
				if item.ID == candidate.Step {
					affected = append(affected, procedure.Procedure.ID+"/"+item.ID)
				}
			}
		}
	case actorContextR3DScopeApplicationPattern:
		if candidate.ApplicationPattern == "" || candidate.Procedure != "" || candidate.Step != "" {
			return nil, fmt.Errorf("Application Pattern scoped R3F candidate must name only a Pattern")
		}
		for _, procedure := range corpus {
			for _, item := range procedure.Steps {
				for _, pattern := range item.ApplicationPatterns {
					if pattern == candidate.ApplicationPattern {
						affected = append(affected, procedure.Procedure.ID+"/"+item.ID)
						break
					}
				}
			}
		}
	default:
		return nil, fmt.Errorf("unsupported R3F scope %q", candidate.Scope)
	}
	if len(affected) == 0 {
		return nil, fmt.Errorf("R3F candidate resolved no Business Action")
	}
	return actorContextR0SortedUnique(affected), nil
}

func actorContextR3FFindStep(f fixture, stepID string) (step, error) {
	var found *step
	for i := range f.Steps {
		if f.Steps[i].ID != stepID {
			continue
		}
		if found != nil {
			return step{}, fmt.Errorf("Step %q is ambiguous in Procedure %q", stepID, f.Procedure.ID)
		}
		found = &f.Steps[i]
	}
	if found == nil {
		return step{}, fmt.Errorf("Step %q not found in Procedure %q", stepID, f.Procedure.ID)
	}
	return *found, nil
}
