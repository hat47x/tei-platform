package digestchangereuse

import (
	"context"
	"strings"
	"testing"
	"time"

	"tei/core/generation"
	"tei/internal/composition"
)

// The three questions CON-12c needs, answered on the layer that carries the
// digest. If any of these fail, the pinning story is broken.
func TestTheDigestChangeIsObservedAtEachStage(t *testing.T) {
	report, err := Run()
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	if len(report.Outcomes) != 3 {
		t.Fatalf("expected 3 outcomes, got %d", len(report.Outcomes))
	}

	// Before the change, the lease carries the old digest.
	if report.Outcomes[0].Digest != OldDigest {
		t.Fatalf("before the change the digest was %q", report.Outcomes[0].Digest)
	}

	// A new acquisition after the change must carry the NEW digest. If it still
	// carries the old one, the change did not take effect for new work.
	if !report.NewLeaseUsesNewDigest {
		t.Fatalf("a new acquisition kept the old digest: %+v", report.Outcomes[1])
	}

	// A lease pinned before the change must keep the OLD digest. If it moved to
	// the new one, an in-flight call would silently switch artifacts.
	if !report.OldLeaseReused {
		t.Fatalf("a pinned lease moved to the new digest: %+v", report.Outcomes[2])
	}
}

// The report must name which digest each stage carries, so a reader is not
// left comparing hex strings.
func TestEachOutcomeNamesWhichDigestItHolds(t *testing.T) {
	report, err := Run()
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	want := []string{"the OLD digest", "the NEW digest", "the OLD digest"}
	for i, w := range want {
		if report.Outcomes[i].Matches != w {
			t.Fatalf("outcome %d says %q, want %q", i, report.Outcomes[i].Matches, w)
		}
	}
}

// Publish refuses a binding that keeps the same generation number, so the
// change must be a real generation step. This pins the precondition the
// reference relies on.
func TestPublishRefusesTheSameGeneration(t *testing.T) {
	old := Binding(1, OldDigest)
	router, err := generation.NewRouter(old, composition.NewScope())
	if err != nil {
		t.Fatalf("create router: %v", err)
	}
	same := Binding(1, NewDigest)
	if _, err := router.Publish(1, same, composition.NewScope(),
		func(generation.Binding) error { return nil }); err == nil {
		t.Fatal("publishing the same generation number was accepted")
	}
}

// Publish refuses a different contract, so a digest change alone cannot be used
// to swap providers across contracts.
func TestPublishRefusesADifferentContract(t *testing.T) {
	old := Binding(1, OldDigest)
	router, err := generation.NewRouter(old, composition.NewScope())
	if err != nil {
		t.Fatalf("create router: %v", err)
	}
	next := Binding(2, NewDigest)
	next.ContractID = "sales.other"
	if _, err := router.Publish(1, next, composition.NewScope(),
		func(generation.Binding) error { return nil }); err == nil {
		t.Fatal("publishing across contracts was accepted")
	}
}

// After the old lease releases, the drained channel closes. This is what makes
// "the old artifact is still in use" observable rather than assumed.
func TestTheOldGenerationDrainsAfterRelease(t *testing.T) {
	old := Binding(1, OldDigest)
	router, err := generation.NewRouter(old, composition.NewScope())
	if err != nil {
		t.Fatalf("create router: %v", err)
	}
	lease, err := router.Acquire(func(generation.Binding) error { return nil })
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	retired, err := router.Publish(1, Binding(2, NewDigest), composition.NewScope(),
		func(generation.Binding) error { return nil })
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	// While the lease is still held, Drain must time out rather than dispose the
	// generation. A timeout preserves the generation's effects for recovery; it
	// is not evidence of a successful cleanup.
	shortCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := retired.Drain(shortCtx); err == nil {
		t.Fatal("the old generation drained while a lease was still held")
	}

	lease.Release()
	drainCtx, drainCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer drainCancel()
	if err := retired.Drain(drainCtx); err != nil {
		t.Fatalf("the old generation did not drain after the last lease released: %v", err)
	}
}

func TestReportStatesAuthorityAndItsLimits(t *testing.T) {
	report, err := Run()
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if report.Authority != Authority || report.Schema != Schema || report.RuleFingerprint == "" {
		t.Fatalf("bad header: %+v", report)
	}
	found := false
	for _, note := range report.Notes {
		if strings.Contains(note, "no real cutover") {
			found = true
		}
	}
	if !found {
		t.Fatal("the notes do not state that no real cutover happened")
	}
}
