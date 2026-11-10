package bindingguardedexecutor

import (
	"testing"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/deployment/executor"
	"tei/internal/environment"
)

func envBinding(action deployment.ActionKind, executorID string) environment.Environment {
	return environment.Environment{
		ID: "development",
		DeploymentExecutors: []environment.DeploymentExecutorBinding{{
			Action: action, Executor: core.CanonicalID(executorID),
		}},
	}
}

func cmd(action deployment.ActionKind) executor.Command {
	return executor.Command{
		ActionID: "reference/apply", Kind: string(action),
		IdempotencyKey: "k",
	}
}

// The adapter's hard-coded id matches the binding: the guard serves it.
func TestAMatchingIDIsServed(t *testing.T) {
	env := envBinding(deployment.ActionDatabaseRelease, "migration.atlas")
	outcome := Resolve(env, Request{
		Command:     cmd(deployment.ActionDatabaseRelease),
		RequestedID: "migration.atlas",
	}, map[core.CanonicalID]bool{"migration.atlas": true})

	if outcome.Status != StatusServed || !outcome.Served {
		t.Fatalf("outcome = %+v", outcome)
	}
	if outcome.BoundID != "migration.atlas" {
		t.Fatalf("bound id = %q", outcome.BoundID)
	}
}

// This is the blindspot: the adapter wants "migration.atlas" but the binding
// selects another executor. The old path would have run the hard-coded one.
func TestAnIDTheBindingDidNotSelectIsRefused(t *testing.T) {
	env := envBinding(deployment.ActionDatabaseRelease, "migration.other")
	outcome := Resolve(env, Request{
		Command:     cmd(deployment.ActionDatabaseRelease),
		RequestedID: "migration.atlas",
	}, map[core.CanonicalID]bool{"migration.atlas": true, "migration.other": true})

	if outcome.Served {
		t.Fatal("an executor the binding did not select was served")
	}
	if outcome.Status != StatusNotMatched {
		t.Fatalf("status = %q", outcome.Status)
	}
	if outcome.BoundID != "migration.other" {
		t.Fatalf("bound id = %q", outcome.BoundID)
	}
}

// With no binding for the action, nothing is served and no id is invented.
func TestNoBindingServesNothing(t *testing.T) {
	env := environment.Environment{ID: "development"}
	outcome := Resolve(env, Request{
		Command:     cmd(deployment.ActionDatabaseRelease),
		RequestedID: "migration.atlas",
	}, map[core.CanonicalID]bool{"migration.atlas": true})

	if outcome.Served || outcome.Status != StatusNotBound {
		t.Fatalf("outcome = %+v", outcome)
	}
	if outcome.BoundID != "" {
		t.Fatalf("a missing binding produced an id: %q", outcome.BoundID)
	}
}

// A binding that names an executor the Host does not hold is refused; the guard
// does not pretend the executor exists.
func TestAnUnregisteredBoundExecutorIsRefused(t *testing.T) {
	env := envBinding(deployment.ActionDatabaseRelease, "migration.atlas")
	outcome := Resolve(env, Request{
		Command:     cmd(deployment.ActionDatabaseRelease),
		RequestedID: "migration.atlas",
	}, map[core.CanonicalID]bool{})

	if outcome.Served || outcome.Status != StatusUnknown {
		t.Fatalf("outcome = %+v", outcome)
	}
}

// A run mixes served and refused requests and keeps the counts separate.
func TestARunSeparatesServedFromRefused(t *testing.T) {
	env := envBinding(deployment.ActionDatabaseRelease, "migration.atlas")
	report := Run(Input{
		Environment: env,
		Registered:  []core.CanonicalID{"migration.atlas"},
		Requests: []Request{
			{Command: cmd(deployment.ActionDatabaseRelease), RequestedID: "migration.atlas"},
			// The adapter hard-codes another id: refused.
			{Command: cmd(deployment.ActionDatabaseRelease), RequestedID: "runtime.native"},
		},
	})
	if report.Served != 1 || report.Refused != 1 {
		t.Fatalf("served=%d refused=%d, want 1 and 1", report.Served, report.Refused)
	}
}

func TestReportStatesAuthorityAndItsLimits(t *testing.T) {
	report := Run(Input{Environment: environment.Environment{ID: "development"}})
	if report.Authority != Authority || report.Schema != Schema || report.RuleFingerprint == "" {
		t.Fatalf("bad header: %+v", report)
	}
	found := false
	for _, note := range report.Notes {
		if note == "the guard does not prove the executor works or that an effect occurred" {
			found = true
		}
	}
	if !found {
		t.Fatal("the notes do not bound what served means")
	}
}
