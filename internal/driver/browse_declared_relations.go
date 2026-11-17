package driver

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/database/persistence"
)

// DeclaredRelationsStatus says whether the Workspace declares which physical relation a
// browse Pattern's Data lives in.
type DeclaredRelationsStatus string

const (
	// DeclaredRelationsResolved: exactly one Persistence Mapping binds the Pattern's Data to
	// the Datasource's target Database.
	DeclaredRelationsResolved DeclaredRelationsStatus = "resolved"
	// DeclaredRelationsNone: no Persistence Mapping binds the Data in that Database.
	DeclaredRelationsNone DeclaredRelationsStatus = "none"
	// DeclaredRelationsAmbiguous: more than one Persistence Mapping does. None is chosen.
	DeclaredRelationsAmbiguous DeclaredRelationsStatus = "ambiguous"
)

// DeclaredBrowseRelations is what the Workspace DECLARES about where a browse Pattern's Data
// is stored. It is read from the Persistence Mapping only. The native SQL text is never
// parsed for table names, so this is a declaration to compare an observation against, never
// an observation itself.
type DeclaredBrowseRelations struct {
	Data       core.CanonicalID        `json:"data"`
	Database   core.CanonicalID        `json:"database"`
	Status     DeclaredRelationsStatus `json:"status"`
	Tables     []string                `json:"tables,omitempty"`
	PrimaryKey []core.CanonicalID      `json:"primaryKey,omitempty"`
	// Columns are the declared physical columns of the declared relation, by Semantic
	// attribute. They are what a result column's provenance is compared against.
	Columns []DeclaredColumn `json:"columns,omitempty"`
	Note    string           `json:"note,omitempty"`
}

// DeclaredColumn is one Persistence Mapping column declaration.
type DeclaredColumn struct {
	Attribute core.CanonicalID `json:"attribute"`
	Column    string           `json:"column"`
}

// DeclaredBrowseRelations resolves the declared storage of a browse Pattern's Data in the
// Datasource's target Database. A Pattern that has no Data or Datasource is an error: only a
// resolvable browse Pattern has anything to declare.
func (r Resolver) DeclaredBrowseRelations(applicationID, patternID core.CanonicalID) (DeclaredBrowseRelations, error) {
	resolved, err := r.ResolvePattern(applicationID, patternID)
	if err != nil {
		return DeclaredBrowseRelations{}, err
	}
	if resolved.Data == nil || resolved.Datasource == nil {
		return DeclaredBrowseRelations{}, fmt.Errorf("Pattern %sはData・Datasourceを持つbrowse Patternではありません", patternID)
	}
	out := DeclaredBrowseRelations{Data: resolved.Data.ID, Database: resolved.Datasource.TargetDatabase}
	var matches []persistence.Mapping
	for _, mapping := range r.world.Persistence {
		if mapping.SemanticData == out.Data && mapping.Database == out.Database {
			matches = append(matches, mapping)
		}
	}
	switch len(matches) {
	case 0:
		out.Status = DeclaredRelationsNone
		out.Note = fmt.Sprintf("Persistence MappingがData %sをDatabase %sへ束縛していません", out.Data, out.Database)
	case 1:
		out.Status = DeclaredRelationsResolved
		out.Tables = []string{matches[0].Table}
		out.PrimaryKey = append([]core.CanonicalID(nil), matches[0].PrimaryKey...)
		for _, column := range matches[0].Columns {
			out.Columns = append(out.Columns, DeclaredColumn{Attribute: column.Attribute, Column: column.Column})
		}
	default:
		out.Status = DeclaredRelationsAmbiguous
		out.Note = fmt.Sprintf("%d件のPersistence MappingがData %sをDatabase %sへ束縛しています", len(matches), out.Data, out.Database)
	}
	return out, nil
}
