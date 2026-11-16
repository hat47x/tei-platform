package validation

import (
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
)

// A rule that compares two dates of a record is data in the World. Each check
// it takes part in reports its own code, produced here one change at a time
// from a World the validator accepts.
func TestSemanticNotAfterCodesAreProduced(t *testing.T) {
	const rule = "organization.membership.period_in_order"
	valid := func() ir.World {
		return ir.World{
			Semantics: []semantic.Namespace{
				{
					ID:    "organization",
					Types: []semantic.Type{{ID: "organization.day", Symbol: "Day", Kind: semantic.TypeAlias, Underlying: "std.date"}},
					Data: []semantic.Data{
						{
							ID: "organization.membership",
							Attributes: []semantic.Attribute{
								{ID: "organization.membership.starts_on", Symbol: "startsOn", TypeRef: "std.date"},
								// An alias of a calendar date is a calendar date.
								{ID: "organization.membership.ends_on", Symbol: "endsOn", TypeRef: "organization.day"},
								{ID: "organization.membership.registered_at", Symbol: "registeredAt", TypeRef: "std.datetime"},
								{ID: "organization.membership.allocation", Symbol: "allocation", TypeRef: "std.decimal"},
							},
						},
						{
							ID: "organization.department",
							Attributes: []semantic.Attribute{
								{ID: "organization.department.closed_on", Symbol: "closedOn", TypeRef: "std.date"},
							},
							Relations: []semantic.Relation{
								{ID: "organization.department.members", Symbol: "members", Target: "organization.membership", Cardinality: semantic.CardinalityMany},
							},
						},
					},
					Validations: []semantic.Validation{
						{ID: rule, Symbol: "period_in_order", Kind: semantic.ValidationNotAfter,
							Attribute: "organization.membership.starts_on", Compared: "organization.membership.ends_on", IfPresent: true},
						{ID: "organization.membership.allocation_positive", Symbol: "allocation_positive",
							Kind: semantic.ValidationPositive, Attribute: "organization.membership.allocation"},
					},
				},
				{ID: "std", Types: []semantic.Type{
					{ID: "std.date", Symbol: "Date", Kind: semantic.TypePrimitive},
					{ID: "std.datetime", Symbol: "Datetime", Kind: semantic.TypePrimitive},
					{ID: "std.decimal", Symbol: "Decimal", Kind: semantic.TypePrimitive},
				}},
			},
		}
	}
	if diagnostics := Validate(valid()); len(diagnostics) != 0 {
		t.Fatalf("baseline world is rejected: %+v", diagnostics)
	}

	cases := []struct {
		name    string
		mutate  func(w *ir.World)
		want    string
		subject string
	}{
		{"nothing to compare with", func(w *ir.World) {
			w.Semantics[0].Validations[0].Compared = ""
		}, "SEMANTIC_VALIDATION_COMPARED_ATTRIBUTE_REQUIRED", rule},
		{"compared with an attribute nobody declared", func(w *ir.World) {
			w.Semantics[0].Validations[0].Compared = "organization.membership.ends_at"
		}, "UNKNOWN_SEMANTIC_VALIDATION_COMPARED_ATTRIBUTE", rule},
		{"compared with itself", func(w *ir.World) {
			w.Semantics[0].Validations[0].Compared = "organization.membership.starts_on"
		}, "SEMANTIC_VALIDATION_NOT_AFTER_SAME_ATTRIBUTE", rule},
		{"compared with a date of another data", func(w *ir.World) {
			w.Semantics[0].Validations[0].Compared = "organization.department.closed_on"
		}, "SEMANTIC_VALIDATION_NOT_AFTER_REQUIRES_SAME_DATA", rule},
		{"compared with a moment in time", func(w *ir.World) {
			w.Semantics[0].Validations[0].Compared = "organization.membership.registered_at"
		}, "SEMANTIC_VALIDATION_NOT_AFTER_REQUIRES_DATE_ATTRIBUTES", rule},
		{"a number as the first date", func(w *ir.World) {
			w.Semantics[0].Validations[0].Attribute = "organization.membership.allocation"
		}, "SEMANTIC_VALIDATION_NOT_AFTER_REQUIRES_DATE_ATTRIBUTES", rule},
		{"a relation", func(w *ir.World) {
			w.Semantics[0].Validations[0].Relation = "organization.department.members"
		}, "SEMANTIC_VALIDATION_RELATION_NOT_ALLOWED", rule},
		{"another kind with an attribute to compare with", func(w *ir.World) {
			w.Semantics[0].Validations[1].Compared = "organization.membership.starts_on"
		}, "SEMANTIC_VALIDATION_COMPARED_ATTRIBUTE_NOT_ALLOWED", "organization.membership.allocation_positive"},
		{"another kind with if present", func(w *ir.World) {
			w.Semantics[0].Validations[1].IfPresent = true
		}, "SEMANTIC_VALIDATION_COMPARED_ATTRIBUTE_NOT_ALLOWED", "organization.membership.allocation_positive"},
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
