package toolneutraldevelopment

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestQualificationReceiptAndIntegrationPreflightControlFlow(t *testing.T) {
	for _, command := range []string{"git", "bash", "go", "awk", "grep"} {
		if _, err := exec.LookPath(command); err != nil {
			t.Skipf("%s is unavailable: %v", command, err)
		}
	}

	root := repositoryRoot(t)
	qualify, err := os.ReadFile(filepath.Join(root, "tools", "reference-tool-neutral-development", "qualify.sh"))
	if err != nil {
		t.Fatal(err)
	}
	preflight, err := os.ReadFile(filepath.Join(root, "tools", "reference-tool-neutral-development", "integration-preflight.sh"))
	if err != nil {
		t.Fatal(err)
	}

	tmp := t.TempDir()
	origin := filepath.Join(tmp, "origin.git")
	candidate := filepath.Join(tmp, "candidate")
	integration := filepath.Join(tmp, "integration")
	receipt := filepath.Join(tmp, "qualification.receipt")

	run(t, "", "git", "init", "--bare", origin)
	run(t, "", "git", "init", candidate)
	run(t, candidate, "git", "checkout", "-b", "main")
	configureGit(t, candidate)

	toolDir := filepath.Join(candidate, "tools", "reference-tool-neutral-development")
	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(toolDir, "qualify.sh"), qualify, 0o755)
	writeFile(t, filepath.Join(toolDir, "integration-preflight.sh"), preflight, 0o755)
	writeFile(t, filepath.Join(toolDir, "run.sh"), []byte("#!/usr/bin/env bash\nset -euo pipefail\nprintf 'synthetic rq034 reference: PASS\\n'\n"), 0o755)
	writeFile(t, filepath.Join(candidate, "tools", "repository-health-check.sh"), []byte("#!/usr/bin/env bash\nset -euo pipefail\ntest -z \"$(git status --porcelain=v1 --untracked-files=all)\"\nprintf 'synthetic repository health: PASS\\n'\n"), 0o755)
	writeFile(t, filepath.Join(candidate, ".gitignore"), []byte("ignored.log\n"), 0o644)

	run(t, candidate, "git", "add", ".")
	run(t, candidate, "git", "commit", "-m", "base")
	run(t, candidate, "git", "remote", "add", "origin", origin)
	run(t, candidate, "git", "push", "-u", "origin", "main")
	main := strings.TrimSpace(run(t, candidate, "git", "rev-parse", "HEAD"))

	run(t, candidate, "git", "checkout", "-b", "candidate")
	writeFile(t, filepath.Join(candidate, "candidate.txt"), []byte("candidate\n"), 0o644)
	run(t, candidate, "git", "add", "candidate.txt")
	run(t, candidate, "git", "commit", "-m", "candidate")
	run(t, candidate, "git", "push", "-u", "origin", "candidate")
	head := strings.TrimSpace(run(t, candidate, "git", "rev-parse", "HEAD"))
	tree := strings.TrimSpace(run(t, candidate, "git", "rev-parse", "HEAD^{tree}"))

	output := run(
		t,
		candidate,
		"bash",
		"tools/reference-tool-neutral-development/qualify.sh",
		"--expected-head", head,
		"--expected-main", main,
		"--receipt-file", receipt,
	)
	for _, want := range []string{"qualification=PASS", "receipt-file=" + receipt, "receipt-blob="} {
		if !strings.Contains(output, want) {
			t.Fatalf("qualification output is missing %q:\n%s", want, output)
		}
	}
	receiptText := string(mustReadFile(t, receipt))
	for _, want := range []string{
		"format=tei-rq034-qualification-v3",
		"candidate-head=" + head,
		"candidate-tree=" + tree,
		"expected-main=" + main,
		"repository-health=PASS",
		"tool-neutral-development-reference=PASS",
		"qualification=PASS",
	} {
		if !strings.Contains(receiptText, want+"\n") {
			t.Fatalf("receipt is missing %q:\n%s", want, receiptText)
		}
	}

	run(t, "", "git", "clone", origin, integration)
	run(t, integration, "git", "checkout", "--detach", main)
	configureGit(t, integration)

	t.Run("accepts exact candidate and live main", func(t *testing.T) {
		out := run(
			t,
			candidate,
			"bash",
			"tools/reference-tool-neutral-development/integration-preflight.sh",
			"--receipt-file", receipt,
			"--candidate", head,
			"--expected-main", main,
			"--remote", "origin",
			"--integration-root", integration,
		)
		if !strings.Contains(out, "integration-preflight=PASS") ||
			!strings.Contains(out, "live-main-before="+main) ||
			!strings.Contains(out, "live-main-after="+main) {
			t.Fatalf("preflight output is incomplete:\n%s", out)
		}
	})

	t.Run("does not overwrite an existing receipt", func(t *testing.T) {
		before := string(mustReadFile(t, receipt))
		runFailure(
			t,
			candidate,
			"bash",
			"tools/reference-tool-neutral-development/qualify.sh",
			"--expected-head", head,
			"--expected-main", main,
			"--receipt-file", receipt,
		)
		after := string(mustReadFile(t, receipt))
		if after != before {
			t.Fatal("existing receipt was modified")
		}
	})

	t.Run("rejects a tampered candidate tree", func(t *testing.T) {
		tampered := filepath.Join(tmp, "tampered.receipt")
		bad := strings.Replace(receiptText, "candidate-tree="+tree, "candidate-tree=0000000000000000000000000000000000000000", 1)
		writeFile(t, tampered, []byte(bad), 0o644)
		runFailure(
			t,
			candidate,
			"bash",
			"tools/reference-tool-neutral-development/integration-preflight.sh",
			"--receipt-file", tampered,
			"--candidate", head,
			"--expected-main", main,
			"--remote", "origin",
			"--integration-root", integration,
		)
	})

	t.Run("rejects live main movement with stale local tracking ref", func(t *testing.T) {
		run(t, "", "git", "--git-dir", origin, "update-ref", "refs/heads/main", head)
		t.Cleanup(func() {
			_ = runNoFail("", "git", "--git-dir", origin, "update-ref", "refs/heads/main", main)
		})
		localMain := strings.TrimSpace(run(t, integration, "git", "rev-parse", "origin/main"))
		if localMain != main {
			t.Fatalf("integration tracking ref unexpectedly moved: %s", localMain)
		}
		out := runFailure(
			t,
			candidate,
			"bash",
			"tools/reference-tool-neutral-development/integration-preflight.sh",
			"--receipt-file", receipt,
			"--candidate", head,
			"--expected-main", main,
			"--remote", "origin",
			"--integration-root", integration,
		)
		if !strings.Contains(out, "live origin main moved") &&
			!strings.Contains(out, "live origin main changed") {
			t.Fatalf("live main movement was rejected for an unexpected reason:\n%s", out)
		}
		run(t, "", "git", "--git-dir", origin, "update-ref", "refs/heads/main", main)
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

func configureGit(t *testing.T, dir string) {
	t.Helper()
	run(t, dir, "git", "config", "user.name", "TEI Synthetic")
	run(t, dir, "git", "config", "user.email", "tei@example.invalid")
	run(t, dir, "git", "config", "commit.gpgsign", "false")
}

func writeFile(t *testing.T, path string, content []byte, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, content, mode); err != nil {
		t.Fatal(err)
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return content
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
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("%s %v unexpectedly succeeded:\n%s", name, args, out)
	}
	return string(out)
}

func runNoFail(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	return cmd.Run()
}
