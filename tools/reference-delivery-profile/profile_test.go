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
	if status["tenant-isolation"] != "verified" || status["audit"] != "verified" || status["authentication"] != "verified" {
		t.Fatalf("a SaaS capability was not verified: %+v", receipt.Conformance)
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
	status := map[string]string{}
	for _, check := range receipt.Conformance {
		status[check.Capability] = check.Status
	}
	if status["offline-storage"] != "verified" || status["single-user"] != "unverified" {
		t.Fatalf("unexpected conformance: %+v", receipt.Conformance)
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

func copyIntoCatalog(t *testing.T, dir, name string) {
	t.Helper()
	raw, err := os.ReadFile(pluginPath(t, name))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestCatalogResolvesPluginsWithoutPath(t *testing.T) {
	catalogDir := t.TempDir()
	copyIntoCatalog(t, catalogDir, "audit")
	copyIntoCatalog(t, catalogDir, "authentication")
	profile := Profile{
		Schema: ProfileSchema, ID: "catalog", Version: "1",
		RequiredCapabilities: []string{"audit", "authentication"},
		Plugins: []PluginRef{
			{ID: "audit", Version: "1"},
			{ID: "authentication", Version: "1"},
		},
	}
	receipt, err := ValidateWithCatalog(writeProfile(t, profile), catalogDir)
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Passed {
		t.Fatalf("catalog profile did not pass: %+v", receipt)
	}
	for _, plugin := range receipt.Plugins {
		if plugin.Path == "" || plugin.Digest == "" {
			t.Fatalf("catalog plugin lacks path or digest: %+v", plugin)
		}
	}
}

func TestCatalogMissingVersionIsRejected(t *testing.T) {
	catalogDir := t.TempDir()
	copyIntoCatalog(t, catalogDir, "audit")
	profile := Profile{
		Schema: ProfileSchema, ID: "catalog-missing", Version: "1",
		RequiredCapabilities: []string{"audit"},
		Plugins:              []PluginRef{{ID: "audit", Version: "2"}},
	}
	if _, err := ValidateWithCatalog(writeProfile(t, profile), catalogDir); err == nil {
		t.Fatal("missing catalog version was accepted")
	}
}

func TestCatalogWithoutCatalogDirectoryIsRejected(t *testing.T) {
	profile := Profile{
		Schema: ProfileSchema, ID: "no-catalog", Version: "1",
		RequiredCapabilities: []string{"audit"},
		Plugins:              []PluginRef{{ID: "audit", Version: "1"}},
	}
	if _, err := ValidateWithCatalog(writeProfile(t, profile), ""); err == nil {
		t.Fatal("reference without path and catalog was accepted")
	}
}

func TestCatalogDuplicateVersionIsRejected(t *testing.T) {
	catalogDir := t.TempDir()
	copyIntoCatalog(t, catalogDir, "audit")
	raw, err := os.ReadFile(pluginPath(t, "audit"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(catalogDir, "audit-copy.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadCatalog(catalogDir); err == nil {
		t.Fatal("duplicate catalog version was accepted")
	}
}

func TestVerifySierExercisesExportPackage(t *testing.T) {
	receipt, err := Verify(filepath.Join(testdata(t), "profiles", "sier.json"))
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
	for _, capability := range []string{"export-package", "authentication", "audit"} {
		if status[capability] != "verified" {
			t.Fatalf("%s was not verified: %+v", capability, receipt.Conformance)
		}
	}
}
