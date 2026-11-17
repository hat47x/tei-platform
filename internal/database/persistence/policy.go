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
	// DecimalType reports the physical type of a decimal column that declares
	// a width. A provider whose type cannot carry one must return an error
	// rather than quietly dropping the width. Providers that never see a width
	// still implement PhysicalType for the bare case.
	DecimalType(precision, scale int) (string, error)
}
