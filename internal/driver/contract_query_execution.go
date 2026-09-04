package driver

import (
	"context"
	"fmt"

	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/driver/queryresult"
	"tei/internal/realization"
)

// ContractQueryRowは、Database Contract Queryの実現層が返した一つのSemantic rowを表す。
//
// AttributesのkeyはQuery result Dataに属するSemantic attributeのCanonical IDでなければならない。
// 物理column名やprovider固有のrow表現は、この境界より内側へ持ち込まない。
type ContractQueryRow struct {
	Attributes map[core.CanonicalID]any
}

// ContractQueryResultは、Database Contract Queryの正常な実行結果を表す。
//
// Rowsが空であることはExecution Errorを意味しない。Queryのresult cardinalityに応じて、
// 正常な未検出またはcontract違反としてDriverが判定する。
// Payloadにはprovider固有の非secret Evidenceを保持できるが、runtime parameter値は保存しない。
type ContractQueryResult struct {
	Rows    []ContractQueryRow
	Payload any
}

// ContractQueryExecutorは、Database-ownedなContract Queryを実現方式へ接続するprovider境界である。
//
// Driver CoreはOracle SQL、API、stored procedure等を知らない。Executorは明示された
// Database / Contract / Query target identityとruntime parameterを受け、Semantic rowへ変換済みの結果を返す。
// parameter値は一回の実行にだけ利用し、通常Evidenceへ保存しない。
type ContractQueryExecutor interface {
	ExecuteContractQuery(
		context.Context,
		realization.ContractQueryTarget,
		database.QueryContract,
		map[string]any,
	) (ContractQueryResult, error)
}

// ReferenceContractQueryExecutionは、reference Patternから実行したDatabase Contract Queryの結果を表す。
// runtime parameterは保持せず、解決済みidentityと正常結果だけを残す。
type ReferenceContractQueryExecution struct {
	Resolution ResolvedReferenceContractQuery
	Result     ContractQueryResult
}

// ExecuteReferenceContractQueryは、Semanticへ明示mapping済みの一行からContract Query parameterを準備し、
// 注入されたExecutorを通してDatabase-owned Queryを実行する。
//
// Relation key、reference invocation、Query signatureのいずれも名前や型から補完しない。
// Executorのerrorは正常な0件へ変換せずExecution Errorとして返し、正常結果のrow数は
// Query Contractのresult cardinalityに従って検査する。
func (r Resolver) ExecuteReferenceContractQuery(
	ctx context.Context,
	applicationID, patternID core.CanonicalID,
	source queryresult.MappedResult,
	rowIndex int,
	executor ContractQueryExecutor,
) (ReferenceContractQueryExecution, error) {
	if executor == nil {
		return ReferenceContractQueryExecution{}, fmt.Errorf("reference Pattern %sを実行するContract Query Executorが指定されていません", patternID)
	}

	prepared, err := r.PrepareReferenceContractQuery(applicationID, patternID, source, rowIndex)
	if err != nil {
		return ReferenceContractQueryExecution{}, err
	}

	target := realization.ContractQueryTarget{
		Database: prepared.Resolution.Database.ID,
		Contract: prepared.Resolution.Contract.ID,
		Query:    prepared.Resolution.Invocation.Query,
	}
	if !target.Valid() {
		return ReferenceContractQueryExecution{}, fmt.Errorf("reference Pattern %sのContract Query target identityが不完全です", patternID)
	}

	result, err := executor.ExecuteContractQuery(
		ctx,
		target,
		prepared.Resolution.Query,
		prepared.Parameters,
	)
	if err != nil {
		return ReferenceContractQueryExecution{}, fmt.Errorf(
			"reference Pattern %sのContract Query %sを実行できませんでした: %w",
			patternID,
			target.Identity(),
			err,
		)
	}

	if err := validateContractQueryResultCardinality(prepared.Resolution.Query, len(result.Rows)); err != nil {
		return ReferenceContractQueryExecution{}, fmt.Errorf(
			"reference Pattern %sのContract Query %s resultが契約に一致しません: %w",
			patternID,
			target.Identity(),
			err,
		)
	}
	if err := r.validateContractQuerySemanticRows(prepared.Resolution.Query, result.Rows); err != nil {
		return ReferenceContractQueryExecution{}, fmt.Errorf(
			"reference Pattern %sのContract Query %s resultをSemantic Dataとして受け取れません: %w",
			patternID,
			target.Identity(),
			err,
		)
	}

	return ReferenceContractQueryExecution{
		Resolution: prepared.Resolution,
		Result:     result,
	}, nil
}

func validateContractQueryResultCardinality(query database.QueryContract, rows int) error {
	switch query.ResultCardinality {
	case database.QueryResultOne:
		if rows != 1 {
			return fmt.Errorf("Query %sはoneを要求しますが、正常結果は%d件です", query.Name, rows)
		}
	case database.QueryResultZeroOrOne:
		if rows > 1 {
			return fmt.Errorf("Query %sはzero-or-oneを要求しますが、正常結果は%d件です", query.Name, rows)
		}
	case database.QueryResultMany:
		return nil
	default:
		return fmt.Errorf("Query %sのresult cardinality %qは実行できません", query.Name, query.ResultCardinality)
	}
	return nil
}

func (r Resolver) validateContractQuerySemanticRows(query database.QueryContract, rows []ContractQueryRow) error {
	resultData, err := r.dataByID(query.Result)
	if err != nil {
		return fmt.Errorf("Query %sのresult Data %sを解決できません: %w", query.Name, query.Result, err)
	}
	attributes := map[core.CanonicalID]struct{}{}
	for _, attribute := range resultData.Attributes {
		attributes[attribute.ID] = struct{}{}
	}

	for rowIndex, row := range rows {
		for attributeID := range row.Attributes {
			if _, exists := attributes[attributeID]; !exists {
				return fmt.Errorf(
					"Query %sのresult row %dにresult Data %sへ属さないSemantic attribute %sがあります",
					query.Name,
					rowIndex,
					query.Result,
					attributeID,
				)
			}
		}
	}
	return nil
}
