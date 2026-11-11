package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/hat47x/tei-platform/tools/reference-application-binary/internal/appgen"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}
}

func run() error {
	workspace := flag.String("workspace", "", "TEI workspace root")
	application := flag.String("application", "", "Application ID")
	out := flag.String("out", "", "Output directory for generated sources")
	binary := flag.String("binary", "", "Output binary path")
	goBinary := flag.String("go", "", "Go toolchain path")
	flag.Parse()

	if *workspace == "" || *application == "" {
		return fmt.Errorf("-workspace and -application are required")
	}

	ctx, err := appgen.ExtractModel(*workspace, *application)
	if err != nil {
		return fmt.Errorf("extract model: %w", err)
	}

	generatedAt := time.Now().UTC().Format(time.RFC3339)
	if ctx.BuildArtifactDigest == "" || ctx.BuildArtifactDigest == "sha256:PLACEHOLDER" {
		ctx.BuildArtifactDigest = ""
	}

	sourceDir := *out
	if sourceDir == "" {
		sourceDir, err = os.MkdirTemp("", "tei-app-*")
		if err != nil {
			return fmt.Errorf("temp source dir: %w", err)
		}
		defer os.RemoveAll(sourceDir)
	}
	files, err := appgen.Generate(ctx, sourceDir)
	if err != nil {
		return fmt.Errorf("generate: %w", err)
	}

	binaryPath := *binary
	if binaryPath == "" {
		binaryPath = filepath.Join(sourceDir, "bin", "tei-app-"+*application)
		if filepath.Ext(binaryPath) == "" && runtime.GOOS == "windows" {
			binaryPath += ".exe"
		}
	}

	artifactDigest, err := appgen.Build(*workspace, *application, sourceDir, binaryPath, *goBinary)
	if err != nil {
		return fmt.Errorf("build: %w", err)
	}

	receipt := map[string]any{
		"schemaVersion":      "tei.application-binary.receipt/v0",
		"applicationId":      *application,
		"sourceRevision":     ctx.SourceRevision,
		"modelRevision":      ctx.ModelRevision,
		"toolchain":          ctx.Toolchain,
		"generator":          ctx.GeneratorID + "/" + ctx.GeneratorVersion,
		"artifactDigest":     artifactDigest,
		"generatedAt":        generatedAt,
		"sourceDir":          sourceDir,
		"binaryPath":         binaryPath,
		"generatedFiles":     files,
	}
	b, _ := json.MarshalIndent(receipt, "", "  ")
	fmt.Println(string(b))
	return nil
}
