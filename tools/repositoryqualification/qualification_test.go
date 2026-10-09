package repositoryqualification

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRepositoryLocalQualificationControlFlow(t *testing.T) {
	for _, command := range []string{"git", "bash", "go"} {
		if _, err := exec.LookPath(command); err != nil {
			t.Skipf("%s is unavailable: %v", command, err)
		}
	}

	root := repositoryRoot(t)
	runner, err := os.ReadFile(filepath.Join(root, "tools", "run-repository-local-qualification.sh"))
	if err != nil {
		t.Fatal(err)
	}

	tmp := t.TempDir()
	origin := filepath.Join(tmp, "origin.git")
	work := filepath.Join(tmp, "work")
	receipt := filepath.Join(tmp, "receipt.json")

	run(t, "", "git", "init", "--bare", origin)
	run(t, "", "git", "init", work)
	run(t, work, "git", "checkout", "-b", "main")
	run(t, work, "git", "config", "user.name", "TEI Synthetic")
	run(t, work, "git", "config", "user.email", "tei@example.invalid")
	run(t, work, "git", "config", "commit.gpgsign", "false")

	if err := os.MkdirAll(filepath.Join(work, "tools"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(work, "tools", "run-repository-local-qualification.sh"),
		runner,
		0o755,
	); err != nil {
		t.Fatal(err)
	}
	health := []byte(`#!/usr/bin/env bash
set -euo pipefail
test -z "$(git status --porcelain --untracked-files=all --ignored=matching)"
printf 'synthetic repository health: PASS\n' >&2
`)
	if err := os.WriteFile(filepath.Join(work, "tools", "repository-health-check.sh"), health, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, ".gitignore"), []byte("ignored.log\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	run(t, work, "git", "add", ".")
	run(t, work, "git", "commit", "-m", "base")
	run(t, work, "git", "remote", "add", "origin", origin)
	run(t, work, "git", "push", "-u", "origin", "main")
	main := strings.TrimSpace(run(t, work, "git", "rev-parse", "HEAD"))

	run(t, work, "git", "checkout", "-b", "candidate")
	if err := os.WriteFile(filepath.Join(work, "candidate.txt"), []byte("candidate\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, work, "git", "add", "candidate.txt")
	run(t, work, "git", "commit", "-m", "candidate")
	run(t, work, "git", "push", "-u", "origin", "candidate")
	head := strings.TrimSpace(run(t, work, "git", "rev-parse", "HEAD"))

	t.Run("publishes receipt for an exact clean candidate", func(t *testing.T) {
		output := run(
			t,
			work,
			"bash",
			"tools/run-repository-local-qualification.sh",
			"--expected-head", head,
			"--expected-main", main,
			"--receipt", receipt,
		)
		if !strings.Contains(output, "TEI_REPOSITORY_MERGE_QUALIFICATION_V1") ||
			!strings.Contains(output, "TEI_REPOSITORY_MERGE_QUALIFICATION_END") {
			t.Fatalf("qualification markers are missing:\n%s", output)
		}

		raw, err := os.ReadFile(receipt)
		if err != nil {
			t.Fatal(err)
		}
		var got struct {
			Schema                        string `json:"schema"`
			CandidateHead                 string `json:"candidateHead"`
			ExpectedMain                  string `json:"expectedMain"`
			RepositoryHealthPassed        bool   `json:"repositoryHealthPassed"`
			HeadStable                    bool   `json:"headStable"`
			MainStable                    bool   `json:"mainStable"`
			CleanPre                      bool   `json:"cleanPre"`
			CleanPost                     bool   `json:"cleanPost"`
			CleanIncludesIgnored          bool   `json:"cleanIncludesIgnored"`
			NonstandardIndexStateAbsent   bool   `json:"nonstandardIndexStateAbsent"`
			CandidateContainsExpectedMain bool   `json:"candidateContainsExpectedMain"`
		}
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatal(err)
		}
		if got.Schema != "tei-repository-merge-qualification-v1" ||
			got.CandidateHead != head ||
			got.ExpectedMain != main ||
			!got.RepositoryHealthPassed ||
			!got.HeadStable ||
			!got.MainStable ||
			!got.CleanPre ||
			!got.CleanPost ||
			!got.CleanIncludesIgnored ||
			!got.NonstandardIndexStateAbsent ||
			!got.CandidateContainsExpectedMain {
			t.Fatalf("unexpected receipt: %+v", got)
		}
	})

	t.Run("rejects ignored artifacts", func(t *testing.T) {
		ignored := filepath.Join(work, "ignored.log")
		if err := os.WriteFile(ignored, []byte("ignored\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Remove(ignored) })

		runFailure(
			t,
			work,
			"bash",
			"tools/run-repository-local-qualification.sh",
			"--expected-head", head,
			"--expected-main", main,
		)
		if err := os.Remove(ignored); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("rejects stale main identity", func(t *testing.T) {
		runFailure(
			t,
			work,
			"bash",
			"tools/run-repository-local-qualification.sh",
			"--expected-head", head,
			"--expected-main", head,
		)
	})

	t.Run("rejects receipt inside repository", func(t *testing.T) {
		runFailure(
			t,
			work,
			"bash",
			"tools/run-repository-local-qualification.sh",
			"--expected-head", head,
			"--expected-main", main,
			"--receipt", filepath.Join(work, "receipt.json"),
		)
	})

	t.Run("rejects nonstandard index state", func(t *testing.T) {
		run(t, work, "git", "update-index", "--skip-worktree", "candidate.txt")
		t.Cleanup(func() {
			_ = runNoFail(work, "git", "update-index", "--no-skip-worktree", "candidate.txt")
		})

		runFailure(
			t,
			work,
			"bash",
			"tools/run-repository-local-qualification.sh",
			"--expected-head", head,
			"--expected-main", main,
		)
		run(t, work, "git", "update-index", "--no-skip-worktree", "candidate.txt")
	})
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func run(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, out)
	}
	return string(out)
}

func runFailure(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("%s %v unexpectedly succeeded:\n%s", name, args, out)
	}
	return string(out)
}

func runNoFail(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	return cmd.Run()
}
