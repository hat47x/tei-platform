package crossview

import (
	"tei/internal/application"
)

// PatternRequirements projects explicit Function/Data references from an
// Application Pattern. This is a partial inventory, not a complete change
// qualification: it cannot discover Business responsibility or write authority.
// The caller must add those requirements for operations that need them.
func PatternRequirements(app application.Application, pattern application.Pattern) []Requirement {
	subject := "application-pattern:" + string(pattern.ID)
	result := []Requirement{{Subject: subject, Relation: "pattern-validation", Owner: "validation"}}
	if pattern.Data != "" {
		result = append(result, Requirement{Subject: "semantic-data:" + string(pattern.Data), Relation: "data-reference", Owner: "validation"})
	}
	if pattern.Relation != "" {
		result = append(result, Requirement{Subject: "semantic-relation:" + string(pattern.Relation), Relation: "relation-reference", Owner: "validation"})
	}
	if pattern.Query != "" {
		result = append(result, Requirement{Subject: "query:" + string(pattern.Query), Relation: "query-reference", Owner: "validation"})
	}
	if pattern.Capability != "" {
		result = append(result, Requirement{Subject: "capability:" + string(pattern.Capability), Relation: "capability-reference", Owner: "validation"})
	}
	if pattern.Datasource != "" {
		result = append(result, Requirement{Subject: "application:" + string(app.ID) + "/datasource:" + string(pattern.Datasource), Relation: "datasource-declaration", Owner: "validation"})
	}
	if pattern.ContractQuery != nil {
		result = append(result, Requirement{Subject: "contract:" + string(pattern.ContractQuery.Contract) + "/query:" + string(pattern.ContractQuery.Query), Relation: "contract-query-reference", Owner: "validation"})
	}
	return result
}
