// Package orderrollouthost runs the order-rollout-host-continuity sample
// (sample 6): one deployment command of order_rollout@2026.8.27 is invoked
// under isolation and a resource lease by supervising host A, host A stops,
// host B takes over after expiry, reattaches to the execution state, and the
// command is completed only by a re-observation taken after the command.
//
// Everything here is sample-specific. The product code is used through
// internal/plugin/runtime (Registry, HostSandboxLauncher, StdioProcessLauncher,
// ReconcileDurableInvocation), internal/resource (SupervisorOwnershipStore,
// Broker, adapters) and internal/deployment/durable (the run record).
package orderrollouthost

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tei/internal/core"
	"tei/internal/database/migration"
	"tei/internal/database/release"
	"tei/internal/deployment/durable"
	"tei/internal/deployment/executor"
	"tei/internal/environment"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/workspace"
)

// Fixed inputs. They are read, never written.
const (
	DefaultWorkspace  = "reference/workspace"
	DefaultPluginLock = "reference/workspace/.tei/plugin-lock.json"
	DefaultStartRun   = "reference/order_rollout.durable-run.m0.21.start.json"
	ExpectedActionID  = "sales_expand/apply"
	TargetRelease     = "4.1.0"
	SalesDatabase     = "sales_core"
	OrderApplication  = "order_entry"
	LegacyVersion     = "3.7.0"
	FollowingActionID = "order_app/start"
	CapabilityID      = "migration.atlas"
)

// Paths names the three fixed inputs of step 1.
type Paths struct {
	Workspace  string `json:"workspace"`
	PluginLock string `json:"pluginLock"`
	StartRun   string `json:"startRun"`
}

// DefaultPaths resolves the fixed inputs below the repository root.
func DefaultPaths(repo string) Paths {
	return Paths{Workspace: filepath.Join(repo, DefaultWorkspace), PluginLock: filepath.Join(repo, DefaultPluginLock), StartRun: filepath.Join(repo, DefaultStartRun)}
}

// MigrationStep is one SQL step of the migration the command applies.
type MigrationStep struct {
	Kind string `json:"kind"`
	SQL  string `json:"sql"`
}

// Start is the output of step 1. It is what every later step receives.
type Start struct {
	RunID                    string           `json:"runId"`
	SolutionID               string           `json:"solutionId"`
	SolutionVersion          string           `json:"solutionVersion"`
	Environment              core.CanonicalID `json:"environment"`
	StartRecordDigest        string           `json:"startRecordDigest"`
	PlanFingerprint          string           `json:"planFingerprint"`
	PluginLockFingerprint    string           `json:"pluginLockFingerprint"`
	RuntimePolicyFingerprint string           `json:"runtimePolicyFingerprint"`
	EnvironmentPlanDigest    string           `json:"environmentPlanDigest"`
	Command                  executor.Command `json:"command"`
	InvocationID             string           `json:"invocationId"`
	RequestID                string           `json:"requestId"`
	// The start observation as recorded in the run: a scenario-time value,
	// never evidence of the current state.
	InitialSalesRelease string `json:"initialSalesRelease"`
	InitialObservedAt   string `json:"initialObservedAt"`
	TargetRelease       string `json:"targetRelease"`
	TargetSchemaFP      string `json:"targetSchemaFingerprint"`
	MigrationID         string `json:"migrationId"`
	// Checks of the fixed run against the plan built from the workspace.
	PluginLockMatches    bool `json:"pluginLockMatches"`
	RuntimePolicyMatches bool `json:"runtimePolicyMatches"`

	Run        durable.Run              `json:"-"`
	Plan       pruntime.EnvironmentPlan `json:"-"`
	Migration  []MigrationStep          `json:"-"`
	TargetDef  release.Definition       `json:"-"`
	ReadyCount int                      `json:"readyCommands"`
}

func digest(b []byte) string {
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func jsonDigest(v any) string {
	raw, _ := json.Marshal(v)
	return digest(raw)
}

func findEnv(xs []environment.Environment, id core.CanonicalID) (environment.Environment, bool) {
	for _, e := range xs {
		if e.ID == id {
			return e, true
		}
	}
	return environment.Environment{}, false
}

// LoadStart is step 1: read the fixed start record, build the environment
// plan from the workspace and take the first ready command. It fails unless
// the first ready command is sales_expand/apply.
func LoadStart(p Paths) (Start, error) {
	loaded, err := workspace.Load(p.Workspace)
	if err != nil {
		return Start{}, fmt.Errorf("load workspace: %w", err)
	}
	var lock plugin.Lock
	raw, err := os.ReadFile(p.PluginLock)
	if err != nil {
		return Start{}, err
	}
	if err := json.Unmarshal(raw, &lock); err != nil {
		return Start{}, fmt.Errorf("plugin lock: %w", err)
	}
	runRaw, err := os.ReadFile(p.StartRun)
	if err != nil {
		return Start{}, err
	}
	var run durable.Run
	if err := json.Unmarshal(runRaw, &run); err != nil {
		return Start{}, fmt.Errorf("start record: %w", err)
	}
	env, ok := findEnv(loaded.World.Environments, run.Environment)
	if !ok {
		return Start{}, fmt.Errorf("environment %s not found", run.Environment)
	}
	plan, err := pruntime.BuildPlan(plugin.Catalog{Plugins: loaded.World.Plugins, Bundles: loaded.World.Bundles}, env, lock)
	if err != nil {
		return Start{}, fmt.Errorf("build plan: %w", err)
	}
	cmds := durable.ReadyCommands(run)
	if len(cmds) != 1 || cmds[0].ActionID != ExpectedActionID {
		ids := []string{}
		for _, c := range cmds {
			ids = append(ids, c.ActionID)
		}
		return Start{}, fmt.Errorf("expected the only ready command %s, got %v", ExpectedActionID, ids)
	}
	cmd := cmds[0]
	if cmd.Version != TargetRelease || string(cmd.Database) != SalesDatabase {
		return Start{}, fmt.Errorf("command %s targets %s %s, want %s %s", cmd.ActionID, cmd.Database, cmd.Version, SalesDatabase, TargetRelease)
	}
	var target release.Definition
	for _, d := range loaded.World.DatabaseReleases {
		if string(d.Database) == SalesDatabase && d.Version == TargetRelease {
			target = d.Normalize()
		}
	}
	if target.Version == "" || len(target.Migrations) != 1 {
		return Start{}, fmt.Errorf("release %s %s with exactly one migration not found", SalesDatabase, TargetRelease)
	}
	var steps []MigrationStep
	for _, m := range loaded.World.Migrations {
		if string(m.Database) != SalesDatabase || m.ID != target.Migrations[0] {
			continue
		}
		for _, s := range m.Steps {
			if s.SQL == nil || s.SQL.Provider != environment.DatabaseProvider("postgres") {
				continue
			}
			steps = append(steps, MigrationStep{Kind: string(s.Kind), SQL: strings.TrimSpace(s.SQL.Source)})
		}
	}
	if len(steps) == 0 || steps[0].Kind != string(migration.StepSchema) {
		return Start{}, fmt.Errorf("migration %s has no postgres schema step", target.Migrations[0])
	}
	initial := ""
	if st, ok := run.Initial.DatabaseState(core.CanonicalID(SalesDatabase)); ok {
		initial = st.Release
	}
	s := Start{
		RunID: run.RunID, SolutionID: string(run.SolutionID), SolutionVersion: run.SolutionVersion, Environment: run.Environment,
		StartRecordDigest: digest(runRaw), PlanFingerprint: run.PlanFingerprint,
		PluginLockFingerprint: plan.PluginLockFingerprint, RuntimePolicyFingerprint: plan.RuntimePolicyFingerprint,
		EnvironmentPlanDigest: jsonDigest(plan), Command: cmd,
		InitialSalesRelease: initial, InitialObservedAt: run.Initial.ObservedAt,
		TargetRelease: TargetRelease, TargetSchemaFP: target.SchemaFingerprint, MigrationID: target.Migrations[0],
		PluginLockMatches:    durable.CheckPluginLock(run, plan.PluginLockFingerprint) == nil,
		RuntimePolicyMatches: durable.CheckRuntimePolicy(run, plan.RuntimePolicyFingerprint) == nil,
		Run:                  run, Plan: plan, Migration: steps, TargetDef: target, ReadyCount: len(cmds),
	}
	// The invocation and request identities are derived from the run, the
	// command and its idempotency key, so every later step can recompute and
	// compare them.
	s.InvocationID = "inv-" + strings.TrimPrefix(jsonDigest([]string{s.RunID, cmd.ActionID, cmd.IdempotencyKey, "attempt-1"}), "sha256:")[:24]
	s.RequestID = cmd.ActionID + "#attempt-1"
	return s, nil
}

// Identity is what every record of one deployment run must carry unchanged.
type Identity struct {
	RunID                    string `json:"runId"`
	StartRecordDigest        string `json:"startRecordDigest"`
	PlanFingerprint          string `json:"planFingerprint"`
	PluginLockFingerprint    string `json:"pluginLockFingerprint"`
	RuntimePolicyFingerprint string `json:"runtimePolicyFingerprint"`
	ActionID                 string `json:"actionId"`
	IdempotencyKey           string `json:"idempotencyKey"`
	InvocationID             string `json:"invocationId"`
	RequestID                string `json:"requestId"`
}

// Identity returns the identity step 1 hands to the later steps.
func (s Start) Identity() Identity {
	return Identity{RunID: s.RunID, StartRecordDigest: s.StartRecordDigest, PlanFingerprint: s.PlanFingerprint,
		PluginLockFingerprint: s.PluginLockFingerprint, RuntimePolicyFingerprint: s.RuntimePolicyFingerprint,
		ActionID: s.Command.ActionID, IdempotencyKey: s.Command.IdempotencyKey, InvocationID: s.InvocationID, RequestID: s.RequestID}
}

func (id Identity) complete() bool {
	return id.RunID != "" && id.StartRecordDigest != "" && id.PlanFingerprint != "" && id.PluginLockFingerprint != "" &&
		id.RuntimePolicyFingerprint != "" && id.ActionID != "" && id.IdempotencyKey != "" && id.InvocationID != "" && id.RequestID != ""
}
