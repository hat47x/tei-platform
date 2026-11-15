package sqlite

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/sourceindex"
)

// This reuses the same Source Index byte/provenance primitives as the Graph
// reference test (#4742). It does NOT parse Markdown, select a semantic Role,
// create a Window Operator or accept the provisional independent Expected.
func TestCTKWindowSourcesUseExistingSourceIndex(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "testdata", "tei-md-window")
	raw, err := os.ReadFile(filepath.Join(root, "independent-expected.candidate.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Cases []struct {
			File         string `json:"file"`
			SourceSHA256 string `json:"sourceSHA256"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	// The candidate closure test owns the fixture count/classification.
	// Here each named Source gets the same Artifact/Range validation used by
	// other TEI Source Index clients, without a second provenance scheme.
	for _, c := range fixture.Cases {
		t.Run(c.File, func(t *testing.T) {
			src, err := os.ReadFile(filepath.Join(root, c.File))
			if err != nil {
				t.Fatal(err)
			}
			path := "testdata/tei-md-window/" + c.File
			artifact, err := sourceindex.NewArtifact(path, src)
			if err != nil {
				t.Fatal(err)
			}
			if artifact.Path != path || artifact.Format != "md" ||
				artifact.Fingerprint != "sha256:"+c.SourceSHA256 {
				t.Fatalf("Source Index identity differs from independent Source pin: %+v", artifact)
			}
			index := sourceindex.Index{Artifacts: []sourceindex.Artifact{artifact}}
			read := func(requested string) ([]byte, error) {
				if requested != path {
					return nil, fmt.Errorf("unrecognized source path %q", requested)
				}
				return src, nil
			}
			if err := index.Validate(read); err != nil {
				t.Fatalf("exact Window candidate Source did not validate: %v", err)
			}

			// A known literal in the observation *fixture* gives a byte-span
			// control; finding it is NOT semantic table-cell selection.
			const literal = "| B2 | B | 2 | NULL |"
			at := bytes.Index(src, []byte(literal))
			if at < 0 {
				t.Fatal("required fixture byte witness is missing")
			}
			rng, err := sourceindex.RangeFromOffsets(src, at, at+len(literal))
			if err != nil {
				t.Fatal(err)
			}
			if err := rng.Validate(src); err != nil {
				t.Fatal(err)
			}
			if rng.Start.Offset != at || rng.End.Offset != at+len(literal) ||
				string(src[rng.Start.Offset:rng.End.Offset]) != literal {
				t.Fatalf("Source Index returned an incorrect raw byte range: %+v", rng)
			}
			// Changing only a narrative suffix still invalidates exact Source
			// reuse. Whether Model meaning remains equal is a separate review.
			changed := append(append([]byte(nil), src...), []byte("\nUpdated explanation\n")...)
			if err := index.Validate(func(requested string) ([]byte, error) {
				if requested != path {
					return nil, fmt.Errorf("unrecognized source path %q", requested)
				}
				return changed, nil
			}); err == nil {
				t.Fatal("stale source index was accepted after raw Source changed")
			}
		})
	}
}
