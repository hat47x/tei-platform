package realization

import "tei/internal/core"

// HeaderDetailContractQueryInputBindingは、header-detailのsource instanceにある
// Semantic attributeの値を、collection取得用Database Contract Queryのどのinputへ渡すかを表す。
//
// reference PatternのContractQueryInputBindingとは向きが異なる。header-detailではRelation target側に
// 親identity attributeが存在することを前提にせず、source側の明示Semantic identityだけを保持する。
type HeaderDetailContractQueryInputBinding struct {
	SourceAttribute core.CanonicalID `json:"sourceAttribute"`
	Input           core.LocalName   `json:"input"`
}

// HeaderDetailContractQueryBindingは、一つのheader-detail Application Patternを
// Database-ownedなContract Queryへ接続するApplication固有のrealization境界である。
//
// Pattern自身はRelation-onlyのApplication intentとして維持し、logical Datasourceと取得Contractは
// このrealization側で明示する。Environment、Database instance、provider、SQL sourceは下位の
// composition / realization責務に残し、このmodelへ混ぜない。
type HeaderDetailContractQueryBinding struct {
	Application core.CanonicalID                        `json:"application"`
	Pattern     core.CanonicalID                        `json:"pattern"`
	Datasource  core.LocalName                          `json:"datasource"`
	Target      ContractQueryTarget                     `json:"target"`
	Inputs      []HeaderDetailContractQueryInputBinding `json:"inputs,omitempty"`
}

// Identityは、どのApplication PatternをrealizeするBindingかを示す安定identityを返す。
// DatasourceやContract Queryは選択内容なのでidentityへ含めず、同じPatternへ複数Bindingがある場合は
// 後続Validationで競合として扱えるようにする。
func (b HeaderDetailContractQueryBinding) Identity() string {
	return string(b.Application) + "/" + string(b.Pattern)
}

// Validは、Bindingをidentityとして扱うために必要な最小要素が明示されていることを確認する。
// 不足した値をPattern、Relation、file pathなどから補完しない。
func (b HeaderDetailContractQueryBinding) Valid() bool {
	return b.Application != "" && b.Pattern != "" && b.Datasource != "" && b.Target.Valid()
}
