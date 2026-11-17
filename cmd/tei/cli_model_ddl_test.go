package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	"tei/internal/core"
	"tei/internal/database/persistence"
	"tei/internal/database/tabledef"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/workspace"
)

// writeModelDDLWorldFile builds one mapping, serializes it as a normalized M0
// World and writes it where `-world` reads it. It is the -world half of the
// command's contract: a `tei world` output feeds the same projection.
func writeModelDDLWorldFile(t *testing.T) string {
	t.Helper()
	world := ir.World{
		Semantics: []semantic.Namespace{{
			ID:   "sales",
			Data: []semantic.Data{{ID: "sales.order"}},
		}},
		Persistence: []persistence.Mapping{{
			ID:           "sales_core.order",
			Database:     "sales_core",
			SemanticData: "sales.order",
			Table:        "public.orders",
			PrimaryKey:   []core.CanonicalID{"sales.order.order_id"},
			Columns: []persistence.ColumnMapping{
				{Attribute: "sales.order.order_id", Column: "order_id", Storage: persistence.StorageInt64, Nullability: persistence.Required},
				{Attribute: "sales.order.status", Column: "status", Storage: persistence.StorageVarchar, Nullability: persistence.Required},
			},
		}},
	}
	raw, err := json.Marshal(world)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "world.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func decodeModelDDL(t *testing.T, stdout *bytes.Buffer) modelDDLCLIResponse {
	t.Helper()
	var response modelDDLCLIResponse
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("non-JSON response: %v: %s", err, stdout.String())
	}
	return response
}

func TestCLIModelDDLWorldPath(t *testing.T) {
	worldPath := writeModelDDLWorldFile(t)
	var stdout bytes.Buffer
	code, err := runModelCommand([]string{"ddl", "-world", worldPath, "-format", "sql"}, &stdout)
	if err != nil || code != 0 {
		t.Fatalf("model ddl -world: exit=%d err=%v body=%s", code, err, stdout.String())
	}
	response := decodeModelDDL(t, &stdout)
	if !response.OK || response.Error != nil || response.Result == nil {
		t.Fatalf("unexpected response: %+v", response)
	}
	if response.SchemaVersion != "tei.cli.model-ddl/v0" {
		t.Fatalf("envelope schemaVersion=%q", response.SchemaVersion)
	}
	if response.Result.SchemaVersion != "tei.operations.model-ddl/v0" {
		t.Fatalf("result schemaVersion=%q", response.Result.SchemaVersion)
	}
	want := "CREATE TABLE IF NOT EXISTS public.orders (\n" +
		"  order_id bigint NOT NULL,\n" +
		"  status varchar(255) NOT NULL,\n" +
		"  PRIMARY KEY (order_id)\n" +
		");\n"
	if response.Result.DDL != want {
		t.Fatalf("ddl mismatch:\n got:\n%s\nwant:\n%s", response.Result.DDL, want)
	}
	// A -world read has no Source Index, so the field stays absent.
	if response.Result.SourceIndexFingerprint != "" {
		t.Fatalf("sourceIndexFingerprint should be empty for -world, got %q", response.Result.SourceIndexFingerprint)
	}
}

func TestCLIModelDDLWorkspacePath(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	var stdout bytes.Buffer
	code, err := runModelCommand([]string{"ddl", "-workspace", root, "-format", "json"}, &stdout)
	if err != nil || code != 0 {
		t.Fatalf("model ddl -workspace: exit=%d err=%v body=%s", code, err, stdout.String())
	}
	response := decodeModelDDL(t, &stdout)
	if !response.OK || response.Result == nil {
		t.Fatalf("unexpected response: %+v", response)
	}
	// A Workspace read carries the Source Index fingerprint; a -world read does not.
	if response.Result.SourceIndexFingerprint == "" {
		t.Fatal("sourceIndexFingerprint missing for -workspace")
	}
	if response.Result.Format != "json" {
		t.Fatalf("format=%q", response.Result.Format)
	}
	// The SQL is returned inside the JSON answer too.
	if !strings.Contains(response.Result.DDL, "CREATE TABLE IF NOT EXISTS public.orders") {
		t.Fatalf("ddl missing orders table:\n%s", response.Result.DDL)
	}
	if !strings.Contains(response.Result.DDL, "CREATE INDEX IF NOT EXISTS orders_customer_idx ON public.orders (customer_id);") {
		t.Fatalf("ddl missing orders index:\n%s", response.Result.DDL)
	}
	if len(response.Result.Tables) < 2 {
		t.Fatalf("expected at least two tables, got %d", len(response.Result.Tables))
	}
}

func TestCLIModelDDLSubjectSelectsOneTable(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	var stdout bytes.Buffer
	code, err := runModelCommand([]string{"ddl", "-workspace", root,
		"-subject-kind", "semantic-data", "-subject-id", "sales.order"}, &stdout)
	if err != nil || code != 0 {
		t.Fatalf("model ddl subject: exit=%d err=%v body=%s", code, err, stdout.String())
	}
	response := decodeModelDDL(t, &stdout)
	if len(response.Result.Tables) != 1 || response.Result.Tables[0].QualifiedName != "public.orders" {
		t.Fatalf("subject filter did not select exactly the orders table: %+v", response.Result.Tables)
	}
}

func TestCLIModelDDLErrorContract(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	for _, args := range [][]string{
		{"ddl"},
		{"ddl", "-workspace", root, "-world", "x"},
		{"ddl", "-workspace", root, "-format", "yaml"},
		{"ddl", "-workspace", root, "-format", ""},
		{"ddl", "-workspace", root, "-subject-kind", "semantic-data"},
		{"ddl", "-workspace", root, "-subject-kind", "query", "-subject-id", "x"},
		{"ddl", "-workspace", root, "stray"},
	} {
		var stdout bytes.Buffer
		code, err := runModelCommand(args, &stdout)
		if err != nil || code != 2 {
			t.Fatalf("args %q: exit=%d err=%v", args, code, err)
		}
		response := decodeModelDDL(t, &stdout)
		if response.OK || response.Error == nil || response.Error.Code != "INVALID_ARGUMENT" {
			t.Fatalf("args %q: %+v", args, response)
		}
		if response.SchemaVersion != "tei.cli.model-ddl/v0" {
			t.Fatalf("args %q envelope schemaVersion=%q", args, response.SchemaVersion)
		}
	}
	// A named subject no table backs is its own code and exit 2 (an argument the
	// caller got wrong: a subject that is not a persisted Data element).
	var stdout bytes.Buffer
	code, err := runModelCommand([]string{"ddl", "-workspace", root,
		"-subject-kind", "semantic-data", "-subject-id", "sales.nonexistent"}, &stdout)
	if err != nil || code != 2 {
		t.Fatalf("subject-not-found: exit=%d err=%v", code, err)
	}
	response := decodeModelDDL(t, &stdout)
	if response.OK || response.Error == nil || response.Error.Code != "MODEL_DDL_SUBJECT_NOT_FOUND" {
		t.Fatalf("subject-not-found response: %+v", response)
	}
}

func TestCLIModelDDLHelp(t *testing.T) {
	var stdout bytes.Buffer
	code, err := runModelCommand([]string{"ddl", "-h"}, &stdout)
	if err != nil || code != 0 {
		t.Fatalf("model ddl -h: exit=%d err=%v", code, err)
	}
	if !strings.HasPrefix(stdout.String(), "usage: tei ") {
		t.Fatalf("help does not start with a usage line: %q", stdout.String())
	}
	for _, want := range []string{"-workspace", "-world", "-format", "-subject-kind"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("help omits %q: %q", want, stdout.String())
		}
	}
}

// The command is advertised by `tei commands` with exactly the flags it accepts
// and no write flags: it reads the model and prints, writing nothing to disk.
func TestCLIModelDDLAppearsInCommandsCatalog(t *testing.T) {
	record, ok := discoveryRecord("model ddl")
	if !ok {
		t.Fatal("model ddl is not advertised by tei commands")
	}
	if record.Effect != "read-only" || record.JSONContract != "enveloped" ||
		record.SchemaVersion != "tei.cli.model-ddl/v0" {
		t.Fatalf("unexpected record: %+v", record)
	}
	if !sameStrings(sortedFlagNames(record.RequiredFlags), []string{"workspace", "world"}) {
		t.Fatalf("requiredFlags=%v, want exactly workspace and world", record.RequiredFlags)
	}
	if !sameStrings(sortedFlagNames(record.OptionalFlags), []string{"format", "subject-id", "subject-kind"}) {
		t.Fatalf("optionalFlags=%v", record.OptionalFlags)
	}
	if len(record.WriteFlags) != 0 {
		t.Fatalf("model ddl must declare no writeFlags, got %v", record.WriteFlags)
	}
}

// columnsFromCreateTable and pkFromCreateTable recover the column names (in
// order) and the primary-key columns from one rendered CREATE TABLE statement.
func columnsFromCreateTable(stmt string) []string {
	open := strings.Index(stmt, "(")
	body := stmt[open+1 : strings.LastIndex(stmt, ");")]
	var cols []string
	for _, raw := range strings.Split(body, "\n") {
		line := strings.TrimSuffix(strings.TrimSpace(raw), ",")
		if line == "" ||
			strings.HasPrefix(line, "--") ||
			strings.HasPrefix(line, "PRIMARY KEY") ||
			strings.HasPrefix(line, "CONSTRAINT") ||
			strings.HasPrefix(line, "UNIQUE") ||
			strings.HasPrefix(line, "FOREIGN KEY") {
			continue
		}
		cols = append(cols, strings.Fields(line)[0])
	}
	return cols
}

func pkFromCreateTable(stmt string) []string {
	for _, raw := range strings.Split(stmt, "\n") {
		line := strings.TrimSuffix(strings.TrimSpace(raw), ",")
		if !strings.HasPrefix(line, "PRIMARY KEY") {
			continue
		}
		open := strings.Index(line, "(")
		closeIdx := strings.LastIndex(line, ")")
		var out []string
		for _, part := range strings.Split(line[open+1:closeIdx], ",") {
			out = append(out, strings.TrimSpace(part))
		}
		return out
	}
	return nil
}

// TestCLIModelDDLReferenceWorkspaceSQLParsesBack proves the SQL the command
// emits for the real reference workspace parses back to the same tables, columns
// and primary keys the table-definition derivation carries.
func TestCLIModelDDLReferenceWorkspaceSQLParsesBack(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load reference workspace: %v", err)
	}
	data := map[core.CanonicalID]semantic.Data{}
	for _, ns := range loaded.World.Semantics {
		for _, element := range ns.Data {
			data[element.ID] = element
		}
	}
	byID := map[core.CanonicalID]persistence.Mapping{}
	for _, mapping := range loaded.World.Persistence {
		byID[mapping.ID] = mapping
	}
	want := map[string]struct {
		columns []string
		pk      []string
	}{}
	for _, mapping := range loaded.World.Persistence {
		element, ok := data[mapping.SemanticData]
		if !ok {
			continue
		}
		snapshot := tabledef.TableDefinitionSnapshotWith(mapping, element, func(id core.CanonicalID) (persistence.Mapping, bool) {
			referenced, ok := byID[id]
			return referenced, ok
		})
		table := snapshot.Tables[0]
		var columns []string
		for _, column := range table.Columns {
			columns = append(columns, column.Name)
		}
		want[table.QualifiedName()] = struct {
			columns []string
			pk      []string
		}{columns, table.PrimaryKey}
	}

	var stdout bytes.Buffer
	code, err := runModelCommand([]string{"ddl", "-workspace", root, "-format", "json"}, &stdout)
	if err != nil || code != 0 {
		t.Fatalf("model ddl: exit=%d err=%v body=%s", code, err, stdout.String())
	}
	response := decodeModelDDL(t, &stdout)
	if len(response.Result.Tables) != len(want) {
		t.Fatalf("rendered %d tables, derivation has %d", len(response.Result.Tables), len(want))
	}
	for _, table := range response.Result.Tables {
		w, ok := want[table.QualifiedName]
		if !ok {
			t.Fatalf("rendered unexpected table %q", table.QualifiedName)
		}
		if strings.Count(table.Statement, "CREATE TABLE IF NOT EXISTS") != 1 {
			t.Fatalf("%s does not have exactly one CREATE TABLE", table.QualifiedName)
		}
		if got := columnsFromCreateTable(table.Statement); !reflect.DeepEqual(got, w.columns) {
			t.Fatalf("%s columns=%v, want %v", table.QualifiedName, got, w.columns)
		}
		if got := pkFromCreateTable(table.Statement); !reflect.DeepEqual(got, w.pk) {
			t.Fatalf("%s primary key=%v, want %v", table.QualifiedName, got, w.pk)
		}
	}
}

// TestCLIModelDDLThroughBuiltBinary runs the compiled CLI end to end, the way a
// user does, and asserts the SQL it prints for the reference workspace.
func TestCLIModelDDLThroughBuiltBinary(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the CLI")
	}
	binary := buildCLIForLifecycle(t)
	root := filepath.Join("..", "..", "reference", "workspace")
	cmd := exec.Command(binary, "model", "ddl", "-workspace", root, "-format", "sql")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("tei model ddl failed: %v\nstderr=%s", err, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("enveloped command wrote stderr: %q", stderr.String())
	}
	var response modelDDLCLIResponse
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("non-JSON: %v: %s", err, stdout.String())
	}
	if !response.OK || response.Result == nil || response.SchemaVersion != "tei.cli.model-ddl/v0" {
		t.Fatalf("unexpected response: %+v", response)
	}
	for _, want := range []string{
		"CREATE TABLE IF NOT EXISTS public.order_lines (",
		"CREATE TABLE IF NOT EXISTS public.orders (",
		"CREATE INDEX IF NOT EXISTS orders_customer_idx ON public.orders (customer_id);",
	} {
		if !strings.Contains(response.Result.DDL, want) {
			t.Fatalf("ddl missing %q:\n%s", want, response.Result.DDL)
		}
	}
}

// TestCLIModelDDLExecutesOnPostgres proves the generated DDL is executable
// PostgreSQL, not just well-formed text. It runs only against a disposable
// database the operator has explicitly confirmed, in a scratch schema; without
// TEI_VIEW_PG_DSN it skips, and a skip is not evidence the SQL executes.
func TestCLIModelDDLExecutesOnPostgres(t *testing.T) {
	const databaseName = "tei_model_ddl_scratch"
	dsn := os.Getenv("TEI_VIEW_PG_DSN")
	if dsn == "" {
		t.Skip("TEI_VIEW_PG_DSN is unset; generated DDL was not executed against a real PostgreSQL (a skip is not evidence)")
	}
	if os.Getenv("TEI_VIEW_CONFIRM_ISOLATED") != databaseName {
		t.Fatal("refusing to modify database without explicit disposable database confirmation")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal("cannot initialize disposable PostgreSQL database")
	}
	t.Cleanup(func() { _ = db.Close() })
	var actual string
	if err := db.QueryRow("select current_database()").Scan(&actual); err != nil || actual != databaseName {
		t.Fatalf("disposable PostgreSQL database check failed: actual=%q err=%v", actual, err)
	}

	root := filepath.Join("..", "..", "reference", "workspace")
	var stdout bytes.Buffer
	code, err := runModelCommand([]string{"ddl", "-workspace", root, "-format", "sql"}, &stdout)
	if err != nil || code != 0 {
		t.Fatalf("model ddl: exit=%d err=%v body=%s", code, err, stdout.String())
	}
	response := decodeModelDDL(t, &stdout)

	// The generated DDL declares the public schema; isolate this test's objects
	// in a scratch schema so the run leaves the database as it found it.
	const scratchSchema = "model_ddl_scratch"
	if _, err := db.Exec("drop schema if exists " + scratchSchema + " cascade"); err != nil {
		t.Fatalf("cannot reset scratch schema: %v", err)
	}
	if _, err := db.Exec("create schema " + scratchSchema); err != nil {
		t.Fatalf("cannot create scratch schema: %v", err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec("drop schema if exists " + scratchSchema + " cascade"); err != nil {
			t.Errorf("cannot drop scratch schema: %v", err)
		}
	})
	scoped := strings.ReplaceAll(response.Result.DDL, "public.", scratchSchema+".")
	for _, statement := range strings.Split(scoped, ";") {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("generated DDL did not execute: %v\nstatement:\n%s", err, statement)
		}
	}
	// The tables and index the DDL created must now exist in the scratch schema.
	for _, relation := range []string{scratchSchema + ".order_lines", scratchSchema + ".orders"} {
		var name string
		if err := db.QueryRow("select to_regclass($1)::text", relation).Scan(&name); err != nil || name == "" {
			t.Fatalf("relation %s was not created by the generated DDL (regclass=%q err=%v)", relation, name, err)
		}
	}
}
