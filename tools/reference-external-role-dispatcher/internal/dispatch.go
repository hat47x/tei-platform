package dispatch

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Dispatcher orchestrates role-based work acquisition, execution, and handoff.
type Dispatcher struct {
	cfg     Config
	store   *LeaseStore
	github  *GitHubAdapter
	harness *DSHAdapter
}

// NewDispatcher wires config and adapters together.
func NewDispatcher(cfg Config, store *LeaseStore, gh *GitHubAdapter, harness *DSHAdapter) *Dispatcher {
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
