package orderruledesign

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

// Tools are the existing asset commands, built unchanged from this checkout.
// Every asset command is `package main`; its functions cannot be called from
// outside, so it is called as a program.
type Tools struct {
	Tei              string // cmd/tei
	SourceAdapter    string // tools/reference-xead-source-adapter
	CrossView        string // tools/reference-xead-cross-view
	SharedAcceptance string // tools/reference-shared-acceptance
	BusinessDFD      string // tools/reference-business-dfd
	SubsystemCUD     string // tools/reference-subsystem-cud
	SpecReview       string // tools/reference-spec-review
	PurposePreflight string // tools/reference-purpose-preflight
	ChangeAssurance  string // tools/reference-change-assurance
	Env              []string
	GoExe            string
	Node             string
}

var toolPackages = []struct {
	field func(*Tools) *string
	name  string
	pkg   string
}{
	{func(t *Tools) *string { return &t.Tei }, "tei", "./cmd/tei"},
	{func(t *Tools) *string { return &t.SourceAdapter }, "reference-xead-source-adapter", "./tools/reference-xead-source-adapter"},
	{func(t *Tools) *string { return &t.CrossView }, "reference-xead-cross-view", "./tools/reference-xead-cross-view"},
	{func(t *Tools) *string { return &t.SharedAcceptance }, "reference-shared-acceptance", "./tools/reference-shared-acceptance"},
	{func(t *Tools) *string { return &t.BusinessDFD }, "reference-business-dfd", "./tools/reference-business-dfd"},
	{func(t *Tools) *string { return &t.SubsystemCUD }, "reference-subsystem-cud", "./tools/reference-subsystem-cud"},
	{func(t *Tools) *string { return &t.SpecReview }, "reference-spec-review", "./tools/reference-spec-review"},
	{func(t *Tools) *string { return &t.PurposePreflight }, "reference-purpose-preflight", "./tools/reference-purpose-preflight"},
	{func(t *Tools) *string { return &t.ChangeAssurance }, "reference-change-assurance", "./tools/reference-change-assurance"},
}

// ChildEnv is the environment of every child process: the Go executable of
// the local GOROOT first on PATH, nothing fetched.
func ChildEnv(goExe string) []string {
	env := os.Environ()
	out := make([]string, 0, len(env)+5)
	for _, kv := range env {
		switch {
		case strings.HasPrefix(kv, "PATH="), strings.HasPrefix(kv, "GOPROXY="), strings.HasPrefix(kv, "GOFLAGS="),
			strings.HasPrefix(kv, "GOTOOLCHAIN="), strings.HasPrefix(kv, "GOWORK="), strings.HasPrefix(kv, "GIT_"):
		default:
			out = append(out, kv)
		}
	}
	path := os.Getenv("PATH")
	if goExe != "" {
		path = filepath.Dir(goExe) + string(os.PathListSeparator) + path
	}
	return append(out, "PATH="+path, "GOPROXY=off", "GOFLAGS=-mod=readonly", "GOTOOLCHAIN=local", "GOWORK=off")
}

// BuildTools builds the asset commands with the given Go executable.
func BuildTools(ctx context.Context, repo, goExe, dir string) (Tools, error) {
	if goExe == "" {
		return Tools{}, errors.New("Go executable is required to build the asset commands")
	}
	t := Tools{GoExe: goExe, Env: ChildEnv(goExe)}
	if node, err := exec.LookPath("node"); err == nil {
		t.Node = node
	} else {
		return Tools{}, errors.New("node is required for reference/studio-authoring and reference/business-spec-templates")
	}
	for _, p := range toolPackages {
		out := filepath.Join(dir, p.name)
		*p.field(&t) = out
		cmd := exec.CommandContext(ctx, goExe, "build", "-o", out, p.pkg)
		cmd.Dir = repo
		cmd.Env = t.Env
		if raw, err := cmd.CombinedOutput(); err != nil {
			return Tools{}, fmt.Errorf("go build %s: %v: %s", p.pkg, err, raw)
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

func runCLI(ctx context.Context, env []string, bin, dir string, args ...string) (CLIResult, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = dir
	cmd.Env = env
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

// mustRun runs a command and fails on a nonzero exit.
func (t Tools) mustRun(ctx context.Context, bin, dir string, args ...string) ([]byte, error) {
	res, err := runCLI(ctx, t.Env, bin, dir, args...)
	if err != nil {
		return nil, err
	}
	if res.Exit != 0 {
		return res.Stdout, fmt.Errorf("%s %s: exit %d: %s", filepath.Base(bin), strings.Join(args, " "), res.Exit, res.Stderr)
	}
	return res.Stdout, nil
}

// git runs git with a fixed identity and fixed commit dates. The wall clock
// is never read, so a commit hash is a function of its content.
func (t Tools) git(ctx context.Context, dir string, args ...string) (string, error) {
	full := append([]string{"-c", "user.name=order-rule-design", "-c", "user.email=order-rule-design@example.invalid",
		"-c", "commit.gpgsign=false", "-c", "core.autocrlf=false", "-c", "core.safecrlf=false", "-c", "advice.detachedHead=false"}, args...)
	cmd := exec.CommandContext(ctx, "git", full...)
	cmd.Dir = dir
	env := append([]string{}, t.Env...)
	env = append(env, "GIT_AUTHOR_DATE=2026-10-29T00:00:00+0000", "GIT_COMMITTER_DATE=2026-10-29T00:00:00+0000",
		"GIT_TERMINAL_PROMPT=0", "GIT_CONFIG_NOSYSTEM=1")
	cmd.Env = env
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return strings.TrimSpace(out.String()), fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(errb.String()))
	}
	return strings.TrimSpace(out.String()), nil
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

func fileDigest(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return digestBytes(raw), nil
}

func writeFile(path string, raw []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func removeAll(path string) error { return os.RemoveAll(path) }
