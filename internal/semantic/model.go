package semantic

import "tei/internal/core"

// Namespace is a coarse-grained semantic dependency unit. It is independent
// from repository paths and may be consumed by Applications and Databases.
type Namespace struct {
	ID           core.CanonicalID `json:"id"`
	Types        []Type           `json:"types,omitempty"`
	Data         []Data           `json:"data,omitempty"`
	Capabilities []Capability     `json:"capabilities,omitempty"`
	Validations  []Validation     `json:"validations,omitempty"`
}

type TypeKind string

const (
	TypePrimitive TypeKind = "primitive"
	TypeAlias     TypeKind = "alias"
)

func (k TypeKind) Valid() bool { return k == TypePrimitive || k == TypeAlias }

// Type is a semantic type identity. Primitive types are normally supplied by
// the TEI standard library; workspace definitions generally introduce aliases
// that preserve domain meaning while reusing a lower-level representation.
type Type struct {
	ID         core.CanonicalID `json:"id"`
	Symbol     string           `json:"symbol"`
	Kind       TypeKind         `json:"kind"`
	Underlying core.CanonicalID `json:"underlying,omitempty"`
}

type Data struct {
	ID         core.CanonicalID `json:"id"`
	Symbol     string           `json:"symbol"`
	Attributes []Attribute      `json:"attributes,omitempty"`
	Relations  []Relation       `json:"relations,omitempty"`
}

type Attribute struct {
	ID      core.CanonicalID `json:"id"`
	Symbol  string           `json:"symbol"`
	TypeRef core.CanonicalID `json:"type"`
}

type Cardinality string

const (
	CardinalityOne  Cardinality = "one"
	CardinalityMany Cardinality = "many"
)

func (c Cardinality) Valid() bool { return c == CardinalityOne || c == CardinalityMany }

type Relation struct {
	ID          core.CanonicalID `json:"id"`
	Symbol      string           `json:"symbol"`
	Target      core.CanonicalID `json:"target"`
	Cardinality Cardinality      `json:"cardinality"`
}

// ValidationKindは、業務上の不変条件として実Referenceで確認できた
// 最小のrule種別だけを表す。汎用式言語やrule engineを先回りして導入しない。
type ValidationKind string

const (
	ValidationPositive       ValidationKind = "positive"
	ValidationRelationUnique ValidationKind = "relation-unique"
)

func (k ValidationKind) Valid() bool {
	return k == ValidationPositive || k == ValidationRelationUnique
}

// Validationは、Semantic上の業務ruleを機械可読な形で保持する。
// positiveはAttributeだけを、relation-uniqueはRelationとそのtarget側Attributeを参照する。
// 評価結果のsatisfied / not-satisfiedと、評価自体を完了できなかったerrorは実行境界で分離する。
type Validation struct {
	ID        core.CanonicalID `json:"id"`
	Symbol    string           `json:"symbol"`
	Kind      ValidationKind   `json:"kind"`
	Attribute core.CanonicalID `json:"attribute"`
	Relation  core.CanonicalID `json:"relation,omitempty"`
}

type Parameter struct {
	Name core.LocalName   `json:"name"`
	Type core.CanonicalID `json:"type"`
}

// Outcomeは、Capabilityを正常に実行した結果として得られる業務上の結末を表す。
// 実行そのものを完了できなかった状態はOutcomeへ変換せず、Invocation境界のerrorとして扱う。
type Outcome struct {
	Name    core.LocalName   `json:"name"`
	Payload core.CanonicalID `json:"payload"`
}

type Capability struct {
	ID       core.CanonicalID `json:"id"`
	Symbol   string           `json:"symbol"`
	Inputs   []Parameter      `json:"inputs,omitempty"`
	Output   core.CanonicalID `json:"output,omitempty"`
	Outcomes []Outcome        `json:"outcomes,omitempty"`
}
