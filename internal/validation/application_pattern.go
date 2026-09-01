package validation

import (
	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/subject"
)

type applicationPatternQueryRef struct {
	Application core.CanonicalID
	Datasource  core.LocalName
}

// validateApplicationPatternsは、Pattern単体の構文だけでは判断できない
// Workspace全体の参照整合性を検査する。Patternには参照先の定義内容を複製せず、
// Canonical IDで既存資産へ接続するという境界をここで守る。
func validateApplicationPatterns(w ir.World) []Diagnostic {
	var out []Diagnostic

	semanticData := map[core.CanonicalID]struct{}{}
	for _, ns := range w.Semantics {
		for _, data := range ns.Data {
			semanticData[data.ID] = struct{}{}
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

			switch pattern.Kind {
			case application.PatternBrowse:
				out = append(out, validateBrowsePattern(app, pattern, patternSubject, semanticData, queries)...)
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
		if _, ok := app.DatabaseByAlias(pattern.Datasource); !ok {
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
