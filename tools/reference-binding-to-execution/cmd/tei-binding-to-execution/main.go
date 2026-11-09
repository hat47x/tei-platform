// tei-binding-to-execution is a read-only reference CLI. It resolves one
// environment's deployment-executor binding and dispatches the given commands
// through it. It never modifies the Workspace.
//
// The executor registry is deliberately tiny: the reference registers one
// recorder executor so the path from binding to execution is observable
// without touching an external system. A Host would replace it with its own.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"tei/internal/core"
	"tei/internal/deployment/executor"
	bindingexecution "tei/tools/reference-binding-to-execution"
)

// recorder accepts every command and reports it back. It is not an executor for
// production use: it performs no external effect and proves nothing about state.
type recorder struct{ id string }

func (r recorder) ID() string { return r.id }

func (r recorder) Supports(executor.Command) bool { return true }

func (r recorder) Execute(_ context.Context, c executor.Command) (executor.Result, error) {
	return executor.Result{
		ExecutorID: r.id, ActionID: c.ActionID,
		IdempotencyKey: c.IdempotencyKey, AttemptID: "reference-recorder",
		Outcome: executor.OutcomeAccepted,
	}, nil
}

func main() {
	root := flag.String("workspace", "", "Source Workspace root to read")
	environmentID := flag.String("environment", "", "Environment to resolve the binding from")
	actionID := flag.String("action-id", "", "Command action ID")
	kind := flag.String("kind", "", "Deployment action kind")
	database := flag.String("database", "", "Target database, when the action needs one")
	application := flag.String("application", "", "Target application, when the action needs one")
	flag.Parse()
	if *root == "" || *environmentID == "" || *actionID == "" || *kind == "" {
		fail(fmt.Errorf("-workspace, -environment, -action-id and -kind are required"))
	}

	cmd := executor.Command{
		ActionID: *actionID, IdempotencyKey: "reference-" + *actionID,
		Kind: *kind,
	}
	if *database != "" {
		cmd.Database = core.CanonicalID(*database)
	}
	if *application != "" {
		cmd.Application = core.CanonicalID(*application)
	}

	report, err := bindingexecution.Run(*root, core.CanonicalID(*environmentID), []executor.Command{cmd},
		map[core.CanonicalID]executor.Executor{"migration.atlas": recorder{id: "migration.atlas"}})
	if err != nil {
		fail(err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		fail(err)
	}
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(2) }
