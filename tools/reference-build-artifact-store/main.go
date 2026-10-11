package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"tei/internal/core"
	"tei/internal/plugin"
)

const (
	artifactStoreReceiptSchema = "tei.reference.local-artifact-store-receipt/v0"
	artifactDigestAlgorithm    = "sha256"
)

var (
	errArtifactNotFound = errors.New("artifact not found")
	errArtifactCorrupt  = errors.New("artifact store content is corrupt")
)

var artifactStoreAuthorityLimits = []string{
	"store-availability-is-not-artifact-trust",
	"store-availability-is-not-build-provenance",
	"store-availability-is-not-acceptance-authority",
	"store-availability-is-not-runtime-authority",
	"locator-is-not-content-identity",
}

type localArtifactStore struct {
	root             string
	maxBytes         int64
	algorithmDir     string
	algorithmDirInfo os.FileInfo
}

type artifactStoreReceipt struct {
	Schema          string   `json:"schema"`
	Operation       string   `json:"operation"`
	Digest          string   `json:"digest"`
	Size            int64    `json:"size"`
	Locator         string   `json:"locator"`
	AuthorityLimits []string `json:"authorityLimits"`
}

type artifactStoreEvidence struct {
	Schema               string               `json:"schema"`
	Put                  artifactStoreReceipt `json:"put"`
	Get                  artifactStoreReceipt `json:"get"`
	ReopenedGet          artifactStoreReceipt `json:"reopenedGet"`
	LockedArtifactDigest string               `json:"lockedArtifactDigest"`
	RetrievedDigest      string               `json:"retrievedDigest"`
	BytesEqual           bool                 `json:"bytesEqual"`
	AuthorityLimits      []string             `json:"authorityLimits"`
}

func main() {
	root, err := os.MkdirTemp("", "tei-build-artifact-store-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(root)

	store, err := openLocalArtifactStore(root, 1<<20)
	if err != nil {
		panic(err)
	}
	artifact := []byte("tei local artifact store reference")
	put, err := store.PutBytes(artifact)
	if err != nil {
		panic(err)
	}

	plan := plugin.BuildPlan{
		FormatVersion:               plugin.BuildPlanFormatVersion,
		Plugin:                      plugin.Identity{ID: core.MustCanonicalID("tei.reference.artifact_store_plugin"), Version: "1.0.0"},
		PluginDefinitionFingerprint: strings.Repeat("a", 64),
		Sources: []plugin.BuildSource{{
			Kind:    "reference-source",
			ID:      "plugin-source",
			Locator: "reference://plugin-source",
			Digest:  "sha256:" + strings.Repeat("b", 64),
		}},
		Builder: plugin.BuilderIdentity{
			ID:              core.MustCanonicalID("tei.builder.artifact_store_reference"),
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
	locked, err := plugin.LockBuildOutput(plan, put.Digest)
	if err != nil {
		panic(err)
	}

	firstBytes, get, err := store.GetBytes(locked.ArtifactDigest)
	if err != nil {
		panic(err)
	}
	reopened, err := openLocalArtifactStore(root, 1<<20)
	if err != nil {
		panic(err)
	}
	secondBytes, reopenedGet, err := reopened.GetBytes(locked.ArtifactDigest)
	if err != nil {
		panic(err)
	}
	if !bytes.Equal(firstBytes, artifact) || !bytes.Equal(secondBytes, artifact) {
		panic("artifact store reference retrieved different bytes")
	}

	out := artifactStoreEvidence{
		Schema:               "tei.reference.local-artifact-store-evidence/v0",
		Put:                  put,
		Get:                  get,
		ReopenedGet:          reopenedGet,
		LockedArtifactDigest: locked.ArtifactDigest,
		RetrievedDigest:      digestBytes(secondBytes),
		BytesEqual:           true,
		AuthorityLimits:      append([]string(nil), artifactStoreAuthorityLimits...),
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(out); err != nil {
		panic(err)
	}
}

func openLocalArtifactStore(root string, maxBytes int64) (localArtifactStore, error) {
	if maxBytes <= 0 {
		return localArtifactStore{}, fmt.Errorf("artifact store maxBytes must be positive")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return localArtifactStore{}, err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return localArtifactStore{}, fmt.Errorf("resolve artifact store root: %w", err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return localArtifactStore{}, err
	}
	if !info.IsDir() {
		return localArtifactStore{}, fmt.Errorf("artifact store root is not a directory")
	}
	store := localArtifactStore{
		root:         resolved,
		maxBytes:     maxBytes,
		algorithmDir: filepath.Join(resolved, artifactDigestAlgorithm),
	}
	algorithmDirInfo, err := store.ensureAlgorithmDirectory()
	if err != nil {
		return localArtifactStore{}, err
	}
	store.algorithmDirInfo = algorithmDirInfo
	return store, nil
}

func (s localArtifactStore) PutBytes(value []byte) (artifactStoreReceipt, error) {
	if int64(len(value)) > s.maxBytes {
		return artifactStoreReceipt{}, fmt.Errorf("artifact exceeds store maxBytes")
	}
	digest := digestBytes(value)
	target, err := s.pathForDigest(digest)
	if err != nil {
		return artifactStoreReceipt{}, err
	}

	if _, err := os.Lstat(target); err == nil {
		stored, err := s.readVerified(target, digest)
		if err != nil {
			return artifactStoreReceipt{}, err
		}
		return s.receipt("put-existing", digest, int64(len(stored))), nil
	} else if !os.IsNotExist(err) {
		return artifactStoreReceipt{}, err
	}

	dir := filepath.Dir(target)
	temp, err := os.CreateTemp(dir, ".tei-artifact-*")
	if err != nil {
		return artifactStoreReceipt{}, fmt.Errorf("create artifact store temp file: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)

	if _, err := temp.Write(value); err != nil {
		temp.Close()
		return artifactStoreReceipt{}, fmt.Errorf("write artifact store temp file: %w", err)
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return artifactStoreReceipt{}, fmt.Errorf("sync artifact store temp file: %w", err)
	}
	if err := temp.Chmod(0o444); err != nil {
		temp.Close()
		return artifactStoreReceipt{}, fmt.Errorf("make artifact store temp file read-only: %w", err)
	}
	if err := temp.Close(); err != nil {
		return artifactStoreReceipt{}, err
	}
	if _, err := s.readVerified(tempPath, digest); err != nil {
		return artifactStoreReceipt{}, fmt.Errorf("verify artifact store temp file: %w", err)
	}

	if err := s.verifyAlgorithmDirectory(); err != nil {
		return artifactStoreReceipt{}, err
	}
	if err := os.Link(tempPath, target); err != nil {
		if !os.IsExist(err) {
			return artifactStoreReceipt{}, fmt.Errorf("publish artifact store object without overwrite: %w", err)
		}
		stored, verifyErr := s.readVerified(target, digest)
		if verifyErr != nil {
			return artifactStoreReceipt{}, fmt.Errorf("concurrent artifact store object: %w", verifyErr)
		}
		return s.receipt("put-existing", digest, int64(len(stored))), nil
	}
	stored, err := s.readVerified(target, digest)
	if err != nil {
		return artifactStoreReceipt{}, fmt.Errorf("verify published artifact store object: %w", err)
	}
	return s.receipt("put", digest, int64(len(stored))), nil
}

func (s localArtifactStore) GetBytes(expectedDigest string) ([]byte, artifactStoreReceipt, error) {
	normalized, err := normalizeDigest(expectedDigest)
	if err != nil {
		return nil, artifactStoreReceipt{}, err
	}
	path, err := s.pathForDigest(normalized)
	if err != nil {
		return nil, artifactStoreReceipt{}, err
	}
	value, err := s.readVerified(path, normalized)
	if err != nil {
		return nil, artifactStoreReceipt{}, err
	}
	return value, s.receipt("get", normalized, int64(len(value))), nil
}

func (s localArtifactStore) ensureAlgorithmDirectory() (os.FileInfo, error) {
	if err := os.Mkdir(s.algorithmDir, 0o755); err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf("create artifact store algorithm directory: %w", err)
	}
	info, err := os.Lstat(s.algorithmDir)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, fmt.Errorf("artifact store algorithm path is not a real directory")
	}
	return info, nil
}

func (s localArtifactStore) verifyAlgorithmDirectory() error {
	info, err := os.Lstat(s.algorithmDir)
	if err != nil {
		return fmt.Errorf("%w: artifact store algorithm directory is unavailable", errArtifactCorrupt)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("%w: artifact store algorithm path is not a real directory", errArtifactCorrupt)
	}
	if s.algorithmDirInfo == nil || !os.SameFile(s.algorithmDirInfo, info) {
		return fmt.Errorf("%w: artifact store algorithm directory was replaced", errArtifactCorrupt)
	}
	return nil
}

func (s localArtifactStore) pathForDigest(digest string) (string, error) {
	if err := s.verifyAlgorithmDirectory(); err != nil {
		return "", err
	}
	normalized, err := normalizeDigest(digest)
	if err != nil {
		return "", err
	}
	hexDigest := strings.TrimPrefix(normalized, "sha256:")
	return filepath.Join(s.algorithmDir, hexDigest), nil
}

func (s localArtifactStore) readVerified(path, expectedDigest string) ([]byte, error) {
	normalized, err := normalizeDigest(expectedDigest)
	if err != nil {
		return nil, err
	}
	before, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", errArtifactNotFound, normalized)
		}
		return nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: stored object is not a regular file", errArtifactCorrupt)
	}
	if before.Size() > s.maxBytes {
		return nil, fmt.Errorf("%w: stored object exceeds maxBytes", errArtifactCorrupt)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !os.SameFile(before, opened) {
		return nil, fmt.Errorf("%w: stored object changed before open", errArtifactCorrupt)
	}

	var buffer bytes.Buffer
	hasher := sha256.New()
	limited := io.LimitReader(file, s.maxBytes+1)
	n, err := io.Copy(io.MultiWriter(&buffer, hasher), limited)
	if err != nil {
		return nil, err
	}
	if n > s.maxBytes {
		return nil, fmt.Errorf("%w: stored object exceeds maxBytes", errArtifactCorrupt)
	}
	actual := "sha256:" + hex.EncodeToString(hasher.Sum(nil))
	if actual != normalized {
		return nil, fmt.Errorf("%w: got %s want %s", errArtifactCorrupt, actual, normalized)
	}

	afterOpen, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !os.SameFile(opened, afterOpen) || afterOpen.Size() != n {
		return nil, fmt.Errorf("%w: stored object changed while reading", errArtifactCorrupt)
	}
	afterPath, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("%w: stored object disappeared after read", errArtifactCorrupt)
	}
	if !os.SameFile(opened, afterPath) {
		return nil, fmt.Errorf("%w: stored object path changed while reading", errArtifactCorrupt)
	}
	if err := s.verifyAlgorithmDirectory(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (s localArtifactStore) receipt(operation, digest string, size int64) artifactStoreReceipt {
	hexDigest := strings.TrimPrefix(digest, "sha256:")
	return artifactStoreReceipt{
		Schema:          artifactStoreReceiptSchema,
		Operation:       operation,
		Digest:          digest,
		Size:            size,
		Locator:         filepath.ToSlash(filepath.Join(artifactDigestAlgorithm, hexDigest)),
		AuthorityLimits: append([]string(nil), artifactStoreAuthorityLimits...),
	}
}

func digestBytes(value []byte) string {
	sum := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func normalizeDigest(value string) (string, error) {
	if !strings.HasPrefix(value, "sha256:") {
		return "", fmt.Errorf("artifact digest must use sha256:<64-hex>")
	}
	hexPart := strings.TrimPrefix(value, "sha256:")
	if len(hexPart) != 64 {
		return "", fmt.Errorf("artifact digest must contain 64 hex characters")
	}
	for _, r := range hexPart {
		if !((r >= '0' && r <= '9') ||
			(r >= 'a' && r <= 'f') ||
			(r >= 'A' && r <= 'F')) {
			return "", fmt.Errorf("artifact digest contains non-hex character")
		}
	}
	return "sha256:" + strings.ToLower(hexPart), nil
}
