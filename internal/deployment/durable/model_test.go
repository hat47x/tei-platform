package durable_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"tei/internal/core"
	dbrelease "tei/internal/database/release"
	"tei/internal/deployment"
	"tei/internal/deployment/durable"
	"tei/internal/deployment/executor"
	"tei/internal/releaseplan"
	"tei/internal/workspace"
)

func loadReference(t *testing.T) (releaseplan.Definition, releaseplan.Compiled, deployment.Snapshot, []dbrelease.Definition) {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	var state deployment.Snapshot
	f, err := os.Open(filepath.Join(root, "reference", "observed", "production.deployment-state.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&state); err != nil {
		t.Fatal(err)
	}
	var def releaseplan.Definition
	for _, x := range loaded.World.SolutionReleases {
		if x.ID == "order_rollout" && x.Version == "2026.8.27" {
			def = x
			break
		}
	}
	if def.ID == "" {
		t.Fatal("solution release not found")
	}
	compiled := releaseplan.CompileWithDeployment(def, loaded.World.ApplicationReleases, loaded.World.DatabaseReleases, state)
	if !compiled.Executable {
		t.Fatalf("compiled release plan blocked: %v", compiled.Findings)
	}
	return def, compiled, state, loaded.World.DatabaseReleases
}

func dbState(db core.CanonicalID, version string, provides []dbrelease.ProvidedContract) dbrelease.StateSnapshot {
	return dbrelease.StateSnapshot{Database: db, Source: dbrelease.StateFromRelease, Release: version, Provides: provides, DataReadiness: dbrelease.DataReady}
}

func replaceDB(s deployment.Snapshot, st dbrelease.StateSnapshot) deployment.Snapshot {
	out := s
	found := false
	for i := range out.Databases {
		if out.Databases[i].Database == st.Database {
			out.Databases[i] = st
			found = true
		}
	}
	if !found {
		out.Databases = append(out.Databases, st)
	}
	return out.Normalize()
}

func setRuntime(s deployment.Snapshot, states ...deployment.ApplicationRuntimeState) deployment.Snapshot {
	out := s
	out.ApplicationRuntime = append([]deployment.ApplicationRuntimeState(nil), states...)
	out.ActiveApplications = nil
	for _, st := range states {
		out.ActiveApplications = append(out.ActiveApplications, deployment.ApplicationReleaseRef{Application: st.Application, Version: st.Version})
	}
	return out.Normalize()
}

func TestDurableRunRequiresObservationAndHealthEvidence(t *testing.T) {
	def, compiled, initial, releases := loadReference(t)
	findDB := func(v string) dbrelease.Definition {
		for _, r := range releases {
			if r.Database == "sales_core" && r.Version == v {
				return r
			}
		}
		t.Fatalf("missing db release %s", v)
		return dbrelease.Definition{}
	}
	run := durable.Start("deploy-001", def, compiled, initial)
	if run.Status != durable.RunReady || len(run.ReadyActions) != 1 || run.ReadyActions[0] != "sales_expand/apply" {
		t.Fatalf("unexpected initial run: status=%s ready=%v waiting=%v", run.Status, run.ReadyActions, run.WaitingActions)
	}
	var key string
	for _, a := range run.Actions {
		if a.ID == "sales_expand/apply" {
			key = a.IdempotencyKey
		}
	}
	if key == "" {
		t.Fatal("command action must have stable idempotency key")
	}

	// Repeating the same observation does not complete the side effect or add duplicate events.
	beforeEvents := len(run.Events)
	beforeRevision := run.Revision
	run = durable.Reconcile(run, initial, nil)
	if run.Revision != beforeRevision {
		t.Fatalf("idempotent reconcile changed revision: %d -> %d", beforeRevision, run.Revision)
	}
	if len(run.Events) != beforeEvents || run.ReadyActions[0] != "sales_expand/apply" {
		t.Fatalf("reconcile must be idempotent: %+v", run)
	}

	r41 := findDB("4.1.0")
	afterExpand := replaceDB(initial, dbrelease.StateFromDefinition(r41))
	run = durable.Reconcile(run, afterExpand, nil)
	if run.Status != durable.RunReady || run.ReadyActions[0] != "order_app/start" {
		t.Fatalf("expected app start after expand: %+v", run)
	}

	old := deployment.ApplicationRuntimeState{Application: "order_entry", Version: "3.7.0", Lifecycle: deployment.LifecycleServing, TrafficBasisPoints: 10000, AcceptingWork: true}
	neu := deployment.ApplicationRuntimeState{Application: "order_entry", Version: "3.8.0", Lifecycle: deployment.LifecycleServing, TrafficBasisPoints: 0, AcceptingWork: true}
	started := setRuntime(afterExpand, old, neu)
	run = durable.Reconcile(run, started, nil)
	if run.ReadyActions[0] != "order_app/traffic-1000" {
		t.Fatalf("expected first canary shift: %+v", run)
	}

	old.TrafficBasisPoints, neu.TrafficBasisPoints = 9000, 1000
	ten := setRuntime(afterExpand, old, neu)
	run = durable.Reconcile(run, ten, nil)
	if run.Status != durable.RunWaitingEvidence || len(run.WaitingActions) != 1 || run.WaitingActions[0] != "order_app/health-1000" {
		t.Fatalf("expected health evidence gate: %+v", run)
	}
	health10 := durable.Evidence{ID: "health-10-pass", ActionID: "order_app/health-1000", Kind: durable.EvidenceRolloutHealth, Outcome: durable.EvidencePass, SourceRef: "monitor://order-entry/canary", Message: "error and latency budgets healthy"}
	run = durable.Reconcile(run, ten, []durable.Evidence{health10})
	if run.Status != durable.RunReady || run.ReadyActions[0] != "order_app/traffic-5000" {
		t.Fatalf("expected 50%% shift after health evidence: %+v", run)
	}

	// Reusing exactly the same evidence is idempotent.
	events := len(run.Events)
	revision := run.Revision
	run = durable.Reconcile(run, ten, []durable.Evidence{health10})
	if run.Revision != revision {
		t.Fatalf("duplicate evidence changed revision: %d -> %d", revision, run.Revision)
	}
	if len(run.Events) != events {
		t.Fatalf("duplicate evidence produced duplicate events: %d -> %d", events, len(run.Events))
	}

	old.TrafficBasisPoints, neu.TrafficBasisPoints = 5000, 5000
	fifty := setRuntime(afterExpand, old, neu)
	run = durable.Reconcile(run, fifty, nil)
	health50 := durable.Evidence{ID: "health-50-pass", ActionID: "order_app/health-5000", Kind: durable.EvidenceRolloutHealth, Outcome: durable.EvidencePass, SourceRef: "monitor://order-entry/canary"}
	run = durable.Reconcile(run, fifty, []durable.Evidence{health50})
	if run.ReadyActions[0] != "order_app/traffic-10000" {
		t.Fatalf("expected full cutover: %+v", run)
	}

	old.TrafficBasisPoints, neu.TrafficBasisPoints = 0, 10000
	full := setRuntime(afterExpand, old, neu)
	run = durable.Reconcile(run, full, nil)
	if run.ReadyActions[0] != "retire_order_37/stop-accepting" {
		t.Fatalf("expected stop accepting after full cutover: %+v", run)
	}

	old.AcceptingWork = false
	old.Lifecycle = deployment.LifecycleDraining
	draining := setRuntime(afterExpand, old, neu)
	run = durable.Reconcile(run, draining, nil)
	if run.Status != durable.RunWaitingObservation || run.WaitingActions[0] != "retire_order_37/drain-complete" {
		t.Fatalf("expected drain observation gate: %+v", run)
	}

	old.DrainComplete = true
	old.Lifecycle = deployment.LifecycleQuiescent
	quiescent := setRuntime(afterExpand, old, neu)
	run = durable.Reconcile(run, quiescent, nil)
	if run.Status != durable.RunReady || run.ReadyActions[0] != "retire_order_37/retire" {
		t.Fatalf("expected retire command: %+v", run)
	}

	retired := setRuntime(afterExpand, neu)
	run = durable.Reconcile(run, retired, nil)
	if run.ReadyActions[0] != "sales_contract/apply" {
		t.Fatalf("expected contract release only after retire observation: %+v", run)
	}

	r42 := findDB("4.2.0")
	final := replaceDB(retired, dbrelease.StateFromDefinition(r42))
	run = durable.Reconcile(run, final, nil)
	if run.Status != durable.RunSucceeded {
		t.Fatalf("expected succeeded run: %+v", run)
	}
	for _, a := range run.Actions {
		if a.Status != durable.ActionCompleted {
			t.Fatalf("action not completed: %+v", a)
		}
	}
}

func TestFailedHealthEvidenceBlocksRun(t *testing.T) {
	def, compiled, initial, releases := loadReference(t)
	var r41 dbrelease.Definition
	for _, r := range releases {
		if r.Database == "sales_core" && r.Version == "4.1.0" {
			r41 = r
		}
	}
	run := durable.Start("deploy-fail", def, compiled, initial)
	expanded := replaceDB(initial, dbrelease.StateFromDefinition(r41))
	run = durable.Reconcile(run, expanded, nil)
	old := deployment.ApplicationRuntimeState{Application: "order_entry", Version: "3.7.0", Lifecycle: deployment.LifecycleServing, TrafficBasisPoints: 9000, AcceptingWork: true}
	neu := deployment.ApplicationRuntimeState{Application: "order_entry", Version: "3.8.0", Lifecycle: deployment.LifecycleServing, TrafficBasisPoints: 1000, AcceptingWork: true}
	run = durable.Reconcile(run, setRuntime(expanded, old, neu), nil)
	fail := durable.Evidence{ID: "health-bad", ActionID: "order_app/health-1000", Kind: durable.EvidenceRolloutHealth, Outcome: durable.EvidenceFail, Message: "error budget exceeded"}
	run = durable.Reconcile(run, setRuntime(expanded, old, neu), []durable.Evidence{fail})
	if run.Status != durable.RunNeedsRecovery {
		t.Fatalf("failed evidence must require recovery: %+v", run)
	}
}

func TestPlanFingerprintProtectsResume(t *testing.T) {
	def, compiled, initial, _ := loadReference(t)
	run := durable.Start("deploy-plan", def, compiled, initial)
	changed := def.Normalize()
	for i := range changed.Steps {
		if changed.Steps[i].ID == "order_app" {
			changed.Steps[i].TrafficSteps = []int{2000, 10000}
		}
	}
	changedCompiled := compiled
	if err := durable.CheckPlan(run, changed, changedCompiled); err == nil {
		t.Fatal("expected plan fingerprint mismatch")
	}
	if err := durable.CheckPlan(run, def, compiled); err != nil {
		t.Fatal(err)
	}
}

func TestEvidenceCannotBePreloadedBeforeGateIsActive(t *testing.T) {
	def, compiled, initial, _ := loadReference(t)
	run := durable.Start("deploy-early-evidence", def, compiled, initial)
	e := durable.Evidence{ID: "too-early", ActionID: "order_app/health-1000", Kind: durable.EvidenceRolloutHealth, Outcome: durable.EvidencePass}
	run = durable.Reconcile(run, initial, []durable.Evidence{e})
	if run.Status != durable.RunBlocked {
		t.Fatalf("preloaded evidence must be rejected: %+v", run)
	}
}

func TestRunStructureTamperingIsRejected(t *testing.T) {
	def, compiled, initial, _ := loadReference(t)
	run := durable.Start("deploy-tamper", def, compiled, initial)
	if err := durable.CheckRun(run, def, compiled); err != nil {
		t.Fatal(err)
	}
	for i := range run.Actions {
		if run.Actions[i].ID == "sales_expand/apply" {
			run.Actions[i].IdempotencyKey = "forged"
		}
	}
	if err := durable.CheckRun(run, def, compiled); err == nil {
		t.Fatal("expected modified immutable action structure to be rejected")
	}
}

func TestExecutorAcknowledgementDoesNotCompleteCommand(t *testing.T) {
	def, compiled, initial, _ := loadReference(t)
	run := durable.Start("deploy-executor", def, compiled, initial)
	cmds := durable.ReadyCommands(run)
	if len(cmds) != 1 || cmds[0].ActionID != "sales_expand/apply" {
		t.Fatalf("unexpected ready commands: %+v", cmds)
	}
	result := executor.Result{
		ExecutorID: "reference-executor", AttemptID: "attempt-1",
		ActionID: cmds[0].ActionID, IdempotencyKey: cmds[0].IdempotencyKey,
		Outcome: executor.OutcomeAccepted, ExternalRef: "job://1",
	}
	run = durable.RecordExecutorResult(run, result)
	if run.Status != durable.RunWaitingObservation {
		t.Fatalf("accepted executor result must wait for observation: %+v", run)
	}
	for _, a := range run.Actions {
		if a.ID == result.ActionID && a.Status == durable.ActionCompleted {
			t.Fatal("executor acceptance must not complete action")
		}
	}
	if len(durable.ReadyCommands(run)) != 0 {
		t.Fatalf("accepted command must not be redispatched before reconciliation: %+v", durable.ReadyCommands(run))
	}
}

func TestRetryUsesSameIdempotencyKey(t *testing.T) {
	def, compiled, initial, _ := loadReference(t)
	run := durable.Start("deploy-retry", def, compiled, initial)
	cmd := durable.ReadyCommands(run)[0]
	run = durable.RecordExecutorResult(run, executor.Result{
		ExecutorID: "reference-executor", AttemptID: "attempt-r1",
		ActionID: cmd.ActionID, IdempotencyKey: cmd.IdempotencyKey,
		Outcome: executor.OutcomeRejected, Retryable: true, Message: "temporary API outage",
	})
	if run.Status != durable.RunPaused {
		t.Fatalf("retryable reject must pause for explicit retry: %+v", run)
	}
	run = durable.Retry(run, cmd.ActionID, "operator approved retry")
	if run.Status != durable.RunReady {
		t.Fatalf("retry must make command ready: %+v", run)
	}
	cmds := durable.ReadyCommands(run)
	if len(cmds) != 1 || cmds[0].IdempotencyKey != cmd.IdempotencyKey {
		t.Fatalf("retry must preserve idempotency key: before=%s after=%+v", cmd.IdempotencyKey, cmds)
	}
}

func TestNonRetryableFailureRequiresExplicitRecovery(t *testing.T) {
	def, compiled, initial, _ := loadReference(t)
	run := durable.Start("deploy-recovery", def, compiled, initial)
	cmd := durable.ReadyCommands(run)[0]
	run = durable.RecordExecutorResult(run, executor.Result{
		ExecutorID: "reference-executor", AttemptID: "attempt-fatal",
		ActionID: cmd.ActionID, IdempotencyKey: cmd.IdempotencyKey,
		Outcome: executor.OutcomeRejected, Retryable: false, Message: "migration rejected",
	})
	if run.Status != durable.RunNeedsRecovery {
		t.Fatalf("non-retryable failure must require recovery: %+v", run)
	}
	run = durable.PlanRecovery(run, durable.RecoveryDecision{
		ID: "forward-fix-1", Strategy: durable.RecoveryForwardFix, ActionID: cmd.ActionID,
		TargetSolutionID: "order_rollout_fix", TargetSolutionVersion: "2026.8.28",
		Reason: "apply corrective database release",
	})
	if run.Status != durable.RunRecoveryPlanned || run.Recovery == nil {
		t.Fatalf("expected explicit recovery plan: %+v", run)
	}
}

func TestPauseResumeAndAbortDoNotImplyRollback(t *testing.T) {
	def, compiled, initial, _ := loadReference(t)
	run := durable.Start("deploy-control", def, compiled, initial)
	run = durable.Pause(run, "change window paused")
	if run.Status != durable.RunPaused || len(run.ReadyActions) != 0 {
		t.Fatalf("pause must suppress commands: %+v", run)
	}
	run = durable.Resume(run)
	if run.Status != durable.RunReady || len(run.ReadyActions) == 0 {
		t.Fatalf("resume must restore readiness: %+v", run)
	}
	run = durable.Abort(run, "operator aborted")
	if run.Status != durable.RunAborted || run.Recovery != nil {
		t.Fatalf("abort is terminal but must not invent rollback: %+v", run)
	}
}

func TestRecoveryStartsDistinctLinkedDurableRun(t *testing.T) {
	def, compiled, initial, _ := loadReference(t)
	parent := durable.Start("deploy-parent", def, compiled, initial)
	cmd := durable.ReadyCommands(parent)[0]
	parent = durable.RecordExecutorResult(parent, executor.Result{
		ExecutorID: "migration.atlas", AttemptID: "fatal-1", ActionID: cmd.ActionID,
		IdempotencyKey: cmd.IdempotencyKey, Outcome: executor.OutcomeRejected, Retryable: false,
	})
	parent = durable.PlanRecovery(parent, durable.RecoveryDecision{
		ID: "fix-1", Strategy: durable.RecoveryForwardFix, ActionID: cmd.ActionID,
		TargetSolutionID: "order_rollout_fix", TargetSolutionVersion: "2026.8.28",
	})

	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	var fix releaseplan.Definition
	for _, x := range loaded.World.SolutionReleases {
		if x.ID == "order_rollout_fix" && x.Version == "2026.8.28" {
			fix = x
			break
		}
	}
	if fix.ID == "" {
		t.Fatal("recovery solution release missing")
	}
	fixCompiled := releaseplan.CompileWithDeployment(fix, loaded.World.ApplicationReleases, loaded.World.DatabaseReleases, parent.LastObserved)
	updated, child, err := durable.StartRecovery(parent, "deploy-recovery-1", fix, fixCompiled, parent.LastObserved)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Recovery == nil || updated.Recovery.RecoveryRunID != child.RunID {
		t.Fatalf("parent not linked: %+v", updated.Recovery)
	}
	if child.RecoveryOf == nil || child.RecoveryOf.ParentRunID != parent.RunID || child.RecoveryOf.DecisionID != "fix-1" {
		t.Fatalf("child not linked: %+v", child.RecoveryOf)
	}
	if child.SolutionID != "order_rollout_fix" || child.Status != durable.RunReady {
		t.Fatalf("unexpected recovery child: %+v", child)
	}
}

func TestBoundRunPinsPluginLockFingerprint(t *testing.T) {
	def, compiled, initial, _ := loadReference(t)
	run := durable.StartWithPluginLock("deploy-bound", def, compiled, initial, "lock-a")
	if run.PluginLockFingerprint != "lock-a" || run.FormatVersion != 4 {
		t.Fatalf("run: %#v", run)
	}
	if err := durable.CheckPluginLock(run, "lock-a"); err != nil {
		t.Fatal(err)
	}
	if err := durable.CheckPluginLock(run, "lock-b"); err == nil {
		t.Fatal("expected plugin lock mismatch")
	}
}

func TestResourceBoundRunPinsRuntimePolicyFingerprint(t *testing.T) {
	def, compiled, initial, _ := loadReference(t)
	run := durable.StartWithRuntimePolicy("deploy-policy", def, compiled, initial, "lock-a", "policy-a")
	if run.PluginLockFingerprint != "lock-a" || run.RuntimePolicyFingerprint != "policy-a" || run.FormatVersion != 5 {
		t.Fatalf("run: %#v", run)
	}
	if err := durable.CheckRuntimePolicy(run, "policy-a"); err != nil {
		t.Fatal(err)
	}
	if err := durable.CheckRuntimePolicy(run, "policy-b"); err == nil {
		t.Fatal("expected runtime policy mismatch")
	}
}
