package semantic

import "tei/internal/core"

// Namespace is a coarse-grained semantic dependency unit. It is independent
// from repository paths and may be consumed by Applications and Databases.
type Namespace struct {
	ID           core.CanonicalID `json:"id"`
	Types        []Type           `json:"types,omitempty"`
	Data         []Data           `json:"data,omitempty"`
	Capabilities []Capability     `json:"capabilities,omitempty"`
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

type Parameter struct {
	Name core.LocalName   `json:"name"`
	Type core.CanonicalID `json:"type"`
}

type Capability struct {
	ID     core.CanonicalID `json:"id"`
	Symbol string           `json:"symbol"`
	Inputs []Parameter      `json:"inputs,omitempty"`
	Output core.CanonicalID `json:"output,omitempty"`
}
