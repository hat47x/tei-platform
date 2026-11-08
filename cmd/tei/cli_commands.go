package main

import (
	"flag"
	"fmt"
	"io"
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
	"self-build",
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

// helpText describes one command's flags. `-h` and `-help` ask the CLI to
// explain itself, so answering with INVALID_ARGUMENT and exit 2 made the most
// ordinary exploration look like a mistake (#3185).
//
// The shared FlagSet holds every command's flags, so printing all of them
// would answer a different question than the one asked. When the command has
// an advertised flag list, only those flags are described; otherwise the text
// names the command and points at `tei commands` rather than dumping the
// others' flags.
func helpText(fs *flag.FlagSet, command string) string {
	var out strings.Builder
	fmt.Fprintf(&out, "usage: tei %s [options]\n", command)
	allowed, ok := selectedReadOnlyFlagAllowlist[command]
	if !ok {
		if command == "model catalog" {
			// model catalog parses its own FlagSet, so it is not in the shared
			// allowlist. Its two flags are the whole grammar it accepts.
			fmt.Fprintf(&out, "  -workspace string\n    \tTEI workspace directory\n")
			fmt.Fprintf(&out, "  -namespace string\n    \toptional exact semantic namespace ID\n")
			return out.String()
		}
		if command == "self-build" {
			// self-build is dispatched but deliberately absent from the
			// read-only discovery catalog, so "run `tei commands`" would
			// point at a list that does not contain it (#3414 §4.5).
			fmt.Fprintf(&out, "  -child string\n    \texact child executable to start (never resolved from PATH)\n")
			fmt.Fprintf(&out, "  -child-arg string\n    \tchild arguments as a JSON array of strings\n")
			fmt.Fprintf(&out, "  -child-arg-file string\n    \tJSON array of child arguments in a file\n")
			fmt.Fprintf(&out, "  -artifact string\n    \tpath the child writes its artifact to (required, no-clobber)\n")
			fmt.Fprintf(&out, "  -claimed-self string\n    \tdigest this binary claims about itself; compared with the running image\n")
			return out.String()
		}
		fmt.Fprintf(&out, "run `tei commands` for the advertised commands\n")
		return out.String()
	}
	wanted := make(map[string]bool, len(allowed))
	for _, name := range allowed {
		wanted[name] = true
	}
	fs.VisitAll(func(f *flag.Flag) {
		if !wanted[f.Name] {
			return
		}
		placeholder, usage := flag.UnquoteUsage(f)
		line := "  -" + f.Name
		if placeholder != "" {
			line += " " + placeholder
		}
		fmt.Fprintf(&out, "%s\n", line)
		if usage != "" {
			fmt.Fprintf(&out, "    \t%s\n", usage)
		}
		// A closed value set is part of the grammar, not prose the caller has
		// to find in a document (#3242). Print the same sets `tei commands`
		// advertises, and do not restate them by hand.
		if values, ok := valueSets(command)[f.Name]; ok {
			fmt.Fprintf(&out, "    \tvalues: %s\n", strings.Join(values, ", "))
		}
		if value, ok := valueDefaults(command)[f.Name]; ok {
			fmt.Fprintf(&out, "    \tdefault: %s\n", value)
		}
	})
	return out.String()
}

// printHelp answers a requested usage listing on standard output. It is not an
// argument failure: the caller asked to read the grammar, did not ask for a
// diagnosis, and no command ran.
func printHelp(out io.Writer, fs *flag.FlagSet, command string) {
	io.WriteString(out, helpText(fs, command))
}
