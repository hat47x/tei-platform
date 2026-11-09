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

// Synthetic edit workflow through the actual TEI and comparison CLIs. The
// builtin descriptor is not an invocation or an actual user workload.
func TestDefinitionEditImpactAndPluginSelectionWorkflow(t *testing.T) {
	base := t.TempDir()
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	tei, compare := filepath.Join(base, "tei"+ext), filepath.Join(base, "compare"+ext)
	for _, target := range []struct{ output, pkg string }{{tei, "../../cmd/tei"}, {compare, "."}} {
		cmd := exec.Command("go", "build", "-o", target.output, target.pkg)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("build %s: %v\n%s", target.pkg, err, out)
		}
	}
	run := func(binary string, want int, args ...string) []byte {
		t.Helper()
		cmd := exec.Command(binary, args...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		code := 0
		if err != nil {
			if e, ok := err.(*exec.ExitError); ok {
				code = e.ExitCode()
			} else {
				t.Fatal(err)
			}
		}
		if code != want {
			t.Fatalf("%s %v: exit=%d want=%d stdout=%s stderr=%s", binary, args, code, want, stdout.String(), stderr.String())
		}
		return stdout.Bytes()
	}
	write := func(root, relative, content string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	const (
		modelPath  = "semantic/sales/order.tei"
		appPath    = "applications/orders/application.tei"
		model      = "namespace sales\n\ntype OrderId from std.identifier\ndata Order {\n    attribute orderId type sales.order_id\n    attribute parentId type sales.order_id\n}\n"
		app        = "application orders {\n    uses semantic sales\n    uses semantic customer\n}\n"
		pluginPath = "plugins/example/plugin.tei"
	)
	before, after := filepath.Join(base, "before"), filepath.Join(base, "after")
	for _, root := range []string{before, after} {
		write(root, "tei.workspace.yaml", "workspace: workflow\nformatVersion: 1\nsources:\n  - semantic\n  - applications\n  - plugins\n  - environments\n")
		write(root, modelPath, model)
		write(root, "semantic/customer/id.tei", "namespace customer\n\ntype CustomerId from std.identifier\n")
		write(root, appPath, app)
		write(root, pluginPath, "plugin tei.value.example v1.0.0 {\n    runtime builtin\n    provides deployment-executor runtime.value_example {\n        action application-start\n    }\n}\n")
		write(root, "environments/test/environment.tei", "environment test {\n    plugin tei.value.example ^1\n}\n")
	}
	run(tei, 0, "validate", "-workspace", before)
	var lockEnvelope struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(run(tei, 0, "plugin-lock", "-workspace", before), &lockEnvelope); err != nil || len(lockEnvelope.Result) == 0 {
		t.Fatalf("lock: %v", err)
	}
	lockPath := filepath.Join(base, "lock.json")
	if err := os.WriteFile(lockPath, lockEnvelope.Result, 0o644); err != nil {
		t.Fatal(err)
	}
	run(tei, 0, "plugin-resolve", "-workspace", before, "-environment", "test", "-plugin-lock", lockPath)
	write(after, modelPath, strings.Replace(model, "    attribute parentId", "    attribute note type std.text\n    attribute parentId", 1))
	write(after, appPath, "application orders {\n    uses semantic customer\n    uses semantic sales\n}\n")
	run(tei, 0, "validate", "-workspace", after)
	run(tei, 0, "plugin-resolve", "-workspace", after, "-environment", "test", "-plugin-lock", lockPath)
	var changedLock struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(run(tei, 0, "plugin-lock", "-workspace", after), &changedLock); err != nil || !bytes.Equal(lockEnvelope.Result, changedLock.Result) {
		t.Fatalf("model-only edits changed plugin lock: %v", err)
	}
	var report applicationMembershipComparison
	if err := json.Unmarshal(run(compare, 0, "compare-application-membership", "-before", before, "-after", after), &report); err != nil {
		t.Fatal(err)
	}
	if len(report.StrictChanges) != 1 || report.Membership == nil || len(report.Membership.Changes) != 0 || len(report.SemanticChanges) != 1 || report.SemanticChanges[0].Kind != "attribute" || !strings.Contains(strings.Join(report.Membership.AuthorityLimits, ","), "semantic-and-non-application-changes-not-evaluated") {
		t.Fatalf("semantic edit or comparison scope lost: %+v", report)
	}
	changedFiles := 0
	for _, path := range []string{modelPath, appPath, pluginPath, "environments/test/environment.tei"} {
		b, err := os.ReadFile(filepath.Join(before, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		a, err := os.ReadFile(filepath.Join(after, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(a, b) {
			changedFiles++
		}
	}
	afterModel, err := os.ReadFile(filepath.Join(after, filepath.FromSlash(modelPath)))
	if err != nil || changedFiles != 2 || strings.Count(model, "type sales.order_id") != 2 || strings.Count(string(afterModel), "type sales.order_id") != 2 {
		t.Fatal("unexpected edit or logical-reference count")
	}
	write(after, modelPath, strings.Replace(model, "sales.order_id", "sales.missing", 1))
	if out := run(tei, 1, "plugin-resolve", "-workspace", after, "-environment", "test", "-plugin-lock", lockPath); !bytes.Contains(out, []byte("UNKNOWN_SEMANTIC_ATTRIBUTE_TYPE")) {
		t.Fatalf("missing original validation error: %s", out)
	}
	if out := run(tei, 1, "plugin-resolve", "-workspace", after, "-environment", "test", "-plugin-lock", lockPath, "-diagnostic-candidates"); !bytes.Contains(out, []byte("UNKNOWN_SEMANTIC_ATTRIBUTE_TYPE")) || !bytes.Contains(out, []byte(`"workspaceValidated": false`)) || !bytes.Contains(out, []byte(`"candidates"`)) {
		t.Fatalf("diagnostic state: %s", out)
	}
	report = applicationMembershipComparison{}
	if err := json.Unmarshal(run(compare, 1, "compare-application-membership", "-before", before, "-after", after), &report); err != nil {
		t.Fatal(err)
	}
	if report.Membership != nil || len(report.SemanticChanges) != 0 {
		t.Fatalf("invalid model must not emit candidates: %+v", report)
	}
	t.Log("actual CLIs, synthetic workspace: 2 files edited; 2 stable references and 1 plugin lock unchanged; 1 semantic change visible; invalid type blocks normal selection and comparison, diagnostic stays non-success")
}
