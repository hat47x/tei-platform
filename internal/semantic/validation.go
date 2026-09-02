package semantic

import "tei/internal/core"

// ValidationKindは、実Referenceで意味を確認した業務Validationの種類だけを表す。
// v1では汎用式言語を先に固定せず、必要になった規則から型付きで追加する。
type ValidationKind string

const (
	ValidationAttributePositive       ValidationKind = "attribute-positive"
	ValidationRelationAttributeUnique ValidationKind = "relation-attribute-unique"
)

func (k ValidationKind) Valid() bool {
	return k == ValidationAttributePositive || k == ValidationRelationAttributeUnique
}

// Validationは、TargetやUIから独立した業務上の不変条件を表す。
// 評価時の値やエンジン固有の式は保持せず、既存Semantic assetをCanonical IDで参照する。
type Validation struct {
	ID        core.CanonicalID `json:"id"`
	Kind      ValidationKind   `json:"kind"`
	Attribute core.CanonicalID `json:"attribute"`
	Relation  core.CanonicalID `json:"relation,omitempty"`
}
