package main

import (
	"bytes"
	"errors"
	"flag"
	"io"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

var advertisedFlagPattern = regexp.MustCompile(`(?:^|[\s(])-([a-z][a-z-]*)`)

// advertisedFlags extracts the flag names that `tei commands` advertises for
// one command, from both Required and Optional.
func advertisedFlags(record cliCommandRecord) []string {
	seen := map[string]bool{}
	for _, text := range append(append([]string{}, record.Required...), record.Optional...) {
		for _, match := range advertisedFlagPattern.FindAllStringSubmatch(text, -1) {
			seen[match[1]] = true
		}
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// #2332: what `tei commands` advertises and what the command accepts are one
// set. A flag removed from the implementation or from the catalog fails here.
func TestCommandCatalogFlagsMatchAcceptedFlags(t *testing.T) {
	for _, record := range commandCatalog().Commands {
		if record.Name == "model catalog" {
			continue // parses its own arguments, not the shared FlagSet
		}
		allowed, ok := selectedReadOnlyFlagAllowlist[record.Name]
		if !ok {
			t.Errorf("%s is advertised but has no accepted-flag list", record.Name)
			continue
		}
		want := append([]string{}, allowed...)
		sort.Strings(want)
		got := advertisedFlags(record)
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("%s: advertised flags %v differ from accepted flags %v", record.Name, got, want)
		}
	}
	advertised := map[string]bool{}
	for _, record := range commandCatalog().Commands {
		advertised[record.Name] = true
	}
	for name := range selectedReadOnlyFlagAllowlist {
		if !advertised[name] {
			t.Errorf("%s accepts a flag list but is not advertised", name)
		}
	}
}

func TestAdvertisedCommandsRejectUnrelatedFlags(t *testing.T) {
	unrelated := []string{"reason", "run-id", "strategy", "expected-revision"}
	for command, allowed := range selectedReadOnlyFlagAllowlist {
		t.Run(command, func(t *testing.T) {
			for _, bad := range unrelated {
				fs := flag.NewFlagSet(command, flag.ContinueOnError)
				fs.SetOutput(io.Discard)
				for _, name := range append(append([]string{}, allowed...), bad) {
					fs.String(name, "", "")
				}
				if err := fs.Parse([]string{"-" + bad, "x"}); err != nil {
					t.Fatal(err)
				}
				if err := validateSelectedReadOnlyFlags(fs, command); err == nil ||
					!strings.Contains(err.Error(), "-"+bad) {
					t.Fatalf("%s silently accepted -%s: %v", command, bad, err)
				}
			}
		})
	}
}

func TestAdvertisedCommandsRejectUnrelatedFlagsAtProcessBoundary(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "tei")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	if output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	root := t.TempDir()
	writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: cli_flags\nsources:\n  - semantic\n")
	writeWorkspaceFile(t, root, "semantic/sales/empty.tei", "namespace sales\n")

	run := func(args ...string) (int, string, string) {
		t.Helper()
		cmd := exec.Command(binary, args...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		code := 0
		if err := cmd.Run(); err != nil {
			var exit *exec.ExitError
			if !errors.As(err, &exit) {
				t.Fatalf("CLI execution: %v", err)
			}
			code = exit.ExitCode()
		}
		return code, stdout.String(), stderr.String()
	}

	for _, args := range [][]string{
		{"validate", "-workspace", root, "-expected-fingerprint", "x"},
		{"world", "-workspace", root, "-expected-fingerprint", "x"},
		{"requirements", "-workspace", root, "-limit", "1"},
		{"plugin-lock", "-workspace", root, "-scope", "x"},
		{"workspace-files", "-workspace", root, "-subject-kind", "x"},
		{"source-index", "-workspace", root, "-environment", "x"},
		{"knowledge-context", "-workspace", root, "-environment", "x"},
	} {
		code, stdout, stderr := run(args...)
		if code != 2 || stdout != "" || !strings.Contains(stderr, args[0]+" does not accept -") {
			t.Errorf("argv=%q exit=%d stdout=%q stderr=%q: unrelated flag was not rejected", args, code, stdout, stderr)
		}
	}
	// Advertised flags keep working.
	if code, stdout, stderr := run("validate", "-workspace", root); code != 0 || stdout != "ok\n" || stderr != "" {
		t.Errorf("validate -workspace: exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if code, _, stderr := run("workspace-files", "-workspace", root); code != 0 || stderr != "" {
		t.Errorf("workspace-files -workspace: exit=%d stderr=%q", code, stderr)
	}
}
