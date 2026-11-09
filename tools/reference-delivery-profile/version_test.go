package deliveryprofile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fakeEntry(id, version string) CatalogEntry {
	return CatalogEntry{
		Plugin: Plugin{
			Schema: PluginSchema, ID: id, Version: version, Kind: "delivery-foundation",
			Capabilities: []string{id},
		},
		Path:   id + "-" + version + ".json",
		Digest: "sha256:" + strings.Repeat("0", 64),
	}
}

func TestVersionConstraintsSelectHighestMatch(t *testing.T) {
	catalog := Catalog{"audit": {
		"1.0.0": fakeEntry("audit", "1.0.0"),
		"1.2.0": fakeEntry("audit", "1.2.0"),
		"2.0.0": fakeEntry("audit", "2.0.0"),
	}}
	cases := []struct {
		constraint string
		want       string
	}{
		{"^1", "1.2.0"},
		{"~1.0", "1.0.0"},
		{">=1.1 <2", "1.2.0"},
		{"*", "2.0.0"},
		{"1.0.0", "1.0.0"},
		{"=1.2", "1.2.0"},
	}
	for _, tc := range cases {
		entry, err := catalog.ResolveConstraint("audit", tc.constraint)
		if err != nil {
			t.Fatalf("%s: %v", tc.constraint, err)
		}
		if entry.Plugin.Version != tc.want {
			t.Fatalf("%s: got %s want %s", tc.constraint, entry.Plugin.Version, tc.want)
		}
	}
}

func TestVersionConstraintWithoutMatchIsRejected(t *testing.T) {
	catalog := Catalog{"audit": {"1.0.0": fakeEntry("audit", "1.0.0")}}
	if _, err := catalog.ResolveConstraint("audit", "^2"); err == nil {
		t.Fatal("constraint without a matching version was accepted")
	}
}

func TestLockRoundTripAndVerification(t *testing.T) {
	catalogDir := t.TempDir()
	copyIntoCatalog(t, catalogDir, "audit")
	copyIntoCatalog(t, catalogDir, "authentication")
	profile := Profile{
		Schema: ProfileSchema, ID: "locked", Version: "1",
		RequiredCapabilities: []string{"audit", "authentication"},
		Plugins: []PluginRef{
			{ID: "audit", Version: "^1"},
			{ID: "authentication", Version: "^1"},
		},
	}
	profilePath := writeProfile(t, profile)
	lockPath := filepath.Join(t.TempDir(), "lock.json")
	lock, err := ResolveLock(profilePath, catalogDir, lockPath)
	if err != nil {
		t.Fatal(err)
	}
	if lock.Digest == "" || len(lock.Plugins) != 2 {
		t.Fatalf("unexpected lock: %+v", lock)
	}
	receipt, err := ValidateWithLock(profilePath, catalogDir, lockPath)
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.LockVerified || receipt.LockDigest != lock.Digest {
		t.Fatalf("lock was not verified: %+v", receipt)
	}
}

func TestLockDetectsCatalogVersionChange(t *testing.T) {
	catalogDir := t.TempDir()
	copyIntoCatalog(t, catalogDir, "audit")
	copyIntoCatalog(t, catalogDir, "authentication")
	profile := Profile{
		Schema: ProfileSchema, ID: "locked-change", Version: "1",
		RequiredCapabilities: []string{"audit", "authentication"},
		Plugins: []PluginRef{
			{ID: "audit", Version: "^1"},
			{ID: "authentication", Version: "^1"},
		},
	}
	profilePath := writeProfile(t, profile)
	lockPath := filepath.Join(t.TempDir(), "lock.json")
	if _, err := ResolveLock(profilePath, catalogDir, lockPath); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(catalogDir, "authentication.json"))
	if err != nil {
		t.Fatal(err)
	}
	var plugin Plugin
	if err := json.Unmarshal(raw, &plugin); err != nil {
		t.Fatal(err)
	}
	plugin.Version = "1.1.0"
	changed, err := json.MarshalIndent(plugin, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(catalogDir, "authentication-1.1.0.json"), append(changed, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateWithLock(profilePath, catalogDir, lockPath); err == nil {
		t.Fatal("lock accepted a changed catalog resolution")
	}
}

func TestLockDigestTamperIsRejected(t *testing.T) {
	catalogDir := t.TempDir()
	copyIntoCatalog(t, catalogDir, "audit")
	profile := Profile{
		Schema: ProfileSchema, ID: "tamper", Version: "1",
		RequiredCapabilities: []string{"audit"},
		Plugins:              []PluginRef{{ID: "audit", Version: "^1"}},
	}
	profilePath := writeProfile(t, profile)
	lockPath := filepath.Join(t.TempDir(), "lock.json")
	if _, err := ResolveLock(profilePath, catalogDir, lockPath); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	var lock Lock
	if err := json.Unmarshal(raw, &lock); err != nil {
		t.Fatal(err)
	}
	lock.Plugins[0].Version = "9.9.9"
	changed, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lockPath, append(changed, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadLock(lockPath); err == nil {
		t.Fatal("tampered lock digest was accepted")
	}
}
