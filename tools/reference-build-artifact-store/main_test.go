package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
)

func openTestStore(t *testing.T, maxBytes int64) localArtifactStore {
	t.Helper()
	store, err := openLocalArtifactStore(t.TempDir(), maxBytes)
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func receiptPath(store localArtifactStore, receipt artifactStoreReceipt) string {
	return filepath.Join(store.root, filepath.FromSlash(receipt.Locator))
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
	before, err := os.Stat(receiptPath(store, first))
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.PutBytes(value)
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(receiptPath(store, second))
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
	if err := os.Chmod(receiptPath(store, receipt), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(receiptPath(store, receipt), []byte("tampered"), 0o644); err != nil {
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
	if leftReceipt.Locator != rightReceipt.Locator {
		t.Fatalf("same content-addressed layout produced different relative locators: %q != %q", leftReceipt.Locator, rightReceipt.Locator)
	}
	if receiptPath(left, leftReceipt) == receiptPath(right, rightReceipt) {
		t.Fatal("different stores unexpectedly resolved the same physical path")
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

func TestStoreRejectsReplacedAlgorithmDirectory(t *testing.T) {
	root := t.TempDir()
	store, err := openLocalArtifactStore(root, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(store.algorithmDir); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(store.algorithmDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := store.PutBytes([]byte("must-not-publish")); !errors.Is(err, errArtifactCorrupt) {
		t.Fatalf("replaced algorithm directory was not rejected: %v", err)
	}
}

func TestConcurrentPutLeavesNoTemporaryObjects(t *testing.T) {
	store := openTestStore(t, 1<<20)
	value := []byte("concurrent cleanup")
	const workers = 8
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

	entries, err := os.ReadDir(store.algorithmDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("normal concurrent put left unexpected store entries: %#v", entries)
	}
	wantName := strings.TrimPrefix(digestBytes(value), "sha256:")
	if entries[0].Name() != wantName {
		t.Fatalf("stored object name=%q want=%q", entries[0].Name(), wantName)
	}
}

func TestReceiptLocatorDoesNotLeakStoreRoot(t *testing.T) {
	store := openTestStore(t, 1<<20)
	receipt, err := store.PutBytes([]byte("portable receipt locator"))
	if err != nil {
		t.Fatal(err)
	}
	if filepath.IsAbs(receipt.Locator) {
		t.Fatalf("receipt locator leaked an absolute Host path: %q", receipt.Locator)
	}
	if strings.Contains(filepath.ToSlash(receipt.Locator), filepath.ToSlash(store.root)) {
		t.Fatalf("receipt locator leaked store root: %q", receipt.Locator)
	}
	want := filepath.ToSlash(filepath.Join(artifactDigestAlgorithm, strings.TrimPrefix(receipt.Digest, "sha256:")))
	if receipt.Locator != want {
		t.Fatalf("receipt locator=%q want=%q", receipt.Locator, want)
	}
}

func TestLockedBuildDigestRetrievesExactStoredArtifact(t *testing.T) {
	store := openTestStore(t, 1<<20)
	artifact := []byte("self-build locked artifact")
	receipt, err := store.PutBytes(artifact)
	if err != nil {
		t.Fatal(err)
	}

	plan := plugin.BuildPlan{
		FormatVersion:               plugin.BuildPlanFormatVersion,
		Plugin:                      plugin.Identity{ID: core.MustCanonicalID("tei.reference.store_locked_plugin"), Version: "1.0.0"},
		PluginDefinitionFingerprint: strings.Repeat("a", 64),
		Sources: []plugin.BuildSource{{
			Kind:    "reference-source",
			ID:      "plugin-source",
			Locator: "reference://plugin-source",
			Digest:  "sha256:" + strings.Repeat("b", 64),
		}},
		Builder: plugin.BuilderIdentity{
			ID:              core.MustCanonicalID("tei.builder.store_reference"),
			Version:         "1.0.0",
			ContractVersion: "tei.plugin-build/v1",
			ArtifactDigest:  "sha256:" + strings.Repeat("c", 64),
		},
		Recipe: plugin.BuildRecipe{
			Kind:       "deterministic-reference",
			Entrypoint: "plugin-source",
		},
		OutputKind: plugin.BuildOutputRuntimeArtifact,
		Target:     "host/reference",
	}
	locked, err := plugin.LockBuildOutput(plan, receipt.Digest)
	if err != nil {
		t.Fatal(err)
	}
	if locked.ArtifactDigest != receipt.Digest {
		t.Fatalf("locked artifact digest=%s want=%s", locked.ArtifactDigest, receipt.Digest)
	}

	got, getReceipt, err := store.GetBytes(locked.ArtifactDigest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(artifact) {
		t.Fatalf("LockedBuild digest retrieved wrong bytes: %q", got)
	}
	if getReceipt.Digest != locked.ArtifactDigest {
		t.Fatalf("get receipt digest=%s want locked=%s", getReceipt.Digest, locked.ArtifactDigest)
	}
}
