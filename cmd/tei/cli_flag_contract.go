package main

import (
	"flag"
	"fmt"
	"io"
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
	"inspect": {"workspace", "subject-kind", "subject-id"},
	"view":    {"workspace", "application", "pattern", "format"},
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
	"knowledge-qualify":   {"workspace", "subject-kind", "subject-id", "qualification-requirements"},
	"version":             {"json"},
	"validate":            {"workspace", "world", "json"},
	"world":               {"workspace", "world"},
	"requirements":        {"workspace", "world"},
	"plugin-lock":         {"workspace", "world"},
	"plugin-resolve":      {"workspace", "world", "environment", "plugin-lock"},
	"plugin-runtime-plan": {"workspace", "world", "environment", "plugin-lock"},
}

// Each selected read-only CLI command owns its advertised flags, even though
// the older tei binary shares one FlagSet across many commands. Rejecting an
// unrelated *recognized* flag prevents silent loss of caller intent.
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
	return err
}
