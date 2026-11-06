package main

import (
	"fmt"
	"strings"
)

// acceptedCommands is the single list of command names the CLI dispatches.
// The usage line and the unknown-command rejection both read it, so a name
// that cannot be dispatched is not advertised, and a name that is advertised
// is not reported as an unknown command.
var acceptedCommands = []string{
	"application-release-compatibility",
	"artifacts",
	"commands",
	"consumer-impact",
	"database-compatibility",
	"database-reconcile",
	"dbcheck",
	"deployment-abort",
	"deployment-pause",
	"deployment-provider-plan",
	"deployment-ready-commands",
	"deployment-reconcile",
	"deployment-record-attempt",
	"deployment-recovery",
	"deployment-resume",
	"deployment-retry",
	"deployment-simulate",
	"deployment-start",
	"deployment-start-recovery",
	"desired-schema",
	"grants",
	"index",
	"inspect",
	"issue-grant",
	"knowledge-context",
	"knowledge-data-design-candidate-review",
	"knowledge-qualify",
	"knowledge-review-disclose",
	"knowledge-review-handoff",
	"knowledge-review-plan",
	"migration-fingerprint",
	"model",
	"normalize-postgres-plan",
	"observed-index",
	"operations",
	"periods",
	"plan-migration",
	"release-graph",
	"scenario",
	"schema-diff",
	"solution-release-graph",
	"source-index",
	"trace",
	"validate",
	"version",
	"view",
	"workbench-model",
	"workspace-files",
	"world",
}

// helperDispatchedCommands are handled by helper functions whose command
// name is checked inside the helper or by a result-envelope predicate, so a
// literal `command ==` search in main.go does not see them.
var helperDispatchedCommands = []string{
	"requirements",
	"plugin-lock",
	"plugin-resolve",
	"plugin-runtime-plan",
}

func init() {
	acceptedCommands = append(acceptedCommands, helperDispatchedCommands...)
}

// usageText describes the command grammar without naming options, because
// every command has its own flags.
func usageText() string {
	return "usage: tei <" + strings.Join(acceptedCommands, "|") + "> [options]"
}

// isAcceptedCommand reports whether the first argv element is a command the
// CLI dispatches. It does not validate a command's flags or its inputs.
func isAcceptedCommand(name string) bool {
	for _, accepted := range acceptedCommands {
		if name == accepted {
			return true
		}
	}
	return false
}

// rejectUnknownCommand reports a name the CLI does not dispatch, in the same
// form whether or not a -workspace or -world flag follows it.
func rejectUnknownCommand(name string, errOut interface{ Write([]byte) (int, error) }) {
	fmt.Fprintf(errOut, "unknown command %q\n%s\n", name, usageText())
}
