package migration

import (
	"fmt"

	"tei/internal/database/providers/postgres"
	"tei/internal/database/schemadiff"
	"tei/internal/environment"
)

// Render delegates SQL generation to the selected database provider. The
// common migration layer retains only the cross-provider safety boundary and
// provider selection; provider-specific SQL grammar lives with each provider.
func Render(provider environment.DatabaseProvider, diff schemadiff.Diff) ([]string, error) {
	if len(diff.Unresolved) > 0 {
		return nil, fmt.Errorf("migration SQL rendering requires a fully resolved schema diff: %d unresolved comparison(s)", len(diff.Unresolved))
	}
	switch provider {
	case environment.ProviderPostgres:
		return postgres.RenderMigrationSQL(diff)
	default:
		return nil, fmt.Errorf("migration SQL rendering is not implemented for provider %q", provider)
	}
}
