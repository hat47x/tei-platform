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
				"header-detail Contract Query binding %s is defined more than once",
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
				"Query %s / %s, used by header-detail Contract Query binding %s, needs a typed signature",
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
		out = append(out, errFor(bindingSubject, "HEADER_DETAIL_BINDING_APPLICATION_REQUIRED", "a header-detail Contract Query binding needs Application"))
	} else if _, err := core.ParseCanonicalID(string(binding.Application)); err != nil {
		out = append(out, errFor(bindingSubject, "INVALID_HEADER_DETAIL_BINDING_APPLICATION", "header-detail Contract Query binding Application %q is not a valid Canonical ID", binding.Application))
	}
	if binding.Pattern == "" {
		out = append(out, errFor(bindingSubject, "HEADER_DETAIL_BINDING_PATTERN_REQUIRED", "a header-detail Contract Query binding needs Pattern"))
	} else if _, err := core.ParseCanonicalID(string(binding.Pattern)); err != nil {
		out = append(out, errFor(bindingSubject, "INVALID_HEADER_DETAIL_BINDING_PATTERN", "header-detail Contract Query binding Pattern %q is not a valid Canonical ID", binding.Pattern))
	}
	if binding.Datasource == "" {
		out = append(out, errFor(bindingSubject, "HEADER_DETAIL_BINDING_DATASOURCE_REQUIRED", "a header-detail Contract Query binding needs Datasource"))
	} else if _, err := core.ParseLocalName(string(binding.Datasource)); err != nil {
		out = append(out, errFor(bindingSubject, "INVALID_HEADER_DETAIL_BINDING_DATASOURCE", "header-detail Contract Query binding Datasource %q is not a valid LocalName", binding.Datasource))
	}
	if binding.Target.Database == "" {
		out = append(out, errFor(bindingSubject, "HEADER_DETAIL_BINDING_DATABASE_REQUIRED", "a header-detail Contract Query binding needs a target Database"))
	} else if _, err := core.ParseCanonicalID(string(binding.Target.Database)); err != nil {
		out = append(out, errFor(bindingSubject, "INVALID_HEADER_DETAIL_BINDING_DATABASE", "header-detail Contract Query binding Database %q is not a valid Canonical ID", binding.Target.Database))
	}
	if binding.Target.Contract == "" {
		out = append(out, errFor(bindingSubject, "HEADER_DETAIL_BINDING_CONTRACT_REQUIRED", "a header-detail Contract Query binding needs Contract"))
	} else if _, err := core.ParseCanonicalID(string(binding.Target.Contract)); err != nil {
		out = append(out, errFor(bindingSubject, "INVALID_HEADER_DETAIL_BINDING_CONTRACT", "header-detail Contract Query binding Contract %q is not a valid Canonical ID", binding.Target.Contract))
	}
	if binding.Target.Query == "" {
		out = append(out, errFor(bindingSubject, "HEADER_DETAIL_BINDING_QUERY_REQUIRED", "a header-detail Contract Query binding needs a Query name"))
	} else if _, err := core.ParseLocalName(string(binding.Target.Query)); err != nil {
		out = append(out, errFor(bindingSubject, "INVALID_HEADER_DETAIL_BINDING_QUERY", "header-detail Contract Query binding Query name %q is not a valid LocalName", binding.Target.Query))
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
		*out = append(*out, errFor(bindingSubject, "UNKNOWN_HEADER_DETAIL_BINDING_APPLICATION", "header-detail Contract Query binding %s refers to Application %s, which is not found", binding.Identity(), binding.Application))
		return resolvedHeaderDetailBinding{}, false
	}
	if len(apps) > 1 {
		*out = append(*out, errFor(bindingSubject, "AMBIGUOUS_HEADER_DETAIL_BINDING_APPLICATION", "header-detail Contract Query binding %s refers to Application %s, which has %d candidates and cannot be resolved uniquely", binding.Identity(), binding.Application, len(apps)))
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
		*out = append(*out, errFor(bindingSubject, "UNKNOWN_HEADER_DETAIL_BINDING_PATTERN", "header-detail Contract Query binding %s refers to Pattern %s, which Application %s does not have", binding.Identity(), binding.Pattern, app.ID))
		return resolvedHeaderDetailBinding{}, false
	}
	if len(patterns) > 1 {
		*out = append(*out, errFor(bindingSubject, "AMBIGUOUS_HEADER_DETAIL_BINDING_PATTERN", "header-detail Contract Query binding %s refers to Pattern %s in Application %s, which has %d candidates and cannot be resolved uniquely", binding.Identity(), binding.Pattern, app.ID, len(patterns)))
		return resolvedHeaderDetailBinding{}, false
	}
	pattern := patterns[0]
	if pattern.Kind != application.PatternHeaderDetail {
		*out = append(*out, errFor(bindingSubject, "HEADER_DETAIL_BINDING_REQUIRES_HEADER_DETAIL_PATTERN", "Pattern %s is %s, and a header-detail Contract Query binding can only realize a header-detail pattern", pattern.ID, pattern.Kind))
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
		*out = append(*out, errFor(bindingSubject, "UNKNOWN_HEADER_DETAIL_BINDING_DATASOURCE", "header-detail Contract Query binding %s refers to Datasource %s, which Application %s does not have", binding.Identity(), binding.Datasource, app.ID))
		return resolvedHeaderDetailBinding{}, false
	}
	if len(datasources) > 1 {
		*out = append(*out, errFor(bindingSubject, "AMBIGUOUS_HEADER_DETAIL_BINDING_DATASOURCE", "header-detail Contract Query binding %s refers to Datasource %s in Application %s, which has %d candidates and cannot be resolved uniquely", binding.Identity(), binding.Datasource, app.ID, len(datasources)))
		return resolvedHeaderDetailBinding{}, false
	}
	datasource := datasources[0]
	if datasource.Access != database.AccessRead && datasource.Access != database.AccessReadWrite {
		*out = append(*out, errFor(bindingSubject, "HEADER_DETAIL_BINDING_DATASOURCE_NOT_READABLE", "header-detail Contract Query binding %s Datasource %s has access %s, so it cannot be used to read a collection", binding.Identity(), binding.Datasource, datasource.Access))
		return resolvedHeaderDetailBinding{}, false
	}
	if datasource.TargetDatabase != binding.Target.Database {
		*out = append(*out, errFor(bindingSubject, "HEADER_DETAIL_BINDING_DATABASE_MISMATCH", "header-detail Contract Query binding %s Datasource %s refers to Database %s, but the Contract Query target is %s", binding.Identity(), binding.Datasource, datasource.TargetDatabase, binding.Target.Database))
		return resolvedHeaderDetailBinding{}, false
	}

	requirements := requiredContractRequirements(datasource, binding.Target.Contract)
	if len(requirements) == 0 {
		*out = append(*out, errFor(bindingSubject, "HEADER_DETAIL_BINDING_CONTRACT_NOT_REQUIRED", "header-detail Contract Query binding %s uses Contract %s from Datasource %s, but the Application does not require that Contract", binding.Identity(), binding.Datasource, binding.Target.Contract))
		return resolvedHeaderDetailBinding{}, false
	}
	if len(requirements) > 1 {
		*out = append(*out, errFor(bindingSubject, "AMBIGUOUS_HEADER_DETAIL_BINDING_CONTRACT_REQUIREMENT", "header-detail Contract Query binding %s uses Contract %s on Datasource %s, whose requires has %d candidates and cannot be resolved uniquely", binding.Identity(), binding.Target.Contract, binding.Datasource, len(requirements)))
		return resolvedHeaderDetailBinding{}, false
	}

	databases := make([]database.Database, 0, 1)
	for _, candidate := range w.Databases {
		if candidate.ID == binding.Target.Database {
			databases = append(databases, candidate)
		}
	}
	if len(databases) == 0 {
		*out = append(*out, errFor(bindingSubject, "UNKNOWN_HEADER_DETAIL_BINDING_DATABASE", "header-detail Contract Query binding %s refers to Database %s, which is not found", binding.Identity(), binding.Target.Database))
		return resolvedHeaderDetailBinding{}, false
	}
	if len(databases) > 1 {
		*out = append(*out, errFor(bindingSubject, "AMBIGUOUS_HEADER_DETAIL_BINDING_DATABASE", "header-detail Contract Query binding %s refers to Database %s, which has %d candidates and cannot be resolved uniquely", binding.Identity(), binding.Target.Database, len(databases)))
		return resolvedHeaderDetailBinding{}, false
	}

	contracts := make([]database.Contract, 0, 1)
	for _, candidate := range databases[0].Contracts {
		if candidate.ID == binding.Target.Contract {
			contracts = append(contracts, candidate)
		}
	}
	if len(contracts) == 0 {
		*out = append(*out, errFor(bindingSubject, "UNKNOWN_HEADER_DETAIL_BINDING_CONTRACT", "header-detail Contract Query binding %s uses Contract %s, which Database %s does not have", binding.Identity(), binding.Target.Contract, databases[0].ID))
		return resolvedHeaderDetailBinding{}, false
	}
	if len(contracts) > 1 {
		*out = append(*out, errFor(bindingSubject, "AMBIGUOUS_HEADER_DETAIL_BINDING_CONTRACT", "header-detail Contract Query binding %s uses Contract %s in Database %s, which has %d candidates and cannot be resolved uniquely", binding.Identity(), binding.Target.Contract, databases[0].ID, len(contracts)))
		return resolvedHeaderDetailBinding{}, false
	}
	contract := contracts[0]
	if contract.Version.Major != requirements[0].CompatibleMajor {
		*out = append(*out, errFor(bindingSubject, "HEADER_DETAIL_BINDING_CONTRACT_MAJOR_VERSION_MISMATCH", "header-detail Contract Query binding %s requires compatibility with Contract %s major version %d, but Database %s has v%d.%d", binding.Identity(), contract.ID, requirements[0].CompatibleMajor, databases[0].ID, contract.Version.Major, contract.Version.Minor))
		return resolvedHeaderDetailBinding{}, false
	}

	queries := make([]database.QueryContract, 0, 1)
	for _, candidate := range contract.Queries {
		if candidate.Name == string(binding.Target.Query) {
			queries = append(queries, candidate)
		}
	}
	if len(queries) == 0 {
		*out = append(*out, errFor(bindingSubject, "UNKNOWN_HEADER_DETAIL_BINDING_QUERY", "header-detail Contract Query binding %s uses Query %s, which Contract %s does not have", binding.Identity(), binding.Target.Query, contract.ID))
		return resolvedHeaderDetailBinding{}, false
	}
	if len(queries) > 1 {
		*out = append(*out, errFor(bindingSubject, "AMBIGUOUS_HEADER_DETAIL_BINDING_QUERY", "header-detail Contract Query binding %s uses Query %s in Contract %s, which has %d candidates and cannot be resolved uniquely", binding.Identity(), binding.Target.Query, contract.ID, len(queries)))
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
			"header-detail Contract Query binding %s needs at least one input binding that passes a collection condition from a source instance",
			binding.Identity(),
		))
		return out
	}
	for _, inputBinding := range binding.Inputs {
		if inputBinding.SourceAttribute == "" {
			out = append(out, errFor(bindingSubject, "HEADER_DETAIL_BINDING_SOURCE_ATTRIBUTE_REQUIRED", "a header-detail Contract Query input binding needs a source attribute"))
		} else {
			if _, err := core.ParseCanonicalID(string(inputBinding.SourceAttribute)); err != nil {
				out = append(out, errFor(bindingSubject, "INVALID_HEADER_DETAIL_BINDING_SOURCE_ATTRIBUTE", "header-detail Contract Query binding source attribute %q is not a valid Canonical ID", inputBinding.SourceAttribute))
			}
			if _, duplicate := seenAttributes[inputBinding.SourceAttribute]; duplicate {
				out = append(out, errFor(bindingSubject, "DUPLICATE_HEADER_DETAIL_BINDING_SOURCE_ATTRIBUTE", "header-detail Contract Query binding %s uses source attribute %s in more than one Query input binding", binding.Identity(), inputBinding.SourceAttribute))
			} else {
				seenAttributes[inputBinding.SourceAttribute] = struct{}{}
			}
		}
		if inputBinding.Input == "" {
			out = append(out, errFor(bindingSubject, "HEADER_DETAIL_BINDING_INPUT_REQUIRED", "a header-detail Contract Query input binding needs a Query input name"))
			continue
		}
		if _, err := core.ParseLocalName(string(inputBinding.Input)); err != nil {
			out = append(out, errFor(bindingSubject, "INVALID_HEADER_DETAIL_BINDING_INPUT", "header-detail Contract Query input name %q is not a valid LocalName", inputBinding.Input))
		}
		if _, duplicate := seenInputs[inputBinding.Input]; duplicate {
			out = append(out, errFor(bindingSubject, "DUPLICATE_HEADER_DETAIL_BINDING_INPUT", "header-detail Contract Query binding %s uses Query input %s in more than one binding", binding.Identity(), inputBinding.Input))
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
				"header-detail Contract Query binding %s source attribute %s does not belong to source Data %s of Relation %s",
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
				"header-detail Contract Query binding %s input %s is not in the signature of Contract Query %s / %s",
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
				"header-detail Contract Query binding %s passes source attribute %s (type %s) to Query input %s (type %s), but the Semantic type identity does not match",
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
				"header-detail Contract Query binding %s has no binding that passes a value to input %s of Contract Query %s / %s",
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
			"header-detail Pattern %s Relation %s targets %s, but Contract Query %s / %s returns %s",
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
			"header-detail Pattern %s requires Contract Query %s / %s to return many for the collection read, but the result cardinality is %s",
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
