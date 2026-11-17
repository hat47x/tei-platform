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

	_ "github.com/jackc/pgx/v5/stdlib"
)

// The physical table, text input and composite key are independent of the
// order-line fixture. The expected row IDs and Rule answers are hand-written.
func TestDestinationIndependentShapeViaCLIFromPostgres(t *testing.T) {
	const databaseName = "tei_destination_order_e2e"
	dsn := os.Getenv("TEI_VIEW_PG_DSN")
	if dsn == "" {
		t.Skip("TEI_VIEW_PG_DSN is unset; real PostgreSQL destination not run")
	}
	if os.Getenv("TEI_VIEW_PG_CONFIRM_ISOLATED") != databaseName {
		t.Fatal("refusing to modify database without explicit disposable database confirmation")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal("cannot initialize disposable PostgreSQL database")
	}
	defer db.Close()
	var actualName string
	if err := db.QueryRow("select current_database()").Scan(&actualName); err != nil || actualName != databaseName {
		t.Fatalf("isolated database mismatch: database=%q err=%v", actualName, err)
	}
	if _, err := db.Exec("drop table if exists public.shipment_positions"); err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`create table public.shipment_positions (
		shipment_code varchar not null,
		segment varchar not null,
		token varchar not null,
		sequence_num bigint not null,
		sku varchar not null,
		units numeric(12,3) not null,
		unit_cost numeric(12,2) not null,
		primary key (segment, token)
	)`)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec("drop table if exists public.shipment_positions"); err != nil {
			t.Errorf("cannot clean up disposable shipment positions: %v", err)
		}
	})
	if _, err := db.Exec(`insert into public.shipment_positions
		(shipment_code, segment, token, sequence_num, sku, units, unit_cost)
		values ('0007', 'a/b', 'c', 1, 'X', 1, 10),
		       ('0007', 'a', 'b/c', 2, 'Y', 0, 20),
		       ('7', 'other', 'item', 1, 'Z', 2, 30)`); err != nil {
		t.Fatal(err)
	}

	root := t.TempDir()
	copyTreeForDestination(t, filepath.Join("..", "..", "reference", "workspace"), root)
	changed := 0
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || (!strings.HasSuffix(path, ".tei") && !strings.HasSuffix(path, ".sql")) {
			return nil
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		old := string(b)
		next := old
		for _, change := range [][2]string{
			{"sales.order.lines", "sales.order.shipment_positions"},
			{"sales.order_line", "sales.shipment_position"},
			{"relation lines to sales.shipment_position", "relation shipmentPositions to sales.shipment_position"},
			{"data OrderLine {", "data ShipmentPosition {"},
			{"type OrderLineId from", "type ShipmentPositionId from"},
			{"public.order_lines", "public.shipment_positions"},
		} {
			next = strings.ReplaceAll(next, change[0], change[1])
		}
		if next != old {
			changed++
			return os.WriteFile(path, []byte(next), 0o644)
		}
		return nil
	})
	if err != nil || changed < 7 {
		t.Fatalf("disposable Workspace rewrite failed: changed=%d err=%v", changed, err)
	}
	rewrite := func(relative, before, after string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(relative))
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Count(string(b), before) != 1 {
			t.Fatalf("expected one occurrence of %q in %s", before, relative)
		}
		if err := os.WriteFile(path, []byte(strings.Replace(string(b), before, after, 1)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	rewrite("semantic/sales/order.tei",
		"attribute orderLineId type sales.shipment_position_id",
		"attribute segment type std.text\n    attribute token type std.text")
	rewrite("databases/sales-core/persistence/order-line.tei",
		"column sales.shipment_position.order_line_id -> order_line_id storage int64 required",
		"column sales.shipment_position.segment -> segment storage varchar required\n    column sales.shipment_position.token -> token storage varchar required")
	rewrite("databases/sales-core/persistence/order-line.tei",
		"primary-key sales.shipment_position.order_line_id",
		"primary-key sales.shipment_position.segment\n    primary-key sales.shipment_position.token")
	for _, c := range []struct{ old, new string }{
		{"line_number -> line_number", "line_number -> sequence_num"},
		{"item_code -> item_code", "item_code -> sku"},
		{"quantity -> quantity", "quantity -> units"},
		{"unit_price -> unit_price", "unit_price -> unit_cost"},
	} {
		rewrite("databases/sales-core/persistence/order-line.tei", c.old, c.new)
	}
	rewrite("databases/sales-core/relation-storage/order-line.tei",
		"component sales.order.order_id -> order_id storage int64 required",
		"component sales.order.external_id -> shipment_code storage varchar required")
	rewrite("databases/sales-core/contracts/order-line.tei",
		"input order_id type sales.order_id", "input shipment_code type std.text")
	rewrite("applications/order-entry/realizations/header-detail/order-lines.tei",
		"input sales.order.order_id to order_id", "input sales.order.external_id to shipment_code")
	rewrite("databases/sales-core/query-realizations/order-line-find-by-order.tei",
		"input order_id -> order_id", "input shipment_code -> shipment_code")
	rewrite("databases/sales-core/query-realizations/order-line-find-by-order.tei",
		"column order_line_id -> sales.shipment_position.order_line_id",
		"column segment -> sales.shipment_position.segment\n        column token -> sales.shipment_position.token")
	for _, c := range []struct{ old, new string }{
		{"column line_number ->", "column sequence_num ->"},
		{"column item_code ->", "column sku ->"},
		{"column quantity ->", "column units ->"},
		{"column unit_price ->", "column unit_cost ->"},
	} {
		rewrite("databases/sales-core/query-realizations/order-line-find-by-order.tei", c.old, c.new)
	}
	query := filepath.Join(root, "databases", "sales-core", "query-realizations", "order-line-find-by-order.sql")
	if err := os.WriteFile(query, []byte(`SELECT segment, token, sequence_num, sku, units, unit_cost
FROM public.shipment_positions
WHERE shipment_code = :shipment_code
ORDER BY sequence_num, segment, token;
`), 0o644); err != nil {
		t.Fatal(err)
	}

	binary := buildCLIForLifecycle(t)
	validated := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root)
	if validated.code != 0 {
		t.Fatalf("independent Workspace does not validate: stdout=%s stderr=%s", validated.stdout, validated.stderr)
	}
	type report struct {
		OK     bool `json:"ok"`
		Result struct {
			Relation      string   `json:"relation"`
			Coverage      string   `json:"coverage"`
			KeyAttributes []string `json:"keyAttributes"`
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
	read := func(input string) report {
		t.Helper()
		got := runCLIProcessWithStreams(t, binary,
			"view", "-workspace", root, "-application", "order_entry", "-pattern", "order_entry.order_lines",
			"-environment", "development", "-input", "shipment_code="+input, "-format", "json")
		if got.code != 0 || strings.Contains(got.stdout+got.stderr, dsn) {
			t.Fatalf("independent read failed or exposed DSN: code=%d stdout=%s stderr=%s", got.code, got.stdout, got.stderr)
		}
		var result report
		if err := json.Unmarshal([]byte(got.stdout), &result); err != nil || !result.OK {
			t.Fatalf("independent read report: err=%v stdout=%s", err, got.stdout)
		}
		if result.Result.Relation != "sales.order.shipment_positions" || result.Result.Coverage != "partial" ||
			!reflect.DeepEqual(result.Result.KeyAttributes, []string{"sales.shipment_position.segment", "sales.shipment_position.token"}) {
			t.Fatalf("independent relation, key or coverage lost: %+v", result.Result)
		}
		return result
	}
	zeroPrefixed := read("0007")
	first, second := `["a/b","c"]`, `["a","b/c"]`
	if len(zeroPrefixed.Result.Rows) != 2 || zeroPrefixed.Result.Rows[0].Key != first || zeroPrefixed.Result.Rows[1].Key != second {
		t.Fatalf("composite row identities were collapsed or text input lost its zeros: %+v", zeroPrefixed.Result.Rows)
	}
	want := map[string]string{first: "satisfied", second: "not-satisfied"}
	for _, row := range zeroPrefixed.Result.Rows {
		if row.Fields["sales.shipment_position.quantity"].State != "present" ||
			zeroPrefixed.Result.PerRow[row.Key]["sales.shipment_position.quantity_positive"] != want[row.Key] ||
			zeroPrefixed.Result.PerRow[row.Key]["sales.shipment_position.line_number_positive"] != "satisfied" {
			t.Fatalf("independent row %s did not match hand-written expectation: %+v", row.Key, zeroPrefixed.Result.PerRow[row.Key])
		}
	}
	const unique = "sales.order.shipment_positions.line_number_unique"
	if zeroPrefixed.Result.PerRelation[unique] != "undecided" || zeroPrefixed.Result.PerRelationUndecided[unique] == "" {
		t.Fatal("the independent incomplete Relation was accepted as complete")
	}
	unprefixed := read("7")
	if len(unprefixed.Result.Rows) != 1 || unprefixed.Result.Rows[0].Key != `["other","item"]` {
		t.Fatalf("a text input was coerced or the wrong parent was fetched: %+v", unprefixed.Result.Rows)
	}
}
