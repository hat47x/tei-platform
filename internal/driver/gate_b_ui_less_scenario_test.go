package driver_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	"tei/internal/query"
	"tei/internal/realization"
	"tei/internal/security/legacyauth"
	"tei/internal/security/legacyauthorization"
	"tei/internal/semantic"
	"tei/internal/semantic/capabilityinvoke"
	"tei/internal/semantic/validationeval"
)

// gateBUIlessScenarioEvidenceは、既存Driver境界を一つのReference Scenarioとして
// 接続できたことだけをtest-localに記録する。production契約ではない。
type gateBUIlessScenarioEvidence struct {
	BrowseQuery             core.CanonicalID
	RecordPattern           core.CanonicalID
	ReferencePattern        core.CanonicalID
	ReferenceInstance       core.CanonicalID
	HeaderDetailPattern     core.CanonicalID
	HeaderDetailInstance    core.CanonicalID
	Validation              map[core.CanonicalID]validationeval.Result
	AuthorizationOutcome    string
	CommandCapability       core.CanonicalID
	ObservationReadIdentity string
	ObservationNature       driver.ObservationEvidenceNature
	ObservedState           string
}

func TestGateBUIlessReferenceScenarioConnectsExistingDriverBoundariesWithoutNewRuntimeMeaning(t *testing.T) {
	ctx := context.Background()
	world := referenceWorld(t)
	resolver := driver.NewResolver(world)
	evidence := gateBUIlessScenarioEvidence{}

	browse := browseQueryExecutorFunc(func(
		_ context.Context,
		q query.EmbeddedSQL,
		datasource application.DatabaseRequirement,
		parameters map[string]any,
	) (driver.BrowseQueryResult, error) {
		if q.ID != "order_entry.recent_orders" || datasource.Alias != "sales" || datasource.TargetDatabase != "sales_core" {
			return driver.BrowseQueryResult{}, fmt.Errorf("browse resolutionがReference authoringと一致しません")
		}
		if _, ok := parameters["from"]; !ok {
			return driver.BrowseQueryResult{}, fmt.Errorf("recent_ordersのfrom parameterがありません")
		}
		return driver.BrowseQueryResult{
			Raw: queryresult.RawResult{
				Columns: []string{"total_amount", "status", "ordered_at", "customer_id", "order_id"},
				Rows:    [][]any{{3100.00, "READY", time.Date(2026, 9, 3, 11, 30, 0, 0, time.UTC), int64(42), int64(1001)}},
			},
			Payload: "synthetic-gate-b-browse",
		}, nil
	})
	browseExecution, err := resolver.ExecuteBrowseQuery(
		ctx,
		"order_entry",
		"order_entry.recent_orders",
		browse,
		map[string]any{"from": time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)},
	)
	if err != nil {
		t.Fatalf("Gate B browseを実行できません: %v", err)
	}
	if len(browseExecution.Result.Mapped.Rows) != 1 {
		t.Fatalf("Gate B browse row=%#v", browseExecution.Result.Mapped.Rows)
	}
	evidence.BrowseQuery = browseExecution.Resolution.Query.ID

	record, err := resolver.ResolvePattern("order_entry", "order_entry.order_record")
	if err != nil {
		t.Fatalf("order record Patternを解決できません: %v", err)
	}
	if record.Data == nil || record.Data.ID != "sales.order" || record.Datasource != nil || record.Query != nil {
		t.Fatalf("record PatternへCanonicalにない取得意味を補っています: %#v", record)
	}
	evidence.RecordPattern = record.Pattern.ID

	customerExecutor := &recordingContractQueryExecutor{
		result: driver.ContractQueryResult{Rows: []driver.ContractQueryRow{{Attributes: map[core.CanonicalID]any{
			"customer.customer.customer_id": int64(42),
		}}}},
	}
	customerExecution, err := resolver.ExecuteReferenceContractQueryInEnvironment(
		ctx,
		"development",
		"order_entry",
		"order_entry.customer_reference",
		browseExecution.Result.Mapped,
		0,
		contractQueryExecutorRegistry("master_data.dev", customerExecutor),
	)
	if err != nil {
		t.Fatalf("Gate B Customer Referenceを実行できません: %v", err)
	}
	if customerExecutor.parameters["id"] != int64(42) {
		t.Fatalf("browse mapped rowからCustomer Relation keyを渡せていません: %#v", customerExecutor.parameters)
	}
	evidence.ReferencePattern = customerExecution.Execution.Resolution.Pattern.ID
	evidence.ReferenceInstance = customerExecution.Executor.Instance.ID

	lineExecutor := &recordingContractQueryExecutor{
		result: driver.ContractQueryResult{Rows: []driver.ContractQueryRow{
			{Attributes: map[core.CanonicalID]any{"sales.order_line.order_line_id": int64(11)}},
			{Attributes: map[core.CanonicalID]any{"sales.order_line.order_line_id": int64(12)}},
		}},
	}
	lineExecution, err := resolver.ExecuteHeaderDetailContractQueryInEnvironment(
		ctx,
		"development",
		"order_entry",
		"order_entry.order_lines",
		browseExecution.Result.Mapped,
		0,
		headerDetailExecutorRegistry(lineExecutor),
	)
	if err != nil {
		t.Fatalf("Gate B OrderLine header-detailを実行できません: %v", err)
	}
	if lineExecutor.parameters["order_id"] != int64(1001) {
		t.Fatalf("browse mapped rowからOrder identityをheader-detailへ渡せていません: %#v", lineExecutor.parameters)
	}
	if len(lineExecution.Execution.Result.Rows) != 2 {
		t.Fatalf("OrderLine collection=%#v", lineExecution.Execution.Result.Rows)
	}
	evidence.HeaderDetailPattern = lineExecution.Execution.Resolution.Pattern.ID
	evidence.HeaderDetailInstance = lineExecution.Executor.Instance.ID

	quantityPositive, lineNumberUnique := orderEntryValidationPair(t, world)
	validationSource := submitOrderValidationSource{
		quantity:    mustRat(t, "2.5"),
		lineNumbers: []string{"1", "2"},
	}
	state := &syntheticOrderStateStore{states: map[string]string{"order-1001": "pending"}}
	reader := &syntheticOrderStateReader{store: state}
	commandResult, err := runSubmitOrderGateScenario(
		ctx,
		resolver,
		gateBCompositeAdapters(state),
		"synthetic-order-editor",
		"synthetic-proof",
		validationSource,
		[]semantic.Validation{quantityPositive, lineNumberUnique},
	)
	if err != nil {
		t.Fatalf("Gate B Validation / Authorization / Commandを実行できません: %v", err)
	}
	if !commandResult.CommandWasInvoked || commandResult.Authorization.Outcome != "allowed" {
		t.Fatalf("Gate B Command gate=%#v", commandResult)
	}
	if commandResult.Validation[quantityPositive.ID] != validationeval.ResultSatisfied ||
		commandResult.Validation[lineNumberUnique.ID] != validationeval.ResultSatisfied {
		t.Fatalf("Gate B Validation=%#v", commandResult.Validation)
	}
	if reader.calls != 0 {
		t.Fatalf("Command実行だけでObservationを暗黙実行しています: calls=%d", reader.calls)
	}
	evidence.Validation = commandResult.Validation
	evidence.AuthorizationOutcome = string(commandResult.Authorization.Outcome)
	evidence.CommandCapability = commandResult.Command.Capability

	observation, err := driver.ExecuteObservationRead(
		ctx,
		string(reader.Identity()),
		driver.ObservationEvidenceSynthetic,
		func(ctx context.Context) (any, error) { return reader.Read(ctx, "order-1001") },
	)
	if err != nil {
		t.Fatalf("Gate B Observation readを実行できません: %v", err)
	}
	observedState, ok := observation.Result.(string)
	if !ok || observedState != "submitted" {
		t.Fatalf("Gate B Observed state=%#v", observation.Result)
	}
	evidence.ObservationReadIdentity = observation.ReadIdentity
	evidence.ObservationNature = observation.Nature
	evidence.ObservedState = observedState

	if evidence.BrowseQuery != "order_entry.recent_orders" ||
		evidence.RecordPattern != "order_entry.order_record" ||
		evidence.ReferencePattern != "order_entry.customer_reference" ||
		evidence.ReferenceInstance != "master_data.dev" ||
		evidence.HeaderDetailPattern != "order_entry.order_lines" ||
		evidence.HeaderDetailInstance != "sales_core.dev" ||
		evidence.AuthorizationOutcome != "allowed" ||
		evidence.CommandCapability != "sales.order.submit" ||
		evidence.ObservationReadIdentity != "synthetic.order_state.read" ||
		evidence.ObservationNature != driver.ObservationEvidenceSynthetic ||
		evidence.ObservedState != "submitted" {
		t.Fatalf("Gate B UI-less Scenario Evidenceが不完全です: %#v", evidence)
	}
}

func gateBCompositeAdapters(state *syntheticOrderStateStore) driver.BindingAdapterResolver {
	return bindingAdapterResolverFunc(func(_ context.Context, binding realization.CustomGoBinding) (capabilityinvoke.Adapter, error) {
		switch binding.Identity() {
		case "order_entry/security.authenticate":
			return capabilityAdapterFunc(func(_ context.Context, capability core.CanonicalID, _ any) (capabilityinvoke.BusinessResult, error) {
				return capabilityinvoke.BusinessResult{
					Capability: capability,
					Mode:       capabilityinvoke.ResultOutcome,
					Outcome:    "authenticated",
					Payload:    legacyauth.AuthenticatedSubject{SubjectID: "synthetic-order-editor"},
				}, nil
			}), nil
		case "order_entry/security.authorize":
			return capabilityAdapterFunc(func(_ context.Context, capability core.CanonicalID, input any) (capabilityinvoke.BusinessResult, error) {
				authorizationInput, ok := input.(legacyauthorization.Input)
				if !ok {
					return capabilityinvoke.BusinessResult{}, fmt.Errorf("synthetic Authorization入力型が不正です")
				}
				return capabilityinvoke.BusinessResult{
					Capability: capability,
					Mode:       capabilityinvoke.ResultOutcome,
					Outcome:    "allowed",
					Payload: legacyauthorization.AuthorizationGrant{
						SubjectID: authorizationInput.SubjectID,
						Action:    authorizationInput.Action,
						Resource:  authorizationInput.Resource,
					},
				}, nil
			}), nil
		case "order_entry/sales.order.submit":
			return capabilityAdapterFunc(func(_ context.Context, capability core.CanonicalID, _ any) (capabilityinvoke.BusinessResult, error) {
				state.states["order-1001"] = "submitted"
				return capabilityinvoke.BusinessResult{
					Capability: capability,
					Mode:       capabilityinvoke.ResultOutput,
					Payload:    map[string]any{"accepted": true},
				}, nil
			}), nil
		default:
			return nil, fmt.Errorf("未登録のBindingです: %s", binding.Identity())
		}
	})
}
