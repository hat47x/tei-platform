package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func openTestStore(t *testing.T, maxBytes int64) localArtifactStore {
	t.Helper()
	store, err := openLocalArtifactStore(t.TempDir(), maxBytes)
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func TestPutAndGetByExactDigest(t *testing.T) {
	store := openTestStore(t, 1<<20)
	value := []byte("exact artifact bytes")
	put, err := store.PutBytes(value)
	if err != nil {
		t.Fatal(err)
	}
	if put.Schema != artifactStoreReceiptSchema || put.Digest != digestBytes(value) {
		t.Fatalf("put receipt: %#v", put)
	}

	got, receipt, err := store.GetBytes(put.Digest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(value) {
		t.Fatalf("stored bytes changed: %q", got)
	}
	if receipt.Digest != put.Digest || receipt.Operation != "get" {
		t.Fatalf("get receipt: %#v", receipt)
	}
}

func TestRepeatedPutOfSameBytesIsIdempotent(t *testing.T) {
	store := openTestStore(t, 1<<20)
	value := []byte("idempotent artifact")
	first, err := store.PutBytes(value)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(first.Locator)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.PutBytes(value)
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(second.Locator)
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest != second.Digest {
		t.Fatalf("same bytes produced different digests: %s != %s", first.Digest, second.Digest)
	}
	if !os.SameFile(before, after) {
		t.Fatal("idempotent put replaced the existing content-addressed object")
	}
	if second.Operation != "put-existing" {
		t.Fatalf("second put operation: %s", second.Operation)
	}
}

func TestConcurrentPutOfSameBytesConverges(t *testing.T) {
	store := openTestStore(t, 1<<20)
	value := []byte("concurrent exact artifact")
	const workers = 12
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := store.PutBytes(value)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	got, _, err := store.GetBytes(digestBytes(value))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(value) {
		t.Fatalf("concurrent put changed bytes: %q", got)
	}
}

func TestGetRejectsCorruptedStoredBytes(t *testing.T) {
	store := openTestStore(t, 1<<20)
	value := []byte("expected")
	receipt, err := store.PutBytes(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(receipt.Locator, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(receipt.Locator, []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, _, err := store.GetBytes(receipt.Digest); !errors.Is(err, errArtifactCorrupt) {
		t.Fatalf("corrupted stored bytes were not rejected: %v", err)
	}
	if _, err := store.PutBytes(value); !errors.Is(err, errArtifactCorrupt) {
		t.Fatalf("put silently repaired corrupted content-addressed path: %v", err)
	}
}

func TestGetRejectsUnknownDigest(t *testing.T) {
	store := openTestStore(t, 1<<20)
	unknown := "sha256:" + strings.Repeat("a", 64)
	if _, _, err := store.GetBytes(unknown); !errors.Is(err, errArtifactNotFound) {
		t.Fatalf("unknown digest error: %v", err)
	}
}

func TestDigestIdentityDoesNotDependOnStoreLocator(t *testing.T) {
	left := openTestStore(t, 1<<20)
	right := openTestStore(t, 1<<20)
	value := []byte("same artifact in different stores")
	leftReceipt, err := left.PutBytes(value)
	if err != nil {
		t.Fatal(err)
	}
	rightReceipt, err := right.PutBytes(value)
	if err != nil {
		t.Fatal(err)
	}
	if leftReceipt.Digest != rightReceipt.Digest {
		t.Fatal("store locator changed artifact content identity")
	}
	if leftReceipt.Locator == rightReceipt.Locator {
		t.Fatal("different stores unexpectedly reported the same locator")
	}
}

func TestGetAcceptsValidUppercaseDigestHex(t *testing.T) {
	store := openTestStore(t, 1<<20)
	value := []byte("case canonicalization")
	receipt, err := store.PutBytes(value)
	if err != nil {
		t.Fatal(err)
	}
	upper := "sha256:" + strings.ToUpper(strings.TrimPrefix(receipt.Digest, "sha256:"))
	got, gotReceipt, err := store.GetBytes(upper)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(value) {
		t.Fatalf("uppercase digest lookup returned wrong bytes: %q", got)
	}
	if gotReceipt.Digest != receipt.Digest {
		t.Fatalf("receipt did not normalize digest: %s", gotReceipt.Digest)
	}
}

func TestStoreRejectsOversizedPutAndStoredObject(t *testing.T) {
	store := openTestStore(t, 4)
	if _, err := store.PutBytes([]byte("12345")); err == nil {
		t.Fatal("oversized put was accepted")
	}

	expected := digestBytes([]byte("tiny"))
	path, err := store.pathForDigest(expected)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("oversized"), 0o444); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.GetBytes(expected); !errors.Is(err, errArtifactCorrupt) {
		t.Fatalf("oversized stored object was not rejected: %v", err)
	}
}

func TestStoreRejectsSymlinkObject(t *testing.T) {
	store := openTestStore(t, 1<<20)
	value := []byte("symlink target")
	expected := digestBytes(value)
	path, err := store.pathForDigest(expected)
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(target, value, 0o444); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, _, err := store.GetBytes(expected); !errors.Is(err, errArtifactCorrupt) {
		t.Fatalf("symlink object was not rejected: %v", err)
	}
}

func TestStoreRejectsSymlinkAlgorithmDirectory(t *testing.T) {
	root := t.TempDir()
	target := t.TempDir()
	if err := os.Symlink(target, filepath.Join(root, artifactDigestAlgorithm)); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := openLocalArtifactStore(root, 1<<20); err == nil {
		t.Fatal("symlink algorithm directory was accepted")
	}
}

func TestStoreBoundsAreReferenceProfileNotContentIdentity(t *testing.T) {
	root := t.TempDir()
	small, err := openLocalArtifactStore(root, 8)
	if err != nil {
		t.Fatal(err)
	}
	value := []byte("12345678")
	receipt, err := small.PutBytes(value)
	if err != nil {
		t.Fatal(err)
	}
	large, err := openLocalArtifactStore(root, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	got, _, err := large.GetBytes(receipt.Digest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(value) {
		t.Fatal("changing local read bound changed content identity")
	}
}

func TestArtifactReceiptDoesNotPromoteStoreAuthority(t *testing.T) {
	store := openTestStore(t, 1<<20)
	receipt, err := store.PutBytes([]byte("authority limits"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"store-availability-is-not-artifact-trust",
		"store-availability-is-not-build-provenance",
		"store-availability-is-not-acceptance-authority",
		"store-availability-is-not-runtime-authority",
		"locator-is-not-content-identity",
	} {
		found := false
		for _, got := range receipt.AuthorityLimits {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("authority limit %q missing from receipt: %#v", want, receipt)
		}
	}
}
