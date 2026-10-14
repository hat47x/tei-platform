package localcpureuse

// This test uses the *real* checked-in Reference Plugin Catalog and Go's
// production Catalog.ResolveEnvironment. Executor Binding is supplied as an
// explicit test-only value: Reference does not thereby gain execution rights.
// No plugins are started and no database, credentials or network are accessed.

import (
	"testing"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/plugin"
	"tei/internal/realization"
)

const referenceOracleQueryExecutor core.CanonicalID = "database.query.oracle"

func resolvedReferenceCatalog(t *testing.T, env environment.Environment, wPlugins []plugin.Definition, wBundles []plugin.Bundle) plugin.ResolvedSet {
	t.Helper()
	resolved, err := (plugin.Catalog{Plugins: wPlugins, Bundles: wBundles}).ResolveEnvironment(env)
	if err != nil {
		t.Fatalf("resolve real Reference Plugin Catalog for %s: %v", env.ID, err)
	}
	if resolved.Environment != env.ID {
		t.Fatalf("unexpected resolved Environment %s, wanted %s", resolved.Environment, env.ID)
	}
	return resolved
}

func referenceOracleQueryCapability(t *testing.T, set plugin.ResolvedSet) plugin.ResolvedCapability {
	t.Helper()
	cap, ok := set.Capability(plugin.CapabilityContractQueryExecutor, referenceOracleQueryExecutor)
	if !ok {
		t.Fatal("real Reference Catalog did not resolve exactly one Oracle Contract Query executor Capability")
	}
	if cap.Plugin.ID != "tei.database.oracle" || cap.Plugin.Version != "1.0.0" ||
		cap.Capability.Kind != plugin.CapabilityContractQueryExecutor ||
		!cap.Capability.SupportsDatabaseProvider(environment.ProviderOracle) {
		t.Fatalf("unexpected real Reference Oracle Query Capability: %+v", cap)
	}
	return cap
}

func actualCatalogTestBinding(envID, instanceID core.CanonicalID) realization.ContractQueryExecutorBinding {
	return realization.ContractQueryExecutorBinding{
		Environment: envID,
		Instance:    instanceID,
		Target:      customerTarget,
		Executor:    referenceOracleQueryExecutor,
	}
}

func TestGoReferenceActualCatalogHonorsRealDevelopmentBindingAndProductionGap(t *testing.T) {
	w := referenceWorld(t)
	_ = referenceQuery(t, w)
	for _, id := range []core.CanonicalID{"development", "production"} {
		t.Run(string(id), func(t *testing.T) {
			env := referenceEnvironment(t, w, id)
			plugins := resolvedReferenceCatalog(t, env, w.Plugins, w.Bundles)
			referenceOracleQueryCapability(t, plugins)
			order := referenceInstance(t, env, "order_entry", "master")
			repair := referenceInstance(t, env, "repair_intake", "customer_master")
			if order.ID != repair.ID || order.Provider != environment.ProviderOracle {
				t.Fatalf("Reference routes drifted for %s: %+v %+v", id, order, repair)
			}
			// Reference now explicitly binds development's customer query; production
			// deliberately has no such declaration. Never infer one from the provider.
			var matches []realization.ContractQueryExecutorBinding
			for _, b := range w.ContractQueryExecutorBindings {
				if b.Environment == env.ID && b.Instance == order.ID && b.Target == customerTarget {
					matches = append(matches, b)
				}
			}
			switch id {
			case "development":
				if len(matches) != 1 || matches[0].Executor != referenceOracleQueryExecutor {
					t.Fatalf("development requires exactly its checked-in Oracle executor Binding; got %+v", matches)
				}
				resolved, err := realization.ResolveContractQueryExecutor(
					env, plugins, w.ContractQueryExecutorBindings, order.ID, customerTarget)
				if err != nil {
					t.Fatalf("actual Reference development Binding must resolve: %v", err)
				}
				if resolved.Binding != matches[0] ||
					resolved.Instance.ID != order.ID ||
					resolved.Capability.Plugin.ID != "tei.database.oracle" ||
					resolved.Capability.Capability.ID != referenceOracleQueryExecutor {
					t.Fatalf("real declared Binding selected wrong Go Capability: %+v", resolved)
				}
			case "production":
				if len(matches) != 0 {
					t.Fatalf("Reference production acquired an executor Binding: %+v", matches)
				}
				if _, err := realization.ResolveContractQueryExecutor(
					env, plugins, w.ContractQueryExecutorBindings, order.ID, customerTarget); err == nil {
					t.Fatal("production Catalog Capability was selected without an explicit Binding")
				}
			}
		})
	}
}

func TestGoReferenceActualCatalogWithTestOnlyExplicitBindingResolves(t *testing.T) {
	w := referenceWorld(t)
	for _, id := range []core.CanonicalID{"development", "production"} {
		t.Run(string(id), func(t *testing.T) {
			env := referenceEnvironment(t, w, id)
			plugins := resolvedReferenceCatalog(t, env, w.Plugins, w.Bundles)
			realCap := referenceOracleQueryCapability(t, plugins)
			order := referenceInstance(t, env, "order_entry", "master")
			repair := referenceInstance(t, env, "repair_intake", "customer_master")
			binding := actualCatalogTestBinding(env.ID, order.ID)
			for _, instance := range []environment.DatabaseInstance{order, repair} {
				resolved, err := realization.ResolveContractQueryExecutor(
					env, plugins, []realization.ContractQueryExecutorBinding{binding},
					instance.ID, customerTarget)
				if err != nil {
					t.Fatalf("actual Catalog + test-only Binding failed: %v", err)
				}
				if resolved.Instance.ID != instance.ID ||
					resolved.Binding != binding ||
					resolved.Capability.Plugin != realCap.Plugin ||
					resolved.Capability.Capability.ID != realCap.Capability.ID {
					t.Fatalf("wrong typed resolution from actual Catalog: %+v", resolved)
				}
			}
		})
	}
}

func TestGoReferenceActualCatalogBindingCannotSelectWrongCapability(t *testing.T) {
	w := referenceWorld(t)
	env := referenceEnvironment(t, w, "development")
	instance := referenceInstance(t, env, "order_entry", "master")
	base := resolvedReferenceCatalog(t, env, w.Plugins, w.Bundles)
	referenceOracleQueryCapability(t, base)
	targetBinding := actualCatalogTestBinding(env.ID, instance.ID)

	cases := []struct {
		name  string
		alter func(*plugin.ResolvedSet, *realization.ContractQueryExecutorBinding)
	}{
		{"missing_executor_capability", func(s *plugin.ResolvedSet, _ *realization.ContractQueryExecutorBinding) {
			filtered := make([]plugin.ResolvedCapability, 0, len(s.Capabilities))
			for _, cap := range s.Capabilities {
				if cap.Capability.Kind != plugin.CapabilityContractQueryExecutor || cap.Capability.ID != referenceOracleQueryExecutor {
					filtered = append(filtered, cap)
				}
			}
			s.Capabilities = filtered
		}},
		{"wrong_capability_kind", func(_ *plugin.ResolvedSet, b *realization.ContractQueryExecutorBinding) {
			b.Executor = "database.oracle" // real Catalog has database-provider, not a Query executor
		}},
		{"duplicate_capability", func(s *plugin.ResolvedSet, _ *realization.ContractQueryExecutorBinding) {
			cap, ok := s.Capability(plugin.CapabilityContractQueryExecutor, referenceOracleQueryExecutor)
			if !ok {
				t.Fatal("actual catalog missing query executor")
			}
			s.Capabilities = append(s.Capabilities, cap)
		}},
		{"wrong_provider", func(s *plugin.ResolvedSet, _ *realization.ContractQueryExecutorBinding) {
			for i := range s.Capabilities {
				if s.Capabilities[i].Capability.Kind == plugin.CapabilityContractQueryExecutor &&
					s.Capabilities[i].Capability.ID == referenceOracleQueryExecutor {
					s.Capabilities[i].Capability.DatabaseProviders = []environment.DatabaseProvider{environment.ProviderPostgres}
				}
			}
		}},
		{"wrong_executor_id", func(_ *plugin.ResolvedSet, b *realization.ContractQueryExecutorBinding) {
			b.Executor = "missing.query.executor"
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resolved := resolvedReferenceCatalog(t, env, w.Plugins, w.Bundles)
			binding := targetBinding
			tc.alter(&resolved, &binding)
			if _, err := realization.ResolveContractQueryExecutor(
				env, resolved, []realization.ContractQueryExecutorBinding{binding},
				instance.ID, customerTarget); err == nil {
				t.Fatal("Go accepted ambiguous or incompatible Capability")
			}
		})
	}
}
