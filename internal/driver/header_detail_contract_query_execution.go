package driver

import (
	"context"
	"fmt"

	"tei/internal/core"
	"tei/internal/driver/queryresult"
)

// HeaderDetailContractQueryExecutionは、header-detail Patternから実行した
// collection取得用Database Contract Queryの正常結果を表す。
// runtime parameterは保持せず、解決済みidentityとSemantic rowだけを残す。
type HeaderDetailContractQueryExecution struct {
	Resolution ResolvedHeaderDetailContractQuery
	Result     ContractQueryResult
}

// ExecuteHeaderDetailContractQueryは、明示mapping済みのsource rowからparameterを準備し、
// 既存ContractQueryExecutorを通してcollection取得用Database Contract Queryを実行する。
//
// header-detail専用のDatabase実行方式は導入しない。Executor errorは空collectionへ変換せず、
// 正常な0件とExecution Errorを区別する。正常結果はQuery ContractのmanyとSemantic result Dataに従って検査する。
func (r Resolver) ExecuteHeaderDetailContractQuery(
	ctx context.Context,
	applicationID, patternID core.CanonicalID,
	source queryresult.MappedResult,
	rowIndex int,
	executor ContractQueryExecutor,
) (HeaderDetailContractQueryExecution, error) {
	if executor == nil {
		return HeaderDetailContractQueryExecution{}, fmt.Errorf("header-detail Pattern %sを実行するContract Query Executorが指定されていません", patternID)
	}

	prepared, err := r.PrepareHeaderDetailContractQuery(applicationID, patternID, source, rowIndex)
	if err != nil {
		return HeaderDetailContractQueryExecution{}, err
	}
	target := prepared.Resolution.Binding.Target
	if !target.Valid() {
		return HeaderDetailContractQueryExecution{}, fmt.Errorf("header-detail Pattern %sのContract Query target identityが不完全です", patternID)
	}

	result, err := executor.ExecuteContractQuery(ctx, target, prepared.Resolution.Query, prepared.Parameters)
	if err != nil {
		return HeaderDetailContractQueryExecution{}, fmt.Errorf(
			"header-detail Pattern %sのContract Query %sを実行できませんでした: %w",
			patternID,
			target.Identity(),
			err,
		)
	}
	if err := validateContractQueryResultCardinality(prepared.Resolution.Query, len(result.Rows)); err != nil {
		return HeaderDetailContractQueryExecution{}, fmt.Errorf(
			"header-detail Pattern %sのContract Query %s resultが契約に一致しません: %w",
			patternID,
			target.Identity(),
			err,
		)
	}
	if err := r.validateContractQuerySemanticRows(prepared.Resolution.Query, result.Rows); err != nil {
		return HeaderDetailContractQueryExecution{}, fmt.Errorf(
			"header-detail Pattern %sのContract Query %s resultをSemantic Dataとして受け取れません: %w",
			patternID,
			target.Identity(),
			err,
		)
	}

	return HeaderDetailContractQueryExecution{
		Resolution: prepared.Resolution,
		Result:     result,
	}, nil
}
