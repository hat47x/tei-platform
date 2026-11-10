package validation

import (
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/environment"
	"tei/internal/ir"
	"tei/internal/query"
	"tei/internal/semantic"
)

// The last standalone codes in this package: a period whose business date is
// not an attribute of its data (#4032).
func TestPeriodBusinessDateCodeIsProduced(t *testing.T) {
	world := ir.World{
		Semantics: []semantic.Namespace{
			{
				ID: "sales",
				Data: []semantic.Data{
					{
						ID: "sales.order",
						Attributes: []semantic.Attribute{
							{ID: "sales.order.order_date", Symbol: "orderDate", TypeRef: "std.date"},
						},
					},
				},
				Periods: []semantic.Period{
					{
						ID:           "sales.order.period",
						Data:         "sales.order",
						BusinessDate: core.CanonicalID("sales.order.missing_date"),
						States:       []string{"open", "closing", "closed"},
					},
				},
			},
		},
	}
	assertHasDiagnostic(t, Validate(world), "PERIOD_UNKNOWN_BUSINESS_DATE")
}

// A hook that reads or uses a semantic element the application does not
// declare is reported, because the declaration would never run (#4032).
func TestInteractionHookSemanticNotUsedCodeIsProduced(t *testing.T) {
	world := hookWorld()
	world.Applications[0].SemanticUses = nil
	assertHasDiagnostic(t, validateApplicationPatterns(world), "APPLICATION_PATTERN_INTERACTION_HOOK_SEMANTIC_NOT_USED")
}

// A header-detail binding may only call a Query that carries a typed
// signature, because the binding maps inputs and result columns by name (#4032).
func TestHeaderDetailContractQuerySignatureRequiredCodeIsProduced(t *testing.T) {
	world := headerDetailContractQueryBindingWorld()
	// Drop the signature: no inputs, no result and no cardinality.
	world.Databases[0].Contracts[0].Queries[0] = database.QueryContract{Name: "find_by_order"}
	assertHasDiagnostic(t, ValidateHeaderDetailContractQueryBindings(world), "HEADER_DETAIL_CONTRACT_QUERY_SIGNATURE_REQUIRED")
}

// A query whose datasource is not bound in any environment is a warning, not
// an error, but it still has to be reported (#4032).
func TestUnboundDatasourceCodeIsProduced(t *testing.T) {
	world := ir.World{
		Applications: []application.Application{
			{
				ID: "order_entry",
				Databases: []application.DatabaseRequirement{
					{Alias: "sales", TargetDatabase: "sales_core", Access: database.AccessRead},
				},
			},
		},
		Queries: []query.EmbeddedSQL{
			{
				ID:          "order_entry.find",
				Application: "order_entry",
				Datasource:  core.LocalName("sales"),
				Dialect:     environment.ProviderOracle,
			},
		},
	}
	found := false
	for _, diagnostic := range Validate(world) {
		if diagnostic.Code == "UNBOUND_DATASOURCE" {
			found = true
			if diagnostic.Severity != Warning {
				t.Fatalf("UNBOUND_DATASOURCE is a warning about an unbound datasource, got severity %q", diagnostic.Severity)
			}
		}
	}
	if !found {
		t.Fatal("a datasource that no environment binds was not reported")
	}
}

// A reference Pattern that names one Query, while the Contract declares the
// same Query name twice, cannot be resolved to one signature (#4032).
func TestAmbiguousReferenceContractQueryCodeIsProduced(t *testing.T) {
	world := referenceContractQueryWorld(validReferencePattern())
	query := world.Databases[0].Contracts[0].Queries[0]
	world.Databases[0].Contracts[0].Queries = append(world.Databases[0].Contracts[0].Queries, query)
	assertHasDiagnostic(t, ValidateReferenceContractQueryInvocations(world), "AMBIGUOUS_REFERENCE_CONTRACT_QUERY")
}
