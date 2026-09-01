package persistence

import (
	"tei/internal/database/physical"
	"tei/internal/environment"
)

// IndexDefaults are provider-owned effective defaults used while projecting a
// Persistence Mapping into a Desired Physical Schema.
type IndexDefaults struct {
	Method              string
	UniqueNullTreatment physical.UniqueNullTreatment
}

// PhysicalRealizationPolicy is the offline provider policy required by Desired
// realization. It deliberately excludes connected runtime operations such as
// introspection and does not require context.Context or DatabaseInstance.
type PhysicalRealizationPolicy interface {
	Provider() environment.DatabaseProvider
	PhysicalType(StorageType) (string, error)
	IndexDefaults() IndexDefaults
	NormalizeIndexKey(physical.IndexKey, string) physical.IndexKey
}
