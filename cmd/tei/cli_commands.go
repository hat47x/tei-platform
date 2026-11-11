package main

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"tei/internal/documentdraft"
	"tei/internal/operations"
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
// writeFlagHelp renders one FlagSet the way `-h` shows it. `wanted` filters the
// names when the caller has an allowlist; nil means every flag is part of the
// grammar.
func writeFlagHelp(out *strings.Builder, fs *flag.FlagSet, command string, wanted map[string]bool) {
	fs.VisitAll(func(f *flag.Flag) {
		if wanted != nil && !wanted[f.Name] {
			return
		}
		placeholder, usage := flag.UnquoteUsage(f)
		line := "  -" + f.Name
		if placeholder != "" {
			line += " " + placeholder
		}
		fmt.Fprintf(out, "%s\n", line)
		if usage != "" {
			fmt.Fprintf(out, "    \t%s\n", usage)
		}
		if values, ok := valueSets(command)[f.Name]; ok {
			fmt.Fprintf(out, "    \tvalues: %s\n", strings.Join(values, ", "))
		}
		if value, ok := valueDefaults(command)[f.Name]; ok {
			fmt.Fprintf(out, "    \tdefault: %s\n", value)
		}
	})
}

// selfBuildHelpFlags holds both grammars the command accepts. It is built from
// the §4 FlagSet so the two cannot drift, and adds the primitive-only flags that
// the §4 form does not register.
func selfBuildHelpFlags() *flag.FlagSet {
	var scratch selfBuildOuterArgs
	outer := newSelfBuildOuterFlagSet(&scratch)
	fs := flag.NewFlagSet("self-build", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	outer.VisitAll(func(f *flag.Flag) {
		fs.Var(f.Value, f.Name, f.Usage)
	})
	fs.String("child-arg-file", "", "JSON array of child arguments in a file")
	fs.String("artifact", "", "path the child writes its artifact to (no-clobber)")
	fs.String("claimed-self", "", "digest this binary claims about itself; compared with the running image")
	return fs
}

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
		if command == "model locate" {
			// model locate parses its own FlagSet too; these four flags are its
			// whole grammar.
			fmt.Fprintf(&out, "  -workspace string\n    \tTEI workspace directory\n")
			fmt.Fprintf(&out, "  -subject-kind string\n    \texact subject kind from model catalog entries[].subject.kind\n")
			fmt.Fprintf(&out, "  -subject-id string\n    \texact subject ID\n")
			fmt.Fprintf(&out, "  -limit int\n    \tmaximum entries per list (children, siblings); default %d, at most %d\n", operations.LocationDefaultLimit, operations.LocationMaxLimit)
			return out.String()
		}
		if command == "model overview" {
			// model overview parses its own FlagSet; two flags are its grammar.
			fmt.Fprintf(&out, "  -workspace string\n    \tTEI workspace directory\n")
			fmt.Fprintf(&out, "  -limit int\n    \tmaximum sections listed; totals still cover every section; default %d, at most %d\n", operations.OverviewDefaultLimit, operations.OverviewMaxLimit)
			// A section count is not a count of everything the workspace declares:
			// the sections mirror the model catalog (#4128). Say so here, where a
			// designer looks before reading the number.
			fmt.Fprintf(&out, "\nsections mirror the model catalog roots. The response's sectionScope names\nthe kinds the catalog covers and the declared kinds it does not, so a section\ncount is never a count of every declaration.\n")
			return out.String()
		}
		if command == "model document-draft" {
			// model document-draft parses its own FlagSet; these flags are its
			// whole grammar.
			fmt.Fprintf(&out, "  -workspace string\n    \tTEI workspace directory\n")
			fmt.Fprintf(&out, "  -document-kind string\n    \tdocument kind: %s\n", strings.Join(documentdraft.SupportedDocumentKinds(), "|"))
			fmt.Fprintf(&out, "  -subject-kind string\n    \texact subject kind: %s\n", strings.Join(documentdraft.SupportedElementKinds(), "|"))
			fmt.Fprintf(&out, "  -subject-id string\n    \texact subject ID\n")
			fmt.Fprintf(&out, "  -relation-id string\n    \toptional exact semantic-relation ID declared by the subject\n")
			fmt.Fprintf(&out, "  -application string\n    \texact application ID; required by %s\n", documentdraft.DocumentKindCRUDTable)
			fmt.Fprintf(&out, "  -frozen\n    \tpin the draft to the model revision it is read at (a fixed delivery)\n")
			fmt.Fprintf(&out, "  -title string\n    \toptional document title\n")
			fmt.Fprintf(&out, "\n%s takes -subject-kind and -subject-id. %s takes -application instead and\nrefuses both element selectors. The result is a derived read-only projection:\nsaving the document it produces is not TEI model ingestion and not Accepted\nMeaning.\n",
				documentdraft.DocumentKindElementExplanation, documentdraft.DocumentKindCRUDTable)
			return out.String()
		}
		if command == "model document-reconcile" {
			// model document-reconcile parses its own FlagSet; two flags are its
			// grammar.
			fmt.Fprintf(&out, "  -workspace string\n    \tTEI workspace directory\n")
			fmt.Fprintf(&out, "  -draft string\n    \tpath to the stored draft JSON\n")
			fmt.Fprintf(&out, "\nthe comparison is read-only. It names which occurrences need\nreconfirmation and never updates the draft, the document or a Source.\n")
			return out.String()
		}
		if command == "dbcheck" {
			// dbcheck is dispatched but deliberately absent from the read-only
			// discovery catalog, so it cannot carry an allowlist entry; the one
			// flag it reads is stated here instead of dumping the shared set
			// (#3943).
			fmt.Fprintf(&out, "  -dsn string\n    \tdbcheck: PostgreSQL DSN (defaults to TEI_TEST_POSTGRES_DSN)\n")
			return out.String()
		}
		if command == "self-build" {
			// self-build is dispatched but deliberately absent from the
			// read-only discovery catalog, so "run `tei commands`" would point
			// at a list that does not contain it (#3414 §4.5).
			//
			// Two forms share this name: the §4 workspace form and the primitive
			// -child/-artifact form (§4.6). The text is derived from one FlagSet
			// holding both grammars, so a flag added to either form appears here
			// without a second edit — restating them by hand is what left the
			// catalog flags missing from -h.
			writeFlagHelp(&out, selfBuildHelpFlags(), command, nil)
			return out.String()
		}
		fmt.Fprintf(&out, "run `tei commands` for the advertised commands\n")
		return out.String()
	}
	wanted := make(map[string]bool, len(allowed))
	for _, name := range allowed {
		wanted[name] = true
	}
	writeFlagHelp(&out, fs, command, wanted)
	return out.String()
}

// requestedHelp reports whether the arguments ask for the usage listing.
// Only the flag form counts, so a value that happens to be "-h" after a
// positional argument is not taken for a help request.
func requestedHelp(args []string) bool {
	for _, arg := range args {
		if arg == "--" {
			return false
		}
		if arg == "-h" || arg == "-help" || arg == "--help" {
			return true
		}
	}
	return false
}

// printHelp answers a requested usage listing on standard output. It is not an
// argument failure: the caller asked to read the grammar, did not ask for a
// diagnosis, and no command ran.
func printHelp(out io.Writer, fs *flag.FlagSet, command string) {
	io.WriteString(out, helpText(fs, command))
}
