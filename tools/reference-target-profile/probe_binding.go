package targetprofile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	generatorselection "tei/tools/reference-generator-selection"
)

// snapshotProbeSource resolves the receipt's identity, rather than assuming
// that a directory has the same name. Only verified single-file bytes are
// copied to the private probe directory; later edits of the candidate are not
// consumed by its compiler or interpreter.
func snapshotProbeSource(candidatesDir, snapshotDir string, selected generatorselection.CandidateReceipt) (generatorselection.CandidateManifest, error) {
	var manifest generatorselection.CandidateManifest
	if !selected.Passed || selected.ID == "" || selected.SourceDigest == "" {
		return manifest, fmt.Errorf("selected candidate has incomplete source evidence")
	}
	entries, err := os.ReadDir(candidatesDir)
	if err != nil {
		return manifest, err
	}
	candidateDir := ""
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(candidatesDir, entry.Name())
		raw, err := os.ReadFile(filepath.Join(dir, "candidate.json"))
		if err != nil {
			return manifest, err
		}
		var candidate generatorselection.CandidateManifest
		if err := json.Unmarshal(raw, &candidate); err != nil {
			return manifest, err
		}
		if candidate.ID != selected.ID {
			continue
		}
		if candidateDir != "" {
			return manifest, fmt.Errorf("selected candidate identity is ambiguous")
		}
		candidateDir, manifest = dir, candidate
	}
	if candidateDir == "" {
		return manifest, fmt.Errorf("selected candidate identity was not found")
	}
	language := manifest.Language
	if language == "" {
		language = generatorselection.LanguageGo
	}
	if language != selected.Language {
		return manifest, fmt.Errorf("selected candidate language changed")
	}
	if manifest.SourceDigest != selected.SourceDigest {
		return manifest, fmt.Errorf("candidate source digest differs from selection")
	}
	extension := ""
	switch language {
	case generatorselection.LanguageGo:
		extension = ".go"
		if manifest.Source != "main.go" {
			return manifest, fmt.Errorf("Go probe supports only the selected main.go source")
		}
	case generatorselection.LanguagePython:
		extension = ".py"
	case generatorselection.LanguageJavaScript:
		extension = ".js"
	default:
		return manifest, fmt.Errorf("unsupported long-running candidate language: %s", language)
	}
	if strings.ContainsAny(manifest.Source, "/\\") || filepath.Ext(manifest.Source) != extension {
		return manifest, fmt.Errorf("probe requires a single source file matching its language")
	}
	sourcePath := filepath.Join(candidateDir, manifest.Source)
	info, err := os.Lstat(sourcePath)
	if err != nil {
		return manifest, err
	}
	if !info.Mode().IsRegular() {
		return manifest, fmt.Errorf("probe source must be a regular file")
	}
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return manifest, err
	}
	if digestBytes(source) != selected.SourceDigest {
		return manifest, fmt.Errorf("actual source digest differs from selection")
	}
	if err := os.WriteFile(filepath.Join(snapshotDir, manifest.Source), source, 0600); err != nil {
		return manifest, err
	}
	return manifest, nil
}
