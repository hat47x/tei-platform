package threeelement

import (
	"sort"
)

// ValidationReport collects derived diagnostics for a model.
type ValidationReport struct {
	SchemaVersion    string `json:"schemaVersion"`
	ModelID          string `json:"modelId"`
	ModelFingerprint string `json:"modelFingerprint"`
	OK               bool   `json:"ok"`
	// OK only concerns the existing structural Diagnostic error count.
	// It cannot be read as approval of contradictory or incomplete CRUD.
	OKScope                string              `json:"okScope"`
	CRUDConflictStatus     string              `json:"crudConflictStatus"`
	ErrorCount             int                 `json:"errorCount"`
	WarningCount           int                 `json:"warningCount"`
	InfoCount              int                 `json:"infoCount"`
	Diagnostics            []Diagnostic        `json:"diagnostics"`
	UnusedData             []string            `json:"unusedData"`
	UnreferencedFunctions  []string            `json:"unreferencedFunctions"`
	MissingBusinessMapping []string            `json:"missingBusinessMapping"`
	ContradictoryCRUDs     []CRUDDefinition    `json:"contradictoryCrudDefinitions,omitempty"`
	UnknownDependencies    [][2]string         `json:"unknownDependencies,omitempty"`
	SameNameConflicts      map[string][]string `json:"sameNameConflicts,omitempty"`
	AuthorityLimits        []string            `json:"authorityLimits,omitempty"`
}

// ValidateModel builds a full validation report.
func ValidateModel(model ThreeElementModel) ValidationReport {
	diags := model.Validate()
	report := ValidationReport{
		SchemaVersion:    model.SchemaVersion,
		ModelID:          model.ModelID,
		ModelFingerprint: model.ModelFingerprint,
		Diagnostics:      diags,
		AuthorityLimits:  model.AuthorityLimits,
		OKScope:          "structural-diagnostics-only",
	}
	for _, d := range diags {
		switch d.Severity {
		case "error":
			report.ErrorCount++
		case "warning":
			report.WarningCount++
		case "info":
			report.InfoCount++
		}
	}
	report.OK = report.ErrorCount == 0
	report.UnusedData = model.UnusedData()
	report.UnreferencedFunctions = model.UnreferencedFunctions()
	report.MissingBusinessMapping = model.MissingBusinessFunctionMapping()
	report.ContradictoryCRUDs = model.ContradictoryCRUDs()
	report.CRUDConflictStatus = "no-conflicts-detected"
	if len(report.ContradictoryCRUDs) > 0 {
		// A conflict requires separate review; it is not automatically a
		// structural hard error or permission to use a last-writer projection.
		report.CRUDConflictStatus = "conflicts-require-review"
	}
	report.UnknownDependencies = model.UnknownDependencies()
	report.SameNameConflicts = model.SameNameConflicts()
	return report
}

// SortedDiagnostics returns diagnostics sorted by severity then code.
func SortedDiagnostics(diags []Diagnostic) []Diagnostic {
	out := append([]Diagnostic(nil), diags...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Severity != out[j].Severity {
			return severityRank(out[i].Severity) < severityRank(out[j].Severity)
		}
		if out[i].Code != out[j].Code {
			return out[i].Code < out[j].Code
		}
		return out[i].Message < out[j].Message
	})
	return out
}
