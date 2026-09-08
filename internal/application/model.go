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
	PatternBrowse       PatternKind = "browse"
	PatternRecord       PatternKind = "record"
	PatternReference    PatternKind = "reference"
	PatternCommand      PatternKind = "command"
	PatternHeaderDetail PatternKind = "header-detail"
)

func (k PatternKind) Valid() bool {
	return k == PatternBrowse ||
		k == PatternRecord ||
		k == PatternReference ||
		k == PatternCommand ||
		k == PatternHeaderDetail
}

// ContractQueryInputBindingは、reference PatternでRelation target keyの値を
// Database Contract Queryのどのinputへ渡すかを明示する。
// 型やinput名の一致から対応を推測せず、Semantic attributeのidentityをそのまま保持する。
type ContractQueryInputBinding struct {
	TargetAttribute core.CanonicalID `json:"targetAttribute"`
	Input           core.LocalName   `json:"input"`
}

// ContractQueryInvocationは、一つのreference Patternが利用するDatabase Contract Queryを表す。
// Contract QueryのsignatureやDatabase固有の実現方法はそれぞれの所有層に残し、
// ここではApplicationが選んだContract / QueryとRelation keyからのinput対応だけを保持する。
type ContractQueryInvocation struct {
	Contract core.CanonicalID            `json:"contract"`
	Query    core.LocalName              `json:"query"`
	Inputs   []ContractQueryInputBinding `json:"inputs,omitempty"`
}

// PatternにはApplication上の意図だけを保持する。
// Semantic定義、Query本文、Capability契約、物理パス、Target固有の表現は既存の層に残し、ここではidentityで参照する。
type Pattern struct {
	ID            core.CanonicalID         `json:"id"`
	Kind          PatternKind              `json:"kind"`
	Data          core.CanonicalID         `json:"data,omitempty"`
	Relation      core.CanonicalID         `json:"relation,omitempty"`
	Query         core.CanonicalID         `json:"query,omitempty"`
	Capability    core.CanonicalID         `json:"capability,omitempty"`
	Datasource    core.LocalName           `json:"datasource,omitempty"`
	ContractQuery *ContractQueryInvocation `json:"contractQuery,omitempty"`
}

type Application struct {
	ID           core.CanonicalID      `json:"id"`
	SemanticUses []core.CanonicalID    `json:"semanticUses,omitempty"`
	Validations  []core.CanonicalID    `json:"validations,omitempty"`
	Databases    []DatabaseRequirement `json:"databases,omitempty"`
	Patterns     []Pattern             `json:"patterns,omitempty"`
}

// HasDatabaseAliasは、Application-localなdatasource alias宣言が一件以上存在するかだけを返す。
// Validation等のdeclaration existence確認では返却値を持つlookupを使わず、このAPIを利用する。
func (a Application) HasDatabaseAlias(alias core.LocalName) bool {
	for _, d := range a.Databases {
		if d.Alias == alias {
			return true
		}
	}
	return false
}

// DatabaseByAliasは既存caller移行中のcompatibility lookupである。
// uniqueなら宣言内容を返す。duplicateでは宣言の存在だけをtrueで返し、
// 一件目の内容をauthorityとして漏らさないためzero valueを返す。
// declaration existence確認ではHasDatabaseAlias、実行・authority境界ではDatabaseByAliasExactを利用する。
func (a Application) DatabaseByAlias(alias core.LocalName) (DatabaseRequirement, bool) {
	var match DatabaseRequirement
	found := false
	for _, d := range a.Databases {
		if d.Alias != alias {
			continue
		}
		if found {
			return DatabaseRequirement{}, true
		}
		match = d
		found = true
	}
	return match, found
}

// DatabaseByAliasExactは、Application-localなdatasource aliasがちょうど一件だけ定義されている場合に解決する。
// 実行・authority境界では曖昧なCanonical宣言から一件を黙って選ばないため、このresolverを利用する。
func (a Application) DatabaseByAliasExact(alias core.LocalName) (DatabaseRequirement, bool) {
	var match DatabaseRequirement
	found := false
	for _, d := range a.Databases {
		if d.Alias != alias {
			continue
		}
		if found {
			return DatabaseRequirement{}, false
		}
		match = d
		found = true
	}
	return match, found
}
