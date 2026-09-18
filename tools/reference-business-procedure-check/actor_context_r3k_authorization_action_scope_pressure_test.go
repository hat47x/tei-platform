package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

const (
	actorContextR3KLocusBusinessStep       = "business-step"
	actorContextR3KLocusApplicationPattern = "application-pattern"
	actorContextR3KLocusCrossView          = "cross-view"
)

type actorContextR3KActionScopeCandidateProbe struct {
	Locus               string
	Action              string
	SourceRef           string
	Procedure           string
	Step                string
	ApplicationPattern  string
	BusinessAddresses   []string
}

func TestActorContextR3KCurrentRecentOrdersCaseIsScopePressureNotActionAuthority(t *testing.T) {
	root := repoPath()
	primary, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	followUp, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-follow-up-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}

	for _, item := range []struct {
		f fixture
		step string
	}{
		{primary, "BP-00"},
		{followUp, "FU-00"},
	} {
		requirement, err := deriveActorContextR3BAuthorizationRequirementProbe(item.f, item.step)
		if err != nil {
			t.Fatal(err)
		}
		if requirement.Status != actorContextR3BAuthorizationUnresolved {
			t.Fatalf("recent_orders second action case unexpectedly became a qualifying required Authorization interaction: %s/%s %+v",
				item.f.Procedure.ID, item.step, requirement)
		}
	}

	patternBytes, err := os.ReadFile(filepath.Join(
		root,
		"reference", "workspace", "applications", "order-entry", "patterns", "recent-orders.tei",
	))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(patternBytes), "pattern order_entry.recent_orders browse") {
		t.Fatal("test setup changed: recent_orders is no longer a browse Pattern")
	}

	policy, err := readLegacyAuthorizationFixtureProbe(filepath.Join(root, "reference", "legacy-auth", "authorization.json"))
	if err != nil {
		t.Fatal(err)
	}
	policyBrowse := false
	for _, rule := range policy.Rules {
		if rule.Action == "browse" &&
			rule.Resource == "order_entry.recent_orders" &&
			rule.Effect == "allow" {
			policyBrowse = true
			break
		}
	}
	if !policyBrowse {
		t.Fatal("test setup changed: legacy browse/recent_orders allow rule missing")
	}

	// Both hints corroborate a likely value, but neither authors AuthorizationAction.
	actionAuthority := actorContextR3JDeriveActionAuthorityGate(
		actorContextR3BAuthorizationUnresolved,
		"order_entry.recent_orders",
		"test-only:resource-candidate",
		false,
		true,
		true,
		nil,
	)
	if actionAuthority.Action != "" {
		t.Fatalf("Pattern kind + Legacy policy generated Authorization action authority: %+v", actionAuthority)
	}
}

func TestActorContextR3KActionLociHaveDifferentReach(t *testing.T) {
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

	business, err := actorContextR3KAffectedAddresses(corpus, actorContextR3KActionScopeCandidateProbe{
		Locus:      actorContextR3KLocusBusinessStep,
		Action:     "browse",
		SourceRef:  "test-only:business-step-action",
		Procedure:  "order-entry.order-handling",
		Step:       "BP-00",
	})
	if err != nil {
		t.Fatal(err)
	}
	pattern, err := actorContextR3KAffectedAddresses(corpus, actorContextR3KActionScopeCandidateProbe{
		Locus:              actorContextR3KLocusApplicationPattern,
		Action:             "browse",
		SourceRef:          "test-only:pattern-action",
		ApplicationPattern: "order_entry.recent_orders",
	})
	if err != nil {
		t.Fatal(err)
	}
	crossView, err := actorContextR3KAffectedAddresses(corpus, actorContextR3KActionScopeCandidateProbe{
		Locus:              actorContextR3KLocusCrossView,
		Action:             "browse",
		SourceRef:          "test-only:cross-view-action",
		ApplicationPattern: "order_entry.recent_orders",
		BusinessAddresses:  []string{"order-entry.follow-up/FU-00"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(business, []string{"order-entry.order-handling/BP-00"}) {
		t.Fatalf("Business Step action leaked across Pattern reuse: %v", business)
	}
	if !reflect.DeepEqual(pattern, []string{
		"order-entry.follow-up/FU-00",
		"order-entry.order-handling/BP-00",
	}) {
		t.Fatalf("Pattern action did not reach every Pattern use: %v", pattern)
	}
	if !reflect.DeepEqual(crossView, []string{"order-entry.follow-up/FU-00"}) {
		t.Fatalf("cross-view action did not stay on explicitly selected edge: %v", crossView)
	}
	if reflect.DeepEqual(business, pattern) || reflect.DeepEqual(pattern, crossView) {
		t.Fatalf("distinct action-authoring loci collapsed: business=%v pattern=%v crossView=%v", business, pattern, crossView)
	}
}

func TestActorContextR3KActionScopeDoesNotAuthorRequirementScope(t *testing.T) {
	candidate := actorContextR3KActionScopeCandidateProbe{
		Locus:              actorContextR3KLocusApplicationPattern,
		Action:             "browse",
		SourceRef:          "test-only:pattern-action",
		ApplicationPattern: "order_entry.recent_orders",
	}

	typ := reflect.TypeOf(candidate)
	for _, forbidden := range []string{
		"RequirementStatus",
		"AuthorizationRequired",
		"Winner",
		"Preferred",
		"Confidence",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R3K action-scope candidate introduced forbidden requirement/selection shortcut field %q", forbidden)
		}
	}
}

func actorContextR3KAffectedAddresses(
	corpus []fixture,
	candidate actorContextR3KActionScopeCandidateProbe,
) ([]string, error) {
	if strings.TrimSpace(candidate.Action) == "" || strings.TrimSpace(candidate.SourceRef) == "" {
		return nil, fmt.Errorf("R3K action candidate requires explicit action and provenance")
	}

	var affected []string
	switch candidate.Locus {
	case actorContextR3KLocusBusinessStep:
		if candidate.Procedure == "" || candidate.Step == "" ||
			candidate.ApplicationPattern != "" || len(candidate.BusinessAddresses) != 0 {
			return nil, fmt.Errorf("Business Step locus must name Procedure + Step only")
		}
		for _, f := range corpus {
			if f.Procedure.ID != candidate.Procedure {
				continue
			}
			for _, step := range f.Steps {
				if step.ID == candidate.Step {
					affected = append(affected, f.Procedure.ID+"/"+step.ID)
				}
			}
		}
	case actorContextR3KLocusApplicationPattern:
		if candidate.ApplicationPattern == "" ||
			candidate.Procedure != "" || candidate.Step != "" || len(candidate.BusinessAddresses) != 0 {
			return nil, fmt.Errorf("Application Pattern locus must name only a Pattern")
		}
		for _, f := range corpus {
			for _, step := range f.Steps {
				for _, pattern := range step.ApplicationPatterns {
					if pattern == candidate.ApplicationPattern {
						affected = append(affected, f.Procedure.ID+"/"+step.ID)
						break
					}
				}
			}
		}
	case actorContextR3KLocusCrossView:
		if candidate.ApplicationPattern == "" ||
			candidate.Procedure != "" || candidate.Step != "" || len(candidate.BusinessAddresses) == 0 {
			return nil, fmt.Errorf("cross-view locus requires Pattern + explicit Business addresses")
		}
		allowed := map[string]bool{}
		for _, f := range corpus {
			for _, step := range f.Steps {
				for _, pattern := range step.ApplicationPatterns {
					if pattern == candidate.ApplicationPattern {
						allowed[f.Procedure.ID+"/"+step.ID] = true
					}
				}
			}
		}
		for _, address := range candidate.BusinessAddresses {
			if !allowed[address] {
				return nil, fmt.Errorf("cross-view address %q does not use Pattern %q", address, candidate.ApplicationPattern)
			}
			affected = append(affected, address)
		}
	default:
		return nil, fmt.Errorf("unsupported R3K action locus %q", candidate.Locus)
	}

	if len(affected) == 0 {
		return nil, fmt.Errorf("R3K action candidate resolved no Business address")
	}
	sort.Strings(affected)
	return actorContextR0SortedUnique(affected), nil
}
