package management

import (
	"testing"

	"tei/internal/environment"
	"tei/internal/plugin"
)

// TestRoleTableDoesNotInventContractKinds pins the requirement that the
// management view of Language Plugin, Operator, Target Generator, UI extension,
// External integration and test/diagnostic tooling must not introduce a second
// Capability enum. Represented roles name only kinds plugin.CapabilityKinds
// already accepts, and the unrepresented ones say so.
func TestRoleTableDoesNotInventContractKinds(t *testing.T) {
	accepted := map[string]bool{}
	for _, kind := range plugin.CapabilityKinds {
		accepted[string(kind)] = true
	}
	roles := roleStatuses()
	if len(roles) == 0 {
		t.Fatalf("the role table must not be empty")
	}

	seen := map[string]bool{}
	for _, role := range roles {
		seen[role.Target] = true
		if role.Status == RoleRepresented {
			for _, kind := range role.CapabilityKinds {
				if !accepted[kind] {
					t.Fatalf("role %s claims kind %q, which plugin.CapabilityKinds does not accept", role.Target, kind)
				}
			}
		}
	}
	for _, target := range []string{
		"language-plugin", "operator", "data-provider", "target-generator",
		"ui-studio-extension", "external-integration", "test-diagnostic",
		"deployment-runtime",
	} {
		if !seen[target] {
			t.Errorf("the role table must name the management target %q", target)
		}
	}

	// A kind accepted by the contract must appear in the table, even if only as
	// "not in the management classification", so a new kind never silently
	// disappears.
	listed := map[string]bool{}
	for _, role := range roles {
		for _, kind := range role.CapabilityKinds {
			listed[kind] = true
		}
	}
	for kind := range accepted {
		if !listed[kind] {
			t.Errorf("kind %q is accepted by the contract but missing from the role table", kind)
		}
	}
}

func TestRoleTableKeepsUnrepresentedTargetsOut(t *testing.T) {
	for _, role := range roleStatuses() {
		switch role.Target {
		case "language-plugin", "operator", "test-diagnostic":
			if role.Status != RoleNotAdopted {
				t.Fatalf("%s must be %s, got %s", role.Target, RoleNotAdopted, role.Status)
			}
			if len(role.CapabilityKinds) != 0 {
				t.Fatalf("%s must not name a kind: %+v", role.Target, role.CapabilityKinds)
			}
		case "target-generator":
			if role.Status != RoleBuildInput {
				t.Fatalf("%s must be %s, got %s", role.Target, RoleBuildInput, role.Status)
			}
		}
	}
}

func TestReportCarriesRoleTableAndAuthorityLimits(t *testing.T) {
	env := environment.Environment{ID: cid(t, "development")}
	report, err := Derive(Input{World: world(nil, env), Environment: env.ID})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if len(report.Roles) == 0 {
		t.Fatalf("the report must carry the role table")
	}
	if len(report.AuthorityLimits) == 0 {
		t.Fatalf("the report must name what it does not guarantee")
	}
	found := false
	for _, limit := range report.AuthorityLimits {
		if limit == "" {
			continue
		}
		if containsWord(limit, "no permission") || containsWord(limit, "grants no permission") {
			found = true
		}
	}
	if !found {
		// The limit list must state that diagnosis grants no permission.
		for _, limit := range report.AuthorityLimits {
			t.Logf("authority limit: %s", limit)
		}
		t.Fatalf("the report must state that diagnosis grants no permission")
	}
}

func containsWord(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle ||
		indexOf(haystack, needle) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
