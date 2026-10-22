// Package ddlentry is the entry point that asks a database Provider to render
// the DDL for the persistence mappings of one database. It adds no SQL grammar
// of its own: statement rendering stays with the Provider (ADR-0015), and what
// the Provider cannot render is reported per table instead of being filled in.
package ddlentry

import (
	"fmt"
	"sort"

	"tei/internal/core"
	"tei/internal/database/migration"
	"tei/internal/database/persistence"
	"tei/internal/database/physical"
	"tei/internal/database/realization"
	"tei/internal/database/schemadiff"
	"tei/internal/environment"
	"tei/internal/ir"
)

const Schema = "tei.reference.ddl-entry/v0"

type Table struct {
	Name       string   `json:"name"`
	Statements []string `json:"statements"`
}

type Unresolved struct {
	Table  string `json:"table"`
	Reason string `json:"reason"`
}

type Result struct {
	Schema   string `json:"schema"`
	Database string `json:"database"`
	Provider string `json:"provider"`
	// Tables lists the tables the Provider could render, ordered so that a
	// table comes after the tables its foreign keys reference; ties stay in
	// name order. Running the statements in this order creates every table.
	Tables []Table `json:"tables"`
	// Unresolved lists tables the Provider refused or could not realize.
	Unresolved []Unresolved `json:"unresolved"`
	// Script is the whole database rendered as one diff, in an order that can be
	// executed (every table before the foreign keys that reference it). It is
	// empty when any table is unresolved: a partial script is never offered.
	Script []string `json:"script"`
	// Note states what the statements are: a fresh-create script derived from
	// the declared mappings, not a migration for an existing database.
	Note string `json:"note"`
}

// Render realizes the mappings of one database and renders each table on its
// own, so a table the Provider refuses does not hide the others. Each table's
// foreign keys are rendered with that table, so tables are emitted after the
// tables they reference; a table on a foreign-key cycle, or one that references
// a table that could not be rendered, is reported instead of emitted in an
// order that would fail.
func Render(world ir.World, databaseID string, provider environment.DatabaseProvider) (Result, error) {
	out := Result{
		Schema: Schema, Database: databaseID, Provider: string(provider),
		Tables: []Table{}, Unresolved: []Unresolved{}, Script: []string{},
		Note: "宣言された物理対応から導いた新規作成用のDDLです。既存のデータベースへの移行文ではありません。",
	}
	var mappings []persistence.Mapping
	for _, m := range world.Persistence {
		if string(m.Database) == databaseID {
			mappings = append(mappings, m)
		}
	}
	if len(mappings) == 0 {
		return out, fmt.Errorf("no persistence mapping declares database %q", databaseID)
	}
	desired, err := realization.RealizeDesired(mappings, databaseID, provider)
	if err != nil {
		return out, fmt.Errorf("realize desired schema: %w", err)
	}
	tables := append([]physical.Table(nil), desired.Tables...)
	sort.Slice(tables, func(i, j int) bool { return tables[i].Name < tables[j].Name })
	ordered, cyclic := orderByForeignKeys(tables)
	for _, t := range cyclic {
		out.Unresolved = append(out.Unresolved, Unresolved{Table: t.Name, Reason: "外部キーが表の間で循環しているか、循環する表を参照しているため、表ごとの新規作成順を決められません"})
	}
	failed := map[string]string{}
	for _, t := range cyclic {
		failed[tableKey(t.Schema, t.Name)] = t.Name
	}
	for _, t := range ordered {
		if ref, ok := failedReference(t, failed); ok {
			out.Unresolved = append(out.Unresolved, Unresolved{Table: t.Name, Reason: fmt.Sprintf("外部キーの参照先の表 %s を出力できないため、この表も出力しません", ref)})
			failed[tableKey(t.Schema, t.Name)] = t.Name
			continue
		}
		single := physical.SchemaSnapshot{Database: core.CanonicalID(databaseID), Tables: []physical.Table{t}}
		empty := physical.SchemaSnapshot{Database: core.CanonicalID(databaseID)}
		diff, err := schemadiff.Compare(single, empty)
		if err != nil {
			out.Unresolved = append(out.Unresolved, Unresolved{Table: t.Name, Reason: err.Error()})
			failed[tableKey(t.Schema, t.Name)] = t.Name
			continue
		}
		statements, err := migration.Render(provider, diff)
		if err != nil {
			out.Unresolved = append(out.Unresolved, Unresolved{Table: t.Name, Reason: err.Error()})
			failed[tableKey(t.Schema, t.Name)] = t.Name
			continue
		}
		out.Tables = append(out.Tables, Table{Name: t.Name, Statements: statements})
	}
	if len(out.Unresolved) == 0 {
		diff, err := schemadiff.Compare(desired, physical.SchemaSnapshot{Database: core.CanonicalID(databaseID)})
		if err != nil {
			return out, fmt.Errorf("compare whole database: %w", err)
		}
		script, err := migration.Render(provider, diff)
		if err != nil {
			return out, fmt.Errorf("render whole database: %w", err)
		}
		out.Script = script
	}
	return out, nil
}

func tableKey(schema, name string) string { return schema + "\x00" + name }

// references returns the keys of the other tables in byKey that t's foreign
// keys point to. A reference outside the rendered tables, or to t itself, adds
// no ordering constraint.
func references(t physical.Table, byKey map[string]bool) []string {
	self := tableKey(t.Schema, t.Name)
	var refs []string
	for _, fk := range t.ForeignKeys {
		k := tableKey(fk.ReferencedSchema, fk.ReferencedTable)
		if k != self && byKey[k] {
			refs = append(refs, k)
		}
	}
	return refs
}

// orderByForeignKeys returns the tables that can be created one by one, each
// after the tables it references, keeping name order among tables that are
// ready at the same time. Tables on or behind a cycle are returned separately.
func orderByForeignKeys(tables []physical.Table) (ordered, cyclic []physical.Table) {
	byKey := map[string]bool{}
	for _, t := range tables {
		byKey[tableKey(t.Schema, t.Name)] = true
	}
	done := map[string]bool{}
	remaining := tables
	for len(remaining) > 0 {
		var next []physical.Table
		progressed := false
		for _, t := range remaining {
			ready := true
			for _, k := range references(t, byKey) {
				if !done[k] {
					ready = false
					break
				}
			}
			if ready && !progressed {
				ordered = append(ordered, t)
				done[tableKey(t.Schema, t.Name)] = true
				progressed = true
				continue
			}
			next = append(next, t)
		}
		if !progressed {
			return ordered, next
		}
		remaining = next
	}
	return ordered, nil
}

func failedReference(t physical.Table, failed map[string]string) (string, bool) {
	for _, fk := range t.ForeignKeys {
		if name, ok := failed[tableKey(fk.ReferencedSchema, fk.ReferencedTable)]; ok && name != t.Name {
			return name, true
		}
	}
	return "", false
}
