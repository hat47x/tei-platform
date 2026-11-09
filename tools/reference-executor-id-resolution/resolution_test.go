package executoridresolution

import (
	"testing"

	"tei/internal/deployment"
	"tei/internal/environment"
	"tei/internal/plugin"
)

// workspaceWithBinding writes the smallest readable Workspace whose environment
// binds one deployment executor for database-release.
func workspaceWithBinding(t *testing.T, executorID string) string {
	t.Helper()
	root := t.TempDir()
	write(t, root, "tei.workspace.yaml",
		"workspace: executor_resolution\nformatVersion: 1\nsources:\n  - environments\n")
	write(t, root, "environments/development/environment.tei",
		"environment development {\n"+
			"    deployment-executor database-release for database sales_core -> "+executorID+"\n"+
			"}\n")
	return root
}

func envWith(t *testing.T, executorID string) environment.Environment {
	t.Helper()
	return environment.Environment{
		ID: "development",
		DeploymentExecutors: []environment.DeploymentExecutorBinding{{
			Action:   deployment.ActionDatabaseRelease,
			Scope:    environment.ExecutorScopeDatabase,
			Target:   "sales_core",
			Executor: coreID(executorID),
		}},
	}
}

// TestABoundExecutorIsResolvedFromTheBindingNotFromCode is the whole point of
// C-3: the id comes from the Environment, and the report shows whether it
// agrees with the id a reference implementation currently hard-codes.
func TestABoundExecutorIsResolvedFromTheBindingNotFromCode(t *testing.T) {
	root := workspaceWithBinding(t, "migration.atlas")
	report, err := Resolve(Input{
		WorkspaceRoot: root,
		Environment:   envWith(t, "migration.atlas"),
		Action:        string(deployment.ActionDatabaseRelease),
		Database:      "sales_core",
		HardCodedID:   "migration.atlas",
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if report.ResolvedID != "migration.atlas" {
		t.Fatalf("resolved = %q", report.ResolvedID)
	}
	if report.HardCodedID != report.ResolvedID {
		t.Fatalf("the hard-coded id and the resolved id disagree: %q vs %q",
			report.HardCodedID, report.ResolvedID)
	}
	// The lock is empty here, so the plan cannot admit the capability.
	if report.Status != StatusLockRejected && report.Status != StatusNotInPlan {
		t.Fatalf("status = %q, want a refusal without a lock", report.Status)
	}
	if report.Status == StatusResolved {
		t.Fatal("an empty lock must not resolve to a usable executor")
	}
}

// TestADifferentHardCodedIDIsReported shows the comparison the reference exists
// for: a code-level id that differs from the binding's selection.
func TestADifferentHardCodedIDIsReported(t *testing.T) {
	root := workspaceWithBinding(t, "migration.atlas")
	report, err := Resolve(Input{
		WorkspaceRoot: root,
		Environment:   envWith(t, "migration.atlas"),
		Action:        string(deployment.ActionDatabaseRelease),
		Database:      "sales_core",
		HardCodedID:   "runtime.reference",
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if report.ResolvedID == report.HardCodedID {
		t.Fatalf("the two ids were expected to differ: %q", report.ResolvedID)
	}
	if report.ResolvedID != "migration.atlas" {
		t.Fatalf("resolution followed the hard-coded id instead of the binding: %q", report.ResolvedID)
	}
}

// TestNoBindingIsRefusedNotGuessed: an environment without a matching binding
// yields a refusal, and the resolved id stays empty.
func TestNoBindingIsRefusedNotGuessed(t *testing.T) {
	root := workspaceWithBinding(t, "migration.atlas")
	report, err := Resolve(Input{
		WorkspaceRoot: root,
		Environment:   environment.Environment{ID: "development"},
		Action:        string(deployment.ActionDatabaseRelease),
		Database:      "sales_core",
		HardCodedID:   "migration.atlas",
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if report.Status != StatusBindingMissing {
		t.Fatalf("status = %q", report.Status)
	}
	if report.ResolvedID != "" {
		t.Fatalf("a missing binding produced an id: %q", report.ResolvedID)
	}
	if report.Admitted {
		t.Fatal("a missing binding must not be reported as admitted")
	}
}

// TestAnUnsupportedActionIsNotAdmittedAsResolved: the binding names an executor
// for database-release, and the requested action is application-start.
func TestAnUnsupportedActionIsNotAdmittedAsResolved(t *testing.T) {
	root := workspaceWithBinding(t, "migration.atlas")
	report, err := Resolve(Input{
		WorkspaceRoot: root,
		Environment:   envWith(t, "migration.atlas"),
		Action:        string(deployment.ActionApplicationStart),
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if report.Status == StatusResolved {
		t.Fatalf("an unbound action resolved: %+v", report)
	}
}

func TestReportStatesAuthorityAndItsLimits(t *testing.T) {
	root := workspaceWithBinding(t, "migration.atlas")
	report, err := Resolve(Input{
		WorkspaceRoot: root,
		Environment:   envWith(t, "migration.atlas"),
		Action:        string(deployment.ActionDatabaseRelease),
		Database:      "sales_core",
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if report.Authority != Authority || report.Schema != Schema || report.RuleFingerprint == "" {
		t.Fatalf("bad header: %+v", report)
	}
	if len(report.Notes) == 0 {
		t.Fatal("the report must state what it does not establish")
	}
	if report.Status == StatusResolved {
		t.Fatal("an empty lock must not resolve")
	}
}

// keep the plugin import meaningful while the fixture has no lock content.
var _ = plugin.Lock{}
