package index

import (
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func TestBuildIndexesValidationsAndApplicationPatterns(t *testing.T) {
	w := ir.World{
		Semantics: []semantic.Namespace{
			{
				ID: "sales",
				Validations: []semantic.Validation{
					{
						ID:        "sales.order.total_positive",
						Kind:      semantic.ValidationPositive,
						Attribute: "sales.order.total",
					},
					{
						ID:        "sales.order.customer_unique",
						Kind:      semantic.ValidationRelationUnique,
						Attribute: "customer.customer.customer_id",
						Relation:  "sales.order.customer",
					},
				},
			},
		},
		Applications: []application.Application{
			{
				ID:          "order_entry",
				Validations: []core.CanonicalID{"sales.order.total_positive"},
				Patterns: []application.Pattern{
					{
						ID:         "order_entry.customer_reference",
						Kind:       application.PatternReference,
						Relation:   "sales.order.customer",
						Datasource: "master",
						ContractQuery: &application.ContractQueryInvocation{
							Contract: "master_data.customer",
							Query:    "find_by_id",
							Inputs: []application.ContractQueryInputBinding{
								{
									TargetAttribute: "customer.customer.customer_id",
									Input:           "id",
								},
							},
						},
					},
					{
						ID:    "order_entry.recent_orders",
						Kind:  application.PatternBrowse,
						Data:  "sales.order",
						Query: "order_entry.recent_orders",
					},
					{
						ID:         "order_entry.submit",
						Kind:       application.PatternCommand,
						Data:       "sales.order",
						Capability: "sales.submit",
					},
				},
			},
		},
	}

	got := Build(w)

	for _, id := range []string{
		"sales.order.total_positive",
		"sales.order.customer_unique",
	} {
		if !hasSemanticElement(got, "validation", id) {
			t.Fatalf("semantic validation %s is missing from index: %+v", id, got.SemanticElements)
		}
	}

	for _, want := range []SemanticDependency{
		{
			FromKind: "semantic-validation",
			From:     "sales.order.total_positive",
			Relation: "attribute",
			ToKind:   "semantic-attribute",
			To:       "sales.order.total",
		},
		{
			FromKind: "semantic-validation",
			From:     "sales.order.customer_unique",
			Relation: "relation",
			ToKind:   "semantic-relation",
			To:       "sales.order.customer",
		},
	} {
		if !hasSemanticDependency(got, want) {
			t.Fatalf("semantic dependency is missing: %+v", want)
		}
	}

	if len(got.ApplicationPatterns) != 3 {
		t.Fatalf("application pattern index count=%d want=3: %+v", len(got.ApplicationPatterns), got.ApplicationPatterns)
	}
	if got.ApplicationPatterns[0].Pattern != "order_entry.customer_reference" ||
		got.ApplicationPatterns[1].Pattern != "order_entry.recent_orders" ||
		got.ApplicationPatterns[2].Pattern != "order_entry.submit" {
		t.Fatalf("application pattern index is not deterministic: %+v", got.ApplicationPatterns)
	}

	customerReference := got.ApplicationPatterns[0]
	if customerReference.Application != "order_entry" ||
		customerReference.Kind != string(application.PatternReference) ||
		customerReference.Data != "" ||
		customerReference.Relation != "sales.order.customer" ||
		customerReference.Datasource != "master" ||
		customerReference.Contract != "master_data.customer" ||
		customerReference.ContractQuery != "master_data.customer/find_by_id" ||
		len(customerReference.ContractQueryInputs) != 1 ||
		customerReference.ContractQueryInputs[0].TargetAttribute != "customer.customer.customer_id" ||
		customerReference.ContractQueryInputs[0].Input != "id" {
		t.Fatalf("customer reference pattern lost meaning-bearing fields: %+v", customerReference)
	}

	for _, want := range []DeclaredDependency{
		{
			FromKind: "application",
			From:     "order_entry",
			Relation: "uses-validation",
			ToKind:   "semantic-validation",
			To:       "sales.order.total_positive",
		},
		{
			FromKind: "application",
			From:     "order_entry",
			Relation: "owns-pattern",
			ToKind:   "application-pattern",
			To:       "order_entry.customer_reference",
		},
		{
			FromKind: "application-pattern",
			From:     "order_entry.customer_reference",
			Relation: "relation",
			ToKind:   "semantic-relation",
			To:       "sales.order.customer",
		},
		{
			FromKind: "application-pattern",
			From:     "order_entry.customer_reference",
			Relation: "contract",
			ToKind:   "database-contract",
			To:       "master_data.customer",
		},
		{
			FromKind: "application-pattern",
			From:     "order_entry.customer_reference",
			Relation: "contract-query",
			ToKind:   "database-contract-query",
			To:       "master_data.customer/find_by_id",
		},
		{
			FromKind: "application-pattern",
			From:     "order_entry.customer_reference",
			Relation: "contract-query-input",
			ToKind:   "semantic-attribute",
			To:       "customer.customer.customer_id",
		},
		{
			FromKind: "application-pattern",
			From:     "order_entry.recent_orders",
			Relation: "data",
			ToKind:   "semantic-data",
			To:       "sales.order",
		},
		{
			FromKind: "application-pattern",
			From:     "order_entry.recent_orders",
			Relation: "query",
			ToKind:   "query",
			To:       "order_entry.recent_orders",
		},
		{
			FromKind: "application-pattern",
			From:     "order_entry.submit",
			Relation: "capability",
			ToKind:   "semantic-capability",
			To:       "sales.submit",
		},
	} {
		if !hasDeclaredDependency(got, want) {
			t.Fatalf("declared dependency is missing: %+v", want)
		}
	}
}

func hasSemanticElement(d Derived, kind, id string) bool {
	for _, e := range d.SemanticElements {
		if e.Kind == kind && string(e.ID) == id {
			return true
		}
	}
	return false
}

func hasSemanticDependency(d Derived, want SemanticDependency) bool {
	for _, dep := range d.SemanticDependencies {
		if dep == want {
			return true
		}
	}
	return false
}

func hasDeclaredDependency(d Derived, want DeclaredDependency) bool {
	for _, dep := range d.DeclaredDependencies {
		if dep == want {
			return true
		}
	}
	return false
}
