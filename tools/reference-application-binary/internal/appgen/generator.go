package appgen

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const templatesDir = "templates"

// Generate writes generated Go source files into outDir.
func Generate(ctx *GenerationContext, outDir string) ([]string, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir output: %w", err)
	}

	if ctx.BuildArtifactDigest == "" {
		ctx.BuildArtifactDigest = "sha256:PLACEHOLDER"
	}

	files := map[string]string{
		"main.go":       "main.go.tmpl",
		"config.go":     "config.go.tmpl",
		"models.go":     "models.go.tmpl",
		"queries.go":    "queries.go.tmpl",
		"commands.go":   "commands.go.tmpl",
		"validations.go": "validations.go.tmpl",
		"errors.go":     "errors.go.tmpl",
		"provenance.go": "provenance.go.tmpl",
	}

	var written []string
	for dst, tmpl := range files {
		src, err := os.ReadFile(filepath.Join(templatesDir, tmpl))
		if err != nil {
			return nil, fmt.Errorf("read template %s: %w", tmpl, err)
		}
		rendered, err := renderTemplate(tmpl, string(src), ctx)
		if err != nil {
			return nil, fmt.Errorf("render %s: %w", tmpl, err)
		}
		path := filepath.Join(outDir, dst)
		if err := os.WriteFile(path, []byte(rendered), 0o644); err != nil {
			return nil, fmt.Errorf("write %s: %w", path, err)
		}
		written = append(written, path)
	}

	manifest := map[string]any{
		"schemaVersion": ctx.SchemaVersion,
		"applicationId": ctx.ApplicationID,
		"files":        written,
	}
	manifestPath := filepath.Join(outDir, "manifest.json")
	manifestBytes, _ := json.MarshalIndent(manifest, "", "  ")
	if err := os.WriteFile(manifestPath, manifestBytes, 0o644); err != nil {
		return nil, fmt.Errorf("write manifest: %w", err)
	}
	written = append(written, manifestPath)
	return written, nil
}
