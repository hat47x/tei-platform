package externaladoptionauthority

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Tools are the existing asset CLIs, built unchanged from this checkout.
// Both stage 1 and stage 2 assets are `package main`; their functions cannot
// be called from outside, so they are called as programs.
type Tools struct {
	SUIAdoption string // tools/reference-sui-adoption
	Catalog     string // tools/reference-semantic-extension-catalog
}

// BuildTools builds the two CLIs with the given Go executable. The caller's
// environment decides GOFLAGS, GOPROXY and GOTOOLCHAIN; nothing is fetched
// when GOPROXY=off.
func BuildTools(ctx context.Context, repo, goExe, dir string) (Tools, error) {
	if goExe == "" {
		return Tools{}, errors.New("Go executable is required to build the asset CLIs")
	}
	t := Tools{
		SUIAdoption: filepath.Join(dir, "reference-sui-adoption"),
		Catalog:     filepath.Join(dir, "reference-semantic-extension-catalog"),
	}
	for out, pkg := range map[string]string{
		t.SUIAdoption: "./tools/reference-sui-adoption",
		t.Catalog:     "./tools/reference-semantic-extension-catalog",
	} {
		cmd := exec.CommandContext(ctx, goExe, "build", "-o", out, pkg)
		cmd.Dir = repo
		if raw, err := cmd.CombinedOutput(); err != nil {
			return Tools{}, fmt.Errorf("go build %s: %v: %s", pkg, err, raw)
		}
	}
	return t, nil
}

// CLIResult is one program run.
type CLIResult struct {
	Args   []string `json:"args"`
	Exit   int      `json:"exit"`
	Stdout []byte   `json:"-"`
	Stderr string   `json:"stderr,omitempty"`
}

func runCLI(ctx context.Context, bin, dir string, args ...string) (CLIResult, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = dir
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	res := CLIResult{Args: args, Stdout: out.Bytes(), Stderr: strings.TrimSpace(errb.String())}
	var exit *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exit):
		res.Exit = exit.ExitCode()
	default:
		return res, fmt.Errorf("run %s: %w", filepath.Base(bin), err)
	}
	return res, nil
}

// copyTree copies regular files under src into dst. It never follows a
// symlink and never writes into src.
func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("%s is not a regular file", path)
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// treeDigest is the digest of the sorted (slash path, file digest) list.
func treeDigest(root string) (string, error) {
	type entry struct {
		Path   string `json:"path"`
		Digest string `json:"digest"`
	}
	var entries []entry
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		entries = append(entries, entry{Path: filepath.ToSlash(rel), Digest: digestBytes(raw)})
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return digestJSON(entries)
}

func writeFile(path string, raw []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}
