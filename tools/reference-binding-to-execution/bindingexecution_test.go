package bindingexecution

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/deployment/executor"
)

// fakeExecutor records what it was asked to do. It never invents a result the
// binding did not select.
type fakeExecutor struct {
	id      string
	support deployment.ActionKind
	// wrongID makes the executor report a different ID than the registry key,
	// so ValidateResult can be exercised.
	wrongID bool
}

func (f fakeExecutor) ID() string { return f.id }

func (f fakeExecutor) Supports(c executor.Command) bool {
	return c.Kind == string(f.support)
}

func (f fakeExecutor) Execute(_ context.Context, c executor.Command) (executor.Result, error) {
	reported := f.id
	if f.wrongID {
		reported = "someone.else"
	}
	return executor.Result{
		ExecutorID: reported, ActionID: c.ActionID,
		IdempotencyKey: c.IdempotencyKey, AttemptID: "a1",
		Outcome: executor.OutcomeAccepted,
	}, nil
}

// writeWorkspace creates the smallest readable Workspace with one environment
// that binds a deployment executor.
func writeWorkspace(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"tei.workspace.yaml": "workspace: bind_exec\nformatVersion: 1\nsources:\n  - environments\n",
		"environments/development/environment.tei": "" +
			"environment development {\n" +
			"    deployment-executor database-release for database sales_core -> migration.atlas\n" +
			"}\n",
	}
	for name, content := range files {
		path := dir + "/" + name
		if err := writeFile(path, content); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return dir
}

func TestABoundExecutorIsSelectedByTheEnvironment(t *testing.T) {
	root := writeWorkspace(t)
	report, err := Run(root, "development", []executor.Command{{
		ActionID: "db/apply", IdempotencyKey: "k1",
		Kind: string(deployment.ActionDatabaseRelease), Database: "sales_core",
	}}, map[core.CanonicalID]executor.Executor{
		"migration.atlas": fakeExecutor{id: "migration.atlas", support: deployment.ActionDatabaseRelease},
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if report.Ran != 1 || report.Refused != 0 {
		t.Fatalf("report = %+v", report)
	}
	if report.Outcomes[0].ExecutorID != "migration.atlas" || !report.Outcomes[0].BindingResolved {
		t.Fatalf("outcome = %+v", report.Outcomes[0])
	}
}

func TestAnUnregisteredExecutorIsRefusedNotFaked(t *testing.T) {
	root := writeWorkspace(t)
	report, err := Run(root, "development", []executor.Command{{
		ActionID: "db/apply", IdempotencyKey: "k1",
		Kind: string(deployment.ActionDatabaseRelease), Database: "sales_core",
	}}, map[core.CanonicalID]executor.Executor{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if report.Ran != 0 || report.Refused != 1 {
		t.Fatalf("report = %+v", report)
	}
	if report.Outcomes[0].Error == "" {
		t.Fatal("a refusal must explain itself")
	}
}

func TestNoBindingIsRefused(t *testing.T) {
	root := writeWorkspace(t)
	report, err := Run(root, "development", []executor.Command{{
		ActionID: "start", IdempotencyKey: "k2",
		Kind: string(deployment.ActionApplicationStart), Application: "order_entry",
	}}, map[core.CanonicalID]executor.Executor{
		"migration.atlas": fakeExecutor{id: "migration.atlas", support: deployment.ActionDatabaseRelease},
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if report.Ran != 0 || report.Refused != 1 {
		t.Fatalf("report = %+v", report)
	}
}

func TestAResultFromAnotherExecutorIsUnverifiedAfterInvocation(t *testing.T) {
	root := writeWorkspace(t)
	report, err := Run(root, "development", []executor.Command{{
		ActionID: "db/apply", IdempotencyKey: "k1",
		Kind: string(deployment.ActionDatabaseRelease), Database: "sales_core",
	}}, map[core.CanonicalID]executor.Executor{
		"migration.atlas": fakeExecutor{
			id: "migration.atlas", support: deployment.ActionDatabaseRelease, wrongID: true,
		},
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if report.Ran != 1 || report.Refused != 0 || report.Unverified != 1 ||
		len(report.Outcomes) != 1 || !report.Outcomes[0].ExecutorInvoked ||
		report.Outcomes[0].BindingResolved || report.Outcomes[0].Error == "" {
		t.Fatalf("a post-invocation invalid acknowledgement was classified as no-execution: %+v", report)
	}
}

func TestUnknownEnvironmentIsRefused(t *testing.T) {
	root := writeWorkspace(t)
	report, err := Run(root, "production", []executor.Command{{
		ActionID: "db/apply", IdempotencyKey: "k1",
		Kind: string(deployment.ActionDatabaseRelease), Database: "sales_core",
	}}, map[core.CanonicalID]executor.Executor{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if report.Ran != 0 || len(report.Outcomes) != 0 {
		t.Fatalf("report = %+v", report)
	}
	if report.Environment != "production" {
		t.Fatalf("environment = %q", report.Environment)
	}
}

func TestReportStatesAuthorityAndSaysWhatItDoesNotClaim(t *testing.T) {
	a := Report{Schema: Schema, Authority: Authority, RuleFingerprint: fingerprint(), Notes: notes()}
	if a.Authority != Authority || a.Schema != Schema || a.RuleFingerprint == "" {
		t.Fatalf("bad header: %+v", a)
	}
	if len(a.Notes) == 0 {
		t.Fatal("the report must state what it does not establish")
	}
}

// writeFile creates a parent directory before writing, so the Workspace
// fixture does not need a separate setup step.
func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
