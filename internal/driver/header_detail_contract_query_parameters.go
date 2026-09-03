package driver

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/driver/queryresult"
)

// PreparedHeaderDetailContractQueryは、header-detail Patternのsource instanceから
// collection取得用Contract Queryを一回呼び出すための解決結果とruntime parameterを保持する。
//
// Parametersは一回の実行にだけ使う値であり、Canonical Worldや通常Evidenceへ保存しない。
type PreparedHeaderDetailContractQuery struct {
	Resolution ResolvedHeaderDetailContractQuery
	Parameters map[string]any
}

// PrepareHeaderDetailContractQueryは、Semanticへ明示mapping済みのsource rowと
// HeaderDetailContractQueryBindingだけを使ってContract Query parameterを組み立てる。
//
// RelationKeyは要求しない。source attribute名、Query input名、列順、物理Foreign Key、
// Persistence Mappingから対応を推測せず、Bindingに明示されたSourceAttribute -> Inputだけを利用する。
func (r Resolver) PrepareHeaderDetailContractQuery(
	applicationID, patternID core.CanonicalID,
	source queryresult.MappedResult,
	rowIndex int,
) (PreparedHeaderDetailContractQuery, error) {
	resolved, err := r.ResolveHeaderDetailContractQuery(applicationID, patternID)
	if err != nil {
		return PreparedHeaderDetailContractQuery{}, err
	}

	if rowIndex < 0 || rowIndex >= len(source.Rows) {
		return PreparedHeaderDetailContractQuery{}, fmt.Errorf("header-detail Pattern %sへ渡すmapped row %dがありません: rows=%d", patternID, rowIndex, len(source.Rows))
	}
	row := source.Rows[rowIndex]
	if len(row) != len(source.Columns) {
		return PreparedHeaderDetailContractQuery{}, fmt.Errorf(
			"header-detail Pattern %sへ渡すmapped row %dは列数が一致しません: columns=%d values=%d",
			patternID,
			rowIndex,
			len(source.Columns),
			len(row),
		)
	}

	valueIndex := make(map[core.CanonicalID]int, len(source.Columns))
	for index, column := range source.Columns {
		if column.Attribute == "" {
			return PreparedHeaderDetailContractQuery{}, fmt.Errorf("header-detail Pattern %sへ渡すmapped resultにSemantic attributeが未指定の列があります", patternID)
		}
		if _, duplicate := valueIndex[column.Attribute]; duplicate {
			return PreparedHeaderDetailContractQuery{}, fmt.Errorf("header-detail Pattern %sへ渡すmapped resultでSemantic attribute %sが重複しています", patternID, column.Attribute)
		}
		valueIndex[column.Attribute] = index
	}

	sourceAttributes, err := uniqueSemanticAttributes(resolved.Relation.Source.Attributes)
	if err != nil {
		return PreparedHeaderDetailContractQuery{}, fmt.Errorf("header-detail Pattern %sのRelation %s source Dataを一意に解釈できません: %w", patternID, resolved.Relation.Relation.ID, err)
	}

	if len(resolved.Binding.Inputs) == 0 {
		return PreparedHeaderDetailContractQuery{}, fmt.Errorf("header-detail Pattern %sのContract Query bindingにsource attributeからQuery inputへの対応がありません", patternID)
	}

	queryInputs := make(map[core.LocalName]core.CanonicalID, len(resolved.Query.Inputs))
	for _, input := range resolved.Query.Inputs {
		if input.Name == "" || input.Type == "" {
			return PreparedHeaderDetailContractQuery{}, fmt.Errorf("header-detail Pattern %sが利用するContract Query %sのtyped inputが不完全です", patternID, resolved.Query.Name)
		}
		if _, duplicate := queryInputs[input.Name]; duplicate {
			return PreparedHeaderDetailContractQuery{}, fmt.Errorf("header-detail Pattern %sが利用するContract Query %sでinput %sが重複しています", patternID, resolved.Query.Name, input.Name)
		}
		queryInputs[input.Name] = input.Type
	}
	if len(queryInputs) == 0 {
		return PreparedHeaderDetailContractQuery{}, fmt.Errorf("header-detail Pattern %sが利用するContract Query %sにtyped inputがありません", patternID, resolved.Query.Name)
	}
	if resolved.Query.Result == "" || !resolved.Query.ResultCardinality.Valid() {
		return PreparedHeaderDetailContractQuery{}, fmt.Errorf("header-detail Pattern %sが利用するContract Query %sのresult契約が不完全です", patternID, resolved.Query.Name)
	}
	if resolved.Query.Result != resolved.Relation.Relation.Target {
		return PreparedHeaderDetailContractQuery{}, fmt.Errorf(
			"header-detail Pattern %sのRelation %sは%sをtargetとしますが、Contract Query %sは%sを返します",
			patternID,
			resolved.Relation.Relation.ID,
			resolved.Relation.Relation.Target,
			resolved.Query.Name,
			resolved.Query.Result,
		)
	}
	if resolved.Query.ResultCardinality != database.QueryResultMany {
		return PreparedHeaderDetailContractQuery{}, fmt.Errorf(
			"header-detail Pattern %sのContract Query %sはmanyを返す必要がありますが、result cardinalityは%sです",
			patternID,
			resolved.Query.Name,
			resolved.Query.ResultCardinality,
		)
	}

	parameters := make(map[string]any, len(resolved.Binding.Inputs))
	seenSourceAttributes := make(map[core.CanonicalID]struct{}, len(resolved.Binding.Inputs))
	seenInputs := make(map[core.LocalName]struct{}, len(resolved.Binding.Inputs))
	for _, binding := range resolved.Binding.Inputs {
		if binding.SourceAttribute == "" || binding.Input == "" {
			return PreparedHeaderDetailContractQuery{}, fmt.Errorf("header-detail Pattern %sのContract Query input bindingが不完全です", patternID)
		}
		if _, duplicate := seenSourceAttributes[binding.SourceAttribute]; duplicate {
			return PreparedHeaderDetailContractQuery{}, fmt.Errorf("header-detail Pattern %sでsource attribute %sのContract Query bindingが重複しています", patternID, binding.SourceAttribute)
		}
		seenSourceAttributes[binding.SourceAttribute] = struct{}{}
		if _, duplicate := seenInputs[binding.Input]; duplicate {
			return PreparedHeaderDetailContractQuery{}, fmt.Errorf("header-detail Pattern %sでContract Query input %sのbindingが重複しています", patternID, binding.Input)
		}
		seenInputs[binding.Input] = struct{}{}

		sourceAttribute, exists := sourceAttributes[binding.SourceAttribute]
		if !exists {
			return PreparedHeaderDetailContractQuery{}, fmt.Errorf(
				"header-detail Pattern %sのbinding元attribute %sはRelation %sのsource Data %sに属していません",
				patternID,
				binding.SourceAttribute,
				resolved.Relation.Relation.ID,
				resolved.Relation.Source.ID,
			)
		}
		inputType, exists := queryInputs[binding.Input]
		if !exists {
			return PreparedHeaderDetailContractQuery{}, fmt.Errorf("header-detail Pattern %sのbinding先input %sはContract Query %sのsignatureにありません", patternID, binding.Input, resolved.Query.Name)
		}
		if sourceAttribute.TypeRef != inputType {
			return PreparedHeaderDetailContractQuery{}, fmt.Errorf(
				"header-detail Pattern %sはsource attribute %s（型 %s）をContract Query input %s（型 %s）へ渡せません",
				patternID,
				binding.SourceAttribute,
				sourceAttribute.TypeRef,
				binding.Input,
				inputType,
			)
		}
		valuePosition, exists := valueIndex[binding.SourceAttribute]
		if !exists {
			return PreparedHeaderDetailContractQuery{}, fmt.Errorf("header-detail Pattern %sを辿るために必要なsource attribute %sがmapped resultにありません", patternID, binding.SourceAttribute)
		}
		parameters[string(binding.Input)] = row[valuePosition]
	}

	for inputName := range queryInputs {
		if _, bound := seenInputs[inputName]; !bound {
			return PreparedHeaderDetailContractQuery{}, fmt.Errorf("header-detail Pattern %sではContract Query input %sへ値を渡すbindingがありません", patternID, inputName)
		}
	}
	if len(seenInputs) != len(queryInputs) || len(parameters) != len(queryInputs) {
		return PreparedHeaderDetailContractQuery{}, fmt.Errorf(
			"header-detail Pattern %sのContract Query parameterを過不足なく組み立てられません: bindings=%d parameters=%d inputs=%d",
			patternID,
			len(seenInputs),
			len(parameters),
			len(queryInputs),
		)
	}

	return PreparedHeaderDetailContractQuery{Resolution: resolved, Parameters: parameters}, nil
}
