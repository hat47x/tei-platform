package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestApplicationMembershipComparisonActualCLI(t *testing.T) {
	base := t.TempDir()
	binary := filepath.Join(base, "reference-acceptance")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", binary, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v: %s", err, out)
	}
	write := func(root, path, contents string) {
		t.Helper()
		path = filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	before, after := filepath.Join(base, "before"), filepath.Join(base, "after")
	for _, root := range []string{before, after} {
		write(root, "tei.workspace.yaml", "workspace: comparison\nformatVersion: 1\nsources:\n  - semantic\n  - applications\n")
		write(root, "semantic/alpha.tei", "namespace alpha\n\ntype AlphaId from std.identifier\n")
		write(root, "semantic/beta.tei", "namespace beta\n\ntype BetaId from std.identifier\n")
	}
	const appPath = "applications/example/application.tei"
	const original = "application example {\n    uses semantic alpha\n    uses semantic beta\n}\n"
	const reordered = "application example {\n    uses semantic beta\n    uses semantic alpha\n}\n"
	write(before, appPath, original)
	write(after, appPath, reordered)
	run := func(wantExit int, args ...string) applicationMembershipComparison {
		t.Helper()
		cmd := exec.Command(binary, append([]string{"compare-application-membership", "-before", before, "-after", after}, args...)...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		code := 0
		if err != nil {
			if ex, ok := err.(*exec.ExitError); ok {
				code = ex.ExitCode()
			} else {
				t.Fatal(err)
			}
		}
		if code != wantExit {
			t.Fatalf("exit=%d want=%d stdout=%s stderr=%s", code, wantExit, stdout.String(), stderr.String())
		}
		if code == 2 {
			return applicationMembershipComparison{}
		}
		var report applicationMembershipComparison
		if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
			t.Fatalf("report: %v: %s", err, stdout.String())
		}
		return report
	}
	report := run(0)
	if !report.ComparisonOnly || len(report.StrictChanges) != 1 || report.Membership == nil || len(report.Membership.Changes) != 0 || !strings.Contains(strings.Join(report.AuthorityLimits, ","), "not-an-acceptance-preview-or-decision") {
		t.Fatalf("reorder comparison: %+v", report)
	}
	write(after, appPath, "application example {\n    uses semantic alpha\n}\n")
	report = run(0)
	if len(report.StrictChanges) != 1 || report.Membership == nil || len(report.Membership.Changes) != 1 {
		t.Fatalf("removed namespace: %+v", report)
	}
	write(after, appPath, "application example {\n    uses semantic missing\n}\n")
	report = run(1)
	found := false
	for _, d := range report.AfterDiagnostics {
		found = found || d.Code == "UNKNOWN_APPLICATION_SEMANTIC_NAMESPACE"
	}
	if report.Membership != nil || len(report.StrictChanges) != 0 || !found {
		t.Fatalf("invalid workspace: %+v", report)
	}
	diagnostic := run(1, "-include-unvalidated-candidates")
	if diagnostic.WorkspaceValidationPassed || diagnostic.Membership == nil || len(diagnostic.Membership.Changes) != 1 || !strings.Contains(strings.Join(diagnostic.AuthorityLimits, ","), "workspace-validation-errors-candidates-unverified") {
		t.Fatalf("unvalidated candidate must retain non-success limits: %+v", diagnostic)
	}
	found = false
	for _, d := range diagnostic.AfterDiagnostics {
		found = found || d.Code == "UNKNOWN_APPLICATION_SEMANTIC_NAMESPACE"
	}
	if !found {
		t.Fatalf("unvalidated mode hid original diagnostic: %+v", diagnostic)
	}
	write(after, appPath, "application example {\n    invalid syntax\n}\n")
	run(2, "-include-unvalidated-candidates") // Load failures cannot be bypassed.
	run(2, "-out", filepath.Join(base, "not-allowed.json"))
	if _, err := os.Stat(filepath.Join(base, "not-allowed.json")); !os.IsNotExist(err) {
		t.Fatalf("read-only command wrote output file: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(before, filepath.FromSlash(appPath)))
	if err != nil || string(got) != original {
		t.Fatalf("before source was changed: %v", err)
	}
	t.Log("actual CLI: reordered membership differs only in strict view; removal appears in both; invalid model returns diagnostics and no candidates; no output file or source mutation")
}
