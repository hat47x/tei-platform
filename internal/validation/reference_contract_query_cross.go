package validation

import (
	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/subject"
)

type referenceCrossAttribute struct {
	Owner core.CanonicalID
	Type  core.CanonicalID
}

type resolvedReferenceCross struct {
	Relation semantic.Relation
	Query    database.QueryContract
}

// ValidateReferenceContractQueryCrossLayerは、reference Patternが明示したContract Query呼び出しについて、
// Semantic Relation key・Application-ownedなinput binding・Database Contract Query signatureを横断して検査する。
//
// 各層のidentityが欠落・重複して一意に解決できない場合は、その層自身のValidationへ診断を委ねる。
// このValidationでは、三層が一意に解決できて初めて判断できる対応関係だけを扱う。
func ValidateReferenceContractQueryCrossLayer(w ir.World) []Diagnostic {
	attributes := referenceCrossAttributes(w)
	var out []Diagnostic

	for _, app := range w.Applications {
		for _, pattern := range app.Patterns {
			if pattern.Kind != application.PatternReference || pattern.ContractQuery == nil {
				continue
			}

			resolved, ok := resolveReferenceCross(w, app, pattern)
			if !ok {
				continue
			}

			patternSubject := subject.Subject{Kind: "application-pattern", ID: string(pattern.ID)}
			invocation := *pattern.ContractQuery
			relation := resolved.Relation
			query := resolved.Query

			if len(relation.Keys) == 0 {
				out = append(out, errFor(
					patternSubject,
					"REFERENCE_CONTRACT_QUERY_RELATION_KEY_REQUIRED",
					"reference Pattern %sがRelation %sをContract Queryで辿るには、Relation keyの明示が必要です",
					pattern.ID,
					relation.ID,
				))
				continue
			}

			if !hasTypedQuerySignature(query) {
				out = append(out, errFor(
					patternSubject,
					"REFERENCE_CONTRACT_QUERY_SIGNATURE_REQUIRED",
					"reference Pattern %sが利用するContract Query %s / %sにはtyped signatureが必要です",
					pattern.ID,
					invocation.Contract,
					invocation.Query,
				))
				continue
			}

			targetKeys := map[core.CanonicalID]struct{}{}
			for _, key := range relation.Keys {
				targetKeys[key.TargetAttribute] = struct{}{}
			}

			bindingsByTarget := map[core.CanonicalID]application.ContractQueryInputBinding{}
			bindingsByInput := map[core.LocalName]application.ContractQueryInputBinding{}
			for _, binding := range invocation.Inputs {
				bindingsByTarget[binding.TargetAttribute] = binding
				bindingsByInput[binding.Input] = binding
				if _, ok := targetKeys[binding.TargetAttribute]; !ok {
					out = append(out, errFor(
						patternSubject,
						"REFERENCE_CONTRACT_QUERY_BINDING_TARGET_NOT_RELATION_KEY",
						"reference Pattern %sはtarget attribute %sをQuery input %sへ渡しますが、このattributeはRelation %sのtarget keyではありません",
						pattern.ID,
						binding.TargetAttribute,
						binding.Input,
						relation.ID,
					))
				}
			}

			for _, key := range relation.Keys {
				if _, ok := bindingsByTarget[key.TargetAttribute]; !ok {
					out = append(out, errFor(
						patternSubject,
						"REFERENCE_CONTRACT_QUERY_RELATION_KEY_NOT_BOUND",
						"reference Pattern %sではRelation %sのtarget key %sをContract Query inputへ渡すbindingがありません",
						pattern.ID,
						relation.ID,
						key.TargetAttribute,
					))
				}
			}

			queryInputs := map[core.LocalName]database.QueryInput{}
			for _, input := range query.Inputs {
				queryInputs[input.Name] = input
			}

			for _, binding := range invocation.Inputs {
				input, ok := queryInputs[binding.Input]
				if !ok {
					out = append(out, errFor(
						patternSubject,
						"REFERENCE_CONTRACT_QUERY_INPUT_NOT_IN_SIGNATURE",
						"reference Pattern %sのbinding先input %sはContract Query %s / %sのsignatureにありません",
						pattern.ID,
						binding.Input,
						invocation.Contract,
						invocation.Query,
					))
					continue
				}

				attribute, attributeFound := attributes[binding.TargetAttribute]
				if attributeFound && attribute.Type != input.Type {
					out = append(out, errFor(
						patternSubject,
						"REFERENCE_CONTRACT_QUERY_INPUT_TYPE_MISMATCH",
						"reference Pattern %sはtarget attribute %s（型 %s）をQuery input %s（型 %s）へ渡しますが、Semantic型identityが一致しません",
						pattern.ID,
						binding.TargetAttribute,
						attribute.Type,
						binding.Input,
						input.Type,
					))
				}
			}

			for _, input := range query.Inputs {
				if _, ok := bindingsByInput[input.Name]; !ok {
					out = append(out, errFor(
						patternSubject,
						"REFERENCE_CONTRACT_QUERY_INPUT_NOT_BOUND",
						"reference Pattern %sではContract Query %s / %sのinput %sへ値を渡すbindingがありません",
						pattern.ID,
						invocation.Contract,
						invocation.Query,
						input.Name,
					))
				}
			}

			if query.Result != relation.Target {
				out = append(out, errFor(
					patternSubject,
					"REFERENCE_CONTRACT_QUERY_RESULT_DATA_MISMATCH",
					"reference Pattern %sのRelation %sは%sをtargetとしますが、Contract Query %s / %sは%sを返します",
					pattern.ID,
					relation.ID,
					relation.Target,
					invocation.Contract,
					invocation.Query,
					query.Result,
				))
			}

			if !referenceQueryCardinalityCompatible(relation.Cardinality, query.ResultCardinality) {
				out = append(out, errFor(
					patternSubject,
					"REFERENCE_CONTRACT_QUERY_RESULT_CARDINALITY_MISMATCH",
					"reference Pattern %sのRelation %sはcardinality %sですが、Contract Query %s / %sのresult cardinalityは%sです",
					pattern.ID,
					relation.ID,
					relation.Cardinality,
					invocation.Contract,
					invocation.Query,
					query.ResultCardinality,
				))
			}
		}
	}

	return out
}

func referenceCrossAttributes(w ir.World) map[core.CanonicalID]referenceCrossAttribute {
	out := map[core.CanonicalID]referenceCrossAttribute{}
	for _, ns := range w.Semantics {
		for _, data := range ns.Data {
			for _, attribute := range data.Attributes {
				out[attribute.ID] = referenceCrossAttribute{Owner: data.ID, Type: attribute.TypeRef}
			}
		}
	}
	return out
}

func resolveReferenceCross(w ir.World, app application.Application, pattern application.Pattern) (resolvedReferenceCross, bool) {
	var relations []semantic.Relation
	for _, ns := range w.Semantics {
		for _, data := range ns.Data {
			for _, relation := range data.Relations {
				if relation.ID == pattern.Relation {
					relations = append(relations, relation)
				}
			}
		}
	}
	if len(relations) != 1 {
		return resolvedReferenceCross{}, false
	}

	datasource, ok := app.DatabaseByAlias(pattern.Datasource)
	if !ok {
		return resolvedReferenceCross{}, false
	}

	var databases []database.Database
	for _, candidate := range w.Databases {
		if candidate.ID == datasource.TargetDatabase {
			databases = append(databases, candidate)
		}
	}
	if len(databases) != 1 {
		return resolvedReferenceCross{}, false
	}

	invocation := pattern.ContractQuery
	var contracts []database.Contract
	for _, candidate := range databases[0].Contracts {
		if candidate.ID == invocation.Contract {
			contracts = append(contracts, candidate)
		}
	}
	if len(contracts) != 1 {
		return resolvedReferenceCross{}, false
	}

	var queries []database.QueryContract
	for _, candidate := range contracts[0].Queries {
		if candidate.Name == string(invocation.Query) {
			queries = append(queries, candidate)
		}
	}
	if len(queries) != 1 {
		return resolvedReferenceCross{}, false
	}

	return resolvedReferenceCross{Relation: relations[0], Query: queries[0]}, true
}

func hasTypedQuerySignature(query database.QueryContract) bool {
	return len(query.Inputs) > 0 || query.Result != "" || query.ResultCardinality != ""
}

func referenceQueryCardinalityCompatible(relation semantic.Cardinality, result database.QueryResultCardinality) bool {
	switch relation {
	case semantic.CardinalityOne:
		return result == database.QueryResultOne || result == database.QueryResultZeroOrOne
	case semantic.CardinalityMany:
		return result == database.QueryResultMany
	default:
		return false
	}
}
