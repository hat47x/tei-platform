package orderrollouthost

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"tei/internal/deployment/durable"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/resource"
)

const (
	hostA          = "host-a"
	hostB          = "host-b"
	releaseBarrier = "b-release"
)

type takeoverOpts struct {
	aOffset, bOffset string
	afterStop        func(s *scenario)
	bWorkspace       string
	openGate         bool
}

type takeoverRun struct {
	writer          *legacyWriter
	a               CallRecord
	b               TakeoverRecord
	aErr, bErr      string
	takeoverSeq     int64
	refusedWhileRun int64
}

// runTakeover runs steps 2-4 with host A and host B as separate processes:
// host A invokes the command, host B waits; once host A has seen the
// acceptance notice, renewed the lease and heartbeat its ownership, and host
// B has been refused at least once, host A is stopped (SIGSTOP). Host B takes
// over after the ownership stood still, fences the lease and reconciles.
// Then host A resumes and retries every operation with its stale copies, and
// only then host B returns the lease.
func (s *scenario) runTakeover(o takeoverOpts) takeoverRun {
	s.t.Helper()
	var r takeoverRun
	r.writer = s.startWriter()
	s.waitFor("legacy writes before the command", 20*time.Second, func() (bool, error) { return r.writer.n() >= 5, nil })
	a := s.spawn("a", hostA, "-clock-offset", o.aOffset, "-ownership-ttl", "3s", "-lease-slack", "400ms", "-apply-hold-ms", "600", "-probe-stale", "-observe-window", "1s")
	bargs := []string{"-clock-offset", o.bOffset, "-ownership-ttl", "3s", "-release-barrier", releaseBarrier, "-observe-window", "1s"}
	if o.bWorkspace != "" {
		bargs = append(bargs, "-workspace", o.bWorkspace)
	}
	b := s.spawn("b", hostB, bargs...)
	s.waitFor("acceptance notice", 60*time.Second, s.count(s.control, `SELECT count(*) FROM orhc.call_event WHERE kind='accepted-notice' AND actor=$1`, hostA))
	s.waitFor("an accepted lease renewal by host A", 30*time.Second, s.count(s.control, `SELECT count(*) FROM orhc.lease_event WHERE op='renew' AND accepted AND actor=$1`, hostA))
	s.waitFor("an ownership heartbeat by host A", 30*time.Second, s.count(s.control, `SELECT count(*) FROM orhc.supervisor_ownership WHERE owner_id=$1 AND heartbeat_seq > 0`, hostA))
	s.waitFor("a refused claim by host B", 30*time.Second, s.count(s.control, `SELECT count(*) FROM orhc.ownership_event WHERE op='claim-refused' AND actor=$1`, hostB))
	if err := StopProcess(a.cmd.Process.Pid); err != nil {
		s.t.Fatal(err)
	}
	r.refusedWhileRun = s.scalar(s.control, `SELECT count(*) FROM orhc.ownership_event WHERE op='claim-refused' AND actor=$1`, hostB)
	if o.afterStop != nil {
		o.afterStop(s)
	}
	s.waitFor("host B reconciliation", 90*time.Second, s.count(s.control, `SELECT count(*) FROM orhc.call_event WHERE kind='reconciled' AND actor=$1`, hostB))
	r.takeoverSeq = s.scalar(s.control, `SELECT seq FROM orhc.ownership_event WHERE op='takeover' AND actor=$1`, hostB)
	if o.openGate {
		if _, err := s.sales.Exec(s.ctx, `UPDATE tei_atlas.job SET gate_open=true`); err != nil {
			s.t.Fatal(err)
		}
		s.waitFor("the migration to finish", 60*time.Second, s.count(s.sales, `SELECT count(*) FROM tei_atlas.job WHERE lifecycle='succeeded'`))
	}
	if err := ContinueProcess(a.cmd.Process.Pid); err != nil {
		s.t.Fatal(err)
	}
	s.waitExit(a, 90*time.Second)
	if err := OpenBarrier(s.ctx, s.control, releaseBarrier); err != nil {
		s.t.Fatal(err)
	}
	s.waitExit(b, 60*time.Second)
	r.a, r.aErr = readReport[CallRecord](s.t, a.report)
	r.b, r.bErr = readReport[TakeoverRecord](s.t, b.report)
	if r.aErr != "" || r.bErr != "" {
		s.t.Fatalf("host errors: A=%q B=%q", r.aErr, r.bErr)
	}
	return r
}

// checkFencing checks P-OPS-01 on the persisted events: after the takeover
// no operation of host A was accepted and every probe was rejected, only host
// B returned the lease, and nothing is left active.
func (s *scenario) checkFencing(r takeoverRun, wantProbes map[string]bool) map[string]any {
	s.t.Helper()
	acceptedStale := s.scalar(s.control, `SELECT count(*) FROM (SELECT seq, accepted FROM orhc.lease_event WHERE actor=$1 UNION ALL SELECT seq, accepted FROM orhc.ownership_event WHERE actor=$1) x WHERE seq > $2 AND accepted`, hostA, r.takeoverSeq)
	rejectedStale := s.scalar(s.control, `SELECT count(*) FROM (SELECT seq, accepted FROM orhc.lease_event WHERE actor=$1 UNION ALL SELECT seq, accepted FROM orhc.ownership_event WHERE actor=$1) x WHERE seq > $2 AND NOT accepted`, hostA, r.takeoverSeq)
	releasedBy := s.scalar(s.control, `SELECT count(*) FROM orhc.lease_event WHERE op='release' AND accepted AND actor=$1`, hostB)
	releasedByA := s.scalar(s.control, `SELECT count(*) FROM orhc.lease_event WHERE op='release' AND accepted AND actor=$1`, hostA)
	active := s.scalar(s.control, `SELECT count(*) FROM orhc.resource_lease WHERE state <> 'released'`)
	owners := s.scalar(s.control, `SELECT count(*) FROM orhc.supervisor_ownership`)
	if acceptedStale != 0 || rejectedStale == 0 {
		s.t.Errorf("host A after takeover: accepted=%d rejected=%d", acceptedStale, rejectedStale)
	}
	if releasedBy != 1 || releasedByA != 0 || active != 0 || owners != 0 || !r.b.LeaseReleased || !r.b.OwnershipReleased || r.b.LeaseStateAfter != "released" {
		s.t.Errorf("release: byB=%d byA=%d active=%d owners=%d report=%+v", releasedBy, releasedByA, active, owners, r.b)
	}
	probes := map[string]bool{}
	for _, p := range r.a.StaleProbes {
		probes[p.Op] = p.Rejected
	}
	for op, want := range wantProbes {
		if got, ok := probes[op]; !ok || got != want {
			s.t.Errorf("stale probe %s rejected=%v (present %v), want %v; probes=%+v", op, got, ok, want, r.a.StaleProbes)
		}
	}
	if !r.a.CallFailed || r.a.Result != nil {
		s.t.Errorf("host A call must fail after the takeover: %+v", r.a)
	}
	if r.b.FencingToken <= r.a.FencingToken || r.b.PreviousOwner != hostA || r.b.PreviousToken != r.a.FencingToken {
		s.t.Errorf("tokens A=%d B=%d previous=%s/%d", r.a.FencingToken, r.b.FencingToken, r.b.PreviousOwner, r.b.PreviousToken)
	}
	return map[string]any{"acceptedAfterTakeoverByA": acceptedStale, "rejectedAfterTakeoverByA": rejectedStale, "leaseReleasedByB": releasedBy,
		"leaseReleasedByA": releasedByA, "activeLeases": active, "ownershipRows": owners, "probes": probes,
		"tokenA": r.a.FencingToken, "tokenB": r.b.FencingToken, "refusedClaims": r.b.RefusedClaims, "takeoverWaitMs": r.b.TakeoverWaitMS,
		"naiveAtRefusals": countBools(r.b.NaiveAtRefusals), "naiveAtTakeover": r.b.NaiveAtTakeover}
}

func countBools(xs []bool) map[string]int {
	out := map[string]int{"true": 0, "false": 0}
	for _, x := range xs {
		if x {
			out["true"]++
		} else {
			out["false"]++
		}
	}
	return out
}

func supervisionStates(pi CallRecord) []string {
	var out []string
	if pi.Invocation == nil {
		return out
	}
	for _, x := range pi.Invocation.ResourceLeaseSupervision {
		out = append(out, string(x.State))
	}
	for _, x := range pi.Invocation.ResourceLeaseOwnership {
		out = append(out, string(x.State))
	}
	return out
}

func (s *scenario) checkNoLeak(extra ...string) []string {
	s.t.Helper()
	found := LeakScan(s.leakText(extra...), s.secrets)
	if len(found) != 0 {
		s.t.Errorf("concrete values found in the outputs of steps 1-5: %v", found)
	}
	// Positive control: the scan sees a value that is planted.
	if planted := LeakScan(s.leakText("planted "+s.sec.Endpoint), s.secrets); !slices.Contains(planted, "endpoint") {
		s.t.Errorf("leak scan did not find a planted value")
	}
	return found
}

func handleIdentity(start Start, h pruntime.DurableInvocationHandle) Identity {
	return Identity{RunID: h.RunID, StartRecordDigest: start.StartRecordDigest, PlanFingerprint: start.PlanFingerprint, PluginLockFingerprint: h.PluginLockFingerprint,
		RuntimePolicyFingerprint: h.RuntimePolicyFingerprint, ActionID: h.ActionID, IdempotencyKey: start.Command.IdempotencyKey, InvocationID: h.InvocationID, RequestID: h.RequestID}
}

// TestPostgresThroughHostTakeover is the end-to-end run of steps 1-5:
// P-OPS-01 with skewed host clocks (A +5m, B -3m), P-MIG-03 with order_entry
// 3.7.0 writing before, during and after the expand, completion only by the
// re-observation, the leak check over every output, and one run record that
// carries the same identity from step 1 to step 5.
func TestPostgresThroughHostTakeover(t *testing.T) {
	s := newScenario(t, "main")
	var chain Chain
	stage1 := s.writeOut("step1.json", s.start)
	chain.Append("1-start", s.start.Identity(), s.start.PluginLockMatches && s.start.RuntimePolicyMatches && s.start.ReadyCount == 1,
		map[string]any{"command": s.start.Command, "initialSalesRelease": s.start.InitialSalesRelease, "initialObservedAt": s.start.InitialObservedAt, "environmentPlanDigest": s.start.EnvironmentPlanDigest})

	r := s.runTakeover(takeoverOpts{aOffset: "5m", bOffset: "-3m", openGate: true})

	// Step 2 (host A): the call, its lease and its ownership.
	st2 := supervisionStates(r.a)
	renewals := 0
	if r.a.Invocation != nil {
		renewals = len(r.a.Invocation.ResourceLeaseRenewals)
	}
	ok2 := r.a.AcceptedSeq > 0 && renewals >= 1 && slices.Contains(st2, string(resource.OwnershipHeartbeat)) && r.a.Identity == s.start.Identity() &&
		r.a.Invocation != nil && r.a.Invocation.Sandboxed && r.a.NetworkAllowed == 1
	if !ok2 {
		t.Errorf("step 2: accepted=%d renewals=%d states=%v identity=%v", r.a.AcceptedSeq, renewals, st2, r.a.Identity == s.start.Identity())
	}
	chain.Append("2-invoke-host-a", r.a.Identity, ok2, map[string]any{"leaseId": r.a.LeaseID, "owner": r.a.Owner, "fencingToken": r.a.FencingToken,
		"acceptedNoticeSeq": r.a.AcceptedSeq, "renewals": renewals, "supervision": st2, "callFailed": r.a.CallFailed})

	// Step 3 (host B): takeover and fencing.
	fencing := s.checkFencing(r, map[string]bool{"ownership-heartbeat": true, "ownership-reclaim": true, "ownership-release": true, "lease-renew": true,
		"lease-release": true, "lease-fence": true, "runtime-reattach": true, "invocation-reconcile": true})
	clock := r.b.NaiveAtTakeover == false && countBools(r.b.NaiveAtRefusals)["true"] == 0
	if !clock {
		t.Errorf("A ahead, B behind: a host-clock design would not have taken over: naive at takeover=%v refusals=%v", r.b.NaiveAtTakeover, r.b.NaiveAtRefusals)
	}
	handover := Handover{CallLease: r.a.LeaseID, TakeoverLease: r.b.LeaseID, CallOwner: r.a.Owner, TakeoverPrevOwner: r.b.PreviousOwner, CallToken: r.a.FencingToken,
		TakeoverPrevToken: r.b.PreviousToken, TakeoverOwner: r.b.Owner, TakeoverToken: r.b.FencingToken}
	if r.b.Recovery != nil {
		handover.HandleLease, handover.HandleOwner, handover.HandleToken = r.b.Recovery.Handle.ResourceLeaseID, r.b.Recovery.Handle.SupervisorOwnerID, r.b.Recovery.Handle.FencingToken
	}
	if err := handover.Verify(); err != nil {
		t.Error(err)
	}
	chain.Append("3-takeover-host-b", r.b.Identity, !t.Failed(), map[string]any{"fencing": fencing, "clockOffsets": map[string]string{"a": "5m", "b": "-3m"}})

	// Step 4 (host B): reconciliation.
	if r.b.Recovery == nil || r.b.Recovery.Outcome != pruntime.RecoveryReattached || r.b.Reattaches != 1 || r.b.Recovery.Handle.RuntimeFencingToken != r.b.FencingToken {
		t.Fatalf("step 4: recovery=%+v err=%s reattaches=%d", r.b.Recovery, r.b.RecoveryError, r.b.Reattaches)
	}
	chain.Append("4-reconcile-host-b", handleIdentity(s.start, r.b.Recovery.Handle), true, map[string]any{"outcome": r.b.Recovery.Outcome,
		"handleRevision": r.b.Recovery.Handle.Revision, "runtimeFencingToken": r.b.Recovery.Handle.RuntimeFencingToken, "reconciledSeq": r.b.ReconciledSeq})

	// Step 5: the run record and the re-observation.
	attempt := AttemptFromRecovery(s.start, *r.b.Recovery)
	fixed := Reflect(s.start, s.start.Run, attempt, r.b.ReconciledSeq, FixedObservation(s.start.Run), nil)
	none := Reflect(s.start, s.start.Run, attempt, r.b.ReconciledSeq, nil, nil)
	obs, obsErr := Observe(s.ctx, s.control, s.sales, s.start)
	refl := Reflect(s.start, s.start.Run, attempt, r.b.ReconciledSeq, &obs, obsErr)
	if fixed.Arrival != ArrivalUnconfirmed || none.Arrival != ArrivalUnconfirmed || fixed.ActionStatus == string(durable.ActionCompleted) {
		t.Errorf("acceptance or the scenario-fixed observation completed the command: fixed=%s none=%s", fixed.Arrival, none.Arrival)
	}
	if refl.Arrival != ArrivalCompleted || !slices.Contains(refl.ReadyActions, FollowingActionID) || refl.ActionStatus != string(durable.ActionCompleted) {
		t.Errorf("step 5: %+v obsErr=%v", refl, obsErr)
	}
	if err := SaveRun(s.ctx, s.control, refl.Run); err != nil {
		t.Fatal(err)
	}
	salesContract := actionStatus(refl.Run, "sales_contract/apply")
	mig := r.writer.finish()
	columns := strings.Join(obs.LegacyColumns, ",")
	okMig := mig["errors"] == 0 && mig["lostOrChanged"] == 0 && mig["phases"].(map[string]int)["applying"] >= 1 && mig["phases"].(map[string]int)["after-apply"] >= 1 &&
		columns == "order_id,customer_id,ordered_at,status,total_amount,external_id" && slices.Equal(obs.Releases, []string{"4.1.0"}) && salesContract == string(durable.ActionPending) && obs.JobCount == 1
	if !okMig {
		t.Errorf("P-MIG-03: %v columns=%s releases=%v contract=%s jobs=%d", mig, columns, obs.Releases, salesContract, obs.JobCount)
	}
	nulls := s.scalar(s.sales, `SELECT count(*) FROM public.orders WHERE status IS NULL`)
	unknownFilled := s.scalar(s.sales, `SELECT count(*) FROM public.orders WHERE order_id <= 20 AND status='UNKNOWN'`)
	reflJSON := s.writeOut("step5.json", map[string]any{"reflection": refl, "refusedFixed": fixed, "refusedWithoutObservation": none, "run": refl.Run, "legacyWrites": mig})
	chain.Append("5-observe-and-reflect", refl.Identity, refl.Arrival == ArrivalCompleted && okMig, map[string]any{"arrival": refl.Arrival, "observationSeq": obs.Seq,
		"reconciledSeq": r.b.ReconciledSeq, "readyActions": refl.ReadyActions, "fixedObservation": fixed.Arrival, "withoutObservation": none.Arrival,
		"legacyWrites": mig, "nullStatusAfter": nulls, "legacyRowsFilledUnknown": unknownFilled})

	// Leak check over the outputs of all steps, then the run record.
	leaks := s.checkNoLeak(stage1, reflJSON)
	chain.Append("leak-check", s.start.Identity(), len(leaks) == 0, map[string]any{"found": leaks, "checkedValues": len(s.secrets)})
	required := []string{"1-start", "2-invoke-host-a", "3-takeover-host-b", "4-reconcile-host-b", "5-observe-and-reflect", "leak-check"}
	if err := chain.Verify(required); err != nil {
		t.Error(err)
	}
	if dir := os.Getenv(outDirEnv); dir != "" {
		raw, _ := json.MarshalIndent(chain, "", "  ")
		if err := os.WriteFile(filepath.Join(dir, "chain.json"), append(raw, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	tampered := Chain{Records: append([]Record(nil), chain.Records...)}
	tampered.Records[3].Identity.RequestID = "sales_expand/apply#attempt-2"
	if tampered.Verify(required) == nil {
		t.Error("a changed identity in step 4 was not detected")
	}
	observed(t, "fencing", fencing)
	observed(t, "legacyWrites", mig)
	observed(t, "step5", map[string]any{"arrival": refl.Arrival, "ready": refl.ReadyActions, "fixed": fixed.Arrival, "none": none.Arrival, "obsSeq": obs.Seq, "reconciledSeq": r.b.ReconciledSeq, "legacyRowsFilledUnknown": unknownFilled, "nullStatusAfter": nulls})
	observed(t, "chain", map[string]any{"records": len(chain.Records), "head": chain.Head(), "leaks": leaks})
}

// TestPostgresPTX02RuntimeUnknownStaysUnresolved: host A is stopped and the
// tool process is killed while it waits; the registry still says running
// but the execution makes no progress. Host B reconciles to "waiting
// observation", does not re-execute, and the command stays unresolved.
// Host clocks are skewed the other way (A -3m, B +5m): a host-clock design
// would have taken over from the live host A.
func TestPostgresPTX02RuntimeUnknownStaysUnresolved(t *testing.T) {
	s := newScenario(t, "unknown")
	r := s.runTakeover(takeoverOpts{aOffset: "-3m", bOffset: "5m", afterStop: func(s *scenario) {
		pid := s.scalar(s.sales, `SELECT os_pid FROM tei_atlas.job`)
		if err := KillProcess(int(pid)); err != nil {
			s.t.Fatal(err)
		}
	}})
	// The runtime-side token is not advanced when the state is unknown, so
	// host A's raw reattach on the tool registry is not rejected there; the
	// product reconciliation rejects host A's stale token.
	fencing := s.checkFencing(r, map[string]bool{"ownership-heartbeat": true, "ownership-reclaim": true, "ownership-release": true, "lease-renew": true,
		"lease-release": true, "lease-fence": true, "runtime-reattach": false, "invocation-reconcile": true})
	if countBools(r.b.NaiveAtRefusals)["true"] == 0 {
		t.Errorf("B ahead, A behind: expected refusals where a host-clock design would have taken over: %v", r.b.NaiveAtRefusals)
	}
	if r.b.Recovery == nil || r.b.Recovery.Outcome != pruntime.RecoveryWaitingObserve || r.b.Reattaches != 0 {
		t.Fatalf("recovery=%+v err=%s", r.b.Recovery, r.b.RecoveryError)
	}
	attempt := AttemptFromRecovery(s.start, *r.b.Recovery)
	obs, obsErr := Observe(s.ctx, s.control, s.sales, s.start)
	refl := Reflect(s.start, s.start.Run, attempt, r.b.ReconciledSeq, &obs, obsErr)
	mig := r.writer.finish()
	if refl.Arrival != ArrivalUnresolved || refl.ActionStatus != string(durable.ActionPending) || len(refl.ReadyActions) != 0 ||
		len(durable.ReadyCommands(refl.Run)) != 0 || !slices.Equal(refl.WaitingActions, []string{ExpectedActionID}) || obs.JobCount != 1 || len(obs.Releases) != 0 {
		t.Errorf("unknown must stay unresolved: %+v obs=%+v err=%v", refl, obs, obsErr)
	}
	if mig["errors"] != 0 || mig["lostOrChanged"] != 0 {
		t.Errorf("legacy writes: %v", mig)
	}
	if err := SaveRun(s.ctx, s.control, refl.Run); err != nil {
		t.Fatal(err)
	}
	rj := s.writeOut("step5.json", map[string]any{"reflection": refl, "run": refl.Run})
	leaks := s.checkNoLeak(rj)
	observed(t, "result", map[string]any{"fencing": fencing, "outcome": r.b.Recovery.Outcome, "findings": r.b.Recovery.Findings, "arrival": refl.Arrival,
		"runStatus": refl.RunStatus, "waiting": refl.WaitingActions, "jobs": obs.JobCount, "legacyWrites": mig, "leaks": leaks})
}

// TestPostgresPTX02RuntimeLostArrivalUnconfirmed: the tool process is killed
// and its registry entry is lost. Host B reconciles to "runtime lost"
// without re-executing; with the target instance unreadable the result is
// "arrival unconfirmed", and a later re-observation shows it unresolved.
func TestPostgresPTX02RuntimeLostArrivalUnconfirmed(t *testing.T) {
	s := newScenario(t, "lost")
	r := s.runTakeover(takeoverOpts{aOffset: "2m", bOffset: "0s", afterStop: func(s *scenario) {
		pid := s.scalar(s.sales, `SELECT os_pid FROM tei_atlas.job`)
		if err := KillProcess(int(pid)); err != nil {
			s.t.Fatal(err)
		}
		if _, err := s.sales.Exec(s.ctx, `DELETE FROM tei_atlas.job`); err != nil {
			s.t.Fatal(err)
		}
	}})
	fencing := s.checkFencing(r, map[string]bool{"ownership-heartbeat": true, "ownership-reclaim": true, "ownership-release": true, "lease-renew": true,
		"lease-release": true, "lease-fence": true, "invocation-reconcile": true})
	if r.b.Recovery == nil || r.b.Recovery.Outcome != pruntime.RecoveryRuntimeLost {
		t.Fatalf("recovery=%+v err=%s", r.b.Recovery, r.b.RecoveryError)
	}
	attempt := AttemptFromRecovery(s.start, *r.b.Recovery)
	unreadable, err := pgxpool.New(s.ctx, "postgres://nobody@127.0.0.1:1/none?sslmode=disable&connect_timeout=2")
	if err != nil {
		t.Fatal(err)
	}
	defer unreadable.Close()
	obsA, errA := Observe(s.ctx, s.control, unreadable, s.start)
	unconfirmed := Reflect(s.start, s.start.Run, attempt, r.b.ReconciledSeq, &obsA, errA)
	obsB, errB := Observe(s.ctx, s.control, s.sales, s.start)
	later := Reflect(s.start, s.start.Run, attempt, r.b.ReconciledSeq, &obsB, errB)
	mig := r.writer.finish()
	if unconfirmed.Arrival != ArrivalUnconfirmed || unconfirmed.RunStatus != string(durable.RunWaitingObservation) || len(unconfirmed.ReadyActions) != 0 {
		t.Errorf("unreadable target: %+v", unconfirmed)
	}
	if later.Arrival != ArrivalUnresolved || len(later.ReadyActions) != 0 || obsB.JobCount != 0 || len(obsB.Releases) != 0 {
		t.Errorf("later observation: %+v obs=%+v", later, obsB)
	}
	if err := SaveRun(s.ctx, s.control, unconfirmed.Run); err != nil {
		t.Fatal(err)
	}
	rj := s.writeOut("step5.json", map[string]any{"unconfirmed": unconfirmed, "later": later, "run": unconfirmed.Run})
	leaks := s.checkNoLeak(rj)
	observed(t, "result", map[string]any{"fencing": fencing, "outcome": r.b.Recovery.Outcome, "unconfirmed": unconfirmed.Arrival, "note": unconfirmed.ObservationNote,
		"later": later.Arrival, "legacyWrites": mig, "leaks": leaks})
}

// TestPostgresPlanChangedRejectedBeforeObservation: the environment plan is
// updated between the call and the handover (host B builds it from a copy
// of the workspace with another lease policy). Host B takes over the
// ownership but the product reconciliation refuses the old call record
// before observing or reattaching anything; the command stays unresolved.
func TestPostgresPlanChangedRejectedBeforeObservation(t *testing.T) {
	s := newScenario(t, "plan")
	ws := filepath.Join(t.TempDir(), "workspace")
	copyWorkspace(t, filepath.Join(s.repo, DefaultWorkspace), ws)
	r := s.runTakeover(takeoverOpts{aOffset: "0s", bOffset: "1m", bWorkspace: ws})
	fencing := s.checkFencing(r, map[string]bool{"ownership-heartbeat": true, "ownership-reclaim": true, "ownership-release": true, "lease-renew": true,
		"lease-release": true, "lease-fence": true, "invocation-reconcile": true})
	if r.b.Recovery != nil || !strings.Contains(r.b.RecoveryError, "runtime policy fingerprint changed") || r.b.Observes != 0 || r.b.Reattaches != 0 || r.b.PlanMatchesRun {
		t.Fatalf("plan change: recovery=%+v err=%q observes=%d reattaches=%d planMatches=%v", r.b.Recovery, r.b.RecoveryError, r.b.Observes, r.b.Reattaches, r.b.PlanMatchesRun)
	}
	h, ok, err := (&PgHandleStore{Pool: s.control}).Load(s.ctx, s.start.InvocationID)
	if err != nil || !ok || h.Revision != 1 || h.SupervisorOwnerID != hostA {
		t.Fatalf("the refused reconciliation must not update the handle: %+v %v", h, err)
	}
	attempt := AttemptFromRefusal(s.start, s.start.InvocationID, "environment plan changed")
	obs, obsErr := Observe(s.ctx, s.control, s.sales, s.start)
	refl := Reflect(s.start, s.start.Run, attempt, r.b.ReconciledSeq, &obs, obsErr)
	mig := r.writer.finish()
	if refl.Arrival != ArrivalUnresolved || len(refl.ReadyActions) != 0 || len(durable.ReadyCommands(refl.Run)) != 0 {
		t.Errorf("plan change must leave the command unresolved: %+v", refl)
	}
	rj := s.writeOut("step5.json", map[string]any{"reflection": refl, "run": refl.Run})
	leaks := s.checkNoLeak(rj)
	observed(t, "result", map[string]any{"fencing": fencing, "recoveryError": r.b.RecoveryError, "observes": r.b.Observes, "reattaches": r.b.Reattaches,
		"handleRevision": h.Revision, "arrival": refl.Arrival, "legacyWrites": mig, "leaks": leaks})
}

// TestPostgresStage2LeaseFailureFailsCall: the tool never looks at its
// lease; when the host's lease renewal or ownership heartbeat fails, the
// whole call fails, the tool process is stopped before it applies anything,
// and the lease and the ownership are returned.
func TestPostgresStage2LeaseFailureFailsCall(t *testing.T) {
	for _, c := range []struct {
		name         string
		renew, hb    bool
		wantState    string
		ownershipTTL time.Duration
	}{
		{"renewal", true, false, string(resource.SupervisionRenewalFailed), 30 * time.Second},
		{"heartbeat", false, true, string(resource.OwnershipLost), 600 * time.Millisecond},
	} {
		t.Run(c.name, func(t *testing.T) {
			s := newScenario(t, "s2"+c.name)
			cfg := HostConfig{Role: "a", Owner: hostA, Paths: DefaultPaths(s.repo), BinDir: s.binDir, OwnershipTTL: c.ownershipTTL, LeaseSlack: 300 * time.Millisecond,
				FailRenew: c.renew, FailHeartbeat: c.hb, ChildArgs: true, ObserveWindow: time.Second}
			ctx, cancel := context.WithTimeout(s.ctx, 90*time.Second)
			defer cancel()
			began := time.Now()
			rec, err := RunHostA(ctx, cfg, s.sec)
			elapsed := time.Since(began)
			if err != nil {
				t.Fatal(err)
			}
			states := supervisionStates(rec)
			jobs := &JobRuntime{Pool: s.sales, ObserveWindow: time.Second}
			var lc pruntime.RuntimeLifecycle
			if rec.RuntimeHandle != "" {
				o, err := jobs.ObserveInvocation(s.ctx, rec.RuntimeHandle)
				if err != nil {
					t.Fatal(err)
				}
				lc = o.Lifecycle
			}
			revisions := s.scalar(s.sales, `SELECT count(*) FROM tei_atlas.revision`)
			active := s.scalar(s.control, `SELECT count(*) FROM orhc.resource_lease WHERE state <> 'released'`)
			owners := s.scalar(s.control, `SELECT count(*) FROM orhc.supervisor_ownership`)
			if !rec.CallFailed || rec.Result != nil || !slices.Contains(states, c.wantState) || rec.AcceptedSeq == 0 || lc != pruntime.RuntimeUnknown || revisions != 0 || active != 0 || owners != 0 {
				t.Errorf("call=%v result=%v states=%v accepted=%d tool=%s revisions=%d active=%d owners=%d err=%s", rec.CallFailed, rec.Result, states, rec.AcceptedSeq, lc, revisions, active, owners, rec.CallError)
			}
			rj := s.writeOut("step2.json", rec)
			leaks := s.checkNoLeak(rj)
			observed(t, "result", map[string]any{"callFailed": rec.CallFailed, "states": states, "toolAfter": lc, "revisions": revisions, "elapsedMs": elapsed.Milliseconds(), "leaks": leaks})
		})
	}
}
