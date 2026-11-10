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

// InteractionLookupは、項目の値を参照Patternから得る束縛を表す。
// 参照先のPatternと、そのRelation target側のattributeをidentityで指定する。
type InteractionLookup struct {
	Pattern         core.CanonicalID `json:"pattern"`
	TargetAttribute core.CanonicalID `json:"targetAttribute"`
}

// InteractionItemは、対話に現れる項目と、その値の由来を表す。
// 表示上の幅・部品・配置はここへ入れず、表示プロファイル側に残す。
type InteractionItem struct {
	Attribute core.CanonicalID   `json:"attribute"`
	Lookup    *InteractionLookup `json:"lookup,omitempty"`
}

// InteractionNavigationは、他のPatternからこのPatternへ入る遷移を表す。
type InteractionNavigation struct {
	From core.CanonicalID `json:"from"`
}

// Interactionは、Patternが持つ対話の宣言を表す。
// Pattern Kindを増やさず、項目の並び・参照入力・遷移・起動を保持する。
type Interaction struct {
	Items       []InteractionItem       `json:"items,omitempty"`
	Navigations []InteractionNavigation `json:"navigations,omitempty"`
	Commands    []InteractionCommand    `json:"commands,omitempty"`
	Hooks       []Hook                  `json:"hooks,omitempty"`
}

// InteractionCommandは、対話から起動できるCapabilityへの参照である。
// 効果や結末はCapability側に置き、ここでは起動の意図と入力の結びつけだけを持つ。
type InteractionCommand struct {
	Capability core.CanonicalID      `json:"capability"`
	Inputs     []CommandInputBinding `json:"inputs,omitempty"`
}

// Hookは、対話の前後に差し込む処理の宣言である。
// 実装はL4のPlugin・Providerに置き、ここでは位相・読み書き・失敗時の扱いだけを持つ。
type Hook struct {
	Phase string `json:"phase"`
	// Scope is the unit the hook decides for: one row, the record, or one item.
	// X-TEA's event script had no such declaration, so a hook that decided per row
	// and a hook that decided per record were told apart only by reading the script.
	// An empty scope means the pattern's own unit, which is the row for a browse or
	// record pattern and the header for a header-detail pattern.
	Scope    string             `json:"scope,omitempty"`
	Reads    []core.CanonicalID `json:"reads,omitempty"`
	Uses     []core.CanonicalID `json:"uses,omitempty"`
	Produces []core.CanonicalID `json:"produces,omitempty"`
	Failure  string             `json:"failure,omitempty"`
}

// CommandInputBindingは、Commandの入力を対話の項目へ結びつける。
type CommandInputBinding struct {
	Name      string           `json:"name"`
	Attribute core.CanonicalID `json:"attribute"`
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
	Interaction   *Interaction             `json:"interaction,omitempty"`
	Aggregates    []Aggregate              `json:"aggregates,omitempty"`
	Filters       []Filter                 `json:"filters,omitempty"`
	Sorts         []Sort                   `json:"sorts,omitempty"`
	PageSize      int                      `json:"pageSize,omitempty"`
	PageBreak     *PageBreak               `json:"pageBreak,omitempty"`
	Layout        *Layout                  `json:"layout,omitempty"`
	PrintTemplate string                   `json:"printTemplate,omitempty"`
}

// Layoutは、対話の項目の並びを宣言する。画面座標ではなく、意味の順序だけを持つ。
type Layout struct {
	Order []core.CanonicalID `json:"order,omitempty"`
}

// PageBreakは、header-detailの印刷で1ページに収める明細行数と、見出しの繰返しを宣言する。
type PageBreak struct {
	Rows         int  `json:"rows"`
	RepeatHeader bool `json:"repeatHeader,omitempty"`
}

// Sortは、一覧の並び順をQueryの契約として宣言する。
type Sort struct {
	Attribute core.CanonicalID `json:"attribute"`
	Direction string           `json:"direction"`
}

// Filterは、一覧の絞り込みに使う項目と演算子の宣言である。
// 演算子は画面の切替えではなく、Queryが受け付ける条件として宣言する。
type Filter struct {
	Attribute core.CanonicalID `json:"attribute"`
	Operators []string         `json:"operators,omitempty"`
	Parameter string           `json:"parameter,omitempty"`
}

// Aggregateは、header-detailの明細に対する集計の宣言である。
type Aggregate struct {
	Name      string           `json:"name"`
	Attribute core.CanonicalID `json:"attribute"`
	Function  string           `json:"function"`
}

type Application struct {
	ID              core.CanonicalID      `json:"id"`
	SemanticUses    []core.CanonicalID    `json:"semanticUses,omitempty"`
	Validations     []core.CanonicalID    `json:"validations,omitempty"`
	Databases       []DatabaseRequirement `json:"databases,omitempty"`
	Patterns        []Pattern             `json:"patterns,omitempty"`
	Packages        []Package             `json:"packages,omitempty"`
	PackageBindings []PackageBinding      `json:"packageBindings,omitempty"`
}

// PackageBindingは、Applicationが使うPackageの版を固定する。
type PackageBinding struct {
	ID      core.CanonicalID `json:"id"`
	Version string           `json:"version"`
}

// Packageは、既存Patternを束ねる派生の宣言である。Patternの定義を複製しない。
type Package struct {
	ID          core.CanonicalID     `json:"id"`
	Version     string               `json:"version"`
	Members     []PackageMember      `json:"members,omitempty"`
	Requires    []PackageRequirement `json:"requires,omitempty"`
	Projection  string               `json:"projection,omitempty"`
	Conformance string               `json:"conformance,omitempty"`
}

// PackageMemberは、Packageが参照するPatternと、その種別である。
type PackageMember struct {
	Kind    PatternKind      `json:"kind"`
	Pattern core.CanonicalID `json:"pattern"`
}

// PackageRequirementは、Packageが前提にするSemantic要素への参照である。
type PackageRequirement struct {
	Kind string           `json:"kind"`
	ID   core.CanonicalID `json:"id"`
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
