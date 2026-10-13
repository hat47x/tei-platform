package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tei/internal/sourceindex"
)

// acceptedSourcePath retains the shared Source Index's relative path policy.
// All accepted-source operations below use a single opened os.Root instead
// of rebuilding absolute write paths from an untrusted Workspace filename.
func acceptedSourcePath(path string) (string, error) {
	normalized, err := sourceindex.NormalizeWorkspacePath(path)
	if err != nil || normalized != path {
		return "", fmt.Errorf("invalid accepted Source path %q: %v", path, err)
	}
	return filepath.FromSlash(path), nil
}

// checkAcceptedSourcePath rejects symbolic links at EACH existing component,
// not only at the final filename. os.Root confines operations even if a
// component is replaced after this check; the check is not an atomic lock.
func checkAcceptedSourcePath(files *os.Root, relative string, mayCreate bool) error {
	name, err := acceptedSourcePath(relative)
	if err != nil {
		return err
	}
	parts := strings.Split(name, string(filepath.Separator))
	for i := range parts {
		path := filepath.Join(parts[:i+1]...)
		info, err := files.Lstat(path)
		if errors.Is(err, os.ErrNotExist) && mayCreate {
			continue
		}
		if err != nil {
			return fmt.Errorf("inspect Source component %s: %w", path, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("Source path %s is a symbolic link", path)
		}
		if i < len(parts)-1 && !info.IsDir() {
			return fmt.Errorf("Source parent %s is not a directory", path)
		}
		if i == len(parts)-1 && !mayCreate && !info.Mode().IsRegular() {
			return fmt.Errorf("Source %s is not a regular file", path)
		}
	}
	return nil
}

// sourceReadWithinRoot opens only inside the Workspace's pinned root. This
// is a mutating Acceptance-side guard, not a replacement for the canonical
// Workspace Loader/Source Index's independent read boundary (#1017).
func sourceReadWithinRoot(files *os.Root, relative string) ([]byte, error) {
	if err := checkAcceptedSourcePath(files, relative, false); err != nil {
		return nil, err
	}
	path, err := acceptedSourcePath(relative)
	if err != nil {
		return nil, err
	}
	return files.ReadFile(path)
}

// sourceMakeDirs creates only Source parents in the pinned root and returns
// their relative paths for a subsequent best-effort rollback.
func sourceMakeDirs(files *os.Root, relativeParent string) ([]string, error) {
	parts := strings.Split(relativeParent, string(filepath.Separator))
	var created []string
	for i := range parts {
		dir := filepath.Join(parts[:i+1]...)
		if dir == "." || dir == "" {
			continue
		}
		info, err := files.Lstat(dir)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
				return created, fmt.Errorf("Source parent %s is not a real directory", dir)
			}
			continue
		}
		if !errors.Is(err, os.ErrNotExist) {
			return created, fmt.Errorf("inspect Source parent %s: %w", dir, err)
		}
		if err := files.Mkdir(dir, 0o755); err != nil {
			return created, fmt.Errorf("create Source parent %s: %w", dir, err)
		}
		created = append(created, dir)
	}
	return created, nil
}

// sourceWriteWithinRoot publishes a completed Source file under the pinned
// Workspace root. Existing Source uses a rooted same-directory Rename; a new
// Source uses a create-only hard link, so another writer's new filename cannot
// be silently overwritten between the absence check and publication.
//
// Neither approach is a compare-and-swap for an existing file. Other writers
// that do not cooperate with Acceptance may still change bytes in the interval
// between fingerprint recheck and Rename. This remains an explicit limitation.
func sourceWriteWithinRoot(files *os.Root, relative string, content []byte, created bool) error {
	name, err := acceptedSourcePath(relative)
	if err != nil {
		return err
	}
	if err := checkAcceptedSourcePath(files, relative, created); err != nil {
		return err
	}
	info, err := files.Lstat(name)
	switch {
	case created && err == nil:
		return os.ErrExist
	case created && !errors.Is(err, os.ErrNotExist):
		return err
	case !created && err != nil:
		return err
	}

	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return err
	}
	temp := filepath.Join(filepath.Dir(name), ".tei-accept-"+hex.EncodeToString(random[:]))
	file, err := files.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer files.Remove(temp)
	writeErr := error(nil)
	if _, err := file.Write(content); err != nil {
		writeErr = err
	}
	if writeErr == nil && !created {
		// File.Chmod acts on the opened temporary file, not a racy path.
		writeErr = file.Chmod(info.Mode().Perm())
	}
	if writeErr == nil {
		writeErr = file.Sync()
	}
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	// Detect a late, obvious link swap before publication. The rooted Rename
	// or Link still provides the actual Workspace containment boundary.
	if err := checkAcceptedSourcePath(files, relative, created); err != nil {
		return err
	}
	if created {
		return files.Link(temp, name)
	}
	return files.Rename(temp, name)
}

// sourceRemoveWithinRoot is used by rollback only after verifying the bytes
// left by this same Acceptance. External edits are never blindly deleted.
func sourceRemoveWithinRoot(files *os.Root, relative string) error {
	if err := checkAcceptedSourcePath(files, relative, false); err != nil {
		return err
	}
	name, err := acceptedSourcePath(relative)
	if err != nil {
		return err
	}
	return files.Remove(name)
}

// sourceRemoveDirsIfEmpty never removes nonempty parents holding another
// actor's Source. All paths remain relative to the pinned Workspace root.
func sourceRemoveDirsIfEmpty(files *os.Root, createdDirs []string) []string {
	var incomplete []string
	for i := len(createdDirs) - 1; i >= 0; i-- {
		dir := createdDirs[i]
		err := files.Remove(dir)
		if err == nil || errors.Is(err, os.ErrNotExist) {
			continue
		}
		opened, openErr := files.Open(dir)
		if openErr == nil {
			entries, readErr := opened.ReadDir(1)
			_ = opened.Close()
			if readErr == nil && len(entries) > 0 {
				continue
			}
		}
		incomplete = append(incomplete, dir)
	}
	return incomplete
}
