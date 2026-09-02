package driver

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/driver/queryresult"
	"tei/internal/semantic"
)

// PreparedReferenceContractQueryは、reference Patternが選んだContract Queryと、
// その一回の呼び出しに必要なruntime parameterをまとめる。
//
// Parametersは実行時の値であり、Canonical Worldや通常Evidenceへ保存する情報ではない。
// この型にはJSON tagを付けず、永続化用の契約として扱わない。
type PreparedReferenceContractQuery struct {
	Resolution ResolvedReferenceContractQuery
	Parameters map[string]any
}

// PrepareReferenceContractQueryは、Semanticへ明示mapping済みの一行から、
// Relation keyとreference Patternのinput bindingだけを使ってContract Query parameterを組み立てる。
//
// attribute名、列順、Semantic型の一致、物理Foreign Key等からbindingを推測しない。
// runtime値の型変換もこの境界では行わず、mapped resultに観測された値をそのまま渡す。
func (r Resolver) PrepareReferenceContractQuery(
	applicationID, patternID core.CanonicalID,
	source queryresult.MappedResult,
	rowIndex int,
) (PreparedReferenceContractQuery, error) {
	patternResolution, err := r.ResolvePattern(applicationID, patternID)
	if err != nil {
		return PreparedReferenceContractQuery{}, err
	}
	if patternResolution.Relation == nil {
		return PreparedReferenceContractQuery{}, fmt.Errorf("reference Pattern %sのRelationが解決されていません", patternID)
	}

	resolved, err := r.ResolveReferenceContractQuery(applicationID, patternID)
	if err != nil {
		return PreparedReferenceContractQuery{}, err
	}

	if rowIndex < 0 || rowIndex >= len(source.Rows) {
		return PreparedReferenceContractQuery{}, fmt.Errorf("reference Pattern %sへ渡すmapped row %dがありません: rows=%d", patternID, rowIndex, len(source.Rows))
	}
	row := source.Rows[rowIndex]
	if len(row) != len(source.Columns) {
		return PreparedReferenceContractQuery{}, fmt.Errorf(
			"reference Pattern %sへ渡すmapped row %dは列数が一致しません: columns=%d values=%d",
			patternID,
			rowIndex,
			len(source.Columns),
			len(row),
		)
	}

	valueIndex := make(map[core.CanonicalID]int, len(source.Columns))
	for index, column := range source.Columns {
		if column.Attribute == "" {
			return PreparedReferenceContractQuery{}, fmt.Errorf("reference Pattern %sへ渡すmapped resultにSemantic attributeが未指定の列があります", patternID)
		}
		if _, duplicate := valueIndex[column.Attribute]; duplicate {
			return PreparedReferenceContractQuery{}, fmt.Errorf("reference Pattern %sへ渡すmapped resultでSemantic attribute %sが重複しています", patternID, column.Attribute)
		}
		valueIndex[column.Attribute] = index
	}

	relation := patternResolution.Relation.Relation
	if len(relation.Keys) == 0 {
		return PreparedReferenceContractQuery{}, fmt.Errorf("reference Pattern %sのRelation %sにtraversal keyがありません", patternID, relation.ID)
	}

	sourceAttributes, err := uniqueSemanticAttributes(patternResolution.Relation.Source.Attributes)
	if err != nil {
		return PreparedReferenceContractQuery{}, fmt.Errorf("reference Pattern %sのRelation %s source Dataを一意に解釈できません: %w", patternID, relation.ID, err)
	}
	targetAttributes, err := uniqueSemanticAttributes(patternResolution.Relation.Target.Attributes)
	if err != nil {
		return PreparedReferenceContractQuery{}, fmt.Errorf("reference Pattern %sのRelation %s target Dataを一意に解釈できません: %w", patternID, relation.ID, err)
	}

	targetToSource := make(map[core.CanonicalID]core.CanonicalID, len(relation.Keys))
	for _, key := range relation.Keys {
		if key.SourceAttribute == "" || key.TargetAttribute == "" {
			return PreparedReferenceContractQuery{}, fmt.Errorf("reference Pattern %sのRelation %sに不完全なtraversal keyがあります", patternID, relation.ID)
		}
		sourceAttribute, sourceExists := sourceAttributes[key.SourceAttribute]
		if !sourceExists {
			return PreparedReferenceContractQuery{}, fmt.Errorf("reference Pattern %sのRelation %sがsource Dataにないattribute %sをkeyとして参照しています", patternID, relation.ID, key.SourceAttribute)
		}
		targetAttribute, targetExists := targetAttributes[key.TargetAttribute]
		if !targetExists {
			return PreparedReferenceContractQuery{}, fmt.Errorf("reference Pattern %sのRelation %sがtarget Dataにないattribute %sをkeyとして参照しています", patternID, relation.ID, key.TargetAttribute)
		}
		if sourceAttribute.TypeRef != targetAttribute.TypeRef {
			return PreparedReferenceContractQuery{}, fmt.Errorf(
				"reference Pattern %sのRelation %sはsource attribute %s（型 %s）とtarget attribute %s（型 %s）をkey対応できません",
				patternID,
				relation.ID,
				key.SourceAttribute,
				sourceAttribute.TypeRef,
				key.TargetAttribute,
				targetAttribute.TypeRef,
			)
		}
		if prior, duplicate := targetToSource[key.TargetAttribute]; duplicate {
			return PreparedReferenceContractQuery{}, fmt.Errorf(
				"reference Pattern %sのRelation %sではtarget attribute %sにsource attribute %sと%sが対応付けられており、一意に辿れません",
				patternID,
				relation.ID,
				key.TargetAttribute,
				prior,
				key.SourceAttribute,
			)
		}
		targetToSource[key.TargetAttribute] = key.SourceAttribute
	}

	bindingsByTarget := make(map[core.CanonicalID]core.LocalName, len(resolved.Invocation.Inputs))
	boundInputs := make(map[core.LocalName]struct{}, len(resolved.Invocation.Inputs))
	for _, binding := range resolved.Invocation.Inputs {
		if binding.TargetAttribute == "" || binding.Input == "" {
			return PreparedReferenceContractQuery{}, fmt.Errorf("reference Pattern %sのContract Query input bindingが不完全です", patternID)
		}
		if _, relationTargetKey := targetToSource[binding.TargetAttribute]; !relationTargetKey {
			return PreparedReferenceContractQuery{}, fmt.Errorf("reference Pattern %sのbinding元attribute %sはRelation %sのtarget keyではありません", patternID, binding.TargetAttribute, relation.ID)
		}
		if _, duplicate := bindingsByTarget[binding.TargetAttribute]; duplicate {
			return PreparedReferenceContractQuery{}, fmt.Errorf("reference Pattern %sでtarget attribute %sのContract Query bindingが重複しています", patternID, binding.TargetAttribute)
		}
		if _, duplicate := boundInputs[binding.Input]; duplicate {
			return PreparedReferenceContractQuery{}, fmt.Errorf("reference Pattern %sでContract Query input %sのbindingが重複しています", patternID, binding.Input)
		}
		bindingsByTarget[binding.TargetAttribute] = binding.Input
		boundInputs[binding.Input] = struct{}{}
	}

	queryInputs := make(map[core.LocalName]core.CanonicalID, len(resolved.Query.Inputs))
	for _, input := range resolved.Query.Inputs {
		if input.Name == "" || input.Type == "" {
			return PreparedReferenceContractQuery{}, fmt.Errorf("reference Pattern %sが利用するContract Query %sのtyped inputが不完全です", patternID, resolved.Query.Name)
		}
		if _, duplicate := queryInputs[input.Name]; duplicate {
			return PreparedReferenceContractQuery{}, fmt.Errorf("reference Pattern %sが利用するContract Query %sでinput %sが重複しています", patternID, resolved.Query.Name, input.Name)
		}
		queryInputs[input.Name] = input.Type
	}
	if len(queryInputs) == 0 {
		return PreparedReferenceContractQuery{}, fmt.Errorf("reference Pattern %sが利用するContract Query %sにtyped inputがありません", patternID, resolved.Query.Name)
	}
	if resolved.Query.Result == "" || !resolved.Query.ResultCardinality.Valid() {
		return PreparedReferenceContractQuery{}, fmt.Errorf("reference Pattern %sが利用するContract Query %sのresult契約が不完全です", patternID, resolved.Query.Name)
	}
	if resolved.Query.Result != relation.Target {
		return PreparedReferenceContractQuery{}, fmt.Errorf("reference Pattern %sのRelation %sは%sをtargetとしますが、Contract Query %sは%sを返します", patternID, relation.ID, relation.Target, resolved.Query.Name, resolved.Query.Result)
	}
	if !referenceContractQueryCardinalityCompatible(relation.Cardinality, resolved.Query.ResultCardinality) {
		return PreparedReferenceContractQuery{}, fmt.Errorf("reference Pattern %sのRelation %sはcardinality %sですが、Contract Query %sのresult cardinalityは%sです", patternID, relation.ID, relation.Cardinality, resolved.Query.Name, resolved.Query.ResultCardinality)
	}

	parameters := make(map[string]any, len(relation.Keys))
	for _, key := range relation.Keys {
		inputName, bound := bindingsByTarget[key.TargetAttribute]
		if !bound {
			return PreparedReferenceContractQuery{}, fmt.Errorf("reference Pattern %sではRelation target key %sをContract Query inputへ渡すbindingがありません", patternID, key.TargetAttribute)
		}
		inputType, exists := queryInputs[inputName]
		if !exists {
			return PreparedReferenceContractQuery{}, fmt.Errorf("reference Pattern %sのbinding先input %sはContract Query %sのsignatureにありません", patternID, inputName, resolved.Query.Name)
		}
		targetAttribute := targetAttributes[key.TargetAttribute]
		if targetAttribute.TypeRef != inputType {
			return PreparedReferenceContractQuery{}, fmt.Errorf(
				"reference Pattern %sはtarget attribute %s（型 %s）をContract Query input %s（型 %s）へ渡せません",
				patternID,
				key.TargetAttribute,
				targetAttribute.TypeRef,
				inputName,
				inputType,
			)
		}

		valuePosition, exists := valueIndex[key.SourceAttribute]
		if !exists {
			return PreparedReferenceContractQuery{}, fmt.Errorf("reference Pattern %sを辿るために必要なsource key %sがmapped resultにありません", patternID, key.SourceAttribute)
		}
		parameters[string(inputName)] = row[valuePosition]
	}

	for inputName := range queryInputs {
		if _, bound := boundInputs[inputName]; !bound {
			return PreparedReferenceContractQuery{}, fmt.Errorf("reference Pattern %sではContract Query input %sへ値を渡すbindingがありません", patternID, inputName)
		}
	}
	if len(boundInputs) != len(queryInputs) || len(parameters) != len(queryInputs) {
		return PreparedReferenceContractQuery{}, fmt.Errorf(
			"reference Pattern %sのContract Query parameterを過不足なく組み立てられません: bindings=%d parameters=%d inputs=%d",
			patternID,
			len(boundInputs),
			len(parameters),
			len(queryInputs),
		)
	}

	return PreparedReferenceContractQuery{Resolution: resolved, Parameters: parameters}, nil
}

func uniqueSemanticAttributes(attributes []semantic.Attribute) (map[core.CanonicalID]semantic.Attribute, error) {
	out := make(map[core.CanonicalID]semantic.Attribute, len(attributes))
	for _, attribute := range attributes {
		if attribute.ID == "" {
			return nil, fmt.Errorf("Semantic attributeにCanonical IDがありません")
		}
		if _, duplicate := out[attribute.ID]; duplicate {
			return nil, fmt.Errorf("Semantic attribute %sが重複しています", attribute.ID)
		}
		out[attribute.ID] = attribute
	}
	return out, nil
}

func referenceContractQueryCardinalityCompatible(relation semantic.Cardinality, result database.QueryResultCardinality) bool {
	switch relation {
	case semantic.CardinalityOne:
		return result == database.QueryResultOne || result == database.QueryResultZeroOrOne
	case semantic.CardinalityMany:
		return result == database.QueryResultMany
	default:
		return false
	}
}
