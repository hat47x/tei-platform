package localcpureuse

// Integration of checked-in Reference declarations into the EXISTING Go Driver
// runtime registry. Every executor below is an in-memory test double.
// No Oracle or Plugin Runtime invocation, permissions, secrets or data export.
import (
	"context"
	"errors"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	"tei/internal/environment"
	"tei/internal/realization"
)

type recordingReferenceExecutor struct {
	provider environment.DatabaseProvider
	called   int
	target   realization.ContractQueryTarget
	query    database.QueryContract
	inputs   map[string]any
	result   driver.ContractQueryResult
	err      error
}

func (e *recordingReferenceExecutor) DatabaseProvider() environment.DatabaseProvider {
	return e.provider
}
func (e *recordingReferenceExecutor) ExecuteContractQuery(
	_ context.Context, target realization.ContractQueryTarget,
	query database.QueryContract, inputs map[string]any,
) (driver.ContractQueryResult, error) {
	e.called++
	e.target = target
	e.query = query
	e.inputs = make(map[string]any, len(inputs))
	for k, v := range inputs {
		e.inputs[k] = v
	}
	return e.result, e.err
}

type referenceCall struct {
	name            string
	app             core.CanonicalID
	pattern         core.CanonicalID
	sourceAttribute core.CanonicalID
	instance        core.CanonicalID
}

var referenceCalls = []referenceCall{
	{"order_entry", "order_entry", "order_entry.customer_reference", "sales.order.customer_id", "master_data.dev"},
	{"repair_intake", "repair_intake", "repair_intake.customer_reference", "repair_intake.intake.customer_id", "master_data.dev"},
}

func sourceMapped(ref referenceCall) queryresult.MappedResult {
	return queryresult.MappedResult{
		Columns: []queryresult.Column{{Raw: "customer_id", Attribute: ref.sourceAttribute}},
		Rows:    [][]any{{int64(42)}},
	}
}
func registryFor(ref referenceCall, env core.CanonicalID, executor driver.ContractQueryExecutor) driver.ContractQueryExecutorMap {
	return driver.ContractQueryExecutorMap{
		{Environment: env, Instance: ref.instance, Capability: "database.query.oracle"}: executor,
	}
}
func customerResult(n int) driver.ContractQueryResult {
	rows := make([]driver.ContractQueryRow, n)
	for i := 0; i < n; i++ {
		rows[i] = driver.ContractQueryRow{Attributes: map[core.CanonicalID]any{
			"customer.customer.customer_id": int64(i + 42),
		}}
	}
	return driver.ContractQueryResult{Rows: rows}
}
func executeReference(t *testing.T, ref referenceCall, registry driver.ContractQueryExecutorRegistry, envID core.CanonicalID) (driver.ReferenceContractQueryEnvironmentExecution, error) {
	t.Helper()
	return driver.NewResolver(referenceWorld(t)).ExecuteReferenceContractQueryInEnvironment(
		context.Background(), envID, ref.app, ref.pattern, sourceMapped(ref), 0, registry)
}

func TestGoReferenceRealWorkspaceRequiresExactRuntimeRegistry(t *testing.T) {
	for _, ref := range referenceCalls {
		t.Run(ref.name, func(t *testing.T) {
			// Real developer Binding and real resolved Catalog exist. This does
			// not create a runtime implementation in the registry.
			_, err := executeReference(t, ref, driver.ContractQueryExecutorMap{}, "development")
			if err == nil || !strings.Contains(err.Error(), "runtime ExecutorがRegistryにありません") {
				t.Fatalf("missing runtime Registry implementation was not rejected: %v", err)
			}
			fake := &recordingReferenceExecutor{provider: environment.ProviderOracle, result: customerResult(1)}
			for name, key := range map[string]driver.ContractQueryExecutorRuntimeKey{
				"other_environment": {Environment: "production", Instance: ref.instance, Capability: "database.query.oracle"},
				"other_instance":    {Environment: "development", Instance: "master_data.other", Capability: "database.query.oracle"},
				"other_capability":  {Environment: "development", Instance: ref.instance, Capability: "database.query.postgres"},
			} {
				t.Run(name, func(t *testing.T) {
					_, err := executeReference(t, ref, driver.ContractQueryExecutorMap{key: fake}, "development")
					if err == nil || !strings.Contains(err.Error(), "runtime ExecutorがRegistryにありません") {
						t.Fatalf("runtime Registry accepted mismatched %s key: %v", name, err)
					}
					if fake.called != 0 {
						t.Fatalf("wrong runtime implementation invoked %d times", fake.called)
					}
				})
			}
		})
	}
}

func TestGoReferenceRealWorkspaceTwoAppsExecuteOnlyInjectedFake(t *testing.T) {
	for _, ref := range referenceCalls {
		t.Run(ref.name, func(t *testing.T) {
			for _, rows := range []int{0, 1} {
				fake := &recordingReferenceExecutor{provider: environment.ProviderOracle, result: customerResult(rows)}
				got, err := executeReference(t, ref, registryFor(ref, "development", fake), "development")
				if err != nil {
					t.Fatalf("real Reference and in-memory fake: %v", err)
				}
				if fake.called != 1 || fake.target != customerTarget ||
					fake.query.Name != "find_by_id" || fake.query.ResultCardinality != database.QueryResultZeroOrOne ||
					len(fake.inputs) != 1 || fake.inputs["id"] != int64(42) {
					t.Fatalf("incorrect exact Query boundary: calls=%d target=%+v query=%s inputs=%+v",
						fake.called, fake.target, fake.query.Name, fake.inputs)
				}
				if got.Executor.Instance.ID != ref.instance ||
					got.Executor.Capability.Capability.ID != "database.query.oracle" ||
					len(got.Execution.Result.Rows) != rows {
					t.Fatalf("incorrect Go Driver composition: %+v", got.Executor)
				}
			}
		})
	}
}

func TestGoReferenceRealWorkspaceRejectsBadFakeBeforeOrAfterCall(t *testing.T) {
	for _, ref := range referenceCalls {
		t.Run(ref.name, func(t *testing.T) {
			badProvider := &recordingReferenceExecutor{provider: environment.ProviderPostgres, result: customerResult(1)}
			if _, err := executeReference(t, ref, registryFor(ref, "development", badProvider), "development"); err == nil || !strings.Contains(err.Error(), "provider postgres") {
				t.Fatalf("wrong runtime Provider was accepted: %v", err)
			}
			if badProvider.called != 0 {
				t.Fatal("provider mismatch invoked executor")
			}
			multiple := &recordingReferenceExecutor{provider: environment.ProviderOracle, result: customerResult(2)}
			if _, err := executeReference(t, ref, registryFor(ref, "development", multiple), "development"); err == nil || !strings.Contains(err.Error(), "zero-or-one") {
				t.Fatalf("result cardinality violation accepted: %v", err)
			}
			wrongData := &recordingReferenceExecutor{provider: environment.ProviderOracle,
				result: driver.ContractQueryResult{Rows: []driver.ContractQueryRow{{
					Attributes: map[core.CanonicalID]any{"sales.order.order_id": int64(2)},
				}}}}
			if _, err := executeReference(t, ref, registryFor(ref, "development", wrongData), "development"); err == nil || !strings.Contains(err.Error(), "Semantic attribute") {
				t.Fatalf("result Data mismatch accepted: %v", err)
			}
			fail := &recordingReferenceExecutor{provider: environment.ProviderOracle, err: errors.New("synthetic-unavailable")}
			if _, err := executeReference(t, ref, registryFor(ref, "development", fail), "development"); err == nil || !strings.Contains(err.Error(), "synthetic-unavailable") {
				t.Fatalf("executor error silently converted to zero rows: %v", err)
			}
		})
	}
}

func TestGoReferenceProductionDoesNotFallbackToDevelopmentExecutor(t *testing.T) {
	for _, ref := range referenceCalls {
		t.Run(ref.name, func(t *testing.T) {
			fake := &recordingReferenceExecutor{provider: environment.ProviderOracle, result: customerResult(1)}
			_, err := executeReference(t, ref, registryFor(ref, "development", fake), "production")
			if err == nil || !strings.Contains(err.Error(), "executor Bindingがありません") {
				t.Fatalf("production should have no explicit Contract Query executor Binding: %v", err)
			}
			if fake.called != 0 {
				t.Fatal("development fake executor invoked for production")
			}
		})
	}
}
