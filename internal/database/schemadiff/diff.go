package schemadiff

import (
	"fmt"
	"sort"
	"strings"

	"tei/internal/database/physical"
)

type ChangeKind string

const (
	CreateTable      ChangeKind = "create-table"
	DropTable        ChangeKind = "drop-table"
	AddColumn        ChangeKind = "add-column"
	DropColumn       ChangeKind = "drop-column"
	AlterColumn      ChangeKind = "alter-column"
	ChangePrimaryKey ChangeKind = "change-primary-key"
	AddIndex         ChangeKind = "add-index"
	DropIndex        ChangeKind = "drop-index"
)

type Change struct {
	Kind             ChangeKind       `json:"kind"`
	Table            string           `json:"table"`
	Column           *physical.Column `json:"column,omitempty"`
	BeforeColumn     *physical.Column `json:"beforeColumn,omitempty"`
	PrimaryKey       []string         `json:"primaryKey,omitempty"`
	BeforePrimaryKey []string         `json:"beforePrimaryKey,omitempty"`
	Index            *physical.Index  `json:"index,omitempty"`
}

type Diff struct {
	Database string   `json:"database"`
	Changes  []Change `json:"changes,omitempty"`
}

func Compare(desired, actual physical.SchemaSnapshot) (Diff, error) {
	if desired.Database != actual.Database {
		return Diff{}, fmt.Errorf("schema database mismatch: desired %s actual %s", desired.Database, actual.Database)
	}
	d := Diff{Database: string(desired.Database)}
	dTables := tableMap(desired)
	aTables := tableMap(actual)
	var names []string
	seen := map[string]struct{}{}
	for n := range dTables {
		seen[n] = struct{}{}
		names = append(names, n)
	}
	for n := range aTables {
		if _, ok := seen[n]; !ok {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	for _, n := range names {
		dt, dok := dTables[n]
		at, aok := aTables[n]
		switch {
		case dok && !aok:
			copy := dt
			d.Changes = append(d.Changes, Change{Kind: CreateTable, Table: n, Column: nil, PrimaryKey: copy.PrimaryKey})
			for _, c := range dt.Columns {
				cc := c
				d.Changes = append(d.Changes, Change{Kind: AddColumn, Table: n, Column: &cc})
			}
			for _, idx := range dt.Indexes {
				ii := idx
				d.Changes = append(d.Changes, Change{Kind: AddIndex, Table: n, Index: &ii})
			}
		case !dok && aok:
			d.Changes = append(d.Changes, Change{Kind: DropTable, Table: n})
		case dok && aok:
			d.Changes = append(d.Changes, diffTable(n, dt, at)...)
		}
	}
	return d, nil
}

func diffTable(name string, desired, actual physical.Table) []Change {
	var out []Change
	dc, ac := colMap(desired), colMap(actual)
	var names []string
	seen := map[string]struct{}{}
	for n := range dc {
		seen[n] = struct{}{}
		names = append(names, n)
	}
	for n := range ac {
		if _, ok := seen[n]; !ok {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	for _, n := range names {
		d, dok := dc[n]
		a, aok := ac[n]
		switch {
		case dok && !aok:
			cc := d
			out = append(out, Change{Kind: AddColumn, Table: name, Column: &cc})
		case !dok && aok:
			cc := a
			out = append(out, Change{Kind: DropColumn, Table: name, BeforeColumn: &cc})
		case dok && aok && !sameColumn(d, a):
			nd, od := d, a
			out = append(out, Change{Kind: AlterColumn, Table: name, Column: &nd, BeforeColumn: &od})
		}
	}
	if !sameStrings(desired.PrimaryKey, actual.PrimaryKey) {
		out = append(out, Change{Kind: ChangePrimaryKey, Table: name, PrimaryKey: append([]string(nil), desired.PrimaryKey...), BeforePrimaryKey: append([]string(nil), actual.PrimaryKey...)})
	}
	di, ai := idxMap(desired), idxMap(actual)
	var idxNames []string
	seenIdx := map[string]struct{}{}
	for n := range di {
		seenIdx[n] = struct{}{}
		idxNames = append(idxNames, n)
	}
	for n := range ai {
		if _, ok := seenIdx[n]; !ok {
			idxNames = append(idxNames, n)
		}
	}
	sort.Strings(idxNames)
	for _, n := range idxNames {
		d, dok := di[n]
		a, aok := ai[n]
		if dok && !aok {
			ii := d
			out = append(out, Change{Kind: AddIndex, Table: name, Index: &ii})
		} else if !dok && aok {
			ii := a
			out = append(out, Change{Kind: DropIndex, Table: name, Index: &ii})
		} else if dok && aok && !sameIndex(d, a) {
			old := a
			newi := d
			out = append(out, Change{Kind: DropIndex, Table: name, Index: &old}, Change{Kind: AddIndex, Table: name, Index: &newi})
		}
	}
	return out
}

func tableMap(s physical.SchemaSnapshot) map[string]physical.Table {
	m := map[string]physical.Table{}
	for _, t := range s.Tables {
		m[norm(t.QualifiedName())] = t
	}
	return m
}
func colMap(t physical.Table) map[string]physical.Column {
	m := map[string]physical.Column{}
	for _, c := range t.Columns {
		m[norm(c.Name)] = c
	}
	return m
}
func idxMap(t physical.Table) map[string]physical.Index {
	m := map[string]physical.Index{}
	for _, i := range t.Indexes {
		m[norm(i.Name)] = i
	}
	return m
}
func norm(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
func sameColumn(a, b physical.Column) bool {
	return norm(a.Name) == norm(b.Name) && norm(a.PhysicalType) == norm(b.PhysicalType) && a.Nullable == b.Nullable && strings.TrimSpace(a.Default) == strings.TrimSpace(b.Default)
}
func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if norm(a[i]) != norm(b[i]) {
			return false
		}
	}
	return true
}
func sameIndex(a, b physical.Index) bool {
	return a.Unique == b.Unique && norm(a.Method) == norm(b.Method) && sameStrings(a.Columns, b.Columns)
}
