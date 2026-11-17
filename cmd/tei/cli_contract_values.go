package main

import (
	"tei/internal/knowledgecontext"
	"tei/internal/modelref"
	"tei/internal/operations"
)

// Closed value sets advertised by `tei commands` (#3242). They point at the
// packages that validate the values, so the catalogue cannot drift from the
// implementation. Open-ended arguments (path, ID, fingerprint, scope) have no
// set and are not listed.
var (
	// inspect resolves the canonical kinds inspectCanonical knows.
	inspectSubjectKinds = operations.InspectionSubjectKinds
	// trace projects the narrower set derivedbridge can place in the graph.
	traceSubjectKinds = operations.TraceSubjectKinds
	traceDirections   = modelref.TraceDirections
	traceModes        = modelref.TraceModes
	// view renders text, json, and the two HTML documents.
	viewFormats = []string{"text", "json", "html", "print"}
	// discloses are the Knowledge sections a disclosure call may select.
	discloses = knowledgecontext.SelectableDisclosureSections
	// candidateKinds are the two Data Design candidate families.
	candidateKinds = []string{
		knowledgecontext.DataDesignCandidateKindBusinessIdentity,
		knowledgecontext.DataDesignCandidateKindFunctionalDependency,
	}
)

// argumentValueSets returns the closed value sets for one catalogue entry, and
// the Go flag type behind each so a caller knows a comma list is allowed.
func argumentValueSets(name string) (map[string][]string, map[string]string) {
	switch name {
	case "inspect":
		return map[string][]string{"subject-kind": inspectSubjectKinds},
			map[string]string{"subject-kind": "string"}
	case "model search":
		// The same closed set inspect resolves: a search restricted to a kind
		// this surface cannot read would answer nothing and look like "this
		// kind declares no such text".
		return map[string][]string{"kind": inspectSubjectKinds},
			map[string]string{"kind": "string"}
	case "trace":
		return map[string][]string{
				"subject-kind": traceSubjectKinds,
				"direction":    traceDirections,
				"mode":         traceModes,
			},
			map[string]string{
				"subject-kind": "string", "direction": "string", "mode": "string",
			}
	case "view":
		return map[string][]string{"format": viewFormats},
			map[string]string{"format": "string"}
	case "knowledge-context":
		return map[string][]string{"disclose": discloses},
			map[string]string{"disclose": "comma-separated string"}
	case "knowledge-data-design-candidate-review":
		return map[string][]string{"data-design-candidate-kind": candidateKinds},
			map[string]string{"data-design-candidate-kind": "string"}
	case "plugin-plan":
		// The closed action set the planner accepts; there is deliberately no
		// apply, because applying is a separate qualified operation (#2550).
		return map[string][]string{"action": operations.PlanActions()},
			map[string]string{"action": "string"}
	case "model ddl":
		// The same closed set the operation accepts: sql (the DDL text) or json
		// (the structured tables); both answers still carry result.ddl.
		return map[string][]string{"format": {operations.ModelDDLFormatSQL, operations.ModelDDLFormatJSON}},
			map[string]string{"format": "string"}
	}
	return nil, nil
}

// valueSets, valueTypes and valueDefaults are the catalogue's accessors. They
// keep commandCatalog() readable and let a caller read one dimension.
func valueSets(name string) map[string][]string {
	sets, _ := argumentValueSets(name)
	return sets
}

func valueTypes(name string) map[string]string {
	_, types := argumentValueSets(name)
	return types
}

func valueDefaults(name string) map[string]string {
	return argumentDefaults(name)
}

// argumentDefaults returns the default of each named flag, when it has one.
// An absent entry means "no default: the value is required or has to be given".
func argumentDefaults(name string) map[string]string {
	switch name {
	case "trace":
		// The same values the flag defaults use, so an advertised default cannot
		// drift from the value an omitted flag applies (#3605).
		return map[string]string{
			"direction": modelref.TraceDefaultDirection,
			"mode":      modelref.TraceDefaultMode,
		}
	case "view":
		return map[string]string{"format": "text"}
	case "model ddl":
		// The same default the flag applies, so an advertised default cannot
		// drift from the value an omitted -format uses.
		return map[string]string{"format": operations.ModelDDLFormatSQL}
	}
	return nil
}
