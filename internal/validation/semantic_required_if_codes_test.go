package validation

import (
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// A conditional requirement names two attributes of one Data. Each way it can
// fail to name them is reported with its own code, produced here one change at
// a time from a World the validator accepts (#4032).

const requiredIfRuleID = "sales.order_line.reason_required"

func requiredIfWorld() ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{
			{
				ID: "sales",
				Data: []semantic.Data{
					{
						ID: "sales.order_line",
						Attributes: []semantic.Attribute{
							{ID: "sales.order_line.discount", Symbol: "discount", TypeRef: "std.decimal"},
							{ID: "sales.order_line.reason", Symbol: "reason", TypeRef: "std.text"},
							{ID: "sales.order_line.at", Symbol: "at", TypeRef: "std.datetime"},
							{ID: "sales.order_line.on", Symbol: "on", TypeRef: "std.date"},
							{ID: "sales.order_line.flag", Symbol: "flag", TypeRef: "std.boolean"},
						},
					},
					{
						ID: "sales.order",
						Attributes: []semantic.Attribute{
							{ID: "sales.order.note", Symbol: "note", TypeRef: "std.text"},
						},
						Relations: []semantic.Relation{
							{ID: "sales.order.lines", Symbol: "lines", Target: "sales.order_line", Cardinality: semantic.CardinalityMany},
						},
					},
				},
				Validations: []semantic.Validation{
					{
						ID: requiredIfRuleID, Symbol: "reason_required",
						Kind:      semantic.ValidationRequiredIf,
						Attribute: "sales.order_line.reason", Compared: "sales.order_line.discount",
					},
				},
			},
			{ID: "std", Types: []semantic.Type{
				{ID: "std.text", Symbol: "Text", Kind: semantic.TypePrimitive},
				{ID: "std.decimal", Symbol: "Decimal", Kind: semantic.TypePrimitive},
				{ID: "std.datetime", Symbol: "Datetime", Kind: semantic.TypePrimitive},
				{ID: "std.date", Symbol: "Date", Kind: semantic.TypePrimitive},
				{ID: "std.boolean", Symbol: "Boolean", Kind: semantic.TypePrimitive},
			}},
		},
	}
}

// The control: the shape the cases below break is accepted first, so a case
// cannot pass because the whole World was already rejected.
func TestRequiredIfBaselineIsAccepted(t *testing.T) {
	if diagnostics := Validate(requiredIfWorld()); len(diagnostics) != 0 {
		t.Fatalf("baseline world is rejected: %+v", diagnostics)
	}
}

func TestRequiredIfCodesAreProduced(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*ir.World)
		want   string
	}{
		{"no attribute makes the subject required", func(w *ir.World) {
			w.Semantics[0].Validations[0].Compared = ""
		}, "SEMANTIC_VALIDATION_COMPARED_ATTRIBUTE_REQUIRED"},
		{"the trigger attribute is not declared", func(w *ir.World) {
			w.Semantics[0].Validations[0].Compared = "sales.order_line.missing"
		}, "UNKNOWN_SEMANTIC_VALIDATION_COMPARED_ATTRIBUTE"},
		{"the attribute is made required by itself", func(w *ir.World) {
			w.Semantics[0].Validations[0].Compared = "sales.order_line.reason"
		}, "SEMANTIC_VALIDATION_REQUIRED_IF_SAME_ATTRIBUTE"},
		{"the trigger belongs to another data", func(w *ir.World) {
			w.Semantics[0].Validations[0].Compared = "sales.order.note"
		}, "SEMANTIC_VALIDATION_REQUIRED_IF_REQUIRES_SAME_DATA"},
		{"a relation on a rule about one record", func(w *ir.World) {
			w.Semantics[0].Validations[0].Relation = "sales.order.lines"
		}, "SEMANTIC_VALIDATION_RELATION_NOT_ALLOWED"},
		{"if present on a rule about presence", func(w *ir.World) {
			w.Semantics[0].Validations[0].IfPresent = true
		}, "SEMANTIC_VALIDATION_COMPARED_ATTRIBUTE_NOT_ALLOWED"},
		{"the attribute is not declared", func(w *ir.World) {
			w.Semantics[0].Validations[0].Attribute = "sales.order_line.nothing"
		}, "UNKNOWN_SEMANTIC_VALIDATION_ATTRIBUTE"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			world := requiredIfWorld()
			tc.mutate(&world)
			diagnostics := Validate(world)
			if len(diagnostics) != 1 {
				t.Fatalf("diagnostics=%+v, want exactly one", diagnostics)
			}
			got := diagnostics[0]
			if got.Code != tc.want || got.Severity != Error || got.Subject == nil ||
				got.Subject.Kind != "semantic-validation" || got.Subject.ID != requiredIfRuleID {
				t.Fatalf("diagnostic=%+v, want %s on %s", got, tc.want, requiredIfRuleID)
			}
		})
	}
}

// A rule about presence reads whether a value is there, whatever its type. So
// the validator must not require a number, a date, or any other particular
// type: refusing one would state a constraint the rule does not have. This is
// the difference from "is positive" and "is not after", which do constrain the
// type.
func TestRequiredIfAcceptsAnyAttributeType(t *testing.T) {
	for _, trigger := range []string{
		"sales.order_line.discount", // decimal
		"sales.order_line.at",       // a moment in time
		"sales.order_line.on",       // a calendar date
		"sales.order_line.flag",     // boolean
		"sales.order_line.reason",   // text, also the subject
	} {
		t.Run(trigger, func(t *testing.T) {
			world := requiredIfWorld()
			world.Semantics[0].Validations[0].Compared = core.CanonicalID(trigger)
			if trigger == "sales.order_line.reason" {
				// The same attribute is refused for another reason, which the
				// sibling case covers; skip it here.
				t.Skip("covered by the same-attribute case")
			}
			if diagnostics := Validate(world); len(diagnostics) != 0 {
				t.Fatalf("%s was refused: %+v", trigger, diagnostics)
			}
		})
	}
}
