package main

import (
	"context"
	"errors"
	"testing"
	"time"

	dispatch "tei/tools/reference-external-role-dispatcher/internal"
)

func TestClaimRunEscalateReview(t *testing.T) {
	cfg := dispatch.Config{
		Schema: dispatch.Schema,
		Models: []dispatch.Model{
			{ID: "worker", Harness: "dsh", Roles: []string{"implementer"}, MaxConcurrent: 1, Enabled: true, Priority: 10, CostTier: "low"},
			{ID: "doctor", Harness: "dsh", Roles: []string{"diagnostician"}, MaxConcurrent: 1, Enabled: true, Priority: 20, CostTier: "medium"},
		},
		Roles: []dispatch.RoleConfig{
			{Name: "implementer", ModelIDs: []string{"worker"}},
			{Name: "diagnostician", ModelIDs: []string{"doctor"}},
		},
		GitHub: dispatch.GitHubConfig{
			Repo:         "owner/repo",
			AllowedRepos: []string{"owner/repo"},
			ReadyState:   "ready",
			WorkingState: "working",
			ReviewState:  "review",
			BlockedState: "blocked",
		},
		Harness: dispatch.HarnessConfig{
			DefaultHarness: "dsh",
			DSH:            dispatch.DSHConfig{Profile: "headless", CLI: "dsh", DefaultDryRun: true, TaskTimeout: "5m"},
		},
		Lease: dispatch.LeaseConfig{DefaultTTL: "10m", DBPath: t.TempDir() + "/lease.db"},
	}

	store, err := dispatch.NewLeaseStore(cfg.Lease.DBPath)
	if err != nil {
		t.Fatalf("NewLeaseStore: %v", err)
	}
	defer store.Close()

	gh := dispatch.NewGitHubAdapter(cfg.GitHub.Repo, "gh", true)
	harness := dispatch.NewDSHAdapter(cfg.Harness.DSH.Profile, cfg.Harness.DSH.CLI, cfg.Harness.DSH.DefaultDryRun, 5*time.Minute)
	d := dispatch.NewDispatcher(cfg, store, gh, harness)

	ctx := context.Background()

	// Claim a ready issue for implementer.
	res, err := d.ClaimReadyIssue(ctx, "implementer")
	if err != nil {
		t.Fatalf("ClaimReadyIssue: %v", err)
	}
	if res.Claim.Role != "implementer" {
		t.Errorf("ClaimReadyIssue role = %s, want implementer", res.Claim.Role)
	}
	if res.Claim.ModelID != "worker" {
		t.Errorf("ClaimReadyIssue model = %s, want worker", res.Claim.ModelID)
	}

	// Run with failing task -> should escalate to diagnostician.
	result, err := d.RunClaim(ctx, res.Claim, "fail task")
	if err == nil {
		t.Fatalf("RunClaim expected error for failing task")
	}
	if result == nil {
		t.Fatalf("RunClaim result is nil")
	}
	if result.DryRun != true {
		t.Errorf("RunClaim dryRun = %v, want true", result.DryRun)
	}

	// Request review (dry-run, should not error).
	prNumber, err := d.RequestPRReview(ctx, res.Claim, "title", "body", "head", "main", "reviewer")
	if err != nil {
		t.Fatalf("RequestPRReview: %v", err)
	}
	if prNumber != -1 {
		t.Errorf("RequestPRReview prNumber = %d, want -1 (dry-run)", prNumber)
	}
}

func TestResolveRoleModelSwap(t *testing.T) {
	cfg := dispatch.Config{
		Schema: dispatch.Schema,
		Models: []dispatch.Model{
			{ID: "m1", Harness: "dsh", Roles: []string{"implementer"}, MaxConcurrent: 1, Enabled: true, Priority: 10, CostTier: "low"},
			{ID: "m2", Harness: "dsh", Roles: []string{"implementer"}, MaxConcurrent: 1, Enabled: false, Priority: 5, CostTier: "low"},
		},
		Roles: []dispatch.RoleConfig{
			{Name: "implementer", ModelIDs: []string{"m1", "m2"}},
		},
		GitHub: dispatch.GitHubConfig{Repo: "owner/repo", AllowedRepos: []string{"owner/repo"}},
	}

	m, err := dispatch.ResolveRoleModel(cfg, "implementer")
	if err != nil {
		t.Fatalf("ResolveRoleModel: %v", err)
	}
	if m.ID != "m1" {
		t.Errorf("ResolveRoleModel = %s, want m1", m.ID)
	}

	// Disable m1, should fallback to m2? No, m2 is disabled too, so should error.
	cfg.Models[0].Enabled = false
	_, err = dispatch.ResolveRoleModel(cfg, "implementer")
	if err == nil {
		t.Fatalf("ResolveRoleModel expected error when all models disabled")
	}
}

func TestRequestReviewRejection(t *testing.T) {
	cfg := dispatch.Config{
		Schema: dispatch.Schema,
		Models: []dispatch.Model{
			{ID: "worker", Harness: "dsh", Roles: []string{"implementer"}, MaxConcurrent: 1, Enabled: true, Priority: 10, CostTier: "low"},
		},
		Roles: []dispatch.RoleConfig{
			{Name: "implementer", ModelIDs: []string{"worker"}},
		},
		GitHub: dispatch.GitHubConfig{
			Repo:         "owner/repo",
			AllowedRepos: []string{"owner/repo"},
			ReadyState:   "ready",
			WorkingState: "working",
			ReviewState:  "review",
			BlockedState: "blocked",
		},
		Harness: dispatch.HarnessConfig{
			DefaultHarness: "dsh",
			DSH:            dispatch.DSHConfig{Profile: "headless", CLI: "dsh", DefaultDryRun: true, TaskTimeout: "5m"},
		},
		Lease: dispatch.LeaseConfig{DefaultTTL: "10m", DBPath: t.TempDir() + "/lease.db"},
	}

	store, err := dispatch.NewLeaseStore(cfg.Lease.DBPath)
	if err != nil {
		t.Fatalf("NewLeaseStore: %v", err)
	}
	defer store.Close()

	gh := dispatch.NewGitHubAdapter(cfg.GitHub.Repo, "gh", true)
	gh.ForceRequestReviewError = errors.New("simulated review rejection")
	harness := dispatch.NewDSHAdapter(cfg.Harness.DSH.Profile, cfg.Harness.DSH.CLI, cfg.Harness.DSH.DefaultDryRun, 5*time.Minute)
	d := dispatch.NewDispatcher(cfg, store, gh, harness)

	ctx := context.Background()
	res, err := d.ClaimReadyIssue(ctx, "implementer")
	if err != nil {
		t.Fatalf("ClaimReadyIssue: %v", err)
	}

	prNumber, err := d.RequestPRReview(ctx, res.Claim, "title", "body", "head", "main", "reviewer")
	if err == nil {
		t.Fatalf("RequestPRReview expected error for review rejection")
	}
	if prNumber != -1 {
		t.Errorf("RequestPRReview prNumber = %d, want -1 on failure", prNumber)
	}
}
