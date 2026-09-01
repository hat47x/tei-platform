package postgres

import (
	"fmt"
	"strings"

	"tei/internal/database/persistence"
	"tei/internal/database/physical"
	"tei/internal/environment"
)

// RealizationPolicy owns deterministic PostgreSQL Desired Physical realization
// rules that do not require a live database connection.
type RealizationPolicy struct{}

func (RealizationPolicy) Provider() environment.DatabaseProvider { return environment.ProviderPostgres }

func (RealizationPolicy) PhysicalType(storage persistence.StorageType) (string, error) {
	switch storage {
	case persistence.StorageInt64:
		return "pg_catalog.int8", nil
	case persistence.StorageVarchar:
		return "pg_catalog.varchar", nil
	case persistence.StorageText:
		return "pg_catalog.text", nil
	case persistence.StorageDateTimeTZ:
		return "pg_catalog.timestamptz", nil
	case persistence.StorageDecimal:
		return "pg_catalog.numeric", nil
	case persistence.StorageJSON:
		return "pg_catalog.jsonb", nil
	default:
		return "", fmt.Errorf("unsupported storage type %q for provider %q", storage, environment.ProviderPostgres)
	}
}

func (RealizationPolicy) IndexDefaults() persistence.IndexDefaults {
	return persistence.IndexDefaults{
		Method:              "btree",
		UniqueNullTreatment: physical.UniqueNullTreatmentDistinct,
	}
}

func (RealizationPolicy) NormalizeIndexKey(key physical.IndexKey, method string) physical.IndexKey {
	if !strings.EqualFold(strings.TrimSpace(method), "btree") {
		return key
	}

	desc := key.Order == physical.IndexOrderDesc
	if key.Order == physical.IndexOrderAsc {
		key.Order = physical.IndexOrderDefault
	}
	if !desc && key.NullsOrder == physical.IndexNullsOrderLast {
		key.NullsOrder = physical.IndexNullsOrderDefault
	} else if desc && key.NullsOrder == physical.IndexNullsOrderFirst {
		key.NullsOrder = physical.IndexNullsOrderDefault
	}
	return key
}

var _ persistence.PhysicalRealizationPolicy = RealizationPolicy{}
