package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// The test hook runs after the ordinary Preview/Review but before the
// Acceptance Source write. A real outside-Workspace directory with identical
// bytes used to fool a plain filepath.Join + ReadFile + atomic Rename write.
func TestDesktopExternalParentSymlinkSwapCannotRedirectAcceptedSource(t *testing.T) {
	root := scratchWorkspace(t)
	candidate := mustJSON(t, unitPriceCandidate(revisionOfDir(t, root)))
	ev, err := evaluate(root, candidate)
	if err != nil || !ev.preview.Acceptable {
		t.Fatalf("initial Preview failed: %+v %v", ev.preview, err)
	}
	review := buildReviewProjection(ev).ReviewFingerprint
	sales := filepath.Join(root, "semantic", "sales")
	outside := t.TempDir()
	original, err := os.ReadFile(filepath.Join(sales, "validation.tei"))
	if err != nil {
		t.Fatal(err)
	}
	outsideFile := filepath.Join(outside, "validation.tei")
	if err := os.WriteFile(outsideFile, original, 0o600); err != nil {
		t.Fatal(err)
	}
	didSwap := false
	beforeEachWriteHook = func(index int, edit SourceEdit) {
		if index != 0 {
			return
		}
		detached := filepath.Join(root, "semantic", "sales-detached")
		if err := os.Rename(sales, detached); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, sales); err != nil {
			t.Skipf("parent symlink unavailable: %v", err)
		}
		didSwap = true
	}
	defer func() { beforeEachWriteHook = nil }()
	decision, err := accept(root, candidate, acceptOptions{ExpectedReviewFingerprint: review})
	if err != nil {
		t.Fatal(err)
	}
	if !didSwap {
		t.Fatal("parent swap was not injected")
	}
	if decision.Decision != "refused" || decision.SourceChanged ||
		!hasCode(refusalCodes(decision), "SOURCE_UNREADABLE") {
		t.Fatalf("Workspace-external parent symlink bypassed Acceptance: %+v", decision)
	}
	current, err := os.ReadFile(outsideFile)
	if err != nil || !bytes.Equal(current, original) {
		t.Fatalf("Acceptance mutated Source outside Workspace: %v", err)
	}
}

func TestDesktopExternalLeafSymlinkSwapCannotRedirectAcceptedSource(t *testing.T) {
	root := scratchWorkspace(t)
	candidate := mustJSON(t, unitPriceCandidate(revisionOfDir(t, root)))
	target := filepath.Join(root, "semantic", "sales", "validation.tei")
	original, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside-validation.tei")
	if err := os.WriteFile(outside, original, 0o600); err != nil {
		t.Fatal(err)
	}
	didSwap := false
	beforeEachWriteHook = func(index int, edit SourceEdit) {
		if index != 0 {
			return
		}
		if err := os.Rename(target, target+".detached"); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, target); err != nil {
			t.Skipf("leaf symlink unavailable: %v", err)
		}
		didSwap = true
	}
	defer func() { beforeEachWriteHook = nil }()
	decision, err := accept(root, candidate, acceptOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !didSwap || decision.Decision != "refused" || decision.SourceChanged ||
		!hasCode(refusalCodes(decision), "SOURCE_UNREADABLE") {
		t.Fatalf("Source leaf symlink escaped the write boundary: %+v", decision)
	}
	current, err := os.ReadFile(outside)
	if err != nil || !bytes.Equal(current, original) {
		t.Fatalf("outside target was modified through leaf link: %v", err)
	}
}

func TestDesktopExternalNewSourceParentSymlinkRefused(t *testing.T) {
	root := scratchWorkspaceBeforeSecondApplication(t)
	candidate := mustJSON(t, secondApplicationCandidate(revisionOfDir(t, root)))
	outside := t.TempDir()
	outsideFile := filepath.Join(outside, "application.tei")
	sentinel := []byte("outside file must remain unchanged\n")
	if err := os.WriteFile(outsideFile, sentinel, 0o600); err != nil {
		t.Fatal(err)
	}
	parent := filepath.Join(root, "applications", "repair-intake")
	didSwap := false
	beforeEachWriteHook = func(index int, edit SourceEdit) {
		if index != 0 {
			return
		}
		if err := os.Symlink(outside, parent); err != nil {
			t.Skipf("new Source parent symlink unavailable: %v", err)
		}
		didSwap = true
	}
	defer func() { beforeEachWriteHook = nil }()
	decision, err := accept(root, candidate, acceptOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !didSwap || decision.Decision != "refused" || decision.SourceChanged ||
		(!hasCode(refusalCodes(decision), "SOURCE_WRITE_FAILED") &&
			!hasCode(refusalCodes(decision), "SOURCE_UNREADABLE")) {
		t.Fatalf("new Source through outside parent was not refused: %+v", decision)
	}
	current, err := os.ReadFile(outsideFile)
	if err != nil || !bytes.Equal(current, sentinel) {
		t.Fatalf("new Source escaped Workspace root: %v", err)
	}
}

// New files are published with a rooted create-only Link rather than a
// Rename that could silently overwrite an external process's new file.
func TestAcceptedSourcePublicationIsCreateOnlyAcrossGoroutines(t *testing.T) {
	root := scratchWorkspace(t)
	files, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer files.Close()
	const target = "semantic/sales/accept-concurrent-test.tei"
	contents := [][]byte{[]byte("writer A\n"), []byte("writer B\n")}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, content := range contents {
		wg.Add(1)
		go func(data []byte) {
			defer wg.Done()
			results <- sourceWriteWithinRoot(files, target, data, true)
		}(content)
	}
	wg.Wait()
	close(results)
	success, exists := 0, 0
	for err := range results {
		switch {
		case err == nil:
			success++
		case errors.Is(err, os.ErrExist):
			exists++
		default:
			t.Fatalf("unexpected result of concurrent create: %v", err)
		}
	}
	if success != 1 || exists != 1 {
		t.Fatalf("create-only Source publication was not exclusive: success=%d exists=%d", success, exists)
	}
	contentsNow, err := files.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(contentsNow, contents[0]) && !bytes.Equal(contentsNow, contents[1]) {
		t.Fatalf("published file was not from one complete writer: %q", contentsNow)
	}
}

func TestDesktopNamedWorkspaceRootSwapRefusesBeforeSourceWrite(t *testing.T) {
	root := scratchWorkspace(t)
	candidate := mustJSON(t, unitPriceCandidate(revisionOfDir(t, root)))
	ev, err := evaluate(root, candidate)
	if err != nil || !ev.preview.Acceptable {
		t.Fatalf("initial shared Preview failed: %+v %v", ev.preview, err)
	}
	review := buildReviewProjection(ev).ReviewFingerprint
	outside := t.TempDir()
	original, err := os.ReadFile(filepath.Join(root, "semantic", "sales", "validation.tei"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(outside, "semantic", "sales"), 0o755); err != nil {
		t.Fatal(err)
	}
	outsideFile := filepath.Join(outside, "semantic", "sales", "validation.tei")
	if err := os.WriteFile(outsideFile, original, 0o600); err != nil {
		t.Fatal(err)
	}
	didSwap := false
	beforeEachWriteHook = func(index int, edit SourceEdit) {
		if index != 0 {
			return
		}
		if err := os.Rename(root, root+"-detached"); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, root); err != nil {
			t.Skipf("Workspace root symlink unavailable: %v", err)
		}
		didSwap = true
	}
	defer func() { beforeEachWriteHook = nil }()
	decision, err := accept(root, candidate, acceptOptions{ExpectedReviewFingerprint: review})
	if err != nil {
		t.Fatal(err)
	}
	if !didSwap || decision.Decision != "refused" || decision.SourceChanged ||
		!hasCode(refusalCodes(decision), "SOURCE_WORKSPACE_REPLACED") {
		t.Fatalf("replaced Workspace identity was not rejected: %+v", decision)
	}
	now, err := os.ReadFile(outsideFile)
	if err != nil || !bytes.Equal(now, original) {
		t.Fatalf("outside replacement Workspace was modified: %v", err)
	}
}
