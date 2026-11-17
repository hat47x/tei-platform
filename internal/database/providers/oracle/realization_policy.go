package oracle

import (
	"fmt"

	"tei/internal/database/persistence"
	"tei/internal/database/physical"
	"tei/internal/environment"
)

// RealizationPolicy owns the offline Oracle Desired Physical realization rules
// that TEI already supported before a connected Oracle Provider exists.
type RealizationPolicy struct{}

func (RealizationPolicy) Provider() environment.DatabaseProvider { return environment.ProviderOracle }

func (RealizationPolicy) PhysicalType(storage persistence.StorageType) (string, error) {
	switch storage {
	case persistence.StorageInt64:
		return "NUMBER(19)", nil
	case persistence.StorageVarchar:
		return "VARCHAR2(255)", nil
	case persistence.StorageText:
		return "CLOB", nil
	case persistence.StorageDateTimeTZ:
		return "TIMESTAMP WITH TIME ZONE", nil
	case persistence.StorageDecimal:
		return "NUMBER", nil
	case persistence.StorageJSON:
		return "CLOB", nil
	default:
		return "", fmt.Errorf("unsupported storage type %q for provider %q", storage, environment.ProviderOracle)
	}
}

// DecimalType renders a declared decimal width as Oracle's NUMBER(p,s).
func (RealizationPolicy) DecimalType(precision, scale int) (string, error) {
	if precision < 1 || scale < 0 || scale > precision {
		return "", fmt.Errorf("decimal width (%d,%d) is not a valid Oracle NUMBER", precision, scale)
	}
	return fmt.Sprintf("NUMBER(%d,%d)", precision, scale), nil
}

func (RealizationPolicy) IndexDefaults() persistence.IndexDefaults {
	return persistence.IndexDefaults{}
}

func (RealizationPolicy) NormalizeIndexKey(key physical.IndexKey, _ string) physical.IndexKey {
	return key
}

var _ persistence.PhysicalRealizationPolicy = RealizationPolicy{}
