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
	// Tables lists the tables the Provider could render, in name order.
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
// own, so a table the Provider refuses does not hide the others.
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
	desired, naming, err := realization.RealizeDesiredWithUnresolved(mappings, databaseID, provider)
	if err != nil {
		return out, fmt.Errorf("realize desired schema: %w", err)
	}
	// A table whose generated constraint names collide is reported with the
	// reason and not rendered; PostgreSQL would otherwise pick another name.
	namingReason := make(map[string]string, len(naming))
	for _, u := range naming {
		namingReason[u.Table] = u.Reason
	}
	tables := append([]physical.Table(nil), desired.Tables...)
	sort.Slice(tables, func(i, j int) bool { return tables[i].Name < tables[j].Name })
	for _, t := range tables {
		if reason, ok := namingReason[t.QualifiedName()]; ok {
			out.Unresolved = append(out.Unresolved, Unresolved{Table: t.Name, Reason: reason})
			continue
		}
		single := physical.SchemaSnapshot{Database: core.CanonicalID(databaseID), Tables: []physical.Table{t}}
		empty := physical.SchemaSnapshot{Database: core.CanonicalID(databaseID)}
		diff, err := schemadiff.Compare(single, empty)
		if err != nil {
			out.Unresolved = append(out.Unresolved, Unresolved{Table: t.Name, Reason: err.Error()})
			continue
		}
		statements, err := migration.Render(provider, diff)
		if err != nil {
			out.Unresolved = append(out.Unresolved, Unresolved{Table: t.Name, Reason: err.Error()})
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
