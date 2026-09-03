package driver

import (
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/realization"
)

// ResolvedHeaderDetailContractQueryは、header-detail Patternをcollection取得用
// Database Contract Queryへ接続する独立realizationの解決結果を表す。
//
// Pattern自身はRelation-onlyのまま維持し、DatasourceとContract Queryは
// HeaderDetailContractQueryBindingを正本として解決する。RelationKey、physical FK、
// Persistence Mapping、provider情報から不足したidentityを補完しない。
type ResolvedHeaderDetailContractQuery struct {
	Pattern    application.Pattern
	Relation   ResolvedRelation
	Datasource application.DatabaseRequirement
	Database   database.Database
	Contract   database.Contract
	Query      database.QueryContract
	Binding    realization.HeaderDetailContractQueryBinding
}

// ResolveHeaderDetailContractQueryは、Application + Pattern identityに完全一致する
// HeaderDetailContractQueryBindingを一つだけ選び、logical DatasourceからDatabase Contract Queryまでを解決する。
func (r Resolver) ResolveHeaderDetailContractQuery(
	applicationID, patternID core.CanonicalID,
) (ResolvedHeaderDetailContractQuery, error) {
	patternResolution, err := r.ResolvePattern(applicationID, patternID)
	if err != nil {
		return ResolvedHeaderDetailContractQuery{}, err
	}
	if patternResolution.Pattern.Kind != application.PatternHeaderDetail {
		return ResolvedHeaderDetailContractQuery{}, fmt.Errorf("Pattern %sはheader-detailではありません: %s", patternResolution.Pattern.ID, patternResolution.Pattern.Kind)
	}
	if patternResolution.Relation == nil {
		return ResolvedHeaderDetailContractQuery{}, fmt.Errorf("header-detail Pattern %sのRelationが解決されていません", patternResolution.Pattern.ID)
	}

	var binding realization.HeaderDetailContractQueryBinding
	bindingCount := 0
	for _, candidate := range r.world.HeaderDetailContractQueryBindings {
		if candidate.Application != applicationID || candidate.Pattern != patternID {
			continue
		}
		binding = candidate
		bindingCount++
	}
	if bindingCount == 0 {
		return ResolvedHeaderDetailContractQuery{}, fmt.Errorf("header-detail Pattern %sのContract Query bindingがありません", patternID)
	}
	if bindingCount > 1 {
		return ResolvedHeaderDetailContractQuery{}, fmt.Errorf("header-detail Pattern %sのContract Query bindingが複数あります: %d件", patternID, bindingCount)
	}
	if !binding.Valid() {
		return ResolvedHeaderDetailContractQuery{}, fmt.Errorf("header-detail Pattern %sのContract Query bindingが不完全です", patternID)
	}

	datasource, err := readableDatasource(patternResolution.Application, binding.Datasource)
	if err != nil {
		return ResolvedHeaderDetailContractQuery{}, fmt.Errorf("header-detail Pattern %sのDatasource %sを解決できません: %w", patternID, binding.Datasource, err)
	}
	if datasource.TargetDatabase != binding.Target.Database {
		return ResolvedHeaderDetailContractQuery{}, fmt.Errorf(
			"header-detail Pattern %sのDatasource %sはDatabase %sを参照しますが、Contract Query targetは%sです",
			patternID,
			binding.Datasource,
			datasource.TargetDatabase,
			binding.Target.Database,
		)
	}

	db, err := r.databaseByID(binding.Target.Database)
	if err != nil {
		return ResolvedHeaderDetailContractQuery{}, fmt.Errorf("header-detail Pattern %sのDatabase %sを解決できません: %w", patternID, binding.Target.Database, err)
	}
	requirement, err := requiredContract(datasource, binding.Target.Contract)
	if err != nil {
		return ResolvedHeaderDetailContractQuery{}, fmt.Errorf("header-detail Pattern %sのContract %sを利用できません: %w", patternID, binding.Target.Contract, err)
	}
	contract, err := contractByID(db, binding.Target.Contract)
	if err != nil {
		return ResolvedHeaderDetailContractQuery{}, fmt.Errorf("header-detail Pattern %sのContract %sをDatabase %sで解決できません: %w", patternID, binding.Target.Contract, db.ID, err)
	}
	if contract.Version.Major != requirement.CompatibleMajor {
		return ResolvedHeaderDetailContractQuery{}, fmt.Errorf(
			"header-detail Pattern %sが要求するContract %sの互換majorは%dですが、Database %sのContractはv%d.%dです",
			patternID,
			binding.Target.Contract,
			requirement.CompatibleMajor,
			db.ID,
			contract.Version.Major,
			contract.Version.Minor,
		)
	}
	queryContract, err := contractQueryByName(contract, binding.Target.Query)
	if err != nil {
		return ResolvedHeaderDetailContractQuery{}, fmt.Errorf("header-detail Pattern %sのContract Query %sを解決できません: %w", patternID, binding.Target.Query, err)
	}

	return ResolvedHeaderDetailContractQuery{
		Pattern:    patternResolution.Pattern,
		Relation:   *patternResolution.Relation,
		Datasource: datasource,
		Database:   db,
		Contract:   contract,
		Query:      queryContract,
		Binding:    binding,
	}, nil
}
