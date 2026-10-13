package main

// A read-only view of the Database Contract Query requested by an existing
// Application Pattern. This is not a Provider resolver or runtime authority.

import (
	"tei/internal/application"
	"tei/internal/ir"
)

// DeclaredContractQuery is a locally derived authoring hint, not a second
// Contract/Provider registry. "declared-requirement" means only that the
// Application has authored a matching dependency.
type DeclaredContractQuery struct {
	Status          string `json:"status"`
	Reason          string `json:"reason"`
	DatasourceAlias string `json:"datasourceAlias,omitempty"`
	TargetDatabase  string `json:"targetDatabase,omitempty"`
	Contract        string `json:"contract,omitempty"`
	Query           string `json:"query,omitempty"`
	CompatibleMajor int    `json:"compatibleMajor,omitempty"`
}

func ObserveDeclaredContractQuery(w ir.World, applicationID, patternID string) DeclaredContractQuery {
	out := DeclaredContractQuery{Status: "unresolved", Reason: "application-or-pattern-not-unique"}
	var app application.Application
	var pattern application.Pattern
	appCount, patternCount := 0, 0
	for _, candidate := range w.Applications {
		if string(candidate.ID) != applicationID {
			continue
		}
		appCount++
		app = candidate
		for _, p := range candidate.Patterns {
			if string(p.ID) == patternID {
				patternCount++
				pattern = p
			}
		}
	}
	if appCount != 1 || patternCount != 1 {
		return out
	}
	if pattern.Kind != application.PatternReference {
		out.Status, out.Reason = "not-reference", "not-a-reference-query-pattern"
		return out
	}
	if pattern.ContractQuery == nil {
		out.Status, out.Reason = "not-declared", "reference-contract-query-not-declared"
		return out
	}
	out.DatasourceAlias = string(pattern.Datasource)
	out.Contract = string(pattern.ContractQuery.Contract)
	out.Query = string(pattern.ContractQuery.Query)
	if out.DatasourceAlias == "" || out.Contract == "" || out.Query == "" {
		out.Reason = "reference-contract-query-incomplete"
		return out
	}
	// Use the existing exact alias resolver; do not choose one of multiple
	// declarations or infer a Database from a query name.
	db, ok := app.DatabaseByAliasExact(pattern.Datasource)
	if !ok {
		out.Reason = "datasource-alias-not-unique"
		return out
	}
	out.TargetDatabase = string(db.TargetDatabase)
	if out.TargetDatabase == "" {
		out.Reason = "target-database-not-declared"
		return out
	}
	matches := 0
	for _, required := range db.RequiredContracts {
		if string(required.ContractID) == out.Contract {
			matches++
			out.CompatibleMajor = required.CompatibleMajor
		}
	}
	if matches != 1 || out.CompatibleMajor <= 0 {
		out.Reason = "contract-requirement-not-unique-or-absent"
		return out
	}
	out.Status = "declared-requirement"
	out.Reason = "application-declares-contract-dependency-provider-not-verified"
	return out
}
