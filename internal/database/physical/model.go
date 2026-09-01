package physical

import (
	"sort"
	"strings"

	"tei/internal/core"
)

// SchemaSnapshot is observed physical metadata for one Database definition.
// It is not canonical application/domain meaning and is safe to regenerate by
// database introspection.
type SchemaSnapshot struct {
	Database    core.CanonicalID `json:"database"`
	CapturedAt  string           `json:"capturedAt,omitempty"`
	Fingerprint string           `json:"fingerprint,omitempty"`
	Tables      []Table          `json:"tables,omitempty"`
}

type TableKind string

const (
	TableKindTable            TableKind = "table"
	TableKindView             TableKind = "view"
	TableKindMaterializedView TableKind = "materialized-view"
)

type Table struct {
	Schema            string             `json:"schema,omitempty"`
	Name              string             `json:"name"`
	Kind              TableKind          `json:"kind,omitempty"`
	Columns           []Column           `json:"columns,omitempty"`
	PrimaryKey        []string           `json:"primaryKey,omitempty"`
	UniqueConstraints []UniqueConstraint `json:"uniqueConstraints,omitempty"`
	ForeignKeys       []ForeignKey       `json:"foreignKeys,omitempty"`
	Indexes           []Index            `json:"indexes,omitempty"`
}

type Column struct {
	Name         string `json:"name"`
	PhysicalType string `json:"physicalType,omitempty"`
	Nullable     bool   `json:"nullable"`
	Default      string `json:"default,omitempty"`
	Ordinal      int    `json:"ordinal,omitempty"`
}

// UniqueConstraint records relational uniqueness intent separately from the
// index that a provider may use to implement it. Name is retained for trace and
// future DDL, while structural comparison uses the ordered Columns.
type UniqueConstraint struct {
	Name    string   `json:"name,omitempty"`
	Columns []string `json:"columns"`
}

type ForeignKey struct {
	Name              string   `json:"name,omitempty"`
	Columns           []string `json:"columns"`
	ReferencedSchema  string   `json:"referencedSchema,omitempty"`
	ReferencedTable   string   `json:"referencedTable"`
	ReferencedColumns []string `json:"referencedColumns"`
}

type IndexOrder string

const (
	IndexOrderDefault IndexOrder = "default"
	IndexOrderAsc     IndexOrder = "asc"
	IndexOrderDesc    IndexOrder = "desc"
)

type IndexNullsOrder string

const (
	IndexNullsOrderDefault IndexNullsOrder = "default"
	IndexNullsOrderFirst   IndexNullsOrder = "first"
	IndexNullsOrderLast    IndexNullsOrder = "last"
)

type UniqueNullTreatment string

const (
	UniqueNullTreatmentDistinct    UniqueNullTreatment = "distinct"
	UniqueNullTreatmentNotDistinct UniqueNullTreatment = "not-distinct"
)

type ProviderExpressionRepresentation string

const (
	ProviderExpressionAuthored          ProviderExpressionRepresentation = "authored"
	ProviderExpressionProviderNormalized ProviderExpressionRepresentation = "provider-normalized"
)

// ExpressionNormalization identifies the provider-owned normalization
// authority that produced a provider-normalized expression. ProviderVersion is
// optional until a provider can supply a stable version scope offline or from
// introspection without making Workspace loading depend on a live connection.
type ExpressionNormalization struct {
	Authority       string `json:"authority"`
	ProviderVersion string `json:"providerVersion,omitempty"`
}

// ProviderExpression keeps database-owned SQL semantics outside the Core SQL
// model. Representation distinguishes authoritative author text from text that
// a provider reconstructed from its internal representation. Provider and
// Dialect prevent equal text from being assumed equivalent across DBMSs.
type ProviderExpression struct {
	Provider       string                           `json:"provider"`
	Dialect        string                           `json:"dialect,omitempty"`
	Representation ProviderExpressionRepresentation `json:"representation,omitempty"`
	Source         string                           `json:"source"`
	Normalization  *ExpressionNormalization         `json:"normalization,omitempty"`
}

// IndexKey is one ordered search key. Exactly one of Column or Expression must
// be populated by producers. Per-key options describe effective physical
// behavior after provider-side default normalization.
type IndexKey struct {
	Column        string              `json:"column,omitempty"`
	Expression    *ProviderExpression `json:"expression,omitempty"`
	Collation     string              `json:"collation,omitempty"`
	OperatorClass string              `json:"operatorClass,omitempty"`
	Order         IndexOrder          `json:"order"`
	NullsOrder    IndexNullsOrder     `json:"nullsOrder"`
}

// IndexState is actual observation only. Desired Physical Schema does not ask
// for an invalid, non-ready or non-live index.
type IndexState struct {
	Valid bool `json:"valid"`
	Ready bool `json:"ready"`
	Live  bool `json:"live"`
}

func (s IndexState) Usable() bool { return s.Valid && s.Ready && s.Live }

// Index records Database-owned access-path intent. Keys, include columns,
// predicate and uniqueness semantics are structural; State is actual lifecycle
// observation and is intentionally excluded from structural comparison.
type Index struct {
	Name                string              `json:"name"`
	Keys                []IndexKey          `json:"keys,omitempty"`
	IncludedColumns     []string            `json:"includedColumns,omitempty"`
	Predicate           *ProviderExpression `json:"predicate,omitempty"`
	Unique              bool                `json:"unique,omitempty"`
	UniqueNullTreatment UniqueNullTreatment `json:"uniqueNullTreatment,omitempty"`
	Method              string              `json:"method,omitempty"`
	State               *IndexState         `json:"state,omitempty"`
}

func ColumnIndexKey(column string) IndexKey {
	return IndexKey{Column: column, Order: IndexOrderDefault, NullsOrder: IndexNullsOrderDefault}
}

func (t Table) QualifiedName() string {
	if strings.TrimSpace(t.Schema) == "" {
		return t.Name
	}
	return t.Schema + "." + t.Name
}

// Table resolves qualified names exactly and unqualified names only when they
// identify one unique table in the snapshot. This prevents an analyzer from
// silently resolving "orders" when multiple schemas expose that name.
func (s SchemaSnapshot) Table(name string) (Table, bool) {
	needle := normalizeName(name)
	qualified := strings.Contains(needle, ".")
	var matches []Table
	for _, t := range s.Tables {
		q := normalizeName(t.QualifiedName())
		n := normalizeName(t.Name)
		if (qualified && q == needle) || (!qualified && n == needle) {
			matches = append(matches, t)
		}
	}
	if len(matches) != 1 {
		return Table{}, false
	}
	return matches[0], true
}

func (t Table) Column(name string) (Column, bool) {
	needle := normalizeName(name)
	if dot := strings.LastIndex(needle, "."); dot >= 0 {
		needle = needle[dot+1:]
	}
	for _, c := range t.Columns {
		if normalizeName(c.Name) == needle {
			return c, true
		}
	}
	return Column{}, false
}

// Normalize returns a stable ordering suitable for fingerprints, diffs and
// golden tests. It never changes the meaning of physical objects.
func (s SchemaSnapshot) Normalize() SchemaSnapshot {
	out := s
	out.Tables = append([]Table(nil), s.Tables...)
	sort.Slice(out.Tables, func(i, j int) bool {
		return normalizeName(out.Tables[i].QualifiedName()) < normalizeName(out.Tables[j].QualifiedName())
	})
	for i := range out.Tables {
		t := &out.Tables[i]
		t.Columns = append([]Column(nil), t.Columns...)
		sort.Slice(t.Columns, func(i, j int) bool {
			if t.Columns[i].Ordinal != t.Columns[j].Ordinal && t.Columns[i].Ordinal != 0 && t.Columns[j].Ordinal != 0 {
				return t.Columns[i].Ordinal < t.Columns[j].Ordinal
			}
			return normalizeName(t.Columns[i].Name) < normalizeName(t.Columns[j].Name)
		})
		t.PrimaryKey = append([]string(nil), t.PrimaryKey...)
		t.UniqueConstraints = append([]UniqueConstraint(nil), t.UniqueConstraints...)
		sort.Slice(t.UniqueConstraints, func(i, j int) bool {
			ki := uniqueConstraintOrderKey(t.UniqueConstraints[i])
			kj := uniqueConstraintOrderKey(t.UniqueConstraints[j])
			if ki != kj {
				return ki < kj
			}
			return normalizeName(t.UniqueConstraints[i].Name) < normalizeName(t.UniqueConstraints[j].Name)
		})
		t.ForeignKeys = append([]ForeignKey(nil), t.ForeignKeys...)
		sort.Slice(t.ForeignKeys, func(i, j int) bool {
			ki := foreignKeyOrderKey(t.ForeignKeys[i])
			kj := foreignKeyOrderKey(t.ForeignKeys[j])
			if ki != kj {
				return ki < kj
			}
			return normalizeName(t.ForeignKeys[i].Name) < normalizeName(t.ForeignKeys[j].Name)
		})
		t.Indexes = append([]Index(nil), t.Indexes...)
		for j := range t.Indexes {
			t.Indexes[j].Keys = append([]IndexKey(nil), t.Indexes[j].Keys...)
			t.Indexes[j].IncludedColumns = append([]string(nil), t.Indexes[j].IncludedColumns...)
		}
		sort.Slice(t.Indexes, func(i, j int) bool { return normalizeName(t.Indexes[i].Name) < normalizeName(t.Indexes[j].Name) })
	}
	return out
}

func uniqueConstraintOrderKey(uc UniqueConstraint) string {
	parts := make([]string, len(uc.Columns))
	for i, column := range uc.Columns {
		parts[i] = normalizeName(column)
	}
	return strings.Join(parts, "\x1f")
}

func foreignKeyOrderKey(fk ForeignKey) string {
	parts := make([]string, 0, len(fk.Columns)+len(fk.ReferencedColumns)+3)
	for _, column := range fk.Columns {
		parts = append(parts, normalizeName(column))
	}
	parts = append(parts, "->")
	if normalizeName(fk.ReferencedSchema) != "" {
		parts = append(parts, normalizeName(fk.ReferencedSchema)+"."+normalizeName(fk.ReferencedTable))
	} else {
		parts = append(parts, normalizeName(fk.ReferencedTable))
	}
	parts = append(parts, "::")
	for _, column := range fk.ReferencedColumns {
		parts = append(parts, normalizeName(column))
	}
	return strings.Join(parts, "\x1f")
}

func normalizeName(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
