package dispatch

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// Harness is the common execution interface for DSH/StepCode.
type Harness interface {
	Execute(ctx context.Context, task string, modelID string) (*RunResult, error)
	IsAvailable(ctx context.Context) bool
	SetDryRun(v bool)
}

// Dispatcher orchestrates role-based work acquisition, execution, and handoff.
type Dispatcher struct {
	cfg     Config
	store   *LeaseStore
	github  *GitHubAdapter
	harness Harness
}

// NewDispatcher wires config and adapters together.
func NewDispatcher(cfg Config, store *LeaseStore, gh *GitHubAdapter, harness Harness) *Dispatcher {
	return &Dispatcher{cfg: cfg, store: store, github: gh, harness: harness}
}

// ClaimResult reports the outcome of a claim attempt.
type ClaimResult struct {
	Claim *Claim     `json:"claim"`
	Issue ReadyIssue `json:"issue"`
}

// ClaimReadyIssue finds an issue labeled route:<role> and state:ready, then claims it exclusively.
func (d *Dispatcher) ClaimReadyIssue(ctx context.Context, role string) (*ClaimResult, error) {
	issues, err := d.github.ListReadyIssues(ctx, role)
	if err != nil {
		return nil, fmt.Errorf("list ready issues: %w", err)
	}
	if len(issues) == 0 {
		return nil, fmt.Errorf("no ready issues for role %q", role)
	}

	model, err := ResolveRoleModel(d.cfg, role)
	if err != nil {
		return nil, fmt.Errorf("resolve model: %w", err)
	}

	for _, issue := range issues {
		ttl, _ := time.ParseDuration(d.cfg.Lease.DefaultTTL)
		claim, err := d.store.ClaimIssue(ctx, issue.Number, role, model.ID, model.Harness, "", ttl)
		if err == ErrAlreadyClaimed {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("claim issue %d: %w", issue.Number, err)
		}
		if err := d.github.TransitionToWorking(ctx, issue.Number, role); err != nil {
			_ = d.store.Release(ctx, issue.Number)
			return nil, fmt.Errorf("transition to working: %w", err)
		}
		_ = d.store.UpdateState(ctx, issue.Number, "working")
		return &ClaimResult{Claim: claim, Issue: issue}, nil
	}
	return nil, fmt.Errorf("all ready issues for role %q already claimed", role)
}

// ReleaseIssue releases a claim and transitions the issue back to ready.
func (d *Dispatcher) ReleaseIssue(ctx context.Context, issueNumber int64) error {
	if err := d.store.Release(ctx, issueNumber); err != nil {
		return err
	}
	// Label transition back to ready is done by caller if needed.
	return nil
}

// RunClaim executes the task for a claimed issue and handles escalation or review.
func (d *Dispatcher) RunClaim(ctx context.Context, claim *Claim, task string) (*RunResult, error) {
	if claim == nil {
		return nil, errors.New("claim is nil")
	}
	result, err := d.harness.Execute(ctx, task, claim.ModelID)
	if err != nil {
		return nil, fmt.Errorf("execute task: %w", err)
	}
	if err := ValidateRunResult(result); err != nil {
		// Escalate to diagnostician on failure.
		escRole := "diagnostician"
		escModel, mErr := ResolveRoleModel(d.cfg, escRole)
		if mErr != nil {
			return result, fmt.Errorf("escalate: %w", mErr)
		}
		escTTL, _ := time.ParseDuration(d.cfg.Lease.DefaultTTL)
		escClaim, cErr := d.store.ClaimIssue(ctx, claim.IssueNumber, escRole, escModel.ID, escModel.Harness, "", escTTL)
		if cErr != nil {
			return result, fmt.Errorf("escalate claim failed: %w", cErr)
		}
		_ = escClaim
		_ = d.store.UpdateState(ctx, claim.IssueNumber, "blocked")
		_ = d.github.TransitionToReview(ctx, claim.IssueNumber)
		return result, fmt.Errorf("task failed, escalated to %q: %w", escRole, err)
	}
	_ = d.store.UpdateState(ctx, claim.IssueNumber, "review")
	_ = d.github.TransitionToReview(ctx, claim.IssueNumber)
	return result, nil
}

// RunClaimAsync starts a task for a claimed issue and returns immediately.
// It records the execution metadata to lease store for later monitoring.
// Note: This is a best-effort async implementation; actual async depends on CLI support.
func (d *Dispatcher) RunClaimAsync(ctx context.Context, claim *Claim, task string) (*RunResult, error) {
	if claim == nil {
		return nil, errors.New("claim is nil")
	}

	// Use the first available async-capable worker if configured.
	// Currently falls back to synchronous execution with process tracking.
	cmd := buildAsyncCommand(d.cfg, claim, task)
	if cmd == nil {
		return d.RunClaim(ctx, claim, task)
	}

	workDir := cmd.Dir
	if workDir == "" {
		workDir, _ = os.Getwd()
	}
	executionID := fmt.Sprintf("exec-%d-%d", claim.IssueNumber, time.Now().UnixNano())

	// Update lease with execution metadata
	_ = d.store.UpdateHeadSHA(ctx, claim.IssueNumber, "")
	_ = d.store.UpdateState(ctx, claim.IssueNumber, "working")

	result := &RunResult{
		Harness:   claim.Harness,
		ModelID:   claim.ModelID,
		DryRun:    d.github.IsDryRun(),
		SessionID: executionID,
		Stdout:    fmt.Sprintf("[async] started %s for issue %d in %s", cmd.Path, claim.IssueNumber, workDir),
	}

	if d.github.IsDryRun() {
		return result, nil
	}

	if err := cmd.Start(); err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to start async worker: %v", err)
		result.ExitCode = 1
		return result, nil
	}

	result.ProcessID = cmd.Process.Pid
	return result, nil
}

func buildAsyncCommand(cfg Config, claim *Claim, task string) *exec.Cmd {
	// For now, only build async commands when not in dry-run mode.
	if cfg.Harness.DefaultHarness == "" || cfg.GitHub.Repo == "" {
		return nil
	}

	switch claim.Harness {
	case "step-code":
		stepCfg := cfg.Harness.StepCode
		if stepCfg.CLI == "" {
			stepCfg.CLI = "step"
		}
		args := []string{"--json", "--task", task}
		if stepCfg.Profile != "" {
			args = append(args, "--profile", stepCfg.Profile)
		}
		if stepCfg.Plan != "" {
			args = append(args, "--plan", stepCfg.Plan)
		}
		if stepCfg.WorkDir != "" {
			cmd := exec.Command(stepCfg.CLI, args...)
			cmd.Dir = stepCfg.WorkDir
			return cmd
		}
		return exec.Command(stepCfg.CLI, args...)
	case "dsh":
		fallthrough
	default:
		dshCfg := cfg.Harness.DSH
		if dshCfg.CLI == "" {
			dshCfg.CLI = "dsh"
		}
		args := []string{"--profile", dshCfg.Profile, "--json", "--task", task}
		if claim.ModelID != "" {
			args = append(args, "--model", claim.ModelID)
		}
		if dshCfg.TaskTimeout != "" {
			if ttl, err := time.ParseDuration(dshCfg.TaskTimeout); err == nil {
				timeoutCtx, cancel := context.WithTimeout(context.Background(), ttl)
				defer cancel()
				cmd := exec.CommandContext(timeoutCtx, dshCfg.CLI, args...)
				return cmd
			}
		}
		return exec.Command(dshCfg.CLI, args...)
	}
}

// RequestPRReview creates a PR and requests review from the specified review role.
func (d *Dispatcher) RequestPRReview(ctx context.Context, claim *Claim, title, body, head, base, reviewRole string) (int64, error) {
	prNumber, err := d.github.CreatePR(ctx, title, body, head, base)
	if err != nil {
		return 0, fmt.Errorf("create pr: %w", err)
	}
	if prNumber > 0 {
		_ = d.store.UpdatePR(ctx, claim.IssueNumber, prNumber)
	}
	if err := d.github.RequestReview(ctx, prNumber, reviewRole); err != nil {
		return prNumber, fmt.Errorf("request review: %w", err)
	}
	return prNumber, nil
}

// SweepExpiredLeases expires old claims so they can be reclaimed.
func (d *Dispatcher) SweepExpiredLeases(ctx context.Context) ([]Claim, error) {
	return d.store.SweepExpired(ctx)
}
