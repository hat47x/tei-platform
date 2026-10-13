package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"tei/internal/desktophost"
)

// This test-only subprocess stands in for an external editor/Git process.
// It is deliberately not a Shared Acceptance actor and never invokes Accept.
func TestDesktopExternalSourceChangeWorker(t *testing.T) {
	if os.Getenv("TEI_DESKTOP_EXTERNAL_SOURCE_WORKER") != "1" {
		return
	}
	target := os.Getenv("TEI_DESKTOP_EXTERNAL_SOURCE_TARGET")
	if !filepath.IsAbs(target) {
		t.Fatal("external editor target must be absolute")
	}
	original, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, append(original, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	fmt.Printf("EXTERNAL_SOURCE_WRITER_PID=%d\n", os.Getpid())
}

func externalProcessEditsSource(t *testing.T, target string) []byte {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestDesktopExternalSourceChangeWorker$", "-test.v")
	cmd.Env = append(os.Environ(),
		"TEI_DESKTOP_EXTERNAL_SOURCE_WORKER=1",
		"TEI_DESKTOP_EXTERNAL_SOURCE_TARGET="+target)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("external editor process failed: %v\n%s", err, output)
	}
	tag := "EXTERNAL_SOURCE_WRITER_PID="
	at := strings.Index(string(output), tag)
	if at < 0 {
		t.Fatalf("external editor did not report its process: %s", output)
	}
	pidText := strings.Fields(string(output)[at+len(tag):])
	if len(pidText) == 0 {
		t.Fatalf("external editor response has no PID: %s", output)
	}
	pid, err := strconv.Atoi(pidText[0])
	if err != nil || pid == os.Getpid() {
		t.Fatalf("external editor was not a distinct OS process: %q %v", pidText[0], err)
	}
	changed, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	return changed
}

func TestDesktopExternalProcessEditAfterReviewRefusesStaleAccept(t *testing.T) {
	root := scratchWorkspace(t)
	observed, err := desktophost.ObserveSource(root)
	if err != nil {
		t.Fatal(err)
	}
	candidate := mustJSON(t, unitPriceCandidate(revisionOfDir(t, root)))
	ev, err := evaluate(root, candidate)
	if err != nil || !ev.preview.Acceptable {
		t.Fatalf("initial shared Preview failed: %+v %v", ev.preview, err)
	}
	reviewFP := buildReviewProjection(ev).ReviewFingerprint
	target := filepath.Join(root, "semantic", "sales", "validation.tei")
	external := externalProcessEditsSource(t, target)
	if _, err := desktophost.CheckSourceFresh(observed); !errors.Is(err, desktophost.ErrSourceChanged) {
		t.Fatalf("Host missed externally written bytes without a watcher: %v", err)
	}
	decision, err := accept(root, candidate, acceptOptions{ExpectedReviewFingerprint: reviewFP})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Decision != "refused" || decision.SourceChanged ||
		!hasCode(refusalCodes(decision), "STALE_BASE_MODEL_REVISION") {
		t.Fatalf("Shared Acceptance accepted a stale candidate: %+v", decision)
	}
	current, err := os.ReadFile(target)
	if err != nil || string(current) != string(external) {
		t.Fatalf("stale refusal destroyed another process's edit: %v", err)
	}
}

func TestDesktopExternalProcessEditInsideAcceptWriteWindowRefusesAndRollsBack(t *testing.T) {
	root := scratchWorkspace(t)
	observed, err := desktophost.ObserveSource(root)
	if err != nil {
		t.Fatal(err)
	}
	c := unitPriceCandidate(revisionOfDir(t, root))
	c.Delta = append(c.Delta, Operation{
		Op: opAddValidation, Namespace: "sales", ID: "sales.order.total_positive",
		Kind: "positive", Attribute: "sales.order.total",
		Placement: "semantic/sales/order.tei",
	})
	candidate := mustJSON(t, c)
	ev, err := evaluate(root, candidate)
	if err != nil || !ev.preview.Acceptable {
		t.Fatalf("shared Preview failed: %+v %v", ev.preview, err)
	}
	if len(ev.edits) != 2 || ev.edits[0].Path != "semantic/sales/order.tei" ||
		ev.edits[1].Path != "semantic/sales/validation.tei" {
		t.Fatalf("scratch fixture no longer has the expected two Source edits: %+v", ev.edits)
	}
	reviewFP := buildReviewProjection(ev).ReviewFingerprint
	first := filepath.Join(root, "semantic", "sales", "order.tei")
	second := filepath.Join(root, "semantic", "sales", "validation.tei")
	beforeFirst, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	var external []byte
	beforeEachWriteHook = func(index int, edit SourceEdit) {
		if index == 1 {
			external = externalProcessEditsSource(t, second)
		}
	}
	defer func() { beforeEachWriteHook = nil }()
	decision, err := accept(root, candidate, acceptOptions{ExpectedReviewFingerprint: reviewFP})
	if err != nil {
		t.Fatal(err)
	}
	if len(external) == 0 {
		t.Fatal("the separate editor process never changed the Source")
	}
	if decision.Decision != "refused" || decision.SourceChanged ||
		!hasCode(refusalCodes(decision), "SOURCE_CHANGED_DURING_ACCEPT") {
		t.Fatalf("Shared Acceptance passed the real external edit: %+v", decision)
	}
	afterFirst, err := os.ReadFile(first)
	if err != nil || string(afterFirst) != string(beforeFirst) {
		t.Fatalf("partial first write was not restored: %v", err)
	}
	afterSecond, err := os.ReadFile(second)
	if err != nil || string(afterSecond) != string(external) {
		t.Fatalf("rollback overwrote the other editor's committed change: %v", err)
	}
	if _, err := desktophost.CheckSourceFresh(observed); !errors.Is(err, desktophost.ErrSourceChanged) {
		t.Fatalf("Host did not observe the post-refusal external Source: %v", err)
	}
}
