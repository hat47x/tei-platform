package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	dispatch "tei/tools/reference-external-role-dispatcher/internal"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: dispatcher <claim|release|run|sweep|model> [options]")
		os.Exit(2)
	}
	command := os.Args[1]
	fs := flag.NewFlagSet(command, flag.ExitOnError)
	configPath := fs.String("config", "tools/reference-external-role-dispatcher/config.example.json", "dispatcher config JSON")
	dryRun := fs.Bool("dry-run", true, "run without mutating GitHub state")
	role := fs.String("role", "", "required role for claim/run/model")
	issue := fs.Int64("issue", 0, "issue number for release/run")
	task := fs.String("task", "", "task description for run")
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

	harness := dispatch.NewDSHAdapter(cfg.Harness.DSH.Profile, cfg.Harness.DSH.CLI, cfg.Harness.DSH.DefaultDryRun, 5*time.Minute)
	d := dispatch.NewDispatcher(cfg, store, gh, harness)

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
