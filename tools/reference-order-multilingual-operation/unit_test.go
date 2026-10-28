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
