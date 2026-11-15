package main

import (
	"os"
	"path/filepath"
	"testing"
)

// S2 groundwork: the entry can start the child through a boundary launcher while still
// recording the child, not the launcher, as the builder. This test uses a launcher that
// only execs its arguments, so the observable behaviour must be identical to a direct run;
// what it pins is that the recorded executable and digest stay the child's.
func TestDriveSelfBuildInKeepsTheChildAsTheRecordedBuilder(t *testing.T) {
	dir := t.TempDir()
	child := buildSelfBuildChild(t, dir)

	artifact := filepath.Join(dir, "artifact")
	state := filepath.Join(dir, "state")
	argv := []string{
		"-artifact", artifact,
		"-digest", digestOfBytes([]byte("artifact-bytes")),
		"-state", state,
	}

	direct, err := driveSelfBuild(child, argv, artifact, "")
	if err != nil {
		t.Fatalf("direct run: %v", err)
	}
	if err := os.RemoveAll(artifact); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(state); err != nil {
		t.Fatal(err)
	}

	boundary := &selfBuildBoundary{
		Name:     "test-launcher",
		Launcher: []string{"/bin/sh", "-c", `exec "$@"`, "sh"},
	}
	wrapped, err := driveSelfBuildIn(child, argv, artifact, "", boundary)
	if err != nil {
		t.Fatalf("boundary run: %v", err)
	}

	if wrapped.Invoked.Executable != direct.Invoked.Executable {
		t.Fatalf("executable = %q, want the child %q", wrapped.Invoked.Executable, direct.Invoked.Executable)
	}
	if wrapped.Invoked.ExecutableDigest != direct.Invoked.ExecutableDigest {
		t.Fatalf("digest = %q, want the child's %q", wrapped.Invoked.ExecutableDigest, direct.Invoked.ExecutableDigest)
	}
	// The argv must show what actually ran, so a reader is not told the child ran alone.
	if len(wrapped.Invoked.Argv) <= len(direct.Invoked.Argv) {
		t.Fatalf("argv = %v, want the launcher preceding the child", wrapped.Invoked.Argv)
	}
	if wrapped.Invoked.Argv[0] != "/bin/sh" {
		t.Fatalf("argv[0] = %q, want the launcher", wrapped.Invoked.Argv[0])
	}
	if wrapped.ChildArtifact != direct.ChildArtifact {
		t.Fatalf("artifact digest = %q, want %q", wrapped.ChildArtifact, direct.ChildArtifact)
	}
}

// A boundary without a launcher is a caller mistake, not a silent direct run.
func TestDriveSelfBuildInRejectsAnEmptyLauncher(t *testing.T) {
	dir := t.TempDir()
	child := buildSelfBuildChild(t, dir)
	artifact := filepath.Join(dir, "artifact")

	_, err := driveSelfBuildIn(child, []string{"-artifact", artifact}, artifact, "",
		&selfBuildBoundary{Name: "empty"})
	if err == nil {
		t.Fatal("an empty launcher should be refused")
	}
}
