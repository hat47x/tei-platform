package validation

import (
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/subject"
)

// ValidateContractQuerySQLRealizationsは、Database-owned Contract QueryのSQL realizationについて、
// target Database / Contract / Query、signature、result Semantic Dataとの参照整合性を確認する。
//
// 物理SQL本文を解析してparameterや列の意味を推測せず、authoringで明示されたmappingだけを検査する。
func ValidateContractQuerySQLRealizations(w ir.World) []Diagnostic {
	dataByID := map[core.CanonicalID]semantic.Data{}
	for _, namespace := range w.Semantics {
		for _, data := range namespace.Data {
			dataByID[data.ID] = data
		}
	}

	seen := map[string]struct{}{}
	var out []Diagnostic
	for _, realizationDef := range w.ContractQuerySQLRealizations {
		identity := realizationDef.Identity()
		realizationSubject := subject.Subject{Kind: "contract-query-sql-realization", ID: identity}
		if _, duplicate := seen[identity]; duplicate {
			out = append(out, errFor(
				realizationSubject,
				"DUPLICATE_CONTRACT_QUERY_SQL_REALIZATION",
				"Contract Query SQL realization %sが複数定義されています",
				identity,
			))
			continue
		}
		seen[identity] = struct{}{}

		db, ok, ambiguous := databaseForContractQueryRealization(w, realizationDef.Target.Database)
		if ambiguous {
			out = append(out, errFor(
				realizationSubject,
				"AMBIGUOUS_CONTRACT_QUERY_SQL_REALIZATION_DATABASE",
				"Contract Query SQL realization %sが参照するDatabase %sは複数定義されています",
				identity,
				realizationDef.Target.Database,
			))
			continue
		}
		if !ok {
			out = append(out, errFor(
				realizationSubject,
				"UNKNOWN_CONTRACT_QUERY_SQL_REALIZATION_DATABASE",
				"Contract Query SQL realization %sは存在しないDatabase %sを参照しています",
				identity,
				realizationDef.Target.Database,
			))
			continue
		}

		contract, ok, ambiguous := contractForSQLRealization(db, realizationDef.Target.Contract)
		if ambiguous {
			out = append(out, errFor(
				realizationSubject,
				"AMBIGUOUS_CONTRACT_QUERY_SQL_REALIZATION_CONTRACT",
				"Contract Query SQL realization %sが参照するContract %sはDatabase %sで複数定義されています",
				identity,
				realizationDef.Target.Contract,
				db.ID,
			))
			continue
		}
		if !ok {
			out = append(out, errFor(
				realizationSubject,
				"UNKNOWN_CONTRACT_QUERY_SQL_REALIZATION_CONTRACT",
				"Contract Query SQL realization %sはDatabase %sに存在しないContract %sを参照しています",
				identity,
				db.ID,
				realizationDef.Target.Contract,
			))
			continue
		}

		queryContract, ok, ambiguous := queryForSQLRealization(contract, realizationDef.Target.Query)
		if ambiguous {
			out = append(out, errFor(
				realizationSubject,
				"AMBIGUOUS_CONTRACT_QUERY_SQL_REALIZATION_QUERY",
				"Contract Query SQL realization %sが参照するQuery %sはContract %sで複数定義されています",
				identity,
				realizationDef.Target.Query,
				contract.ID,
			))
			continue
		}
		if !ok {
			out = append(out, errFor(
				realizationSubject,
				"UNKNOWN_CONTRACT_QUERY_SQL_REALIZATION_QUERY",
				"Contract Query SQL realization %sはContract %sに存在しないQuery %sを参照しています",
				identity,
				contract.ID,
				realizationDef.Target.Query,
			))
			continue
		}

		if err := realizationDef.ValidateContract(queryContract); err != nil {
			out = append(out, errFor(
				realizationSubject,
				"INVALID_CONTRACT_QUERY_SQL_REALIZATION",
				"Contract Query SQL realization %sがQuery signatureと整合しません: %v",
				identity,
				err,
			))
			continue
		}

		resultData, exists := dataByID[queryContract.Result]
		if !exists {
			// result Data自体の欠如はDatabase Contract Query signature Validationへ委ね、
			// ここでは派生した二次診断を増やさない。
			continue
		}
		attributes := map[core.CanonicalID]struct{}{}
		for _, attribute := range resultData.Attributes {
			attributes[attribute.ID] = struct{}{}
		}
		for _, mapping := range realizationDef.ResultColumns {
			if _, exists := attributes[mapping.Attribute]; exists {
				continue
			}
			out = append(out, errFor(
				realizationSubject,
				"CONTRACT_QUERY_SQL_RESULT_ATTRIBUTE_OUTSIDE_RESULT_DATA",
				"Contract Query SQL realization %sのraw列 %qは、Query result Data %sに属さないSemantic attribute %sへmappingされています",
				identity,
				mapping.Column,
				queryContract.Result,
				mapping.Attribute,
			))
		}
	}
	return out
}

func databaseForContractQueryRealization(w ir.World, id core.CanonicalID) (database.Database, bool, bool) {
	var found database.Database
	count := 0
	for _, db := range w.Databases {
		if db.ID == id {
			found = db
			count++
		}
	}
	return found, count == 1, count > 1
}

func contractForSQLRealization(db database.Database, id core.CanonicalID) (database.Contract, bool, bool) {
	var found database.Contract
	count := 0
	for _, contract := range db.Contracts {
		if contract.ID == id {
			found = contract
			count++
		}
	}
	return found, count == 1, count > 1
}

func queryForSQLRealization(contract database.Contract, name core.LocalName) (database.QueryContract, bool, bool) {
	var found database.QueryContract
	count := 0
	for _, queryContract := range contract.Queries {
		if queryContract.Name == string(name) {
			found = queryContract
			count++
		}
	}
	return found, count == 1, count > 1
}
