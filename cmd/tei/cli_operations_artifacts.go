package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tei/internal/artifact"
	"tei/internal/operations"
)

// runOperationsCommand prints the derived operation surface. Gaps are part of
// the output: an operation that could not be derived is shown as a gap, not as
// a success.
func runOperationsCommand(workspacePath, applicationID string) error {
	if workspacePath == "" {
		return fmt.Errorf("operations requires -workspace")
	}
	surface, err := operations.SurfaceWorkspace(workspacePath, applicationID)
	if err != nil {
		return err
	}
	writeJSON(surface)
	return nil
}

// runArtifactsCommand prints the derived artifacts and their manifest. Findings
// from the consistency check are part of the output; an incomplete set is
// reported as such. With -out, one file per artifact and a manifest file are
// written instead, and the manifest is printed.
func runArtifactsCommand(workspacePath, applicationID, outDir string) error {
	if workspacePath == "" {
		return fmt.Errorf("artifacts requires -workspace")
	}
	if applicationID == "" {
		return fmt.Errorf("artifacts requires -application")
	}
	set, err := artifact.RenderWorkspace(workspacePath, applicationID)
	if err != nil {
		return err
	}
	if outDir == "" {
		writeJSON(set)
		return nil
	}
	if err := writeArtifactFiles(outDir, set); err != nil {
		return err
	}
	writeJSON(set.Manifest)
	return nil
}

// writeArtifactFiles writes one file per artifact plus the manifest. Names come
// from the content identity, so the same meaning writes the same file names and
// an unchanged artifact keeps its file.
func writeArtifactFiles(outDir string, set artifact.WorkspaceSet) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	names := map[string]string{}
	for _, item := range set.Artifacts {
		name, body := artifactFileName(item)
		if previous, duplicated := names[name]; duplicated && previous != item.Digest {
			return fmt.Errorf("two artifacts want the same file name %s; refusing to overwrite", name)
		}
		names[name] = item.Digest
		if err := os.WriteFile(filepath.Join(outDir, name), []byte(body), 0o644); err != nil {
			return err
		}
	}
	manifest, err := json.MarshalIndent(struct {
		Manifest artifact.Manifest  `json:"manifest"`
		Findings []artifact.Finding `json:"findings"`
	}{Manifest: set.Manifest, Findings: set.Findings}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "manifest.json"), append(manifest, '\n'), 0o644)
}

// artifactFileName names the file after what the artifact describes. A table
// definition is named by its table, a function definition by its application and
// the operation surface by its kind.
func artifactFileName(item artifact.Artifact) (string, string) {
	header := strings.TrimSpace(strings.SplitN(item.Content, "\n", 2)[0])
	switch artifact.Kind(item.Kind) {
	case artifact.KindTableDefinition:
		if name := strings.TrimPrefix(header, "table "); name != header && name != "" {
			return "table-" + slug(name) + ".md", item.Content
		}
	case artifact.KindFunctionDefinition:
		if name := strings.TrimPrefix(header, "application "); name != header && name != "" {
			return "function-" + slug(name) + ".md", item.Content
		}
	case artifact.KindOperationSurface:
		return "operation-surface.json", item.Content
	}
	digest := strings.TrimPrefix(item.Digest, "sha256:")
	if len(digest) > 8 {
		digest = digest[:8]
	}
	return slug(item.Kind) + "-" + slug(digest) + ".txt", item.Content
}

func slug(value string) string {
	replaced := strings.NewReplacer(".", "-", "/", "-", " ", "-").Replace(value)
	return strings.ToLower(replaced)
}
