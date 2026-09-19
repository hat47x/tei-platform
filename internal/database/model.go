package database

import "tei/internal/core"

type Access string

const (
	AccessRead      Access = "read"
	AccessWrite     Access = "write"
	AccessReadWrite Access = "read-write"
)

func (a Access) Valid() bool { return a == AccessRead || a == AccessWrite || a == AccessReadWrite }

type Ownership string

const (
	OwnershipManagedHere      Ownership = "managed-here"
	OwnershipExternalObserved Ownership = "external-observed"
)

func (o Ownership) Valid() bool { return o == OwnershipManagedHere || o == OwnershipExternalObserved }

type Database struct {
	ID           core.CanonicalID   `json:"id"`
	SemanticUses []core.CanonicalID `json:"semanticUses,omitempty"`
	Kind         string             `json:"kind"`
	Ownership    Ownership          `json:"ownership"`
	Contracts    []Contract         `json:"contracts,omitempty"`
}

type ContractVersion struct {
	Major int `json:"major"`
	Minor int `json:"minor"`
}
type Contract struct {
	ID      core.CanonicalID   `json:"id"`
	Version ContractVersion    `json:"version"`
	Reads   []core.CanonicalID `json:"reads,omitempty"`
	Writes  []core.CanonicalID `json:"writes,omitempty"`
	Queries []QueryContract    `json:"queries,omitempty"`
}

// QueryInputは、Database Contract Queryが受け取る一つの入力を表す。
// Query名や物理parameter名から意味を推測せず、入力名とSemantic型identityを明示的に保持する。
type QueryInput struct {
	Name core.LocalName   `json:"name"`
	Type core.CanonicalID `json:"type"`
}

// QueryResultCardinalityは、Database Contract Queryが正常に完了したときに返し得る
// Semantic Data instance数の契約を表す。Execution Errorとは別の意味として扱う。
type QueryResultCardinality string

const (
	QueryResultOne       QueryResultCardinality = "one"
	QueryResultZeroOrOne QueryResultCardinality = "zero-or-one"
	QueryResultMany      QueryResultCardinality = "many"
)

func (c QueryResultCardinality) Valid() bool {
	return c == QueryResultOne || c == QueryResultZeroOrOne || c == QueryResultMany
}

// QueryContractは、Databaseが外部へ提供するQueryの意味契約を表す。
// Inputs / Result / ResultCardinalityはDatabase固有SQLやAPIの実現方式を含めず、
// 「何を受け、どのSemantic Dataをどの件数契約で返すか」だけを保持する。
// signature未定義の既存Contractはomitemptyにより従来のCanonical JSONを維持する。
type QueryContract struct {
	Name              string                 `json:"name"`
	Inputs            []QueryInput           `json:"inputs,omitempty"`
	Result            core.CanonicalID       `json:"result,omitempty"`
	ResultCardinality QueryResultCardinality `json:"resultCardinality,omitempty"`
}
type ContractRequirement struct {
	ContractID      core.CanonicalID `json:"contract"`
	CompatibleMajor int              `json:"compatibleMajor"`
}
