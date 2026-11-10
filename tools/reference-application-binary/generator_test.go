package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hat47x/tei-platform/tools/reference-application-binary/internal/appgen"
)

func TestExtractOrderEntryModel(t *testing.T) {
	repo := repoRoot(t)
	workspace := filepath.Join(repo, "reference", "workspace")
	ctx, err := appgen.ExtractModel(workspace, "order_entry")
	if err != nil {
		t.Fatalf("ExtractModel: %v", err)
	}
	if ctx == nil {
		t.Fatal("ctx is nil")
	}
	if ctx.ApplicationID != "order_entry" {
		t.Fatalf("application id: %s", ctx.ApplicationID)
	}
	if ctx.ModelRevision == "" {
		t.Fatal("model revision is empty")
	}
	if ctx.SourceRevision == "" {
		t.Fatal("source revision is empty")
	}
	if len(ctx.Queries) == 0 {
		t.Fatal("expected queries")
	}
	if len(ctx.Commands) == 0 {
		t.Fatal("expected commands")
	}
}

func TestGenerateWritesSources(t *testing.T) {
	repo := repoRoot(t)
	workspace := filepath.Join(repo, "reference", "workspace")
	ctx, err := appgen.ExtractModel(workspace, "order_entry")
	if err != nil {
		t.Fatalf("ExtractModel: %v", err)
	}

	out := t.TempDir()
	files, err := appgen.Generate(ctx, out)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected generated files")
	}
	for _, p := range files {
		if strings.Contains(p, "..") {
			t.Fatalf("path contains ..: %s", p)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read generated file: %v", err)
		}
		if len(b) == 0 {
			t.Fatalf("empty generated file: %s", p)
		}
	}
}

func TestBuildCreatesRunnableBinary(t *testing.T) {
	repo := repoRoot(t)
	workspace := filepath.Join(repo, "reference", "workspace")
	ctx, err := appgen.ExtractModel(workspace, "order_entry")
	if err != nil {
		t.Fatalf("ExtractModel: %v", err)
	}

	sourceDir := t.TempDir()
	_, err = appgen.Generate(ctx, sourceDir)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	binary := filepath.Join(t.TempDir(), "tei-app-order-entry")
	if filepath.Ext(binary) == "" && (os.Getenv("GOOS") == "windows" || runtime.GOOS == "windows") {
		binary += ".exe"
	}

	digest, err := appgen.Build(workspace, "order_entry", sourceDir, binary, "")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.HasPrefix(digest, "sha256:") {
		t.Fatalf("unexpected digest: %s", digest)
	}

	cmd := exec.Command(binary, "provenance")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run provenance: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), ctx.ApplicationID) {
		t.Fatalf("provenance missing application id: %s", out)
	}
}

func TestGeneratedBinaryUnknownCommand(t *testing.T) {
	repo := repoRoot(t)
	workspace := filepath.Join(repo, "reference", "workspace")
	ctx, err := appgen.ExtractModel(workspace, "order_entry")
	if err != nil {
		t.Fatalf("ExtractModel: %v", err)
	}

	sourceDir := t.TempDir()
	_, err = appgen.Generate(ctx, sourceDir)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	binary := filepath.Join(t.TempDir(), "tei-app-order-entry")
	if filepath.Ext(binary) == "" && (os.Getenv("GOOS") == "windows" || runtime.GOOS == "windows") {
		binary += ".exe"
	}

	_, err = appgen.Build(workspace, "order_entry", sourceDir, binary, "")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	cmd := exec.Command(binary, "unknown")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected error for unknown command, got: %s", out)
	}
	if !strings.Contains(string(out), "unknown command") {
		t.Fatalf("unexpected error message: %s", out)
	}
}

func TestCrossCompileLinuxAmd64(t *testing.T) {
	if os.Getenv("GOOS") != "" || os.Getenv("GOARCH") != "" {
		t.Skip("cross-compilation test must run with native GOOS/GOARCH")
	}

	repo := repoRoot(t)
	workspace := filepath.Join(repo, "reference", "workspace")
	ctx, err := appgen.ExtractModel(workspace, "order_entry")
	if err != nil {
		t.Fatalf("ExtractModel: %v", err)
	}

	sourceDir := t.TempDir()
	_, err = appgen.Generate(ctx, sourceDir)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	binary := filepath.Join(t.TempDir(), "tei-app-order-entry-linux-amd64")
	digest, err := appgen.Build(workspace, "order_entry", sourceDir, binary, "linux/amd64")
	if err != nil {
		t.Fatalf("Build linux/amd64: %v", err)
	}
	if !strings.HasPrefix(digest, "sha256:") {
		t.Fatalf("unexpected digest: %s", digest)
	}
	if _, err := os.Stat(binary); err != nil {
		t.Fatalf("linux/amd64 binary missing: %v", err)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			parent := filepath.Dir(wd)
			if parent != wd && filepath.Base(parent) == "tools" {
				wd = filepath.Dir(parent)
			}
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			t.Fatal("repo root not found")
		}
		wd = parent
	}
}
