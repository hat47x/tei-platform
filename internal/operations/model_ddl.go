package operations

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"tei/internal/core"
	"tei/internal/database/persistence"
	"tei/internal/database/physical"
	"tei/internal/database/tabledef"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// ModelDDLSubjectKind is the only subject kind that selects one table: a table
// is the physical realization of a single semantic-data element, so a subject
// that is not a semantic-data element names no table.
const ModelDDLSubjectKind = "semantic-data"

const (
	modelDDLSchemaVersion = "tei.operations.model-ddl/v0"

	// ModelDDLFormatSQL returns the SQL as one string in result.ddl.
	ModelDDLFormatSQL = "sql"
	// ModelDDLFormatJSON returns the structured form in result.tables. The SQL
	// is returned in result.ddl in both formats, so a caller never has to parse
	// SQL out of a structured answer or reconstruct SQL from the structure.
	ModelDDLFormatJSON = "json"
)

// ModelDDLOptions configures one read-only DDL projection. Format selects the
// answer's emphasis; both result.ddl and result.tables are always populated.
// SubjectKind/SubjectID optionally restrict the answer to the one table a
// semantic-data element realizes. SourceIndexFingerprint is supplied by a caller
// that read the World from a Workspace and built the Source Index; it is absent
// when the World came from a -world file, which has no Source Index.
type ModelDDLOptions struct {
	Format                 string
	SubjectKind            string
	SubjectID              string
	SourceIndexFingerprint string
}

// ModelDDLResult is the read-only answer. It is a projection of the declared
// persistence mappings, not an applied migration and not evidence that any
// database holds this schema.
type ModelDDLResult struct {
	SchemaVersion          string          `json:"schemaVersion"`
	Effect                 string          `json:"effect"`
	Format                 string          `json:"format"`
	WorldFingerprint       string          `json:"worldFingerprint"`
	SourceIndexFingerprint string          `json:"sourceIndexFingerprint,omitempty"`
	Tables                 []ModelDDLTable `json:"tables"`
	// DDL is every table's CREATE TABLE and its indexes, concatenated in the
	// deterministic table order. It is present in both formats, so the answer
	// stays machine-readable whether the caller asked for the SQL text or the
	// structured tables.
	DDL             string                   `json:"ddl"`
	UnmappedColumns []ModelDDLUnmappedColumn `json:"unmappedColumns"`
	Counts          ModelDDLCounts           `json:"counts"`
	AuthorityLimits []string                 `json:"authorityLimits"`
}

// ModelDDLTable is one table's structured form. Statement is the CREATE TABLE
// text; IndexStatements are the separate CREATE INDEX statements that follow it.
type ModelDDLTable struct {
	QualifiedName   string   `json:"qualifiedName"`
	Statement       string   `json:"statement"`
	IndexStatements []string `json:"indexStatements,omitempty"`
}

// ModelDDLUnmappedColumn names a column whose declared storage type has no
// PostgreSQL type this projection can defend. The column is emitted as a
// comment, never guessed, and listed here so a caller sees it without parsing
// the SQL.
type ModelDDLUnmappedColumn struct {
	Table        string `json:"table"`
	Column       string `json:"column"`
	DeclaredType string `json:"declaredType"`
}

// ModelDDLCounts summarizes the answer without parsing it: how many tables,
// columns and unmapped columns it carries. These are counts of what was
// rendered, not a claim that every declaration in the World was rendered.
type ModelDDLCounts struct {
	Tables          int `json:"tables"`
	Columns         int `json:"columns"`
	UnmappedColumns int `json:"unmappedColumns"`
}

// ModelDDLSubjectNotFound reports that the requested subject names no
// semantic-data element a persistence mapping realizes. It is distinct from an
// empty answer: the caller named a subject and no table backs it.
type ModelDDLSubjectNotFound struct {
	Kind string
	ID   string
}

func (e *ModelDDLSubjectNotFound) Error() string {
	return fmt.Sprintf("no persistence mapping realizes %s %q in this world", e.Kind, e.ID)
}

// ModelDDL renders one CREATE TABLE statement (and its indexes) per table the
// World's persistence mappings declare. The table structure comes from the one
// derivation in internal/database/tabledef; this projection only renders it as
// PostgreSQL and orders it deterministically, so two runs on the same World
// produce byte-identical text.
func ModelDDL(world ir.World, options ModelDDLOptions) (ModelDDLResult, error) {
	format, err := normalizeModelDDLFormat(options.Format)
	if err != nil {
		return ModelDDLResult{}, err
	}
	subjectKind, subjectID, err := normalizeModelDDLSubject(options.SubjectKind, options.SubjectID)
	if err != nil {
		return ModelDDLResult{}, err
	}
	worldFP, err := worldFingerprint(world)
	if err != nil {
		return ModelDDLResult{}, err
	}

	data := map[core.CanonicalID]semantic.Data{}
	for _, namespace := range world.Semantics {
		for _, element := range namespace.Data {
			data[element.ID] = element
		}
	}
	mappingByID := map[core.CanonicalID]persistence.Mapping{}
	for _, mapping := range world.Persistence {
		mappingByID[mapping.ID] = mapping
	}
	resolve := func(id core.CanonicalID) (persistence.Mapping, bool) {
		referenced, ok := mappingByID[id]
		return referenced, ok
	}

	type renderedTable struct {
		table    ModelDDLTable
		unmapped []ModelDDLUnmappedColumn
		columns  int
	}
	var rendered []renderedTable
	subjectMatched := subjectID == ""
	for _, mapping := range world.Persistence {
		element, ok := data[mapping.SemanticData]
		if !ok {
			continue
		}
		if subjectID != "" {
			if string(mapping.SemanticData) != subjectID {
				continue
			}
			subjectMatched = true
		}
		snapshot := tabledef.TableDefinitionSnapshotWith(mapping, element, resolve)
		if len(snapshot.Tables) == 0 {
			continue
		}
		table, unmapped := renderModelDDLTable(snapshot.Tables[0])
		rendered = append(rendered, renderedTable{
			table:    table,
			unmapped: unmapped,
			columns:  len(snapshot.Tables[0].Columns),
		})
	}
	if !subjectMatched {
		return ModelDDLResult{}, &ModelDDLSubjectNotFound{Kind: subjectKind, ID: subjectID}
	}

	sort.SliceStable(rendered, func(i, j int) bool {
		return rendered[i].table.QualifiedName < rendered[j].table.QualifiedName
	})

	result := ModelDDLResult{
		SchemaVersion:          modelDDLSchemaVersion,
		Effect:                 "read-only",
		Format:                 format,
		WorldFingerprint:       worldFP,
		SourceIndexFingerprint: options.SourceIndexFingerprint,
		Tables:                 []ModelDDLTable{},
		UnmappedColumns:        []ModelDDLUnmappedColumn{},
		AuthorityLimits: []string{
			"model-ddl-is-a-derived-read-only-projection",
			"ddl-is-not-applied-to-any-database",
			"unmapped-column-type-is-reported-not-guessed",
			"ddl-does-not-observe-or-assert-any-live-schema",
		},
	}
	var ddl strings.Builder
	columns := 0
	for _, entry := range rendered {
		result.Tables = append(result.Tables, entry.table)
		result.UnmappedColumns = append(result.UnmappedColumns, entry.unmapped...)
		columns += entry.columns
		ddl.WriteString(entry.table.Statement)
		ddl.WriteString("\n")
		for _, index := range entry.table.IndexStatements {
			ddl.WriteString(index)
			ddl.WriteString("\n")
		}
	}
	result.DDL = ddl.String()
	result.Counts = ModelDDLCounts{
		Tables:          len(result.Tables),
		Columns:         columns,
		UnmappedColumns: len(result.UnmappedColumns),
	}
	return result, nil
}

func normalizeModelDDLFormat(format string) (string, error) {
	switch format {
	case "", ModelDDLFormatSQL:
		return ModelDDLFormatSQL, nil
	case ModelDDLFormatJSON:
		return ModelDDLFormatJSON, nil
	default:
		return "", fmt.Errorf("model ddl: format must be %q or %q", ModelDDLFormatSQL, ModelDDLFormatJSON)
	}
}

func normalizeModelDDLSubject(kind, id string) (string, string, error) {
	if strings.TrimSpace(kind) == "" && strings.TrimSpace(id) == "" {
		return "", "", nil
	}
	if strings.TrimSpace(kind) == "" || strings.TrimSpace(id) == "" {
		return "", "", fmt.Errorf("model ddl: -subject-kind and -subject-id must be given together")
	}
	if kind != ModelDDLSubjectKind {
		return "", "", fmt.Errorf("model ddl: -subject-kind must be %q", ModelDDLSubjectKind)
	}
	return kind, id, nil
}

// ddlItem is one line of a CREATE TABLE body. A real item participates in the
// comma-separated definition list; a comment item (an unmapped column) stands on
// its own line and never carries a separating comma.
type ddlItem struct {
	real bool
	text string
}

func renderModelDDLTable(table physical.Table) (ModelDDLTable, []ModelDDLUnmappedColumn) {
	qualified := table.QualifiedName()
	qualifiedSQL := renderQualifiedName(table.Schema, table.Name)

	columns := append([]physical.Column(nil), table.Columns...)
	sort.SliceStable(columns, func(i, j int) bool {
		if columns[i].Ordinal != columns[j].Ordinal {
			return columns[i].Ordinal < columns[j].Ordinal
		}
		return columns[i].Name < columns[j].Name
	})

	var items []ddlItem
	var unmapped []ModelDDLUnmappedColumn
	for _, column := range columns {
		pgType, ok := postgresColumnType(column.PhysicalType)
		if !ok {
			unmapped = append(unmapped, ModelDDLUnmappedColumn{
				Table:        qualified,
				Column:       column.Name,
				DeclaredType: column.PhysicalType,
			})
			items = append(items, ddlItem{
				real: false,
				text: "-- unmapped: " + quoteIdentifier(column.Name) + " " + column.PhysicalType,
			})
			continue
		}
		definition := quoteIdentifier(column.Name) + " " + pgType
		if !column.Nullable {
			definition += " NOT NULL"
		}
		items = append(items, ddlItem{real: true, text: definition})
	}

	if len(table.PrimaryKey) > 0 {
		items = append(items, ddlItem{real: true, text: "PRIMARY KEY (" + joinColumnList(table.PrimaryKey) + ")"})
	}

	uniques := append([]physical.UniqueConstraint(nil), table.UniqueConstraints...)
	sort.SliceStable(uniques, func(i, j int) bool { return uniques[i].Name < uniques[j].Name })
	for _, unique := range uniques {
		if len(unique.Columns) == 0 {
			continue
		}
		definition := ""
		if name := strings.TrimSpace(unique.Name); name != "" {
			definition = "CONSTRAINT " + quoteIdentifier(name) + " "
		}
		definition += "UNIQUE (" + joinColumnList(unique.Columns) + ")"
		items = append(items, ddlItem{real: true, text: definition})
	}

	foreignKeys := append([]physical.ForeignKey(nil), table.ForeignKeys...)
	sort.SliceStable(foreignKeys, func(i, j int) bool { return foreignKeys[i].Name < foreignKeys[j].Name })
	for _, foreignKey := range foreignKeys {
		if len(foreignKey.Columns) == 0 ||
			len(foreignKey.ReferencedColumns) != len(foreignKey.Columns) ||
			strings.TrimSpace(foreignKey.ReferencedTable) == "" {
			continue
		}
		definition := ""
		if name := strings.TrimSpace(foreignKey.Name); name != "" {
			definition = "CONSTRAINT " + quoteIdentifier(name) + " "
		}
		definition += "FOREIGN KEY (" + joinColumnList(foreignKey.Columns) + ") REFERENCES " +
			renderQualifiedName(foreignKey.ReferencedSchema, foreignKey.ReferencedTable) +
			" (" + joinColumnList(foreignKey.ReferencedColumns) + ")"
		items = append(items, ddlItem{real: true, text: definition})
	}

	realCount := 0
	for _, item := range items {
		if item.real {
			realCount++
		}
	}
	var body strings.Builder
	realIndex := 0
	for _, item := range items {
		body.WriteString("  ")
		body.WriteString(item.text)
		if item.real {
			if realIndex < realCount-1 {
				body.WriteString(",")
			}
			realIndex++
		}
		body.WriteString("\n")
	}
	statement := "CREATE TABLE IF NOT EXISTS " + qualifiedSQL + " (\n" + body.String() + ");"

	indexes := append([]physical.Index(nil), table.Indexes...)
	sort.SliceStable(indexes, func(i, j int) bool { return indexes[i].Name < indexes[j].Name })
	var indexStatements []string
	for _, index := range indexes {
		if stmt, ok := renderModelDDLIndex(qualifiedSQL, index); ok {
			indexStatements = append(indexStatements, stmt)
		}
	}
	return ModelDDLTable{
		QualifiedName:   qualified,
		Statement:       statement,
		IndexStatements: indexStatements,
	}, unmapped
}

func renderModelDDLIndex(table string, index physical.Index) (string, bool) {
	name := strings.TrimSpace(index.Name)
	if name == "" || len(index.Keys) == 0 {
		return "", false
	}
	keys := make([]string, 0, len(index.Keys))
	for _, key := range index.Keys {
		// The table-definition snapshot only produces column keys; an
		// expression key would be a different producer and is not rendered here.
		column := strings.TrimSpace(key.Column)
		if column == "" {
			return "", false
		}
		rendered := quoteIdentifier(column)
		if key.Order == physical.IndexOrderDesc {
			rendered += " DESC"
		}
		keys = append(keys, rendered)
	}
	statement := "CREATE "
	if index.Unique {
		statement += "UNIQUE "
	}
	statement += "INDEX IF NOT EXISTS " + quoteIdentifier(name) + " ON " + table + " (" + strings.Join(keys, ", ") + ");"
	return statement, true
}

// postgresColumnType maps a canonical TEI storage type to a PostgreSQL type. It
// mirrors internal/database/sqlgen/postgres Dialect.PhysicalType, which cannot
// be imported here because that package transitively depends on net and the
// PostgreSQL driver, both outside the operation boundary. The storage types are
// a closed enum, so this stays a one-to-one mirror; an unknown type is reported
// rather than guessed.
func postgresColumnType(storage string) (string, bool) {
	switch persistence.StorageType(storage) {
	case persistence.StorageInt64:
		return "bigint", true
	case persistence.StorageVarchar:
		return "varchar(255)", true
	case persistence.StorageText:
		return "text", true
	case persistence.StorageDateTimeTZ:
		return "timestamptz", true
	case persistence.StorageDecimal:
		return "numeric(18,2)", true
	case persistence.StorageJSON:
		return "jsonb", true
	default:
		return "", false
	}
}

func joinColumnList(columns []string) string {
	out := make([]string, len(columns))
	for i, column := range columns {
		out[i] = quoteIdentifier(column)
	}
	return strings.Join(out, ", ")
}

func renderQualifiedName(schema, name string) string {
	if strings.TrimSpace(schema) == "" {
		return quoteIdentifier(name)
	}
	return quoteIdentifier(schema) + "." + quoteIdentifier(name)
}

var safeIdentifierPattern = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

// postgresReservedWords are the identifiers PostgreSQL will not accept
// unquoted. Quoting them keeps the emitted DDL valid; the common workspace
// identifiers (lowercase snake case) are absent from this set and stay bare.
var postgresReservedWords = map[string]bool{
	"all": true, "analyse": true, "analyze": true, "and": true, "any": true,
	"array": true, "as": true, "asc": true, "asymmetric": true, "authorization": true,
	"binary": true, "both": true, "case": true, "cast": true, "check": true,
	"collate": true, "collation": true, "column": true, "concurrently": true,
	"constraint": true, "create": true, "cross": true, "current_catalog": true,
	"current_date": true, "current_role": true, "current_schema": true,
	"current_time": true, "current_timestamp": true, "current_user": true,
	"default": true, "deferrable": true, "desc": true, "distinct": true, "do": true,
	"else": true, "end": true, "except": true, "false": true, "fetch": true,
	"for": true, "foreign": true, "freeze": true, "from": true, "full": true,
	"grant": true, "group": true, "having": true, "ilike": true, "in": true,
	"initially": true, "inner": true, "intersect": true, "into": true, "is": true,
	"isnull": true, "join": true, "lateral": true, "leading": true, "left": true,
	"like": true, "limit": true, "localtime": true, "localtimestamp": true,
	"natural": true, "not": true, "notnull": true, "null": true, "offset": true,
	"on": true, "only": true, "or": true, "order": true, "outer": true, "overlaps": true,
	"overlay": true, "placing": true, "primary": true, "references": true,
	"returning": true, "right": true, "select": true, "session_user": true,
	"similar": true, "some": true, "symmetric": true, "table": true, "tablesample": true,
	"then": true, "to": true, "trailing": true, "true": true, "union": true,
	"unique": true, "user": true, "using": true, "variadic": true, "verbose": true,
	"when": true, "where": true, "window": true, "with": true,
}

// quoteIdentifier quotes an identifier only when PostgreSQL requires it: an
// empty name, a name that is not a bare lowercase identifier, or a reserved
// word. A double quote inside the name is doubled.
func quoteIdentifier(value string) string {
	if needsQuoting(value) {
		return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
	}
	return value
}

func needsQuoting(value string) bool {
	if value == "" {
		return true
	}
	if !safeIdentifierPattern.MatchString(value) {
		return true
	}
	return postgresReservedWords[strings.ToLower(value)]
}
