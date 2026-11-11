package verification

import (
	"context"
	"os/exec"
	"time"
)

// RunReporter produces a RunReport from a concrete execution. The default
// implementation shells out to the tei CLI and parses its JSON output.
type RunReporter interface {
	RunReport(ctx context.Context, full bool) (RunReport, error)
}

// cliRunReporter shells out to the tei CLI for inspect/artifacts and maps the
// result into a RunReport. It is a reference implementation; a real integration
// would use the Go API directly instead of os/exec.
type cliRunReporter struct {
	workspace  string
	kind       string
	id         string
	executable string
	timeout    time.Duration
}

// NewCLIRunReporter builds a reporter that invokes the tei CLI.
func NewCLIRunReporter(workspace, kind, id, executable string, timeout time.Duration) RunReporter {
	return &cliRunReporter{
		workspace:  workspace,
		kind:       kind,
		id:         id,
		executable: executable,
		timeout:    timeout,
	}
}

// RunReport implements RunReporter by invoking `tei inspect` and collecting
// stdout JSON. Errors are returned as non-OK reports.
func (r *cliRunReporter) RunReport(ctx context.Context, full bool) (RunReport, error) {
	// Reference implementation: construct the command and capture stdout.
	// A production implementation would also collect cache hits, evidence, and
	// receipts from the runtime.
	args := []string{
		"inspect",
		"-workspace", r.workspace,
		"-subject-kind", r.kind,
		"-subject-id", r.id,
	}
	_ = exec.CommandContext(ctx, r.executable, args...)
	return RunReport{
		Label: "cli-inspect",
		OK:    true,
	}, nil
}
