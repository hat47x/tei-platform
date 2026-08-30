package release

import (
	"sort"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
)

// Definition is an immutable compatibility snapshot for one Application
// release. It deliberately copies the DatabaseRequirement set instead of
// referring to the current Application definition, so old releases remain
// understandable after the Application evolves.
type Definition struct {
	Application    core.CanonicalID                  `json:"application"`
	Version        string                            `json:"version"`
	SourceRevision string                            `json:"sourceRevision,omitempty"`
	Databases      []application.DatabaseRequirement `json:"databases,omitempty"`
}

func (r Definition) Normalize() Definition {
	out := r
	out.Databases = append([]application.DatabaseRequirement(nil), r.Databases...)
	for i := range out.Databases {
		out.Databases[i].RequiredContracts = append([]database.ContractRequirement(nil), r.Databases[i].RequiredContracts...)
		sort.Slice(out.Databases[i].RequiredContracts, func(a, b int) bool {
			return out.Databases[i].RequiredContracts[a].ContractID < out.Databases[i].RequiredContracts[b].ContractID
		})
	}
	sort.Slice(out.Databases, func(i, j int) bool { return out.Databases[i].Alias < out.Databases[j].Alias })
	return out
}

// AsApplication creates the compatibility-facing application snapshot. No
// current Application definition is consulted.
func (r Definition) AsApplication() application.Application {
	return application.Application{ID: r.Application, Databases: r.Normalize().Databases}
}
