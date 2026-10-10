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
