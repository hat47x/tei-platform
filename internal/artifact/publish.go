package artifact

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ArtifactFileName names the file after what the artifact describes. A table
// definition is named by its table, a function definition by its application and
// the operation surface by its kind.
func ArtifactFileName(item Artifact) (string, string) {
	header := strings.TrimSpace(strings.SplitN(item.Content, "\n", 2)[0])
	switch Kind(item.Kind) {
	case KindTableDefinition:
		if name := strings.TrimPrefix(header, "table "); name != header && name != "" {
			return "table-" + Slug(name) + ".md", item.Content
		}
	case KindFunctionDefinition:
		if name := strings.TrimPrefix(header, "application "); name != header && name != "" {
			return "function-" + Slug(name) + ".md", item.Content
		}
	case KindOperationSurface:
		return "operation-surface.json", item.Content
	}
	digest := strings.TrimPrefix(item.Digest, "sha256:")
	if len(digest) > 8 {
		digest = digest[:8]
	}
	return Slug(item.Kind) + "-" + Slug(digest) + ".txt", item.Content
}

// Slug replaces path separators and spaces with hyphens and lowercases the
// result so it is safe for file names.
func Slug(value string) string {
	replaced := strings.NewReplacer(".", "-", "/", "-", " ", "-").Replace(value)
	return strings.ToLower(replaced)
}

// WriteManifest writes the manifest envelope to the output directory.
func WriteManifest(outDir string, set WorkspaceSet) error {
	manifest, err := json.MarshalIndent(struct {
		Manifest Manifest  `json:"manifest"`
		Findings []Finding `json:"findings"`
	}{Manifest: set.Manifest, Findings: set.Findings}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "manifest.json"), append(manifest, '\n'), 0o644)
}

// ReadPublishedManifest reads the manifest envelope from the output directory
// and returns the manifest and findings it lists.
func ReadPublishedManifest(outDir string) (Manifest, []Finding, error) {
	raw, err := os.ReadFile(filepath.Join(outDir, "manifest.json"))
	if err != nil {
		return Manifest{}, nil, err
	}
	var envelope struct {
		Manifest Manifest  `json:"manifest"`
		Findings []Finding `json:"findings"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return Manifest{}, nil, fmt.Errorf("manifest.json is not valid JSON: %w", err)
	}
	return envelope.Manifest, envelope.Findings, nil
}
