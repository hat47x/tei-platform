package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSourceAcceptLockHasNoModelRevisionAndFailsClosed(t *testing.T) {
	root := scratchWorkspace(t)
	before := revisionOfDir(t, root)
	candidate := mustJSON(t, unitPriceCandidate(before))
	files, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer files.Close()
	first, err := acquireSourceAcceptLock(files)
	if err != nil {
		t.Fatal(err)
	}
	// Unlike an external lock manager, the marker is deliberately in the
	// Workspace. It must remain outside managed Canonical Source/Revision.
	if actual := revisionOfDir(t, root); actual != before {
		t.Fatalf("cooperative lock changed canonical revision: %s -> %s", before, actual)
	}
	secondRoot, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer secondRoot.Close()
	if second, err := acquireSourceAcceptLock(secondRoot); !errors.Is(err, errSourceAcceptBusy) || second != nil {
		t.Fatalf("second lock holder was admitted: %+v, %v", second, err)
	}
	decision, err := accept(root, candidate, acceptOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Decision != "refused" || decision.SourceChanged ||
		!hasCode(refusalCodes(decision), "SOURCE_ACCEPT_BUSY") {
		t.Fatalf("held lock did not fail closed: %+v", decision)
	}
	if actual := revisionOfDir(t, root); actual != before {
		t.Fatalf("busy refusal changed Canonical Source: %s != %s", actual, before)
	}
	if err := first.release(); err != nil {
		t.Fatal(err)
	}
	if _, err := files.Lstat(sourceAcceptLockName); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("owned lock survived normal release: %v", err)
	}
	accepted, err := accept(root, candidate, acceptOptions{})
	if err != nil || accepted.Decision != "accepted" {
		t.Fatalf("accept after lock release failed: %+v, %v", accepted, err)
	}
	if _, err := files.Lstat(sourceAcceptLockName); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Accepted operation leaked lock: %v", err)
	}
}

func TestSourceAcceptStaleLockRequiresExplicitManualRecovery(t *testing.T) {
	root := scratchWorkspace(t)
	revision := revisionOfDir(t, root)
	candidate := mustJSON(t, unitPriceCandidate(revision))
	files, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer files.Close()
	// A crashed process leaves an indistinguishable, create-only lock
	// behind. No PID/time-based takeover is permitted.
	lock, err := files.OpenFile(sourceAcceptLockName, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lock.Write([]byte("crashed process: never auto-steal\n")); err != nil {
		t.Fatal(err)
	}
	if err := lock.Close(); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		decision, err := accept(root, candidate, acceptOptions{})
		if err != nil || decision.Decision != "refused" ||
			!hasCode(refusalCodes(decision), "SOURCE_ACCEPT_BUSY") {
			t.Fatalf("stale lock was automatically stolen: %+v %v", decision, err)
		}
	}
	if actual := revisionOfDir(t, root); actual != revision {
		t.Fatalf("stale lock or refusal changed Canonical Source: %s != %s", actual, revision)
	}
	// Test-only stand-in for explicit operator decision after independently
	// confirming no acceptance process still owns the Workspace.
	if err := files.Remove(sourceAcceptLockName); err != nil {
		t.Fatal(err)
	}
	decision, err := accept(root, candidate, acceptOptions{})
	if err != nil || decision.Decision != "accepted" {
		t.Fatalf("manual recovery did not restore the common boundary: %+v %v", decision, err)
	}
}

func TestSourceAcceptGuardNeverRemovesAnotherActorLock(t *testing.T) {
	root := scratchWorkspace(t)
	files, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer files.Close()
	guard, err := acquireSourceAcceptLock(files)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate a non-cooperating actor replacing the lock inode. The guard
	// must not remove the replacement, which it does not own.
	if err := files.Remove(sourceAcceptLockName); err != nil {
		t.Fatal(err)
	}
	replacement, err := files.OpenFile(sourceAcceptLockName, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := replacement.Close(); err != nil {
		t.Fatal(err)
	}
	if err := guard.release(); err == nil {
		t.Fatal("stale owner removed an unrelated lock")
	}
	if info, err := files.Lstat(sourceAcceptLockName); err != nil || !info.Mode().IsRegular() {
		t.Fatalf("another actor's lock disappeared: %v", err)
	}
	if err := files.Remove(sourceAcceptLockName); err != nil {
		t.Fatal(err)
	}
}

func TestDesktopAcceptanceLockProcessWorker(t *testing.T) {
	if os.Getenv("TEI_ACCEPT_LOCK_PROCESS_WORKER") != "yes" {
		return
	}
	root := os.Getenv("TEI_ACCEPT_LOCK_WORKSPACE")
	ready := os.Getenv("TEI_ACCEPT_LOCK_READY")
	release := os.Getenv("TEI_ACCEPT_LOCK_RELEASE")
	if !filepath.IsAbs(root) || !filepath.IsAbs(ready) || !filepath.IsAbs(release) {
		t.Fatal("external process fixture needs absolute paths")
	}
	candidate := mustJSON(t, unitPriceCandidate(revisionOfDir(t, root)))
	beforeEachWriteHook = func(index int, edit SourceEdit) {
		if index != 0 {
			return
		}
		if err := os.WriteFile(ready, []byte("lock-held\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		deadline := time.Now().Add(30 * time.Second)
		for {
			if _, err := os.Stat(release); err == nil {
				return
			} else if !errors.Is(err, os.ErrNotExist) {
				t.Fatal(err)
			}
			if time.Now().After(deadline) {
				t.Fatal("another process never released the fixture barrier")
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
	defer func() { beforeEachWriteHook = nil }()
	decision, err := accept(root, candidate, acceptOptions{})
	if err != nil || decision.Decision != "accepted" {
		t.Fatalf("first OS process did not Accept after barrier: %+v %v", decision, err)
	}
	fmt.Println("TEI_ACCEPT_LOCK_PROCESS_DONE")
}

func TestDesktopSharedAcceptanceCrossProcessSerializesWhilePeerReviews(t *testing.T) {
	root := scratchWorkspace(t)
	revision := revisionOfDir(t, root)
	candidate := mustJSON(t, unitPriceCandidate(revision))
	barrier := t.TempDir()
	ready := filepath.Join(barrier, "ready")
	release := filepath.Join(barrier, "release")
	cmd := exec.Command(os.Args[0], "-test.run=^TestDesktopAcceptanceLockProcessWorker$", "-test.v")
	cmd.Env = append(os.Environ(),
		"TEI_ACCEPT_LOCK_PROCESS_WORKER=yes",
		"TEI_ACCEPT_LOCK_WORKSPACE="+root,
		"TEI_ACCEPT_LOCK_READY="+ready,
		"TEI_ACCEPT_LOCK_RELEASE="+release,
	)
	var output bytes.Buffer
	cmd.Stdout, cmd.Stderr = &output, &output
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()
	deadline := time.Now().Add(30 * time.Second)
	for {
		if contents, err := os.ReadFile(ready); err == nil && strings.Contains(string(contents), "lock-held") {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("first Acceptance did not hold its lock in a separate process: %s", output.String())
		}
		time.Sleep(20 * time.Millisecond)
	}
	// This second Acceptance really runs in a different OS process from the
	// first. It must refuse promptly and must not alter Source or steal lock.
	busy, err := accept(root, candidate, acceptOptions{})
	if err != nil || busy.Decision != "refused" || busy.SourceChanged ||
		!hasCode(refusalCodes(busy), "SOURCE_ACCEPT_BUSY") {
		t.Fatalf("second process bypassed first process's lock: %+v %v", busy, err)
	}
	if actual := revisionOfDir(t, root); actual != revision {
		t.Fatalf("blocked second process modified Source: %s != %s", actual, revision)
	}
	if err := os.WriteFile(release, []byte("continue\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("first process never completed common Accept: %v\n%s", err, output.String())
	}
	if !strings.Contains(output.String(), "TEI_ACCEPT_LOCK_PROCESS_DONE") {
		t.Fatalf("child did not confirm shared Acceptance: %s", output.String())
	}
	// A repeated old candidate must now be stale rather than silently
	// replaying the proposal that the first process already accepted.
	stale, err := accept(root, candidate, acceptOptions{})
	if err != nil || stale.Decision != "refused" ||
		!hasCode(refusalCodes(stale), "STALE_BASE_MODEL_REVISION") {
		t.Fatalf("second process reapplied old candidate: %+v %v", stale, err)
	}
	if _, err := os.Stat(filepath.Join(root, sourceAcceptLockName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("completed first Acceptance left a lock: %v", err)
	}
}
