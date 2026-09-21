package driver_test

import (
	"context"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
)

// S8は、S5で受理した第二Application `repair_intake` と、S7で受理したEnvironment bindingが、
// 実際のReference Workspaceから通常のDriver経路で解決・実行できるかを見る。
//
// ここで使うExecutorはtest doubleである。実際のDatabaseへ接続した証拠ではなく、
// Permission / Resource / credentialも与えない（S7が残した `RUNTIME_AUTHORITY_NOT_GRANTED`）。
func repairIntakeMappedResult() queryresult.MappedResult {
	return queryresult.MappedResult{
		Columns: []queryresult.Column{
			{Raw: "customer_id", Attribute: "repair_intake.intake.customer_id"},
		},
		Rows: [][]any{{int64(42)}},
	}
}

func TestSecondApplicationReferencePatternResolvesThroughTheAcceptedEnvironmentBinding(t *testing.T) {
	world := referenceWorld(t)
	executor := &recordingContractQueryExecutor{
		result: driver.ContractQueryResult{
			Rows: []driver.ContractQueryRow{
				{Attributes: map[core.CanonicalID]any{"customer.customer.customer_id": int64(42)}},
			},
		},
	}
	registry := driver.ContractQueryExecutorMap{
		{Environment: "development", Instance: "master_data.dev", Capability: "database.query.oracle"}: executor,
	}

	result, err := driver.NewResolver(world).ExecuteReferenceContractQueryInEnvironment(
		context.Background(),
		"development",
		"repair_intake",
		"repair_intake.customer_reference",
		repairIntakeMappedResult(),
		0,
		registry,
	)
	if err != nil {
		t.Fatalf("受理済みの第二Applicationを通常のDriver経路で実行できません: %v", err)
	}

	// datasource aliasは、S7で受理したEnvironment bindingから解決される。
	if result.Environment.ID != "development" {
		t.Fatalf("environment=%q", result.Environment.ID)
	}
	if result.Executor.Instance.ID != "master_data.dev" {
		t.Fatalf("alias customer_masterの解決先=%q want master_data.dev", result.Executor.Instance.ID)
	}
	// executor bindingは(environment, instance, database, contract, query)で識別され、
	// Applicationでは識別されない。したがってorder_entryのために書かれた既存bindingが
	// そのまま効き、第二Applicationのために新しいEnvironment sourceは要らなかった。
	if result.Executor.Binding.Environment != "development" || result.Executor.Capability.Capability.ID != "database.query.oracle" {
		t.Fatalf("executor binding=%#v", result.Executor)
	}
	if target := result.Executor.Binding.Target; target.Contract != "master_data.customer" || target.Query != "find_by_id" {
		t.Fatalf("contract query target=%#v", target)
	}

	// Relation keyは宣言どおりに渡る。名前の一致から補完しない。
	if got := executor.parameters["id"]; got != int64(42) {
		t.Fatalf("Contract Query parameters=%#v", executor.parameters)
	}
	if len(result.Execution.Result.Rows) != 1 {
		t.Fatalf("rows=%#v", result.Execution.Result.Rows)
	}
	if got := result.Execution.Result.Rows[0].Attributes["customer.customer.customer_id"]; got != int64(42) {
		t.Fatalf("結果が共有Customerのattributeへ戻りません: %#v", result.Execution.Result.Rows[0])
	}
}

// productionでは、S7 bindingによりaliasはmaster_data.prodへ解決するが、current Reference上に
// production側のContract Query executor bindingが一件も無いため実行まで進めない。
// これは第二Application固有ではなく、order_entryでも同じである（S8実行記録）。
//
// ここで確認したいのは、その不在が**別Environmentのexecutorへ暗黙に落ちない**ことである。
func TestSecondApplicationReferencePatternDoesNotFallBackToAnotherEnvironmentExecutor(t *testing.T) {
	world := referenceWorld(t)
	executor := &recordingContractQueryExecutor{
		result: driver.ContractQueryResult{
			Rows: []driver.ContractQueryRow{
				{Attributes: map[core.CanonicalID]any{"customer.customer.customer_id": int64(42)}},
			},
		},
	}
	// developmentのexecutorだけを登録する。productionの実行がこれを使ってはいけない。
	registry := driver.ContractQueryExecutorMap{
		{Environment: "development", Instance: "master_data.dev", Capability: "database.query.oracle"}: executor,
	}

	_, err := driver.NewResolver(world).ExecuteReferenceContractQueryInEnvironment(
		context.Background(),
		"production",
		"repair_intake",
		"repair_intake.customer_reference",
		repairIntakeMappedResult(),
		0,
		registry,
	)
	if err == nil {
		t.Fatal("production側のexecutor bindingが無いのに実行できています")
	}
	if executor.parameters != nil {
		t.Fatalf("別Environmentのexecutorが呼ばれました: %#v", executor.parameters)
	}
	// 失敗はaliasの解決ではなくexecutor bindingの不在であり、production instanceまでは解決している。
	for _, want := range []string{"master_data.prod", "executor Binding"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("失敗理由に %q が現れません: %v", want, err)
		}
	}
}

// 上のtestはexecutor **Binding**（source側）の不在で止まる。runtime registryのlookupが
// Environment / instanceまで正確かは別の性質なので、bindingが解決する状況で確かめる。
func TestSecondApplicationReferencePatternRequiresAnExactRuntimeExecutor(t *testing.T) {
	world := referenceWorld(t)
	cases := map[string]driver.ContractQueryExecutorRuntimeKey{
		"another environment": {Environment: "production", Instance: "master_data.dev", Capability: "database.query.oracle"},
		"another instance":    {Environment: "development", Instance: "master_data.prod", Capability: "database.query.oracle"},
		"another capability":  {Environment: "development", Instance: "master_data.dev", Capability: "database.query.postgres"},
	}
	for name, key := range cases {
		t.Run(name, func(t *testing.T) {
			executor := &recordingContractQueryExecutor{
				result: driver.ContractQueryResult{
					Rows: []driver.ContractQueryRow{
						{Attributes: map[core.CanonicalID]any{"customer.customer.customer_id": int64(42)}},
					},
				},
			}
			_, err := driver.NewResolver(world).ExecuteReferenceContractQueryInEnvironment(
				context.Background(),
				"development",
				"repair_intake",
				"repair_intake.customer_reference",
				repairIntakeMappedResult(),
				0,
				driver.ContractQueryExecutorMap{key: executor},
			)
			if err == nil {
				t.Fatal("一致しないruntime executorで実行できています")
			}
			if executor.parameters != nil {
				t.Fatalf("一致しないexecutorが呼ばれました: %#v", executor.parameters)
			}
		})
	}
}

// Environment bindingが無ければ実行できない。S7の受理が効いていることの裏返しである。
func TestSecondApplicationReferencePatternFailsClosedWithoutTheEnvironmentBinding(t *testing.T) {
	world := referenceWorld(t)
	for i := range world.Environments {
		if world.Environments[i].ID != "development" {
			continue
		}
		kept := world.Environments[i].Bindings[:0:0]
		for _, binding := range world.Environments[i].Bindings {
			if binding.Application == "repair_intake" {
				continue
			}
			kept = append(kept, binding)
		}
		world.Environments[i].Bindings = kept
	}

	registry := driver.ContractQueryExecutorMap{
		{Environment: "development", Instance: "master_data.dev", Capability: "database.query.oracle"}: &recordingContractQueryExecutor{},
	}
	_, err := driver.NewResolver(world).ExecuteReferenceContractQueryInEnvironment(
		context.Background(),
		"development",
		"repair_intake",
		"repair_intake.customer_reference",
		repairIntakeMappedResult(),
		0,
		registry,
	)
	if err == nil {
		t.Fatal("bindingが無いのに実行できています")
	}
}
