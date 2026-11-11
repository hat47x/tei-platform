package dispatch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

// StepCodeConfig holds Step Code CLI specific configuration.
type StepCodeConfig struct {
	CLI          string `json:"cli"`
	DefaultDryRun bool  `json:"defaultDryRun"`
	TaskTimeout  string `json:"taskTimeout"`
	Profile      string `json:"profile,omitempty"`
	Plan         string `json:"plan,omitempty"`
	WorkDir      string `json:"workDir,omitempty"`
}

// StepCodeAdapter provides a thin wrapper around Step Code CLI.
type StepCodeAdapter struct {
	cli          string
	defaultDryRun bool
	taskTimeout  time.Duration
	profile      string
	plan         string
	workDir      string
	dryRun       bool
}

// NewStepCodeAdapter constructs a Step Code adapter.
func NewStepCodeAdapter(cfg StepCodeConfig, taskTimeout time.Duration) *StepCodeAdapter {
	if cfg.CLI == "" {
		cfg.CLI = "step"
	}
	return &StepCodeAdapter{
		cli:          cfg.CLI,
		defaultDryRun: cfg.DefaultDryRun,
		taskTimeout:   taskTimeout,
		profile:      cfg.Profile,
		plan:         cfg.Plan,
		workDir:      cfg.WorkDir,
		dryRun:       cfg.DefaultDryRun,
	}
}

// SetDryRun overrides dry-run mode.
func (a *StepCodeAdapter) SetDryRun(v bool) {
	a.dryRun = v
}

// Execute runs a task via Step Code CLI and returns a common RunResult view.
func (a *StepCodeAdapter) Execute(ctx context.Context, task string, modelID string) (*RunResult, error) {
	stepResult, err := a.executeStep(ctx, task, modelID)
	if err != nil {
		return nil, err
	}
	return &stepResult.RunResult, nil
}

// IsAvailable reports whether the Step Code CLI is reachable.
func (a *StepCodeAdapter) IsAvailable(ctx context.Context) bool {
	if a.dryRun {
		return true
	}
	cmd := exec.CommandContext(ctx, a.cli, "--version")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

// MarshalRunResult encodes a StepExecuteResult to JSON for logging/evidence.
func MarshalStepExecuteResult(r *StepExecuteResult) ([]byte, error) {
	return json.Marshal(r)
}

type stepExecuteResult struct {
	RunResult
	CLI         string `json:"cli"`
	CLIVersion  string `json:"cliVersion,omitempty"`
	WorkDir     string `json:"workDir,omitempty"`
	Profile     string `json:"profile,omitempty"`
	Plan        string `json:"plan,omitempty"`
	ProcessID   int    `json:"processId,omitempty"`
	ExecutionID string `json:"executionId,omitempty"`
	StartTime   string `json:"startTime,omitempty"`
	EndTime     string `json:"endTime,omitempty"`
}

// StepExecuteResult contains Step Code specific execution metadata.
type StepExecuteResult struct {
	RunResult
	CLI         string `json:"cli"`
	CLIVersion  string `json:"cliVersion,omitempty"`
	WorkDir     string `json:"workDir,omitempty"`
	Profile     string `json:"profile,omitempty"`
	Plan        string `json:"plan,omitempty"`
	ProcessID   int    `json:"processId,omitempty"`
	ExecutionID string `json:"executionId,omitempty"`
	StartTime   string `json:"startTime,omitempty"`
	EndTime     string `json:"endTime,omitempty"`
}

func (a *StepCodeAdapter) executeStep(ctx context.Context, task string, modelID string) (*stepExecuteResult, error) {
	result := &stepExecuteResult{
		RunResult: RunResult{
			Harness:   "step-code",
			ModelID:   modelID,
			DryRun:    a.dryRun,
			SessionID: newSessionID(),
		},
		CLI:       a.cli,
		WorkDir:   a.workDir,
		Profile:   a.profile,
		Plan:      a.plan,
		StartTime: time.Now().Format(time.RFC3339),
	}
	result.EndTime = result.StartTime

	if a.dryRun {
		result.Stdout = fmt.Sprintf("[dry-run] step task for model %q: %s", modelID, task)
		if strings.Contains(strings.ToLower(task), "fail") {
			result.ExitCode = 1
			result.ErrorMessage = "simulated failure"
			return result, nil
		}
		result.ExitCode = 0
		return result, nil
	}

	args := []string{"--json", "--task", task}
	if a.profile != "" {
		args = append(args, "--profile", a.profile)
	}
	if a.plan != "" {
		args = append(args, "--plan", a.plan)
	}
	if a.workDir != "" {
		args = append(args, "--workdir", a.workDir)
	}
	if modelID != "" {
		args = append(args, "--model", modelID)
	}

	cmd := exec.CommandContext(ctx, a.cli, args...)
	out, err := cmd.CombinedOutput()
	result.Stdout = string(out)
	if cmd.Process != nil {
		result.ProcessID = cmd.Process.Pid
	}
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
			result.ErrorMessage = exitErr.String()
			return result, nil
		}
		result.ErrorMessage = err.Error()
		return result, fmt.Errorf("execute step task: %w", err)
	}
	result.ExitCode = 0
	version, _ := a.version(ctx)
	result.CLIVersion = version
	return result, nil
}

func (a *StepCodeAdapter) version(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, a.cli, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
