package pluginbuildsupply

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Store is a local content-addressed directory with the layout of
// tools/reference-build-artifact-store (<root>/sha256/<64 lowercase hex>).
// That asset's PutBytes and GetBytes are not reachable from another package,
// so the sample keeps the same rules here: the digest is the identity, a
// corrupt object is refused and never repaired, symlinks are refused.
type Store struct {
	Root     string
	MaxBytes int64
}

// StoreReceipt records one operation. The locator is store-relative.
type StoreReceipt struct {
	Operation string `json:"operation"`
	Digest    string `json:"digest"`
	Locator   string `json:"locator"`
	Bytes     int64  `json:"bytes"`
}

var (
	ErrStoreCorrupt  = errors.New("stored object does not match its digest")
	ErrStoreNotFound = errors.New("digest is not in the store")
	ErrNotStorable   = errors.New("only a reproducible artifact can be stored")
)

func (s Store) path(digest string) (string, string, error) {
	hexPart, ok := strings.CutPrefix(strings.ToLower(digest), "sha256:")
	if !ok || len(hexPart) != 64 || strings.Trim(hexPart, "0123456789abcdef") != "" {
		return "", "", fmt.Errorf("digest %q must be sha256:<64 hex>", digest)
	}
	locator := "sha256/" + hexPart
	return filepath.Join(s.Root, "sha256", hexPart), locator, nil
}

// PutReproducible stores bytes only when they are the artifact a
// reproducibility record established.
func (s Store) PutReproducible(rep Reproducibility, data []byte) (StoreReceipt, error) {
	if !rep.Reproducible || rep.ArtifactDigest == "" || sha256Digest(data) != rep.ArtifactDigest {
		return StoreReceipt{}, ErrNotStorable
	}
	return s.put(data)
}

func (s Store) put(data []byte) (StoreReceipt, error) {
	digest := sha256Digest(data)
	path, locator, err := s.path(digest)
	if err != nil {
		return StoreReceipt{}, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return StoreReceipt{}, err
	}
	if _, err := os.Lstat(path); err == nil {
		existing, err := s.readVerified(path, digest)
		if err != nil {
			return StoreReceipt{}, err
		}
		if !bytes.Equal(existing, data) {
			return StoreReceipt{}, ErrStoreCorrupt
		}
		return StoreReceipt{Operation: "put-existing", Digest: digest, Locator: locator, Bytes: int64(len(data))}, nil
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".put-*")
	if err != nil {
		return StoreReceipt{}, err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return StoreReceipt{}, err
	}
	if err := tmp.Close(); err != nil {
		return StoreReceipt{}, err
	}
	// A hard link never replaces an object another writer placed first.
	if err := os.Link(tmp.Name(), path); err != nil {
		return StoreReceipt{}, fmt.Errorf("publish %s: %w", locator, err)
	}
	if _, err := s.readVerified(path, digest); err != nil {
		return StoreReceipt{}, err
	}
	return StoreReceipt{Operation: "put", Digest: digest, Locator: locator, Bytes: int64(len(data))}, nil
}

// Get returns the bytes of a digest after hashing them again.
func (s Store) Get(digest string) ([]byte, StoreReceipt, error) {
	path, locator, err := s.path(digest)
	if err != nil {
		return nil, StoreReceipt{}, err
	}
	if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
		return nil, StoreReceipt{}, ErrStoreNotFound
	}
	data, err := s.readVerified(path, strings.ToLower(digest))
	if err != nil {
		return nil, StoreReceipt{}, err
	}
	return data, StoreReceipt{Operation: "get", Digest: sha256Digest(data), Locator: locator, Bytes: int64(len(data))}, nil
}

func (s Store) readVerified(path, digest string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("stored object %s is not a regular file", filepath.Base(path))
	}
	limit := s.MaxBytes
	if limit <= 0 {
		limit = 256 << 20
	}
	if info.Size() > limit {
		return nil, fmt.Errorf("stored object exceeds %d bytes", limit)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if sha256Digest(data) != digest {
		return nil, ErrStoreCorrupt
	}
	return data, nil
}
