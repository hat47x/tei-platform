package dispatch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// DSHAdapter provides a thin wrapper around DSH headless/CLI.
type DSHAdapter struct {
	profile       string
	cli           string
	defaultDryRun bool
	taskTimeout   time.Duration
	dryRun        bool
}

// NewDSHAdapter constructs a DSH adapter.
func NewDSHAdapter(profile, cli string, defaultDryRun bool, taskTimeout time.Duration) *DSHAdapter {
	if cli == "" {
		cli = "dsh"
	}
	return &DSHAdapter{
		profile:       profile,
		cli:           cli,
		defaultDryRun: defaultDryRun,
		taskTimeout:   taskTimeout,
		dryRun:        defaultDryRun,
	}
}

// SetDryRun overrides dry-run mode.
func (a *DSHAdapter) SetDryRun(v bool) {
	a.dryRun = v
}

// RunResult is the observable outcome of a harness execution.
type RunResult struct {
	Harness      string `json:"harness"`
	ModelID      string `json:"modelId"`
	SessionID    string `json:"sessionId,omitempty"`
	DryRun       bool   `json:"dryRun"`
	ExitCode     int    `json:"exitCode"`
	Stdout       string `json:"stdout,omitempty"`
	Stderr       string `json:"stderr,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	ProcessID    int    `json:"processId,omitempty"`
}

// Execute runs a task via DSH headless if available; otherwise returns a dry-run result.
func (a *DSHAdapter) Execute(ctx context.Context, task string, modelID string) (*RunResult, error) {
	result := &RunResult{
		Harness:   "dsh",
		ModelID:   modelID,
		DryRun:    a.dryRun,
		SessionID: newSessionID(),
	}
	if a.dryRun {
		result.Stdout = fmt.Sprintf("[dry-run] task for model %q: %s", modelID, task)
		if strings.Contains(strings.ToLower(task), "fail") {
			result.ExitCode = 1
			result.ErrorMessage = "simulated failure"
		}
		return result, nil
	}

	args := []string{"--profile", a.profile, "--json", "--task", task}
	if modelID != "" {
		args = append(args, "--model", modelID)
	}
	cmd := exec.CommandContext(ctx, a.cli, args...)
	out, err := cmd.CombinedOutput()
	result.Stdout = string(out)
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
			result.ErrorMessage = exitErr.String()
			return result, nil
		}
		result.ErrorMessage = err.Error()
		return result, fmt.Errorf("execute dsh task: %w", err)
	}
	result.ExitCode = 0
	return result, nil
}

// IsAvailable reports whether the DSH CLI is reachable.
func (a *DSHAdapter) IsAvailable(ctx context.Context) bool {
	if a.dryRun {
		return true
	}
	cmd := exec.CommandContext(ctx, a.cli, "--version")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

// SessionID generates a simple unique session identifier for tracing.
func newSessionID() string {
	return fmt.Sprintf("session-%d", time.Now().UnixNano())
}

// ValidateRunResult checks whether a run result represents success.
func ValidateRunResult(r *RunResult) error {
	if r == nil {
		return errors.New("run result is nil")
	}
	if r.ExitCode != 0 {
		return fmt.Errorf("harness exited with code %d: %s", r.ExitCode, r.ErrorMessage)
	}
	if strings.Contains(r.Stdout, "error") || strings.Contains(r.Stdout, "Error") {
		return errors.New("harness stdout contains error marker")
	}
	return nil
}

// MarshalRunResult encodes a RunResult to JSON for logging/evidence.
func MarshalRunResult(r *RunResult) ([]byte, error) {
	return json.Marshal(r)
}
