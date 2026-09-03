package validation

import "tei/internal/ir"

// validateQueryResultMappingsは、Validateから呼ばれるdata access系Validationの共通入口である。
//
// もともとのApplication Query result mapping検査は専用関数へ分離し、reference Patternから
// Database Contract Query、SQL realizationへ続く追加契約も同じ通常Validation経路で検査する。
// 各検査の責務そのものは混ぜず、この関数では実行順だけを束ねる。
func validateQueryResultMappings(w ir.World) []Diagnostic {
	out := validateApplicationQueryResultMappings(w)
	out = append(out, validateReferenceDataAccessContracts(w)...)
	return out
}

// validateReferenceDataAccessContractsは、reference PatternからDatabase Contract Queryを経て
// 物理SQL realizationへ到達する経路で必要なCanonical整合性検査をまとめて実行する。
//
// Semantic Relation key、Database Contract Query signature、Application側の呼び出し、
// 三層をまたぐ対応関係、SQL realizationの順に検査する。各Validationの責務は分離したまま保ち、
// 名前や型、物理情報から不足した関係を補完しない。
func validateReferenceDataAccessContracts(w ir.World) []Diagnostic {
	var out []Diagnostic
	out = append(out, ValidateSemanticRelationKeys(w)...)
	out = append(out, ValidateDatabaseContractQuerySignatures(w)...)
	out = append(out, ValidateReferenceContractQueryInvocations(w)...)
	out = append(out, ValidateReferenceContractQueryCrossLayer(w)...)
	out = append(out, ValidateContractQuerySQLRealizations(w)...)
	return out
}
