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
	SourceData core.CanonicalID
	Relation   semantic.Relation
	Query      database.QueryContract
}

// ValidateReferenceContractQueryCrossLayerは、reference Patternが明示したContract Query呼び出しについて、
// Semantic Relation key・Application-ownedなinput binding・Database Contract Query signatureを横断して検査する。
//
// 各層のidentityや参照整合性が成立しない場合は、その層自身のValidationへ診断を委ねる。
// このValidationでは、三層がそれぞれ単独では妥当で、一意に解決できて初めて判断できる対応関係だけを扱う。
func ValidateReferenceContractQueryCrossLayer(w ir.World) []Diagnostic {
	attributes := referenceCrossAttributes(w)
	semanticTypes := referenceCrossSemanticTypes(w)
	semanticData := referenceCrossSemanticData(w)
	var out []Diagnostic

	for _, app := range w.Applications {
		for _, pattern := range app.Patterns {
			if pattern.Kind != application.PatternReference || pattern.ContractQuery == nil {
				continue
			}

			resolved, ok := resolveReferenceCross(w, app, pattern)
			if !ok || !referenceInvocationReadyForCross(*pattern.ContractQuery) {
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
			if !relationKeysReadyForCross(resolved.SourceData, relation, attributes) {
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
			if !querySignatureReadyForCross(query, semanticTypes, semanticData) {
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

				attribute := attributes[binding.TargetAttribute]
				if attribute.Type != input.Type {
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

func referenceCrossSemanticTypes(w ir.World) map[core.CanonicalID]struct{} {
	out := map[core.CanonicalID]struct{}{}
	for _, ns := range w.Semantics {
		for _, typ := range ns.Types {
			out[typ.ID] = struct{}{}
		}
	}
	return out
}

func referenceCrossSemanticData(w ir.World) map[core.CanonicalID]struct{} {
	out := map[core.CanonicalID]struct{}{}
	for _, ns := range w.Semantics {
		for _, data := range ns.Data {
			out[data.ID] = struct{}{}
		}
	}
	return out
}

func resolveReferenceCross(w ir.World, app application.Application, pattern application.Pattern) (resolvedReferenceCross, bool) {
	var relations []resolvedReferenceCross
	for _, ns := range w.Semantics {
		for _, data := range ns.Data {
			for _, relation := range data.Relations {
				if relation.ID == pattern.Relation {
					relations = append(relations, resolvedReferenceCross{SourceData: data.ID, Relation: relation})
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
	invocation := pattern.ContractQuery
	requirements := requiredContractRequirements(datasource, invocation.Contract)
	if len(requirements) != 1 {
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

	var contracts []database.Contract
	for _, candidate := range databases[0].Contracts {
		if candidate.ID == invocation.Contract {
			contracts = append(contracts, candidate)
		}
	}
	if len(contracts) != 1 || contracts[0].Version.Major != requirements[0].CompatibleMajor {
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

	resolved := relations[0]
	resolved.Query = queries[0]
	return resolved, true
}

func referenceInvocationReadyForCross(invocation application.ContractQueryInvocation) bool {
	if invocation.Contract == "" || invocation.Query == "" {
		return false
	}
	seenTargets := map[core.CanonicalID]struct{}{}
	seenInputs := map[core.LocalName]struct{}{}
	for _, binding := range invocation.Inputs {
		if binding.TargetAttribute == "" || binding.Input == "" {
			return false
		}
		if _, duplicate := seenTargets[binding.TargetAttribute]; duplicate {
			return false
		}
		seenTargets[binding.TargetAttribute] = struct{}{}
		if _, duplicate := seenInputs[binding.Input]; duplicate {
			return false
		}
		seenInputs[binding.Input] = struct{}{}
	}
	return true
}

func relationKeysReadyForCross(sourceData core.CanonicalID, relation semantic.Relation, attributes map[core.CanonicalID]referenceCrossAttribute) bool {
	seenPairs := map[string]struct{}{}
	seenTargets := map[core.CanonicalID]struct{}{}
	for _, key := range relation.Keys {
		if key.SourceAttribute == "" || key.TargetAttribute == "" {
			return false
		}
		pair := string(key.SourceAttribute) + "\x00" + string(key.TargetAttribute)
		if _, duplicate := seenPairs[pair]; duplicate {
			return false
		}
		seenPairs[pair] = struct{}{}
		if _, duplicate := seenTargets[key.TargetAttribute]; duplicate {
			return false
		}
		seenTargets[key.TargetAttribute] = struct{}{}

		source, sourceFound := attributes[key.SourceAttribute]
		target, targetFound := attributes[key.TargetAttribute]
		if !sourceFound || !targetFound || source.Owner != sourceData || target.Owner != relation.Target || source.Type != target.Type {
			return false
		}
	}
	return true
}

func hasTypedQuerySignature(query database.QueryContract) bool {
	return len(query.Inputs) > 0 || query.Result != "" || query.ResultCardinality != ""
}

func querySignatureReadyForCross(query database.QueryContract, semanticTypes map[core.CanonicalID]struct{}, semanticData map[core.CanonicalID]struct{}) bool {
	if query.Result == "" || !query.ResultCardinality.Valid() {
		return false
	}
	if _, ok := semanticData[query.Result]; !ok {
		return false
	}
	seenInputs := map[core.LocalName]struct{}{}
	for _, input := range query.Inputs {
		if input.Name == "" || input.Type == "" {
			return false
		}
		if _, duplicate := seenInputs[input.Name]; duplicate {
			return false
		}
		seenInputs[input.Name] = struct{}{}
		if _, ok := semanticTypes[input.Type]; !ok {
			return false
		}
	}
	return true
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
