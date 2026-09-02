package driver

import (
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
)

// ResolvedReferenceContractQueryは、reference Patternが明示したDatabase Contract Queryの解決結果を表す。
// Relation keyやQuery signatureの意味解釈は行わず、Applicationが選んだidentityとDatasource境界だけを確定する。
type ResolvedReferenceContractQuery struct {
	Pattern    application.Pattern
	Datasource application.DatabaseRequirement
	Database   database.Database
	Contract   database.Contract
	Query      database.QueryContract
	Invocation application.ContractQueryInvocation
}

// ResolveReferenceContractQueryは、reference Patternが明示したContract Queryを推測なしで解決する。
// Query名、Semantic型、Database内の唯一性などから呼び出し先を補完せず、Patternのcontract-queryを正本とする。
func (r Resolver) ResolveReferenceContractQuery(
	applicationID, patternID core.CanonicalID,
) (ResolvedReferenceContractQuery, error) {
	resolved, err := r.ResolvePattern(applicationID, patternID)
	if err != nil {
		return ResolvedReferenceContractQuery{}, err
	}
	if resolved.Pattern.Kind != application.PatternReference {
		return ResolvedReferenceContractQuery{}, fmt.Errorf("Pattern %sはreferenceではありません: %s", resolved.Pattern.ID, resolved.Pattern.Kind)
	}
	if resolved.Pattern.ContractQuery == nil {
		return ResolvedReferenceContractQuery{}, fmt.Errorf("reference Pattern %sにContract Query呼び出しが定義されていません", resolved.Pattern.ID)
	}
	if resolved.Datasource == nil {
		return ResolvedReferenceContractQuery{}, fmt.Errorf("reference Pattern %sのDatasourceが解決されていません", resolved.Pattern.ID)
	}

	invocation := *resolved.Pattern.ContractQuery
	targetDatabase, err := r.databaseByID(resolved.Datasource.TargetDatabase)
	if err != nil {
		return ResolvedReferenceContractQuery{}, fmt.Errorf("reference Pattern %sのDatasource %sが参照するDatabaseを解決できません: %w", resolved.Pattern.ID, resolved.Datasource.Alias, err)
	}

	requirement, err := requiredContract(*resolved.Datasource, invocation.Contract)
	if err != nil {
		return ResolvedReferenceContractQuery{}, fmt.Errorf("reference Pattern %sのContract %sを利用できません: %w", resolved.Pattern.ID, invocation.Contract, err)
	}
	contract, err := contractByID(targetDatabase, invocation.Contract)
	if err != nil {
		return ResolvedReferenceContractQuery{}, fmt.Errorf("reference Pattern %sのContract %sをDatabase %sで解決できません: %w", resolved.Pattern.ID, invocation.Contract, targetDatabase.ID, err)
	}
	if contract.Version.Major != requirement.CompatibleMajor {
		return ResolvedReferenceContractQuery{}, fmt.Errorf(
			"reference Pattern %sが要求するContract %sの互換majorは%dですが、Database %sのContractはv%d.%dです",
			resolved.Pattern.ID,
			invocation.Contract,
			requirement.CompatibleMajor,
			targetDatabase.ID,
			contract.Version.Major,
			contract.Version.Minor,
		)
	}

	queryContract, err := contractQueryByName(contract, invocation.Query)
	if err != nil {
		return ResolvedReferenceContractQuery{}, fmt.Errorf("reference Pattern %sのContract Query %sを解決できません: %w", resolved.Pattern.ID, invocation.Query, err)
	}

	return ResolvedReferenceContractQuery{
		Pattern:    resolved.Pattern,
		Datasource: *resolved.Datasource,
		Database:   targetDatabase,
		Contract:   contract,
		Query:      queryContract,
		Invocation: invocation,
	}, nil
}

func (r Resolver) databaseByID(id core.CanonicalID) (database.Database, error) {
	var found database.Database
	count := 0
	for _, candidate := range r.world.Databases {
		if candidate.ID != id {
			continue
		}
		found = candidate
		count++
	}
	if count == 0 {
		return database.Database{}, fmt.Errorf("Database %sが見つかりません", id)
	}
	if count > 1 {
		return database.Database{}, fmt.Errorf("Database %sが複数定義されています: %d件", id, count)
	}
	return found, nil
}

func requiredContract(datasource application.DatabaseRequirement, contractID core.CanonicalID) (database.ContractRequirement, error) {
	var found database.ContractRequirement
	count := 0
	for _, requirement := range datasource.RequiredContracts {
		if requirement.ContractID != contractID {
			continue
		}
		found = requirement
		count++
	}
	if count == 0 {
		return database.ContractRequirement{}, fmt.Errorf("Datasource %sはContract %sを要求していません", datasource.Alias, contractID)
	}
	if count > 1 {
		return database.ContractRequirement{}, fmt.Errorf("Datasource %sでContract %sの要求が重複しています: %d件", datasource.Alias, contractID, count)
	}
	return found, nil
}

func contractByID(db database.Database, id core.CanonicalID) (database.Contract, error) {
	var found database.Contract
	count := 0
	for _, contract := range db.Contracts {
		if contract.ID != id {
			continue
		}
		found = contract
		count++
	}
	if count == 0 {
		return database.Contract{}, fmt.Errorf("Contract %sが見つかりません", id)
	}
	if count > 1 {
		return database.Contract{}, fmt.Errorf("Contract %sが複数定義されています: %d件", id, count)
	}
	return found, nil
}

func contractQueryByName(contract database.Contract, name core.LocalName) (database.QueryContract, error) {
	var found database.QueryContract
	count := 0
	for _, queryContract := range contract.Queries {
		if queryContract.Name != string(name) {
			continue
		}
		found = queryContract
		count++
	}
	if count == 0 {
		return database.QueryContract{}, fmt.Errorf("Contract %sにQuery %sがありません", contract.ID, name)
	}
	if count > 1 {
		return database.QueryContract{}, fmt.Errorf("Contract %sにQuery %sが複数定義されています: %d件", contract.ID, name, count)
	}
	return found, nil
}
