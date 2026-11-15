package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	dispatch "tei/tools/reference-external-role-dispatcher/internal"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: dispatcher <claim|release|run|run-async|sweep|model> [options]")
		os.Exit(2)
	}
	command := os.Args[1]
	fs := flag.NewFlagSet(command, flag.ExitOnError)
	configPath := fs.String("config", "tools/reference-external-role-dispatcher/config.example.json", "dispatcher config JSON")
	dryRun := fs.Bool("dry-run", true, "run without mutating GitHub state")
	role := fs.String("role", "", "required role for claim/run/model")
	issue := fs.Int64("issue", 0, "issue number for release/run")
	task := fs.String("task", "", "task description for run")
	harness := fs.String("harness", "", "override harness: dsh|step-code")
	_ = fs.Parse(os.Args[2:])

	cfg, err := dispatch.LoadConfig(*configPath)
	if err != nil {
		fail(fmt.Errorf("load config: %w", err))
	}

	gh := dispatch.NewGitHubAdapter(cfg.GitHub.Repo, "gh", *dryRun)
	store, err := dispatch.NewLeaseStore(cfg.Lease.DBPath)
	if err != nil {
		fail(fmt.Errorf("open lease store: %w", err))
	}
	defer store.Close()

	taskTimeout, _ := time.ParseDuration(cfg.Harness.DSH.TaskTimeout)
	if taskTimeout <= 0 {
		taskTimeout = 5 * time.Minute
	}

	runner, err := buildHarness(cfg, *harness, taskTimeout, *dryRun)
	if err != nil {
		fail(err)
	}
	d := dispatch.NewDispatcher(cfg, store, gh, runner)

	ctx := context.Background()

	switch command {
	case "claim":
		if *role == "" {
			fail(fmt.Errorf("-role is required"))
		}
		res, err := d.ClaimReadyIssue(ctx, *role)
		if err != nil {
			fail(err)
		}
		emit(map[string]any{
			"event":        "claimed",
			"issue":        res.Issue.Number,
			"role":         res.Claim.Role,
			"model":        res.Claim.ModelID,
			"leaseExpires": res.Claim.LeaseExpires,
		})
	case "release":
		if *issue == 0 {
			fail(fmt.Errorf("-issue is required"))
		}
		if err := d.ReleaseIssue(ctx, *issue); err != nil {
			fail(err)
		}
		emit(map[string]any{"event": "released", "issue": *issue})
	case "run":
		if *role == "" {
			fail(fmt.Errorf("-role is required"))
		}
		if *issue == 0 {
			fail(fmt.Errorf("-issue is required"))
		}
		if *task == "" {
			fail(fmt.Errorf("-task is required"))
		}
		claim, err := store.GetClaim(ctx, *issue)
		if err != nil {
			fail(fmt.Errorf("get claim: %w", err))
		}
		if claim == nil {
			fail(fmt.Errorf("no active claim for issue %d", *issue))
		}
		result, err := d.RunClaim(ctx, claim, *task)
		if err != nil {
			emit(map[string]any{"event": "run_failed", "issue": *issue, "error": err.Error(), "result": sanitizeResult(result)})
			os.Exit(1)
		}
		emit(map[string]any{"event": "run_completed", "issue": *issue, "result": sanitizeResult(result)})
	case "run-async":
		if *role == "" {
			fail(fmt.Errorf("-role is required"))
		}
		if *issue == 0 {
			fail(fmt.Errorf("-issue is required"))
		}
		if *task == "" {
			fail(fmt.Errorf("-task is required"))
		}
		claim, err := store.GetClaim(ctx, *issue)
		if err != nil {
			fail(fmt.Errorf("get claim: %w", err))
		}
		if claim == nil {
			fail(fmt.Errorf("no active claim for issue %d", *issue))
		}
		result, err := d.RunClaimAsync(ctx, claim, *task)
		if err != nil {
			emit(map[string]any{"event": "run_async_failed", "issue": *issue, "error": err.Error(), "result": sanitizeResult(result)})
			os.Exit(1)
		}
		emit(map[string]any{"event": "run_async_started", "issue": *issue, "result": sanitizeResult(result)})
	case "sweep":
		expired, err := d.SweepExpiredLeases(ctx)
		if err != nil {
			fail(err)
		}
		emit(map[string]any{"event": "swept", "expired": len(expired)})
	case "model":
		if *role == "" {
			fail(fmt.Errorf("-role is required"))
		}
		m, err := dispatch.ResolveRoleModel(cfg, *role)
		if err != nil {
			fail(err)
		}
		emit(map[string]any{"role": *role, "model": m.ID, "harness": m.Harness, "priority": m.Priority})
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", command)
		os.Exit(2)
	}
}

func buildHarness(cfg dispatch.Config, harnessOverride string, taskTimeout time.Duration, dryRun bool) (dispatch.Harness, error) {
	target := strings.TrimSpace(strings.ToLower(cfg.Harness.DefaultHarness))
	if harnessOverride != "" {
		target = harnessOverride
	}
	switch target {
	case "step-code", "step":
		stepCfg := cfg.Harness.StepCode
		if stepCfg.CLI == "" {
			stepCfg.CLI = "step"
		}
		stepTaskTimeout, _ := time.ParseDuration(stepCfg.TaskTimeout)
		if stepTaskTimeout <= 0 {
			stepTaskTimeout = taskTimeout
		}
		adapter := dispatch.NewStepCodeAdapter(stepCfg, stepTaskTimeout)
		adapter.SetDryRun(dryRun)
		return adapter, nil
	case "dsh", "":
		adapter := dispatch.NewDSHAdapter(cfg.Harness.DSH.Profile, cfg.Harness.DSH.CLI, cfg.Harness.DSH.DefaultDryRun, taskTimeout)
		adapter.SetDryRun(dryRun)
		return adapter, nil
	default:
		return nil, fmt.Errorf("unsupported harness %q", target)
	}
}

func emit(value any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func sanitizeResult(r *dispatch.RunResult) map[string]any {
	if r == nil {
		return nil
	}
	return map[string]any{
		"harness":      r.Harness,
		"modelId":      r.ModelID,
		"sessionId":    r.SessionID,
		"dryRun":       r.DryRun,
		"exitCode":     r.ExitCode,
		"stdout":       r.Stdout,
		"stderr":       r.Stderr,
		"errorMessage": r.ErrorMessage,
	}
}
