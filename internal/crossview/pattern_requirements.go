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
	if pattern.Interaction != nil {
		seen := map[string]struct{}{}
		add := func(subject, relation string) {
			key := relation + "\x00" + subject
			if _, exists := seen[key]; exists {
				return
			}
			seen[key] = struct{}{}
			result = append(result, Requirement{Subject: subject, Relation: relation, Owner: "validation"})
		}
		for _, item := range pattern.Interaction.Items {
			add("semantic-attribute:"+string(item.Attribute), "interaction-item-reference")
			if item.Lookup != nil {
				add("application-pattern:"+string(item.Lookup.Pattern), "interaction-lookup-reference")
				add("semantic-attribute:"+string(item.Lookup.TargetAttribute), "interaction-lookup-target-reference")
			}
		}
		for _, navigation := range pattern.Interaction.Navigations {
			add("application-pattern:"+string(navigation.From), "interaction-navigation-reference")
		}
	}
	return result
}
