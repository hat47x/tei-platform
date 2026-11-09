package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// The entry builds the child's argv from its own flags plus whatever the caller
// passes through -child-arg. Three of those flags are the entry's own contract
// with the child: -artifact-out, -state and -supply. A caller that repeats them
// in -child-arg would have two sources for one value, and which one the child
// obeys depends on the child's flag parser (Go's takes the last occurrence; some
// parsers take the first). Either way the entry would record one value while the
// child used another, or silently drop what the caller asked for.
//
// This is the same failure the read-only command allowlist already refuses for
// its own flags: silent loss of caller intent.

func TestSelfBuildOuterRefusesReservedFlagsInChildArg(t *testing.T) {
	dir := t.TempDir()
	child := buildSelfBuildChild(t, dir)
	workspace := outerWorkspace(t, dir)
	output := filepath.Join(dir, "artifact")

	for _, reserved := range []string{"-artifact-out", "-state", "-supply"} {
		args := childArgs(t, reserved, filepath.Join(dir, "elsewhere"))
		// The refusal happens before the attempt produces a report, so the
		// error comes back from the command rather than from the record.
		var out bytes.Buffer
		code, err := runSelfBuildCommand([]string{
			"-child", child,
			"-workspace", workspace,
			"-intent-file", outerIntentFile(t, dir),
			"-supply", filepath.Join(dir, "supply"),
			"-state", filepath.Join(dir, "state"),
			"-output", output,
			"-child-arg", args,
		}, &out)
		if code != 2 {
			t.Fatalf("%s in -child-arg was accepted: code=%d out=%s", reserved, code, out.String())
		}
		if err == nil {
			t.Fatalf("%s in -child-arg produced no stated reason", reserved)
		}
		if !bytes.Contains([]byte(err.Error()), []byte(reserved)) {
			t.Fatalf("%s was refused without naming it: %v", reserved, err)
		}
		if _, statErr := os.Stat(output); statErr == nil {
			t.Fatalf("%s in -child-arg still produced an artifact", reserved)
		}
	}
}

// A -child-arg that does not touch the entry's own flags still works.
func TestSelfBuildOuterAllowsOtherChildArgs(t *testing.T) {
	dir := t.TempDir()
	child := buildSelfBuildChild(t, dir)
	workspace := outerWorkspace(t, dir)
	output := filepath.Join(dir, "artifact")
	if err := os.WriteFile(output, []byte("artifact-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	digest, err := digestOfFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(output); err != nil {
		t.Fatal(err)
	}

	report, code := runOuterShape(t,
		"-child", child,
		"-workspace", workspace,
		"-intent-file", outerIntentFile(t, dir),
		"-state", filepath.Join(dir, "state"),
		"-output", output,
		"-child-arg", childArgs(t, "-artifact", output, "-digest", digest))
	if code != 0 {
		t.Fatalf("an ordinary -child-arg was refused: code=%d %+v", code, report.Error)
	}
	if !report.OK {
		t.Fatalf("the attempt did not report ok: %+v", report)
	}
}
