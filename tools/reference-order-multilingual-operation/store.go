package ordermultilingualoperation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Stmt is one SQL statement with `?` placeholders.
type Stmt struct {
	SQL  string `json:"sql"`
	Args []any  `json:"args"`
}

// Store is one relational product. Every call opens its own connection, so
// a read after a write is a re-read from another connection.
type Store interface {
	Name() string
	Setup(ctx context.Context, schema string) error
	Exec(ctx context.Context, stmts ...Stmt) error
	Query(ctx context.Context, sql string, args ...any) ([][]any, error)
	// QuerySQL returns the dialect specific text of a declared order.
	Order(key OrderKey) string
	RowGreater(key OrderKey) string
	Info(ctx context.Context) map[string]string
}

// OrderKey is the sample's declared list order: columns compared in one
// declared way. "binary-utf8" is code point order of the stored UTF-8;
// "icu-und" is PostgreSQL's ICU root collation (language dependent).
type OrderKey struct {
	Columns    []string `json:"columns"`
	Comparison string   `json:"comparison"`
}

// PGStore is a PostgreSQL database reached through pgx.
type PGStore struct{ DSN string }

func (PGStore) Name() string { return "postgresql" }

func pgSQL(sql string) string {
	var b strings.Builder
	n := 0
	for _, r := range sql {
		if r == '?' {
			n++
			b.WriteString("$" + strconv.Itoa(n))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func (s PGStore) connect(ctx context.Context) (*pgx.Conn, error) { return pgx.Connect(ctx, s.DSN) }

// Setup applies schema.sql statement by statement, as the asset's own
// PostgreSQL test does.
func (s PGStore) Setup(ctx context.Context, schema string) error {
	conn, err := s.connect(ctx)
	if err != nil {
		return err
	}
	defer conn.Close(context.Background())
	for _, stmt := range strings.Split(schema, ";") {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		if _, err := conn.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("schema: %w", err)
		}
	}
	return nil
}

// Exec runs the statements in one transaction of a new connection.
func (s PGStore) Exec(ctx context.Context, stmts ...Stmt) error {
	conn, err := s.connect(ctx)
	if err != nil {
		return err
	}
	defer conn.Close(context.Background())
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	for _, st := range stmts {
		if _, err := tx.Exec(ctx, pgSQL(st.SQL), st.Args...); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("%s: %w", st.SQL, err)
		}
	}
	return tx.Commit(ctx)
}

// Query reads on a new connection.
func (s PGStore) Query(ctx context.Context, sql string, args ...any) ([][]any, error) {
	conn, err := s.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close(context.Background())
	rows, err := conn.Query(ctx, pgSQL(sql), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out [][]any
	for rows.Next() {
		v, err := rows.Values()
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func pgCollation(c string) string {
	switch c {
	case "binary-utf8":
		return ` COLLATE "C"`
	case "icu-und":
		return ` COLLATE "und-x-icu"`
	}
	return "" // database default
}

func (PGStore) Order(k OrderKey) string {
	var parts []string
	for i, c := range k.Columns {
		coll := ` COLLATE "C"`
		if i == 0 {
			coll = pgCollation(k.Comparison)
		}
		parts = append(parts, c+coll)
	}
	return strings.Join(parts, ", ")
}

func (s PGStore) RowGreater(k OrderKey) string {
	marks := make([]string, len(k.Columns))
	for i := range marks {
		marks[i] = "?"
	}
	return "(" + s.Order(k) + ") > (" + strings.Join(marks, ", ") + ")"
}

func (s PGStore) Info(ctx context.Context) map[string]string {
	out := map[string]string{}
	for name, q := range map[string]string{
		"serverVersion":   "SHOW server_version",
		"serverEncoding":  "SHOW server_encoding",
		"databaseCollate": "SELECT datcollate FROM pg_database WHERE datname = current_database()",
		"icuUndVersion":   `SELECT coalesce(collversion, '') FROM pg_collation WHERE collname = 'und-x-icu'`,
	} {
		rows, err := s.Query(ctx, q)
		if err != nil || len(rows) == 0 {
			out[name] = "unavailable"
			continue
		}
		out[name] = fmt.Sprint(rows[0][0])
	}
	return out
}

// sqliteRunner executes one request on a SQLite file through Python's
// standard sqlite3 module: one process, one connection. All SQL comes from
// the Go side.
const sqliteRunner = `import json, sqlite3, sys
req = json.load(sys.stdin)
db = sqlite3.connect(req["path"])
db.execute("PRAGMA foreign_keys = ON")
out = {"sqliteVersion": sqlite3.sqlite_version, "rows": []}
try:
    if req.get("script"):
        db.executescript(req["script"])
    for st in req.get("stmts") or []:
        db.execute(st["sql"], st["args"] or [])
    db.commit()
    if req.get("query"):
        cur = db.execute(req["query"]["sql"], req["query"]["args"] or [])
        out["rows"] = [list(r) for r in cur.fetchall()]
except sqlite3.Error as e:
    db.rollback()
    out["error"] = type(e).__name__ + ": " + str(e)
db.close()
json.dump(out, sys.stdout, ensure_ascii=False)
`

// SQLiteStore is a SQLite file reached through a separate Python process
// for every call.
type SQLiteStore struct {
	Path   string
	Python string
}

func (SQLiteStore) Name() string { return "sqlite" }

type sqliteRequest struct {
	Path   string `json:"path"`
	Script string `json:"script,omitempty"`
	Stmts  []Stmt `json:"stmts,omitempty"`
	Query  *Stmt  `json:"query,omitempty"`
}

func (s SQLiteStore) run(ctx context.Context, req sqliteRequest) ([][]any, string, error) {
	req.Path = s.Path
	in, err := json.Marshal(req)
	if err != nil {
		return nil, "", err
	}
	cmd := exec.CommandContext(ctx, s.Python, "-I", "-c", sqliteRunner)
	cmd.Stdin = bytes.NewReader(in)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return nil, "", fmt.Errorf("python sqlite3: %v: %s", err, stderr.String())
	}
	var out struct {
		Version string  `json:"sqliteVersion"`
		Rows    [][]any `json:"rows"`
		Error   string  `json:"error"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return nil, "", err
	}
	if out.Error != "" {
		return nil, out.Version, fmt.Errorf("sqlite: %s", out.Error)
	}
	return out.Rows, out.Version, nil
}

func (s SQLiteStore) Setup(ctx context.Context, schema string) error {
	_, _, err := s.run(ctx, sqliteRequest{Script: schema})
	return err
}

func (s SQLiteStore) Exec(ctx context.Context, stmts ...Stmt) error {
	_, _, err := s.run(ctx, sqliteRequest{Stmts: stmts})
	return err
}

func (s SQLiteStore) Query(ctx context.Context, sql string, args ...any) ([][]any, error) {
	rows, _, err := s.run(ctx, sqliteRequest{Query: &Stmt{SQL: sql, Args: args}})
	return rows, err
}

func (SQLiteStore) Order(k OrderKey) string {
	var parts []string
	for _, c := range k.Columns {
		// BINARY is SQLite's default: memcmp of the UTF-8 bytes.
		parts = append(parts, c+" COLLATE BINARY")
	}
	return strings.Join(parts, ", ")
}

func (s SQLiteStore) RowGreater(k OrderKey) string {
	marks := make([]string, len(k.Columns))
	for i := range marks {
		marks[i] = "?"
	}
	return "(" + s.Order(k) + ") > (" + strings.Join(marks, ", ") + ")"
}

func (s SQLiteStore) Info(ctx context.Context) map[string]string {
	_, v, err := s.run(ctx, sqliteRequest{Query: &Stmt{SQL: "SELECT 1"}})
	if err != nil {
		return map[string]string{"sqliteVersion": "unavailable"}
	}
	return map[string]string{"sqliteVersion": v, "comparison": "BINARY (memcmp of UTF-8)"}
}
