package driver_test

import (
	"context"
	"fmt"
	"math/big"
	"path/filepath"
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
	"tei/internal/semantic/capabilityinvoke"
	"tei/internal/semantic/validationeval"
)

// TestOrderEntryGateBScenarioConnectsMajorBoundariesWithoutUIは、Gate Bで個別に成立した
// Application Driver境界を、一つのtest-localなUI-less Scenarioとして接続する。
//
// Database readとCommand side effectはsynthetic adapterを用いるため、このテストを
// 実PostgreSQL / 実Oracle / 実外部状態のEvidenceへ格上げしない。
func TestOrderEntryGateBScenarioConnectsMajorBoundariesWithoutUI(t *testing.T) {
	ctx := context.Background()
	world := referenceWorld(t)
	resolver := driver.NewResolver(world)

	store, err := legacyauth.LoadFixtureStore(referenceLegacyAuthFixture(t))
	if err != nil {
		t.Fatalf("legacy authentication fixtureを読み込めません: %v", err)
	}
	authorizationRule, err := legacyauthorization.LoadFixtureRule(filepath.Join(filepath.Dir(referenceLegacyAuthFixture(t)), "authorization.json"))
	if err != nil {
		t.Fatalf("legacy authorization fixtureを読み込めません: %v", err)
	}

	now := time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC)
	authentication := legacyauth.Adapter{
		Store: store,
		Verifier: driverCredentialVerifierFunc(func(_ context.Context, ref, presented string) (bool, error) {
			if ref != "credential-ref://legacy/order-editor" {
				return false, fmt.Errorf("未登録のcredential参照です: %s", ref)
			}
			return presented == "editor-secret", nil
		}),
		Now: func() time.Time { return now },
	}
	authorization := legacyauthorization.Adapter{Store: store, Rule: authorizationRule}

	// 1. Authentication。以降のScenarioで利用するsubjectをBusiness Resultから得る。
	authenticationExecution, err := resolver.InvokeCapability(
		ctx,
		"order_entry",
		"security.authenticate",
		bindingAdapterResolverFunc(func(_ context.Context, binding realization.CustomGoBinding) (capabilityinvoke.Adapter, error) {
			if binding.Identity() != "order_entry/security.authenticate" {
				return nil, fmt.Errorf("Authenticationで未登録のBindingです: %s", binding.Identity())
			}
			return authentication, nil
		}),
		legacyauth.Input{LoginID: "legacy-order-editor", Credential: "editor-secret"},
	)
	if err != nil {
		t.Fatalf("Authenticationを実行できません: %v", err)
	}
	if authenticationExecution.Result.Outcome != "authenticated" {
		t.Fatalf("Authentication Business Result=%#v", authenticationExecution.Result)
	}
	subject, ok := authenticationExecution.Result.Payload.(legacyauth.AuthenticatedSubject)
	if !ok || subject.SubjectID == "" {
		t.Fatalf("Authentication Business Resultにsubjectがありません: %#v", authenticationExecution.Result.Payload)
	}

	// 2. browse。同じReference QueryからOrderのSemantic rowを得る。
	from := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	browseExecution, err := resolver.ExecuteBrowseQuery(
		ctx,
		"order_entry",
		"order_entry.recent_orders",
		browseQueryExecutorFunc(func(
			_ context.Context,
			q query.EmbeddedSQL,
			datasource application.DatabaseRequirement,
			parameters map[string]any,
		) (driver.BrowseQueryResult, error) {
			if q.ID != "order_entry.recent_orders" || datasource.Alias != "sales" || datasource.TargetDatabase != "sales_core" {
				return driver.BrowseQueryResult{}, fmt.Errorf("browseの明示identityが変わっています: query=%s datasource=%#v", q.ID, datasource)
			}
			if len(parameters) != 1 || parameters["from"] != from {
				return driver.BrowseQueryResult{}, fmt.Errorf("browse parameter=%#v", parameters)
			}
			return driver.BrowseQueryResult{Raw: queryresult.RawResult{
				Columns: []string{"total_amount", "status", "ordered_at", "customer_id", "order_id"},
				Rows: [][]any{{
					2500.50,
					"OPEN",
					time.Date(2026, 9, 2, 10, 0, 0, 0, time.UTC),
					int64(11),
					int64(1001),
				}},
			}}, nil
		}),
		map[string]any{"from": from},
	)
	if err != nil {
		t.Fatalf("browseを実行できません: %v", err)
	}
	if len(browseExecution.Result.Mapped.Rows) != 1 {
		t.Fatalf("browse mapped rows=%#v", browseExecution.Result.Mapped.Rows)
	}
	orderID := gateBScenarioMappedInt64(t, browseExecution.Result.Mapped, 0, "sales.order.order_id")
	customerID := gateBScenarioMappedInt64(t, browseExecution.Result.Mapped, 0, "sales.order.customer_id")

	// 3. record。同じApplication上のrecord Patternを解決し、選択したbrowse rowと同じsales.orderを扱う。
	recordResolution, err := resolver.ResolvePattern("order_entry", "order_entry.order_record")
	if err != nil {
		t.Fatalf("record Patternを解決できません: %v", err)
	}
	if recordResolution.Data == nil || recordResolution.Data.ID != "sales.order" {
		t.Fatalf("record PatternのSemantic Data=%#v", recordResolution.Data)
	}

	// 4. reference。browse rowのcustomer_idをRelation key / invocation binding経由でContract Queryへ渡す。
	customerExecutor := &recordingContractQueryExecutor{result: driver.ContractQueryResult{
		Rows: []driver.ContractQueryRow{{Attributes: map[core.CanonicalID]any{
			"customer.customer.customer_id": customerID,
		}}},
	}}
	contractQueryExecutors := driver.ContractQueryExecutorMap{
		{
			Environment: "development",
			Instance:    "master_data.dev",
			Capability:  "database.query.oracle",
		}: customerExecutor,
	}
	referenceExecution, err := resolver.ExecuteReferenceContractQueryInEnvironment(
		ctx,
		"development",
		"order_entry",
		"order_entry.customer_reference",
		browseExecution.Result.Mapped,
		0,
		contractQueryExecutors,
	)
	if err != nil {
		t.Fatalf("referenceを実行できません: %v", err)
	}
	if customerExecutor.parameters["id"] != customerID {
		t.Fatalf("browse rowのcustomer_idがreference Query inputへ引き継がれていません: %#v", customerExecutor.parameters)
	}
	if len(referenceExecution.Execution.Result.Rows) != 1 || referenceExecution.Execution.Result.Rows[0].Attributes["customer.customer.customer_id"] != customerID {
		t.Fatalf("Customer Semantic result=%#v", referenceExecution.Execution.Result)
	}

	// 5. header-detail。同じbrowse rowのorder_idをsource-side bindingからfind_by_orderへ渡す。
	quantity1 := mustRat(t, "2.5")
	quantity2 := mustRat(t, "1")
	orderLinesExecutor := &recordingContractQueryExecutor{result: driver.ContractQueryResult{
		Rows: []driver.ContractQueryRow{
			{Attributes: map[core.CanonicalID]any{
				"sales.order_line.order_line_id": int64(5001),
				"sales.order_line.line_number":   "1",
				"sales.order_line.quantity":      quantity1,
			}},
			{Attributes: map[core.CanonicalID]any{
				"sales.order_line.order_line_id": int64(5002),
				"sales.order_line.line_number":   "2",
				"sales.order_line.quantity":      quantity2,
			}},
		},
	}}
	contractQueryExecutors[driver.ContractQueryExecutorRuntimeKey{
		Environment: "development",
		Instance:    "sales_core.dev",
		Capability:  "database.query.postgres",
	}] = orderLinesExecutor
	headerDetailExecution, err := resolver.ExecuteHeaderDetailContractQueryInEnvironment(
		ctx,
		"development",
		"order_entry",
		"order_entry.order_lines",
		browseExecution.Result.Mapped,
		0,
		contractQueryExecutors,
	)
	if err != nil {
		t.Fatalf("header-detailを実行できません: %v", err)
	}
	if orderLinesExecutor.parameters["order_id"] != orderID {
		t.Fatalf("browse rowのorder_idがheader-detail Query inputへ引き継がれていません: %#v", orderLinesExecutor.parameters)
	}
	if len(headerDetailExecution.Execution.Result.Rows) != 2 {
		t.Fatalf("OrderLine collection=%#v", headerDetailExecution.Execution.Result.Rows)
	}

	// 6. Canonical業務Validation。header-detailで得た値を同じScenarioのValidation sourceへ渡す。
	// line_number_positiveは先頭の受注明細の行番号を評価する（Command経路の評価対象と同じ行）。
	orderLineNumbers := gateBScenarioLineNumbers(t, headerDetailExecution.Execution.Result.Rows)
	validationSource := submitOrderValidationSource{
		quantity:    gateBScenarioQuantity(t, headerDetailExecution.Execution.Result.Rows[0]),
		lineNumber:  mustRat(t, orderLineNumbers[0]),
		lineNumbers: orderLineNumbers,
	}
	for _, rule := range orderEntryValidations(t, world) {
		result, err := validationeval.Evaluate(rule, validationSource)
		if err != nil {
			t.Fatalf("Validation %sを評価できません: %v", rule.ID, err)
		}
		if result != validationeval.ResultSatisfied {
			t.Fatalf("Validation %s=%s", rule.ID, result)
		}
	}

	// 7. Authorization。Authenticationで得たsubjectをそのまま使う。
	state := map[int64]string{orderID: "pending"}
	adapters := bindingAdapterResolverFunc(func(_ context.Context, binding realization.CustomGoBinding) (capabilityinvoke.Adapter, error) {
		switch binding.Identity() {
		case "order_entry/security.authorize":
			return authorization, nil
		case "order_entry/sales.order.submit":
			return capabilityAdapterFunc(func(_ context.Context, capability core.CanonicalID, input any) (capabilityinvoke.BusinessResult, error) {
				values, ok := input.(map[string]any)
				if !ok {
					return capabilityinvoke.BusinessResult{}, fmt.Errorf("Command inputの型が不正です: %T", input)
				}
				actualOrderID, ok := values["orderId"].(int64)
				if !ok || actualOrderID != orderID {
					return capabilityinvoke.BusinessResult{}, fmt.Errorf("browseで選択したorder_idとCommand inputが一致しません: %#v", values["orderId"])
				}
				state[actualOrderID] = "submitted"
				return capabilityinvoke.BusinessResult{
					Capability: capability,
					Mode:       capabilityinvoke.ResultOutput,
					Payload: map[string]any{
						"accepted": true,
						"orderId":  actualOrderID,
					},
				}, nil
			}), nil
		default:
			return nil, fmt.Errorf("未登録のBindingです: %s", binding.Identity())
		}
	})
	authorizationExecution, err := resolver.InvokeCapability(
		ctx,
		"order_entry",
		"security.authorize",
		adapters,
		legacyauthorization.Input{
			SubjectID: subject.SubjectID,
			Action:    "execute",
			Resource:  "order_entry.submit_order",
		},
	)
	if err != nil {
		t.Fatalf("Authorizationを実行できません: %v", err)
	}
	if authorizationExecution.Result.Outcome != "allowed" {
		t.Fatalf("Authorization Business Result=%#v", authorizationExecution.Result)
	}

	// 8. Command。Patternを解決してから、同じorder_idをCapability inputへ渡す。
	commandResolution, err := resolver.ResolvePattern("order_entry", "order_entry.submit_order")
	if err != nil {
		t.Fatalf("command Patternを解決できません: %v", err)
	}
	if commandResolution.Capability == nil || commandResolution.Capability.ID != "sales.order.submit" {
		t.Fatalf("command PatternのCapability=%#v", commandResolution.Capability)
	}
	commandExecution, err := resolver.InvokeCapability(
		ctx,
		"order_entry",
		commandResolution.Capability.ID,
		adapters,
		map[string]any{"orderId": orderID},
	)
	if err != nil {
		t.Fatalf("Commandを実行できません: %v", err)
	}
	if commandExecution.Result.Mode != capabilityinvoke.ResultOutput {
		t.Fatalf("Command Business Result=%#v", commandExecution.Result)
	}

	// 9. Observation。Command acknowledgementとは別の明示readとして状態を観測する。
	observation, err := driver.ExecuteObservationRead(
		ctx,
		"synthetic.order_state.read",
		driver.ObservationEvidenceSynthetic,
		func(context.Context) (any, error) {
			return state[orderID], nil
		},
	)
	if err != nil {
		t.Fatalf("Observation readを実行できません: %v", err)
	}
	if observation.Nature != driver.ObservationEvidenceSynthetic || observation.Result != "submitted" {
		t.Fatalf("Observation Evidence=%#v", observation)
	}
}

func gateBScenarioMappedInt64(t *testing.T, mapped queryresult.MappedResult, row int, attribute core.CanonicalID) int64 {
	t.Helper()
	if row < 0 || row >= len(mapped.Rows) {
		t.Fatalf("mapped row %dがありません", row)
	}
	for columnIndex, column := range mapped.Columns {
		if column.Attribute != attribute {
			continue
		}
		value, ok := mapped.Rows[row][columnIndex].(int64)
		if !ok {
			t.Fatalf("Semantic attribute %sの値型が不正です: %T", attribute, mapped.Rows[row][columnIndex])
		}
		return value
	}
	t.Fatalf("Semantic attribute %sがmapped resultにありません", attribute)
	return 0
}

func gateBScenarioQuantity(t *testing.T, row driver.ContractQueryRow) *big.Rat {
	t.Helper()
	value, ok := row.Attributes["sales.order_line.quantity"].(*big.Rat)
	if !ok || value == nil {
		t.Fatalf("OrderLine quantityの値型が不正です: %T", row.Attributes["sales.order_line.quantity"])
	}
	return new(big.Rat).Set(value)
}

func gateBScenarioLineNumbers(t *testing.T, rows []driver.ContractQueryRow) []string {
	t.Helper()
	values := make([]string, len(rows))
	for i, row := range rows {
		value, ok := row.Attributes["sales.order_line.line_number"].(string)
		if !ok || value == "" {
			t.Fatalf("OrderLine line_numberの値型が不正です: row=%d value=%#v", i, row.Attributes["sales.order_line.line_number"])
		}
		values[i] = value
	}
	return values
}
