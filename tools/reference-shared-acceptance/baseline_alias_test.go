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
	safeFile := filepath.Join(root, "unrelated-result.json")
	if err := os.WriteFile(safeFile, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	safeAlias := filepath.Join(root, "safe-output-alias")
	if err := os.Symlink(safeFile, safeAlias); err != nil {
		t.Skipf("scratch output symlink is unavailable: %v", err)
	}
	goldenAlias := filepath.Join(root, "golden-output-alias")
	if err := os.Symlink(golden, goldenAlias); err != nil {
		t.Skipf("Golden output symlink is unavailable: %v", err)
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
		{name: "existing safe scratch symlink", path: safeAlias, want: false},
		{name: "existing Golden symlink", path: goldenAlias, want: true},
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

func TestReferenceBaselineGuardMatchesMixedCaseConservatively(t *testing.T) {
	baseline, _ := baselineAliasFixture(t)
	// Filesystem case-sensitivity differs among Windows, macOS and Linux
	// mounts. A baseline guard must not be bypassable on a case-insensitive
	// mount merely because the spelling uses different case.
	if !isReferenceBaseline(strings.ToUpper(baseline)) {
		t.Fatal("case variant bypassed workspace baseline protection")
	}
	if !isReferenceBaselineOutput(strings.ToUpper(filepath.Join(baseline, "derived.json"))) {
		t.Fatal("case variant bypassed source output protection")
	}
	if !isReferenceBaselineOutput(strings.ToUpper(filepath.Join(filepath.Dir(baseline), "world.json"))) {
		t.Fatal("case variant bypassed derived Golden protection")
	}
}

func TestReferenceBaselineOutputFlagAllowsResolvedScratchSymlink(t *testing.T) {
	// Use the ordinary CLI, not just the classifier, to verify that the
	// safety guard does not disable a legitimate scratch output workflow.
	workspace := scratchWorkspace(t)
	root := t.TempDir()
	output := filepath.Join(root, "revision.json")
	if err := os.WriteFile(output, []byte("before"), 0o600); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "revision-alias.json")
	if err := os.Symlink(output, alias); err != nil {
		t.Skipf("scratch output symlink unavailable: %v", err)
	}

	var stdout, stderr bytes.Buffer
	if code := runCLI([]string{"revision", "-workspace", workspace, "-out", alias}, &stdout, &stderr); code != 0 {
		t.Fatalf("safe scratch output alias rejected: code=%d stdout=%s stderr=%s",
			code, stdout.String(), stderr.String())
	}
	content, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "modelRevision") {
		t.Fatalf("revision output was not written through safe alias: %q", content)
	}
}

func TestReferenceAcceptRejectsCollidingOutputDestinations(t *testing.T) {
	root := t.TempDir()
	existing := filepath.Join(root, "existing.json")
	if err := os.WriteFile(existing, []byte("unchanged"), 0o600); err != nil {
		t.Fatal(err)
	}
	fileAlias := filepath.Join(root, "existing-alias.json")
	if err := os.Symlink(existing, fileAlias); err != nil {
		t.Skipf("output symlinks are unavailable: %v", err)
	}
	directoryAlias := filepath.Join(root, "directory-alias")
	if err := os.Symlink(root, directoryAlias); err != nil {
		t.Skipf("directory symlinks are unavailable: %v", err)
	}
	newFile := filepath.Join(root, "new-output.json")
	if err := os.Mkdir(filepath.Join(root, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name       string
		out        string
		derivedOut string
		allow      bool
	}{
		{name: "identical output", out: existing, derivedOut: existing},
		{name: "existing file alias", out: fileAlias, derivedOut: existing},
		{name: "new file under linked directory", out: filepath.Join(directoryAlias, "new-output.json"), derivedOut: newFile},
		{name: "same destination via dot segments", out: filepath.Join(root, "nested") + string(os.PathSeparator) + ".." + string(os.PathSeparator) + "new-output.json", derivedOut: newFile},
		{name: "case variation", out: existing, derivedOut: strings.ToUpper(existing)},
		{name: "explicit baseline override does not permit collision", out: existing, derivedOut: fileAlias, allow: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{
				"accept", "-workspace", root, "-candidate", filepath.Join(root, "not-present.json"),
				"-expected-review-fingerprint", "synthetic-review",
				"-out", tc.out, "-derived-out", tc.derivedOut,
			}
			if tc.allow {
				args = append(args, "-allow-reference-baseline")
			}
			var stdout, stderr bytes.Buffer
			if code := runCLI(args, &stdout, &stderr); code != 2 ||
				!strings.Contains(stderr.String(), "-derived-out") {
				t.Fatalf("colliding outputs were not rejected before candidate loading: code=%d stdout=%s stderr=%s",
					code, stdout.String(), stderr.String())
			}
			content, err := os.ReadFile(existing)
			if err != nil {
				t.Fatal(err)
			}
			if string(content) != "unchanged" {
				t.Fatalf("existing output overwritten: %q", content)
			}
			if _, err := os.Stat(newFile); !os.IsNotExist(err) {
				t.Fatalf("new output unexpectedly created: %v", err)
			}
		})
	}
}

func TestReferenceAcceptPreventsCandidateOutputOverwrite(t *testing.T) {
	dir := t.TempDir()
	candidate := filepath.Join(dir, "candidate.json")
	if err := os.WriteFile(candidate, []byte("original-candidate"), 0o600); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(dir, "candidate-alias.json")
	if err := os.Symlink(candidate, alias); err != nil {
		t.Skipf("output symlinks are unavailable: %v", err)
	}
	for _, tc := range []struct {
		name string
		flag string
		path string
	}{
		{name: "decision output", flag: "-out", path: candidate},
		{name: "derived output", flag: "-derived-out", path: candidate},
		{name: "decision alias", flag: "-out", path: alias},
		{name: "derived alias", flag: "-derived-out", path: alias},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := runCLI([]string{
				"accept", "-workspace", dir, "-candidate", candidate,
				"-expected-review-fingerprint", "synthetic-review",
				tc.flag, tc.path,
			}, &stdout, &stderr)
			if code != 2 || !strings.Contains(stderr.String(), "candidate") {
				t.Fatalf("candidate overwrite not rejected before acceptance: code=%d output=%s stderr=%s", code, stdout.String(), stderr.String())
			}
			content, err := os.ReadFile(candidate)
			if err != nil {
				t.Fatal(err)
			}
			if string(content) != "original-candidate" {
				t.Fatalf("candidate overwritten: %q", content)
			}
		})
	}
}

func TestReferenceAcceptRejectsCandidateInputAliasWithOverride(t *testing.T) {
	dir := t.TempDir()
	candidate := filepath.Join(dir, "candidate.json")
	if err := os.WriteFile(candidate, []byte("original-candidate"), 0o600); err != nil {
		t.Fatal(err)
	}
	candidateAlias := filepath.Join(dir, "candidate-input-alias.json")
	if err := os.Symlink(candidate, candidateAlias); err != nil {
		t.Skipf("candidate symlinks are unavailable: %v", err)
	}
	directoryAlias := filepath.Join(dir, "directory-alias")
	if err := os.Symlink(dir, directoryAlias); err != nil {
		t.Skipf("directory symlinks are unavailable: %v", err)
	}
	for _, tc := range []struct {
		name string
		candidatePath string
		output string
		flag string
	}{
		{name: "input alias", candidatePath: candidateAlias, output: candidate, flag: "-out"},
		{name: "directory alias", candidatePath: candidate, output: filepath.Join(directoryAlias, "candidate.json"), flag: "-derived-out"},
		{name: "case folded", candidatePath: candidate, output: strings.ToUpper(candidate), flag: "-out"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := runCLI([]string{
				"accept", "-workspace", dir, "-candidate", tc.candidatePath,
				"-expected-review-fingerprint", "synthetic-review",
				tc.flag, tc.output, "-allow-reference-baseline",
			}, &stdout, &stderr)
			if code != 2 || !strings.Contains(stderr.String(), "candidate") {
				t.Fatalf("alias was not rejected before candidate loading: code=%d out=%s err=%s", code, stdout.String(), stderr.String())
			}
			unchanged, err := os.ReadFile(candidate)
			if err != nil {
				t.Fatal(err)
			}
			if string(unchanged) != "original-candidate" {
				t.Fatalf("input candidate overwritten: %q", unchanged)
			}
		})
	}
}
