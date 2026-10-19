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
