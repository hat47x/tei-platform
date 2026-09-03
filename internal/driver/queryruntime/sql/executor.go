package sqlquery

import (
	"context"
	"fmt"

	"tei/internal/database"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	"tei/internal/environment"
	"tei/internal/realization"
)

// RawExecutionは、provider固有SQL runnerが物理Databaseから観測したraw resultと、
// runtime parameter値を含まない非secretな実行Evidenceを保持する。
type RawExecution struct {
	Result  queryresult.RawResult
	Payload any
}

// Runnerは、provider固有のSQL実行だけを担当する。
//
// SQLの意味、Contract inputとの対応、raw列のSemantic mappingは担当しない。
// parametersのkeyはSQL realizationで明示された物理parameter名であり、
// RunnerがContract input名やSemantic attribute名から補完してはならない。
type Runner interface {
	ExecuteSQL(context.Context, string, map[string]any) (RawExecution, error)
}

// Executorは、Database Contract Queryを明示SQL realizationへ接続する共通adapterである。
//
// provider一致だけで別targetのrealizationを選ばず、target + providerに完全一致する
// realizationが一つだけ存在する場合に限って実行する。物理実行は注入されたRunnerへ委譲する。
type Executor struct {
	Provider     environment.DatabaseProvider
	Realizations []realization.ContractQuerySQLRealization
	Runner       Runner
}

var _ driver.ContractQueryExecutor = Executor{}
var _ driver.DatabaseProviderBoundContractQueryExecutor = Executor{}

// DatabaseProviderは、このSQL Executorが実行できるDatabase providerを返す。
// Environment runtime compositionは、この値をDatabase instanceのproviderと照合する。
func (e Executor) DatabaseProvider() environment.DatabaseProvider {
	return e.Provider
}

func (e Executor) ExecuteContractQuery(
	ctx context.Context,
	target realization.ContractQueryTarget,
	contract database.QueryContract,
	parameters map[string]any,
) (driver.ContractQueryResult, error) {
	if !e.Provider.Known() || e.Provider == environment.ProviderGeneric {
		return driver.ContractQueryResult{}, fmt.Errorf("Contract Query SQL ExecutorのDatabase provider %qは実行に使用できません", e.Provider)
	}
	if e.Runner == nil {
		return driver.ContractQueryResult{}, fmt.Errorf("Contract Query SQL Executor %sにSQL Runnerが指定されていません", e.Provider)
	}
	if !target.Valid() {
		return driver.ContractQueryResult{}, fmt.Errorf("Contract Query SQL Executorへ渡されたtarget identityが不完全です")
	}

	realizationDef, err := e.resolveRealization(target)
	if err != nil {
		return driver.ContractQueryResult{}, err
	}
	if err := realizationDef.ValidateContract(contract); err != nil {
		return driver.ContractQueryResult{}, fmt.Errorf("Contract Query SQL realization %sを実行できません: %w", realizationDef.Identity(), err)
	}

	physicalParameters, err := realizationDef.MapRuntimeParameters(parameters)
	if err != nil {
		return driver.ContractQueryResult{}, err
	}
	raw, err := e.Runner.ExecuteSQL(ctx, realizationDef.Source, physicalParameters)
	if err != nil {
		return driver.ContractQueryResult{}, fmt.Errorf("Contract Query SQL realization %sの物理SQL実行に失敗しました: %w", realizationDef.Identity(), err)
	}

	result, err := driver.MapContractQuerySQLResult(realizationDef, raw.Result)
	if err != nil {
		return driver.ContractQueryResult{}, fmt.Errorf("Contract Query SQL realization %sのraw resultをSemantic rowへ変換できません: %w", realizationDef.Identity(), err)
	}
	result.Payload = raw.Payload
	return result, nil
}

func (e Executor) resolveRealization(target realization.ContractQueryTarget) (realization.ContractQuerySQLRealization, error) {
	var matched *realization.ContractQuerySQLRealization
	for i := range e.Realizations {
		candidate := e.Realizations[i]
		if candidate.Provider != e.Provider || candidate.Target != target {
			continue
		}
		if matched != nil {
			return realization.ContractQuerySQLRealization{}, fmt.Errorf(
				"Contract Query %s / provider %sにSQL realizationが複数あります",
				target.Identity(),
				e.Provider,
			)
		}
		copy := candidate
		matched = &copy
	}
	if matched == nil {
		return realization.ContractQuerySQLRealization{}, fmt.Errorf(
			"Contract Query %s / provider %sにSQL realizationがありません",
			target.Identity(),
			e.Provider,
		)
	}
	return *matched, nil
}
