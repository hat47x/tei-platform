//go:build !tei_repair_only

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
)

// This file builds the real Shared Acceptance and evidence audit CLIs once
// per test process instead of paying `go run` (compile, link, first launch)
// on every call. The exe is the same tool: plain `go build`, no extra flags,
// no mock. Nothing is cached across processes, so an edited CLI source is
// always rebuilt by the next `go test`. Build failures are stored and every
// caller fails with the `go build` output.

var (
	toolDirOnce sync.Once
	toolDirPath string
	toolDirErr  error
)

// toolDir is the process-lifetime directory that holds the built exes. It is
// created lazily (so helper re-exec children never create one) and removed by
// TestMain.
func toolDir() (string, error) {
	toolDirOnce.Do(func() {
		toolDirPath, toolDirErr = os.MkdirTemp("", "tei-web-target-tools-")
	})
	return toolDirPath, toolDirErr
}

type toolExe struct {
	pkg    string // package path relative to the repository root, e.g. ./tools/x
	base   string // exe base name
	once   sync.Once
	path   string
	err    error
	builds atomic.Int32 // test-only: how many `go build` runs happened
}

var (
	sharedAcceptanceTool = &toolExe{pkg: "./tools/reference-shared-acceptance", base: "reference-shared-acceptance"}
	evidenceAuditTool    = &toolExe{pkg: "./tools/reference-critical-path-evidence-audit", base: "reference-critical-path-evidence-audit"}
)

func toolRepoRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("cannot resolve test source path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..")), nil
}

// resolve builds the tool on first use and returns the same path or the same
// error to every later caller.
func (e *toolExe) resolve() (string, error) {
	e.once.Do(func() {
		dir, err := toolDir()
		if err != nil {
			e.err = fmt.Errorf("create tool dir: %w", err)
			return
		}
		root, err := toolRepoRoot()
		if err != nil {
			e.err = err
			return
		}
		name := e.base
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		out := filepath.Join(dir, name)
		cmd := exec.Command("go", "build", "-o", out, e.pkg)
		cmd.Dir = root
		e.builds.Add(1)
		if output, err := cmd.CombinedOutput(); err != nil {
			e.err = fmt.Errorf("go build %s: %v\n%s", e.pkg, err, output)
			return
		}
		e.path = out
	})
	return e.path, e.err
}

func (e *toolExe) mustResolve(t *testing.T) string {
	t.Helper()
	path, err := e.resolve()
	if err != nil {
		t.Fatalf("%v", err)
	}
	return path
}

func sharedAcceptanceExe(t *testing.T) string { t.Helper(); return sharedAcceptanceTool.mustResolve(t) }
func evidenceAuditExe(t *testing.T) string    { t.Helper(); return evidenceAuditTool.mustResolve(t) }

func TestMain(m *testing.M) {
	code := m.Run()
	if toolDirPath != "" {
		os.RemoveAll(toolDirPath)
	}
	os.Exit(code)
}

func TestToolExeBuildsOncePerProcess(t *testing.T) {
	first := sharedAcceptanceExe(t)
	second := sharedAcceptanceExe(t)
	if first != second {
		t.Fatalf("path changed between uses: %s vs %s", first, second)
	}
	if n := sharedAcceptanceTool.builds.Load(); n != 1 {
		t.Fatalf("shared acceptance built %d times, want 1", n)
	}
	if _, err := os.Stat(first); err != nil {
		t.Fatal(err)
	}
}

func TestToolExeBadInvocationSurfacesToolOutput(t *testing.T) {
	exe := sharedAcceptanceExe(t)
	output, err := exec.Command(exe, "no-such-subcommand").CombinedOutput()
	if err == nil {
		t.Fatalf("expected a failing invocation, output=%s", output)
	}
	if len(output) == 0 {
		t.Fatal("tool failure carried no output")
	}
}

func TestToolExeBuildFailureIsStoredForEveryCaller(t *testing.T) {
	broken := &toolExe{pkg: "./tools/this-tool-does-not-exist", base: "broken"}
	_, err1 := broken.resolve()
	_, err2 := broken.resolve()
	if err1 == nil || err2 == nil {
		t.Fatalf("broken build must fail for every caller: %v %v", err1, err2)
	}
	if err1.Error() != err2.Error() {
		t.Fatalf("later caller saw a different error:\n%v\n%v", err1, err2)
	}
	if n := broken.builds.Load(); n != 1 {
		t.Fatalf("broken build attempted %d times, want 1", n)
	}
	if msg := err1.Error(); len(msg) < len("go build ")+len(broken.pkg) {
		t.Fatalf("error lacks go build output: %q", msg)
	}
}
