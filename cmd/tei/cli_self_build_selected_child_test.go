package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A caller-selected builder must not be replaced with another executable.
// This is a pre-execution pin: the checked bytes can still change between the
// hash and the OS loader, and no BuildIntent/Provider selection is implied.
func TestSelfBuildSelectedChildPinRefusesMismatchBeforeRunning(t *testing.T) {
	dir := t.TempDir()
	child := buildSelfBuildChild(t, dir)
	pin, err := digestOfFile(child)
	if err != nil {
		t.Fatal(err)
	}
	artifact := filepath.Join(dir, "must-not-exist")

	t.Run("selected B actual A", func(t *testing.T) {
		// A regular file that is not the selected child is sufficient:
		// a successful pin check must never try to execute it.
		replacement := filepath.Join(dir, "unexpected-child")
		if err := os.WriteFile(replacement, []byte("different builder bytes"), 0o600); err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		code, err := runSelfBuildCommand([]string{
			"-child", replacement,
			"-expected-child-digest", pin,
			"-child-arg", childArgs(t, "-artifact", artifact, "-digest", "sha256:wrong"),
			"-artifact", artifact,
		}, &out)
		if code != 1 || err == nil || !strings.Contains(err.Error(), "selected child builder digest mismatch") {
			t.Fatalf("a different executable was not refused before launch: code=%d err=%v", code, err)
		}
		if out.Len() != 0 {
			t.Fatalf("refused builder emitted a success receipt: %s", out.String())
		}
		if _, err := os.Stat(artifact); !os.IsNotExist(err) {
			t.Fatalf("refused builder wrote the artifact or it is inaccessible: %v", err)
		}
	})

	t.Run("invalid pin", func(t *testing.T) {
		code, err := runSelfBuildCommand([]string{
			"-child", child,
			"-expected-child-digest", "sha256:ABC",
			"-child-arg", childArgs(t, "-artifact", artifact, "-digest", "sha256:wrong"),
			"-artifact", artifact,
		}, &bytes.Buffer{})
		if code != 2 || err == nil || !strings.Contains(err.Error(), "INVALID_ARGUMENT") {
			t.Fatalf("invalid pin was accepted: code=%d err=%v", code, err)
		}
		if _, err := os.Stat(artifact); !os.IsNotExist(err) {
			t.Fatalf("invalid pin wrote the artifact: %v", err)
		}
	})

	t.Run("matching pin", func(t *testing.T) {
		var out bytes.Buffer
		code, err := runSelfBuildCommand([]string{
			"-child", child,
			"-expected-child-digest", pin,
			"-child-arg", childArgs(t, "-artifact", artifact, "-digest", "sha256:wrong"),
			"-artifact", artifact,
		}, &out)
		if code != 0 || err != nil {
			t.Fatalf("the selected executable did not start: code=%d err=%v", code, err)
		}
		var receipt selfBuildEntryEvidence
		if err := json.Unmarshal(out.Bytes(), &receipt); err != nil {
			t.Fatal(err)
		}
		if !receipt.ChildPinChecked || !receipt.ChildPinMatched ||
			receipt.SelectedChildDigest != pin || receipt.Invoked.ExecutableDigest != pin ||
			receipt.Invoked.ExitCode != 0 || receipt.L3Claimed {
			t.Fatalf("selected/actual builder was not bound without an L3 claim: %+v", receipt)
		}
		if _, err := os.Stat(artifact); err != nil {
			t.Fatalf("matching builder did not create artifact: %v", err)
		}
		// The test fixture intentionally reports a wrong artifact digest.
		// A matched builder pin must not turn that into a passing byte check.
		if receipt.CrossCheckOK || receipt.FailureCause == "" {
			t.Fatalf("builder pin was incorrectly treated as artifact correctness: %+v", receipt)
		}
	})
}

// A boundary launcher does not override the selected child identity. If the
// child image differs, even a configured launcher must never be started.
func TestSelfBuildPinnedBoundaryLauncherRefusesBeforeExecution(t *testing.T) {
	dir := t.TempDir()
	selected := buildSelfBuildChild(t, dir)
	pin, err := digestOfFile(selected)
	if err != nil {
		t.Fatal(err)
	}
	actual := filepath.Join(dir, "unexpected-child")
	if err := os.WriteFile(actual, []byte("not the selected child"), 0o600); err != nil {
		t.Fatal(err)
	}
	artifact := filepath.Join(dir, "no-boundary-artifact")
	boundary := &selfBuildBoundary{
		Name:     "must-not-start",
		Launcher: []string{filepath.Join(dir, "missing-launcher")},
	}
	_, err = driveSelfBuildIn(actual, nil, artifact, "", boundary, pin)
	if err == nil || !strings.Contains(err.Error(), "selected child builder digest mismatch") {
		t.Fatalf("boundary launcher was allowed to bypass child pin: %v", err)
	}
	if _, err := os.Stat(artifact); !os.IsNotExist(err) {
		t.Fatalf("pin mismatch wrote output through boundary: %v", err)
	}
}
