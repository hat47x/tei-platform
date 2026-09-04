package validation

import "tei/internal/ir"

// validateQueryResultMappingsは、Validateから呼ばれるdata access系Validationの共通入口である。
//
// もともとのApplication Query result mapping検査は専用関数へ分離し、referenceやheader-detailから
// Database Contract Query、SQL realizationへ続く追加契約も同じ通常Validation経路で検査する。
// 各検査の責務そのものは混ぜず、この関数では実行順だけを束ねる。
func validateQueryResultMappings(w ir.World) []Diagnostic {
	out := validateApplicationQueryResultMappings(w)
	out = append(out, validateApplicationDataAccessContracts(w)...)
	return out
}

// validateApplicationDataAccessContractsは、Application PatternからDatabase Contract Queryを経て
// 物理realizationへ到達するdata access境界で必要なCanonical整合性検査をまとめて実行する。
//
// referenceではSemantic Relation keyを使う一方、header-detailのcollection traversalではRelationKeyを
// 暗黙必須化せず、source Semantic attributeからQuery inputへの別realizationを検査する。
// Relation Storageについても、Semantic attribute列と物理support columnを混同せず、Relationの
// source / targetとの整合性だけをこの通常Validation経路から確認する。
// 各Validationの責務は分離したまま保ち、名前や型、物理情報から不足した関係を補完しない。
func validateApplicationDataAccessContracts(w ir.World) []Diagnostic {
	var out []Diagnostic
	out = append(out, ValidateSemanticRelationKeys(w)...)
	out = append(out, ValidatePersistenceRelationStorageMappings(w)...)
	out = append(out, ValidateDatabaseContractQuerySignatures(w)...)
	out = append(out, ValidateReferenceContractQueryInvocations(w)...)
	out = append(out, ValidateReferenceContractQueryCrossLayer(w)...)
	out = append(out, ValidateHeaderDetailContractQueryBindings(w)...)
	out = append(out, ValidateContractQuerySQLRealizations(w)...)
	return out
}
