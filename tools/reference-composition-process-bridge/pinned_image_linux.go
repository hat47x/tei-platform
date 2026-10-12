package compositionprocessbridge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const pinnedReferenceMaxBytes = 64 << 20

// pinnedReferenceHost is a Linux-only proof that verification and process launch
// can refer to one private, unlinked executable inode, not a mutable source path.
// It is intentionally not a production sandbox, signature verifier or Host ABI.
type pinnedReferenceHost struct {
	source     string
	digest     string
	executable *os.File
	launches   int
}

func pinReferenceExecutable(ctx context.Context, source, expected string) (*pinnedReferenceHost, error) {
	if len(expected) != len("sha256:")+64 || !strings.HasPrefix(expected, "sha256:") {
		return nil, fmt.Errorf("invalid expected SHA-256")
	}
	if _, err := hex.DecodeString(strings.TrimPrefix(expected, "sha256:")); err != nil {
		return nil, fmt.Errorf("invalid SHA-256 hex: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	src, err := os.Open(source)
	if err != nil {
		return nil, err
	}
	defer src.Close()
	info, err := src.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > pinnedReferenceMaxBytes {
		return nil, fmt.Errorf("source is not a bounded regular executable")
	}
	dir, err := os.MkdirTemp("", "tei-pinned-reference-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "executable")
	dst, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	h := sha256.New()
	n, copyErr := io.Copy(io.MultiWriter(dst, h), io.LimitReader(src, pinnedReferenceMaxBytes+1))
	if copyErr == nil && n > pinnedReferenceMaxBytes {
		copyErr = fmt.Errorf("source exceeds copy limit")
	}
	if copyErr == nil {
		copyErr = ctx.Err()
	}
	if copyErr == nil && expected != "sha256:"+hex.EncodeToString(h.Sum(nil)) {
		copyErr = errors.New("copied executable differs from expected digest")
	}
	if copyErr == nil {
		copyErr = dst.Chmod(0500)
	}
	if closeErr := dst.Close(); copyErr == nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		return nil, copyErr
	}
	// Close the writer before opening a read-only descriptor: an executable
	// with an outstanding writable descriptor is rejected by Linux (ETXTBSY).
	fd, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	if err := os.Remove(path); err != nil {
		fd.Close()
		return nil, err
	}
	return &pinnedReferenceHost{source: source, digest: expected, executable: fd}, nil
}

func (h *pinnedReferenceHost) Close() error {
	if h.executable == nil {
		return nil
	}
	err := h.executable.Close()
	h.executable = nil
	return err
}
