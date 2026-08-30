package application

import (
	"tei/internal/core"
	"tei/internal/database"
)

type DatabaseRequirement struct {
	Alias             core.LocalName                 `json:"alias"`
	TargetDatabase    core.CanonicalID               `json:"target"`
	Access            database.Access                `json:"access"`
	RequiredContracts []database.ContractRequirement `json:"requires,omitempty"`
}

type Application struct {
	ID           core.CanonicalID      `json:"id"`
	SemanticUses []core.CanonicalID    `json:"semanticUses,omitempty"`
	Databases    []DatabaseRequirement `json:"databases,omitempty"`
}

func (a Application) DatabaseByAlias(alias core.LocalName) (DatabaseRequirement, bool) {
	for _, d := range a.Databases {
		if d.Alias == alias {
			return d, true
		}
	}
	return DatabaseRequirement{}, false
}
