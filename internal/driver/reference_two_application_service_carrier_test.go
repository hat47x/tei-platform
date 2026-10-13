package driver_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/driver"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/realization"
)

// All Host acceptance, artifact verification and process execution below are
// test doubles. This is NOT a real Contract Query v2.4 Service publication.
type sharedCustomerServiceProcess struct {
	received []int64
}

func (p *sharedCustomerServiceProcess) Invoke(
	_ context.Context, _ pruntime.ProcessCommand, req pruntime.Request,
) (pruntime.Response, error) {
	var input struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(req.Payload, &input); err != nil {
		return pruntime.Response{}, err
	}
	p.received = append(p.received, input.ID)
	payload, err := json.Marshal(struct {
		CustomerID int64 `json:"customer_id"`
	}{CustomerID: input.ID})
	if err != nil {
		return pruntime.Response{}, err
	}
	return pruntime.Response{Status: pruntime.ResponseOK, Payload: payload}, nil
}

type sharedCustomerServiceVerifier struct{ calls int }

func (v *sharedCustomerServiceVerifier) Verify(
	_ context.Context, _ plugin.Identity, _ plugin.Artifact,
) (pruntime.ArtifactVerification, error) {
	v.calls++
	return pruntime.ArtifactVerification{VerifierID: "synthetic-shared-customer", DigestVerified: true}, nil
}

type sharedCustomerServiceAcceptance struct {
	calls   int
	revoked bool
}

func (a *sharedCustomerServiceAcceptance) ReadCurrentAcceptedService(
	_ context.Context, selection pruntime.ServiceProviderSelection,
) (pruntime.ServiceAcceptanceRecord, error) {
	a.calls++
	if a.revoked {
		return pruntime.ServiceAcceptanceRecord{}, errors.New("test-only service acceptance revoked")
	}
	return pruntime.ServiceAcceptanceRecord{
		Selection: selection, ContractAcceptanceRef: "test-only-contract",
		ProviderConformanceRef: "test-only-provider", AuthorityRevision: "test-only-revision",
	}, nil
}

// This bridge is deliberately TEST ONLY: Driver owns the real Application/
// Environment/Instance/Capability resolution, then the selected physical
// ContractQueryExecutor delegates to the existing Product Service Carrier.
// No second resolver, permission evaluator or Provider selection is added.
type sharedCustomerServiceExecutor struct {
	host      pruntime.Registry
	plan      pruntime.EnvironmentPlan
	selection pruntime.ServiceProviderSelection
}

func (x *sharedCustomerServiceExecutor) ExecuteContractQuery(
	ctx context.Context, target realization.ContractQueryTarget,
	query database.QueryContract, parameters map[string]any,
) (driver.ContractQueryResult, error) {
	if target.Identity() != "master_data/master_data.customer/find_by_id" ||
		query.Name != "find_by_id" || len(parameters) != 1 {
		return driver.ContractQueryResult{}, fmt.Errorf("unexpected reference Query target/parameters")
	}
	customerID, ok := parameters["id"].(int64)
	if !ok {
		return driver.ContractQueryResult{}, errors.New("customer query ID has wrong type")
	}
	requestPayload, err := json.Marshal(struct {
		ID int64 `json:"id"`
	}{ID: customerID})
	if err != nil {
		return driver.ContractQueryResult{}, err
	}
	execution, err := x.host.InvokeSelectedServiceCarrier(ctx, x.plan, x.selection, "reference-customer-query", requestPayload)
	if err != nil {
		return driver.ContractQueryResult{}, err
	}
	if execution.Selection != x.selection ||
		execution.Runtime.Provenance.Artifact.Digest != x.selection.ArtifactDigest {
		return driver.ContractQueryResult{}, errors.New("runtime Service provenance mismatch")
	}
	var result struct {
		CustomerID int64 `json:"customer_id"`
	}
	if err := json.Unmarshal(execution.Runtime.Response.Payload, &result); err != nil {
		return driver.ContractQueryResult{}, err
	}
	return driver.ContractQueryResult{
		Rows: []driver.ContractQueryRow{{Attributes: map[core.CanonicalID]any{
			"customer.customer.customer_id": result.CustomerID,
		}}},
	}, nil
}

// The two REAL authored Reference Applications have distinct local datasource
// aliases but exactly one Environment/Instance/Capability. The Product Runtime
// Service Carrier is synthetic; only the Driver's source-owned selection is real.
func TestReferenceApplicationsDispatchSameResolvedExecutorThroughServiceCarrier(t *testing.T) {
	world := referenceWorld(t)
	provider := plugin.Identity{ID: "tei.reference.customer-service", Version: "1.0.0"}
	artifact := "sha256:" + strings.Repeat("c", 64)
	capabilityID := core.CanonicalID("database.query.oracle")
	selection := pruntime.ServiceProviderSelection{
		ContractID: "test-only.customer-query-service", ContractVersion: "1.0.0",
		ContractFingerprint: "sha256:" + strings.Repeat("d", 64),
		Scope: "application-runtime", Provider: provider,
		ImplementationID: "test-only-customer-adapter", ArtifactDigest: artifact,
	}
	plan := pruntime.EnvironmentPlan{
		Environment: "development", PluginLockFingerprint: "test-only-plugin-lock",
		Plugins: []pruntime.PluginRuntime{{
			Plugin: provider, DescriptorFingerprint: "test-only-descriptor",
			Boundary: pruntime.BoundaryProcess, Trust: pruntime.TrustExternalProcess,
			Command: "/test-only/customer", SandboxPolicy: plugin.SandboxNone,
			Artifact: plugin.Artifact{Digest: artifact},
		}},
		Capabilities: []pruntime.CapabilityRuntime{{
			Plugin: provider,
			Capability: plugin.Capability{Kind: plugin.CapabilityContractQueryExecutor, ID: capabilityID},
			Boundary: pruntime.BoundaryProcess,
		}},
		ServiceBindings: []pruntime.ServiceCarrierBinding{{
			ServiceProviderSelection: selection,
			CarrierKind: plugin.CapabilityContractQueryExecutor,
			CarrierID: capabilityID, Operation: "query",
		}},
	}
	plan.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(plan)
	process := &sharedCustomerServiceProcess{}
	verifier := &sharedCustomerServiceVerifier{}
	acceptance := &sharedCustomerServiceAcceptance{}
	executor := &sharedCustomerServiceExecutor{
		host: pruntime.Registry{Processes: process, Verifier: verifier, ServiceAcceptance: acceptance},
		plan: plan, selection: selection,
	}
	registry := driver.ContractQueryExecutorMap{
		{Environment: "development", Instance: "master_data.dev", Capability: capabilityID}: executor,
	}
	orderSource := customerMappedResult()
	repairSource := repairIntakeMappedResult()
	repairSource.Rows[0][0] = int64(84)

	for _, tc := range []struct {
		app, pattern core.CanonicalID
		id           int64
		repair       bool
	}{
		{"order_entry", "order_entry.customer_reference", 42, false},
		{"repair_intake", "repair_intake.customer_reference", 84, true},
	} {
		t.Run(string(tc.app), func(t *testing.T) {
			source := orderSource
			if tc.repair {
				source = repairSource
			}
			result, err := driver.NewResolver(world).ExecuteReferenceContractQueryInEnvironment(
				context.Background(), "development", tc.app, tc.pattern, source, 0, registry,
			)
			if err != nil {
				t.Fatal(err)
			}
			if result.Executor.Instance.ID != "master_data.dev" ||
				result.Executor.Capability.Capability.ID != capabilityID ||
				result.Executor.Binding.Environment != "development" {
				t.Fatalf("Driver used a different Environment/Instance/Capability: %+v", result.Executor)
			}
			if len(result.Execution.Result.Rows) != 1 ||
				result.Execution.Result.Rows[0].Attributes["customer.customer.customer_id"] != tc.id {
				t.Fatalf("%s received a different customer's result: %+v", tc.app, result.Execution.Result)
			}
		})
	}
	if !slices.Equal(process.received, []int64{42, 84}) || verifier.calls != 2 || acceptance.calls != 2 {
		t.Fatalf("two Applications did not share one selected Runtime: received=%v verifier=%d accepted=%d",
			process.received, verifier.calls, acceptance.calls)
	}

	// A different Environment has no accepted executor binding; it must not
	// borrow the development Runtime even if that Runtime is installed.
	_, err := driver.NewResolver(world).ExecuteReferenceContractQueryInEnvironment(
		context.Background(), "production", "repair_intake", "repair_intake.customer_reference",
		repairSource, 0, registry,
	)
	if err == nil {
		t.Fatal("production borrowed the development Runtime")
	}
	if len(process.received) != 2 || acceptance.calls != 2 {
		t.Fatal("failed Environment dispatch reached the Runtime")
	}

	// Revoking the shared Service/Provider acceptance denies both logical
	// Applications on their next valid Driver invocation, before the verifier
	// and test process are reached.
	acceptance.revoked = true
	for _, tc := range []struct {
		app, pattern core.CanonicalID
		repair       bool
	}{
		{"order_entry", "order_entry.customer_reference", false},
		{"repair_intake", "repair_intake.customer_reference", true},
	} {
		source := orderSource
		if tc.repair {
			source = repairSource
		}
		_, err := driver.NewResolver(world).ExecuteReferenceContractQueryInEnvironment(
			context.Background(), "development", tc.app, tc.pattern, source, 0, registry,
		)
		if !errors.Is(err, pruntime.ErrServiceAcceptanceDenied) {
			t.Fatalf("%s ignored Host Service revocation: %v", tc.app, err)
		}
	}
	if !slices.Equal(process.received, []int64{42, 84}) || verifier.calls != 2 || acceptance.calls != 4 {
		t.Fatalf("revoked Service was invoked: received=%v verifier=%d accepted=%d",
			process.received, verifier.calls, acceptance.calls)
	}
}
