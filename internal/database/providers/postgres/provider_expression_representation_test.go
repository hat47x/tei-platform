package postgres

import (
	"testing"

	"tei/internal/database/physical"
)

func TestPostgresExpressionCarriesProviderNormalizedProvenance(t *testing.T) {
	expression := postgresExpression(" lower(name) ")
	if expression.Provider != "postgres" || expression.Dialect != "postgresql" || expression.Source != "lower(name)" {
		t.Fatalf("expression=%#v", expression)
	}
	if expression.Representation != physical.ProviderExpressionProviderNormalized {
		t.Fatalf("representation=%q", expression.Representation)
	}
	if expression.Normalization == nil {
		t.Fatal("normalization provenance is nil")
	}
	if expression.Normalization.Authority != postgresExpressionNormalizationAuthority {
		t.Fatalf("authority=%q", expression.Normalization.Authority)
	}
	if expression.Normalization.ProviderVersion != "" {
		t.Fatalf("provider version must remain unknown until the catalog boundary supplies it: %q", expression.Normalization.ProviderVersion)
	}
}
