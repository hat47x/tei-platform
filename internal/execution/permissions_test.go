package execution_test

import (
	"context"
	"strings"
	"testing"

	"tei/internal/access"

	"tei/internal/core"
	"tei/internal/execution"
	"tei/internal/permission"
	"tei/internal/plugin"
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
func grantsFor() []permission.Grant {
	return []permission.Grant{
		{CapabilityKind: "plugin", CapabilityID: "sales.close", Permission: permission.Spec{Kind: permission.KindFilesystem, Access: permission.AccessWrite, Scope: "/var/reports"}},
		{CapabilityKind: "plugin", CapabilityID: "sales.close", Permission: permission.Spec{Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: "db:5432"}},
		{CapabilityKind: "plugin", CapabilityID: "sales.report", Permission: permission.Spec{Kind: permission.KindFilesystem, Access: permission.AccessRead, Scope: "/var/reports"}},
		{CapabilityKind: "service", CapabilityID: "sales.close", Permission: permission.Spec{Kind: permission.KindSecret, Access: permission.AccessRead, Scope: "vault:db"}},
	}
}

func TestGuardFromGrantsSelectsByKindAndIdentity(t *testing.T) {
	guard := execution.GuardFromGrants(grantsFor(), "plugin", "sales.close", nil)
	if len(guard.Granted) != 2 {
		t.Fatalf("only this capability's grants belong to it: %+v", guard.Granted)
	}
	decision, err := guard.Check(core.MustCanonicalID("sales.close"), []permission.Spec{{Kind: permission.KindFilesystem, Access: permission.AccessWrite, Scope: "/var/reports"}})
	if err != nil || !decision.Allowed {
		t.Fatalf("the granted requirement must pass: %+v err=%v", decision, err)
	}
	decision, err = guard.Check(core.MustCanonicalID("sales.close"), []permission.Spec{{Kind: permission.KindSecret, Access: permission.AccessRead, Scope: "vault:db"}})
	if err != nil || decision.Allowed {
		t.Fatalf("another capability kind's grant must not apply: %+v err=%v", decision, err)
	}
}

type accessJournal struct {
	records []access.Record
}

func (j *accessJournal) Append(record access.Record) error {
	j.records = append(j.records, record)
	return nil
}

func TestGuardFromGrantsRecords(t *testing.T) {
	journal := &accessJournal{}
	guard := execution.GuardFromGrants(grantsFor(), "plugin", "sales.close", journal)
	if _, err := guard.Check(core.MustCanonicalID("sales.close"), nil); err != nil {
		t.Fatalf("check: %v", err)
	}
	if len(journal.records) != 1 {
		t.Fatalf("the decision must be recorded: %+v", journal.records)
	}
}

var _ access.Journal = access.File{}

func TestRequirementsFromCatalogKeepsEachCapabilitysDeclarations(t *testing.T) {
	catalog := plugin.Catalog{Plugins: []plugin.Definition{{
		ID:      "tei.demo",
		Version: "1.0.0",
		Provides: []plugin.Capability{
			{ID: "sales.close", Permissions: []permission.Spec{{Kind: permission.KindFilesystem, Access: permission.AccessWrite, Scope: "/var/reports"}}},
			{ID: "sales.report"},
		},
	}}}
	requirements := execution.RequirementsFromCatalog(catalog)
	if got := requirements(core.MustCanonicalID("sales.close")); len(got) != 1 || got[0].Scope != "/var/reports" {
		t.Fatalf("the declaration must be returned: %+v", got)
	}
	if got := requirements(core.MustCanonicalID("sales.report")); len(got) != 0 {
		t.Fatalf("a capability without declarations returns none: %+v", got)
	}
	if got := requirements(core.MustCanonicalID("sales.unknown")); len(got) != 0 {
		t.Fatalf("an unknown capability returns none: %+v", got)
	}
}

func TestRequirementsFromCatalogFeedPermissionPreconditions(t *testing.T) {
	catalog := plugin.Catalog{Plugins: []plugin.Definition{{
		ID:      "tei.demo",
		Version: "1.0.0",
		Provides: []plugin.Capability{{
			ID:          "sales.close",
			Permissions: []permission.Spec{{Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: "db:5432"}},
		}},
	}}}
	preconditions := execution.PermissionPreconditions(execution.RequirementsFromCatalog(catalog), nil)
	if reason := preconditions(core.MustCanonicalID("sales.close")); reason == "" {
		t.Fatal("an ungranted declaration must block the capability")
	}
}
