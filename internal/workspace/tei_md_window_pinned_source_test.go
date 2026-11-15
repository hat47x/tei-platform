package workspace

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/sourceindex"
)

// Window shares the product Workspace's selected Manifest Root and pinned
// bytes without adding a Window-specific Loader or granting Markdown Meaning.
// This is an S0-P test-only preflight, not the missing S0-A Reader connection.
func TestTEIMDWindowManifestPinnedSourcePressure(t *testing.T) {
	fixtureRoot := filepath.Join("..", "..", "testdata", "tei-md-window")
	oracleBytes, err := os.ReadFile(filepath.Join(fixtureRoot, "independent-expected.candidate.json"))
	if err != nil {
		t.Fatal(err)
	}
	var oracle struct {
		Schema   string `json:"schema"`
		Adoption string `json:"adoption"`
		Cases    []struct {
			File         string `json:"file"`
			SourceSHA256 string `json:"sourceSHA256"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(oracleBytes, &oracle); err != nil {
		t.Fatal(err)
	}
	if oracle.Schema != "tei-ctk-window-independent-expected-candidate/v2" ||
		oracle.Adoption != "proposal-only" || len(oracle.Cases) != 5 {
		t.Fatalf("unexpected provisional Window oracle: schema=%q adoption=%q cases=%d",
			oracle.Schema, oracle.Adoption, len(oracle.Cases))
	}
	pins := make(map[string]string, len(oracle.Cases))
	for _, c := range oracle.Cases {
		if c.File == "" || c.SourceSHA256 == "" || pins[c.File] != "" {
			t.Fatalf("invalid Window Source pin: %+v", c)
		}
		pins[c.File] = "sha256:" + c.SourceSHA256
	}

	// All five sources are selectable at the physical S0-P layer, even when
	// their proposed meaning is invalid or their derived display is stale.
	// The product Reader remains unbound for every one of them.
	candidates := []struct {
		file, frame, extraWitness string
	}{
		{"baseline.tei.md", "ROWS BETWEEN 2 PRECEDING AND CURRENT ROW", ""},
		{"frame-narrowed.tei.md", "ROWS BETWEEN 1 PRECEDING AND CURRENT ROW", ""},
		{"layout-only.tei.md", "ROWS BETWEEN 2 PRECEDING AND CURRENT ROW", ""},
		{"duplicate-observation-id.tei.md", "ROWS BETWEEN 2 PRECEDING AND CURRENT ROW", "| A3 | A | 3 | 35 |"},
		{"derived-view-stale.tei.md", "ROWS BETWEEN 1 PRECEDING AND CURRENT ROW", "| A3 | 20 |"},
	}
	if len(pins) != len(candidates) {
		t.Fatalf("expected a closed five-case Window Source inventory, got %d", len(pins))
	}
	for _, candidate := range candidates {
		if pins[candidate.file] == "" {
			t.Fatalf("missing independent Source pin for %s", candidate.file)
		}
		t.Run(candidate.file, func(t *testing.T) {
			original, err := os.ReadFile(filepath.Join(fixtureRoot, candidate.file))
			if err != nil {
				t.Fatal(err)
			}
			const selectedPath = "semantic/window.tei.md"
			const outsidePath = "drafts/window.tei.md"
			root := t.TempDir()
			for _, dir := range []string{"semantic", "drafts"} {
				if err := os.MkdirAll(filepath.Join(root, dir), 0o700); err != nil {
					t.Fatal(err)
				}
			}
			const manifestText = "workspace: window_s0_candidate\nformatVersion: 1\nsources:\n  - semantic\n"
			if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifestText), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(selectedPath)), original, 0o600); err != nil {
				t.Fatal(err)
			}
			// An unselected, same-extension draft must not become managed
			// simply because it shares a name or contains valid-looking rows.
			if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(outsidePath)), original, 0o600); err != nil {
				t.Fatal(err)
			}

			if _, err := Load(root); err == nil ||
				!strings.Contains(err.Error(), "TEI_MD_PARSER_UNBOUND") ||
				!strings.Contains(err.Error(), selectedPath) {
				t.Fatalf("unbound Window Source was admitted: %v", err)
			}
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
				t.Fatal("manifest bytes changed during pinned read")
			}
			manifest, err := parseWorkspaceManifestYAML(string(manifestBytes))
			if err != nil {
				t.Fatal(err)
			}
			if manifest.FormatVersion != 1 || len(manifest.Sources) != 1 ||
				manifest.Sources[0] != "semantic" {
				t.Fatalf("selected Source Root changed: %+v", manifest)
			}
			if err := validateWorkspaceSourceRoots(manifest.Sources); err != nil {
				t.Fatal(err)
			}
			managed, err := scanManagedFiles(pinned, manifest.Sources)
			if err != nil {
				t.Fatal(err)
			}
			selectedCount := 0
			for _, m := range managed {
				switch m.Path {
				case selectedPath:
					if m.SourceRoot != "semantic" {
						t.Fatalf("selected Source acquired the wrong Root: %+v", m)
					}
					selectedCount++
				case outsidePath:
					t.Fatal("unselected draft was promoted to managed Source")
				}
			}
			if selectedCount != 1 {
				t.Fatalf("managed inventory contains %d selected Window Sources, want 1", selectedCount)
			}
			actual, err := readPinnedWorkspaceFile(pinned, selectedPath)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(actual, original) {
				t.Fatal("selected pinned Window bytes differ from the candidate")
			}
			artifact, err := sourceindex.NewArtifact(selectedPath, actual)
			if err != nil {
				t.Fatal(err)
			}
			if artifact.Format != "md" || artifact.Fingerprint != pins[candidate.file] {
				t.Fatalf("candidate Source was not bound to its byte pin: %+v", artifact)
			}
			index := sourceindex.Index{Artifacts: []sourceindex.Artifact{artifact}}
			readPinned := func(path string) ([]byte, error) {
				return readPinnedWorkspaceFile(pinned, path)
			}
			if err := index.Validate(readPinned); err != nil {
				t.Fatalf("selected Source Index did not validate: %v", err)
			}
			// Raw witness spans only; no Markdown table parsing or Rule/Window
			// Operator binding is performed by this test.
			for _, literal := range []string{
				"| B2 | B | 2 | NULL |",
				fmt.Sprintf("| moving | AVG(value) | grp | seq, obs_id | %s |", candidate.frame),
			} {
				if bytes.Count(actual, []byte(literal)) != 1 {
					t.Fatalf("selected candidate has no unique Source witness %q", literal)
				}
				at := bytes.Index(actual, []byte(literal))
				span, err := sourceindex.RangeFromOffsets(actual, at, at+len(literal))
				if err != nil {
					t.Fatal(err)
				}
				if err := span.Validate(actual); err != nil {
					t.Fatal(err)
				}
				if string(actual[span.Start.Offset:span.End.Offset]) != literal {
					t.Fatalf("Source Index byte span did not preserve original %q", literal)
				}
			}

			// Invalid/obsolete candidates must still be retrievable and
			// attributable to their original bytes. Their extra witness
			// remains lexical evidence, never an accepted Meaning claim.
			if candidate.extraWitness != "" {
				literal := []byte(candidate.extraWitness)
				if bytes.Count(actual, literal) != 1 {
					t.Fatalf("negative candidate lacks its unique byte witness %q", literal)
				}
				at := bytes.Index(actual, literal)
				span, err := sourceindex.RangeFromOffsets(actual, at, at+len(literal))
				if err != nil {
					t.Fatal(err)
				}
				if err := span.Validate(actual); err != nil {
					t.Fatal(err)
				}
			}

			changed := bytes.Replace(original, []byte("| A3 | A | 3 | 30 |"),
				[]byte("| A3 | A | 3 | 300 |"), 1)
			if bytes.Equal(changed, original) {
				t.Fatal("expected A3 Source mutation was not applied")
			}
			if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(selectedPath)), changed, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := index.Validate(readPinned); err == nil ||
				!strings.Contains(err.Error(), "fingerprint is stale") {
				t.Fatalf("old Source Index remained valid after changed input: %v", err)
			}
			if _, err := Load(root); err == nil ||
				!strings.Contains(err.Error(), "TEI_MD_PARSER_UNBOUND") {
				t.Fatalf("edited candidate bypassed the unbound-Reader guard: %v", err)
			}
		})
	}
}
