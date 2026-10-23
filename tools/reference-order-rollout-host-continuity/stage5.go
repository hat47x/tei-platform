package orderrollouthost

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"tei/internal/core"
	dbrelease "tei/internal/database/release"
	"tei/internal/deployment"
	"tei/internal/deployment/durable"
	"tei/internal/deployment/executor"
	pruntime "tei/internal/plugin/runtime"
)

// ObservationSource says where an observation came from. Only a
// re-observation taken after the command may complete it.
type ObservationSource string

const (
	SourceReobserved    ObservationSource = "reobserved-after-command"
	SourceScenarioFixed ObservationSource = "scenario-fixed"
)

// Observation is the post-command observation of step 5.
type Observation struct {
	Source ObservationSource `json:"source"`
	// Seq is the logical time of the observation (orhc.event_seq), compared
	// with the logical time of the reconciliation it must follow.
	Seq      int64             `json:"seq"`
	Snapshot deployment.Snapshot `json:"snapshot"`
	// Evidence read from the instance besides the release record.
	IndexPresent   bool     `json:"indexPresent"`
	LegacyColumns  []string `json:"legacyColumns"`
	Releases       []string `json:"releasesRecorded"`
	JobCount       int      `json:"executionCount"`
}

// Observe re-reads the target state: the release the migration tool recorded
// in the instance, the presence of the schema change, and the active
// application instances. It fails (to "arrival unconfirmed") when the
// instance cannot be read.
func Observe(ctx context.Context, control, sales *pgxpool.Pool, s Start) (Observation, error) {
	o := Observation{Source: SourceReobserved}
	if err := control.QueryRow(ctx, `SELECT nextval('orhc.event_seq')`).Scan(&o.Seq); err != nil {
		return o, fmt.Errorf("logical clock unavailable")
	}
	rows, err := sales.Query(ctx, `SELECT release FROM tei_atlas.revision ORDER BY release`)
	if err != nil {
		return o, fmt.Errorf("target instance unavailable")
	}
	for rows.Next() {
		var r string
		if err := rows.Scan(&r); err != nil {
			rows.Close()
			return o, err
		}
		o.Releases = append(o.Releases, r)
	}
	rows.Close()
	if err := sales.QueryRow(ctx, `SELECT count(*) > 0 FROM pg_indexes WHERE schemaname='public' AND tablename='orders' AND indexname='orders_customer_idx'`).Scan(&o.IndexPresent); err != nil {
		return o, fmt.Errorf("target instance unavailable")
	}
	cr, err := sales.Query(ctx, `SELECT column_name FROM information_schema.columns WHERE table_schema='public' AND table_name='orders' ORDER BY ordinal_position`)
	if err != nil {
		return o, fmt.Errorf("target instance unavailable")
	}
	for cr.Next() {
		var c string
		_ = cr.Scan(&c)
		o.LegacyColumns = append(o.LegacyColumns, c)
	}
	cr.Close()
	_ = sales.QueryRow(ctx, `SELECT count(*) FROM tei_atlas.job`).Scan(&o.JobCount)
	snap := deployment.Snapshot{Environment: s.Environment, ObservedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	ar, err := control.Query(ctx, `SELECT DISTINCT application, version FROM orhc.app_instance WHERE active ORDER BY 1,2`)
	if err != nil {
		return o, fmt.Errorf("application runtime registry unavailable")
	}
	for ar.Next() {
		var app, ver string
		_ = ar.Scan(&app, &ver)
		snap.ActiveApplications = append(snap.ActiveApplications, deployment.ApplicationReleaseRef{Application: core.CanonicalID(app), Version: ver})
	}
	ar.Close()
	// The release is the latest recorded one whose schema change is present.
	// Its contracts come from the release definition of that version.
	st := dbrelease.StateSnapshot{Database: core.CanonicalID(SalesDatabase), Source: dbrelease.StateFromObserved, DataReadiness: "ready", ObservedAt: snap.ObservedAt}
	switch {
	case slices.Contains(o.Releases, s.TargetRelease) && o.IndexPresent:
		st.Release = s.TargetRelease
		st.Provides = s.TargetDef.Provides
	case len(o.Releases) == 0 && !o.IndexPresent:
		st.Release = s.InitialSalesRelease
	default:
		return o, fmt.Errorf("release record and schema disagree (releases %v, index %v)", o.Releases, o.IndexPresent)
	}
	snap.Databases = []dbrelease.StateSnapshot{st}
	o.Snapshot = snap.Normalize()
	return o, nil
}

// Arrival is the end state of step 5.
type Arrival string

const (
	ArrivalCompleted   Arrival = "completed"
	ArrivalUnresolved  Arrival = "unresolved"
	ArrivalUnconfirmed Arrival = "arrival-unconfirmed"
)

// Reflection is the output of step 5.
type Reflection struct {
	Identity        Identity        `json:"identity"`
	Attempt         executor.Result `json:"attempt"`
	ReconciledSeq   int64           `json:"reconciledSeq"`
	Observation     *Observation    `json:"observation,omitempty"`
	ObservationNote string          `json:"observationNote,omitempty"`
	Arrival         Arrival         `json:"arrival"`
	RunStatus       string          `json:"runStatus"`
	ActionStatus    string          `json:"actionStatus"`
	ReadyActions    []string        `json:"readyActions"`
	WaitingActions  []string        `json:"waitingActions"`
	Attempts        int             `json:"attempts"`
	RunRevision     int             `json:"runRevision"`
	Run             durable.Run     `json:"-"`
}

// AttemptFromRecovery maps the reconciliation of step 4 to an executor
// acknowledgement. No outcome claims the target state: reattached and
// already-succeeded are "accepted" (waiting for observation); unknown and
// lost are "unknown" and must never be re-executed automatically.
func AttemptFromRecovery(s Start, r pruntime.InvocationRecoveryResult) executor.Result {
	res := executor.Result{ExecutorID: CapabilityID, AttemptID: "attempt:" + r.InvocationID + ":recovered", ActionID: r.Handle.ActionID,
		IdempotencyKey: s.Command.IdempotencyKey, ExternalRef: "invocation:" + r.InvocationID}
	switch r.Outcome {
	case pruntime.RecoveryReattached:
		res.Outcome, res.Message = executor.OutcomeAccepted, "execution reattached by the new supervisor; completion requires observation"
	case pruntime.RecoveryAlreadySucceeded:
		res.Outcome, res.Message = executor.OutcomeAccepted, "execution reported success; completion requires observation"
	case pruntime.RecoveryAlreadyFailed:
		res.Outcome, res.Message = executor.OutcomeRejected, "execution reported failure"
	case pruntime.RecoveryRuntimeLost:
		res.Outcome, res.Message = executor.OutcomeUnknown, "execution lost; automatic re-execution is forbidden"
	default:
		res.Outcome, res.Message = executor.OutcomeUnknown, "execution state unknown; keep the command unresolved"
	}
	return res
}

// Reflect is step 5: record the attempt in the run, then accept an
// observation only if it is a re-observation taken after the reconciliation
// (obs.Seq > reconciledSeq). Without one, the result is "arrival
// unconfirmed" and the run stays waiting. The command is completed only if
// the observation shows sales_core 4.1.0 with order_entry 3.7.0 as the only
// active application.
func Reflect(s Start, run durable.Run, attempt executor.Result, reconciledSeq int64, obs *Observation, obsErr error) Reflection {
	// The durable functions update the action slice in place; work on a
	// deep copy so the caller's run is never changed.
	run = cloneRun(run)
	run = durable.RecordExecutorResult(run, attempt)
	out := Reflection{Identity: s.Identity(), Attempt: attempt, ReconciledSeq: reconciledSeq, Observation: obs}
	out.Identity.ActionID, out.Identity.IdempotencyKey = attempt.ActionID, attempt.IdempotencyKey
	out.Arrival = ArrivalUnconfirmed
	switch {
	case obsErr != nil:
		out.ObservationNote = "no re-observation: " + obsErr.Error()
	case obs == nil:
		out.ObservationNote = "no re-observation"
	case obs.Source != SourceReobserved:
		out.ObservationNote = fmt.Sprintf("observation source %s is not a re-observation after the command", obs.Source)
	case obs.Seq <= reconciledSeq:
		out.ObservationNote = fmt.Sprintf("observation seq %d does not follow the reconciliation seq %d", obs.Seq, reconciledSeq)
	case !(obs.Snapshot.IsActive(OrderApplication, LegacyVersion) && len(obs.Snapshot.ActiveApplications) == 1):
		// The run model completes a database release from the release
		// alone; the target of this command also requires order_entry
		// 3.7.0 to be the only active application, so such an
		// observation is not handed to the run.
		out.Arrival = ArrivalUnresolved
		out.ObservationNote = "observed applications differ from the target state (order_entry 3.7.0 only)"
	default:
		run = durable.Reconcile(run, obs.Snapshot, nil)
		if actionStatus(run, attempt.ActionID) == string(durable.ActionCompleted) {
			out.Arrival = ArrivalCompleted
		} else {
			out.Arrival = ArrivalUnresolved
		}
	}
	out.Run = run
	out.RunStatus = string(run.Status)
	out.RunRevision = run.Revision
	out.ActionStatus = actionStatus(run, attempt.ActionID)
	out.ReadyActions = append([]string{}, run.ReadyActions...)
	out.WaitingActions = append([]string{}, run.WaitingActions...)
	out.Attempts = len(run.Attempts)
	return out
}

func cloneRun(run durable.Run) durable.Run {
	raw, err := json.Marshal(run)
	if err != nil {
		panic(err)
	}
	var out durable.Run
	if err := json.Unmarshal(raw, &out); err != nil {
		panic(err)
	}
	return out
}

func actionStatus(run durable.Run, id string) string {
	for _, a := range run.Actions {
		if a.ID == id {
			return string(a.Status)
		}
	}
	return ""
}

// FixedObservation wraps the run's start observation (scenario time) as an
// observation. It exists to show that it is refused as arrival evidence.
func FixedObservation(run durable.Run) *Observation {
	return &Observation{Source: SourceScenarioFixed, Snapshot: run.Initial}
}
