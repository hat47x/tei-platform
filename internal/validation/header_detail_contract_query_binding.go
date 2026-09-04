package validation

import (
	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/ir"
	"tei/internal/realization"
	"tei/internal/semantic"
	"tei/internal/subject"
)

type headerDetailBindingAttribute struct {
	Owner core.CanonicalID
	Type  core.CanonicalID
}

type resolvedHeaderDetailBinding struct {
	Application application.Application
	Pattern     application.Pattern
	SourceData  core.CanonicalID
	Relation    semantic.Relation
	Datasource  application.DatabaseRequirement
	Contract    database.Contract
	Query       database.QueryContract
}

// ValidateHeaderDetailContractQueryBindingsは、Relation-onlyのheader-detail Patternを
// collection取得用Database Contract Queryへ接続するrealization境界を検査する。
//
// reference Patternとは異なりRelation target keyを要求しない。source Data instanceに存在する
// 明示Semantic attributeからQuery inputへ値を渡し、Query resultがRelation target Dataのmanyであることを
// 確認する。物理Foreign KeyやRelationKeyから不足情報を補完しない。
func ValidateHeaderDetailContractQueryBindings(w ir.World) []Diagnostic {
	attributes := headerDetailBindingAttributes(w)
	seen := map[string]struct{}{}
	var out []Diagnostic

	for _, binding := range w.HeaderDetailContractQueryBindings {
		bindingSubject := subject.Subject{Kind: "header-detail-contract-query-binding", ID: binding.Identity()}
		identity := binding.Identity()
		if _, duplicate := seen[identity]; duplicate {
			out = append(out, errFor(
				bindingSubject,
				"DUPLICATE_HEADER_DETAIL_CONTRACT_QUERY_BINDING",
				"header-detail Contract Query binding %sが複数定義されています",
				identity,
			))
			continue
		}
		seen[identity] = struct{}{}

		out = append(out, validateHeaderDetailBindingIdentity(bindingSubject, binding)...)
		inputDiagnostics := validateHeaderDetailBindingInputsShape(bindingSubject, binding)
		out = append(out, inputDiagnostics...)

		if !headerDetailBindingIdentityReady(binding) || !headerDetailBindingInputsReady(binding) {
			continue
		}

		resolved, ok := resolveHeaderDetailBinding(w, binding, bindingSubject, &out)
		if !ok {
			continue
		}

		query := resolved.Query
		if !hasTypedQuerySignature(query) {
			out = append(out, errFor(
				bindingSubject,
				"HEADER_DETAIL_CONTRACT_QUERY_SIGNATURE_REQUIRED",
				"header-detail Contract Query binding %sが利用するQuery %s / %sにはtyped signatureが必要です",
				identity,
				binding.Target.Contract,
				binding.Target.Query,
			))
			continue
		}
		if !headerDetailQuerySignatureReady(query) {
			// Query signature自身の不整合はDatabase Contract Query signature Validationへ委ねる。
			continue
		}

		out = append(out, validateHeaderDetailBindingCrossLayer(
			bindingSubject,
			binding,
			resolved,
			attributes,
		)...)
	}

	return out
}

func validateHeaderDetailBindingIdentity(
	bindingSubject subject.Subject,
	binding realization.HeaderDetailContractQueryBinding,
) []Diagnostic {
	var out []Diagnostic
	if binding.Application == "" {
		out = append(out, errFor(bindingSubject, "HEADER_DETAIL_BINDING_APPLICATION_REQUIRED", "header-detail Contract Query bindingにはApplicationが必要です"))
	} else if _, err := core.ParseCanonicalID(string(binding.Application)); err != nil {
		out = append(out, errFor(bindingSubject, "INVALID_HEADER_DETAIL_BINDING_APPLICATION", "header-detail Contract Query bindingのApplication %qは有効なCanonical IDではありません", binding.Application))
	}
	if binding.Pattern == "" {
		out = append(out, errFor(bindingSubject, "HEADER_DETAIL_BINDING_PATTERN_REQUIRED", "header-detail Contract Query bindingにはPatternが必要です"))
	} else if _, err := core.ParseCanonicalID(string(binding.Pattern)); err != nil {
		out = append(out, errFor(bindingSubject, "INVALID_HEADER_DETAIL_BINDING_PATTERN", "header-detail Contract Query bindingのPattern %qは有効なCanonical IDではありません", binding.Pattern))
	}
	if binding.Datasource == "" {
		out = append(out, errFor(bindingSubject, "HEADER_DETAIL_BINDING_DATASOURCE_REQUIRED", "header-detail Contract Query bindingにはDatasourceが必要です"))
	} else if _, err := core.ParseLocalName(string(binding.Datasource)); err != nil {
		out = append(out, errFor(bindingSubject, "INVALID_HEADER_DETAIL_BINDING_DATASOURCE", "header-detail Contract Query bindingのDatasource %qは有効なLocalNameではありません", binding.Datasource))
	}
	if binding.Target.Database == "" {
		out = append(out, errFor(bindingSubject, "HEADER_DETAIL_BINDING_DATABASE_REQUIRED", "header-detail Contract Query bindingにはtarget Databaseが必要です"))
	} else if _, err := core.ParseCanonicalID(string(binding.Target.Database)); err != nil {
		out = append(out, errFor(bindingSubject, "INVALID_HEADER_DETAIL_BINDING_DATABASE", "header-detail Contract Query bindingのDatabase %qは有効なCanonical IDではありません", binding.Target.Database))
	}
	if binding.Target.Contract == "" {
		out = append(out, errFor(bindingSubject, "HEADER_DETAIL_BINDING_CONTRACT_REQUIRED", "header-detail Contract Query bindingにはContractが必要です"))
	} else if _, err := core.ParseCanonicalID(string(binding.Target.Contract)); err != nil {
		out = append(out, errFor(bindingSubject, "INVALID_HEADER_DETAIL_BINDING_CONTRACT", "header-detail Contract Query bindingのContract %qは有効なCanonical IDではありません", binding.Target.Contract))
	}
	if binding.Target.Query == "" {
		out = append(out, errFor(bindingSubject, "HEADER_DETAIL_BINDING_QUERY_REQUIRED", "header-detail Contract Query bindingにはQuery名が必要です"))
	} else if _, err := core.ParseLocalName(string(binding.Target.Query)); err != nil {
		out = append(out, errFor(bindingSubject, "INVALID_HEADER_DETAIL_BINDING_QUERY", "header-detail Contract Query bindingのQuery名 %qは有効なLocalNameではありません", binding.Target.Query))
	}
	return out
}

func headerDetailBindingIdentityReady(binding realization.HeaderDetailContractQueryBinding) bool {
	if !binding.Valid() {
		return false
	}
	if _, err := core.ParseCanonicalID(string(binding.Application)); err != nil {
		return false
	}
	if _, err := core.ParseCanonicalID(string(binding.Pattern)); err != nil {
		return false
	}
	if _, err := core.ParseLocalName(string(binding.Datasource)); err != nil {
		return false
	}
	if _, err := core.ParseCanonicalID(string(binding.Target.Database)); err != nil {
		return false
	}
	if _, err := core.ParseCanonicalID(string(binding.Target.Contract)); err != nil {
		return false
	}
	if _, err := core.ParseLocalName(string(binding.Target.Query)); err != nil {
		return false
	}
	return true
}

func resolveHeaderDetailBinding(
	w ir.World,
	binding realization.HeaderDetailContractQueryBinding,
	bindingSubject subject.Subject,
	out *[]Diagnostic,
) (resolvedHeaderDetailBinding, bool) {
	apps := make([]application.Application, 0, 1)
	for _, candidate := range w.Applications {
		if candidate.ID == binding.Application {
			apps = append(apps, candidate)
		}
	}
	if len(apps) == 0 {
		*out = append(*out, errFor(bindingSubject, "UNKNOWN_HEADER_DETAIL_BINDING_APPLICATION", "header-detail Contract Query binding %sが参照するApplication %sが見つかりません", binding.Identity(), binding.Application))
		return resolvedHeaderDetailBinding{}, false
	}
	if len(apps) > 1 {
		*out = append(*out, errFor(bindingSubject, "AMBIGUOUS_HEADER_DETAIL_BINDING_APPLICATION", "header-detail Contract Query binding %sが参照するApplication %sが%d件あり、一意に解決できません", binding.Identity(), binding.Application, len(apps)))
		return resolvedHeaderDetailBinding{}, false
	}
	app := apps[0]

	patterns := make([]application.Pattern, 0, 1)
	for _, candidate := range app.Patterns {
		if candidate.ID == binding.Pattern {
			patterns = append(patterns, candidate)
		}
	}
	if len(patterns) == 0 {
		*out = append(*out, errFor(bindingSubject, "UNKNOWN_HEADER_DETAIL_BINDING_PATTERN", "header-detail Contract Query binding %sが参照するPattern %sはApplication %sにありません", binding.Identity(), binding.Pattern, app.ID))
		return resolvedHeaderDetailBinding{}, false
	}
	if len(patterns) > 1 {
		*out = append(*out, errFor(bindingSubject, "AMBIGUOUS_HEADER_DETAIL_BINDING_PATTERN", "header-detail Contract Query binding %sが参照するPattern %sはApplication %sに%d件あり、一意に解決できません", binding.Identity(), binding.Pattern, app.ID, len(patterns)))
		return resolvedHeaderDetailBinding{}, false
	}
	pattern := patterns[0]
	if pattern.Kind != application.PatternHeaderDetail {
		*out = append(*out, errFor(bindingSubject, "HEADER_DETAIL_BINDING_REQUIRES_HEADER_DETAIL_PATTERN", "Pattern %sは%sですが、header-detail Contract Query bindingはheader-detail Patternだけをrealizeできます", pattern.ID, pattern.Kind))
		return resolvedHeaderDetailBinding{}, false
	}

	relations := make([]resolvedHeaderDetailBinding, 0, 1)
	for _, namespace := range w.Semantics {
		for _, data := range namespace.Data {
			for _, relation := range data.Relations {
				if relation.ID == pattern.Relation {
					relations = append(relations, resolvedHeaderDetailBinding{SourceData: data.ID, Relation: relation})
				}
			}
		}
	}
	if len(relations) != 1 {
		// Relation自身のunknown / ambiguityはApplication Pattern Validationへ委ねる。
		return resolvedHeaderDetailBinding{}, false
	}
	resolved := relations[0]
	if resolved.Relation.Cardinality != semantic.CardinalityMany {
		// cardinality不整合もApplication Pattern Validationへ委ねる。
		return resolvedHeaderDetailBinding{}, false
	}

	datasources := make([]application.DatabaseRequirement, 0, 1)
	for _, candidate := range app.Databases {
		if candidate.Alias == binding.Datasource {
			datasources = append(datasources, candidate)
		}
	}
	if len(datasources) == 0 {
		*out = append(*out, errFor(bindingSubject, "UNKNOWN_HEADER_DETAIL_BINDING_DATASOURCE", "header-detail Contract Query binding %sが参照するDatasource %sはApplication %sにありません", binding.Identity(), binding.Datasource, app.ID))
		return resolvedHeaderDetailBinding{}, false
	}
	if len(datasources) > 1 {
		*out = append(*out, errFor(bindingSubject, "AMBIGUOUS_HEADER_DETAIL_BINDING_DATASOURCE", "header-detail Contract Query binding %sが参照するDatasource %sはApplication %sに%d件あり、一意に解決できません", binding.Identity(), binding.Datasource, app.ID, len(datasources)))
		return resolvedHeaderDetailBinding{}, false
	}
	datasource := datasources[0]
	if datasource.Access != database.AccessRead && datasource.Access != database.AccessReadWrite {
		*out = append(*out, errFor(bindingSubject, "HEADER_DETAIL_BINDING_DATASOURCE_NOT_READABLE", "header-detail Contract Query binding %sのDatasource %sはaccess %sであり、collection取得に利用できません", binding.Identity(), binding.Datasource, datasource.Access))
		return resolvedHeaderDetailBinding{}, false
	}
	if datasource.TargetDatabase != binding.Target.Database {
		*out = append(*out, errFor(bindingSubject, "HEADER_DETAIL_BINDING_DATABASE_MISMATCH", "header-detail Contract Query binding %sのDatasource %sはDatabase %sを参照しますが、Contract Query targetは%sです", binding.Identity(), binding.Datasource, datasource.TargetDatabase, binding.Target.Database))
		return resolvedHeaderDetailBinding{}, false
	}

	requirements := requiredContractRequirements(datasource, binding.Target.Contract)
	if len(requirements) == 0 {
		*out = append(*out, errFor(bindingSubject, "HEADER_DETAIL_BINDING_CONTRACT_NOT_REQUIRED", "header-detail Contract Query binding %sはDatasource %sからContract %sを利用しますが、このContractはApplicationのrequiresにありません", binding.Identity(), binding.Datasource, binding.Target.Contract))
		return resolvedHeaderDetailBinding{}, false
	}
	if len(requirements) > 1 {
		*out = append(*out, errFor(bindingSubject, "AMBIGUOUS_HEADER_DETAIL_BINDING_CONTRACT_REQUIREMENT", "header-detail Contract Query binding %sが利用するContract %sのrequiresがDatasource %sに%d件あり、一意に解決できません", binding.Identity(), binding.Target.Contract, binding.Datasource, len(requirements)))
		return resolvedHeaderDetailBinding{}, false
	}

	databases := make([]database.Database, 0, 1)
	for _, candidate := range w.Databases {
		if candidate.ID == binding.Target.Database {
			databases = append(databases, candidate)
		}
	}
	if len(databases) == 0 {
		*out = append(*out, errFor(bindingSubject, "UNKNOWN_HEADER_DETAIL_BINDING_DATABASE", "header-detail Contract Query binding %sが参照するDatabase %sが見つかりません", binding.Identity(), binding.Target.Database))
		return resolvedHeaderDetailBinding{}, false
	}
	if len(databases) > 1 {
		*out = append(*out, errFor(bindingSubject, "AMBIGUOUS_HEADER_DETAIL_BINDING_DATABASE", "header-detail Contract Query binding %sが参照するDatabase %sが%d件あり、一意に解決できません", binding.Identity(), binding.Target.Database, len(databases)))
		return resolvedHeaderDetailBinding{}, false
	}

	contracts := make([]database.Contract, 0, 1)
	for _, candidate := range databases[0].Contracts {
		if candidate.ID == binding.Target.Contract {
			contracts = append(contracts, candidate)
		}
	}
	if len(contracts) == 0 {
		*out = append(*out, errFor(bindingSubject, "UNKNOWN_HEADER_DETAIL_BINDING_CONTRACT", "header-detail Contract Query binding %sが利用するContract %sはDatabase %sにありません", binding.Identity(), binding.Target.Contract, databases[0].ID))
		return resolvedHeaderDetailBinding{}, false
	}
	if len(contracts) > 1 {
		*out = append(*out, errFor(bindingSubject, "AMBIGUOUS_HEADER_DETAIL_BINDING_CONTRACT", "header-detail Contract Query binding %sが利用するContract %sはDatabase %sに%d件あり、一意に解決できません", binding.Identity(), binding.Target.Contract, databases[0].ID, len(contracts)))
		return resolvedHeaderDetailBinding{}, false
	}
	contract := contracts[0]
	if contract.Version.Major != requirements[0].CompatibleMajor {
		*out = append(*out, errFor(bindingSubject, "HEADER_DETAIL_BINDING_CONTRACT_MAJOR_VERSION_MISMATCH", "header-detail Contract Query binding %sはContract %sのmajor version %dとの互換性を要求していますが、Database %sではv%d.%dです", binding.Identity(), contract.ID, requirements[0].CompatibleMajor, databases[0].ID, contract.Version.Major, contract.Version.Minor))
		return resolvedHeaderDetailBinding{}, false
	}

	queries := make([]database.QueryContract, 0, 1)
	for _, candidate := range contract.Queries {
		if candidate.Name == string(binding.Target.Query) {
			queries = append(queries, candidate)
		}
	}
	if len(queries) == 0 {
		*out = append(*out, errFor(bindingSubject, "UNKNOWN_HEADER_DETAIL_BINDING_QUERY", "header-detail Contract Query binding %sが利用するQuery %sはContract %sにありません", binding.Identity(), binding.Target.Query, contract.ID))
		return resolvedHeaderDetailBinding{}, false
	}
	if len(queries) > 1 {
		*out = append(*out, errFor(bindingSubject, "AMBIGUOUS_HEADER_DETAIL_BINDING_QUERY", "header-detail Contract Query binding %sが利用するQuery %sはContract %sに%d件あり、一意に解決できません", binding.Identity(), binding.Target.Query, contract.ID, len(queries)))
		return resolvedHeaderDetailBinding{}, false
	}

	resolved.Application = app
	resolved.Pattern = pattern
	resolved.Datasource = datasource
	resolved.Contract = contract
	resolved.Query = queries[0]
	return resolved, true
}

func validateHeaderDetailBindingInputsShape(
	bindingSubject subject.Subject,
	binding realization.HeaderDetailContractQueryBinding,
) []Diagnostic {
	seenAttributes := map[core.CanonicalID]struct{}{}
	seenInputs := map[core.LocalName]struct{}{}
	var out []Diagnostic
	if len(binding.Inputs) == 0 {
		out = append(out, errFor(
			bindingSubject,
			"HEADER_DETAIL_BINDING_INPUTS_REQUIRED",
			"header-detail Contract Query binding %sにはsource instanceからcollection取得条件を渡すinput bindingが少なくとも一つ必要です",
			binding.Identity(),
		))
		return out
	}
	for _, inputBinding := range binding.Inputs {
		if inputBinding.SourceAttribute == "" {
			out = append(out, errFor(bindingSubject, "HEADER_DETAIL_BINDING_SOURCE_ATTRIBUTE_REQUIRED", "header-detail Contract Query input bindingにはsource attributeが必要です"))
		} else {
			if _, err := core.ParseCanonicalID(string(inputBinding.SourceAttribute)); err != nil {
				out = append(out, errFor(bindingSubject, "INVALID_HEADER_DETAIL_BINDING_SOURCE_ATTRIBUTE", "header-detail Contract Query bindingのsource attribute %qは有効なCanonical IDではありません", inputBinding.SourceAttribute))
			}
			if _, duplicate := seenAttributes[inputBinding.SourceAttribute]; duplicate {
				out = append(out, errFor(bindingSubject, "DUPLICATE_HEADER_DETAIL_BINDING_SOURCE_ATTRIBUTE", "header-detail Contract Query binding %sではsource attribute %sが複数のQuery input bindingに使われています", binding.Identity(), inputBinding.SourceAttribute))
			} else {
				seenAttributes[inputBinding.SourceAttribute] = struct{}{}
			}
		}
		if inputBinding.Input == "" {
			out = append(out, errFor(bindingSubject, "HEADER_DETAIL_BINDING_INPUT_REQUIRED", "header-detail Contract Query input bindingにはQuery input名が必要です"))
			continue
		}
		if _, err := core.ParseLocalName(string(inputBinding.Input)); err != nil {
			out = append(out, errFor(bindingSubject, "INVALID_HEADER_DETAIL_BINDING_INPUT", "header-detail Contract Query input名 %qは有効なLocalNameではありません", inputBinding.Input))
		}
		if _, duplicate := seenInputs[inputBinding.Input]; duplicate {
			out = append(out, errFor(bindingSubject, "DUPLICATE_HEADER_DETAIL_BINDING_INPUT", "header-detail Contract Query binding %sではQuery input %sが複数のbindingで使われています", binding.Identity(), inputBinding.Input))
		} else {
			seenInputs[inputBinding.Input] = struct{}{}
		}
	}
	return out
}

func headerDetailBindingInputsReady(binding realization.HeaderDetailContractQueryBinding) bool {
	if len(binding.Inputs) == 0 {
		return false
	}
	seenAttributes := map[core.CanonicalID]struct{}{}
	seenInputs := map[core.LocalName]struct{}{}
	for _, inputBinding := range binding.Inputs {
		if inputBinding.SourceAttribute == "" || inputBinding.Input == "" {
			return false
		}
		if _, err := core.ParseCanonicalID(string(inputBinding.SourceAttribute)); err != nil {
			return false
		}
		if _, err := core.ParseLocalName(string(inputBinding.Input)); err != nil {
			return false
		}
		if _, duplicate := seenAttributes[inputBinding.SourceAttribute]; duplicate {
			return false
		}
		seenAttributes[inputBinding.SourceAttribute] = struct{}{}
		if _, duplicate := seenInputs[inputBinding.Input]; duplicate {
			return false
		}
		seenInputs[inputBinding.Input] = struct{}{}
	}
	return true
}

func headerDetailQuerySignatureReady(query database.QueryContract) bool {
	if query.Result == "" || !query.ResultCardinality.Valid() {
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
	}
	return true
}

func validateHeaderDetailBindingCrossLayer(
	bindingSubject subject.Subject,
	binding realization.HeaderDetailContractQueryBinding,
	resolved resolvedHeaderDetailBinding,
	attributes map[core.CanonicalID]headerDetailBindingAttribute,
) []Diagnostic {
	queryInputs := map[core.LocalName]database.QueryInput{}
	for _, input := range resolved.Query.Inputs {
		queryInputs[input.Name] = input
	}
	bindingsByInput := map[core.LocalName]realization.HeaderDetailContractQueryInputBinding{}
	var out []Diagnostic

	for _, inputBinding := range binding.Inputs {
		bindingsByInput[inputBinding.Input] = inputBinding
		attribute, exists := attributes[inputBinding.SourceAttribute]
		if !exists || attribute.Owner != resolved.SourceData {
			out = append(out, errFor(
				bindingSubject,
				"HEADER_DETAIL_BINDING_SOURCE_ATTRIBUTE_OUTSIDE_RELATION_SOURCE",
				"header-detail Contract Query binding %sのsource attribute %sはRelation %sのsource Data %sに属していません",
				binding.Identity(),
				inputBinding.SourceAttribute,
				resolved.Relation.ID,
				resolved.SourceData,
			))
			continue
		}

		queryInput, exists := queryInputs[inputBinding.Input]
		if !exists {
			out = append(out, errFor(
				bindingSubject,
				"HEADER_DETAIL_BINDING_INPUT_NOT_IN_SIGNATURE",
				"header-detail Contract Query binding %sのinput %sはContract Query %s / %sのsignatureにありません",
				binding.Identity(),
				inputBinding.Input,
				binding.Target.Contract,
				binding.Target.Query,
			))
			continue
		}
		if attribute.Type != queryInput.Type {
			out = append(out, errFor(
				bindingSubject,
				"HEADER_DETAIL_BINDING_INPUT_TYPE_MISMATCH",
				"header-detail Contract Query binding %sはsource attribute %s（型 %s）をQuery input %s（型 %s）へ渡しますが、Semantic型identityが一致しません",
				binding.Identity(),
				inputBinding.SourceAttribute,
				attribute.Type,
				inputBinding.Input,
				queryInput.Type,
			))
		}
	}

	for _, input := range resolved.Query.Inputs {
		if _, exists := bindingsByInput[input.Name]; !exists {
			out = append(out, errFor(
				bindingSubject,
				"HEADER_DETAIL_BINDING_QUERY_INPUT_NOT_BOUND",
				"header-detail Contract Query binding %sではContract Query %s / %sのinput %sへ値を渡すbindingがありません",
				binding.Identity(),
				binding.Target.Contract,
				binding.Target.Query,
				input.Name,
			))
		}
	}

	if resolved.Query.Result != resolved.Relation.Target {
		out = append(out, errFor(
			bindingSubject,
			"HEADER_DETAIL_BINDING_RESULT_DATA_MISMATCH",
			"header-detail Pattern %sのRelation %sは%sをtargetとしますが、Contract Query %s / %sは%sを返します",
			binding.Pattern,
			resolved.Relation.ID,
			resolved.Relation.Target,
			binding.Target.Contract,
			binding.Target.Query,
			resolved.Query.Result,
		))
	}
	if resolved.Query.ResultCardinality != database.QueryResultMany {
		out = append(out, errFor(
			bindingSubject,
			"HEADER_DETAIL_BINDING_RESULT_CARDINALITY_NOT_MANY",
			"header-detail Pattern %sのContract Query %s / %sはcollection取得にmanyを必要としますが、result cardinalityは%sです",
			binding.Pattern,
			binding.Target.Contract,
			binding.Target.Query,
			resolved.Query.ResultCardinality,
		))
	}
	return out
}

func headerDetailBindingAttributes(w ir.World) map[core.CanonicalID]headerDetailBindingAttribute {
	out := map[core.CanonicalID]headerDetailBindingAttribute{}
	for _, namespace := range w.Semantics {
		for _, data := range namespace.Data {
			for _, attribute := range data.Attributes {
				out[attribute.ID] = headerDetailBindingAttribute{Owner: data.ID, Type: attribute.TypeRef}
			}
		}
	}
	return out
}
