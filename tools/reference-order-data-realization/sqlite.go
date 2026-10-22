package orderdatarealization

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"tei/internal/database/persistence"
	"tei/internal/driver/queryresult"
	"tei/internal/ir"
)

// PythonEnv overrides the Python interpreter that runs SQLite. A missing
// interpreter is a failure, not a skip.
const PythonEnv = "TEI_ODR_PYTHON"

const sqliteTimeLayout = "2006-01-02T15:04:05.000000Z"

// sqliteRunner executes the request against a SQLite file through Python's
// standard sqlite3 module. All SQL text comes from the Go side.
const sqliteRunner = `import json, sqlite3, sys
req = json.load(sys.stdin)
db = sqlite3.connect(req["path"])
out = {"sqliteVersion": sqlite3.sqlite_version, "results": {}, "probes": {}}
for s in req["setup"]:
    db.execute(s)
for ins in req["inserts"]:
    db.executemany(ins["sql"], ins["rows"])
db.commit()
for name, sql in req["queries"].items():
    cur = db.execute(sql)
    out["results"][name] = {"columns": [d[0] for d in cur.description], "rows": cur.fetchall()}
for name, p in req["probes"].items():
    try:
        db.execute(p["sql"], p["args"])
        out["probes"][name] = "accepted"
    except sqlite3.IntegrityError as e:
        out["probes"][name] = "integrity-error: " + str(e)
    db.rollback()
db.close()
json.dump(out, sys.stdout, ensure_ascii=False)
`

type sqliteRequest struct {
	Path    string                 `json:"path"`
	Setup   []string               `json:"setup"`
	Inserts []sqliteInsert         `json:"inserts"`
	Queries map[string]string      `json:"queries"`
	Probes  map[string]sqliteProbe `json:"probes"`
}

type sqliteInsert struct {
	SQL  string  `json:"sql"`
	Rows [][]any `json:"rows"`
}

type sqliteProbe struct {
	SQL  string `json:"sql"`
	Args []any  `json:"args"`
}

type sqliteResponse struct {
	Version string `json:"sqliteVersion"`
	Results map[string]struct {
		Columns []string `json:"columns"`
		Rows    [][]any  `json:"rows"`
	} `json:"results"`
	Probes map[string]string `json:"probes"`
}

func pythonCommand() (string, error) {
	if p := os.Getenv(PythonEnv); p != "" {
		return p, nil
	}
	p, err := exec.LookPath("python3")
	if err != nil {
		return "", fmt.Errorf("step 10 needs python3 with the sqlite3 module (set %s): %w", PythonEnv, err)
	}
	return p, nil
}

func runSQLite(ctx context.Context, req sqliteRequest) (sqliteResponse, error) {
	py, err := pythonCommand()
	if err != nil {
		return sqliteResponse{}, err
	}
	raw, err := json.Marshal(req)
	if err != nil {
		return sqliteResponse{}, err
	}
	cmd := exec.CommandContext(ctx, py, "-I", "-c", sqliteRunner)
	cmd.Stdin = bytes.NewReader(raw)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return sqliteResponse{}, fmt.Errorf("sqlite runner: %v: %s", err, stderr.String())
	}
	var resp sqliteResponse
	dec := json.NewDecoder(&stdout)
	dec.UseNumber()
	if err := dec.Decode(&resp); err != nil {
		return sqliteResponse{}, err
	}
	return resp, nil
}

// sqliteTable is one table realized in SQLite from a storage mapping.
type sqliteTable struct {
	Name    string
	Columns []string
	Storage []persistence.StorageType
	DDL     []string
}

// sqliteSchema realizes the storage mappings of one database with the
// declared storage-type mapping of the dialect. SQLite has no schema level;
// the table name drops the schema.
func sqliteSchema(world ir.World, decl SQLiteDialectDecl) (map[string]sqliteTable, error) {
	out := map[string]sqliteTable{}
	for _, m := range world.Persistence {
		if string(m.Database) != databaseID {
			continue
		}
		name := m.Table
		if i := strings.LastIndex(name, "."); i >= 0 {
			name = name[i+1:]
		}
		t := sqliteTable{Name: name}
		var defs []string
		colOf := map[string]string{}
		add := func(col string, st persistence.StorageType, req persistence.Nullability) error {
			typ, ok := decl.Storage[string(st)]
			if !ok {
				return fmt.Errorf("storage %s has no declared SQLite type", st)
			}
			d := quoteIdent(col) + " " + typ
			if req == persistence.Required {
				d += " NOT NULL"
			}
			defs = append(defs, d)
			t.Columns = append(t.Columns, col)
			t.Storage = append(t.Storage, st)
			return nil
		}
		for _, c := range m.Columns {
			if err := add(c.Column, c.Storage, c.Nullability); err != nil {
				return nil, err
			}
			colOf[string(c.Attribute)] = c.Column
		}
		for _, rs := range m.RelationStorage {
			for _, comp := range rs.Components {
				if err := add(comp.Column, comp.Storage, comp.Nullability); err != nil {
					return nil, err
				}
			}
		}
		cols := func(attrs []string) string {
			var q []string
			for _, a := range attrs {
				q = append(q, quoteIdent(colOf[a]))
			}
			return strings.Join(q, ", ")
		}
		var pk []string
		for _, a := range m.PrimaryKey {
			pk = append(pk, string(a))
		}
		if len(pk) > 0 {
			defs = append(defs, "PRIMARY KEY ("+cols(pk)+")")
		}
		for _, u := range m.UniqueConstraints {
			var attrs []string
			for _, a := range u.Attributes {
				attrs = append(attrs, string(a))
			}
			defs = append(defs, "CONSTRAINT "+quoteIdent(u.Name)+" UNIQUE ("+cols(attrs)+")")
		}
		t.DDL = append(t.DDL, "CREATE TABLE "+quoteIdent(name)+" (\n  "+strings.Join(defs, ",\n  ")+"\n)")
		for _, ix := range m.Indexes {
			if len(ix.Keys) > 0 || len(ix.IncludedAttributes) > 0 || ix.Predicate != nil || ix.Method != "" || ix.UniqueNullTreatment != "" || len(ix.Attributes) == 0 {
				return nil, fmt.Errorf("index %s: only plain indexes are realized in SQLite by the sample", ix.Name)
			}
			var attrs []string
			for _, a := range ix.Attributes {
				attrs = append(attrs, string(a))
			}
			kw := "INDEX"
			if ix.Unique {
				kw = "UNIQUE INDEX"
			}
			t.DDL = append(t.DDL, "CREATE "+kw+" "+quoteIdent(ix.Name)+" ON "+quoteIdent(name)+" ("+cols(attrs)+")")
		}
		out[name] = t
	}
	return out, nil
}

// pgValueToSQLite encodes a value read from PostgreSQL for the declared
// physical representation in SQLite.
func pgValueToSQLite(v any, st persistence.StorageType) (any, error) {
	if v == nil {
		return nil, nil
	}
	switch st {
	case persistence.StorageDateTimeTZ:
		t, ok := v.(time.Time)
		if !ok {
			return nil, fmt.Errorf("datetime-tz value of type %T", v)
		}
		return t.UTC().Format(sqliteTimeLayout), nil
	case persistence.StorageDecimal, persistence.StorageVarchar, persistence.StorageText:
		switch x := v.(type) {
		case string:
			return x, nil
		case []byte:
			return string(x), nil
		default:
			return nil, fmt.Errorf("%s value of type %T (not exact text)", st, v)
		}
	case persistence.StorageInt64:
		switch x := v.(type) {
		case int64:
			return x, nil
		case int32:
			return int64(x), nil
		}
		return nil, fmt.Errorf("int64 value of type %T", v)
	}
	return nil, fmt.Errorf("storage %s not handled", st)
}

// semanticValue renders the semantic value of a PostgreSQL value.
func semanticValue(v any, st persistence.StorageType) string {
	if v == nil {
		return "<null>"
	}
	switch st {
	case persistence.StorageDateTimeTZ:
		if t, ok := v.(time.Time); ok {
			return "t:" + t.UTC().Format(sqliteTimeLayout)
		}
	case persistence.StorageInt64:
		return fmt.Sprintf("i:%v", v)
	}
	switch x := v.(type) {
	case string:
		return "s:" + x
	case []byte:
		return "s:" + string(x)
	}
	return fmt.Sprintf("?%T:%v", v, v)
}

// sqliteSemanticValue renders the semantic value read back from SQLite; a
// value whose JSON type changed (text became a number) shows the change.
func sqliteSemanticValue(v any, st persistence.StorageType) string {
	if v == nil {
		return "<null>"
	}
	switch x := v.(type) {
	case string:
		switch st {
		case persistence.StorageDateTimeTZ:
			t, err := time.Parse(sqliteTimeLayout, x)
			if err != nil {
				return "bad-time:" + x
			}
			return "t:" + t.UTC().Format(sqliteTimeLayout)
		case persistence.StorageInt64:
			return "s-for-int:" + x
		}
		return "s:" + x
	case json.Number:
		if st == persistence.StorageInt64 {
			return "i:" + x.String()
		}
		return "n:" + x.String()
	}
	return fmt.Sprintf("?%T:%v", v, v)
}

// ValueDiff is one value that did not survive the round trip.
type ValueDiff struct {
	Table    string `json:"table"`
	Key      string `json:"key"`
	Column   string `json:"column"`
	Postgres string `json:"postgres"`
	SQLite   string `json:"sqlite"`
}

// RoundTrip is the result of one dialect.
type RoundTrip struct {
	Dialect          string            `json:"dialect"`
	SQLiteVersion    string            `json:"sqliteVersion"`
	DDL              []string          `json:"ddl"`
	RowsWritten      map[string]int    `json:"rowsWritten"`
	ValuesCompared   int               `json:"valuesCompared"`
	Diffs            []ValueDiff       `json:"diffs"`
	Undeclared       int               `json:"undeclaredDiffs"`
	ListOrderSame    bool              `json:"listOrderSameAsPostgres"`
	UniqueProbes     map[string]string `json:"uniqueProbes"`
	FileRemoved      bool              `json:"sqliteFileRemoved"`
}

// pgTables reads the tables to round trip from PostgreSQL as raw rows keyed by
// column (the realization to compare is physical: every mapped column).
type pgTable struct {
	columns []string
	storage []persistence.StorageType
	rows    [][]any
}

// roundTrip writes the PostgreSQL rows into a SQLite file realized with the
// given dialect, reads them back and compares every value.
func roundTrip(ctx context.Context, dir, label string, world ir.World, decl SQLiteDialectDecl, pg map[string]pgTable, listOrder []string) (RoundTrip, error) {
	rt := RoundTrip{Dialect: label, RowsWritten: map[string]int{}}
	schema, err := sqliteSchema(world, decl)
	if err != nil {
		return rt, err
	}
	path := filepath.Join(dir, "order-data-realization-"+label+".sqlite")
	req := sqliteRequest{Path: path, Queries: map[string]string{}, Probes: map[string]sqliteProbe{}}
	names := make([]string, 0, len(schema))
	for n := range schema {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		t := schema[n]
		req.Setup = append(req.Setup, t.DDL...)
		rt.DDL = append(rt.DDL, t.DDL...)
		src, ok := pg[n]
		if !ok {
			return rt, fmt.Errorf("no PostgreSQL rows for table %s", n)
		}
		if strings.Join(src.columns, ",") != strings.Join(t.Columns, ",") {
			return rt, fmt.Errorf("table %s: PostgreSQL columns %v differ from the realized columns %v", n, src.columns, t.Columns)
		}
		marks := strings.TrimSuffix(strings.Repeat("?, ", len(t.Columns)), ", ")
		var qcols []string
		for _, c := range t.Columns {
			qcols = append(qcols, quoteIdent(c))
		}
		ins := sqliteInsert{SQL: fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", quoteIdent(n), strings.Join(qcols, ", "), marks)}
		for _, row := range src.rows {
			enc := make([]any, len(row))
			for i, v := range row {
				if enc[i], err = pgValueToSQLite(v, t.Storage[i]); err != nil {
					return rt, fmt.Errorf("%s.%s: %w", n, t.Columns[i], err)
				}
			}
			ins.Rows = append(ins.Rows, enc)
		}
		rt.RowsWritten[n] = len(ins.Rows)
		req.Inserts = append(req.Inserts, ins)
		req.Queries[n] = fmt.Sprintf("SELECT %s FROM %s ORDER BY 1", strings.Join(qcols, ", "), quoteIdent(n))
	}
	req.Queries["list"] = `SELECT "order_id" FROM "orders" WHERE "ordered_at" >= '2000-01-01T00:00:00.000000Z' ORDER BY "ordered_at" DESC, "order_id"`
	req.Probes["duplicate-external-id"] = sqliteProbe{SQL: `INSERT INTO "orders" ("order_id","customer_id","ordered_at","status","total_amount","external_id","delivery_note") SELECT 99999, 1, '2026-01-01T00:00:00.000000Z', 'probe', '0', "external_id", NULL FROM "orders" WHERE "external_id" IS NOT NULL LIMIT 1`, Args: []any{}}
	req.Probes["second-null-external-id"] = sqliteProbe{SQL: `INSERT INTO "orders" ("order_id","customer_id","ordered_at","status","total_amount","external_id","delivery_note") VALUES (99998, 1, '2026-01-01T00:00:00.000000Z', 'probe', '0', NULL, NULL)`, Args: []any{}}
	resp, err := runSQLite(ctx, req)
	os.Remove(path)
	_, statErr := os.Stat(path)
	rt.FileRemoved = os.IsNotExist(statErr)
	if err != nil {
		return rt, err
	}
	rt.SQLiteVersion = resp.Version
	rt.UniqueProbes = resp.Probes
	for _, n := range names {
		t, src := schema[n], pg[n]
		got := resp.Results[n]
		if len(got.Rows) != len(src.rows) {
			return rt, fmt.Errorf("table %s: %d rows read back, %d written", n, len(got.Rows), len(src.rows))
		}
		sort.Slice(src.rows, func(i, j int) bool { return fmt.Sprint(src.rows[i][0]) < fmt.Sprint(src.rows[j][0]) })
		byKey := map[string][]any{}
		for _, r := range got.Rows {
			byKey[fmt.Sprint(r[0])] = r
		}
		for _, row := range src.rows {
			key := fmt.Sprint(row[0])
			back, ok := byKey[key]
			if !ok {
				return rt, fmt.Errorf("table %s: row %s not read back", n, key)
			}
			for i, v := range row {
				rt.ValuesCompared++
				p, s := semanticValue(v, t.Storage[i]), sqliteSemanticValue(back[i], t.Storage[i])
				if p != s {
					rt.Diffs = append(rt.Diffs, ValueDiff{Table: n, Key: key, Column: t.Columns[i], Postgres: p, SQLite: s})
					rt.Undeclared++
				}
			}
		}
	}
	var sqliteOrder []string
	for _, r := range resp.Results["list"].Rows {
		sqliteOrder = append(sqliteOrder, fmt.Sprint(r[0]))
	}
	rt.ListOrderSame = strings.Join(sqliteOrder, ",") == strings.Join(listOrder, ",")
	return rt, nil
}

// listOrderOf orders the PostgreSQL list by ordered_at descending and
// order_id, the order the SQLite list query uses; the PostgreSQL query itself
// leaves ties unordered.
func listOrderOf(list queryresult.MappedResult) []string {
	type item struct {
		at time.Time
		id int64
	}
	var items []item
	for i := range list.Rows {
		at, _ := cell(list, i, "sales.order.ordered_at").(time.Time)
		id, _ := cell(list, i, "sales.order.order_id").(int64)
		items = append(items, item{at, id})
	}
	sort.Slice(items, func(i, j int) bool {
		if !items[i].at.Equal(items[j].at) {
			return items[i].at.After(items[j].at)
		}
		return items[i].id < items[j].id
	})
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = fmt.Sprint(it.id)
	}
	return out
}
