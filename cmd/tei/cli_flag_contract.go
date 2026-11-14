package main

import (
	"flag"
	"fmt"
	"io"
	"strings"
)

// selectedReadOnlyFlagError returns a machine-readable argument error for the
// C1 commands. It does not interpret the error as a semantic finding or print
// a second stderr-only diagnostic.
func selectedReadOnlyFlagError(out io.Writer, command string, err error) (int, error) {
	failure := &cliInspectionFailure{Code: "INVALID_ARGUMENT", Message: err.Error()}
	switch command {
	case "inspect":
		return writeInspectionReport(out, cliInspectionReport{
			SchemaVersion: "tei.cli.inspect/v0",
			Error:         failure,
		}, 2)
	case "trace":
		return writeTraceReport(out, cliTraceReport{
			SchemaVersion: "tei.cli.trace/v0",
			Error:         failure,
		}, 2)
	default:
		return 2, fmt.Errorf("unsupported JSON argument-error command %q", command)
	}
}

// selectedReadOnlyFlagAllowlist lists, per command, every flag the command
// reads. commandCatalog advertises the same flags, and a table-driven test
// keeps the two in agreement. `model catalog` is not listed here because it
// parses its own arguments instead of the shared FlagSet.
var selectedReadOnlyFlagAllowlist = map[string][]string{
	"external-candidates": {"evidence-path", "tool-version", "observation-id", "relation-id", "expected-evidence-id", "expected-received-digest", "expected-source-revision"},
	"inspect": {"workspace", "subject-kind", "subject-id"},
	"view":    {"workspace", "application", "pattern", "package", "format", "command", "input", "execute", "reference-store", "period-state", "allow-capability", "operation-log", "lifecycle-state", "lifecycle-command", "lifecycle-owner", "lifecycle-generation"},
	"trace": {
		"workspace", "scope", "subject-kind", "subject-id",
		"direction", "mode", "offset", "limit", "expected-fingerprint",
	},
	"workspace-files": {"workspace"},
	"source-index":    {"workspace", "subject-kind", "subject-id"},
	"knowledge-context": {
		"workspace", "subject-kind", "subject-id", "disclose",
		"expected-data-design-fingerprint", "expected-disclosure-fingerprint",
	},
	"knowledge-qualify": {"workspace", "subject-kind", "subject-id", "qualification-requirements"},
	"knowledge-review-plan": {"workspace", "subject-kind", "subject-id",
		"expected-review-plan-fingerprint"},
	"knowledge-review-handoff": {"workspace", "subject-kind", "subject-id",
		"qualification-requirements", "expected-review-handoff-fingerprint"},
	"knowledge-review-disclose": {"workspace", "subject-kind", "subject-id",
		"review-handoff", "expected-review-handoff-fingerprint", "expected-disclosure-fingerprint"},
	"knowledge-data-design-candidate-review": {"workspace", "subject-kind", "subject-id",
		"data-design-candidate-kind", "data-design-candidate-fingerprint",
		"expected-data-design-fingerprint",
		"expected-data-design-candidate-evidence-requirement-fingerprint",
		"expected-data-design-candidate-review-fingerprint"},
	"version":             {"json"},
	"validate":            {"workspace", "world", "json"},
	"world":               {"workspace", "world"},
	"requirements":        {"workspace", "world"},
	"plugin-lock":         {"workspace", "world", "environment"},
	"plugin-resolve":      {"workspace", "world", "environment", "plugin-lock", "diagnostic-candidates"},
	"plugin-runtime-plan": {"workspace", "world", "environment", "plugin-lock"},
	"plugin-inventory":    {"workspace", "world", "environment", "plugin-lock", "plugin", "diagnose"},
	// three-element is deliberately absent: it parses its own FlagSet (and its
	// diff subcommand a second one) before the shared set is built, so
	// validateSelectedReadOnlyFlags is never called for it. Listing it here
	// breaks the rule this table states — an entry and a catalog record are one
	// set — because `tei commands` does not advertise it.
}

// repeatableFlags lists the flags that collect several values by design. They
// are registered with flag.Var rather than flag.String/Int, so repeating them
// appends instead of overwriting. Every other advertised flag holds one value:
// a second occurrence would silently replace the first, so it is refused
// (#3675).
var repeatableFlags = map[string]map[string]bool{
	"view":                                   {"input": true, "allow-capability": true},
	"validate":                               {},
	"knowledge-context":                      {},
	"inspect":                                {},
	"trace":                                  {},
	"source-index":                           {},
	"workspace-files":                        {},
	"requirements":                           {},
	"plugin-lock":                            {},
	"plugin-resolve":                         {},
	"plugin-runtime-plan":                    {},
	"plugin-inventory":                       {},
	"knowledge-qualify":                      {},
	"knowledge-review-plan":                  {},
	"knowledge-review-handoff":               {},
	"knowledge-review-disclose":              {},
	"knowledge-data-design-candidate-review": {},
	"model catalog":                          {},
	"model locate":                           {},
	"model overview":                         {},
	"model document-draft":                   {"section": true},
}

// validateRepeatableArguments refuses a flag that was supplied more than once
// when the flag holds one value. Go's flag package keeps the last value and
// reports nothing, so a caller that concatenates arguments (an environment
// default plus an explicit override, for example) loses the earlier one without
// a trace. Repeats of a documented repeatable flag are fine.
func validateRepeatableArguments(command string, args []string) error {
	repeatable := repeatableFlags[command]
	seen := map[string]bool{}
	afterValue := false
	for _, arg := range args {
		if afterValue {
			// The previous argument was a flag that takes its value as the next
			// argument, so this one is that value, not a flag.
			afterValue = false
			continue
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" || arg == "--" {
			break // flag parsing stops at the first positional argument
		}
		name := strings.TrimPrefix(arg, "-")
		name = strings.TrimPrefix(name, "-")
		if index := strings.IndexByte(name, '='); index >= 0 {
			name = name[:index]
		} else {
			afterValue = true
		}
		if seen[name] && !repeatable[name] {
			return fmt.Errorf("%s: -%s was given more than once; it holds one value (omit the earlier one)", command, name)
		}
		seen[name] = true
	}
	return nil
}

// Each selected read-only CLI command owns its advertised flags, even though
// the older tei binary shares one FlagSet across many commands. Rejecting an
// unrelated *recognized* flag prevents silent loss of caller intent.
// rejectPositionalArguments turns an unexpected bare word into an argument
// error. Every command reads its inputs from flags, so a positional argument is
// a typo. `validate` and `world` used to ignore them and still report success,
// which makes a mistyped run look like a clean one (#4129).
func rejectPositionalArguments(fs *flag.FlagSet, command string) error {
	if fs.NArg() == 0 {
		return nil
	}
	return fmt.Errorf("%s does not take positional arguments: %s",
		command, strings.Join(fs.Args(), " "))
}

func validateSelectedReadOnlyFlags(fs *flag.FlagSet, command string) error {
	names, ok := selectedReadOnlyFlagAllowlist[command]
	if !ok {
		return nil
	}
	allowed := make(map[string]bool, len(names))
	for _, name := range names {
		allowed[name] = true
	}
	var err error
	fs.Visit(func(f *flag.Flag) {
		if !allowed[f.Name] && err == nil {
			err = fmt.Errorf("%s does not accept -%s", command, f.Name)
		}
	})
	if err != nil {
		return err
	}
	// A string flag given as "" is not the same claim as an omitted flag (#3605).
	// The flag default is what an omitted flag means; an explicit empty string
	// used to fall through to a different value inside the operation (for
	// example trace -mode "" became "full" while the advertised default is
	// "summary"). Reject it here so the advertised default and the applied
	// value stay the same value.
	return validateNoEmptyStringArguments(fs, command)
}

// validateNoEmptyStringArguments rejects an explicitly supplied empty string for
// the read-only commands whose flags feed a closed vocabulary or a fingerprint.
// Only flags that were actually set are inspected, so omitting a flag keeps its
// advertised default.
func validateNoEmptyStringArguments(fs *flag.FlagSet, command string) error {
	guarded := emptyStringGuardedFlags[command]
	if len(guarded) == 0 {
		return nil
	}
	var err error
	fs.Visit(func(f *flag.Flag) {
		if err != nil {
			return
		}
		if guarded[f.Name] && f.Value.String() == "" {
			err = fmt.Errorf("%s: -%s must not be an empty string; omit the flag to use its default", command, f.Name)
		}
	})
	return err
}

// emptyStringGuardedFlags lists, per command, the flags for which an explicit
// empty string is a different claim than an omitted flag: a closed value set
// (-mode, -direction, -format), an exact selector (-scope, -subject-kind,
// -subject-id, -namespace, -application), or a fingerprint that switches on a
// comparison (-expected-fingerprint and friends). A path or an output target
// stays out of this list, because "" there already means "not given" and the
// command reports its own required-argument error.
var emptyStringGuardedFlags = map[string]map[string]bool{
	"inspect": {
		"subject-kind": true, "subject-id": true, "workspace": true,
	},
	"trace": {
		"workspace": true, "scope": true, "subject-kind": true, "subject-id": true,
		"direction": true, "mode": true, "expected-fingerprint": true,
	},
	"source-index": {
		"workspace": true, "subject-kind": true, "subject-id": true,
	},
	"workspace-files": {"workspace": true},
	"knowledge-context": {
		"workspace": true, "subject-kind": true, "subject-id": true,
		"disclose":                         true,
		"expected-data-design-fingerprint": true,
		"expected-disclosure-fingerprint":  true,
	},
	"knowledge-qualify": {
		"workspace": true, "subject-kind": true, "subject-id": true,
		"qualification-requirements": true,
	},
	"knowledge-review-plan": {
		"workspace": true, "subject-kind": true, "subject-id": true,
		"expected-review-plan-fingerprint": true,
	},
	"knowledge-review-handoff": {
		"workspace": true, "subject-kind": true, "subject-id": true,
		"qualification-requirements":          true,
		"expected-review-handoff-fingerprint": true,
	},
	"knowledge-review-disclose": {
		"workspace": true, "subject-kind": true, "subject-id": true,
		"review-handoff":                      true,
		"expected-review-handoff-fingerprint": true,
		"expected-disclosure-fingerprint":     true,
	},
	"knowledge-data-design-candidate-review": {
		"workspace": true, "subject-kind": true, "subject-id": true,
		"data-design-candidate-kind":                                      true,
		"data-design-candidate-fingerprint":                               true,
		"expected-data-design-fingerprint":                                true,
		"expected-data-design-candidate-evidence-requirement-fingerprint": true,
		"expected-data-design-candidate-review-fingerprint":               true,
	},
	"requirements": {"workspace": true},
	"plugin-lock":  {"workspace": true, "environment": true},
	"plugin-resolve": {
		"workspace": true, "environment": true, "plugin-lock": true,
	},
	"plugin-runtime-plan": {
		"workspace": true, "environment": true, "plugin-lock": true,
	},
	"plugin-inventory": {
		"workspace": true, "environment": true, "plugin-lock": true, "plugin": true,
	},
	// three-element reads its own FlagSet, so this guard never runs for it. It
	// reports its own required-argument error instead.
}
