package durable

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	apprelease "tei/internal/application/release"
	"tei/internal/core"
	dbrelease "tei/internal/database/release"
	"tei/internal/deployment"
	"tei/internal/deployment/executor"
	"tei/internal/releaseplan"
)

type ActionKind = deployment.ActionKind

const (
	ActionDatabaseRelease   = deployment.ActionDatabaseRelease
	ActionDatabaseGate      = deployment.ActionDatabaseGate
	ActionApplicationStart  = deployment.ActionApplicationStart
	ActionTrafficShift      = deployment.ActionTrafficShift
	ActionRolloutHealthGate = deployment.ActionRolloutHealthGate
	ActionStopAcceptingWork = deployment.ActionStopAcceptingWork
	ActionDrainCompleteGate = deployment.ActionDrainCompleteGate
	ActionApplicationRetire = deployment.ActionApplicationRetire
)

type ActionMode string

const (
	ModeCommand     ActionMode = "command"
	ModeObservation ActionMode = "observation"
	ModeEvidence    ActionMode = "evidence"
)

type Action struct {
	ID                 string           `json:"id"`
	Step               core.LocalName   `json:"step"`
	Kind               ActionKind       `json:"kind"`
	Mode               ActionMode       `json:"mode"`
	Prerequisites      []string         `json:"prerequisites,omitempty"`
	IdempotencyKey     string           `json:"idempotencyKey,omitempty"`
	Application        core.CanonicalID `json:"application,omitempty"`
	Version            string           `json:"version,omitempty"`
	Database           core.CanonicalID `json:"database,omitempty"`
	Contract           core.CanonicalID `json:"contract,omitempty"`
	CompatibleMajor    int              `json:"compatibleMajor,omitempty"`
	TrafficBasisPoints int              `json:"trafficBasisPoints,omitempty"`
	EvidenceKind       EvidenceKind     `json:"evidenceKind,omitempty"`
}

type ActionStatus string

const (
	ActionPending   ActionStatus = "pending"
	ActionCompleted ActionStatus = "completed"
)

type ActionRecord struct {
	Action
	Status           ActionStatus `json:"status"`
	CompletionSource string       `json:"completionSource,omitempty"`
	EvidenceIDs      []string     `json:"evidenceIds,omitempty"`
}

type EvidenceKind = deployment.EvidenceKind

const EvidenceRolloutHealth = deployment.EvidenceRolloutHealth

type EvidenceOutcome string

const (
	EvidencePass EvidenceOutcome = "pass"
	EvidenceFail EvidenceOutcome = "fail"
)

type Evidence struct {
	ID         string          `json:"id"`
	ActionID   string          `json:"actionId"`
	Kind       EvidenceKind    `json:"kind"`
	Outcome    EvidenceOutcome `json:"outcome"`
	ObservedAt string          `json:"observedAt,omitempty"`
	SourceRef  string          `json:"sourceRef,omitempty"`
	Message    string          `json:"message,omitempty"`
}

type RunStatus string

const (
	RunReady              RunStatus = "ready"
	RunWaitingEvidence    RunStatus = "waiting-evidence"
	RunWaitingObservation RunStatus = "waiting-observation"
	RunPaused             RunStatus = "paused"
	RunNeedsRecovery      RunStatus = "needs-recovery"
	RunRecoveryPlanned    RunStatus = "recovery-planned"
	RunSucceeded          RunStatus = "succeeded"
	RunAborted            RunStatus = "aborted"
	RunBlocked            RunStatus = "blocked"
)

type EventKind string

const (
	EventActionCompleted   EventKind = "action-completed"
	EventEvidenceAccepted  EventKind = "evidence-accepted"
	EventEvidenceRejected  EventKind = "evidence-rejected"
	EventCommandAttempted  EventKind = "command-attempted"
	EventRetryAuthorized   EventKind = "retry-authorized"
	EventRunPaused         EventKind = "run-paused"
	EventRunResumed        EventKind = "run-resumed"
	EventRunAborted        EventKind = "run-aborted"
	EventRecoveryPlanned   EventKind = "recovery-planned"
	EventRecoveryRunLinked EventKind = "recovery-run-linked"
)

type Event struct {
	Sequence   int       `json:"sequence"`
	Kind       EventKind `json:"kind"`
	ActionID   string    `json:"actionId,omitempty"`
	EvidenceID string    `json:"evidenceId,omitempty"`
	AttemptID  string    `json:"attemptId,omitempty"`
	Message    string    `json:"message,omitempty"`
}

type RetryAuthorization struct {
	ActionID       string `json:"actionId"`
	AfterAttemptID string `json:"afterAttemptId"`
	Reason         string `json:"reason,omitempty"`
}

type RecoveryStrategy string

const (
	RecoveryRetry      RecoveryStrategy = "retry"
	RecoveryForwardFix RecoveryStrategy = "forward-fix"
	RecoveryRollback   RecoveryStrategy = "rollback"
	RecoveryManual     RecoveryStrategy = "manual"
)

type RecoveryDecision struct {
	ID                    string           `json:"id"`
	Strategy              RecoveryStrategy `json:"strategy"`
	ActionID              string           `json:"actionId,omitempty"`
	TargetSolutionID      core.CanonicalID `json:"targetSolutionId,omitempty"`
	TargetSolutionVersion string           `json:"targetSolutionVersion,omitempty"`
	Reason                string           `json:"reason,omitempty"`
	RecoveryRunID         string           `json:"recoveryRunId,omitempty"`
}

type RecoveryOrigin struct {
	ParentRunID string           `json:"parentRunId"`
	DecisionID  string           `json:"decisionId"`
	Strategy    RecoveryStrategy `json:"strategy"`
}

type Run struct {
	FormatVersion            int                  `json:"formatVersion"`
	Revision                 int                  `json:"revision"`
	RunID                    string               `json:"runId"`
	SolutionID               core.CanonicalID     `json:"solutionId"`
	SolutionVersion          string               `json:"solutionVersion"`
	Environment              core.CanonicalID     `json:"environment"`
	PlanFingerprint          string               `json:"planFingerprint"`
	PluginLockFingerprint    string               `json:"pluginLockFingerprint,omitempty"`
	RuntimePolicyFingerprint string               `json:"runtimePolicyFingerprint,omitempty"`
	Status                   RunStatus            `json:"status"`
	Initial                  deployment.Snapshot  `json:"initial"`
	LastObserved             deployment.Snapshot  `json:"lastObserved"`
	Actions                  []ActionRecord       `json:"actions"`
	ReadyActions             []string             `json:"readyActions,omitempty"`
	WaitingActions           []string             `json:"waitingActions,omitempty"`
	Evidence                 []Evidence           `json:"evidence,omitempty"`
	Attempts                 []executor.Result    `json:"attempts,omitempty"`
	Retries                  []RetryAuthorization `json:"retries,omitempty"`
	Recovery                 *RecoveryDecision    `json:"recovery,omitempty"`
	RecoveryOf               *RecoveryOrigin      `json:"recoveryOf,omitempty"`
	PauseReason              string               `json:"pauseReason,omitempty"`
	Events                   []Event              `json:"events,omitempty"`
	Findings                 []string             `json:"findings,omitempty"`
}

func PlanFingerprint(def releaseplan.Definition, compiled releaseplan.Compiled) string {
	payload := struct {
		Definition releaseplan.Definition     `json:"definition"`
		Edges      []releaseplan.CompiledEdge `json:"edges"`
	}{Definition: def.Normalize(), Edges: append([]releaseplan.CompiledEdge(nil), compiled.Edges...)}
	sort.Slice(payload.Edges, func(i, j int) bool {
		if payload.Edges[i].From == payload.Edges[j].From {
			if payload.Edges[i].To == payload.Edges[j].To {
				if payload.Edges[i].Relation == payload.Edges[j].Relation {
					return payload.Edges[i].Contract < payload.Edges[j].Contract
				}
				return payload.Edges[i].Relation < payload.Edges[j].Relation
			}
			return payload.Edges[i].To < payload.Edges[j].To
		}
		return payload.Edges[i].From < payload.Edges[j].From
	})
	raw, _ := json.Marshal(payload)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func start(runID string, def releaseplan.Definition, compiled releaseplan.Compiled, initial deployment.Snapshot, formatVersion int, pluginLockFingerprint, runtimePolicyFingerprint string) Run {
	r := Run{
		FormatVersion: formatVersion,
		RunID:         runID, SolutionID: def.ID, SolutionVersion: def.Version, Environment: initial.Environment,
		PlanFingerprint: PlanFingerprint(def, compiled), PluginLockFingerprint: pluginLockFingerprint, RuntimePolicyFingerprint: runtimePolicyFingerprint,
		Initial: initial.Normalize(), LastObserved: initial.Normalize(), Status: RunReady,
	}
	if runID == "" {
		r.Status = RunBlocked
		r.Findings = append(r.Findings, "run id is required")
		return r
	}
	if formatVersion >= 4 && pluginLockFingerprint == "" {
		r.Status = RunBlocked
		r.Findings = append(r.Findings, "plugin lock fingerprint is required for a bound durable deployment run")
		return r
	}
	if formatVersion >= 5 && runtimePolicyFingerprint == "" {
		r.Status = RunBlocked
		r.Findings = append(r.Findings, "runtime policy fingerprint is required for a resource-bound durable deployment run")
		return r
	}
	if !compiled.Executable {
		r.Status = RunBlocked
		r.Findings = append(r.Findings, compiled.Findings...)
		return r
	}
	actions, err := buildActions(def, compiled, initial.Environment, r.PlanFingerprint)
	if err != nil {
		r.Status = RunBlocked
		r.Findings = append(r.Findings, err.Error())
		return r
	}
	for _, a := range actions {
		r.Actions = append(r.Actions, ActionRecord{Action: a, Status: ActionPending})
	}
	return Reconcile(r, initial, nil)
}

func Start(runID string, def releaseplan.Definition, compiled releaseplan.Compiled, initial deployment.Snapshot) Run {
	return start(runID, def, compiled, initial, 3, "", "")
}

// StartWithPluginLock creates a durable run whose runtime implementation set is
// pinned to one environment-specific Plugin Lock fingerprint. This is the
// production-oriented entry point; Start remains for parser/model unit tests
// and legacy artifacts.
func StartWithPluginLock(runID string, def releaseplan.Definition, compiled releaseplan.Compiled, initial deployment.Snapshot, pluginLockFingerprint string) Run {
	return start(runID, def, compiled, initial, 4, pluginLockFingerprint, "")
}

// StartWithRuntimePolicy additionally pins the Environment runtime policy: the
// selected Plugin Lock plus capability permissions and logical resource bindings.
// This prevents an in-flight run from silently resuming after a privilege or
// resource-target change even when the Plugin versions themselves are unchanged.
func StartWithRuntimePolicy(runID string, def releaseplan.Definition, compiled releaseplan.Compiled, initial deployment.Snapshot, pluginLockFingerprint, runtimePolicyFingerprint string) Run {
	return start(runID, def, compiled, initial, 5, pluginLockFingerprint, runtimePolicyFingerprint)
}

// CheckPluginLock prevents a durable run from being resumed under a different
// plugin implementation set. A recovery run is separate and may explicitly
// bind a different lock when it is created.
func CheckPluginLock(run Run, pluginLockFingerprint string) error {
	if run.PluginLockFingerprint == "" {
		return fmt.Errorf("deployment run %s is not bound to a plugin lock", run.RunID)
	}
	if pluginLockFingerprint == "" {
		return fmt.Errorf("plugin lock fingerprint is required to resume deployment run %s", run.RunID)
	}
	if run.PluginLockFingerprint != pluginLockFingerprint {
		return fmt.Errorf("deployment run plugin lock changed: run fingerprint %s, current fingerprint %s", run.PluginLockFingerprint, pluginLockFingerprint)
	}
	return nil
}

// CheckRuntimePolicy prevents a resource-bound run from being resumed after
// permission grants or logical resource bindings changed.
func CheckRuntimePolicy(run Run, runtimePolicyFingerprint string) error {
	if run.FormatVersion < 5 {
		return nil
	}
	if run.RuntimePolicyFingerprint == "" {
		return fmt.Errorf("deployment run %s is not bound to a runtime policy", run.RunID)
	}
	if runtimePolicyFingerprint == "" {
		return fmt.Errorf("runtime policy fingerprint is required to resume deployment run %s", run.RunID)
	}
	if run.RuntimePolicyFingerprint != runtimePolicyFingerprint {
		return fmt.Errorf("deployment run runtime policy changed: run fingerprint %s, current fingerprint %s", run.RuntimePolicyFingerprint, runtimePolicyFingerprint)
	}
	return nil
}

func CheckPlan(run Run, def releaseplan.Definition, compiled releaseplan.Compiled) error {
	got := PlanFingerprint(def, compiled)
	if run.PlanFingerprint != got {
		return fmt.Errorf("deployment plan changed: run fingerprint %s, current fingerprint %s", run.PlanFingerprint, got)
	}
	if run.SolutionID != def.ID || run.SolutionVersion != def.Version {
		return fmt.Errorf("deployment run targets %s@%s, current plan is %s@%s", run.SolutionID, run.SolutionVersion, def.ID, def.Version)
	}
	return nil
}

// CheckRun validates the immutable structure of a serialized run against the
// current compiled plan. Mutable completion/evidence history still belongs in a
// trusted durable store; this structural check prevents accidental editing of
// action identities, dependencies, or idempotency keys during resume.
func CheckRun(run Run, def releaseplan.Definition, compiled releaseplan.Compiled) error {
	if run.FormatVersion != 1 && run.FormatVersion != 2 && run.FormatVersion != 3 && run.FormatVersion != 4 && run.FormatVersion != 5 {
		return fmt.Errorf("unsupported durable deployment run format version %d", run.FormatVersion)
	}
	if err := CheckPlan(run, def, compiled); err != nil {
		return err
	}
	expected, err := buildActions(def, compiled, run.Environment, run.PlanFingerprint)
	if err != nil {
		return err
	}
	if len(expected) != len(run.Actions) {
		return fmt.Errorf("durable run action structure changed: expected %d actions, found %d", len(expected), len(run.Actions))
	}
	byID := map[string]Action{}
	for _, a := range expected {
		byID[a.ID] = a
	}
	for _, rec := range run.Actions {
		exp, ok := byID[rec.ID]
		if !ok {
			return fmt.Errorf("durable run contains unknown action %s", rec.ID)
		}
		actual := rec.Action
		a, _ := json.Marshal(exp)
		b, _ := json.Marshal(actual)
		if string(a) != string(b) {
			return fmt.Errorf("durable run action %s immutable structure was modified", rec.ID)
		}
	}
	return nil
}

// Reconcile advances a durable run only from observed state or explicit gate
// evidence. It never claims that a command succeeded merely because TEI emitted
// an intent. Repeating reconciliation with the same inputs is idempotent.
func Reconcile(run Run, observed deployment.Snapshot, incoming []Evidence) Run {
	before := runDigest(run)
	baseRevision := run.Revision
	if run.Status == RunBlocked || run.Status == RunSucceeded || run.Status == RunAborted || run.Status == RunRecoveryPlanned {
		return finalizeRevision(run, before, baseRevision)
	}
	wasPaused := run.Status == RunPaused
	if run.FormatVersion == 1 || run.FormatVersion == 2 {
		run.FormatVersion = 3
	}
	observed = observed.Normalize()
	if observed.Environment != run.Environment {
		run.Status = RunBlocked
		run.Findings = appendUnique(run.Findings, fmt.Sprintf("observed environment %s does not match run environment %s", observed.Environment, run.Environment))
		return finalizeRevision(run, before, baseRevision)
	}
	run.LastObserved = observed
	// First reconcile already-recorded evidence and observed state. This lets a
	// traffic observation make its following health gate ready in the same
	// resume operation, while preventing evidence from being preloaded before
	// the state it is supposed to evaluate actually exists.
	if failed := completeSatisfiedActions(&run, observed); failed != "" {
		run.Status = RunNeedsRecovery
		run.Findings = appendUnique(run.Findings, failed)
		return finalizeRevision(run, before, baseRevision)
	}
	if err := mergeEvidence(&run, incoming); err != nil {
		run.Status = RunBlocked
		run.Findings = appendUnique(run.Findings, err.Error())
		return finalizeRevision(run, before, baseRevision)
	}
	if failed := completeSatisfiedActions(&run, observed); failed != "" {
		run.Status = RunNeedsRecovery
		run.Findings = appendUnique(run.Findings, failed)
		return finalizeRevision(run, before, baseRevision)
	}

	run.ReadyActions = nil
	run.WaitingActions = nil
	allComplete := true
	hasEvidenceWait := false
	hasObservationWait := false
	for _, rec := range run.Actions {
		if rec.Status == ActionCompleted {
			continue
		}
		allComplete = false
		if !prerequisitesComplete(run.Actions, rec.Prerequisites) {
			continue
		}
		switch rec.Mode {
		case ModeCommand:
			if waitingForCommandObservation(run, rec.ID) {
				hasObservationWait = true
				run.WaitingActions = append(run.WaitingActions, rec.ID)
			} else {
				run.ReadyActions = append(run.ReadyActions, rec.ID)
			}
		case ModeEvidence:
			hasEvidenceWait = true
			run.WaitingActions = append(run.WaitingActions, rec.ID)
		case ModeObservation:
			hasObservationWait = true
			run.WaitingActions = append(run.WaitingActions, rec.ID)
		}
	}
	sort.Strings(run.ReadyActions)
	sort.Strings(run.WaitingActions)
	if allComplete {
		run.Status = RunSucceeded
	} else if run.Status == RunNeedsRecovery {
		run.ReadyActions = nil
		run.WaitingActions = nil
	} else if wasPaused {
		run.Status = RunPaused
		run.ReadyActions = nil
		run.WaitingActions = nil
	} else if len(run.ReadyActions) > 0 {
		run.Status = RunReady
	} else if hasEvidenceWait {
		run.Status = RunWaitingEvidence
	} else if hasObservationWait {
		run.Status = RunWaitingObservation
	} else {
		run.Status = RunBlocked
		run.Findings = appendUnique(run.Findings, "no executable or reconcilable action remains")
	}
	return finalizeRevision(run, before, baseRevision)
}

func runDigest(run Run) string {
	copy := run
	copy.Revision = 0
	raw, _ := json.Marshal(copy)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func finalizeRevision(run Run, before string, base int) Run {
	if runDigest(run) != before {
		run.Revision = base + 1
	} else {
		run.Revision = base
	}
	return run
}

func completeSatisfiedActions(run *Run, observed deployment.Snapshot) string {
	changed := true
	for changed {
		changed = false
		for i := range run.Actions {
			rec := &run.Actions[i]
			if rec.Status == ActionCompleted || !prerequisitesComplete(run.Actions, rec.Prerequisites) {
				continue
			}
			completed, source, evidenceIDs, failed := actionSatisfied(rec.Action, observed, run.Evidence)
			if failed != "" {
				return failed
			}
			if completed {
				rec.Status = ActionCompleted
				rec.CompletionSource = source
				rec.EvidenceIDs = append([]string(nil), evidenceIDs...)
				emitOnce(run, Event{Kind: EventActionCompleted, ActionID: rec.ID, Message: source})
				changed = true
			}
		}
	}
	return ""
}

func buildActions(def releaseplan.Definition, compiled releaseplan.Compiled, env core.CanonicalID, fingerprint string) ([]Action, error) {
	first := map[core.LocalName]string{}
	last := map[core.LocalName]string{}
	var out []Action
	appendAction := func(a Action) {
		if a.Mode == ModeCommand {
			a.IdempotencyKey = fmt.Sprintf("tei:deploy:%s:%s:%s", env, fingerprint[:16], a.ID)
		}
		out = append(out, a)
		if first[a.Step] == "" {
			first[a.Step] = a.ID
		}
		last[a.Step] = a.ID
	}
	for _, step := range def.Steps {
		switch step.Kind {
		case releaseplan.StepDatabaseRelease:
			appendAction(Action{ID: string(step.ID) + "/apply", Step: step.ID, Kind: ActionDatabaseRelease, Mode: ModeCommand, Database: step.Database, Version: step.Version})
		case releaseplan.StepDatabaseGate:
			appendAction(Action{ID: string(step.ID) + "/gate", Step: step.ID, Kind: ActionDatabaseGate, Mode: ModeObservation, Database: step.Database, Contract: step.Contract, CompatibleMajor: step.CompatibleMajor})
		case releaseplan.StepApplicationRelease:
			startID := string(step.ID) + "/start"
			appendAction(Action{ID: startID, Step: step.ID, Kind: ActionApplicationStart, Mode: ModeCommand, Application: step.Application, Version: step.Version})
			prev := startID
			if step.RolloutStrategy == releaseplan.RolloutBlueGreen {
				gid := string(step.ID) + "/health-pre-cutover"
				appendAction(Action{ID: gid, Step: step.ID, Kind: ActionRolloutHealthGate, Mode: ModeEvidence, Prerequisites: []string{prev}, Application: step.Application, Version: step.Version, EvidenceKind: EvidenceRolloutHealth})
				prev = gid
			}
			traffic := step.EffectiveTrafficSteps()
			for i, bps := range traffic {
				aid := fmt.Sprintf("%s/traffic-%d", step.ID, bps)
				appendAction(Action{ID: aid, Step: step.ID, Kind: ActionTrafficShift, Mode: ModeCommand, Prerequisites: []string{prev}, Application: step.Application, Version: step.Version, TrafficBasisPoints: bps})
				prev = aid
				if i < len(traffic)-1 {
					gid := fmt.Sprintf("%s/health-%d", step.ID, bps)
					appendAction(Action{ID: gid, Step: step.ID, Kind: ActionRolloutHealthGate, Mode: ModeEvidence, Prerequisites: []string{prev}, Application: step.Application, Version: step.Version, TrafficBasisPoints: bps, EvidenceKind: EvidenceRolloutHealth})
					prev = gid
				}
			}
		case releaseplan.StepApplicationRetire:
			stopID := string(step.ID) + "/stop-accepting"
			appendAction(Action{ID: stopID, Step: step.ID, Kind: ActionStopAcceptingWork, Mode: ModeCommand, Application: step.Application, Version: step.Version})
			drainID := string(step.ID) + "/drain-complete"
			appendAction(Action{ID: drainID, Step: step.ID, Kind: ActionDrainCompleteGate, Mode: ModeObservation, Prerequisites: []string{stopID}, Application: step.Application, Version: step.Version})
			retireID := string(step.ID) + "/retire"
			appendAction(Action{ID: retireID, Step: step.ID, Kind: ActionApplicationRetire, Mode: ModeCommand, Prerequisites: []string{drainID}, Application: step.Application, Version: step.Version})
		default:
			return nil, fmt.Errorf("unsupported release step kind %s", step.Kind)
		}
	}
	byID := map[string]int{}
	for i := range out {
		byID[out[i].ID] = i
	}
	for _, edge := range compiled.Edges {
		from, to := last[edge.From], first[edge.To]
		if from == "" || to == "" {
			continue
		}
		idx := byID[to]
		out[idx].Prerequisites = appendUniqueString(out[idx].Prerequisites, from)
	}
	for i := range out {
		sort.Strings(out[i].Prerequisites)
	}
	return out, nil
}

func actionSatisfied(a Action, observed deployment.Snapshot, evidence []Evidence) (bool, string, []string, string) {
	switch a.Kind {
	case ActionDatabaseRelease:
		st, ok := observed.DatabaseState(a.Database)
		return ok && st.Release == a.Version, "observed database release state", nil, ""
	case ActionDatabaseGate:
		st, ok := observed.DatabaseState(a.Database)
		if !ok || st.DataReadiness != dbrelease.DataReady {
			return false, "", nil, ""
		}
		for _, c := range st.Provides {
			if c.ID == a.Contract && c.Version.Major == a.CompatibleMajor {
				return true, "observed database contract state", nil, ""
			}
		}
		return false, "", nil, ""
	case ActionApplicationStart:
		if !observed.IsActive(a.Application, a.Version) {
			return false, "", nil, ""
		}
		s, ok := runtimeState(observed, a.Application, a.Version)
		return ok && (s.Lifecycle == deployment.LifecycleStarting || s.Lifecycle == deployment.LifecycleServing), "observed application runtime", nil, ""
	case ActionTrafficShift:
		s, ok := runtimeState(observed, a.Application, a.Version)
		return ok && s.TrafficBasisPoints == a.TrafficBasisPoints, "observed traffic allocation", nil, ""
	case ActionRolloutHealthGate:
		var ids []string
		for _, e := range evidence {
			if e.ActionID != a.ID || e.Kind != a.EvidenceKind {
				continue
			}
			if e.Outcome == EvidenceFail {
				return false, "", nil, fmt.Sprintf("evidence %s failed gate %s: %s", e.ID, a.ID, e.Message)
			}
			if e.Outcome == EvidencePass {
				ids = append(ids, e.ID)
			}
		}
		sort.Strings(ids)
		return len(ids) > 0, "explicit rollout health evidence", ids, ""
	case ActionStopAcceptingWork:
		s, ok := runtimeState(observed, a.Application, a.Version)
		return ok && !s.AcceptingWork, "observed work-admission state", nil, ""
	case ActionDrainCompleteGate:
		s, ok := runtimeState(observed, a.Application, a.Version)
		return ok && s.DrainComplete && s.Lifecycle == deployment.LifecycleQuiescent, "observed drain completion", nil, ""
	case ActionApplicationRetire:
		_, runtime := runtimeState(observed, a.Application, a.Version)
		return !observed.IsActive(a.Application, a.Version) && !runtime, "observed retired release", nil, ""
	default:
		return false, "", nil, ""
	}
}

func runtimeState(s deployment.Snapshot, app core.CanonicalID, version string) (deployment.ApplicationRuntimeState, bool) {
	for _, x := range s.ApplicationRuntime {
		if x.Application == app && x.Version == version {
			return x, true
		}
	}
	return deployment.ApplicationRuntimeState{}, false
}

func prerequisitesComplete(records []ActionRecord, ids []string) bool {
	for _, id := range ids {
		found := false
		for _, r := range records {
			if r.ID == id {
				found = r.Status == ActionCompleted
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func mergeEvidence(run *Run, incoming []Evidence) error {
	byID := map[string]Evidence{}
	for _, e := range run.Evidence {
		byID[e.ID] = e
	}
	for _, e := range incoming {
		if e.ID == "" || e.ActionID == "" {
			return fmt.Errorf("evidence id and actionId are required")
		}
		if e.Outcome != EvidencePass && e.Outcome != EvidenceFail {
			return fmt.Errorf("evidence %s has invalid outcome %s", e.ID, e.Outcome)
		}
		if old, ok := byID[e.ID]; ok {
			a, _ := json.Marshal(old)
			b, _ := json.Marshal(e)
			if string(a) != string(b) {
				return fmt.Errorf("evidence id %s was reused with different content", e.ID)
			}
			continue
		}
		target, ok := actionRecord(run.Actions, e.ActionID)
		if !ok {
			return fmt.Errorf("evidence %s targets unknown action %s", e.ID, e.ActionID)
		}
		if target.Mode != ModeEvidence {
			return fmt.Errorf("evidence %s targets non-evidence action %s", e.ID, e.ActionID)
		}
		if e.Kind != target.EvidenceKind {
			return fmt.Errorf("evidence %s kind %s does not match gate %s kind %s", e.ID, e.Kind, e.ActionID, target.EvidenceKind)
		}
		if !prerequisitesComplete(run.Actions, target.Prerequisites) {
			return fmt.Errorf("evidence %s was supplied before gate %s became active", e.ID, e.ActionID)
		}
		byID[e.ID] = e
		run.Evidence = append(run.Evidence, e)
		kind := EventEvidenceAccepted
		if e.Outcome == EvidenceFail {
			kind = EventEvidenceRejected
		}
		emitOnce(run, Event{Kind: kind, ActionID: e.ActionID, EvidenceID: e.ID, Message: e.Message})
	}
	sort.Slice(run.Evidence, func(i, j int) bool { return run.Evidence[i].ID < run.Evidence[j].ID })
	return nil
}

// ReadyCommands converts currently-ready TEI actions into provider-neutral
// executor commands. The command idempotency key is stable across retries.
func ReadyCommands(run Run) []executor.Command {
	var out []executor.Command
	ready := map[string]bool{}
	for _, id := range run.ReadyActions {
		ready[id] = true
	}
	for _, rec := range run.Actions {
		if !ready[rec.ID] {
			continue
		}
		cmd, ok := CommandForAction(rec)
		if !ok {
			continue
		}
		out = append(out, cmd)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ActionID < out[j].ActionID })
	return out
}

// CommandForAction returns the provider-neutral command identity for a command
// action, independent of whether it is currently ready for dispatch.
func CommandForAction(rec ActionRecord) (executor.Command, bool) {
	if rec.Mode != ModeCommand {
		return executor.Command{}, false
	}
	return executor.Command{ID: rec.ID, ActionID: rec.ID, IdempotencyKey: rec.IdempotencyKey, Kind: string(rec.Kind), Application: rec.Application, Version: rec.Version, Database: rec.Database, TrafficBasisPoints: rec.TrafficBasisPoints}, true
}

// RecordExecutorResult stores an executor acknowledgement without treating it
// as proof of external completion. Accepted/unknown outcomes wait for observed
// state; rejected outcomes require explicit retry or recovery handling.
func RecordExecutorResult(run Run, result executor.Result) Run {
	before := runDigest(run)
	baseRevision := run.Revision
	if run.Status == RunSucceeded || run.Status == RunAborted || run.Status == RunBlocked || run.Status == RunRecoveryPlanned {
		return finalizeRevision(run, before, baseRevision)
	}
	if result.AttemptID == "" || result.ActionID == "" || result.ExecutorID == "" {
		run.Status = RunBlocked
		run.Findings = appendUnique(run.Findings, "executor result requires executorId, attemptId and actionId")
		return finalizeRevision(run, before, baseRevision)
	}
	rec, ok := actionRecord(run.Actions, result.ActionID)
	if !ok || rec.Mode != ModeCommand {
		run.Status = RunBlocked
		run.Findings = appendUnique(run.Findings, fmt.Sprintf("executor result targets non-command or unknown action %s", result.ActionID))
		return finalizeRevision(run, before, baseRevision)
	}
	if result.IdempotencyKey != rec.IdempotencyKey {
		run.Status = RunBlocked
		run.Findings = appendUnique(run.Findings, fmt.Sprintf("executor result %s idempotency key does not match action %s", result.AttemptID, result.ActionID))
		return finalizeRevision(run, before, baseRevision)
	}
	if result.Outcome != executor.OutcomeAccepted && result.Outcome != executor.OutcomeRejected && result.Outcome != executor.OutcomeUnknown {
		run.Status = RunBlocked
		run.Findings = appendUnique(run.Findings, fmt.Sprintf("executor result %s has invalid outcome %s", result.AttemptID, result.Outcome))
		return finalizeRevision(run, before, baseRevision)
	}
	for _, old := range run.Attempts {
		if old.AttemptID != result.AttemptID {
			continue
		}
		a, _ := json.Marshal(old)
		b, _ := json.Marshal(result)
		if string(a) != string(b) {
			run.Status = RunBlocked
			run.Findings = appendUnique(run.Findings, fmt.Sprintf("attempt id %s was reused with different content", result.AttemptID))
		}
		return finalizeRevision(run, before, baseRevision)
	}
	run.Attempts = append(run.Attempts, result)
	emitOnce(&run, Event{Kind: EventCommandAttempted, ActionID: result.ActionID, AttemptID: result.AttemptID, Message: string(result.Outcome) + ": " + result.Message})
	switch result.Outcome {
	case executor.OutcomeRejected:
		if result.Retryable {
			run.Status = RunPaused
			run.PauseReason = fmt.Sprintf("command %s rejected; explicit retry required", result.ActionID)
		} else {
			run.Status = RunNeedsRecovery
			run.Findings = appendUnique(run.Findings, fmt.Sprintf("command %s rejected by executor %s: %s", result.ActionID, result.ExecutorID, result.Message))
		}
	case executor.OutcomeAccepted, executor.OutcomeUnknown:
		run.Status = RunWaitingObservation
	}
	run.ReadyActions = nil
	run.WaitingActions = []string{result.ActionID}
	return finalizeRevision(run, before, baseRevision)
}

func latestAttempt(run Run, actionID string) (executor.Result, bool) {
	for i := len(run.Attempts) - 1; i >= 0; i-- {
		if run.Attempts[i].ActionID == actionID {
			return run.Attempts[i], true
		}
	}
	return executor.Result{}, false
}

func retryAuthorizedAfter(run Run, actionID, attemptID string) bool {
	for i := len(run.Retries) - 1; i >= 0; i-- {
		r := run.Retries[i]
		if r.ActionID == actionID && r.AfterAttemptID == attemptID {
			return true
		}
	}
	return false
}

func waitingForCommandObservation(run Run, actionID string) bool {
	a, ok := latestAttempt(run, actionID)
	if !ok {
		return false
	}
	if retryAuthorizedAfter(run, actionID, a.AttemptID) {
		return false
	}
	return a.Outcome == executor.OutcomeAccepted || a.Outcome == executor.OutcomeUnknown || (a.Outcome == executor.OutcomeRejected && a.Retryable)
}

// Retry explicitly authorizes redispatch of a retryable rejected/unknown
// command. The original idempotency key remains unchanged.
func Retry(run Run, actionID, reason string) Run {
	before := runDigest(run)
	base := run.Revision
	a, ok := latestAttempt(run, actionID)
	if !ok || (a.Outcome != executor.OutcomeUnknown && !(a.Outcome == executor.OutcomeRejected && a.Retryable)) {
		run.Status = RunBlocked
		run.Findings = appendUnique(run.Findings, fmt.Sprintf("action %s has no retryable failed/unknown attempt", actionID))
		return finalizeRevision(run, before, base)
	}
	if retryAuthorizedAfter(run, actionID, a.AttemptID) {
		return finalizeRevision(run, before, base)
	}
	run.Retries = append(run.Retries, RetryAuthorization{ActionID: actionID, AfterAttemptID: a.AttemptID, Reason: reason})
	emitOnce(&run, Event{Kind: EventRetryAuthorized, ActionID: actionID, AttemptID: a.AttemptID, Message: reason})
	run.Status = RunReady
	run.PauseReason = ""
	run.ReadyActions = []string{actionID}
	run.WaitingActions = nil
	return finalizeRevision(run, before, base)
}

func Pause(run Run, reason string) Run {
	before := runDigest(run)
	base := run.Revision
	if run.Status == RunSucceeded || run.Status == RunAborted {
		return finalizeRevision(run, before, base)
	}
	run.Status = RunPaused
	run.PauseReason = reason
	run.ReadyActions = nil
	run.WaitingActions = nil
	emitOnce(&run, Event{Kind: EventRunPaused, Message: reason})
	return finalizeRevision(run, before, base)
}

func Resume(run Run) Run {
	before := runDigest(run)
	base := run.Revision
	if run.Status != RunPaused {
		return finalizeRevision(run, before, base)
	}
	if actionID, ok := unresolvedExplicitRetry(run); ok {
		run.Findings = appendUnique(run.Findings, fmt.Sprintf("action %s requires explicit retry authorization before resume", actionID))
		return finalizeRevision(run, before, base)
	}
	run.Status = RunReady
	run.PauseReason = ""
	emitOnce(&run, Event{Kind: EventRunResumed})
	run = Reconcile(run, run.LastObserved, nil)
	if run.Revision <= base {
		run.Revision = base + 1
	}
	return run
}

func unresolvedExplicitRetry(run Run) (string, bool) {
	for _, rec := range run.Actions {
		if rec.Status == ActionCompleted || rec.Mode != ModeCommand {
			continue
		}
		a, ok := latestAttempt(run, rec.ID)
		if !ok {
			continue
		}
		if a.Outcome == executor.OutcomeRejected && a.Retryable && !retryAuthorizedAfter(run, rec.ID, a.AttemptID) {
			return rec.ID, true
		}
	}
	return "", false
}

// Abort is terminal and deliberately does not imply rollback.
func Abort(run Run, reason string) Run {
	before := runDigest(run)
	base := run.Revision
	if run.Status == RunSucceeded || run.Status == RunAborted {
		return finalizeRevision(run, before, base)
	}
	run.Status = RunAborted
	run.ReadyActions = nil
	run.WaitingActions = nil
	emitOnce(&run, Event{Kind: EventRunAborted, Message: reason})
	return finalizeRevision(run, before, base)
}

// PlanRecovery records an explicit recovery intent. Forward-fix and rollback
// point at a separate Solution Release; TEI never mutates the failed plan or
// assumes a generic inverse operation exists.
func PlanRecovery(run Run, d RecoveryDecision) Run {
	before := runDigest(run)
	base := run.Revision
	if d.ID == "" {
		run.Status = RunBlocked
		run.Findings = appendUnique(run.Findings, "recovery decision id is required")
		return finalizeRevision(run, before, base)
	}
	if d.Strategy == RecoveryRetry {
		return Retry(run, d.ActionID, d.Reason)
	}
	if d.Strategy != RecoveryForwardFix && d.Strategy != RecoveryRollback && d.Strategy != RecoveryManual {
		run.Status = RunBlocked
		run.Findings = appendUnique(run.Findings, fmt.Sprintf("unknown recovery strategy %s", d.Strategy))
		return finalizeRevision(run, before, base)
	}
	if (d.Strategy == RecoveryForwardFix || d.Strategy == RecoveryRollback) && (d.TargetSolutionID == "" || d.TargetSolutionVersion == "") {
		run.Status = RunBlocked
		run.Findings = appendUnique(run.Findings, "forward-fix/rollback recovery requires target solution id and version")
		return finalizeRevision(run, before, base)
	}
	copy := d
	run.Recovery = &copy
	run.ReadyActions = nil
	run.WaitingActions = nil
	if d.Strategy == RecoveryManual {
		run.Status = RunPaused
		run.PauseReason = d.Reason
	} else {
		run.Status = RunRecoveryPlanned
	}
	emitOnce(&run, Event{Kind: EventRecoveryPlanned, ActionID: d.ActionID, Message: string(d.Strategy) + ": " + d.Reason})
	return finalizeRevision(run, before, base)
}

// StartRecovery creates a distinct durable run for a forward-fix or rollback
// Solution Release and links it to the immutable failed run. The failed plan is
// never rewritten in place.
func StartRecovery(parent Run, recoveryRunID string, def releaseplan.Definition, compiled releaseplan.Compiled, observed deployment.Snapshot) (Run, Run, error) {
	if parent.Status != RunRecoveryPlanned || parent.Recovery == nil {
		return parent, Run{}, fmt.Errorf("deployment run %s has no startable recovery plan", parent.RunID)
	}
	d := parent.Recovery
	if d.Strategy != RecoveryForwardFix && d.Strategy != RecoveryRollback {
		return parent, Run{}, fmt.Errorf("recovery strategy %s does not start a separate durable run", d.Strategy)
	}
	if d.TargetSolutionID != def.ID || d.TargetSolutionVersion != def.Version {
		return parent, Run{}, fmt.Errorf("recovery decision targets %s@%s, supplied plan is %s@%s", d.TargetSolutionID, d.TargetSolutionVersion, def.ID, def.Version)
	}
	if recoveryRunID == "" {
		return parent, Run{}, fmt.Errorf("recovery run id is required")
	}
	if d.RecoveryRunID != "" {
		return parent, Run{}, fmt.Errorf("recovery decision %s is already linked to run %s", d.ID, d.RecoveryRunID)
	}
	if observed.Environment != parent.Environment {
		return parent, Run{}, fmt.Errorf("recovery observed environment %s does not match parent environment %s", observed.Environment, parent.Environment)
	}
	child := Start(recoveryRunID, def, compiled, observed)
	if child.Status == RunBlocked {
		return parent, child, fmt.Errorf("recovery run could not start: %v", child.Findings)
	}
	childBefore, childBase := runDigest(child), child.Revision
	child.RecoveryOf = &RecoveryOrigin{ParentRunID: parent.RunID, DecisionID: d.ID, Strategy: d.Strategy}
	child = finalizeRevision(child, childBefore, childBase)

	before, base := runDigest(parent), parent.Revision
	copy := *parent.Recovery
	copy.RecoveryRunID = recoveryRunID
	parent.Recovery = &copy
	emitOnce(&parent, Event{Kind: EventRecoveryRunLinked, Message: recoveryRunID})
	parent = finalizeRevision(parent, before, base)
	return parent, child, nil
}

// StartRecoveryWithPluginLock is the bound counterpart of StartRecovery. The
// child is a new execution artifact and may therefore be pinned to a different
// Plugin Lock from its parent when that change is explicit.
func StartRecoveryWithPluginLock(parent Run, recoveryRunID string, def releaseplan.Definition, compiled releaseplan.Compiled, observed deployment.Snapshot, pluginLockFingerprint string) (Run, Run, error) {
	if pluginLockFingerprint == "" {
		return parent, Run{}, fmt.Errorf("plugin lock fingerprint is required for a bound recovery run")
	}
	updated, child, err := StartRecovery(parent, recoveryRunID, def, compiled, observed)
	if err != nil {
		return updated, child, err
	}
	before, base := runDigest(child), child.Revision
	child.FormatVersion = 4
	child.PluginLockFingerprint = pluginLockFingerprint
	child = finalizeRevision(child, before, base)
	return updated, child, nil
}

// StartRecoveryWithRuntimePolicy is the resource-bound recovery counterpart.
// A recovery run is a new artifact, so an operator may explicitly select a new
// Plugin Lock and runtime policy while preserving the parent history.
func StartRecoveryWithRuntimePolicy(parent Run, recoveryRunID string, def releaseplan.Definition, compiled releaseplan.Compiled, observed deployment.Snapshot, pluginLockFingerprint, runtimePolicyFingerprint string) (Run, Run, error) {
	if pluginLockFingerprint == "" || runtimePolicyFingerprint == "" {
		return parent, Run{}, fmt.Errorf("plugin lock and runtime policy fingerprints are required for a bound recovery run")
	}
	updated, child, err := StartRecovery(parent, recoveryRunID, def, compiled, observed)
	if err != nil {
		return updated, child, err
	}
	before, base := runDigest(child), child.Revision
	child.FormatVersion = 5
	child.PluginLockFingerprint = pluginLockFingerprint
	child.RuntimePolicyFingerprint = runtimePolicyFingerprint
	child = finalizeRevision(child, before, base)
	return updated, child, nil
}

func actionRecord(records []ActionRecord, id string) (ActionRecord, bool) {
	for _, r := range records {
		if r.ID == id {
			return r, true
		}
	}
	return ActionRecord{}, false
}
func emitOnce(run *Run, e Event) {
	for _, old := range run.Events {
		if old.Kind == e.Kind && old.ActionID == e.ActionID && old.EvidenceID == e.EvidenceID && old.AttemptID == e.AttemptID {
			return
		}
	}
	e.Sequence = len(run.Events) + 1
	run.Events = append(run.Events, e)
}
func appendUnique(xs []string, v string) []string {
	for _, x := range xs {
		if x == v {
			return xs
		}
	}
	return append(xs, v)
}
func appendUniqueString(xs []string, v string) []string { return appendUnique(xs, v) }

// Ensure immutable release types remain part of this package's compile-time
// boundary: durable runs refer to snapshots, never mutate release definitions.
var _ apprelease.Definition
