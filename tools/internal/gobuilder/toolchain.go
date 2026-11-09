package gobuilder

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// SelectExecutable resolves one exact Go binary without consulting PATH.
// The embedded Go root may be unavailable in a -trimpath built reference:
// in that case the caller must explicitly pin the executable.
func SelectExecutable(explicit, embeddedGoRoot, goos string) (string, error) {
	path := strings.TrimSpace(explicit)
	if path == "" {
		if embeddedGoRoot == "" {
			return "", fmt.Errorf("Go root is unavailable; specify the exact Go executable with -go")
		}
		path = filepath.Join(embeddedGoRoot, "bin", goExecutableName(goos))
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("resolve Go executable: %w", err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("stat Go executable: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("Go executable is not a regular file: %s", resolved)
	}
	return resolved, nil
}

func goExecutableName(goos string) string {
	if goos == "windows" {
		return "go.exe"
	}
	return "go"
}

// HashToolchain preserves the two Reference builders' historical closure
// encoding byte-for-byte: sorted walk of the selected GOROOT paths, file
// relative path, mode, exact bytes, and newline. The Go binary itself is
// separately digested by the caller and then bound with go version and this
// closure digest. It is not a source/worktree or acceptance identity.
func HashToolchain(goRoot, goos, goarch string) (string, error) {
	if strings.TrimSpace(goRoot) == "" {
		return "", fmt.Errorf("Go toolchain root is empty")
	}
	paths := []string{
		filepath.Join(goRoot, "VERSION"),
		filepath.Join(goRoot, "bin", goExecutableName(goos)),
		filepath.Join(goRoot, "pkg", "tool", goos+"_"+goarch),
		filepath.Join(goRoot, "src"),
	}
	sum := sha256.New()
	for _, root := range paths {
		info, err := os.Lstat(root)
		if err != nil {
			return "", err
		}
		if !info.IsDir() {
			if err := hashToolchainEntry(sum, goRoot, root, info); err != nil {
				return "", err
			}
			continue
		}
		if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			return hashToolchainEntry(sum, goRoot, path, info)
		}); err != nil {
			return "", err
		}
	}
	return "sha256:" + hex.EncodeToString(sum.Sum(nil)), nil
}

func hashToolchainEntry(sum io.Writer, goRoot, path string, info os.FileInfo) error {
	rel, err := filepath.Rel(goRoot, path)
	if err != nil {
		return err
	}
	rel = filepath.ToSlash(rel)
	if _, err := fmt.Fprintf(sum, "%s\x00%o\x00", rel, info.Mode()); err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("Go toolchain symlink is not supported: %s", rel)
	}
	if !info.Mode().IsRegular() {
		_, err := fmt.Fprint(sum, "special\n")
		return err
	}
	if _, err := fmt.Fprint(sum, "file\x00"); err != nil {
		return err
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	if _, err := io.Copy(sum, f); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	_, err = fmt.Fprint(sum, "\n")
	return err
}
