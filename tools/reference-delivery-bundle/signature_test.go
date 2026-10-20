package deliverybundle

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSignAndVerifyBundle(t *testing.T) {
	root := repoRoot(t)
	profile := filepath.Join(root, "tools", "reference-delivery-profile", "testdata", "profiles", "saas.json")
	catalog := filepath.Join(root, "tools", "reference-delivery-profile", "testdata", "plugins")
	dir := t.TempDir()
	bundle := filepath.Join(dir, "bundle.zip")
	if _, err := Pack(profile, catalog, bundle); err != nil {
		t.Fatal(err)
	}
	privateKey := filepath.Join(dir, "private.pem")
	publicKey := filepath.Join(dir, "public.pem")
	if err := GenerateKeypair(privateKey, publicKey); err != nil {
		t.Fatal(err)
	}
	signature := filepath.Join(dir, "bundle.sig")
	if err := WriteSignature(bundle, privateKey, signature); err != nil {
		t.Fatal(err)
	}
	if err := VerifySignature(bundle, signature, publicKey); err != nil {
		t.Fatal(err)
	}
}

func TestTamperedBundleOrSignatureIsRejected(t *testing.T) {
	root := repoRoot(t)
	profile := filepath.Join(root, "tools", "reference-delivery-profile", "testdata", "profiles", "saas.json")
	catalog := filepath.Join(root, "tools", "reference-delivery-profile", "testdata", "plugins")
	dir := t.TempDir()
	bundle := filepath.Join(dir, "bundle.zip")
	if _, err := Pack(profile, catalog, bundle); err != nil {
		t.Fatal(err)
	}
	privateKey := filepath.Join(dir, "private.pem")
	publicKey := filepath.Join(dir, "public.pem")
	if err := GenerateKeypair(privateKey, publicKey); err != nil {
		t.Fatal(err)
	}
	signature := filepath.Join(dir, "bundle.sig")
	if err := WriteSignature(bundle, privateKey, signature); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bundle, append(raw, 0), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := VerifySignature(bundle, signature, publicKey); err == nil {
		t.Fatal("tampered bundle was verified")
	}
}

func TestWrongPublicKeyIsRejected(t *testing.T) {
	root := repoRoot(t)
	profile := filepath.Join(root, "tools", "reference-delivery-profile", "testdata", "profiles", "saas.json")
	catalog := filepath.Join(root, "tools", "reference-delivery-profile", "testdata", "plugins")
	dir := t.TempDir()
	bundle := filepath.Join(dir, "bundle.zip")
	if _, err := Pack(profile, catalog, bundle); err != nil {
		t.Fatal(err)
	}
	privateKey := filepath.Join(dir, "private.pem")
	publicKey := filepath.Join(dir, "public.pem")
	if err := GenerateKeypair(privateKey, publicKey); err != nil {
		t.Fatal(err)
	}
	signature := filepath.Join(dir, "bundle.sig")
	if err := WriteSignature(bundle, privateKey, signature); err != nil {
		t.Fatal(err)
	}
	otherPrivate := filepath.Join(dir, "other-private.pem")
	otherPublic := filepath.Join(dir, "other-public.pem")
	if err := GenerateKeypair(otherPrivate, otherPublic); err != nil {
		t.Fatal(err)
	}
	if err := VerifySignature(bundle, signature, otherPublic); err == nil {
		t.Fatal("signature was verified with a different key")
	}
}
