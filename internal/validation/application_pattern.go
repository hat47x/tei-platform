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
	semanticCapabilities := map[core.CanonicalID]struct{}{}
	for _, ns := range w.Semantics {
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
			semanticCapabilities[capability.ID] = struct{}{}
		}
	}

	queries := map[core.CanonicalID]applicationPatternQueryRef{}
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

			switch pattern.Kind {
			case application.PatternBrowse:
				out = append(out, validateBrowsePattern(app, pattern, patternSubject, semanticData, queries)...)
			case application.PatternRecord:
				out = append(out, validateRecordPattern(app, pattern, patternSubject, semanticData, semanticDataAttributes, patterns, semanticRelations)...)
			case application.PatternReference:
				out = append(out, validateReferencePattern(app, pattern, patternSubject, semanticRelations)...)
			case application.PatternCommand:
				out = append(out, validateCommandPattern(pattern, patternSubject, semanticCapabilities)...)
			case application.PatternHeaderDetail:
				out = append(out, validateHeaderDetailPattern(pattern, patternSubject, semanticRelations)...)
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
		out = append(out, validateRecordPatternInteraction(app, pattern, patternSubject, semanticDataAttributes, patterns, semanticRelations)...)
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

func validateRecordPatternInteraction(
	app application.Application,
	pattern application.Pattern,
	patternSubject subject.Subject,
	semanticDataAttributes map[core.CanonicalID]map[core.CanonicalID]core.CanonicalID,
	patterns map[core.CanonicalID]applicationPatternRef,
	semanticRelations map[core.CanonicalID]applicationPatternRelationRef,
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
	semanticCapabilities map[core.CanonicalID]struct{},
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

	return out
}
