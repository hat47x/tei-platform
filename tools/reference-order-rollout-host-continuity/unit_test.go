package orderrollouthost

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"tei/internal/core"
	dbrelease "tei/internal/database/release"
	"tei/internal/deployment"
	"tei/internal/deployment/durable"
	"tei/internal/deployment/executor"
	pruntime "tei/internal/plugin/runtime"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func loadStart(t *testing.T) Start {
	t.Helper()
	s, err := LoadStart(DefaultPaths(repoRoot(t)))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestStartFromFixedInputs(t *testing.T) {
	s := loadStart(t)
	if s.Command.ActionID != ExpectedActionID || s.ReadyCount != 1 || !s.PluginLockMatches || !s.RuntimePolicyMatches {
		t.Fatalf("start=%+v", s)
	}
	if s.InitialSalesRelease != "4.0.0" || s.InitialObservedAt != "2026-08-27T18:30:00+09:00" || s.TargetRelease != "4.1.0" {
		t.Fatalf("initial=%s at %s target=%s", s.InitialSalesRelease, s.InitialObservedAt, s.TargetRelease)
	}
	var kinds []string
	for _, m := range s.Migration {
		kinds = append(kinds, m.Kind)
	}
	if strings.Join(kinds, ",") != "schema,data,verify" || !strings.Contains(s.Migration[0].SQL, "orders_customer_idx") {
		t.Fatalf("migration=%+v", s.Migration)
	}
	if !s.Identity().complete() {
		t.Fatalf("identity incomplete: %+v", s.Identity())
	}
	again := loadStart(t)
	if again.Identity() != s.Identity() || again.EnvironmentPlanDigest != s.EnvironmentPlanDigest {
		t.Fatal("step 1 is not deterministic")
	}
}

// copyWorkspace copies the fixed workspace and rewrites the lease policy of
// the production environment (an environment plan update).
func copyWorkspace(t *testing.T, from, to string) {
	t.Helper()
	err := filepath.WalkDir(from, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(from, p)
		dst := filepath.Join(to, rel)
		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, raw, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	env := filepath.Join(to, "environments", "production", "environment.tei")
	raw, err := os.ReadFile(env)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(raw), "renew-before 5m", "renew-before 4m", 1)
	if updated == string(raw) {
		t.Fatal("lease policy line not found")
	}
	if err := os.WriteFile(env, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPlanUpdateChangesRuntimePolicy(t *testing.T) {
	root := repoRoot(t)
	ws := filepath.Join(t.TempDir(), "workspace")
	copyWorkspace(t, filepath.Join(root, DefaultWorkspace), ws)
	p := DefaultPaths(root)
	p.Workspace = ws
	updated, err := LoadStart(p)
	if err != nil {
		t.Fatal(err)
	}
	fixed := loadStart(t)
	if updated.RuntimePolicyFingerprint == fixed.RuntimePolicyFingerprint || updated.RuntimePolicyMatches || !updated.PluginLockMatches {
		t.Fatalf("runtime policy %s vs %s matches=%v", updated.RuntimePolicyFingerprint, fixed.RuntimePolicyFingerprint, updated.RuntimePolicyMatches)
	}
	if updated.InvocationID != fixed.InvocationID {
		t.Fatal("invocation identity must not depend on the plan")
	}
}

func observation(s Start, seq int64, release string, apps ...string) *Observation {
	st := dbrelease.StateSnapshot{Database: SalesDatabase, Source: dbrelease.StateFromObserved, Release: release, DataReadiness: "ready"}
	if release == s.TargetRelease {
		st.Provides = s.TargetDef.Provides
	}
	snap := deployment.Snapshot{Environment: s.Environment, Databases: []dbrelease.StateSnapshot{st}}
	for _, a := range apps {
		app, ver, _ := strings.Cut(a, "@")
		snap.ActiveApplications = append(snap.ActiveApplications, deployment.ApplicationReleaseRef{Application: core.CanonicalID(app), Version: ver})
	}
	return &Observation{Source: SourceReobserved, Seq: seq, Snapshot: snap.Normalize()}
}

func recovered(s Start, outcome pruntime.InvocationRecoveryOutcome) executor.Result {
	return AttemptFromRecovery(s, pruntime.InvocationRecoveryResult{InvocationID: s.InvocationID, Outcome: outcome,
		Handle: pruntime.DurableInvocationHandle{ActionID: s.Command.ActionID}})
}

func TestReflectNeedsReobservationAfterTheCommand(t *testing.T) {
	s := loadStart(t)
	accepted := recovered(s, pruntime.RecoveryReattached)
	cases := []struct {
		name    string
		attempt executor.Result
		obs     *Observation
		obsErr  error
		want    Arrival
		ready   bool
	}{
		{"acceptance without observation", accepted, nil, nil, ArrivalUnconfirmed, false},
		{"observation unavailable", accepted, nil, errors.New("target instance unavailable"), ArrivalUnconfirmed, false},
		{"scenario-fixed observation", accepted, FixedObservation(s.Run), nil, ArrivalUnconfirmed, false},
		{"observation before the reconciliation", accepted, observation(s, 10, "4.1.0", "order_entry@3.7.0"), nil, ArrivalUnconfirmed, false},
		{"target observed", accepted, observation(s, 11, "4.1.0", "order_entry@3.7.0"), nil, ArrivalCompleted, true},
		{"target release but 3.8.0 also active", accepted, observation(s, 11, "4.1.0", "order_entry@3.7.0", "order_entry@3.8.0"), nil, ArrivalUnresolved, false},
		{"still 4.0.0 after acceptance", accepted, observation(s, 11, "4.0.0", "order_entry@3.7.0"), nil, ArrivalUnresolved, false},
		{"unknown, still 4.0.0", recovered(s, pruntime.RecoveryWaitingObserve), observation(s, 11, "4.0.0", "order_entry@3.7.0"), nil, ArrivalUnresolved, false},
		{"lost, no observation", recovered(s, pruntime.RecoveryRuntimeLost), nil, errors.New("target instance unavailable"), ArrivalUnconfirmed, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := Reflect(s, s.Run, c.attempt, 10, c.obs, c.obsErr)
			if r.Arrival != c.want {
				t.Fatalf("arrival=%s note=%s", r.Arrival, r.ObservationNote)
			}
			if got := slices.Contains(r.ReadyActions, FollowingActionID); got != c.ready {
				t.Fatalf("following command ready=%v ready=%v", got, r.ReadyActions)
			}
			if c.want != ArrivalCompleted {
				if r.ActionStatus == string(durable.ActionCompleted) && c.want != ArrivalUnresolved {
					t.Fatalf("action completed without arrival")
				}
				if slices.Contains(r.ReadyActions, ExpectedActionID) || len(durable.ReadyCommands(r.Run)) != 0 {
					t.Fatalf("command became dispatchable again: %v", r.ReadyActions)
				}
				if r.RunStatus != string(durable.RunWaitingObservation) {
					t.Fatalf("run status %s", r.RunStatus)
				}
			}
			if r.Attempts != 1 {
				t.Fatalf("attempts=%d", r.Attempts)
			}
		})
	}
}

func TestChainVerifyAndHandover(t *testing.T) {
	s := loadStart(t)
	var c Chain
	for _, st := range []string{"step1", "step2", "step3"} {
		c.Append(st, s.Identity(), true, map[string]any{"x": st})
	}
	if err := c.Verify([]string{"step1", "step2", "step3"}); err != nil {
		t.Fatal(err)
	}
	bad := Chain{Records: append([]Record(nil), c.Records...)}
	bad.Records[1].Identity.InvocationID = "inv-other"
	if err := bad.Verify(nil); !errors.Is(err, ErrChain) {
		t.Fatalf("tampered identity accepted: %v", err)
	}
	h := Handover{CallLease: "l", TakeoverLease: "l", HandleLease: "l", CallOwner: "a", TakeoverPrevOwner: "a", CallToken: 1, TakeoverPrevToken: 1,
		TakeoverOwner: "b", HandleOwner: "b", TakeoverToken: 2, HandleToken: 2}
	if err := h.Verify(); err != nil {
		t.Fatal(err)
	}
	h.TakeoverToken, h.HandleToken = 1, 1
	if err := h.Verify(); err == nil {
		t.Fatal("non-increasing token accepted")
	}
}

func TestLeakScan(t *testing.T) {
	sens := map[string]string{"credential": "postgres://m:pw-123@127.0.0.1:5999/x", "endpoint": "127.0.0.1:5999", "empty": ""}
	if got := LeakScan(`{"resourceId":"production.sales_core.database_endpoint"}`, sens); len(got) != 0 {
		t.Fatalf("false positive %v", got)
	}
	if got := LeakScan("dial 127.0.0.1:5999 failed", sens); !slices.Equal(got, []string{"endpoint"}) {
		t.Fatalf("planted leak not found: %v", got)
	}
}
