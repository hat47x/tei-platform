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
type QueryContract struct {
	Name string `json:"name"`
}
type ContractRequirement struct {
	ContractID      core.CanonicalID `json:"contract"`
	CompatibleMajor int              `json:"compatibleMajor"`
}
