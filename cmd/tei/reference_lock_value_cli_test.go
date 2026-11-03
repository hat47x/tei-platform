package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/plugin"
	"tei/internal/workspace"
)

// The workspace and edits are synthetic and isolated; no reference baseline is changed.
func TestLogicalReferencesAndEnvironmentLockCLI(t *testing.T) {
	repo := testRepositoryRoot(t)
	binary := filepath.Join(t.TempDir(), "tei")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, "./cmd/tei")
	build.Dir = repo
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	root := t.TempDir()
	write := func(path, data string) {
		t.Helper()
		path = filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run := func(exit int, args ...string) string {
		t.Helper()
		cmd := exec.Command(binary, args...)
		cmd.Dir = repo
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		got := 0
		if err != nil {
			if e, ok := err.(*exec.ExitError); ok {
				got = e.ExitCode()
			} else {
				t.Fatal(err)
			}
		}
		if got != exit {
			t.Fatalf("tei %v: exit=%d want=%d stdout=%s stderr=%s", args, got, exit, stdout.String(), stderr.String())
		}
		return stdout.String()
	}
	write("tei.workspace.yaml", "workspace: value_probe\nformatVersion: 1\nsources:\n  - semantic\n  - plugins\n  - environments\n")
	const source = "semantic/sales/order.tei"
	const model = "namespace sales\n\ntype OrderId from std.identifier\ndata Order {\n    attribute orderId type sales.order_id\n    attribute parentId type sales.order_id\n    attribute previousId type sales.order_id\n}\n"
	if strings.Count(model, "type sales.order_id") != 3 {
		t.Fatal("fixture must contain three references to the same type")
	}
	const provider = "plugin tei.value.example v1.0.0 {\n    runtime builtin\n    provides deployment-executor runtime.value_example {\n        action application-start\n    }\n}\n"
	write(source, model)
	write("plugins/example/plugin.tei", provider)
	write("environments/test/environment.tei", "environment test {\n    plugin tei.value.example ^1\n}\n")
	validate := func(exit int) string { return run(exit, "validate", "-workspace", root) }
	lock := func() []byte {
		t.Helper()
		raw := run(0, "plugin-lock", "-workspace", root)
		var result struct {
			Result json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal([]byte(raw), &result); err != nil || len(result.Result) == 0 {
			t.Fatalf("lock envelope: %v: %s", err, raw)
		}
		return result.Result
	}
	resolve := func(exit int, lockPath string) string {
		return run(exit, "plugin-resolve", "-workspace", root, "-environment", "test", "-plugin-lock", lockPath)
	}
	validate(0)
	old := lock()
	oldPath := filepath.Join(root, "old-lock.json")
	if err := os.WriteFile(oldPath, old, 0o644); err != nil {
		t.Fatal(err)
	}
	resolve(0, oldPath)

	// An unrelated attribute edit and a source move need no per-reference digest rewrite.
	write(source, strings.Replace(model, "    attribute orderId type sales.order_id\n", "    attribute orderId type sales.order_id\n    attribute note type std.text\n", 1))
	validate(0)
	if !bytes.Equal(old, lock()) {
		t.Fatal("model-only edit changed environment Plugin Lock")
	}
	resolve(0, oldPath)
	if err := os.Rename(filepath.Join(root, filepath.FromSlash(source)), filepath.Join(root, "semantic", "sales", "moved.tei")); err != nil {
		t.Fatal(err)
	}
	validate(0)
	resolve(0, oldPath)

	write("semantic/sales/moved.tei", strings.Replace(model, "sales.order_id", "sales.missing", 1))
	if out := validate(1); !strings.Contains(out, "UNKNOWN_SEMANTIC_ATTRIBUTE_TYPE") {
		t.Fatalf("missing reference not diagnosed: %s", out)
	}
	// Characterize the existing cross-domain validation, not a desirable contract.
	for _, args := range [][]string{
		{"plugin-lock", "-workspace", root},
		{"plugin-resolve", "-workspace", root, "-environment", "test", "-plugin-lock", oldPath},
	} {
		out := run(1, args...)
		if !strings.Contains(out, "UNKNOWN_SEMANTIC_ATTRIBUTE_TYPE") {
			t.Fatalf("unrelated invalid model did not block %v as expected: %s", args, out)
		}
	}
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load synthetic workspace: %v", err)
	}
	var pinned plugin.Lock
	if err := json.Unmarshal(old, &pinned); err != nil {
		t.Fatal(err)
	}
	if len(loaded.World.Environments) != 1 {
		t.Fatalf("expected one environment, got %d", len(loaded.World.Environments))
	}
	resolved, err := (plugin.Catalog{Plugins: loaded.World.Plugins, Bundles: loaded.World.Bundles}).ResolveEnvironmentWithLock(loaded.World.Environments[0], pinned)
	if err != nil || len(resolved.Plugins) != 1 || string(resolved.Plugins[0].ID) != "tei.value.example" {
		t.Fatalf("underlying locked resolution: %+v, %v", resolved, err)
	}
	t.Log("existing CLI limitation: an unresolved model type also blocks plugin-lock and plugin-resolve in this independent builtin-plugin fixture")
	t.Log("the existing Go resolver can still resolve the exact pinned descriptor; this does not authorize execution or validate the model")
	diagnostic := func(environment, lockPath string) string {
		return run(1, "plugin-resolve", "-workspace", root, "-environment", environment, "-plugin-lock", lockPath, "-diagnostic-candidates")
	}
	out := diagnostic("test", oldPath)
	for _, expected := range []string{`"ok": false`, `"workspaceValidated": false`, `UNKNOWN_SEMANTIC_ATTRIBUTE_TYPE`, `"candidates"`, `tei.value.example`, `UNVALIDATED_CANDIDATES`} {
		if !strings.Contains(out, expected) {
			t.Fatalf("diagnostic missing %s: %s", expected, out)
		}
	}
	if out := diagnostic("missing", oldPath); !strings.Contains(out, "UNKNOWN_ENVIRONMENT") || strings.Contains(out, `"candidates"`) {
		t.Fatalf("unknown environment: %s", out)
	}
	write("environments/test/environment.tei", "environment test {\n    plugin tei.missing ^1\n}\n")
	if out := diagnostic("test", oldPath); !strings.Contains(out, "PLUGIN_RESOLVE_FAILED") || strings.Contains(out, `"candidates"`) {
		t.Fatalf("missing plugin: %s", out)
	}
	write("environments/test/environment.tei", "environment test {\n    plugin tei.value.example ^1\n}\n")
	write("plugins/example/plugin.tei", strings.Replace(provider, "action application-start", "action application-start\n        action application-retire", 1))
	if out := diagnostic("test", oldPath); !strings.Contains(out, "PLUGIN_RESOLVE_FAILED") || strings.Contains(out, `"candidates"`) {
		t.Fatalf("stale lock: %s", out)
	}
	write("plugins/example/plugin.tei", provider)
	t.Log("diagnostic candidates retain validation errors and non-success status; missing environment/plugin and stale lock return no candidates")
	write("semantic/sales/moved.tei", model)
	validate(0)
	beforeTypeChange := run(0, "world", "-workspace", root)
	// Illustrative byte-level comparison only: TEI does not implement this
	// hypothetical per-reference pin policy or a canonical type digest here.
	beforeDigest := sha256.Sum256([]byte("type OrderId from std.identifier"))
	afterDigest := sha256.Sum256([]byte("type OrderId from std.text"))
	if beforeDigest == afterDigest {
		t.Fatal("illustrative type-declaration hashes unexpectedly match")
	}
	changedType := strings.Replace(model, "type OrderId from std.identifier", "type OrderId from std.text", 1)
	write("semantic/sales/moved.tei", changedType)
	validate(0)
	if beforeTypeChange == run(0, "world", "-workspace", root) {
		t.Fatal("changed type was not reflected in the model")
	}
	if strings.Count(changedType, "type sales.order_id") != 3 || !bytes.Equal(old, lock()) {
		t.Fatal("type edit unexpectedly changed references or Plugin Lock")
	}
	resolve(0, oldPath)
	write("semantic/sales/moved.tei", model)

	// A descriptor change is a lock change, not a reason to rewrite the model.
	write("plugins/example/plugin.tei", strings.Replace(provider, "action application-start", "action application-start\n        action application-retire", 1))
	if out := resolve(2, oldPath); !strings.Contains(out, "fingerprint diverged") {
		t.Fatalf("changed descriptor not diagnosed: %s", out)
	}
	newPath := filepath.Join(root, "new-lock.json")
	if err := os.WriteFile(newPath, lock(), 0o644); err != nil {
		t.Fatal(err)
	}
	resolve(0, newPath)
	if got, err := os.ReadFile(filepath.Join(root, "semantic", "sales", "moved.tei")); err != nil || string(got) != model {
		t.Fatalf("model changed during lock update: %v", err)
	}
	t.Log("real CLI: 3 logical type references; type edit and file move accepted without reference rewrites, 1 missing reference rejected; 1 changed descriptor required 1 lock regeneration")
	t.Log("hypothetical comparison only: if every reference stored the exact declaration-byte hash, this type edit would require updating 3 pins; no per-reference pin implementation or timing was tested")
}
