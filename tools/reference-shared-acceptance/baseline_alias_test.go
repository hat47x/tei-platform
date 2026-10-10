package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func baselineAliasFixture(t *testing.T) (baseline, alias string) {
	t.Helper()
	root := t.TempDir()
	baseline = filepath.Join(root, "reference", "workspace")
	if err := os.MkdirAll(baseline, 0o755); err != nil {
		t.Fatal(err)
	}
	alias = filepath.Join(root, "scratch-alias")
	if err := os.Symlink(baseline, alias); err != nil {
		t.Skipf("workspace symlinks are unavailable: %v", err)
	}
	return baseline, alias
}

func TestReferenceBaselineProtectionFollowsWorkspaceAlias(t *testing.T) {
	baseline, alias := baselineAliasFixture(t)
	if !isReferenceBaseline(baseline) {
		t.Fatal("direct reference/workspace must remain protected")
	}
	if !isReferenceBaseline(alias) {
		t.Fatal("symlink alias bypassed reference/workspace protection")
	}
	// Keep the unclean spelling: a symlink is resolved before ".." by the
	// filesystem, whereas filepath.Abs would discard that segment first.
	aliasThroughParent := alias + string(os.PathSeparator) + ".." + string(os.PathSeparator) + "workspace"
	aliasInfo, err := os.Stat(aliasThroughParent)
	if err != nil {
		t.Fatal(err)
	}
	baselineInfo, err := os.Stat(baseline)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(aliasInfo, baselineInfo) {
		t.Fatal("test setup: alias through parent does not refer to baseline")
	}
	if !isReferenceBaseline(aliasThroughParent) {
		t.Fatal("symlink alias with parent segment bypassed baseline protection")
	}
	ordinary := filepath.Join(filepath.Dir(alias), "ordinary-scratch")
	if err := os.Mkdir(ordinary, 0o755); err != nil {
		t.Fatal(err)
	}
	if isReferenceBaseline(ordinary) {
		t.Fatal("ordinary scratch workspace was classified as protected")
	}
}

func TestReferenceBaselineWriteCommandsRejectSymlinkAlias(t *testing.T) {
	baseline, alias := baselineAliasFixture(t)
	root := filepath.Dir(alias)
	candidate := filepath.Join(root, "candidate.json")
	input := filepath.Join(root, "confirmation.json")
	for _, file := range []string{candidate, input} {
		if err := os.WriteFile(file, []byte("{}"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cases := []struct {
		name string
		args []string
	}{
		{
			name: "accept",
			args: []string{"accept", "-workspace", alias, "-candidate", candidate,
				"-expected-review-fingerprint", "synthetic-review"},
		},
		{
			name: "confirm-record",
			args: []string{"confirm-record", "-workspace", alias, "-input", input},
		},
		{
			name: "accept alias through parent",
			args: []string{"accept", "-workspace", alias + string(os.PathSeparator) + ".." + string(os.PathSeparator) + "workspace",
				"-candidate", candidate, "-expected-review-fingerprint", "synthetic-review"},
		},
		{
			name: "confirm-record alias through parent",
			args: []string{"confirm-record", "-workspace", alias + string(os.PathSeparator) + ".." + string(os.PathSeparator) + "workspace",
				"-input", input},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := runCLI(tc.args, &stdout, &stderr); code != 2 ||
				!strings.Contains(stderr.String(), "reference/workspace") {
				t.Fatalf("baseline write guard not applied: exit=%d stdout=%s stderr=%s",
					code, stdout.String(), stderr.String())
			}
			entries, err := os.ReadDir(baseline)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 0 {
				t.Fatalf("protected baseline was modified: %+v", entries)
			}
		})
	}
}

func TestReferenceBaselineOutputPaths(t *testing.T) {
	baseline, alias := baselineAliasFixture(t)
	root := filepath.Dir(alias)
	golden := filepath.Join(root, "reference", "world.json")
	if err := os.WriteFile(golden, []byte("golden"), 0o600); err != nil {
		t.Fatal(err)
	}
	dangling := filepath.Join(root, "dangling-output")
	if err := os.Symlink(filepath.Join(baseline, "new-output.json"), dangling); err != nil {
		t.Skipf("output symlink is unavailable: %v", err)
	}
	sep := string(os.PathSeparator)
	cases := []struct {
		name string
		path string
		want bool
	}{
		{name: "derived golden", path: golden, want: true},
		{name: "existing baseline root", path: baseline, want: true},
		{name: "new file in baseline", path: filepath.Join(baseline, "new-output.json"), want: true},
		{name: "new nested file in baseline", path: filepath.Join(baseline, "new", "nested.json"), want: true},
		{name: "output through workspace alias", path: filepath.Join(alias, "new-output.json"), want: true},
		{name: "output through alias and parent", path: alias + sep + ".." + sep + "workspace" + sep + "new-output.json", want: true},
		{name: "dangling output symlink", path: dangling, want: true},
		{name: "ordinary scratch output", path: filepath.Join(root, "scratch", "new.json"), want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isReferenceBaselineOutput(tc.path); got != tc.want {
				t.Fatalf("output path %q protected=%t want %t", tc.path, got, tc.want)
			}
		})
	}
}

func TestReferenceBaselineOutputFlagsRejectProtectedDestinations(t *testing.T) {
	baseline, alias := baselineAliasFixture(t)
	root := filepath.Dir(alias)
	golden := filepath.Join(root, "reference", "world.json")
	if err := os.WriteFile(golden, []byte("original golden"), 0o600); err != nil {
		t.Fatal(err)
	}
	candidate := filepath.Join(root, "candidate.json")
	if err := os.WriteFile(candidate, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	target := alias + string(os.PathSeparator) + ".." + string(os.PathSeparator) +
		"workspace" + string(os.PathSeparator) + "preview.json"
	cases := []struct {
		name string
		args []string
	}{
		{
			name: "revision output into baseline",
			args: []string{"revision", "-workspace", root, "-out", target},
		},
		{
			name: "review output into golden",
			args: []string{"review", "-workspace", root, "-candidate", candidate,
				"-out", golden},
		},
		{
			name: "accept derived output into golden",
			args: []string{"accept", "-workspace", root, "-candidate", candidate,
				"-expected-review-fingerprint", "synthetic-review", "-derived-out", golden},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := runCLI(tc.args, &stdout, &stderr)
			if code != 2 || !strings.Contains(stderr.String(), "-allow-reference-baseline") {
				t.Fatalf("protected output was not rejected: exit=%d stdout=%s stderr=%s",
					code, stdout.String(), stderr.String())
			}
			content, err := os.ReadFile(golden)
			if err != nil {
				t.Fatal(err)
			}
			if string(content) != "original golden" {
				t.Fatalf("golden overwritten: %q", content)
			}
			if _, err := os.Stat(filepath.Join(baseline, "preview.json")); !os.IsNotExist(err) {
				t.Fatalf("preview output was written to protected baseline: %v", err)
			}
		})
	}
}
