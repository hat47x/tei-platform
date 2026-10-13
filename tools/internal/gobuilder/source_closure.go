package gobuilder

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// SourceMember represents a selected compiler input. The repository-relative
// path is significant: identical bytes at a different build path may produce
// different artifacts and must invalidate the closure identity.
type SourceMember struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
}

// GoListPackage is the subset of "go list -deps -json" metadata that
// identifies files actually selected by the Go compiler for a target.
type GoListModule struct {
	Path string
	Dir  string
}

type GoListPackage struct {
	Dir          string
	ImportPath   string
	Standard     bool
	Module       *GoListModule
	GoFiles      []string
	CgoFiles     []string
	CFiles       []string
	CXXFiles     []string
	MFiles       []string
	HFiles       []string
	FFiles       []string
	SFiles       []string
	SwigFiles    []string
	SwigCXXFiles []string
	SysoFiles    []string
	EmbedFiles   []string
}

// PackageBuildFiles retains all selected Go, assembly, native-compiler,
// object, and //go:embed inputs reported by the compiler resolver. Unselected
// files must not be included merely because they exist in the repository.
func PackageBuildFiles(pkg GoListPackage) []string {
	var out []string
	for _, files := range [][]string{
		pkg.GoFiles,
		pkg.CgoFiles,
		pkg.CFiles,
		pkg.CXXFiles,
		pkg.MFiles,
		pkg.HFiles,
		pkg.FFiles,
		pkg.SFiles,
		pkg.SwigFiles,
		pkg.SwigCXXFiles,
		pkg.SysoFiles,
		pkg.EmbedFiles,
	} {
		out = append(out, files...)
	}
	sort.Strings(out)
	return out
}

// SourceClosureDigest preserves the existing Plugin and Runtime Engine
// reference format exactly: sorted path, NUL, sha256 digest, newline.
func SourceClosureDigest(members []SourceMember) string {
	ordered := append([]SourceMember(nil), members...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Path < ordered[j].Path })
	sum := sha256.New()
	for _, member := range ordered {
		fmt.Fprintf(sum, "%s\x00%s\n", member.Path, member.Digest)
	}
	return "sha256:" + hex.EncodeToString(sum.Sum(nil))
}

// ResolveSourceClosure selects actual compiler inputs through offline go list.
// The Git index is an authorization check, never a replacement for go list's
// selected source set. External modules, untracked files, and symlink members
// are rejected rather than recorded as a complete source closure.
func ResolveSourceClosure(repo, goPath, packagePath, stateDir string) ([]SourceMember, string, error) {
	cmd := exec.Command(goPath, "list", "-deps", "-json", "-mod=readonly", packagePath)
	cmd.Dir = repo
	env, err := Environment(os.Environ(), stateDir, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return nil, "", fmt.Errorf("prepare Go source closure environment: %w", err)
	}
	cmd.Env = env
	raw, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, "", fmt.Errorf("resolve Go package source closure: %w: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, "", fmt.Errorf("resolve Go package source closure: %w", err)
	}
	return resolveSourceClosureFromList(repo, raw)
}

func resolveSourceClosureFromList(repo string, raw []byte) ([]SourceMember, string, error) {
	root, err := filepath.EvalSymlinks(repo)
	if err != nil {
		root = filepath.Clean(repo)
	}
	tracked, err := trackedSourcePaths(repo)
	if err != nil {
		return nil, "", err
	}
	paths := map[string]struct{}{"go.mod": {}}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	for {
		var pkg GoListPackage
		if err := decoder.Decode(&pkg); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, "", fmt.Errorf("decode Go package source closure: %w", err)
		}
		if pkg.Standard {
			continue
		}
		if pkg.Module == nil {
			return nil, "", fmt.Errorf("Go dependency %s has no module identity", pkg.ImportPath)
		}
		moduleDir, err := filepath.EvalSymlinks(pkg.Module.Dir)
		if err != nil {
			moduleDir = filepath.Clean(pkg.Module.Dir)
		}
		if moduleDir != root {
			return nil, "", fmt.Errorf("Go package source closure contains external module %s from %s", pkg.Module.Path, pkg.Module.Dir)
		}
		for _, name := range PackageBuildFiles(pkg) {
			full := filepath.Join(pkg.Dir, name)
			rel, err := filepath.Rel(repo, full)
			if err != nil {
				return nil, "", err
			}
			rel = filepath.ToSlash(rel)
			if rel == ".." || strings.HasPrefix(rel, "../") {
				return nil, "", fmt.Errorf("Go package source escapes repository: %s", full)
			}
			paths[rel] = struct{}{}
		}
	}
	ordered := make([]string, 0, len(paths))
	for path := range paths {
		ordered = append(ordered, path)
	}
	sort.Strings(ordered)
	members := make([]SourceMember, 0, len(ordered))
	for _, rel := range ordered {
		if _, ok := tracked[rel]; !ok {
			return nil, "", fmt.Errorf("Go package source closure member is not Git-tracked: %s", rel)
		}
		path := filepath.Join(repo, filepath.FromSlash(rel))
		info, err := os.Lstat(path)
		if err != nil {
			return nil, "", err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return nil, "", fmt.Errorf("Go package source closure member is not a regular file: %s", rel)
		}
		digest, err := digestSourceFile(path)
		if err != nil {
			return nil, "", err
		}
		members = append(members, SourceMember{Path: rel, Digest: digest})
	}
	return members, SourceClosureDigest(members), nil
}

func trackedSourcePaths(repo string) (map[string]struct{}, error) {
	cmd := exec.Command("git", "-C", repo, "ls-files", "-z")
	raw, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("read Git-tracked source paths: %w", err)
	}
	out := map[string]struct{}{}
	for _, path := range strings.Split(string(raw), "\x00") {
		if path == "" {
			continue
		}
		out[filepath.ToSlash(path)] = struct{}{}
	}
	return out, nil
}

func digestSourceFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(sum.Sum(nil)), nil
}
