package driver_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"tei/internal/composition"
	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/environment"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/servicehost"
)

// The authoring and Environment Bindings below are loaded from the REAL
// reference Workspace. The current deployment reader, Host principals,
// Provider acceptance, artifact verifier and process are still test doubles.
// Database Contract Query and test-only Service Contract are not equivalent
// accepted Contracts merely because this bridge invokes the same test process.
func TestReferenceWorkspaceBindingsGateHostCurrentServiceRoutes(t *testing.T) {
	world := referenceWorld(t)
	const contractID = "test-only.customer-query-service"
	contract := composition.Contract{
		ID: contractID, Version: "1.0.0", Shape: "test-only-customer-id",
	}
	digest := "sha256:" + strings.Repeat("c", 64)
	physical := plugin.Identity{ID: "tei.reference.customer-service", Version: "1.0.0"}
	capabilityID := core.CanonicalID("database.query.oracle")
	selected := pruntime.ServiceProviderSelection{
		ContractID: contractID, ContractVersion: contract.Version,
		ContractFingerprint: "sha256:" + strings.Repeat("d", 64),
		Scope:               "application-runtime",
		Provider:            physical,
		ImplementationID:    "test-only-customer-adapter",
		ArtifactDigest:      digest,
	}
	process := &sharedCustomerServiceProcess{}
	verifier := &sharedCustomerServiceVerifier{}
	acceptance := &sharedCustomerServiceAcceptance{}
	runtimeHost := pruntime.Registry{
		Processes: process, Verifier: verifier, ServiceAcceptance: acceptance,
	}
	patterns := map[core.CanonicalID]core.CanonicalID{
		"order_entry":   "order_entry.customer_reference",
		"repair_intake": "repair_intake.customer_reference",
	}
	bindings := map[core.CanonicalID]composition.Binding{}
	routes := map[core.CanonicalID]servicehost.SelectedServiceCarrierRoute{}
	for _, app := range []core.CanonicalID{"order_entry", "repair_intake"} {
		binding := composition.Binding{
			ContractID: contractID, ContractVersion: contract.Version,
			ProviderID: "logical-" + string(app), ArtifactDigest: digest, Generation: 1,
		}
		bindings[app] = binding
		plan := pruntime.EnvironmentPlan{
			Environment: "development", PluginLockFingerprint: "test-only-lock",
			Plugins: []pruntime.PluginRuntime{{
				Plugin: physical, DescriptorFingerprint: "test-only-descriptor",
				Boundary: pruntime.BoundaryProcess, Trust: pruntime.TrustExternalProcess,
				Command: "/test-only/customer", SandboxPolicy: plugin.SandboxNone,
				Artifact: plugin.Artifact{Digest: digest},
			}},
			Capabilities: []pruntime.CapabilityRuntime{{
				Plugin:     physical,
				Capability: plugin.Capability{Kind: plugin.CapabilityContractQueryExecutor, ID: capabilityID},
				Boundary:   pruntime.BoundaryProcess,
			}},
			ServiceBindings: []pruntime.ServiceCarrierBinding{{
				ServiceProviderSelection: selected,
				CarrierKind:              plugin.CapabilityContractQueryExecutor,
				CarrierID:                capabilityID,
				Operation:                "find_by_id",
			}},
		}
		plan.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(plan)
		routes[app] = servicehost.SelectedServiceCarrierRoute{
			Binding: binding, Selection: selected, Plan: plan,
		}
	}
	reads := map[core.CanonicalID]int{}
	current := func(
		_ context.Context, app, env core.CanonicalID, pinned composition.Binding,
	) (servicehost.SelectedServiceCarrierRoute, error) {
		reads[app]++
		if env != "development" || pinned != bindings[app] {
			return servicehost.SelectedServiceCarrierRoute{}, errors.New("test Host generation mismatch")
		}
		resolved, err := driver.NewResolver(world).ResolveReferenceContractQuery(app, patterns[app])
		if err != nil {
			return servicehost.SelectedServiceCarrierRoute{}, err
		}
		if resolved.Database.ID != "master_data" ||
			resolved.Contract.ID != "master_data.customer" ||
			resolved.Query.Name != "find_by_id" {
			return servicehost.SelectedServiceCarrierRoute{}, errors.New("unexpected authored Database Contract Query")
		}
		// The source checks the current typed Workspace Binding for THIS
		// Application and its own datasource alias; matching physical Plugin
		// identity alone is insufficient.
		envCount, matched := 0, 0
		var instance core.CanonicalID
		for _, environment := range world.Environments {
			if environment.ID != env {
				continue
			}
			envCount++
			for _, b := range environment.Bindings {
				if b.Application == app && b.Alias == resolved.Datasource.Alias {
					matched++
					instance = b.Instance
				}
			}
			if _, ok := environment.InstanceByID(instance); !ok {
				return servicehost.SelectedServiceCarrierRoute{}, errors.New("current Database instance missing")
			}
		}
		if envCount != 1 || matched != 1 || instance != "master_data.dev" {
			return servicehost.SelectedServiceCarrierRoute{}, errors.New("test current Environment Binding not accepted")
		}
		return routes[app], nil
	}
	makeInvoker := func(app core.CanonicalID) *servicehost.SelectedServiceCarrierInvoker {
		t.Helper()
		scoped, err := servicehost.BindHostApplicationEnvironmentRoute(app, "development", current)
		if err != nil {
			t.Fatal(err)
		}
		scope := "reference/" + string(app)
		logical := "logical-" + string(app)
		registry := composition.NewRegistry()
		if _, err := registry.Apply(0, []composition.Provider{{
			ID: logical, Scope: scope, Offers: contract,
		}}, nil); err != nil {
			t.Fatal(err)
		}
		selectedProvider, err := registry.Select(logical, scope)
		if err != nil {
			t.Fatal(err)
		}
		coordinator, err := composition.NewSelectedCoordinator(
			registry, selectedProvider, bindings[app], composition.NewScope(),
			func(actual composition.Binding) error {
				if actual != bindings[app] {
					return errors.New("test Host generation withdrawn")
				}
				return nil
			},
		)
		if err != nil {
			t.Fatal(err)
		}
		invoker, err := servicehost.NewSelectedServiceCarrierInvoker(coordinator, runtimeHost, scoped)
		if err != nil {
			t.Fatal(err)
		}
		return invoker
	}
	order := makeInvoker("order_entry")
	repair := makeInvoker("repair_intake")
	invoke := func(app core.CanonicalID, invoker *servicehost.SelectedServiceCarrierInvoker, id int64) error {
		t.Helper()
		payload, err := json.Marshal(struct {
			ID int64 `json:"id"`
		}{ID: id})
		if err != nil {
			return err
		}
		result, err := invoker.Invoke(context.Background(), string(app)+"-request", payload)
		if err != nil {
			return err
		}
		if result.Generation != bindings[app] || result.Service.Selection != selected ||
			result.Service.Runtime.Provenance.Environment != "development" {
			return fmt.Errorf("Host invocation changed pinned Application, Provider or Environment")
		}
		return nil
	}
	check := func(orderReads, repairReads, acceptedCount, verifierCount int, ids []int64) {
		t.Helper()
		if reads["order_entry"] != orderReads || reads["repair_intake"] != repairReads ||
			acceptance.calls != acceptedCount || verifier.calls != verifierCount ||
			!slices.Equal(process.received, ids) {
			t.Fatalf("current source/Runtime counts mismatch: read=%v accepted=%d verify=%d process=%v",
				reads, acceptance.calls, verifier.calls, process.received)
		}
	}
	if err := invoke("order_entry", order, 42); err != nil {
		t.Fatal(err)
	}
	if err := invoke("repair_intake", repair, 84); err != nil {
		t.Fatal(err)
	}
	check(1, 1, 2, 2, []int64{42, 84})

	// Re-reading a changed, in-memory Environment is NOT automatic live
	// reload from a production Deployment source. It is a test Host reader.
	dev := -1
	for i := range world.Environments {
		if world.Environments[i].ID == "development" {
			dev = i
			break
		}
	}
	if dev < 0 {
		t.Fatal("reference Workspace lacks development Environment")
	}
	saved := append([]environment.ApplicationDatabaseBinding(nil), world.Environments[dev].Bindings...)
	withoutOrder := make([]environment.ApplicationDatabaseBinding, 0, len(saved))
	removed := 0
	for _, b := range saved {
		if b.Application == "order_entry" && b.Alias == "master" {
			removed++
			continue
		}
		withoutOrder = append(withoutOrder, b)
	}
	if removed != 1 {
		t.Fatalf("expected one authored order/master Binding, got %d", removed)
	}
	world.Environments[dev].Bindings = withoutOrder
	if err := invoke("order_entry", order, 42); !errors.Is(err, servicehost.ErrServiceCarrierGenerationUnavailable) {
		t.Fatalf("removed order Binding reached shared Provider: %v", err)
	}
	check(2, 1, 2, 2, []int64{42, 84})
	if err := invoke("repair_intake", repair, 84); err != nil {
		t.Fatalf("independent repair Binding was affected: %v", err)
	}
	check(2, 2, 3, 3, []int64{42, 84, 84})

	// Reassigning the same alias to a different real instance must not borrow
	// the already-admitted master_data.dev Service Route.
	reassigned := append([]environment.ApplicationDatabaseBinding(nil), saved...)
	for i := range reassigned {
		if reassigned[i].Application == "order_entry" && reassigned[i].Alias == "master" {
			reassigned[i].Instance = "sales_core.dev"
		}
	}
	world.Environments[dev].Bindings = reassigned
	if err := invoke("order_entry", order, 42); !errors.Is(err, servicehost.ErrServiceCarrierGenerationUnavailable) {
		t.Fatalf("reassigned Database instance borrowed the old Service Route: %v", err)
	}
	check(3, 2, 3, 3, []int64{42, 84, 84})

	// Duplicate authoring is ambiguous even when both candidates name the
	// same physical Database; it cannot silently authorize the old Route.
	world.Environments[dev].Bindings = append(
		append([]environment.ApplicationDatabaseBinding(nil), saved...),
		environment.ApplicationDatabaseBinding{
			Application: "order_entry", Alias: "master", Instance: "master_data.dev",
		},
	)
	if err := invoke("order_entry", order, 42); !errors.Is(err, servicehost.ErrServiceCarrierGenerationUnavailable) {
		t.Fatalf("ambiguous Application Binding was accepted: %v", err)
	}
	check(4, 2, 3, 3, []int64{42, 84, 84})

	world.Environments[dev].Bindings = saved
	acceptance.revoked = true
	if err := invoke("order_entry", order, 42); !errors.Is(err, pruntime.ErrServiceAcceptanceDenied) {
		t.Fatalf("current shared Provider revocation was bypassed by order: %v", err)
	}
	if err := invoke("repair_intake", repair, 84); !errors.Is(err, pruntime.ErrServiceAcceptanceDenied) {
		t.Fatalf("current shared Provider revocation was bypassed by repair: %v", err)
	}
	check(5, 3, 5, 3, []int64{42, 84, 84})

	acceptance.revoked = false
	if err := invoke("order_entry", order, 42); err != nil {
		t.Fatalf("restored current Binding and Provider did not recover: %v", err)
	}
	check(6, 3, 6, 4, []int64{42, 84, 84, 42})
}
