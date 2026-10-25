package deliverybundle

import (
	"os"
	"path/filepath"
	"testing"
)

func packSigned(t *testing.T) (bundle, signature, publicKey string, manifest Manifest) {
	t.Helper()
	root := repoRoot(t)
	profile := filepath.Join(root, "tools", "reference-delivery-profile", "testdata", "profiles", "saas.json")
	catalog := filepath.Join(root, "tools", "reference-delivery-profile", "testdata", "plugins")
	dir := t.TempDir()
	bundle = filepath.Join(dir, "bundle.zip")
	packed, err := Pack(profile, catalog, bundle)
	if err != nil {
		t.Fatal(err)
	}
	privateKey := filepath.Join(dir, "private.pem")
	publicKey = filepath.Join(dir, "public.pem")
	if err := GenerateKeypair(privateKey, publicKey); err != nil {
		t.Fatal(err)
	}
	signature = filepath.Join(dir, "bundle.sig")
	if err := WriteSignature(bundle, privateKey, signature); err != nil {
		t.Fatal(err)
	}
	return bundle, signature, publicKey, packed.Manifest
}

func TestInstallIsIdempotent(t *testing.T) {
	bundle, signature, publicKey, manifest := packSigned(t)
	catalog := filepath.Join(t.TempDir(), "catalog")
	first, err := Install(bundle, signature, publicKey, catalog)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Installed) != len(manifest.Plugins) || len(first.Skipped) != 0 {
		t.Fatalf("unexpected first install: %+v", first)
	}
	second, err := Install(bundle, signature, publicKey, catalog)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Installed) != 0 || len(second.Skipped) != len(manifest.Plugins) {
		t.Fatalf("unexpected second install: %+v", second)
	}
}

func TestInstallRejectsDifferentExistingFile(t *testing.T) {
	bundle, signature, publicKey, manifest := packSigned(t)
	catalog := filepath.Join(t.TempDir(), "catalog")
	if err := os.MkdirAll(catalog, 0o755); err != nil {
		t.Fatal(err)
	}
	file := manifest.Plugins[0].File
	if err := os.WriteFile(filepath.Join(catalog, file), []byte("different"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(bundle, signature, publicKey, catalog); err == nil {
		t.Fatal("install overwrote a different existing file")
	}
}

func TestInstallRejectsWrongPublicKey(t *testing.T) {
	bundle, signature, _, _ := packSigned(t)
	dir := t.TempDir()
	otherPrivate := filepath.Join(dir, "other-private.pem")
	otherPublic := filepath.Join(dir, "other-public.pem")
	if err := GenerateKeypair(otherPrivate, otherPublic); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(bundle, signature, otherPublic, filepath.Join(dir, "catalog")); err == nil {
		t.Fatal("install accepted a signature from another key")
	}
}

func TestFreshInstallCreatesCompleteCatalog(t *testing.T) {
	bundle, signature, publicKey, manifest := packSigned(t)
	catalog := filepath.Join(t.TempDir(), "fresh-catalog")
	receipt, err := InstallFresh(bundle, signature, publicKey, catalog)
	if err != nil {
		t.Fatal(err)
	}
	if len(receipt.Installed) != len(manifest.Plugins) {
		t.Fatalf("installed=%d want %d", len(receipt.Installed), len(manifest.Plugins))
	}
	for _, plugin := range manifest.Plugins {
		if _, err := os.Stat(filepath.Join(catalog, plugin.File)); err != nil {
			t.Fatalf("missing %s: %v", plugin.File, err)
		}
	}
}

func TestFreshInstallRefusesExistingTarget(t *testing.T) {
	bundle, signature, publicKey, _ := packSigned(t)
	catalog := filepath.Join(t.TempDir(), "existing")
	if err := os.MkdirAll(catalog, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := InstallFresh(bundle, signature, publicKey, catalog); err == nil {
		t.Fatal("fresh install accepted an existing catalog")
	}
}

func TestFreshInstallFailureLeavesNoCatalog(t *testing.T) {
	bundle, signature, _, _ := packSigned(t)
	dir := t.TempDir()
	otherPrivate := filepath.Join(dir, "other-private.pem")
	otherPublic := filepath.Join(dir, "other-public.pem")
	if err := GenerateKeypair(otherPrivate, otherPublic); err != nil {
		t.Fatal(err)
	}
	catalog := filepath.Join(dir, "catalog")
	if _, err := InstallFresh(bundle, signature, otherPublic, catalog); err == nil {
		t.Fatal("fresh install accepted a wrong key")
	}
	if _, err := os.Stat(catalog); !os.IsNotExist(err) {
		t.Fatalf("failed fresh install left a catalog: %v", err)
	}
}

func TestMergeInstallPreflightsAllConflicts(t *testing.T) {
	bundle, signature, publicKey, manifest := packSigned(t)
	if len(manifest.Plugins) < 2 {
		t.Fatal("test bundle needs at least two plugins")
	}
	catalog := filepath.Join(t.TempDir(), "catalog")
	if err := os.MkdirAll(catalog, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(catalog, manifest.Plugins[1].File), []byte("different"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(bundle, signature, publicKey, catalog); err == nil {
		t.Fatal("conflict was accepted")
	}
	if _, err := os.Stat(filepath.Join(catalog, manifest.Plugins[0].File)); !os.IsNotExist(err) {
		t.Fatal("install wrote an earlier file before detecting a later conflict")
	}
}
