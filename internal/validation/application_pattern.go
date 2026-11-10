package validation

import (
	"strings"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/subject"
)

type applicationPatternQueryRef struct {
	Application core.CanonicalID
	Datasource  core.LocalName
}

type applicationPatternRelationRef struct {
	Source      core.CanonicalID
	Target      core.CanonicalID
	Cardinality semantic.Cardinality
}

type applicationPatternRef struct {
	Application core.CanonicalID
	Pattern     application.Pattern
}

// ValidateApplicationPatternsは、Application Patternが参照する既存資産との整合性を検査する。
// Pattern単体の構文検査とは分け、Workspace全体を読み込んだ後に他のValidationと合成できる形で提供する。
func ValidateApplicationPatterns(w ir.World) []Diagnostic {
	return validateApplicationPatterns(w)
}

// validateApplicationPatternsは、Pattern単体の構文だけでは判断できない
// Workspace全体の参照整合性を検査する。Patternには参照先の定義内容を複製せず、
// Canonical IDで既存資産へ接続するという境界をここで守る。
func validateApplicationPatterns(w ir.World) []Diagnostic {
	var out []Diagnostic

	semanticData := map[core.CanonicalID]struct{}{}
	semanticDataAttributes := map[core.CanonicalID]map[core.CanonicalID]core.CanonicalID{}
	semanticRelations := map[core.CanonicalID]applicationPatternRelationRef{}
	semanticCapabilities := map[core.CanonicalID]core.CanonicalID{}
	semanticCapabilityInputs := map[core.CanonicalID]map[string]core.CanonicalID{}
	// semanticTypes holds the declared type of every identifier a binding may
	// name. A capability parameter and a data attribute can have different
	// types, and a binding that joins them has to agree; without this map there
	// was nothing to compare.
	semanticTypes := map[core.CanonicalID]semantic.Type{}
	for _, ns := range w.Semantics {
		for _, typ := range ns.Types {
			if _, exists := semanticTypes[typ.ID]; exists {
				continue
			}
			semanticTypes[typ.ID] = typ
		}
		for _, data := range ns.Data {
			semanticData[data.ID] = struct{}{}
			attributes := map[core.CanonicalID]core.CanonicalID{}
			for _, attribute := range data.Attributes {
				attributes[attribute.ID] = attribute.TypeRef
			}
			semanticDataAttributes[data.ID] = attributes
			for _, relation := range data.Relations {
				if _, exists := semanticRelations[relation.ID]; exists {
					// Relation自身の重複は既存Semantic Validationが報告する。
					// Pattern側では最初の定義を使い、同じ原因のDiagnosticを増やさない。
					continue
				}
				semanticRelations[relation.ID] = applicationPatternRelationRef{
					Source:      data.ID,
					Target:      relation.Target,
					Cardinality: relation.Cardinality,
				}
			}
		}
		for _, capability := range ns.Capabilities {
			if _, exists := semanticCapabilities[capability.ID]; exists {
				// Capability自身の重複は既存Semantic Validationが報告する。
				// Pattern側では存在確認だけを行い、同じ原因のDiagnosticを増やさない。
				continue
			}
			inputs := map[string]core.CanonicalID{}
			for _, input := range capability.Inputs {
				inputs[string(input.Name)] = input.Type
			}
			semanticCapabilityInputs[capability.ID] = inputs
			semanticCapabilities[capability.ID] = ns.ID
		}
	}

	queries := map[core.CanonicalID]applicationPatternQueryRef{}
	queryInputs := map[core.CanonicalID]map[string]core.CanonicalID{}
	for _, query := range w.Queries {
		if _, exists := queries[query.ID]; exists {
			// Query自身の重複は既存Validationが報告する。
			// Pattern側では最初の定義を使い、同じ原因のDiagnosticを増やさない。
			continue
		}
		queries[query.ID] = applicationPatternQueryRef{
			Application: query.Application,
			Datasource:  query.Datasource,
		}
		inputs := map[string]core.CanonicalID{}
		for _, input := range query.Inputs {
			inputs[string(input.Name)] = input.Type
		}
		queryInputs[query.ID] = inputs
	}

	seenPatternIDs := map[core.CanonicalID]core.CanonicalID{}
	patterns := map[core.CanonicalID]applicationPatternRef{}
	for _, app := range w.Applications {
		for _, pattern := range app.Patterns {
			if _, exists := patterns[pattern.ID]; !exists {
				patterns[pattern.ID] = applicationPatternRef{Application: app.ID, Pattern: pattern}
			}
		}
	}
	// commandCapabilities maps each application to the capabilities its command
	// patterns declare. A hook reaches a capability only through such a command,
	// so the lookup is what makes "declared but unreachable" visible.
	commandCapabilities := map[core.CanonicalID]map[core.CanonicalID]struct{}{}
	for _, app := range w.Applications {
		for _, pattern := range app.Patterns {
			if pattern.Kind != application.PatternCommand || pattern.Capability == "" {
				continue
			}
			if commandCapabilities[app.ID] == nil {
				commandCapabilities[app.ID] = map[core.CanonicalID]struct{}{}
			}
			commandCapabilities[app.ID][pattern.Capability] = struct{}{}
		}
	}
	for _, app := range w.Applications {
		for _, pattern := range app.Patterns {
			patternSubject := subject.Subject{Kind: "application-pattern", ID: string(pattern.ID)}

			if owner, exists := seenPatternIDs[pattern.ID]; exists {
				out = append(out, errFor(
					patternSubject,
					"DUPLICATE_APPLICATION_PATTERN",
					"application pattern %s is declared more than once, by applications %s and %s",
					pattern.ID,
					owner,
					app.ID,
				))
			} else {
				seenPatternIDs[pattern.ID] = app.ID
			}

			if !pattern.Kind.Valid() {
				out = append(out, errFor(
					patternSubject,
					"INVALID_APPLICATION_PATTERN_KIND",
					"application pattern %s has unsupported kind %q",
					pattern.ID,
					pattern.Kind,
				))
				continue
			}

			if pattern.Interaction != nil && pattern.Kind != application.PatternRecord {
				out = append(out, errFor(
					patternSubject,
					"APPLICATION_PATTERN_INTERACTION_NOT_ALLOWED",
					"%s pattern %s does not accept an interaction; the first interaction contract is limited to record patterns",
					pattern.Kind,
					pattern.ID,
				))
			}

			if pattern.Kind != application.PatternHeaderDetail && len(pattern.Aggregates) > 0 {
				out = append(out, errFor(
					patternSubject,
					"APPLICATION_PATTERN_AGGREGATES_NOT_ALLOWED",
					"%s pattern %s does not accept aggregates; aggregates belong to header-detail",
					pattern.Kind,
					pattern.ID,
				))
			}

			if pattern.Kind == application.PatternBrowse {
				out = append(out, validateBrowseFilters(pattern, patternSubject, semanticDataAttributes, queryInputs)...)
				out = append(out, validateBrowseSorts(pattern, patternSubject, semanticDataAttributes)...)
				out = append(out, validateBrowseQueryParameters(pattern, patternSubject, queryInputs)...)
			} else {
				if len(pattern.Filters) > 0 {
					out = append(out, errFor(
						patternSubject,
						"APPLICATION_PATTERN_FILTERS_NOT_ALLOWED",
						"%s pattern %s does not accept filters; filters belong to browse",
						pattern.Kind,
						pattern.ID,
					))
				}
				if len(pattern.Sorts) > 0 {
					out = append(out, errFor(
						patternSubject,
						"APPLICATION_PATTERN_SORTS_NOT_ALLOWED",
						"%s pattern %s does not accept sorts; sorts belong to browse",
						pattern.Kind,
						pattern.ID,
					))
				}
			}

			if pattern.PageSize != 0 {
				switch {
				case pattern.Kind != application.PatternBrowse:
					out = append(out, errFor(
						patternSubject,
						"APPLICATION_PATTERN_PAGE_SIZE_NOT_ALLOWED",
						"%s pattern %s does not accept page-size; page-size belongs to browse",
						pattern.Kind,
						pattern.ID,
					))
				case pattern.PageSize < 1 || pattern.PageSize > 1000:
					out = append(out, errFor(
						patternSubject,
						"APPLICATION_PATTERN_PAGE_SIZE_INVALID",
						"browse pattern %s page-size %d is outside 1..1000",
						pattern.ID,
						pattern.PageSize,
					))
				}
			}

			if pattern.PageBreak != nil {
				switch {
				case pattern.Kind != application.PatternHeaderDetail:
					out = append(out, errFor(
						patternSubject,
						"APPLICATION_PATTERN_PAGE_BREAK_NOT_ALLOWED",
						"%s pattern %s does not accept page-break; page-break belongs to header-detail",
						pattern.Kind,
						pattern.ID,
					))
				case pattern.PageBreak.Rows < 1 || pattern.PageBreak.Rows > 1000:
					out = append(out, errFor(
						patternSubject,
						"APPLICATION_PATTERN_PAGE_BREAK_INVALID",
						"header-detail pattern %s page-break rows %d is outside 1..1000",
						pattern.ID,
						pattern.PageBreak.Rows,
					))
				}
			}

			if pattern.PrintTemplate != "" && pattern.Kind == application.PatternCommand {
				out = append(out, errFor(
					patternSubject,
					"APPLICATION_PATTERN_PRINT_TEMPLATE_NOT_ALLOWED",
					"command pattern %s does not accept print-template",
					pattern.ID,
				))
			}

			if pattern.Layout != nil {
				switch {
				case pattern.Kind != application.PatternRecord:
					out = append(out, errFor(
						patternSubject,
						"APPLICATION_PATTERN_LAYOUT_NOT_ALLOWED",
						"%s pattern %s does not accept layout; layout belongs to record",
						pattern.Kind,
						pattern.ID,
					))
				default:
					attributes := semanticDataAttributes[pattern.Data]
					seen := map[core.CanonicalID]bool{}
					for _, attribute := range pattern.Layout.Order {
						if _, ok := attributes[attribute]; !ok {
							out = append(out, errFor(
								patternSubject,
								"UNKNOWN_APPLICATION_PATTERN_LAYOUT_ORDER",
								"record pattern %s layout refers to %s that is not an attribute of %s",
								pattern.ID,
								attribute,
								pattern.Data,
							))
						}
						if seen[attribute] {
							out = append(out, errFor(
								patternSubject,
								"APPLICATION_PATTERN_LAYOUT_ORDER_DUPLICATE",
								"record pattern %s layout orders %s more than once",
								pattern.ID,
								attribute,
							))
						}
						seen[attribute] = true
					}
				}
			}

			switch pattern.Kind {
			case application.PatternBrowse:
				out = append(out, validateBrowsePattern(app, pattern, patternSubject, semanticData, queries)...)
			case application.PatternRecord:
				out = append(out, validateRecordPattern(app, pattern, patternSubject, semanticData, semanticDataAttributes, patterns, semanticRelations, semanticCapabilities, semanticCapabilityInputs, semanticTypes, commandCapabilities)...)
			case application.PatternReference:
				out = append(out, validateReferencePattern(app, pattern, patternSubject, semanticRelations)...)
			case application.PatternCommand:
				out = append(out, validateCommandPattern(pattern, patternSubject, semanticCapabilities)...)
			case application.PatternHeaderDetail:
				out = append(out, validateHeaderDetailPattern(pattern, patternSubject, semanticRelations, semanticDataAttributes)...)
			}
		}
	}

	return out
}

func validateBrowsePattern(
	app application.Application,
	pattern application.Pattern,
	patternSubject subject.Subject,
	semanticData map[core.CanonicalID]struct{},
	queries map[core.CanonicalID]applicationPatternQueryRef,
) []Diagnostic {
	var out []Diagnostic

	if pattern.Data == "" {
		out = append(out, errFor(
			patternSubject,
			"APPLICATION_PATTERN_DATA_REQUIRED",
			"browse pattern %s requires semantic data",
			pattern.ID,
		))
	} else if _, ok := semanticData[pattern.Data]; !ok {
		out = append(out, errFor(
			patternSubject,
			"UNKNOWN_APPLICATION_PATTERN_DATA",
			"browse pattern %s refers to unknown semantic data %s",
			pattern.ID,
			pattern.Data,
		))
	}

	if pattern.Relation != "" {
		out = append(out, errFor(
			patternSubject,
			"APPLICATION_PATTERN_RELATION_NOT_ALLOWED",
			"browse pattern %s does not accept a relation",
			pattern.ID,
		))
	}
	if pattern.Capability != "" {
		out = append(out, errFor(
			patternSubject,
			"APPLICATION_PATTERN_CAPABILITY_NOT_ALLOWED",
			"browse pattern %s does not accept a capability",
			pattern.ID,
		))
	}

	var queryRef applicationPatternQueryRef
	queryFound := false
	if pattern.Query == "" {
		out = append(out, errFor(
			patternSubject,
			"APPLICATION_PATTERN_QUERY_REQUIRED",
			"browse pattern %s requires a query",
			pattern.ID,
		))
	} else {
		queryRef, queryFound = queries[pattern.Query]
		if !queryFound {
			out = append(out, errFor(
				patternSubject,
				"UNKNOWN_APPLICATION_PATTERN_QUERY",
				"browse pattern %s refers to unknown query %s",
				pattern.ID,
				pattern.Query,
			))
		} else if queryRef.Application != app.ID {
			out = append(out, errFor(
				patternSubject,
				"APPLICATION_PATTERN_QUERY_APPLICATION_MISMATCH",
				"browse pattern %s belongs to application %s but refers to query %s owned by application %s",
				pattern.ID,
				app.ID,
				pattern.Query,
				queryRef.Application,
			))
		}
	}

	if pattern.Datasource != "" {
		if !app.HasDatabaseAlias(pattern.Datasource) {
			out = append(out, errFor(
				patternSubject,
				"UNKNOWN_APPLICATION_PATTERN_DATASOURCE",
				"browse pattern %s refers to unknown datasource alias %s in application %s",
				pattern.ID,
				pattern.Datasource,
				app.ID,
			))
		}
		if queryFound && queryRef.Application == app.ID && queryRef.Datasource != pattern.Datasource {
			out = append(out, errFor(
				patternSubject,
				"APPLICATION_PATTERN_DATASOURCE_QUERY_MISMATCH",
				"browse pattern %s declares datasource %s but query %s uses datasource %s",
				pattern.ID,
				pattern.Datasource,
				pattern.Query,
				queryRef.Datasource,
			))
		}
	}

	return out
}

func validateRecordPattern(
	app application.Application,
	pattern application.Pattern,
	patternSubject subject.Subject,
	semanticData map[core.CanonicalID]struct{},
	semanticDataAttributes map[core.CanonicalID]map[core.CanonicalID]core.CanonicalID,
	patterns map[core.CanonicalID]applicationPatternRef,
	semanticRelations map[core.CanonicalID]applicationPatternRelationRef,
	semanticCapabilities map[core.CanonicalID]core.CanonicalID,
	semanticCapabilityInputs map[core.CanonicalID]map[string]core.CanonicalID,
	semanticTypes map[core.CanonicalID]semantic.Type,
	commandCapabilities map[core.CanonicalID]map[core.CanonicalID]struct{},
) []Diagnostic {
	var out []Diagnostic

	if pattern.Data == "" {
		out = append(out, errFor(
			patternSubject,
			"APPLICATION_PATTERN_DATA_REQUIRED",
			"record pattern %s requires semantic data",
			pattern.ID,
		))
	} else if _, ok := semanticData[pattern.Data]; !ok {
		out = append(out, errFor(
			patternSubject,
			"UNKNOWN_APPLICATION_PATTERN_DATA",
			"record pattern %s refers to unknown semantic data %s",
			pattern.ID,
			pattern.Data,
		))
	} else {
		out = append(out, validateRecordPatternInteraction(app, pattern, patternSubject, semanticDataAttributes, patterns, semanticRelations, semanticCapabilities, semanticCapabilityInputs, semanticTypes, commandCapabilities)...)
	}

	if pattern.Relation != "" {
		out = append(out, errFor(
			patternSubject,
			"APPLICATION_PATTERN_RELATION_NOT_ALLOWED",
			"record pattern %s does not accept a relation",
			pattern.ID,
		))
	}
	if pattern.Query != "" {
		out = append(out, errFor(
			patternSubject,
			"APPLICATION_PATTERN_QUERY_NOT_ALLOWED",
			"record pattern %s does not accept a query",
			pattern.ID,
		))
	}
	if pattern.Capability != "" {
		out = append(out, errFor(
			patternSubject,
			"APPLICATION_PATTERN_CAPABILITY_NOT_ALLOWED",
			"record pattern %s does not accept a capability",
			pattern.ID,
		))
	}
	if pattern.Datasource != "" {
		out = append(out, errFor(
			patternSubject,
			"APPLICATION_PATTERN_DATASOURCE_NOT_ALLOWED",
			"record pattern %s does not accept a datasource",
			pattern.ID,
		))
	}

	return out
}

// bothPrimitive reports whether two declared types both resolve to a std
// primitive. A Data reference is not a primitive, so a binding that names one
// side as Data is left to the Command boundary's value checking.
func bothPrimitive(types map[core.CanonicalID]semantic.Type, left, right core.CanonicalID) bool {
	return isPrimitive(underlyingType(types, left)) && isPrimitive(underlyingType(types, right))
}

// isPrimitive reports whether a type identifier is a std primitive.
func isPrimitive(id core.CanonicalID) bool {
	return len(id) > 4 && string(id)[:4] == "std."
}

// sameType reports whether two declared types are the same type.
//
// Aliases are followed to their underlying type on both sides, so
// `type OrderId from std.identifier` and `std.identifier` agree. A cycle or a
// missing definition stops the walk and compares what was reached, so a broken
// declaration is not reported twice: the type layer reports it, and this check
// stays silent rather than guessing.
func sameType(types map[core.CanonicalID]semantic.Type, left, right core.CanonicalID) bool {
	return underlyingType(types, left) == underlyingType(types, right)
}

// underlyingType follows alias declarations to the type they ultimately name.
func underlyingType(types map[core.CanonicalID]semantic.Type, id core.CanonicalID) core.CanonicalID {
	current := id
	seen := map[core.CanonicalID]bool{}
	for depth := 0; depth < 32; depth++ {
		if seen[current] {
			return current
		}
		seen[current] = true
		typ, ok := types[current]
		if !ok || typ.Kind != semantic.TypeAlias || typ.Underlying == "" {
			return current
		}
		current = typ.Underlying
	}
	return current
}

func validateRecordPatternInteraction(
	app application.Application,
	pattern application.Pattern,
	patternSubject subject.Subject,
	semanticDataAttributes map[core.CanonicalID]map[core.CanonicalID]core.CanonicalID,
	patterns map[core.CanonicalID]applicationPatternRef,
	semanticRelations map[core.CanonicalID]applicationPatternRelationRef,
	semanticCapabilities map[core.CanonicalID]core.CanonicalID,
	semanticCapabilityInputs map[core.CanonicalID]map[string]core.CanonicalID,
	semanticTypes map[core.CanonicalID]semantic.Type,
	commandCapabilities map[core.CanonicalID]map[core.CanonicalID]struct{},
) []Diagnostic {
	var out []Diagnostic
	if pattern.Interaction == nil {
		return nil
	}

	dataAttributes := semanticDataAttributes[pattern.Data]
	seen := map[core.CanonicalID]struct{}{}
	for _, item := range pattern.Interaction.Items {
		if _, exists := seen[item.Attribute]; exists {
			out = append(out, errFor(
				patternSubject,
				"APPLICATION_PATTERN_INTERACTION_ITEM_DUPLICATE",
				"record pattern %s declares interaction item %s more than once",
				pattern.ID,
				item.Attribute,
			))
			continue
		}
		seen[item.Attribute] = struct{}{}

		itemType, ok := dataAttributes[item.Attribute]
		if !ok {
			out = append(out, errFor(
				patternSubject,
				"UNKNOWN_APPLICATION_PATTERN_INTERACTION_ITEM",
				"record pattern %s declares interaction item %s that is not an attribute of %s",
				pattern.ID,
				item.Attribute,
				pattern.Data,
			))
			continue
		}
		if item.Lookup == nil {
			continue
		}

		ref, ok := patterns[item.Lookup.Pattern]
		if !ok {
			out = append(out, errFor(
				patternSubject,
				"UNKNOWN_APPLICATION_PATTERN_INTERACTION_LOOKUP_PATTERN",
				"record pattern %s interaction item %s refers to unknown lookup pattern %s",
				pattern.ID,
				item.Attribute,
				item.Lookup.Pattern,
			))
			continue
		}
		if ref.Application != app.ID {
			out = append(out, errFor(
				patternSubject,
				"APPLICATION_PATTERN_INTERACTION_LOOKUP_APPLICATION_MISMATCH",
				"record pattern %s belongs to application %s but its lookup refers to pattern %s owned by application %s",
				pattern.ID,
				app.ID,
				item.Lookup.Pattern,
				ref.Application,
			))
			continue
		}
		if ref.Pattern.Kind != application.PatternReference {
			out = append(out, errFor(
				patternSubject,
				"APPLICATION_PATTERN_INTERACTION_LOOKUP_NOT_REFERENCE",
				"record pattern %s interaction item %s lookup target %s is a %s pattern, not a reference pattern",
				pattern.ID,
				item.Attribute,
				item.Lookup.Pattern,
				ref.Pattern.Kind,
			))
			continue
		}

		relation, ok := semanticRelations[ref.Pattern.Relation]
		if !ok {
			// 参照Pattern自身のRelation不整合はreference PatternのValidationが報告する。
			continue
		}
		if !applicationUsesNamespace(app, relation.Target) {
			out = append(out, errFor(
				patternSubject,
				"APPLICATION_PATTERN_INTERACTION_LOOKUP_SEMANTIC_NOT_USED",
				"record pattern %s interaction item %s lookup target %s belongs to a semantic namespace that application %s does not use",
				pattern.ID,
				item.Attribute,
				relation.Target,
				app.ID,
			))
			continue
		}
		targetAttributes := semanticDataAttributes[relation.Target]
		targetType, ok := targetAttributes[item.Lookup.TargetAttribute]
		if !ok {
			out = append(out, errFor(
				patternSubject,
				"UNKNOWN_APPLICATION_PATTERN_INTERACTION_LOOKUP_ATTRIBUTE",
				"record pattern %s interaction item %s lookup target %s does not have attribute %s",
				pattern.ID,
				item.Attribute,
				relation.Target,
				item.Lookup.TargetAttribute,
			))
			continue
		}
		if itemType != targetType {
			out = append(out, errFor(
				patternSubject,
				"APPLICATION_PATTERN_INTERACTION_LOOKUP_TYPE_MISMATCH",
				"record pattern %s interaction item %s has type %s but lookup target %s has type %s",
				pattern.ID,
				item.Attribute,
				itemType,
				item.Lookup.TargetAttribute,
				targetType,
			))
		}
	}

	usedSemantics := map[core.CanonicalID]bool{}
	for _, namespace := range app.SemanticUses {
		usedSemantics[namespace] = true
	}
	for _, command := range pattern.Interaction.Commands {
		namespace, ok := semanticCapabilities[command.Capability]
		if !ok {
			out = append(out, errFor(
				patternSubject,
				"UNKNOWN_APPLICATION_PATTERN_INTERACTION_COMMAND",
				"record pattern %s declares interaction command %s that is not a capability",
				pattern.ID,
				command.Capability,
			))
			continue
		}
		if !usedSemantics[namespace] {
			out = append(out, errFor(
				patternSubject,
				"APPLICATION_PATTERN_INTERACTION_COMMAND_SEMANTIC_NOT_USED",
				"record pattern %s declares interaction command %s from namespace %s that application %s does not use",
				pattern.ID,
				command.Capability,
				namespace,
				app.ID,
			))
		}
		capabilityInputs := semanticCapabilityInputs[command.Capability]
		seenInputs := map[string]bool{}
		for _, binding := range command.Inputs {
			if _, ok := capabilityInputs[binding.Name]; !ok {
				out = append(out, errFor(
					patternSubject,
					"UNKNOWN_APPLICATION_PATTERN_INTERACTION_COMMAND_INPUT",
					"record pattern %s binds command %s input %s that the capability does not declare",
					pattern.ID,
					command.Capability,
					binding.Name,
				))
			}
			if seenInputs[binding.Name] {
				out = append(out, errFor(
					patternSubject,
					"APPLICATION_PATTERN_INTERACTION_COMMAND_INPUT_DUPLICATE",
					"record pattern %s binds command %s input %s more than once",
					pattern.ID,
					command.Capability,
					binding.Name,
				))
			}
			seenInputs[binding.Name] = true
			parameterType, parameterOK := capabilityInputs[binding.Name]
			// A binding joins an attribute to a capability parameter. Two
			// failures are possible, and the more specific one is reported.
			//
			// A parameter whose type names a Data is fed by one of that Data's
			// attributes. Binding another Data's attribute supplies a value of
			// the wrong record, and nothing downstream inspects the record
			// identity: the Command boundary checks the value's shape, not which
			// Data it came from.
			if parameterOK {
				if _, namesData := semanticDataAttributes[parameterType]; namesData {
					if _, attributeOfThatData := semanticDataAttributes[parameterType][binding.Attribute]; !attributeOfThatData {
						out = append(out, errFor(
							patternSubject,
							"APPLICATION_PATTERN_INTERACTION_COMMAND_INPUT_DATA_MISMATCH",
							"record pattern %s binds command %s input %s of data %s to attribute %s that is not an attribute of %s",
							pattern.ID,
							command.Capability,
							binding.Name,
							parameterType,
							binding.Attribute,
							parameterType,
						))
					}
				}
			}
			attributeType, ok := dataAttributes[binding.Attribute]
			if !ok {
				out = append(out, errFor(
					patternSubject,
					"UNKNOWN_APPLICATION_PATTERN_INTERACTION_COMMAND_BINDING_ATTRIBUTE",
					"record pattern %s binds command %s input %s to %s that is not an attribute of %s",
					pattern.ID,
					command.Capability,
					binding.Name,
					binding.Attribute,
					pattern.Data,
				))
				continue
			}
			// When both sides are primitives, a difference is a real defect: the
			// bound value would reach the capability as a value of a type it
			// does not declare. When either side names a Data rather than a
			// primitive, the correspondence is not a type equality — the
			// Data-typed parameter case is handled above — so it is not
			// compared here.
			if parameterOK && bothPrimitive(semanticTypes, attributeType, parameterType) && !sameType(semanticTypes, attributeType, parameterType) {
				out = append(out, errFor(
					patternSubject,
					"APPLICATION_PATTERN_INTERACTION_COMMAND_INPUT_TYPE_MISMATCH",
					"record pattern %s binds command %s input %s of type %s to attribute %s of type %s",
					pattern.ID,
					command.Capability,
					binding.Name,
					parameterType,
					binding.Attribute,
					attributeType,
				))
			}
		}
	}

	seenHookPhases := map[string]bool{}
	for _, hook := range pattern.Interaction.Hooks {
		switch hook.Phase {
		case "before-read", "before-create", "after-create", "before-update", "after-update", "before-delete", "after-delete":
		default:
			out = append(out, errFor(patternSubject, "APPLICATION_PATTERN_INTERACTION_HOOK_PHASE_UNSUPPORTED", "record pattern %s hook uses unsupported phase %s", pattern.ID, hook.Phase))
		}
		// One phase names one hook. Two hooks with the same phase would leave the
		// second silently unexecuted, because the runner resolves a phase to one
		// declaration.
		if seenHookPhases[hook.Phase] {
			out = append(out, errFor(patternSubject, "APPLICATION_PATTERN_INTERACTION_HOOK_PHASE_DUPLICATE", "record pattern %s declares more than one %s hook", pattern.ID, hook.Phase))
		}
		seenHookPhases[hook.Phase] = true
		switch hook.Failure {
		case "", "reject", "warn":
		default:
			out = append(out, errFor(patternSubject, "APPLICATION_PATTERN_INTERACTION_HOOK_FAILURE_UNSUPPORTED", "record pattern %s hook %s uses unsupported failure %s", pattern.ID, hook.Phase, hook.Failure))
		}
		// The scope says what the hook decides for. An empty scope means the
		// pattern's own unit, so a scope the pattern does not look at is refused:
		// a per-row hook on a record pattern would be declared and then never
		// applied per row.
		switch hook.Scope {
		case "":
		case "row", "record", "item", "header":
			if kind := pattern.Kind; !scopeFitsPattern(kind, hook.Scope) {
				out = append(out, errFor(patternSubject, "APPLICATION_PATTERN_INTERACTION_HOOK_SCOPE_MISMATCH", "record pattern %s hook %s declares scope %s that the pattern kind does not look at", pattern.ID, hook.Phase, hook.Scope))
			}
		default:
			out = append(out, errFor(patternSubject, "APPLICATION_PATTERN_INTERACTION_HOOK_SCOPE_UNSUPPORTED", "record pattern %s hook %s uses unsupported scope %s", pattern.ID, hook.Phase, hook.Scope))
		}
		for _, attribute := range hook.Reads {
			if _, ok := dataAttributes[attribute]; !ok {
				out = append(out, errFor(patternSubject, "UNKNOWN_APPLICATION_PATTERN_INTERACTION_HOOK_READ", "record pattern %s hook %s reads %s that is not an attribute of %s", pattern.ID, hook.Phase, attribute, pattern.Data))
			}
		}
		for _, attribute := range hook.Produces {
			if _, ok := dataAttributes[attribute]; !ok {
				out = append(out, errFor(patternSubject, "UNKNOWN_APPLICATION_PATTERN_INTERACTION_HOOK_PRODUCE", "record pattern %s hook %s produces %s that is not an attribute of %s", pattern.ID, hook.Phase, attribute, pattern.Data))
			}
		}
		for _, capability := range hook.Uses {
			namespace, ok := semanticCapabilities[capability]
			if !ok {
				out = append(out, errFor(patternSubject, "UNKNOWN_APPLICATION_PATTERN_INTERACTION_HOOK_USE", "record pattern %s hook %s uses %s that is not a capability", pattern.ID, hook.Phase, capability))
				continue
			}
			if !usedSemantics[namespace] {
				out = append(out, errFor(patternSubject, "APPLICATION_PATTERN_INTERACTION_HOOK_SEMANTIC_NOT_USED", "record pattern %s hook %s uses %s from namespace %s that application %s does not use", pattern.ID, hook.Phase, capability, namespace, app.ID))
			}
			// A hook reaches a capability through the Command boundary, so the
			// application must declare a command pattern for it. Without one the
			// hook could be declared and validated but never run, which is what
			// fail-closed means here: refuse the declaration instead of leaving a
			// hook that silently cannot execute.
			if _, declared := commandCapabilities[app.ID][capability]; !declared {
				out = append(out, errFor(patternSubject, "APPLICATION_PATTERN_INTERACTION_HOOK_USE_WITHOUT_COMMAND", "record pattern %s hook %s uses %s but application %s declares no command pattern for it", pattern.ID, hook.Phase, capability, app.ID))
			}
		}
	}

	for _, navigation := range pattern.Interaction.Navigations {
		ref, ok := patterns[navigation.From]
		if !ok {
			out = append(out, errFor(
				patternSubject,
				"UNKNOWN_APPLICATION_PATTERN_INTERACTION_NAVIGATION",
				"record pattern %s declares navigation from unknown pattern %s",
				pattern.ID,
				navigation.From,
			))
			continue
		}
		if ref.Application != app.ID {
			out = append(out, errFor(
				patternSubject,
				"APPLICATION_PATTERN_INTERACTION_NAVIGATION_APPLICATION_MISMATCH",
				"record pattern %s belongs to application %s but its navigation source %s belongs to application %s",
				pattern.ID,
				app.ID,
				navigation.From,
				ref.Application,
			))
			continue
		}
		if ref.Pattern.Kind != application.PatternBrowse {
			out = append(out, errFor(
				patternSubject,
				"APPLICATION_PATTERN_INTERACTION_NAVIGATION_NOT_BROWSE",
				"record pattern %s navigation source %s is a %s pattern, not a browse pattern",
				pattern.ID,
				navigation.From,
				ref.Pattern.Kind,
			))
			continue
		}
		if ref.Pattern.Data != pattern.Data {
			out = append(out, errFor(
				patternSubject,
				"APPLICATION_PATTERN_INTERACTION_NAVIGATION_DATA_MISMATCH",
				"record pattern %s uses data %s but navigation source %s uses data %s",
				pattern.ID,
				pattern.Data,
				navigation.From,
				ref.Pattern.Data,
			))
		}
	}
	return out
}

func applicationUsesNamespace(app application.Application, id core.CanonicalID) bool {
	namespace, _, ok := strings.Cut(string(id), ".")
	if !ok {
		return false
	}
	for _, used := range app.SemanticUses {
		if string(used) == namespace {
			return true
		}
	}
	return false
}

func validateReferencePattern(
	app application.Application,
	pattern application.Pattern,
	patternSubject subject.Subject,
	semanticRelations map[core.CanonicalID]applicationPatternRelationRef,
) []Diagnostic {
	var out []Diagnostic

	if pattern.Relation == "" {
		out = append(out, errFor(
			patternSubject,
			"APPLICATION_PATTERN_RELATION_REQUIRED",
			"reference pattern %s requires a semantic relation",
			pattern.ID,
		))
	} else if _, ok := semanticRelations[pattern.Relation]; !ok {
		out = append(out, errFor(
			patternSubject,
			"UNKNOWN_APPLICATION_PATTERN_RELATION",
			"reference pattern %s refers to unknown semantic relation %s",
			pattern.ID,
			pattern.Relation,
		))
	}

	if pattern.Datasource == "" {
		out = append(out, errFor(
			patternSubject,
			"APPLICATION_PATTERN_DATASOURCE_REQUIRED",
			"reference pattern %s requires a datasource",
			pattern.ID,
		))
	} else if !app.HasDatabaseAlias(pattern.Datasource) {
		out = append(out, errFor(
			patternSubject,
			"UNKNOWN_APPLICATION_PATTERN_DATASOURCE",
			"reference pattern %s refers to unknown datasource alias %s in application %s",
			pattern.ID,
			pattern.Datasource,
			app.ID,
		))
	} else if datasource, ok := app.DatabaseByAliasExact(pattern.Datasource); ok && datasource.Access == database.AccessWrite {
		out = append(out, errFor(
			patternSubject,
			"APPLICATION_PATTERN_DATASOURCE_NOT_READABLE",
			"reference pattern %s uses datasource %s whose access mode %s does not allow reads",
			pattern.ID,
			pattern.Datasource,
			datasource.Access,
		))
	}

	if pattern.Data != "" {
		out = append(out, errFor(
			patternSubject,
			"APPLICATION_PATTERN_DATA_NOT_ALLOWED",
			"reference pattern %s does not accept semantic data because the relation already identifies its source and target",
			pattern.ID,
		))
	}
	if pattern.Query != "" {
		out = append(out, errFor(
			patternSubject,
			"APPLICATION_PATTERN_QUERY_NOT_ALLOWED",
			"reference pattern %s does not accept a query",
			pattern.ID,
		))
	}
	if pattern.Capability != "" {
		out = append(out, errFor(
			patternSubject,
			"APPLICATION_PATTERN_CAPABILITY_NOT_ALLOWED",
			"reference pattern %s does not accept a capability",
			pattern.ID,
		))
	}

	return out
}

func validateCommandPattern(
	pattern application.Pattern,
	patternSubject subject.Subject,
	semanticCapabilities map[core.CanonicalID]core.CanonicalID,
) []Diagnostic {
	var out []Diagnostic

	if pattern.Capability == "" {
		out = append(out, errFor(
			patternSubject,
			"APPLICATION_PATTERN_CAPABILITY_REQUIRED",
			"command pattern %s requires a semantic capability",
			pattern.ID,
		))
	} else if _, ok := semanticCapabilities[pattern.Capability]; !ok {
		out = append(out, errFor(
			patternSubject,
			"UNKNOWN_APPLICATION_PATTERN_CAPABILITY",
			"command pattern %s refers to unknown semantic capability %s",
			pattern.ID,
			pattern.Capability,
		))
	}

	if pattern.Data != "" {
		out = append(out, errFor(
			patternSubject,
			"APPLICATION_PATTERN_DATA_NOT_ALLOWED",
			"command pattern %s does not accept semantic data because the capability already declares its inputs and result contract",
			pattern.ID,
		))
	}
	if pattern.Relation != "" {
		out = append(out, errFor(
			patternSubject,
			"APPLICATION_PATTERN_RELATION_NOT_ALLOWED",
			"command pattern %s does not accept a relation",
			pattern.ID,
		))
	}
	if pattern.Query != "" {
		out = append(out, errFor(
			patternSubject,
			"APPLICATION_PATTERN_QUERY_NOT_ALLOWED",
			"command pattern %s does not accept a query",
			pattern.ID,
		))
	}
	if pattern.Datasource != "" {
		out = append(out, errFor(
			patternSubject,
			"APPLICATION_PATTERN_DATASOURCE_NOT_ALLOWED",
			"command pattern %s does not accept a datasource because implementation resources remain outside the Application Pattern contract",
			pattern.ID,
		))
	}

	return out
}

func validateHeaderDetailPattern(
	pattern application.Pattern,
	patternSubject subject.Subject,
	semanticRelations map[core.CanonicalID]applicationPatternRelationRef,
	semanticDataAttributes map[core.CanonicalID]map[core.CanonicalID]core.CanonicalID,
) []Diagnostic {
	var out []Diagnostic

	if pattern.Relation == "" {
		out = append(out, errFor(
			patternSubject,
			"APPLICATION_PATTERN_RELATION_REQUIRED",
			"header-detail pattern %s requires a semantic relation",
			pattern.ID,
		))
	} else if relation, ok := semanticRelations[pattern.Relation]; !ok {
		out = append(out, errFor(
			patternSubject,
			"UNKNOWN_APPLICATION_PATTERN_RELATION",
			"header-detail pattern %s refers to unknown semantic relation %s",
			pattern.ID,
			pattern.Relation,
		))
	} else if relation.Cardinality != semantic.CardinalityMany {
		out = append(out, errFor(
			patternSubject,
			"APPLICATION_PATTERN_RELATION_NOT_MANY",
			"header-detail pattern %s requires a many relation but %s has cardinality %s",
			pattern.ID,
			pattern.Relation,
			relation.Cardinality,
		))
	}

	if pattern.Data != "" {
		out = append(out, errFor(
			patternSubject,
			"APPLICATION_PATTERN_DATA_NOT_ALLOWED",
			"header-detail pattern %s does not accept semantic data because the relation already identifies the header and detail data",
			pattern.ID,
		))
	}
	if pattern.Query != "" {
		out = append(out, errFor(
			patternSubject,
			"APPLICATION_PATTERN_QUERY_NOT_ALLOWED",
			"header-detail pattern %s does not accept a query",
			pattern.ID,
		))
	}
	if pattern.Capability != "" {
		out = append(out, errFor(
			patternSubject,
			"APPLICATION_PATTERN_CAPABILITY_NOT_ALLOWED",
			"header-detail pattern %s does not accept a capability",
			pattern.ID,
		))
	}
	if pattern.Datasource != "" {
		out = append(out, errFor(
			patternSubject,
			"APPLICATION_PATTERN_DATASOURCE_NOT_ALLOWED",
			"header-detail pattern %s does not accept a datasource because data access remains outside this Pattern contract",
			pattern.ID,
		))
	}

	relation, relationOK := semanticRelations[pattern.Relation]
	detailAttributes := semanticDataAttributes[relation.Target]
	seenAggregates := map[string]bool{}
	for _, aggregate := range pattern.Aggregates {
		if aggregate.Name == "" {
			out = append(out, errFor(patternSubject, "APPLICATION_PATTERN_AGGREGATE_NAME_REQUIRED", "header-detail pattern %s has an aggregate without a name", pattern.ID))
			continue
		}
		if seenAggregates[aggregate.Name] {
			out = append(out, errFor(patternSubject, "APPLICATION_PATTERN_AGGREGATE_DUPLICATE", "header-detail pattern %s declares aggregate %s more than once", pattern.ID, aggregate.Name))
		}
		seenAggregates[aggregate.Name] = true
		switch aggregate.Function {
		case "sum", "count", "min", "max", "avg":
		default:
			out = append(out, errFor(patternSubject, "APPLICATION_PATTERN_AGGREGATE_FUNCTION_UNSUPPORTED", "header-detail pattern %s aggregate %s uses unsupported function %s", pattern.ID, aggregate.Name, aggregate.Function))
		}
		if !relationOK {
			continue
		}
		if _, ok := detailAttributes[aggregate.Attribute]; !ok {
			out = append(out, errFor(patternSubject, "UNKNOWN_APPLICATION_PATTERN_AGGREGATE_ATTRIBUTE", "header-detail pattern %s aggregate %s refers to %s that is not an attribute of %s", pattern.ID, aggregate.Name, aggregate.Attribute, relation.Target))
		}
	}

	return out
}

func validateBrowseFilters(
	pattern application.Pattern,
	patternSubject subject.Subject,
	semanticDataAttributes map[core.CanonicalID]map[core.CanonicalID]core.CanonicalID,
	queryInputs map[core.CanonicalID]map[string]core.CanonicalID,
) []Diagnostic {
	var out []Diagnostic
	attributes := semanticDataAttributes[pattern.Data]
	seen := map[core.CanonicalID]bool{}
	for _, filter := range pattern.Filters {
		if _, ok := attributes[filter.Attribute]; !ok {
			out = append(out, errFor(patternSubject, "UNKNOWN_APPLICATION_PATTERN_FILTER_ATTRIBUTE", "browse pattern %s filter refers to %s that is not an attribute of %s", pattern.ID, filter.Attribute, pattern.Data))
		}
		if len(filter.Operators) == 0 {
			out = append(out, errFor(patternSubject, "APPLICATION_PATTERN_FILTER_OPERATORS_REQUIRED", "browse pattern %s filter %s requires at least one operator", pattern.ID, filter.Attribute))
		}
		for _, operator := range filter.Operators {
			switch operator {
			case "eq", "ne", "ge", "gt", "le", "lt", "scan", "generic":
			default:
				out = append(out, errFor(patternSubject, "APPLICATION_PATTERN_FILTER_OPERATOR_UNSUPPORTED", "browse pattern %s filter %s uses unsupported operator %s", pattern.ID, filter.Attribute, operator))
			}
		}
		if filter.Parameter != "" {
			inputs := queryInputs[pattern.Query]
			parameterType, ok := inputs[filter.Parameter]
			if !ok {
				out = append(out, errFor(patternSubject, "UNKNOWN_APPLICATION_PATTERN_FILTER_PARAMETER", "browse pattern %s filter %s binds unknown query input %s", pattern.ID, filter.Attribute, filter.Parameter))
			} else if attributeType, exists := attributes[filter.Attribute]; exists && attributeType != parameterType {
				out = append(out, errFor(patternSubject, "APPLICATION_PATTERN_FILTER_PARAMETER_TYPE_MISMATCH", "browse pattern %s filter %s has type %s but query input %s has type %s", pattern.ID, filter.Attribute, attributeType, filter.Parameter, parameterType))
			}
		}
		if seen[filter.Attribute] {
			out = append(out, errFor(patternSubject, "APPLICATION_PATTERN_FILTER_DUPLICATE", "browse pattern %s declares filter %s more than once", pattern.ID, filter.Attribute))
		}
		seen[filter.Attribute] = true
	}
	return out
}

// validateBrowseQueryParametersは、Patternが宣言した演算子・page-sizeの受け口が、Queryのinputとして
// 実在し型が合うことだけを確かめる。native SQL本文は解析しない。SQLがその値を実際に使うかは
// 実DBでの独立Expectedで確かめる。
func validateBrowseQueryParameters(
	pattern application.Pattern,
	patternSubject subject.Subject,
	queryInputs map[core.CanonicalID]map[string]core.CanonicalID,
) []Diagnostic {
	var out []Diagnostic
	inputs := queryInputs[pattern.Query]
	check := func(name, wantType, unknownCode, mismatchCode, what string) {
		if name == "" {
			return
		}
		got, ok := inputs[name]
		switch {
		case !ok:
			out = append(out, errFor(patternSubject, unknownCode, "browse pattern %s %s binds unknown query input %s", pattern.ID, what, name))
		case string(got) != wantType:
			out = append(out, errFor(patternSubject, mismatchCode, "browse pattern %s %s needs %s but query input %s has type %s", pattern.ID, what, wantType, name, got))
		}
	}
	for _, filter := range pattern.Filters {
		check(filter.OperatorParameter, "std.code", "UNKNOWN_APPLICATION_PATTERN_FILTER_OPERATOR_PARAMETER",
			"APPLICATION_PATTERN_FILTER_OPERATOR_PARAMETER_TYPE_MISMATCH", "filter "+string(filter.Attribute)+" operator-parameter")
	}
	if pattern.PageSizeParameter != "" || pattern.PageOffsetParameter != "" {
		if pattern.PageSize == 0 {
			out = append(out, errFor(patternSubject, "APPLICATION_PATTERN_PAGE_PARAMETER_REQUIRES_PAGE_SIZE",
				"browse pattern %s declares a page parameter without page-size", pattern.ID))
		}
		check(pattern.PageSizeParameter, "std.integer", "UNKNOWN_APPLICATION_PATTERN_PAGE_PARAMETER",
			"APPLICATION_PATTERN_PAGE_PARAMETER_TYPE_MISMATCH", "page-size parameter")
		check(pattern.PageOffsetParameter, "std.integer", "UNKNOWN_APPLICATION_PATTERN_PAGE_PARAMETER",
			"APPLICATION_PATTERN_PAGE_PARAMETER_TYPE_MISMATCH", "page-size offset")
	}
	return out
}

func validateBrowseSorts(
	pattern application.Pattern,
	patternSubject subject.Subject,
	semanticDataAttributes map[core.CanonicalID]map[core.CanonicalID]core.CanonicalID,
) []Diagnostic {
	var out []Diagnostic
	attributes := semanticDataAttributes[pattern.Data]
	seen := map[core.CanonicalID]bool{}
	for _, sort := range pattern.Sorts {
		if _, ok := attributes[sort.Attribute]; !ok {
			out = append(out, errFor(patternSubject, "UNKNOWN_APPLICATION_PATTERN_SORT_ATTRIBUTE", "browse pattern %s sort refers to %s that is not an attribute of %s", pattern.ID, sort.Attribute, pattern.Data))
		}
		switch sort.Direction {
		case "asc", "desc":
		default:
			out = append(out, errFor(patternSubject, "APPLICATION_PATTERN_SORT_DIRECTION_UNSUPPORTED", "browse pattern %s sort %s uses unsupported direction %s", pattern.ID, sort.Attribute, sort.Direction))
		}
		if seen[sort.Attribute] {
			out = append(out, errFor(patternSubject, "APPLICATION_PATTERN_SORT_DUPLICATE", "browse pattern %s declares sort %s more than once", pattern.ID, sort.Attribute))
		}
		seen[sort.Attribute] = true
	}
	return out
}
