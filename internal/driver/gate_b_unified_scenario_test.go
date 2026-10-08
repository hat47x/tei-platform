package driver_test

import (
	"context"
	"fmt"
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

type gateBUnifiedScenarioEvidence struct {
	Nature             string
	Steps              []string
	Authentication     capabilityinvoke.BusinessResult
	BrowseRowCount     int
	RecordData         core.CanonicalID
	Customer           driver.ContractQueryResult
	OrderLines         driver.ContractQueryResult
	Validation         map[core.CanonicalID]validationeval.Result
	Authorization      capabilityinvoke.BusinessResult
	Command            capabilityinvoke.BusinessResult
	Observation        driver.ObservationReadExecution
	ExpectedState      string
	ObservationMatched bool
}

func TestGateBUnifiedSyntheticScenarioReplaysOrderEntryWithoutUI(t *testing.T) {
	world := referenceWorld(t)
	resolver := driver.NewResolver(world)
	ctx := context.Background()

	store, err := legacyauth.LoadFixtureStore(referenceLegacyAuthFixture(t))
	if err != nil {
		t.Fatalf("legacy authentication fixtureを読み込めません: %v", err)
	}
	authorizationRule, err := legacyauthorization.LoadFixtureRule(filepath.Join(filepath.Dir(referenceLegacyAuthFixture(t)), "authorization.json"))
	if err != nil {
		t.Fatalf("legacy authorization fixtureを読み込めません: %v", err)
	}

	now := time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC)
	authenticationAdapter := legacyauth.Adapter{
		Store: store,
		Verifier: driverCredentialVerifierFunc(func(_ context.Context, ref, presented string) (bool, error) {
			if ref != "credential-ref://legacy/order-editor" {
				return false, fmt.Errorf("未登録のcredential参照です: %s", ref)
			}
			return presented == "editor-secret", nil
		}),
		Now: func() time.Time { return now },
	}
	authorizationAdapter := legacyauthorization.Adapter{Store: store, Rule: authorizationRule}
	state := map[int64]string{1001: "pending"}
	adapters := unifiedGateScenarioAdapters(authenticationAdapter, authorizationAdapter, state)

	evidence := gateBUnifiedScenarioEvidence{
		Nature:        "synthetic",
		Validation:    map[core.CanonicalID]validationeval.Result{},
		ExpectedState: "submitted",
	}

	authentication, err := resolver.InvokeCapability(
		ctx,
		"order_entry",
		"security.authenticate",
		adapters,
		legacyauth.Input{LoginID: "legacy-order-editor", Credential: "editor-secret"},
	)
	if err != nil {
		t.Fatalf("Authenticationを実行できません: %v", err)
	}
	if authentication.Result.Outcome != "authenticated" {
		t.Fatalf("Authentication result=%#v", authentication.Result)
	}
	subject, ok := authentication.Result.Payload.(legacyauth.AuthenticatedSubject)
	if !ok || subject.SubjectID == "" {
		t.Fatalf("Authentication結果にsubjectがありません: %#v", authentication.Result)
	}
	evidence.Authentication = authentication.Result
	evidence.Steps = append(evidence.Steps, "authentication")

	browse, err := resolver.ExecuteBrowseQuery(
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
				return driver.BrowseQueryResult{}, fmt.Errorf("browseのCanonical解決結果が変わっています: query=%s datasource=%#v", q.ID, datasource)
			}
			if len(parameters) != 0 {
				return driver.BrowseQueryResult{}, fmt.Errorf("統合Scenarioで未定義のbrowse parameterを受け取りました: %#v", parameters)
			}
			return driver.BrowseQueryResult{Raw: queryresult.RawResult{
				Columns: []string{"total_amount", "status", "ordered_at", "customer_id", "order_id"},
				Rows: [][]any{{
					2500.50,
					"OPEN",
					time.Date(2026, 9, 3, 10, 30, 0, 0, time.UTC),
					int64(42),
					int64(1001),
				}},
			}}, nil
		}),
		nil,
	)
	if err != nil {
		t.Fatalf("browseを実行できません: %v", err)
	}
	if len(browse.Result.Mapped.Rows) != 1 {
		t.Fatalf("browse row数=%d, want 1", len(browse.Result.Mapped.Rows))
	}
	evidence.BrowseRowCount = len(browse.Result.Mapped.Rows)
	evidence.Steps = append(evidence.Steps, "browse")

	order, err := semanticMappedRow(browse.Result.Mapped, 0)
	if err != nil {
		t.Fatalf("browse結果からSemantic Order rowを取得できません: %v", err)
	}
	orderID, ok := order["sales.order.order_id"].(int64)
	if !ok || orderID != 1001 {
		t.Fatalf("同じOrder instanceを特定できません: %#v", order["sales.order.order_id"])
	}

	record, err := resolver.ResolvePattern("order_entry", "order_entry.order_record")
	if err != nil {
		t.Fatalf("record Patternを解決できません: %v", err)
	}
	if record.Data == nil || record.Data.ID != "sales.order" {
		t.Fatalf("record Data=%#v", record.Data)
	}
	if record.Datasource != nil || record.Query != nil || record.Relation != nil || record.Capability != nil {
		t.Fatalf("record Patternへ実現情報を推測しています: %#v", record)
	}
	evidence.RecordData = record.Data.ID
	evidence.Steps = append(evidence.Steps, "record")

	customerExecutor := &recordingContractQueryExecutor{result: driver.ContractQueryResult{
		Rows: []driver.ContractQueryRow{{Attributes: map[core.CanonicalID]any{
			"customer.customer.customer_id": int64(42),
			"customer.customer.name":        "Synthetic Customer",
		}}},
	}}
	customer, err := resolver.ExecuteReferenceContractQuery(
		ctx,
		"order_entry",
		"order_entry.customer_reference",
		browse.Result.Mapped,
		0,
		customerExecutor,
	)
	if err != nil {
		t.Fatalf("Customer referenceを実行できません: %v", err)
	}
	if customerExecutor.parameters["id"] != int64(42) || len(customer.Result.Rows) != 1 {
		t.Fatalf("Customer referenceの明示parameter/resultが変わっています: parameters=%#v result=%#v", customerExecutor.parameters, customer.Result)
	}
	evidence.Customer = customer.Result
	evidence.Steps = append(evidence.Steps, "reference")

	lineQuantity := mustRat(t, "2.5")
	orderLineExecutor := &recordingContractQueryExecutor{result: driver.ContractQueryResult{
		Rows: []driver.ContractQueryRow{
			{Attributes: map[core.CanonicalID]any{
				"sales.order_line.order_line_id": int64(11),
				"sales.order_line.line_number":   int64(1),
				"sales.order_line.item_code":     "ITEM-A",
				"sales.order_line.quantity":      lineQuantity,
				"sales.order_line.unit_price":    mustRat(t, "1000"),
			}},
			{Attributes: map[core.CanonicalID]any{
				"sales.order_line.order_line_id": int64(12),
				"sales.order_line.line_number":   int64(2),
				"sales.order_line.item_code":     "ITEM-B",
				"sales.order_line.quantity":      mustRat(t, "1"),
				"sales.order_line.unit_price":    mustRat(t, "500.5"),
			}},
		},
	}}
	orderLines, err := resolver.ExecuteHeaderDetailContractQuery(
		ctx,
		"order_entry",
		"order_entry.order_lines",
		browse.Result.Mapped,
		0,
		orderLineExecutor,
	)
	if err != nil {
		t.Fatalf("OrderLine header-detailを実行できません: %v", err)
	}
	if orderLineExecutor.parameters["order_id"] != orderID || len(orderLines.Result.Rows) != 2 {
		t.Fatalf("同じOrder instanceからOrderLineを辿れていません: parameters=%#v result=%#v", orderLineExecutor.parameters, orderLines.Result)
	}
	if orderLines.Result.Rows[0].Attributes["sales.order_line.quantity"] != lineQuantity {
		t.Fatalf("header-detail結果のquantityが変わっています: %#v", orderLines.Result.Rows[0].Attributes)
	}
	evidence.OrderLines = orderLines.Result
	evidence.Steps = append(evidence.Steps, "header-detail")

	// line_number_positiveは先頭の受注明細を評価する。その行番号は、header-detailで読んだ行から取る（fixtureの値を前提にしない）。
	orderLineNumbers := gateBScenarioLineNumbers(t, orderLines.Result.Rows)
	validationSource := submitOrderValidationSource{
		quantity:    lineQuantity,
		lineNumber:  mustRat(t, orderLineNumbers[0]),
		lineNumbers: []string{"1", "2"},
	}
	for _, rule := range orderEntryValidations(t, world) {
		validationID := rule.ID
		result, err := validationeval.Evaluate(rule, validationSource)
		if err != nil {
			t.Fatalf("Validation %sを評価できません: %v", validationID, err)
		}
		evidence.Validation[validationID] = result
		if result != validationeval.ResultSatisfied {
			t.Fatalf("Validation %s=%s, want satisfied", validationID, result)
		}
	}
	evidence.Steps = append(evidence.Steps, "validation")

	authorization, err := resolver.InvokeCapability(
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
	if authorization.Result.Outcome != "allowed" {
		t.Fatalf("Authorization result=%#v", authorization.Result)
	}
	evidence.Authorization = authorization.Result
	evidence.Steps = append(evidence.Steps, "authorization")

	command, err := resolver.InvokeCapability(
		ctx,
		"order_entry",
		"sales.order.submit",
		adapters,
		order,
	)
	if err != nil {
		t.Fatalf("submit Commandを実行できません: %v", err)
	}
	if command.Result.Capability != "sales.order.submit" || command.Result.Mode != capabilityinvoke.ResultOutput {
		t.Fatalf("Command Business Result=%#v", command.Result)
	}
	ack, ok := command.Result.Payload.(map[string]any)
	if !ok || ack["accepted"] != true {
		t.Fatalf("Command acknowledgement=%#v", command.Result.Payload)
	}
	if _, exists := ack["observedState"]; exists {
		t.Fatalf("Command acknowledgementへObserved stateを混入させています: %#v", ack)
	}
	evidence.Command = command.Result
	evidence.Steps = append(evidence.Steps, "command")

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
	observedState, ok := observation.Result.(string)
	if !ok {
		t.Fatalf("Observation resultの型=%T", observation.Result)
	}
	evidence.Observation = observation
	evidence.ObservationMatched = observedState == evidence.ExpectedState
	evidence.Steps = append(evidence.Steps, "observation")

	wantSteps := []string{
		"authentication",
		"browse",
		"record",
		"reference",
		"header-detail",
		"validation",
		"authorization",
		"command",
		"observation",
	}
	if fmt.Sprint(evidence.Steps) != fmt.Sprint(wantSteps) {
		t.Fatalf("Scenario順序=%v, want %v", evidence.Steps, wantSteps)
	}
	if evidence.Nature != "synthetic" || evidence.Observation.Nature != driver.ObservationEvidenceSynthetic {
		t.Fatalf("synthetic Evidenceの性質を失っています: %#v", evidence)
	}
	if !evidence.ObservationMatched || observedState != "submitted" {
		t.Fatalf("Observation=%q expected=%q matched=%v", observedState, evidence.ExpectedState, evidence.ObservationMatched)
	}
}

func semanticMappedRow(mapped queryresult.MappedResult, rowIndex int) (map[core.CanonicalID]any, error) {
	if rowIndex < 0 || rowIndex >= len(mapped.Rows) {
		return nil, fmt.Errorf("mapped row %dがありません", rowIndex)
	}
	row := mapped.Rows[rowIndex]
	if len(row) != len(mapped.Columns) {
		return nil, fmt.Errorf("mapped row %dの列数%dがcolumn数%dと一致しません", rowIndex, len(row), len(mapped.Columns))
	}
	values := make(map[core.CanonicalID]any, len(mapped.Columns))
	for i, column := range mapped.Columns {
		if column.Attribute == "" {
			return nil, fmt.Errorf("mapped column %dにSemantic attributeがありません", i)
		}
		if _, exists := values[column.Attribute]; exists {
			return nil, fmt.Errorf("Semantic attribute %sがmapped rowに複数あります", column.Attribute)
		}
		values[column.Attribute] = row[i]
	}
	return values, nil
}

func unifiedGateScenarioAdapters(
	authentication legacyauth.Adapter,
	authorization legacyauthorization.Adapter,
	state map[int64]string,
) driver.BindingAdapterResolver {
	return bindingAdapterResolverFunc(func(_ context.Context, binding realization.CustomGoBinding) (capabilityinvoke.Adapter, error) {
		switch binding.Identity() {
		case "order_entry/security.authenticate":
			return authentication, nil
		case "order_entry/security.authorize":
			return authorization, nil
		case "order_entry/sales.order.submit":
			return capabilityAdapterFunc(func(_ context.Context, capability core.CanonicalID, input any) (capabilityinvoke.BusinessResult, error) {
				order, ok := input.(map[core.CanonicalID]any)
				if !ok {
					return capabilityinvoke.BusinessResult{}, fmt.Errorf("Command inputの型が不正です: %T", input)
				}
				orderID, ok := order["sales.order.order_id"].(int64)
				if !ok {
					return capabilityinvoke.BusinessResult{}, fmt.Errorf("Command inputにsales.order.order_idがありません: %#v", order["sales.order.order_id"])
				}
				state[orderID] = "submitted"
				return capabilityinvoke.BusinessResult{
					Capability: capability,
					Mode:       capabilityinvoke.ResultOutput,
					Payload: map[string]any{
						"accepted": true,
					},
				}, nil
			}), nil
		default:
			return nil, fmt.Errorf("未登録のBindingです: %s", binding.Identity())
		}
	})
}
