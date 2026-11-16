package validation

import (
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
)

// A rule about the periods of the rows of a relation is data in the World.
// Each check it takes part in reports its own code, produced here one change
// at a time from a World the validator accepts.
func TestSemanticNoOverlapCodesAreProduced(t *testing.T) {
	const rule = "organization.department.members.periods_apart"
	valid := func() ir.World {
		return ir.World{
			Semantics: []semantic.Namespace{
				{
					ID: "organization",
					Data: []semantic.Data{
						{
							ID: "organization.membership",
							Attributes: []semantic.Attribute{
								{ID: "organization.membership.person_id", Symbol: "personId", TypeRef: "std.identifier"},
								{ID: "organization.membership.starts_on", Symbol: "startsOn", TypeRef: "std.date"},
								{ID: "organization.membership.ends_on", Symbol: "endsOn", TypeRef: "std.date"},
								{ID: "organization.membership.registered_at", Symbol: "registeredAt", TypeRef: "std.datetime"},
							},
						},
						{
							ID: "organization.department",
							Attributes: []semantic.Attribute{
								{ID: "organization.department.opened_on", Symbol: "openedOn", TypeRef: "std.date"},
								{ID: "organization.department.closed_on", Symbol: "closedOn", TypeRef: "std.date"},
							},
							Relations: []semantic.Relation{
								{ID: "organization.department.members", Symbol: "members", Target: "organization.membership", Cardinality: semantic.CardinalityMany},
								{ID: "organization.department.head", Symbol: "head", Target: "organization.membership", Cardinality: semantic.CardinalityOne},
							},
						},
					},
					Validations: []semantic.Validation{
						{ID: rule, Symbol: "periods_apart", Kind: semantic.ValidationNoOverlap,
							Attribute: "organization.membership.starts_on", Compared: "organization.membership.ends_on",
							Relation: "organization.department.members", GroupBy: "organization.membership.person_id", OpenEnded: true},
						{ID: "organization.membership.period_in_order", Symbol: "period_in_order", Kind: semantic.ValidationNotAfter,
							Attribute: "organization.membership.starts_on", Compared: "organization.membership.ends_on", IfPresent: true},
					},
				},
				{ID: "std", Types: []semantic.Type{
					{ID: "std.date", Symbol: "Date", Kind: semantic.TypePrimitive},
					{ID: "std.datetime", Symbol: "Datetime", Kind: semantic.TypePrimitive},
					{ID: "std.identifier", Symbol: "Identifier", Kind: semantic.TypePrimitive},
				}},
			},
		}
	}
	if diagnostics := Validate(valid()); len(diagnostics) != 0 {
		t.Fatalf("baseline world is rejected: %+v", diagnostics)
	}
	// Without a grouping and without an open end the rule is still accepted.
	plain := valid()
	plain.Semantics[0].Validations[0].GroupBy, plain.Semantics[0].Validations[0].OpenEnded = "", false
	if diagnostics := Validate(plain); len(diagnostics) != 0 {
		t.Fatalf("a rule without grouping is rejected: %+v", diagnostics)
	}

	const other = "organization.membership.period_in_order"
	cases := []struct {
		name    string
		mutate  func(w *ir.World)
		want    string
		subject string
	}{
		{"no relation", func(w *ir.World) { w.Semantics[0].Validations[0].Relation = "" }, "SEMANTIC_VALIDATION_RELATION_REQUIRED", rule},
		{"a relation nobody declared", func(w *ir.World) { w.Semantics[0].Validations[0].Relation = "organization.department.staff" }, "UNKNOWN_SEMANTIC_VALIDATION_RELATION", rule},
		{"a relation to one row", func(w *ir.World) { w.Semantics[0].Validations[0].Relation = "organization.department.head" }, "SEMANTIC_VALIDATION_RELATION_NOT_MANY", rule},
		{"no end of the period", func(w *ir.World) { w.Semantics[0].Validations[0].Compared = "" }, "SEMANTIC_VALIDATION_COMPARED_ATTRIBUTE_REQUIRED", rule},
		{"an end nobody declared", func(w *ir.World) { w.Semantics[0].Validations[0].Compared = "organization.membership.ends_at" }, "UNKNOWN_SEMANTIC_VALIDATION_COMPARED_ATTRIBUTE", rule},
		{"a grouping nobody declared", func(w *ir.World) { w.Semantics[0].Validations[0].GroupBy = "organization.membership.person" }, "UNKNOWN_SEMANTIC_VALIDATION_GROUP_ATTRIBUTE", rule},
		{"the same attribute starts and ends the period", func(w *ir.World) {
			w.Semantics[0].Validations[0].Compared = "organization.membership.starts_on"
		}, "SEMANTIC_VALIDATION_NO_OVERLAP_SAME_ATTRIBUTE", rule},
		{"an end outside the rows of the relation", func(w *ir.World) {
			w.Semantics[0].Validations[0].Compared = "organization.department.closed_on"
		}, "SEMANTIC_VALIDATION_ATTRIBUTE_OUTSIDE_RELATION_TARGET", rule},
		{"a grouping outside the rows of the relation", func(w *ir.World) {
			w.Semantics[0].Validations[0].GroupBy = "organization.department.opened_on"
		}, "SEMANTIC_VALIDATION_ATTRIBUTE_OUTSIDE_RELATION_TARGET", rule},
		{"an end that is a moment in time", func(w *ir.World) {
			w.Semantics[0].Validations[0].Compared = "organization.membership.registered_at"
		}, "SEMANTIC_VALIDATION_NO_OVERLAP_REQUIRES_DATE_ATTRIBUTES", rule},
		{"a start that is not a date", func(w *ir.World) {
			w.Semantics[0].Validations[0].Attribute = "organization.membership.person_id"
		}, "SEMANTIC_VALIDATION_NO_OVERLAP_REQUIRES_DATE_ATTRIBUTES", rule},
		{"if present on a period rule", func(w *ir.World) { w.Semantics[0].Validations[0].IfPresent = true }, "SEMANTIC_VALIDATION_COMPARED_ATTRIBUTE_NOT_ALLOWED", rule},
		{"another kind with a grouping", func(w *ir.World) {
			w.Semantics[0].Validations[1].GroupBy = "organization.membership.person_id"
		}, "SEMANTIC_VALIDATION_PERIOD_GROUPING_NOT_ALLOWED", other},
		{"another kind with an open end", func(w *ir.World) { w.Semantics[0].Validations[1].OpenEnded = true }, "SEMANTIC_VALIDATION_PERIOD_GROUPING_NOT_ALLOWED", other},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			world := valid()
			tc.mutate(&world)
			diagnostics := Validate(world)
			if len(diagnostics) != 1 {
				t.Fatalf("diagnostics=%+v, want exactly one", diagnostics)
			}
			got := diagnostics[0]
			if got.Code != tc.want || got.Severity != Error || got.Subject == nil ||
				got.Subject.Kind != "semantic-validation" || got.Subject.ID != tc.subject {
				t.Fatalf("diagnostic=%+v, want %s on %s", got, tc.want, tc.subject)
			}
		})
	}
}
