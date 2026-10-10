package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func gitTest(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	data, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, data)
	}
	return strings.TrimSpace(string(data))
}

func exampleCheckout(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	gitTest(t, root, "init", "-q")
	gitTest(t, root, "config", "user.name", "TEI Probe Test")
	gitTest(t, root, "config", "user.email", "test@invalid.example")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("fixture\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "add", "README.md")
	gitTest(t, root, "commit", "-qm", "fixture")
	head := gitTest(t, root, "rev-parse", "HEAD")
	gitTest(t, root, "update-ref", "refs/remotes/origin/main", head)
	return root, head
}

func TestProbeMatchesLocalCheckoutButNeverClaimsFullQualification(t *testing.T) {
	root, head := exampleCheckout(t)
	gitTest(t, root, "remote", "add", "origin", "https://token:SECRET@github.com/example/repo.git")
	before := gitTest(t, root, "status", "--porcelain=v1", "--untracked-files=all")
	got := probe(root, head, head)
	if got.Git != "available" || got.Go != "available" || got.Checkout != "present" ||
		got.ExpectedHeadMatch != "match" || got.TrackingMainMatch != "match" ||
		got.MainAncestor != "match" || !got.EligibleForLocalGate ||
		got.CheckoutClean != "clean" || got.IgnoredArtifacts != "absent" || got.HiddenIndexFlags != "absent" || got.HistoryOverrides != "absent" {
		t.Fatalf("unexpected preflight: %#v", got)
	}
	if got.RepositoryHealth != "not-run" || got.LiveRemoteMain != "not-checked" || got.Credentials != "not-checked" {
		t.Fatalf("preflight claimed more evidence than measured: %#v", got)
	}
	encoded, err := json.Marshal(got)
	if err != nil || strings.Contains(string(encoded), "SECRET") || strings.Contains(string(encoded), "github.com") {
		t.Fatalf("secret or remote location disclosed: %s: %v", encoded, err)
	}
	after := gitTest(t, root, "status", "--porcelain=v1", "--untracked-files=all")
	if before != after {
		t.Fatalf("read-only probe changed checkout: %q -> %q", before, after)
	}
}

func TestProbeRejectsMissingOrStaleSourceState(t *testing.T) {
	root, head := exampleCheckout(t)
	wrong := strings.Repeat("a", 40)
	if got := probe(root, wrong, head); got.EligibleForLocalGate || got.ExpectedHeadMatch != "mismatch" {
		t.Fatalf("stale expected head accepted: %#v", got)
	}
	if got := probe(root, head, wrong); got.EligibleForLocalGate || got.TrackingMainMatch != "mismatch" {
		t.Fatalf("stale origin/main accepted: %#v", got)
	}
	gitTest(t, root, "update-ref", "-d", "refs/remotes/origin/main")
	if got := probe(root, head, head); got.EligibleForLocalGate || got.TrackingMainMatch != "unavailable" {
		t.Fatalf("missing origin/main accepted: %#v", got)
	}
	sub := filepath.Join(root, "sub")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}
	if got := probe(sub, head, head); got.EligibleForLocalGate || got.Checkout != "unavailable" {
		t.Fatalf("subdirectory impersonated repo root: %#v", got)
	}
	if got := probe(t.TempDir(), head, head); got.EligibleForLocalGate || got.Checkout != "unavailable" {
		t.Fatalf("directory without Git accepted: %#v", got)
	}
}

func TestProbeRequiresMainAncestry(t *testing.T) {
	root, original := exampleCheckout(t)
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("changed\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "add", "README.md")
	gitTest(t, root, "commit", "-qm", "second")
	head := gitTest(t, root, "rev-parse", "HEAD")
	// An unrelated commit cannot be a valid ancestor of the candidate.
	if got := probe(root, head, original); !got.EligibleForLocalGate || got.MainAncestor != "match" {
		t.Fatalf("proper ancestor rejected: %#v", got)
	}
	gitTest(t, root, "update-ref", "refs/remotes/origin/main", head)
	gitTest(t, root, "checkout", "-q", "--detach", original)
	if got := probe(root, original, head); got.EligibleForLocalGate || got.MainAncestor != "mismatch" {
		t.Fatalf("candidate missing main accepted: %#v", got)
	}
}

func TestArgumentsRejectMissingSHAAndUnknownCommands(t *testing.T) {
	root, head := exampleCheckout(t)
	if code := run([]string{"--root", root, "--expected-head", "123", "--expected-main", head}); code != 2 {
		t.Fatalf("malformed sha exit %d", code)
	}
	if code := run([]string{"--root", root, "--expected-head", head, "--expected-main", head, "--command", "rm -rf"}); code != 2 {
		t.Fatalf("unknown executable argument exit %d", code)
	}
}

func TestProbeRejectsDirtyUntrackedAndIgnoredCheckout(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*testing.T, string)
		field  string
	}{
		{"tracked modification", func(t *testing.T, root string) {
			if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("modified\n"), 0644); err != nil {
				t.Fatal(err)
			}
		}, "dirty"},
		{"untracked file", func(t *testing.T, root string) {
			if err := os.WriteFile(filepath.Join(root, "new-file.txt"), []byte("new\n"), 0644); err != nil {
				t.Fatal(err)
			}
		}, "dirty"},
		{"ignored credential artifact", func(t *testing.T, root string) {
			if err := os.WriteFile(filepath.Join(root, ".git", "info", "exclude"), []byte("credential.env\n"), 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "credential.env"), []byte("SHOULD_NOT_LEAK\n"), 0644); err != nil {
				t.Fatal(err)
			}
		}, "ignored"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, head := exampleCheckout(t)
			tc.change(t, root)
			got := probe(root, head, head)
			if got.EligibleForLocalGate {
				t.Fatalf("preflight accepted unclean repo: %#v", got)
			}
			if tc.field == "dirty" && got.CheckoutClean != "dirty" {
				t.Fatalf("dirty tracked/untracked content not reported: %#v", got)
			}
			if tc.field == "ignored" && (got.IgnoredArtifacts != "present" || got.CheckoutClean != "clean") {
				t.Fatalf("ignored content not reported independently: %#v", got)
			}
			encoded, err := json.Marshal(got)
			if err != nil || strings.Contains(string(encoded), "SHOULD_NOT_LEAK") || strings.Contains(string(encoded), "credential.env") {
				t.Fatalf("artifact content/path appeared in status: %s %v", encoded, err)
			}
		})
	}
}

func TestProbeRejectsHiddenIndexFlags(t *testing.T) {
	for _, flag := range []string{"--assume-unchanged", "--skip-worktree"} {
		t.Run(flag, func(t *testing.T) {
			root, head := exampleCheckout(t)
			gitTest(t, root, "update-index", flag, "README.md")
			got := probe(root, head, head)
			if got.EligibleForLocalGate || got.HiddenIndexFlags != "present" {
				t.Fatalf("hidden tracked entry considered clean: %#v", got)
			}
		})
	}
}

func TestProbeDoesNotRunRepositoryConfiguredFSMonitor(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX executable fixture")
	}
	root, head := exampleCheckout(t)
	marker := filepath.Join(t.TempDir(), "fsmonitor-was-executed")
	hook := filepath.Join(t.TempDir(), "monitor.sh")
	content := "#!/bin/sh\n: > " + strconv.Quote(marker) + "\nexit 0\n"
	if err := os.WriteFile(hook, []byte(content), 0700); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "config", "core.fsmonitor", hook)
	got := probe(root, head, head)
	if !got.EligibleForLocalGate {
		t.Fatalf("otherwise valid checkout rejected: %#v", got)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("untrusted repository fsmonitor was executed; stat error=%v", err)
	}
}

func TestProbeRejectsCommitReplacementThatForgesMainAncestry(t *testing.T) {
	root, original := exampleCheckout(t)
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("changed\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitTest(t, root, "add", "README.md")
	gitTest(t, root, "commit", "-qm", "real main descendant")
	newMain := gitTest(t, root, "rev-parse", "HEAD")
	gitTest(t, root, "update-ref", "refs/remotes/origin/main", newMain)
	gitTest(t, root, "checkout", "-q", "--detach", original)
	originalTree := gitTest(t, root, "rev-parse", "HEAD^{tree}")
	fake := gitTest(t, root, "commit-tree", originalTree, "-p", newMain, "-m", "forged ancestor")
	gitTest(t, root, "replace", original, fake)
	// This succeeds without the protective Git --no-replace-objects option.
	gitTest(t, root, "merge-base", "--is-ancestor", newMain, original)
	got := probe(root, original, newMain)
	if got.EligibleForLocalGate || got.HistoryOverrides != "present" {
		t.Fatalf("forged ancestor was accepted: %#v", got)
	}
}

func TestProbeRejectsLegacyGrafts(t *testing.T) {
	root, head := exampleCheckout(t)
	path := filepath.Join(root, ".git", "info", "grafts")
	if err := os.WriteFile(path, []byte(head+" "+strings.Repeat("f", 40)+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	got := probe(root, head, head)
	if got.EligibleForLocalGate || got.HistoryOverrides != "present" {
		t.Fatalf("grafted history accepted: %#v", got)
	}
}

func TestCommandOutputRefusesTruncationAndCancelledExecution(t *testing.T) {
	root, _ := exampleCheckout(t)
	if data, err := commandOutputWithContext(context.Background(), root, 1, "rev-parse", "HEAD"); err == nil || data != "" {
		t.Fatalf("oversized Git output treated as a complete observation: %q %v", data, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if data, err := commandOutputWithContext(ctx, root, 1<<20, "rev-parse", "HEAD"); err == nil || data != "" {
		t.Fatalf("cancelled command returned a successful observation: %q %v", data, err)
	}
}
