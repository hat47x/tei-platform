package ordermultilingualoperation

import (
	"context"
	"encoding/json"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func newTestChain(t *testing.T) *Chain {
	t.Helper()
	return NewChain(repoRoot(t), t.TempDir())
}

func runStagesOneToFour(t *testing.T, c *Chain) {
	t.Helper()
	ctx := context.Background()
	steps := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"stage1", c.Stage1}, {"stage2", c.Stage2}, {"stage3", c.Stage3},
		{"identity", c.IdentityChecks}, {"tamper", c.FaultBundleTampering}, {"pdb02", c.FaultPDB02}, {"stage4", c.Stage4},
	}
	for _, s := range steps {
		if err := s.fn(ctx); err != nil {
			t.Fatalf("%s: %v", s.name, err)
		}
	}
}

func dump(v any) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

// Stages 1-4 need neither a database nor Node, so they run everywhere. The
// through test repeats them as part of the one chain.
func TestStagesOneToFourWithoutDatabase(t *testing.T) {
	c := newTestChain(t)
	runStagesOneToFour(t, c)
	for _, f := range c.Rec.Faults {
		if !f.Passed {
			t.Errorf("fault %s did not pass: %s", f.ID, dump(f.Observed))
		}
	}
	for _, ch := range c.Rec.Completion {
		if !ch.Passed {
			t.Errorf("check %s did not pass: %s", ch.ID, dump(ch.Observed))
		}
	}
	for _, r := range c.Matrix {
		if !r.Agrees {
			t.Errorf("matrix row %s %s %s %s: before=%s after=%s expected=%s rule=%s (%s)", r.Kind, r.Target, r.Locale, r.Provider,
				r.Before.Status, r.After.Status, r.Expected, r.RuleAfter, r.After.Diagnostic)
		}
	}
	if c.B1.Digest == "" || c.B1.Digest == c.B2.Digest {
		t.Fatalf("bundle digests: %s %s", c.B1.Digest, c.B2.Digest)
	}
}

func TestCheckBundleAgainstSourcesListsOnlyRevisedDisplays(t *testing.T) {
	c := newTestChain(t)
	runStagesOneToFour(t, c)
	stale := c.U.CheckBundleAgainstSources(c.B1, c.Rel2)
	want := "en-US sales.order|en-US sales.order.submit|ja-JP sales.order|ja-JP sales.order.submit"
	if strings.Join(stale, "|") != want {
		t.Fatalf("stale displays: %v", stale)
	}
	if len(c.U.CheckBundleAgainstSources(c.B2, c.Rel2)) != 0 {
		t.Fatal("regenerated bundle reported stale")
	}
}

func TestPageDefectsCountsLossAndDuplicates(t *testing.T) {
	full := []string{"a", "b", "c", "d"}
	if d, m := PageDefects(full, []string{"a", "b", "c", "d"}); d != 0 || m != 0 {
		t.Fatalf("clean pages: %d %d", d, m)
	}
	if d, m := PageDefects(full, []string{"a", "b", "b", "d"}); d != 1 || m != 1 {
		t.Fatalf("defective pages: %d %d", d, m)
	}
}

func TestRenderGoSelectsPluralAndGroups(t *testing.T) {
	v := map[string]string{"one": "Confirmed {count} order.", "other": "Confirmed {count} orders."}
	for _, tc := range []struct {
		locale string
		n      int64
		want   string
	}{{"en-US", 1, "Confirmed 1 order."}, {"en-US", 0, "Confirmed 0 orders."}, {"en-US", 12345, "Confirmed 12,345 orders."},
		{"ja-JP", 1, "Confirmed 1 orders."}} {
		r, err := RenderGo(RenderCase{ID: "x", Locale: tc.locale, Count: tc.n, Variants: v})
		if err != nil || r.Text != tc.want {
			t.Fatalf("%s %d: %+v %v", tc.locale, tc.n, r, err)
		}
	}
	if _, err := RenderGo(RenderCase{Locale: "en-US", Count: -1, Variants: v}); err == nil {
		t.Fatal("negative count accepted")
	}
}

func TestVerifyThroughNeedsEveryStageAndOneIdentity(t *testing.T) {
	r := Record{Unified: DefaultUnified()}
	for st := 1; st <= 6; st++ {
		out := map[string]any{}
		switch st {
		case 3:
			out["bundleDigest"] = "sha256:a"
		case 4:
			out["bundleBefore"], out["bundleAfter"] = "sha256:a", "sha256:b"
		case 6:
			out["bundleDigest"] = "sha256:b"
		}
		rel := "r1"
		if st >= 4 {
			rel = "r2"
		}
		r.AddStage(StageRecord{Stage: st, LockFP: "L", Release: rel, Outputs: out})
	}
	if res := VerifyThrough(r); !res.Complete {
		t.Fatalf("complete chain refused: %+v", res)
	}
	missing := r
	missing.Stages = append(append([]StageRecord(nil), r.Stages[:4]...), r.Stages[5:]...)
	if res := VerifyThrough(missing); res.Complete || len(res.MissingStages) != 1 || res.MissingStages[0] != 5 {
		t.Fatalf("missing stage accepted: %+v", res)
	}
	swapped := r
	swapped.Stages = append([]StageRecord(nil), r.Stages...)
	swapped.Stages[2].Owner = "std.order_entry@1.2.0"
	if res := VerifyThrough(swapped); res.Complete {
		t.Fatal("foreign owner accepted")
	}
	relocked := r
	relocked.Stages = append([]StageRecord(nil), r.Stages...)
	relocked.Stages[3].LockFP = "M"
	relocked.Stages[3].Digest = stageDigest(relocked.Stages[3])
	if res := VerifyThrough(relocked); res.Complete {
		t.Fatal("different lock in stage 4 accepted")
	}
}
