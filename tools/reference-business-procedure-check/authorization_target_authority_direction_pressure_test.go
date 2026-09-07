package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

type legacyAuthorizationRuleProbe struct {
	OrganizationCode string `json:"organization_code"`
	RoleCode         string `json:"role_code"`
	Action           string `json:"action"`
	Resource         string `json:"resource"`
	Effect           string `json:"effect"`
}

type legacyAuthorizationFixtureProbe struct {
	Schema string                         `json:"schema"`
	Rules  []legacyAuthorizationRuleProbe `json:"rules"`
}

func TestO1ProtectedTargetCanBeCorroboratedByLegacyPolicyWithoutMakingPolicyAuthoritative(t *testing.T) {
	root := repoPath()
	procedure, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := deriveD4ApplicationPatternTargetOverlayProbe(procedure, "BP-06", []string{"order_entry.submit_order"})
	if err != nil {
		t.Fatal(err)
	}
	if len(candidate.ApplicationPatternTargets) != 1 {
		t.Fatalf("test setup failed: O1 protected target=%v", candidate.ApplicationPatternTargets)
	}
	target := candidate.ApplicationPatternTargets[0]

	policy, err := readLegacyAuthorizationFixtureProbe(filepath.Join(root, "reference", "legacy-auth", "authorization.json"))
	if err != nil {
		t.Fatal(err)
	}
	if diagnostics := authorizationTargetCorroborationDiagnostics("BP-06", candidate.ApplicationPatternTargets, policy); len(diagnostics) != 0 {
		t.Fatalf("current Legacy Authorization policy should corroborate O1 target: %+v", diagnostics)
	}

	changedPolicy := policy
	changedPolicy.Rules = append([]legacyAuthorizationRuleProbe(nil), policy.Rules...)
	for i := range changedPolicy.Rules {
		if changedPolicy.Rules[i].Action == "execute" && changedPolicy.Rules[i].Resource == target {
			changedPolicy.Rules[i].Resource = "order_entry.order_record"
		}
	}
	diagnostics := authorizationTargetCorroborationDiagnostics("BP-06", candidate.ApplicationPatternTargets, changedPolicy)
	if err := checkExpectedDiagnostics(diagnostics, "BUSINESS_APPLICATION_PATTERN_TARGET_AUTHORIZATION_POLICY_MISMATCH"); err != nil {
		t.Fatalf("Authorization policy driftをcross-view diagnosticへ露出できません: %v\n%+v", err, diagnostics)
	}
	if candidate.ApplicationPatternTargets[0] != target {
		t.Fatalf("Legacy Authorization policy change rewrote Business protected target: got=%q want=%q", candidate.ApplicationPatternTargets[0], target)
	}
}

func TestLegacyAuthorizationPolicyCannotBackfillMissingO1ProtectedTarget(t *testing.T) {
	root := repoPath()
	procedure, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := deriveD4ApplicationPatternTargetOverlayProbe(procedure, "BP-06", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidate.ApplicationPatternTargets) != 0 {
		t.Fatalf("test setup failed: missing O1 target was not empty: %v", candidate.ApplicationPatternTargets)
	}

	policy, err := readLegacyAuthorizationFixtureProbe(filepath.Join(root, "reference", "legacy-auth", "authorization.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !legacyAuthorizationAllowsTarget(policy, "execute", "order_entry.submit_order") {
		t.Fatal("test setup failed: Legacy Authorization policy lacks the submit target rule")
	}
	if diagnostics := authorizationTargetCorroborationDiagnostics("BP-06", candidate.ApplicationPatternTargets, policy); len(diagnostics) != 0 {
		t.Fatalf("missing Business targetをpolicy-only mismatchとして扱いました: %+v", diagnostics)
	}
	if len(candidate.ApplicationPatternTargets) != 0 {
		t.Fatalf("Legacy Authorization policy backfilled Business protected target: %v", candidate.ApplicationPatternTargets)
	}
}

func readLegacyAuthorizationFixtureProbe(path string) (legacyAuthorizationFixtureProbe, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return legacyAuthorizationFixtureProbe{}, err
	}
	var fixture legacyAuthorizationFixtureProbe
	if err := json.Unmarshal(data, &fixture); err != nil {
		return legacyAuthorizationFixtureProbe{}, err
	}
	return fixture, nil
}

func legacyAuthorizationAllowsTarget(fixture legacyAuthorizationFixtureProbe, action, target string) bool {
	for _, rule := range fixture.Rules {
		if rule.Action == action && rule.Resource == target && rule.Effect == "allow" {
			return true
		}
	}
	return false
}

func authorizationTargetCorroborationDiagnostics(stepID string, targets []string, fixture legacyAuthorizationFixtureProbe) []diagnostic {
	result := report{}
	for _, target := range targets {
		if legacyAuthorizationAllowsTarget(fixture, "execute", target) {
			continue
		}
		result.Diagnostics = append(result.Diagnostics, diagnostic{
			Code:     "BUSINESS_APPLICATION_PATTERN_TARGET_AUTHORIZATION_POLICY_MISMATCH",
			Severity: severityArchitecturePolicy,
			Step:     stepID,
			Message:  fmt.Sprintf("Business protected target %q is not corroborated by the current Legacy Authorization execute/allow policy", target),
		})
	}
	sortDiagnostics(result.Diagnostics)
	return result.Diagnostics
}
