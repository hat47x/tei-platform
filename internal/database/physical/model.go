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
	Schema      string       `json:"schema,omitempty"`
	Name        string       `json:"name"`
	Kind        TableKind    `json:"kind,omitempty"`
	Columns     []Column     `json:"columns,omitempty"`
	PrimaryKey  []string     `json:"primaryKey,omitempty"`
	ForeignKeys []ForeignKey `json:"foreignKeys,omitempty"`
	Indexes     []Index      `json:"indexes,omitempty"`
}

type Column struct {
	Name         string `json:"name"`
	PhysicalType string `json:"physicalType,omitempty"`
	Nullable     bool   `json:"nullable"`
	Default      string `json:"default,omitempty"`
	Ordinal      int    `json:"ordinal,omitempty"`
}

type ForeignKey struct {
	Name              string   `json:"name,omitempty"`
	Columns           []string `json:"columns"`
	ReferencedSchema  string   `json:"referencedSchema,omitempty"`
	ReferencedTable   string   `json:"referencedTable"`
	ReferencedColumns []string `json:"referencedColumns"`
}

type Index struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns,omitempty"`
	Unique  bool     `json:"unique,omitempty"`
	Method  string   `json:"method,omitempty"`
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
		sort.Slice(t.Indexes, func(i, j int) bool { return t.Indexes[i].Name < t.Indexes[j].Name })
	}
	return out
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
