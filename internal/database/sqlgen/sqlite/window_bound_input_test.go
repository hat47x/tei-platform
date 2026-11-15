package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// This is a C0 SQLite adapter test. The input observations and output
// expectations are independent, provisional JSON candidates. No production
// Markdown Reader, accepted TEI Meaning or Provider consumer is exercised.
func TestCTKWindowTypedCandidateInputsDriveRealSQLiteRows(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "testdata", "tei-md-window")
	raw, err := os.ReadFile(filepath.Join(root, "independent-expected.candidate.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Schema            string                `json:"schema"`
		Adoption          string                `json:"adoption"`
		InputObservations []ctkWindowInputRow   `json:"inputObservations"`
		Cases             []ctkWindowOracleCase `json:"cases"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Schema != "tei-ctk-window-independent-expected-candidate/v2" ||
		fixture.Adoption != "proposal-only" {
		t.Fatalf("unrecognized unaccepted candidate fixture: %s / %s", fixture.Schema, fixture.Adoption)
	}
	if err := checkCTKWindowCaseClosure(fixture.Cases); err != nil {
		t.Fatal(err)
	}
	typedRows, err := validateCTKWindowInputRows(fixture.InputObservations)
	if err != nil {
		t.Fatal(err)
	}
	cases := make(map[string]ctkWindowOracleCase, len(fixture.Cases))
	for _, c := range fixture.Cases {
		cases[c.File] = c
	}

	const originalFrame = "ROWS BETWEEN 2 PRECEDING AND CURRENT ROW"
	const narrowedFrame = "ROWS BETWEEN 1 PRECEDING AND CURRENT ROW"
	for _, tc := range []struct {
		file      string
		preceding int
	}{
		{"baseline.tei.md", 2},
		{"frame-narrowed.tei.md", 1},
	} {
		t.Run(tc.file, func(t *testing.T) {
			c := cases[tc.file]
			if !c.Valid || len(c.Outputs) != len(typedRows) || c.MovingPrecedingRows != tc.preceding {
				t.Fatalf("incomplete result oracle for %s", tc.file)
			}
			source, err := os.ReadFile(filepath.Join(root, tc.file))
			if err != nil {
				t.Fatal(err)
			}
			if !pinnedWindowCandidateSource(source, c.SourceSHA256) {
				t.Fatal("provisional source/Expected closure has expired")
			}

			db := openCTKSQLite(t)
			ctx := context.Background()
			if _, err := db.ExecContext(ctx, "CREATE TABLE observations (obs_id TEXT PRIMARY KEY, grp TEXT NOT NULL, seq INTEGER NOT NULL, value INTEGER)"); err != nil {
				t.Fatal(err)
			}
			stmt, err := db.PrepareContext(ctx, "INSERT INTO observations (obs_id, grp, seq, value) VALUES (?, ?, ?, ?)")
			if err != nil {
				t.Fatal(err)
			}
			for _, input := range fixture.InputObservations {
				row := typedRows[input.ObsID]
				if _, err := stmt.ExecContext(ctx, input.ObsID, row.Group, row.Seq, row.Value); err != nil {
					t.Fatalf("insert independently authored input %s: %v", input.ObsID, err)
				}
			}
			if err := stmt.Close(); err != nil {
				t.Fatal(err)
			}

			// Reuse the existing SQLite dialect's Window calculation, but
			// select from the real bound-input table rather than the original
			// hard-coded VALUES CTE. This is still not TEI Query lowering.
			opts := ctkWindowSelect()
			opts.CTE = ""
			opts.Table = "observations"
			if tc.preceding == 1 {
				if strings.Count(opts.Window, originalFrame) != 1 {
					t.Fatal("unexpected original Window frame")
				}
				opts.Window = strings.Replace(opts.Window, originalFrame, narrowedFrame, 1)
			}
			generated, err := NewDialect().Select(ctx, opts)
			if err != nil {
				t.Fatal(err)
			}
			expected := make(map[string]ctkWindowOracleRow, len(c.Outputs))
			for _, row := range c.Outputs {
				if _, exists := expected[row.ObsID]; exists {
					t.Fatalf("duplicate result oracle ID %s", row.ObsID)
				}
				expected[row.ObsID] = row
			}
			rows, err := db.QueryContext(ctx, generated.Statement, generated.Args...)
			if err != nil {
				t.Fatalf("run generated Window SQL with bound rows: %v\n%s", err, generated.Statement)
			}
			seen := make(map[string]bool, len(expected))
			for rows.Next() {
				var group, id string
				var sequence int64
				var value sql.NullInt64
				var moving, runningRows, runningRange float64
				if err := rows.Scan(&group, &id, &sequence, &value, &moving, &runningRows, &runningRange); err != nil {
					t.Fatal(err)
				}
				if seen[id] {
					t.Fatalf("duplicate result row %s", id)
				}
				want, ok := expected[id]
				input, inputOK := typedRows[id]
				if !ok || !inputOK {
					t.Fatalf("result from unpinned input ID %q", id)
				}
				seen[id] = true
				if group != input.Group || sequence != input.Seq || value != input.Value {
					t.Errorf("bound source row %s changed: group=%s, seq=%d, value=%+v; want %+v", id, group, sequence, value, input)
				}
				if moving != want.Moving || runningRows != want.CumulativeRows || runningRange != want.CumulativeRange {
					t.Errorf("generated Window result %s = (%g,%g,%g), independent Expected = (%g,%g,%g)",
						id, moving, runningRows, runningRange, want.Moving, want.CumulativeRows, want.CumulativeRange)
				}
			}
			if err := rows.Err(); err != nil {
				t.Fatal(err)
			}
			if err := rows.Close(); err != nil {
				t.Fatal(err)
			}
			if len(seen) != len(expected) {
				t.Fatalf("result closure has %d rows, independently expected %d", len(seen), len(expected))
			}

			if tc.preceding == 2 {
				// Change actual persisted input without changing generated
				// SQL or the independent Expected. A3's old average must
				// stop matching: (10 + 20 + 300) / 3 = 110, not 20.
				result, err := db.ExecContext(ctx, "UPDATE observations SET value = ? WHERE obs_id = ?", int64(300), "A3")
				if err != nil {
					t.Fatal(err)
				}
				if affected, err := result.RowsAffected(); err != nil || affected != 1 {
					t.Fatalf("A3 mutation affected %d rows: %v", affected, err)
				}
				rows, err := db.QueryContext(ctx, generated.Statement, generated.Args...)
				if err != nil {
					t.Fatal(err)
				}
				foundA3 := false
				for rows.Next() {
					var group, id string
					var sequence int64
					var value sql.NullInt64
					var moving, runningRows, runningRange float64
					if err := rows.Scan(&group, &id, &sequence, &value, &moving, &runningRows, &runningRange); err != nil {
						t.Fatal(err)
					}
					if id == "A3" {
						foundA3 = true
						if !value.Valid || value.Int64 != 300 || moving != 110 || moving == expected[id].Moving {
							t.Errorf("mutated A3 was not reflected in same generated SQL: value=%+v moving=%g", value, moving)
						}
					}
				}
				if err := rows.Err(); err != nil {
					t.Fatal(err)
				}
				if err := rows.Close(); err != nil {
					t.Fatal(err)
				}
				if !foundA3 {
					t.Fatal("mutated input A3 not found in generated SQL output")
				}
			}
		})
	}
}
