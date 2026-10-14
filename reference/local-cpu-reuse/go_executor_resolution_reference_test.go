package localcpureuse

// Go-native Reference integration evidence, not a runtime implementation.
// Every workspace is the checked-in, public Reference. Synthetic executor
// Capability below is a test fixture and NEVER authorizes a Provider call.

import (
	"path/filepath"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/environment"
	"tei/internal/ir"
	"tei/internal/plugin"
	"tei/internal/realization"
	"tei/internal/workspace"
)

var customerTarget = realization.ContractQueryTarget{
	Database: "master_data", Contract: "master_data.customer", Query: "find_by_id",
}

func referenceWorld(t *testing.T) ir.World {
	t.Helper()
	loaded, err := workspace.Load(filepath.Join("..", "workspace"))
	if err != nil {
		t.Fatalf("load checked-in Reference Workspace: %v", err)
	}
	return loaded.World
}

func referenceApplication(t *testing.T, w ir.World, id core.CanonicalID) application.Application {
	t.Helper()
	var matched *application.Application
	for i := range w.Applications {
		if w.Applications[i].ID != id {
			continue
		}
		if matched != nil {
			t.Fatalf("ambiguous Reference Application identity: %s", id)
		}
		matched = &w.Applications[i]
	}
	if matched == nil {
		t.Fatalf("missing Reference Application: %s", id)
	}
	return *matched
}

func referenceEnvironment(t *testing.T, w ir.World, id core.CanonicalID) environment.Environment {
	t.Helper()
	var matched *environment.Environment
	for i := range w.Environments {
		if w.Environments[i].ID != id {
			continue
		}
		if matched != nil {
			t.Fatalf("ambiguous Reference Environment identity: %s", id)
		}
		matched = &w.Environments[i]
	}
	if matched == nil {
		t.Fatalf("missing Reference Environment: %s", id)
	}
	return *matched
}

func referenceInstance(t *testing.T, env environment.Environment, appID core.CanonicalID, alias core.LocalName) environment.DatabaseInstance {
	t.Helper()
	var instanceID core.CanonicalID
	found := false
	for _, binding := range env.Bindings {
		if binding.Application != appID || binding.Alias != alias {
			continue
		}
		if found {
			t.Fatalf("ambiguous Application-local datasource %s.%s", appID, alias)
		}
		instanceID, found = binding.Instance, true
	}
	if !found {
		t.Fatalf("no declared Application-local datasource %s.%s", appID, alias)
	}
	var matched *environment.DatabaseInstance
	for i := range env.Instances {
		if env.Instances[i].ID != instanceID {
			continue
		}
		if matched != nil {
			t.Fatalf("ambiguous Database instance %s", instanceID)
		}
		matched = &env.Instances[i]
	}
	if matched == nil {
		t.Fatalf("missing Database instance %s", instanceID)
	}
	if matched.Definition != customerTarget.Database {
		t.Fatalf("wrong Database definition for %s.%s: %s", appID, alias, matched.Definition)
	}
	return *matched
}

func referenceQuery(t *testing.T, w ir.World) database.QueryContract {
	t.Helper()
	var contract *database.Contract
	for _, db := range w.Databases {
		if db.ID != customerTarget.Database {
			continue
		}
		for i := range db.Contracts {
			if db.Contracts[i].ID == customerTarget.Contract {
				if contract != nil {
					t.Fatal("ambiguous Reference Contract identity")
				}
				contract = &db.Contracts[i]
			}
		}
	}
	if contract == nil {
		t.Fatal("missing Reference Contract")
	}
	if contract.Version.Major != 2 || contract.Version.Minor != 4 {
		t.Fatalf("Reference Contract version changed: %+v", contract.Version)
	}
	var q *database.QueryContract
	for i := range contract.Queries {
		if contract.Queries[i].Name == string(customerTarget.Query) {
			if q != nil {
				t.Fatal("ambiguous Reference Contract Query")
			}
			q = &contract.Queries[i]
		}
	}
	if q == nil {
		t.Fatal("missing Reference Contract Query")
	}
	if len(q.Inputs) != 1 || q.Inputs[0].Name != "id" ||
		q.Inputs[0].Type != "customer.customer_id" ||
		q.Result != "customer.customer" ||
		q.ResultCardinality != database.QueryResultZeroOrOne {
		t.Fatalf("Reference Contract Query signature changed: %+v", q)
	}
	return *q
}

// This fake ResolvedSet proves ONLY typed Go resolver behavior with explicit
// authoring. It is not obtained from the Reference Plugin catalog or installed.
func syntheticOracleResolvedSet(envID core.CanonicalID) plugin.ResolvedSet {
	return plugin.ResolvedSet{
		Environment: envID,
		Capabilities: []plugin.ResolvedCapability{{
			Plugin: plugin.Identity{ID: "research.synthetic.oracle", Version: "1.0.0"},
			Capability: plugin.Capability{
				Kind: plugin.CapabilityContractQueryExecutor,
				ID: "research.synthetic.customer.query",
				DatabaseProviders: []environment.DatabaseProvider{environment.ProviderOracle},
			},
		}},
	}
}

func syntheticBinding(envID, instanceID core.CanonicalID) realization.ContractQueryExecutorBinding {
	return realization.ContractQueryExecutorBinding{
		Environment: envID,
		Instance: instanceID,
		Target: customerTarget,
		Executor: "research.synthetic.customer.query",
	}
}

func TestGoReferenceCustomerQueryRoutesRemainApplicationLocal(t *testing.T) {
	w := referenceWorld(t)
	_ = referenceQuery(t, w)
	for _, envID := range []core.CanonicalID{"development", "production"} {
		t.Run(string(envID), func(t *testing.T) {
			env := referenceEnvironment(t, w, envID)
			order := referenceApplication(t, w, "order_entry")
			repair := referenceApplication(t, w, "repair_intake")
			if len(order.Patterns) == 0 || len(repair.Patterns) == 0 {
				t.Fatal("Reference Patterns not declared")
			}
			left := referenceInstance(t, env, order.ID, "master")
			right := referenceInstance(t, env, repair.ID, "customer_master")
			if left.ID != right.ID || left.Provider != environment.ProviderOracle {
				t.Fatalf("Reference Application datasource declarations drifted: left=%+v right=%+v", left, right)
			}
			// A provider-compatible Capability does not imply an executor binding.
			_, err := realization.ResolveContractQueryExecutor(
				env, syntheticOracleResolvedSet(env.ID), nil, left.ID, customerTarget)
			if err == nil {
				t.Fatal("Go resolver inferred an executor without explicit Binding")
			}
		})
	}
}

func TestGoReferenceExplicitExecutorBindingIsOnlySyntheticResolution(t *testing.T) {
	w := referenceWorld(t)
	for _, envID := range []core.CanonicalID{"development", "production"} {
		t.Run(string(envID), func(t *testing.T) {
			env := referenceEnvironment(t, w, envID)
			order := referenceInstance(t, env, "order_entry", "master")
			repair := referenceInstance(t, env, "repair_intake", "customer_master")
			binding := syntheticBinding(env.ID, order.ID)
			plugins := syntheticOracleResolvedSet(env.ID)
			for _, instance := range []environment.DatabaseInstance{order, repair} {
				resolved, err := realization.ResolveContractQueryExecutor(
					env, plugins, []realization.ContractQueryExecutorBinding{binding},
					instance.ID, customerTarget)
				if err != nil {
					t.Fatalf("explicit synthetic resolution: %v", err)
				}
				if resolved.Instance.ID != instance.ID ||
					resolved.Binding.Target != customerTarget ||
					resolved.Capability.Capability.Kind != plugin.CapabilityContractQueryExecutor ||
					resolved.Capability.Capability.ID != binding.Executor {
					t.Fatalf("wrong typed Go resolution: %+v", resolved)
				}
			}
		})
	}
}

func TestGoReferenceExecutorNegativeBoundaries(t *testing.T) {
	w := referenceWorld(t)
	env := referenceEnvironment(t, w, "development")
	instance := referenceInstance(t, env, "order_entry", "master")
	baseBinding := syntheticBinding(env.ID, instance.ID)
	type testCase struct {
		name string
		bindings []realization.ContractQueryExecutorBinding
		plugins plugin.ResolvedSet
		instance core.CanonicalID
		target realization.ContractQueryTarget
	}
	badProvider := syntheticOracleResolvedSet(env.ID)
	badProvider.Capabilities[0].Capability.DatabaseProviders =
		[]environment.DatabaseProvider{environment.ProviderPostgres}
	badKind := syntheticOracleResolvedSet(env.ID)
	badKind.Capabilities[0].Capability.Kind = plugin.CapabilityDatabaseProvider
	foreign := baseBinding
	foreign.Environment = "production"
	tests := []testCase{
		{name:"missing", plugins:syntheticOracleResolvedSet(env.ID), instance:instance.ID, target:customerTarget},
		{name:"foreign_environment", bindings:[]realization.ContractQueryExecutorBinding{foreign}, plugins:syntheticOracleResolvedSet(env.ID), instance:instance.ID, target:customerTarget},
		{name:"duplicate", bindings:[]realization.ContractQueryExecutorBinding{baseBinding,baseBinding}, plugins:syntheticOracleResolvedSet(env.ID), instance:instance.ID, target:customerTarget},
		{name:"wrong_provider", bindings:[]realization.ContractQueryExecutorBinding{baseBinding}, plugins:badProvider, instance:instance.ID, target:customerTarget},
		{name:"wrong_capability_kind", bindings:[]realization.ContractQueryExecutorBinding{baseBinding}, plugins:badKind, instance:instance.ID, target:customerTarget},
		{name:"wrong_instance", bindings:[]realization.ContractQueryExecutorBinding{baseBinding}, plugins:syntheticOracleResolvedSet(env.ID), instance:"sales_core.dev", target:customerTarget},
		{name:"wrong_target", bindings:[]realization.ContractQueryExecutorBinding{baseBinding}, plugins:syntheticOracleResolvedSet(env.ID), instance:instance.ID, target:realization.ContractQueryTarget{Database:"master_data",Contract:"master_data.customer",Query:"other"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := realization.ResolveContractQueryExecutor(
				env, tc.plugins, tc.bindings, tc.instance, tc.target); err == nil {
				t.Fatal("unexpected synthetic executor selection")
			}
		})
	}
}
