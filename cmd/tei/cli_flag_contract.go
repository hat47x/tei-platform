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

// Each selected read-only CLI command owns its advertised flags, even though
// the older tei binary shares one FlagSet across many commands. Rejecting an
// unrelated *recognized* flag prevents silent loss of caller intent.
func validateSelectedReadOnlyFlags(fs *flag.FlagSet, command string) error {
	var names []string
	switch command {
	case "inspect":
		names = []string{"workspace", "subject-kind", "subject-id"}
	case "trace":
		names = []string{
			"workspace", "scope", "subject-kind", "subject-id",
			"direction", "mode", "offset", "limit", "expected-fingerprint",
		}
	default:
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
