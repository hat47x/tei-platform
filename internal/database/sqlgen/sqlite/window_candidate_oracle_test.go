package sqlite

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// pinnedWindowCandidateSource matches raw bytes, not parsed Markdown meaning.
// It prevents an edited fixture from silently retaining old expected values
// while the production Markdown Reader is not yet connected.
func pinnedWindowCandidateSource(source []byte, sha256Hex string) bool {
	digest := sha256.Sum256(source)
	return len(sha256Hex) == 64 && fmt.Sprintf("%x", digest) == sha256Hex
}

// TestCTKWindowCandidateIndependentOracleAgainstSQLite does not interpret the
// Markdown source as meaning. It first verifies exact Source bytes, then tests
// fixed independent Expected values against generated SQLite SQL. Passing is
// C0 evidence, not acceptance of a *.tei.md Reader or Query/Provider chain.
func TestCTKWindowCandidateIndependentOracleAgainstSQLite(t *testing.T) {
	type oracleRow struct {
		ObsID           string  `json:"obs_id"`
		Moving          float64 `json:"moving"`
		CumulativeRows  float64 `json:"cumulative_rows"`
		CumulativeRange float64 `json:"cumulative_range"`
	}
	type oracleCase struct {
		File                string      `json:"file"`
		SourceSHA256        string      `json:"sourceSHA256"`
		Valid               bool        `json:"valid"`
		MovingPrecedingRows int         `json:"movingPrecedingRows"`
		Outputs             []oracleRow `json:"outputs"`
	}
	var oracle struct {
		Schema   string       `json:"schema"`
		Adoption string       `json:"adoption"`
		Cases    []oracleCase `json:"cases"`
	}
	fixturePath := filepath.Join("..", "..", "..", "..", "testdata", "tei-md-window", "independent-expected.candidate.json")
	raw, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixed independent oracle: %v", err)
	}
	if err := json.Unmarshal(raw, &oracle); err != nil {
		t.Fatalf("decode independent oracle: %v", err)
	}
	if oracle.Schema != "tei-ctk-window-independent-expected-candidate/v1" ||
		oracle.Adoption != "proposal-only" {
		t.Fatalf("unexpected oracle identity or adoption: %q, %q", oracle.Schema, oracle.Adoption)
	}

	byFile := make(map[string]oracleCase, len(oracle.Cases))
	for _, c := range oracle.Cases {
		if c.File == "" {
			t.Fatal("unnamed candidate fixture")
		}
		if _, exists := byFile[c.File]; exists {
			t.Fatalf("duplicate oracle case %q", c.File)
		}
		byFile[c.File] = c
		// Source is not interpreted as TEI meaning. Pin its exact bytes:
		// otherwise changes to value, Frame or derived view go unnoticed.
		source, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "testdata", "tei-md-window", c.File))
		if err != nil {
			t.Fatalf("read Markdown candidate %s: %v", c.File, err)
		}
		if !pinnedWindowCandidateSource(source, c.SourceSHA256) {
			t.Fatalf("STALE_SOURCE_CLOSURE: Markdown candidate %s no longer matches its independent Expected sourceSHA256; review and re-pin expected separately", c.File)
		}
	}

	const originalFrame = "ROWS BETWEEN 2 PRECEDING AND CURRENT ROW"
	const reducedFrame = "ROWS BETWEEN 1 PRECEDING AND CURRENT ROW"
	for _, name := range []string{"baseline.tei.md", "frame-narrowed.tei.md"} {
		t.Run(name, func(t *testing.T) {
			c, ok := byFile[name]
			if !ok || !c.Valid || len(c.Outputs) != 7 {
				t.Fatalf("case %s must have seven explicitly authored expected rows", name)
			}
			if (name == "baseline.tei.md" && c.MovingPrecedingRows != 2) ||
				(name == "frame-narrowed.tei.md" && c.MovingPrecedingRows != 1) {
				t.Fatalf("unexpected moving frame for %s: %d", name, c.MovingPrecedingRows)
			}
			wantByID := make(map[string]oracleRow, len(c.Outputs))
			for _, want := range c.Outputs {
				if want.ObsID == "" {
					t.Fatal("expected observation with empty ID")
				}
				if _, dup := wantByID[want.ObsID]; dup {
					t.Fatalf("duplicate expected observation %s", want.ObsID)
				}
				wantByID[want.ObsID] = want
			}

			opts := ctkWindowSelect()
			if strings.Count(opts.Window, originalFrame) != 1 {
				t.Fatalf("SQLite reference has unexpected frame count: %s", opts.Window)
			}
			if c.MovingPrecedingRows == 1 {
				opts.Window = strings.Replace(opts.Window, originalFrame, reducedFrame, 1)
			}
			generated, err := NewDialect().Select(context.Background(), opts)
			if err != nil {
				t.Fatalf("generate SQLite SQL: %v", err)
			}
			db := openCTKSQLite(t)
			rows, err := db.QueryContext(context.Background(), generated.Statement, generated.Args...)
			if err != nil {
				t.Fatalf("execute generated SQLite SQL: %v\n%s", err, generated.Statement)
			}
			defer rows.Close()

			seen := make(map[string]bool, 7)
			for rows.Next() {
				var group, id string
				var seq int64
				var value sql.NullInt64
				var moving, runningRows, runningRange float64
				if err := rows.Scan(&group, &id, &seq, &value, &moving, &runningRows, &runningRange); err != nil {
					t.Fatalf("scan SQL row: %v", err)
				}
				want, ok := wantByID[id]
				if !ok || seen[id] {
					t.Fatalf("unexpected or repeated observation ID %q", id)
				}
				seen[id] = true
				if moving != want.Moving || runningRows != want.CumulativeRows ||
					runningRange != want.CumulativeRange {
					t.Errorf("%s = (%v,%v,%v), independent Expected (%v,%v,%v)",
						id, moving, runningRows, runningRange,
						want.Moving, want.CumulativeRows, want.CumulativeRange)
				}
				if group == "" || seq == 0 || value.Valid != (id != "B2") {
					t.Errorf("source value/identity changed for %s: group=%s seq=%d value=%+v", id, group, seq, value)
				}
			}
			if err := rows.Err(); err != nil {
				t.Fatalf("iterate SQL rows: %v", err)
			}
			if len(seen) != len(c.Outputs) {
				t.Fatalf("observed %d rows, Expected %d", len(seen), len(c.Outputs))
			}
		})
	}
}

// Before a formal Markdown Reader exists, changes to the authoritative
// observations or derived display invalidate the raw Source pin. That alone
// does not determine which changes affect accepted Meaning.
func TestCTKWindowSourcePinRejectsUnreviewedChanges(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "testdata", "tei-md-window")
	fixture, err := os.ReadFile(filepath.Join(root, "baseline.tei.md"))
	if err != nil {
		t.Fatal(err)
	}
	var oracle struct {
		Cases []struct {
			File         string `json:"file"`
			SourceSHA256 string `json:"sourceSHA256"`
		} `json:"cases"`
	}
	b, err := os.ReadFile(filepath.Join(root, "independent-expected.candidate.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &oracle); err != nil {
		t.Fatal(err)
	}
	var pinned string
	for _, c := range oracle.Cases {
		if c.File == "baseline.tei.md" {
			pinned = c.SourceSHA256
			break
		}
	}
	if !pinnedWindowCandidateSource(fixture, pinned) {
		t.Fatal("baseline Source is not pinned to independent Expected candidate")
	}
	for _, change := range []struct {
		name, old, newer string
	}{
		{"observation value", "| A3 | A | 3 | 30 |", "| A3 | A | 3 | 300 |"},
		{"display-only text", "# 系列別Window平均：基準", "# 系列別Window平均：基準（編集）"},
	} {
		t.Run(change.name, func(t *testing.T) {
			mutated := bytes.Replace(fixture, []byte(change.old), []byte(change.newer), 1)
			if bytes.Equal(mutated, fixture) {
				t.Fatalf("fixture lacks controlled mutation %q", change.old)
			}
			if pinnedWindowCandidateSource(mutated, pinned) {
				t.Fatalf("changed Source bytes unexpectedly retain old pin: %s", change.name)
			}
		})
	}
}
