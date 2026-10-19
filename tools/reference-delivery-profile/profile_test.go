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
