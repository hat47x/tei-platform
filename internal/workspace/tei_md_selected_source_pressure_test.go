package workspace

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/sourceindex"
	teimdreader "tei/reference/tei-md-graph-reader"
)

// This is deliberately a test-only preflight, not a product Markdown
// admission path. Load must continue to reject an unbound *.tei.md parser.
func TestTEIMDManifestScopedPinnedCandidatePressure(t *testing.T) {
	for _, fixture := range []string{"baseline.tei.md", "rule-by-target.tei.md"} {
		t.Run(fixture, func(t *testing.T) {
			original, err := os.ReadFile(filepath.Join("..", "..", "testdata", "tei-md-graph", fixture))
			if err != nil {
				t.Fatal(err)
			}
			root := t.TempDir()
			const selectedPath = "semantic/model.tei.md"
			const outsidePath = "drafts/not-declared.tei.md"
			for _, dir := range []string{"semantic", "drafts"} {
				if err := os.MkdirAll(filepath.Join(root, dir), 0o700); err != nil {
					t.Fatal(err)
				}
			}
			const manifestText = "workspace: tei_md_candidate\nformatVersion: 1\nsources:\n  - semantic\n"
			if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifestText), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(selectedPath)), original, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(outsidePath)), original, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "semantic", "README.md"), []byte("# documentation only\n"), 0o600); err != nil {
				t.Fatal(err)
			}

			// Product admission has NOT been made: parsing never starts.
			if _, err := Load(root); err == nil ||
				!strings.Contains(err.Error(), "TEI_MD_PARSER_UNBOUND") ||
				!strings.Contains(err.Error(), selectedPath) {
				t.Fatalf("unbound selected Markdown did not fail closed: %v", err)
			}

			// Rehearse the existing manifest/root/read capabilities with one
			// pinned Root, not a second parser or an unscoped pathname read.
			pinned, err := openWorkspaceRoot(root)
			if err != nil {
				t.Fatal(err)
			}
			defer pinned.Close()
			manifestBytes, err := readPinnedWorkspaceFile(pinned, "tei.workspace.yaml")
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(manifestBytes, []byte(manifestText)) {
				t.Fatal("manifest read from pinned root differed")
			}
			manifest, err := parseWorkspaceManifestYAML(string(manifestBytes))
			if err != nil {
				t.Fatal(err)
			}
			if manifest.FormatVersion != 1 || len(manifest.Sources) != 1 ||
				manifest.Sources[0] != "semantic" {
				t.Fatalf("unexpected selected source roots: %+v", manifest)
			}
			if err := validateWorkspaceSourceRoots(manifest.Sources); err != nil {
				t.Fatal(err)
			}
			inventory, err := scanManagedFiles(pinned, manifest.Sources)
			if err != nil {
				t.Fatal(err)
			}
			var selected, outside bool
			for _, item := range inventory {
				if item.Path == selectedPath && item.SourceRoot == "semantic" {
					selected = true
				}
				if item.Path == outsidePath {
					outside = true
				}
			}
			if !selected || outside {
				t.Fatalf("managed inventory did not preserve selection: %+v", inventory)
			}
			selectedBytes, err := readPinnedWorkspaceFile(pinned, selectedPath)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(selectedBytes, original) {
				t.Fatal("pinned read did not retain the authored bytes")
			}
			graphRule, err := teimdreader.ParseRuleCandidate(selectedBytes)
			if err != nil {
				t.Fatal(err)
			}
			if len(graphRule.Graph.Nodes) != 2 || len(graphRule.Graph.Edges) != 2 ||
				graphRule.Rule.ID != "node.edge_ids" {
				t.Fatalf("reference candidate lost authored data: %+v", graphRule)
			}

			// Current Source Index can carry exact bytes and ranges but has
			// no selected Language/Role or accepted semantic Subject here.
			artifact, err := sourceindex.NewArtifact(selectedPath, selectedBytes)
			if err != nil {
				t.Fatal(err)
			}
			if artifact.Format != "md" || artifact.Fingerprint != sourceindex.Fingerprint(original) {
				t.Fatalf("physical artifact identity drifted: %+v", artifact)
			}
			span := graphRule.Rule.ConditionSource
			location, err := sourceindex.RangeFromOffsets(selectedBytes, span.Start, span.End)
			if err != nil {
				t.Fatal(err)
			}
			if location.Start.Line != span.Line {
				t.Fatalf("raw/canonical line differs: %d vs %d", span.Line, location.Start.Line)
			}
			if err := location.Validate(selectedBytes); err != nil {
				t.Fatal(err)
			}
			index := sourceindex.Index{Artifacts: []sourceindex.Artifact{artifact}}
			readPinned := func(p string) ([]byte, error) {
				return readPinnedWorkspaceFile(pinned, p)
			}
			if err := index.Validate(readPinned); err != nil {
				t.Fatalf("original pinned Source Index: %v", err)
			}

			// In-place mutation after the read invalidates the old evidence.
			updated := append(bytes.Clone(original), []byte("\n# Changed documentation\n")...)
			if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(selectedPath)), updated, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := index.Validate(readPinned); err == nil ||
				!strings.Contains(err.Error(), "fingerprint is stale") {
				t.Fatalf("old Index accepted changed bytes: %v", err)
			}
			// A fresh product Load still cannot silently use either version.
			if _, err := Load(root); err == nil ||
				!strings.Contains(err.Error(), "TEI_MD_PARSER_UNBOUND") {
				t.Fatalf("mutation promoted an unbound Markdown parser: %v", err)
			}
		})
	}
}
