package main

import (
	"database/sql"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// checkDestinationAlternateModel exercises another model through the same
// product "tei view" command. Its source changes and expected answers live in
// a separately authored fixture, rather than being copied from CLI output.
func checkDestinationAlternateModel(t *testing.T, db *sql.DB, binary, dsn string) {
	t.Helper()
	fixture := filepath.Join("..", "..", "testdata", "destination-order-change", "alternate-model")
	var model struct {
		Replacements [][2]string `json:"replacements"`
		Patches      []struct {
			Path  string `json:"path"`
			Old   string `json:"old"`
			Value string `json:"value"`
		} `json:"patches"`
	}
	rawModel, err := os.ReadFile(filepath.Join(fixture, "model.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(rawModel, &model); err != nil {
		t.Fatal(err)
	}
	var expected struct {
		Relation      string   `json:"relation"`
		KeyAttributes []string `json:"keyAttributes"`
		Input         string   `json:"input"`
		UnitRule      string   `json:"unitRule"`
		SlotRule      string   `json:"slotRule"`
		UniqueRule    string   `json:"uniqueRule"`
		Checks        map[string]struct {
			Keys   []string `json:"keys"`
			Units  []string `json:"units"`
			Unique string   `json:"unique"`
		} `json:"checks"`
	}
	rawExpected, err := os.ReadFile(filepath.Join(fixture, "expected.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(rawExpected, &expected); err != nil {
		t.Fatal(err)
	}
	if expected.Input != "batch_key" || len(expected.Checks) != 3 {
		t.Fatal("alternate model fixture is incomplete")
	}
	if _, err := db.Exec("DROP TABLE IF EXISTS public.inventory_entries"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec("DROP TABLE IF EXISTS public.inventory_entries"); err != nil {
			t.Errorf("cannot clean up alternate database table: %v", err)
		}
	})
	sqlBytes, err := os.ReadFile(filepath.Join(fixture, "seed.sql"))
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range strings.Split(string(sqlBytes), ";") {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("alternate PostgreSQL fixture did not apply: %v", err)
		}
	}

	root := t.TempDir()
	copyTreeForDestination(t, filepath.Join("..", "..", "reference", "workspace"), root)
	seen := make([]int, len(model.Replacements))
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		if !strings.HasSuffix(path, ".tei") && !strings.HasSuffix(path, ".sql") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		updated := string(raw)
		for i, pair := range model.Replacements {
			count := strings.Count(updated, pair[0])
			seen[i] += count
			updated = strings.ReplaceAll(updated, pair[0], pair[1])
		}
		if updated != string(raw) {
			return os.WriteFile(path, []byte(updated), 0o644)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for i, count := range seen {
		if count == 0 {
			t.Fatalf("alternate semantic model did not apply replacement %q", model.Replacements[i][0])
		}
	}
	for _, patch := range model.Patches {
		target := filepath.Join(root, filepath.FromSlash(patch.Path))
		raw, err := os.ReadFile(target)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Count(string(raw), patch.Old) != 1 {
			t.Fatalf("model patch %s is stale: expected one %q", patch.Path, patch.Old)
		}
		updated := strings.Replace(string(raw), patch.Old, patch.Value, 1)
		if err := os.WriteFile(target, []byte(updated), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	validated := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root)
	if validated.code != 0 {
		t.Fatalf("alternate model rejected: stdout=%s stderr=%s", validated.stdout, validated.stderr)
	}
	for _, batch := range []string{"1001", "1002", "1003"} {
		t.Run(batch, func(t *testing.T) {
			response := runCLIProcessWithStreams(t, binary,
				"view", "-workspace", root, "-application", "order_entry",
				"-pattern", "order_entry.order_lines", "-environment", "development",
				"-input", expected.Input+"="+batch, "-format", "json",
			)
			if response.code != 0 || strings.Contains(response.stdout+response.stderr, dsn) {
				t.Fatalf("alternate model failed or disclosed connection details: code=%d stdout=%s stderr=%s",
					response.code, response.stdout, response.stderr)
			}
			var result struct {
				OK     bool `json:"ok"`
				Result struct {
					Relation      string   `json:"relation"`
					KeyAttributes []string `json:"keyAttributes"`
					Coverage      string   `json:"coverage"`
					Rows          []struct {
						Key    string `json:"key"`
						Fields map[string]struct {
							State string `json:"state"`
						} `json:"fields"`
					} `json:"rows"`
					PerRow               map[string]map[string]string `json:"perRow"`
					PerRelation          map[string]string            `json:"perRelation"`
					PerRelationUndecided map[string]string            `json:"perRelationUndecided"`
				} `json:"result"`
			}
			if err := json.Unmarshal([]byte(response.stdout), &result); err != nil || !result.OK {
				t.Fatalf("invalid view response: err=%v output=%s", err, response.stdout)
			}
			if result.Result.Relation != expected.Relation ||
				!reflect.DeepEqual(result.Result.KeyAttributes, expected.KeyAttributes) ||
				result.Result.Coverage != "partial" {
				t.Fatalf("alternate model lost semantic identity or overclaimed coverage: %+v", result.Result)
			}
			want := expected.Checks[batch]
			if len(result.Result.Rows) != len(want.Keys) || len(want.Units) != len(want.Keys) {
				t.Fatalf("batch %s has %d rows, want %d", batch, len(result.Result.Rows), len(want.Keys))
			}
			keys := make([]string, 0, len(result.Result.Rows))
			for i, row := range result.Result.Rows {
				keys = append(keys, row.Key)
				if row.Fields["sales.inventory_entry.units"].State != "present" ||
					result.Result.PerRow[row.Key][expected.UnitRule] != want.Units[i] ||
					result.Result.PerRow[row.Key][expected.SlotRule] != "satisfied" {
					t.Fatalf("row %s has wrong field/read decision: %+v", row.Key, result.Result)
				}
			}
			if !reflect.DeepEqual(keys, want.Keys) ||
				len(result.Result.PerRow) != len(want.Keys) ||
				result.Result.PerRelation[expected.UniqueRule] != want.Unique {
				t.Fatalf("batch %s: keys=%v decisions=%v unique=%s; want %+v",
					batch, keys, result.Result.PerRow, result.Result.PerRelation[expected.UniqueRule], want)
			}
			if want.Unique == "undecided" && result.Result.PerRelationUndecided[expected.UniqueRule] == "" {
				t.Fatal("Query result was treated as complete without a completeness declaration")
			}
		})
	}
	old := runCLIProcessWithStreams(t, binary,
		"view", "-workspace", root, "-application", "order_entry",
		"-pattern", "order_entry.order_lines", "-environment", "development",
		"-input", "order_id=1001", "-format", "json",
	)
	if old.code == 0 {
		t.Fatal("the old Contract Query input name was silently accepted")
	}
}
