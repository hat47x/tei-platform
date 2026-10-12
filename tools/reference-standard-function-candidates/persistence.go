package main

import (
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/database/persistence"
	"tei/internal/ir"
)

// DeclaredWriteMapping is only a match between three pieces of Canonical Source:
// a Semantic Data, a Database-owned Mapping and an Application's write requirement.
// It is not a runtime Provider, a write contract, or a grant of current authority.
type DeclaredWriteMapping struct {
	Mapping     persistence.Mapping
	Datasource  application.DatabaseRequirement
}

// MatchDeclaredWriteMapping never guesses an Application's target Database from
// its Capability name, a Semantic attribute name or physical table. A Data may
// legitimately have several mappings; without one unique writable Application
// target, no mapping is selected. A declaration missing any of the three edges
// cannot be treated as a ready-to-run write.
func MatchDeclaredWriteMapping(w ir.World, app application.Application, data core.CanonicalID) (*DeclaredWriteMapping, error) {
	var candidates []DeclaredWriteMapping
	for _, m := range w.Persistence {
		if m.SemanticData != data {
			continue
		}
		for _, requirement := range app.Databases {
			if requirement.TargetDatabase != m.Database {
				continue
			}
			if requirement.Access != database.AccessWrite && requirement.Access != database.AccessReadWrite {
				continue
			}
			candidates = append(candidates, DeclaredWriteMapping{
				Mapping:    m,
				Datasource: requirement,
			})
		}
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	if len(candidates) > 1 {
		return nil, fmt.Errorf("Data %s has %d writable persistence paths in Application %s; explicit target selection is required", data, len(candidates), app.ID)
	}

	selected := candidates[0]
	databaseCount := 0
	for _, d := range w.Databases {
		if d.ID == selected.Mapping.Database {
			databaseCount++
		}
	}
	if databaseCount != 1 {
		return nil, fmt.Errorf("Data %s has persistence Mapping %s referencing Database %s with %d canonical Database declarations", data, selected.Mapping.ID, selected.Mapping.Database, databaseCount)
	}
	if selected.Mapping.ID == "" || selected.Datasource.Alias == "" {
		return nil, fmt.Errorf("Data %s has an incomplete writable persistence declaration", data)
	}
	return &selected, nil
}
