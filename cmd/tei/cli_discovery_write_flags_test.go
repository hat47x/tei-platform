package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestDiscoveryWriteFlagsMatchObservedFileEffects pins #3182: `effect:
// "read-only"` means the command does not change the canonical meaning, not
// that the process writes nothing. `view -execute` with -operation-log or
// -lifecycle-state does create files, so discovery names those flags.
//
// The test runs the real binary and checks the file system before and after:
// a write flag that creates nothing, and a file write whose flag is not
// announced, both fail here.
func TestDiscoveryWriteFlagsMatchObservedFileEffects(t *testing.T) {
	binary := buildCLIForLifecycle(t)
	root := productSliceWorkspace(t)
	dir := t.TempDir()

	writeFlags := map[string][]string{}
	for _, record := range commandCatalog().Commands {
		writeFlags[record.Name] = record.WriteFlags
	}
	if got := writeFlags["view"]; len(got) != 2 ||
		!containsString(got, "-operation-log") || !containsString(got, "-lifecycle-state") {
		t.Fatalf("view writeFlags=%v, want -operation-log and -lifecycle-state", got)
	}

	execute := []string{
		"view", "-workspace", root, "-application", "order_entry",
		"-pattern", "order_entry.order_record", "-command", "sales.submit_order",
		"-input", "order=ORD-1", "-execute", "-format", "json",
	}

	for _, tc := range []struct {
		name string
		flag string
		path string
		args []string
	}{
		{
			name: "-operation-log",
			flag: "-operation-log",
			path: filepath.Join(dir, "records.json"),
		},
		{
			name: "-lifecycle-state",
			flag: "-lifecycle-state",
			path: filepath.Join(dir, "lifecycle.db"),
			args: []string{"-lifecycle-command", "cli-write-flag", "-lifecycle-owner", "owner-a", "-lifecycle-generation", "1"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := os.Stat(tc.path); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("%s already exists before the run", tc.path)
			}
			args := append(append([]string{}, execute...), tc.flag, tc.path)
			args = append(args, tc.args...)
			command := exec.Command(binary, args...)
			var stdout, stderr bytes.Buffer
			command.Stdout, command.Stderr = &stdout, &stderr
			runErr := command.Run()
			if runErr != nil {
				var exit *exec.ExitError
				if !errors.As(runErr, &exit) {
					t.Fatalf("CLI execution: %v", runErr)
				}
				if exit.ExitCode() != 0 {
					t.Fatalf("argv=%q exit=%d stdout=%q stderr=%q", args, exit.ExitCode(), stdout.String(), stderr.String())
				}
			}
			info, err := os.Stat(tc.path)
			if err != nil {
				t.Fatalf("%s was announced as a write but the run created nothing: %v", tc.flag, err)
			}
			if info.Size() == 0 {
				t.Fatalf("%s created an empty file", tc.flag)
			}
			if !containsString(writeFlags["view"], tc.flag) {
				t.Fatalf("%s writes %s but discovery does not announce it", tc.flag, tc.path)
			}
		})
	}

	// Without a write flag the same execution must not create a file in the
	// working directory; the write is opt-in, not a side effect of -execute.
	entriesBefore, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(binary, execute...)
	command.Dir = dir
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("plain -execute: %v (stdout=%q stderr=%q)", err, stdout.String(), stderr.String())
	}
	entriesAfter, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entriesAfter {
		found := false
		for _, before := range entriesBefore {
			if before.Name() == entry.Name() {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("plain -execute created %q in the working directory", entry.Name())
		}
	}

	// Only view is qualified for now: another command that writes a named file
	// must not be silently treated as writing nothing.
	for _, record := range commandCatalog().Commands {
		if record.Name == "view" {
			continue
		}
		if len(record.WriteFlags) != 0 {
			t.Fatalf("%s advertises writeFlags %v but only view is qualified here", record.Name, record.WriteFlags)
		}
	}
	for _, record := range commandCatalog().Commands {
		for _, flag := range record.WriteFlags {
			if !strings.HasPrefix(flag, "-") {
				t.Fatalf("%s writeFlags must name flags, got %q", record.Name, flag)
			}
			if !containsString(advertisedFlags(record), strings.TrimPrefix(flag, "-")) {
				t.Fatalf("%s write flag %q is not among its advertised flags", record.Name, flag)
			}
		}
	}

	// The field is additive: a record without it keeps the previous shape.
	raw, err := json.Marshal(commandCatalog())
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Commands []map[string]json.RawMessage `json:"commands"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	withField, without := 0, 0
	for _, cmd := range decoded.Commands {
		if _, ok := cmd["writeFlags"]; ok {
			withField++
		} else {
			without++
		}
	}
	if withField != 1 || without != len(decoded.Commands)-1 {
		t.Fatalf("writeFlags appears on %d records, want only view", withField)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
