package operations

import (
	"errors"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/database/persistence"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// modelDDLWorld is an implementation-independent fixture: two tables, one
// foreign key that must be resolved to the referenced table's real name, one
// unique constraint, one index, and one column whose declared storage type has
// no PostgreSQL mapping.
func modelDDLWorld() ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{{
			ID: "sales",
			Data: []semantic.Data{
				{ID: "sales.customer"},
				{ID: "sales.order"},
			},
		}},
		Persistence: []persistence.Mapping{
			{
				ID:           "sales_core.customer",
				Database:     "sales_core",
				SemanticData: "sales.customer",
				Table:        "public.customers",
				PrimaryKey:   []core.CanonicalID{"sales.customer.customer_id"},
				Columns: []persistence.ColumnMapping{
					{Attribute: "sales.customer.customer_id", Column: "customer_id", Storage: persistence.StorageInt64, Nullability: persistence.Required},
					{Attribute: "sales.customer.email", Column: "email", Storage: persistence.StorageVarchar, Nullability: persistence.Required},
					{Attribute: "sales.customer.name", Column: "name", Storage: persistence.StorageText, Nullability: persistence.Optional},
				},
				UniqueConstraints: []persistence.UniqueConstraintMapping{
					{Name: "customers_email_key", Attributes: []core.CanonicalID{"sales.customer.email"}},
				},
			},
			{
				ID:           "sales_core.order",
				Database:     "sales_core",
				SemanticData: "sales.order",
				Table:        "public.orders",
				PrimaryKey:   []core.CanonicalID{"sales.order.order_id"},
				Columns: []persistence.ColumnMapping{
					{Attribute: "sales.order.order_id", Column: "order_id", Storage: persistence.StorageInt64, Nullability: persistence.Required},
					{Attribute: "sales.order.customer_id", Column: "customer_id", Storage: persistence.StorageInt64, Nullability: persistence.Required},
					{Attribute: "sales.order.status", Column: "status", Storage: persistence.StorageVarchar, Nullability: persistence.Required},
					{Attribute: "sales.order.notes", Column: "notes", Storage: persistence.StorageText, Nullability: persistence.Optional},
					{Attribute: "sales.order.region", Column: "region", Storage: persistence.StorageType("geometry"), Nullability: persistence.Optional},
				},
				ForeignKeys: []persistence.ForeignKeyMapping{
					{
						Name:                 "orders_customer_fk",
						Attributes:           []core.CanonicalID{"sales.order.customer_id"},
						ReferencedMapping:    "sales_core.customer",
						ReferencedAttributes: []core.CanonicalID{"sales.customer.customer_id"},
					},
				},
				Indexes: []persistence.IndexMapping{
					{Name: "orders_status_idx", Attributes: []core.CanonicalID{"sales.order.status"}},
				},
			},
		},
	}
}

const modelDDLWant = `CREATE TABLE IF NOT EXISTS public.customers (
  customer_id bigint NOT NULL,
  email varchar(255) NOT NULL,
  name text,
  PRIMARY KEY (customer_id),
  CONSTRAINT customers_email_key UNIQUE (email)
);
CREATE TABLE IF NOT EXISTS public.orders (
  order_id bigint NOT NULL,
  customer_id bigint NOT NULL,
  status varchar(255) NOT NULL,
  notes text,
  -- unmapped: region geometry
  PRIMARY KEY (order_id),
  CONSTRAINT orders_customer_fk FOREIGN KEY (customer_id) REFERENCES public.customers (customer_id)
);
CREATE INDEX IF NOT EXISTS orders_status_idx ON public.orders (status);
`

func TestModelDDLRendersDeterministicPostgreSQL(t *testing.T) {
	world := modelDDLWorld()
	first, err := ModelDDL(world, ModelDDLOptions{})
	if err != nil {
		t.Fatalf("ModelDDL: %v", err)
	}
	second, err := ModelDDL(world, ModelDDLOptions{})
	if err != nil {
		t.Fatalf("ModelDDL (second run): %v", err)
	}
	if first.DDL != second.DDL {
		t.Fatalf("two runs differ:\nfirst:\n%s\nsecond:\n%s", first.DDL, second.DDL)
	}
	if first.DDL != modelDDLWant {
		t.Fatalf("DDL mismatch:\n got:\n%s\nwant:\n%s", first.DDL, modelDDLWant)
	}
}

func TestModelDDLResultEnvelopeAndFingerprint(t *testing.T) {
	result, err := ModelDDL(modelDDLWorld(), ModelDDLOptions{Format: ModelDDLFormatJSON})
	if err != nil {
		t.Fatalf("ModelDDL: %v", err)
	}
	if result.SchemaVersion != "tei.operations.model-ddl/v0" {
		t.Fatalf("schemaVersion=%q", result.SchemaVersion)
	}
	if result.Effect != "read-only" || result.Format != ModelDDLFormatJSON {
		t.Fatalf("effect=%q format=%q", result.Effect, result.Format)
	}
	if !strings.HasPrefix(result.WorldFingerprint, "sha256:") {
		t.Fatalf("worldFingerprint=%q", result.WorldFingerprint)
	}
	// The SQL is returned inside the JSON answer too, so the structured form is
	// never the only machine-readable shape.
	if result.DDL != modelDDLWant {
		t.Fatalf("json format dropped the SQL:\n%s", result.DDL)
	}
	// A -world read has no Source Index; the field must stay absent.
	if result.SourceIndexFingerprint != "" {
		t.Fatalf("sourceIndexFingerprint should be empty, got %q", result.SourceIndexFingerprint)
	}
	if len(result.Tables) != 2 {
		t.Fatalf("tables=%d, want 2", len(result.Tables))
	}
	if result.Tables[0].QualifiedName != "public.customers" || result.Tables[1].QualifiedName != "public.orders" {
		t.Fatalf("tables not sorted by qualified name: %q, %q",
			result.Tables[0].QualifiedName, result.Tables[1].QualifiedName)
	}
	if got := result.Tables[1].IndexStatements; len(got) != 1 ||
		got[0] != "CREATE INDEX IF NOT EXISTS orders_status_idx ON public.orders (status);" {
		t.Fatalf("orders index statements=%v", got)
	}
}

func TestModelDDLReportsUnmappedColumnWithoutGuessing(t *testing.T) {
	result, err := ModelDDL(modelDDLWorld(), ModelDDLOptions{})
	if err != nil {
		t.Fatalf("ModelDDL: %v", err)
	}
	if len(result.UnmappedColumns) != 1 {
		t.Fatalf("unmappedColumns=%v, want exactly one", result.UnmappedColumns)
	}
	got := result.UnmappedColumns[0]
	if got.Table != "public.orders" || got.Column != "region" || got.DeclaredType != "geometry" {
		t.Fatalf("unmapped column=%+v", got)
	}
	// The column is a comment, never a guessed type, and it does not break the
	// comma placement of the real definitions around it.
	if !strings.Contains(result.DDL, "  -- unmapped: region geometry\n") {
		t.Fatalf("DDL lacks the unmapped comment:\n%s", result.DDL)
	}
	if strings.Contains(result.DDL, "region geometry") && strings.Contains(result.DDL, "region numeric") {
		t.Fatalf("a type was guessed for the unmapped column:\n%s", result.DDL)
	}
}

func TestModelDDLSubjectSelectsOneTable(t *testing.T) {
	result, err := ModelDDL(modelDDLWorld(), ModelDDLOptions{
		SubjectKind: ModelDDLSubjectKind,
		SubjectID:   "sales.customer",
	})
	if err != nil {
		t.Fatalf("ModelDDL: %v", err)
	}
	if len(result.Tables) != 1 || result.Tables[0].QualifiedName != "public.customers" {
		t.Fatalf("tables=%v, want only public.customers", result.Tables)
	}
	if strings.Contains(result.DDL, "public.orders") {
		t.Fatalf("subject filter leaked another table:\n%s", result.DDL)
	}
}

func TestModelDDLSubjectNotFoundIsAnError(t *testing.T) {
	_, err := ModelDDL(modelDDLWorld(), ModelDDLOptions{
		SubjectKind: ModelDDLSubjectKind,
		SubjectID:   "sales.nonexistent",
	})
	var notFound *ModelDDLSubjectNotFound
	if !errors.As(err, &notFound) {
		t.Fatalf("want *ModelDDLSubjectNotFound, got %v", err)
	}
	if notFound.ID != "sales.nonexistent" {
		t.Fatalf("subject id=%q", notFound.ID)
	}
}

func TestModelDDLRejectsBadOptions(t *testing.T) {
	for _, tc := range []struct {
		name    string
		options ModelDDLOptions
	}{
		{"format", ModelDDLOptions{Format: "yaml"}},
		{"kind without id", ModelDDLOptions{SubjectKind: ModelDDLSubjectKind}},
		{"id without kind", ModelDDLOptions{SubjectID: "sales.order"}},
		{"wrong kind", ModelDDLOptions{SubjectKind: "query", SubjectID: "sales.order"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ModelDDL(modelDDLWorld(), tc.options); err == nil {
				t.Fatalf("options %+v were accepted", tc.options)
			}
		})
	}
}

func TestModelDDLWorldFingerprintFollowsTheModel(t *testing.T) {
	world := modelDDLWorld()
	before, err := ModelDDL(world, ModelDDLOptions{})
	if err != nil {
		t.Fatalf("ModelDDL: %v", err)
	}
	world.Persistence[1].Table = "public.sales_orders"
	after, err := ModelDDL(world, ModelDDLOptions{})
	if err != nil {
		t.Fatalf("ModelDDL: %v", err)
	}
	if before.WorldFingerprint == after.WorldFingerprint {
		t.Fatalf("changing the table did not change the world fingerprint")
	}
}

func TestModelDDLCountsAreReported(t *testing.T) {
	result, err := ModelDDL(modelDDLWorld(), ModelDDLOptions{})
	if err != nil {
		t.Fatalf("ModelDDL: %v", err)
	}
	// Two tables; customer has 3 columns, order has 5 (one of them unmapped).
	if result.Counts.Tables != 2 {
		t.Fatalf("counts.tables=%d, want 2", result.Counts.Tables)
	}
	if result.Counts.Columns != 8 {
		t.Fatalf("counts.columns=%d, want 8", result.Counts.Columns)
	}
	if result.Counts.UnmappedColumns != 1 {
		t.Fatalf("counts.unmappedColumns=%d, want 1", result.Counts.UnmappedColumns)
	}
	if result.Counts.Tables != len(result.Tables) ||
		result.Counts.UnmappedColumns != len(result.UnmappedColumns) {
		t.Fatalf("counts disagree with the lists: %+v", result.Counts)
	}
}

func TestModelDDLUnmappedColumnsHaveMatchingCommentLines(t *testing.T) {
	result, err := ModelDDL(modelDDLWorld(), ModelDDLOptions{})
	if err != nil {
		t.Fatalf("ModelDDL: %v", err)
	}
	if len(result.UnmappedColumns) == 0 {
		t.Fatal("fixture must carry at least one unmapped column")
	}
	commentLines := 0
	for _, line := range strings.Split(result.DDL, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "-- unmapped:") {
			commentLines++
		}
	}
	if commentLines != len(result.UnmappedColumns) {
		t.Fatalf("ddl has %d '-- unmapped' lines but %d unmappedColumns",
			commentLines, len(result.UnmappedColumns))
	}
	for _, entry := range result.UnmappedColumns {
		want := "  -- unmapped: " + entry.Column + " " + entry.DeclaredType + "\n"
		if !strings.Contains(result.DDL, want) {
			t.Fatalf("no matching comment line for %+v (want %q)", entry, want)
		}
	}
}

// ddlParsed is what the small test parser recovers from the rendered SQL: the
// tables in order, their columns in order (an unmapped column is recovered from
// its comment, so the round-trip still counts it), each primary key and each
// index column list.
type ddlParsed struct {
	order   []string
	columns map[string][]string
	pk      map[string][]string
	indexes map[string][]string
}

func parseDDL(ddl string) ddlParsed {
	out := ddlParsed{
		columns: map[string][]string{},
		pk:      map[string][]string{},
		indexes: map[string][]string{},
	}
	tableRe := regexp.MustCompile(`(?s)CREATE TABLE IF NOT EXISTS (\S+) \((.*?)\n\);`)
	for _, m := range tableRe.FindAllStringSubmatch(ddl, -1) {
		name := m[1]
		out.order = append(out.order, name)
		for _, raw := range strings.Split(m[2], "\n") {
			line := strings.TrimSuffix(strings.TrimSpace(raw), ",")
			switch {
			case line == "":
			case strings.HasPrefix(line, "-- unmapped: "):
				out.columns[name] = append(out.columns[name], strings.Fields(strings.TrimPrefix(line, "-- unmapped: "))[0])
			case strings.HasPrefix(line, "PRIMARY KEY"):
				out.pk[name] = parenColumns(line)
			case strings.HasPrefix(line, "CONSTRAINT"), strings.HasPrefix(line, "UNIQUE"), strings.HasPrefix(line, "FOREIGN KEY"):
				// A table constraint, not a column.
			default:
				out.columns[name] = append(out.columns[name], strings.Fields(line)[0])
			}
		}
	}
	indexRe := regexp.MustCompile(`CREATE (?:UNIQUE )?INDEX IF NOT EXISTS \S+ ON (\S+) \(([^)]*)\);`)
	for _, m := range indexRe.FindAllStringSubmatch(ddl, -1) {
		for _, col := range strings.Split(m[2], ", ") {
			out.indexes[m[1]] = append(out.indexes[m[1]], strings.TrimSpace(col))
		}
	}
	return out
}

func parenColumns(s string) []string {
	open := strings.Index(s, "(")
	close := strings.LastIndex(s, ")")
	if open < 0 || close < 0 {
		return nil
	}
	var out []string
	for _, part := range strings.Split(s[open+1:close], ",") {
		out = append(out, strings.TrimSpace(part))
	}
	return out
}

// TestModelDDLSQLParsesBackToRenderedStructure proves the emitted SQL is
// well-formed and round-trips: parsing it yields exactly the tables, columns
// (in declared order) and primary keys the mapping declared, each column once.
func TestModelDDLSQLParsesBackToRenderedStructure(t *testing.T) {
	result, err := ModelDDL(modelDDLWorld(), ModelDDLOptions{})
	if err != nil {
		t.Fatalf("ModelDDL: %v", err)
	}
	parsed := parseDDL(result.DDL)
	want := map[string]struct {
		columns []string
		pk      []string
	}{
		"public.customers": {columns: []string{"customer_id", "email", "name"}, pk: []string{"customer_id"}},
		"public.orders":    {columns: []string{"order_id", "customer_id", "status", "notes", "region"}, pk: []string{"order_id"}},
	}
	if len(parsed.order) != len(want) {
		t.Fatalf("parsed %d tables %v, want %d", len(parsed.order), parsed.order, len(want))
	}
	if parsed.order[0] != "public.customers" || parsed.order[1] != "public.orders" {
		t.Fatalf("tables not in qualified-name order: %v", parsed.order)
	}
	for name, w := range want {
		if !reflect.DeepEqual(parsed.columns[name], w.columns) {
			t.Fatalf("%s columns=%v, want %v", name, parsed.columns[name], w.columns)
		}
		if !reflect.DeepEqual(parsed.pk[name], w.pk) {
			t.Fatalf("%s primary key=%v, want %v", name, parsed.pk[name], w.pk)
		}
		seen := map[string]bool{}
		for _, col := range parsed.columns[name] {
			if seen[col] {
				t.Fatalf("%s renders column %q more than once", name, col)
			}
			seen[col] = true
		}
	}
	if got := parsed.indexes["public.orders"]; !reflect.DeepEqual(got, []string{"status"}) {
		t.Fatalf("orders index columns=%v, want [status]", got)
	}
}
