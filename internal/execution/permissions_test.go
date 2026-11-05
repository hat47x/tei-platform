package execution_test

import (
	"context"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/execution"
	"tei/internal/permission"
	"tei/internal/semantic"
)

func permissionRequirements(capabilityID core.CanonicalID) []permission.Spec {
	if capabilityID != "sales.close" {
		return nil
	}
	return []permission.Spec{
		{Kind: permission.KindFilesystem, Access: permission.AccessWrite, Scope: "/var/reports"},
		{Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: "db:5432"},
	}
}

func TestPermissionPreconditionsAllowCoveredRequirements(t *testing.T) {
	preconditions := execution.PermissionPreconditions(permissionRequirements, []permission.Spec{
		{Kind: permission.KindFilesystem, Access: permission.AccessReadWrite, Scope: "/var"},
		{Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: "db:5432"},
	})
	if reason := preconditions(core.MustCanonicalID("sales.close")); reason != "" {
		t.Fatalf("covered requirements must not block: %q", reason)
	}
}

func TestPermissionPreconditionsNameEveryMissingRequirement(t *testing.T) {
	preconditions := execution.PermissionPreconditions(permissionRequirements, []permission.Spec{
		{Kind: permission.KindFilesystem, Access: permission.AccessRead, Scope: "/var/reports"},
	})
	reason := preconditions(core.MustCanonicalID("sales.close"))
	if reason == "" {
		t.Fatal("an uncovered requirement must block")
	}
	for _, want := range []string{"filesystem:write /var/reports", "network:connect db:5432"} {
		if !strings.Contains(reason, want) {
			t.Fatalf("the reason must name %q: %q", want, reason)
		}
	}
}

func TestPermissionPreconditionsIgnoreCapabilitiesWithoutRequirements(t *testing.T) {
	preconditions := execution.PermissionPreconditions(permissionRequirements, nil)
	if reason := preconditions(core.MustCanonicalID("sales.report")); reason != "" {
		t.Fatalf("a capability without requirements is not blocked: %q", reason)
	}
}

func TestPermissionPreconditionsWithoutAMappingDoNothing(t *testing.T) {
	preconditions := execution.PermissionPreconditions(nil, nil)
	if reason := preconditions(core.MustCanonicalID("sales.close")); reason != "" {
		t.Fatalf("no mapping means nothing known, so nothing is refused here: %q", reason)
	}
}

func TestPermissionPreconditionsBlockAnUnattendedRun(t *testing.T) {
	journal := &runJournal{}
	called := false
	step := func(_ context.Context, operation core.CanonicalID) execution.StepResult {
		called = true
		return execution.StepResult{Operation: string(operation), Status: "resolved"}
	}
	plan := semantic.RunPlan{ID: "sales.monthly_close", Operations: []core.CanonicalID{"sales.close"}, Autonomy: "unattended"}
	preconditions := execution.CombinePreconditions(execution.PermissionPreconditions(permissionRequirements, nil))
	_, results, err := execution.RunPlan(context.Background(), plan, []string{"2026-10"}, journal, step, preconditions)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if called || len(results) != 1 || results[0].Status != "blocked" {
		t.Fatalf("an uncovered permission must block the run: called=%v results=%+v", called, results)
	}
}
