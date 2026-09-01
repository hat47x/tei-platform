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

// PatternKindは、Applicationが所有する意図の種類を表す。
// v1では実Referenceで意味を確認できたKindから順に有効化し、未検証の分類は先回りして固定しない。
type PatternKind string

const (
	PatternBrowse    PatternKind = "browse"
	PatternRecord    PatternKind = "record"
	PatternReference PatternKind = "reference"
)

func (k PatternKind) Valid() bool {
	return k == PatternBrowse || k == PatternRecord || k == PatternReference
}

// PatternにはApplication上の意図だけを保持する。
// Semantic定義、Query本文、物理パス、Target固有の表現は既存の層に残し、ここではidentityで参照する。
type Pattern struct {
	ID         core.CanonicalID `json:"id"`
	Kind       PatternKind      `json:"kind"`
	Data       core.CanonicalID `json:"data,omitempty"`
	Relation   core.CanonicalID `json:"relation,omitempty"`
	Query      core.CanonicalID `json:"query,omitempty"`
	Datasource core.LocalName   `json:"datasource,omitempty"`
}

type Application struct {
	ID           core.CanonicalID      `json:"id"`
	SemanticUses []core.CanonicalID    `json:"semanticUses,omitempty"`
	Databases    []DatabaseRequirement `json:"databases,omitempty"`
	Patterns     []Pattern             `json:"patterns,omitempty"`
}

func (a Application) DatabaseByAlias(alias core.LocalName) (DatabaseRequirement, bool) {
	for _, d := range a.Databases {
		if d.Alias == alias {
			return d, true
		}
	}
	return DatabaseRequirement{}, false
}
