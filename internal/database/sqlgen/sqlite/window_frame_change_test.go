package sqlite

import (
	"context"
	"database/sql"
	"strings"
	"testing"
)

// CTK-14 provides an independently pinned SQL-adapter oracle for a future
// accepted *.tei.md -> Query -> Target path. It does not exercise the TEI
// Markdown reader, meaning acceptance, CLI entry, or plugin composition.
func TestCTKWindowFrameChangeAltersGeneratedSQLiteResult(t *testing.T) {
	const originalFrame = "ROWS BETWEEN 2 PRECEDING AND CURRENT ROW"
	const narrowedFrame = "ROWS BETWEEN 1 PRECEDING AND CURRENT ROW"

	cases := []struct {
		name    string
		narrow  bool
		moving  map[string]float64
	}{
		{
			name:   "current and two preceding rows",
			moving: map[string]float64{"A1": 10, "A2": 15, "A3": 20, "A4": 30, "B1": 5, "B2": 5, "B3": 10},
		},
		{
			name:   "current and one preceding row",
			narrow: true,
			moving: map[string]float64{"A1": 10, "A2": 15, "A3": 25, "A4": 35, "B1": 5, "B2": 5, "B3": 15},
		},
	}
	// These two independent cumulative windows must not change with the
	// frame used only by the moving-average window.
	cumulativeRows := map[string]float64{
		"A1": 10, "A2": 15, "A3": 20, "A4": 25,
		"B1": 5, "B2": 5, "B3": 10,
	}
	cumulativeRange := map[string]float64{
		"A1": 10, "A2": 15, "A3": 20, "A4": 25,
		"B1": 5, "B2": 10, "B3": 10,
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := openCTKSQLite(t)
			ctx := context.Background()
			opts := ctkWindowSelect()
			if strings.Count(opts.Window, originalFrame) != 1 {
				t.Fatalf("expected exactly one original frame, got %q", opts.Window)
			}
			if tc.narrow {
				opts.Window = strings.Replace(opts.Window, originalFrame, narrowedFrame, 1)
			}
			generated, err := NewDialect().Select(ctx, opts)
			if err != nil {
				t.Fatalf("generate SQLite statement: %v", err)
			}
			if len(generated.Args) != 0 {
				t.Fatalf("unexpected SQL arguments: %#v", generated.Args)
			}
			if tc.narrow && !strings.Contains(generated.Statement, narrowedFrame) {
				t.Fatalf("generated SQL lacks changed frame: %q", generated.Statement)
			}
			rows, err := db.QueryContext(ctx, generated.Statement, generated.Args...)
			if err != nil {
				t.Fatalf("execute generated SQL: %v\n%s", err, generated.Statement)
			}
			defer rows.Close()

			seen := make(map[string]bool, 7)
			for rows.Next() {
				var group, id string
				var seq int64
				var value sql.NullInt64
				var moving, runningRows, runningRange float64
				if err := rows.Scan(&group, &id, &seq, &value, &moving, &runningRows, &runningRange); err != nil {
					t.Fatalf("scan observation: %v", err)
				}
				wantMoving, known := tc.moving[id]
				if !known || seen[id] {
					t.Fatalf("unknown or duplicated observation %q", id)
				}
				seen[id] = true
				if group == "" || seq == 0 || (id == "B2") != !value.Valid {
					t.Errorf("observation %s changed identity or NULL semantics: group=%q seq=%d value=%+v", id, group, seq, value)
				}
				if moving != wantMoving {
					t.Errorf("%s moving = %v, want %v", id, moving, wantMoving)
				}
				if runningRows != cumulativeRows[id] || runningRange != cumulativeRange[id] {
					t.Errorf("%s cumulative averages changed: got (%v,%v), want (%v,%v)",
						id, runningRows, runningRange, cumulativeRows[id], cumulativeRange[id])
				}
			}
			if err := rows.Err(); err != nil {
				t.Fatalf("read query result: %v", err)
			}
			if len(seen) != 7 {
				t.Errorf("observed %d rows, want 7", len(seen))
			}
		})
	}
}
