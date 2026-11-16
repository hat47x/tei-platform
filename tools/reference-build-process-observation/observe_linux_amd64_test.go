//go:build linux && amd64

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// The test binary doubles as the observed command: with TEI_OBSERVE_HELPER set
// it opens the files it is told to and starts the programs it is told to.
func TestMain(m *testing.M) {
	switch os.Getenv("TEI_OBSERVE_HELPER") {
	case "":
		os.Exit(m.Run())
	case "parent":
		// Reads one file, then starts two children one after the other.
		readAll(os.Getenv("TEI_OBSERVE_READ"))
		for _, role := range []string{"first", "second"} {
			cmd := exec.Command(os.Args[0])
			cmd.Env = append(os.Environ(), "TEI_OBSERVE_HELPER="+role)
			cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
			if err := cmd.Run(); err != nil {
				os.Exit(3)
			}
		}
		os.Exit(7)
	case "first":
		_ = os.WriteFile(filepath.Join(os.Getenv("TEI_OBSERVE_DIR"), "first.out"), []byte("1"), 0o644)
		// A grandchild: it must be attributed to the first step.
		if tool := os.Getenv("TEI_OBSERVE_TOOL"); tool != "" {
			_ = exec.Command(tool).Run()
		}
		os.Exit(0)
	case "second":
		readAll(filepath.Join(os.Getenv("TEI_OBSERVE_DIR"), "first.out"))
		_, _ = os.Open(filepath.Join(os.Getenv("TEI_OBSERVE_DIR"), "missing"))
		os.Exit(0)
	}
}

func readAll(path string) {
	if path != "" {
		_, _ = os.ReadFile(path)
	}
}

func fileDigest(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func TestObserveRecordsExecsStepsAndOpens(t *testing.T) {
	tool, err := exec.LookPath("true")
	if err != nil {
		t.Skipf("no `true` program on this host: %v", err)
	}
	tool, err = filepath.EvalSymlinks(tool)
	if err != nil {
		t.Fatal(err)
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	input := filepath.Join(dir, "input.txt")
	if err := os.WriteFile(input, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TEI_OBSERVE_HELPER", "parent")
	t.Setenv("TEI_OBSERVE_DIR", dir)
	t.Setenv("TEI_OBSERVE_READ", input)
	t.Setenv("TEI_OBSERVE_TOOL", tool)
	out := filepath.Join(dir, "record.json")
	code, err := run(out, true, []string{self})
	if err != nil {
		t.Fatal(err)
	}
	// The observed command's own exit code comes back unchanged.
	if code != 7 {
		t.Fatalf("exit code = %d, want 7", code)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var record Record
	if err := json.Unmarshal(raw, &record); err != nil {
		t.Fatal(err)
	}
	if record.ExitCode != 7 || !record.FilesTraced || len(record.NotObserved) == 0 {
		t.Fatalf("record header: exit %d traced %v notObserved %d", record.ExitCode, record.FilesTraced, len(record.NotObserved))
	}

	// Four execs, in order: the command, the first child, its tool, the second child.
	selfDigest, toolDigest := fileDigest(t, self), fileDigest(t, tool)
	want := []struct {
		parentSeq, step int
		digest          string
	}{
		{-1, 0, selfDigest},
		{0, 1, selfDigest},
		{1, 1, toolDigest},
		{0, 3, selfDigest},
	}
	if len(record.Execs) != len(want) {
		t.Fatalf("execs = %d, want %d: %+v", len(record.Execs), len(want), record.Execs)
	}
	for i, w := range want {
		e := record.Execs[i]
		if e.Seq != i || e.ParentSeq != w.parentSeq || e.Step != w.step || e.ImageDigest != w.digest {
			t.Errorf("exec %d = seq %d parent %d step %d digest %s; want parent %d step %d digest %s",
				i, e.Seq, e.ParentSeq, e.Step, e.ImageDigest, w.parentSeq, w.step, w.digest)
		}
	}
	if len(record.Execs[1].EnvNames) == 0 || record.Execs[1].Argv[0] != self {
		t.Errorf("the first child's argv/env were not recorded: %+v", record.Execs[1])
	}

	type key struct {
		path string
		step int
	}
	uses := map[key]FileUse{}
	for _, use := range record.Files {
		k := key{use.Path, use.Step}
		merged := uses[k]
		merged.Read += use.Read
		merged.Write += use.Write
		merged.Failed += use.Failed
		uses[k] = merged
	}
	firstOut := filepath.Join(dir, "first.out")
	if u := uses[key{input, 0}]; u.Read != 1 || u.Write != 0 {
		t.Errorf("the command's read of the input: %+v", u)
	}
	if u := uses[key{firstOut, 1}]; u.Write != 1 {
		t.Errorf("the first step's write: %+v", u)
	}
	// The second step read what the first step wrote: this is the hand-over
	// (or the leak) a reconciler has to be able to see.
	if u := uses[key{firstOut, 3}]; u.Read != 1 || u.Write != 0 {
		t.Errorf("the second step's read of the first step's output: %+v", u)
	}
	if u := uses[key{filepath.Join(dir, "missing"), 3}]; u.Failed != 1 || u.Read != 0 {
		t.Errorf("the failed open: %+v", u)
	}
	if _, leaked := uses[key{firstOut, 0}]; leaked {
		t.Errorf("the command was credited with an open only its steps made")
	}
}

func TestExecOnlyObservationRecordsNoPaths(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	t.Setenv("TEI_OBSERVE_HELPER", "parent")
	t.Setenv("TEI_OBSERVE_DIR", dir)
	out := filepath.Join(dir, "record.json")
	if _, err := run(out, false, []string{self}); err != nil {
		t.Fatal(err)
	}
	var record Record
	raw, _ := os.ReadFile(out)
	if err := json.Unmarshal(raw, &record); err != nil {
		t.Fatal(err)
	}
	if record.FilesTraced || len(record.Files) != 0 || len(record.Execs) != 3 {
		t.Fatalf("traced %v files %d execs %d", record.FilesTraced, len(record.Files), len(record.Execs))
	}
	found := false
	for _, note := range record.NotObserved {
		if note == "opened paths (run with -files)" {
			found = true
		}
	}
	if !found {
		t.Fatalf("an exec-only record did not say paths were not observed: %v", record.NotObserved)
	}
}
