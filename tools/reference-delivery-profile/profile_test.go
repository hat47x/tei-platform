package deliveryprofile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test source path is unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func testdata(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "tools", "reference-delivery-profile", "testdata")
}

func pluginPath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join(testdata(t), "plugins", name+".json")
}

func writeProfile(t *testing.T, profile Profile) string {
	t.Helper()
	raw, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "profile.json")
	if err := os.WriteFile(path, append(raw, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestValidProfilesProvideRequiredCapabilities(t *testing.T) {
	for _, name := range []string{"saas", "individual", "sier"} {
		receipt, err := Validate(filepath.Join(testdata(t), "profiles", name+".json"))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !receipt.Passed || len(receipt.Missing) != 0 {
			t.Fatalf("%s: %+v", name, receipt)
		}
		if len(receipt.Plugins) == 0 || len(receipt.ProvidedCapabilities) == 0 {
			t.Fatalf("%s: empty composition: %+v", name, receipt)
		}
	}
}

func TestMissingCapabilityFailsClosed(t *testing.T) {
	profile := Profile{
		Schema: ProfileSchema, ID: "saas-missing", Version: "1",
		RequiredCapabilities: []string{"tenant-isolation", "audit"},
		Plugins:              []PluginRef{{ID: "audit", Version: "1", Path: pluginPath(t, "audit")}},
	}
	receipt, err := Validate(writeProfile(t, profile))
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Passed || len(receipt.Missing) != 1 || receipt.Missing[0] != "tenant-isolation" {
		t.Fatalf("unexpected receipt: %+v", receipt)
	}
}

func TestDuplicatePluginIsRejected(t *testing.T) {
	profile := Profile{
		Schema: ProfileSchema, ID: "duplicate", Version: "1",
		RequiredCapabilities: []string{"audit"},
		Plugins: []PluginRef{
			{ID: "audit", Version: "1", Path: pluginPath(t, "audit")},
			{ID: "audit", Version: "1", Path: pluginPath(t, "audit")},
		},
	}
	if _, err := Validate(writeProfile(t, profile)); err == nil {
		t.Fatal("duplicate plugin was accepted")
	}
}

func TestUnknownPluginPathIsRejected(t *testing.T) {
	profile := Profile{
		Schema: ProfileSchema, ID: "unknown", Version: "1",
		RequiredCapabilities: []string{"audit"},
		Plugins:              []PluginRef{{ID: "audit", Version: "1", Path: "/nonexistent/plugin.json"}},
	}
	if _, err := Validate(writeProfile(t, profile)); err == nil {
		t.Fatal("unknown plugin path was accepted")
	}
}

func TestPluginVersionMismatchIsRejected(t *testing.T) {
	profile := Profile{
		Schema: ProfileSchema, ID: "version", Version: "1",
		RequiredCapabilities: []string{"audit"},
		Plugins:              []PluginRef{{ID: "audit", Version: "2", Path: pluginPath(t, "audit")}},
	}
	if _, err := Validate(writeProfile(t, profile)); err == nil {
		t.Fatal("plugin version mismatch was accepted")
	}
}

func TestEmptyRequiredCapabilitiesIsRejected(t *testing.T) {
	profile := Profile{
		Schema: ProfileSchema, ID: "empty", Version: "1",
		Plugins: []PluginRef{{ID: "audit", Version: "1", Path: pluginPath(t, "audit")}},
	}
	if _, err := Validate(writeProfile(t, profile)); err == nil {
		t.Fatal("empty requiredCapabilities was accepted")
	}
}

func TestVerifySaaSExercisesTenantIsolation(t *testing.T) {
	receipt, err := Verify(filepath.Join(testdata(t), "profiles", "saas.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Passed {
		t.Fatalf("verify failed: %+v", receipt)
	}
	status := map[string]string{}
	for _, check := range receipt.Conformance {
		status[check.Capability] = check.Status
	}
	if status["tenant-isolation"] != "verified" {
		t.Fatalf("tenant-isolation was not verified: %+v", receipt.Conformance)
	}
	if status["audit"] != "unverified" || status["authentication"] != "unverified" {
		t.Fatalf("unexpected unverified statuses: %+v", receipt.Conformance)
	}
}

func TestVerifyIndividualReportsNoTenantIsolationCheck(t *testing.T) {
	receipt, err := Verify(filepath.Join(testdata(t), "profiles", "individual.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Passed {
		t.Fatalf("verify failed: %+v", receipt)
	}
	if len(receipt.Conformance) != 2 {
		t.Fatalf("conformance=%+v", receipt.Conformance)
	}
	for _, check := range receipt.Conformance {
		if check.Status != "unverified" {
			t.Fatalf("unexpected check: %+v", check)
		}
	}
}

func TestMissingDependencyIsRejected(t *testing.T) {
	profile := Profile{
		Schema: ProfileSchema, ID: "missing-dependency", Version: "1",
		RequiredCapabilities: []string{"authentication"},
		Plugins:              []PluginRef{{ID: "authentication", Version: "1", Path: pluginPath(t, "authentication")}},
	}
	if _, err := Validate(writeProfile(t, profile)); err == nil {
		t.Fatal("authentication without its audit dependency was accepted")
	}
}

func TestConflictIsRejected(t *testing.T) {
	profile := Profile{
		Schema: ProfileSchema, ID: "conflict", Version: "1",
		RequiredCapabilities: []string{"offline-storage", "tenant-isolation"},
		Plugins: []PluginRef{
			{ID: "offline-storage", Version: "1", Path: pluginPath(t, "offline-storage")},
			{ID: "tenant-isolation", Version: "1", Path: filepath.Join(repoRoot(t), "tools", "reference-tenant-isolation", "plugin.json")},
		},
	}
	if _, err := Validate(writeProfile(t, profile)); err == nil {
		t.Fatal("conflicting plugins were accepted")
	}
}

func TestDependencyOrderIsResolved(t *testing.T) {
	receipt, err := Validate(filepath.Join(testdata(t), "profiles", "sier.json"))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"audit", "authentication", "export-package"}
	if len(receipt.Order) != len(want) {
		t.Fatalf("order=%v want %v", receipt.Order, want)
	}
	for i := range want {
		if receipt.Order[i] != want[i] {
			t.Fatalf("order=%v want %v", receipt.Order, want)
		}
	}
}

func TestDependencyCycleIsRejected(t *testing.T) {
	dir := t.TempDir()
	writePlugin := func(id string, requires []string) string {
		plugin := Plugin{
			Schema: PluginSchema, ID: id, Version: "1", Kind: "delivery-foundation",
			Capabilities: []string{id}, Requires: requires,
		}
		raw, err := json.MarshalIndent(plugin, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, id+".json")
		if err := os.WriteFile(path, append(raw, '\n'), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	profile := Profile{
		Schema: ProfileSchema, ID: "cycle", Version: "1",
		RequiredCapabilities: []string{"a"},
		Plugins: []PluginRef{
			{ID: "a", Version: "1", Path: writePlugin("a", []string{"b"})},
			{ID: "b", Version: "1", Path: writePlugin("b", []string{"a"})},
		},
	}
	if _, err := Validate(writeProfile(t, profile)); err == nil {
		t.Fatal("dependency cycle was accepted")
	}
}
