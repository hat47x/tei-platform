package realization

import (
	"fmt"

	"tei/internal/database/persistence"
	"tei/internal/database/physical"
	"tei/internal/database/providers/oracle"
	"tei/internal/database/providers/postgres"
	"tei/internal/environment"
)

// PolicyFor assembles one built-in offline realization policy. Provider
// selection belongs to composition, not to Persistence Mapping semantics.
func PolicyFor(provider environment.DatabaseProvider) (persistence.PhysicalRealizationPolicy, error) {
	switch provider {
	case environment.ProviderPostgres:
		return postgres.RealizationPolicy{}, nil
	case environment.ProviderOracle:
		return oracle.RealizationPolicy{}, nil
	default:
		return nil, fmt.Errorf("physical realization policy is not implemented for provider %q", provider)
	}
}

// RealizeDesired is the composition-level convenience entry point used by
// callers that start from a provider ID rather than an already assembled
// PhysicalRealizationPolicy.
func RealizeDesired(mappings []persistence.Mapping, databaseID string, provider environment.DatabaseProvider) (physical.SchemaSnapshot, error) {
	policy, err := PolicyFor(provider)
	if err != nil {
		return physical.SchemaSnapshot{}, err
	}
	return persistence.RealizeDesired(mappings, databaseID, policy)
}
