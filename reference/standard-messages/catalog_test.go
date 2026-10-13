package standardmessages

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
)

func orderMessage() Definition {
	return Definition{
		Ref: Ref{Owner: plugin.Identity{ID: "sales.order_ui", Version: "1.0.0"}, ID: core.CanonicalID("order.confirmed")},
		SourceLocale: "en-US", Text: "Order {orderId} was confirmed", Format: FormatNamedV0,
		Parameters: []Parameter{{Name: "orderId", Kind: ParameterText}},
	}
}

func TestBaseMessagesNeedNoLocalizationPlugin(t *testing.T) {
	d := orderMessage()
	cat := Catalog{Definitions: []Definition{d}}
	got, err := cat.Lookup(d.Ref)
	if err != nil || got.Text != d.Text {
		t.Fatalf("lookup: %+v %v", got, err)
	}
	if err := ValidateArguments(got, map[string]ParameterKind{"orderId": ParameterText}); err != nil {
		t.Fatal(err)
	}
	got.Parameters[0].Kind = ParameterBoolean
	second, err := cat.Lookup(d.Ref)
	if err != nil || second.Parameters[0].Kind != ParameterText {
		t.Fatalf("shared catalog mutated: %+v %v", second, err)
	}
	for _, wrong := range []map[string]ParameterKind{nil, {"orderId": ParameterInteger}, {"orderId": ParameterText, "extra": ParameterText}} {
		if ValidateArguments(d, wrong) == nil {
			t.Fatalf("accepted invalid invocation signature: %+v", wrong)
		}
	}
}

func TestMessageValidationRejectsUnsafeOrAmbiguousReference(t *testing.T) {
	cases := []struct {
		name  string
		apply func(*Definition)
	}{
		{"missing-owner", func(d *Definition) { d.Ref.Owner.ID = "" }},
		{"invalid-owner-version", func(d *Definition) { d.Ref.Owner.Version = "v1" }},
		{"missing-message-id", func(d *Definition) { d.Ref.ID = "" }},
		{"invalid-locale", func(d *Definition) { d.SourceLocale = "JA_jp" }},
		{"undeclared-placeholder", func(d *Definition) { d.Text = "Order {unknown}" }},
		{"missing-placeholder", func(d *Definition) { d.Text = "Order confirmed" }},
		{"nested-expression", func(d *Definition) { d.Text = "Order {{orderId}}" }},
		{"extra-close", func(d *Definition) { d.Text = "Order {orderId}}" }},
		{"unsupported-MF2", func(d *Definition) { d.Format = "mf2" }},
		{"unknown-kind", func(d *Definition) { d.Parameters[0].Kind = "external" }},
		{"duplicate-parameters", func(d *Definition) { d.Parameters = append(d.Parameters, d.Parameters[0]) }},
		{"plain-with-parameters", func(d *Definition) { d.Format = FormatPlain }},
		{"empty-text", func(d *Definition) { d.Text = " " }},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			d := orderMessage()
			tt.apply(&d)
			if err := Validate(d); err == nil {
				t.Fatalf("invalid contract accepted: %+v", d)
			}
		})
	}
	d := orderMessage()
	if err := (Catalog{Definitions: []Definition{d, d}}).Validate(); err == nil || !strings.Contains(err.Error(), "duplicate-message") {
		t.Fatalf("duplicate accepted: %v", err)
	}
	if _, err := (Catalog{Definitions: []Definition{d}}).Lookup(Ref{Owner: d.Ref.Owner, ID: "not_found"}); err == nil {
		t.Fatal("missing reference materialized")
	}
}

func TestRevisionIncludesSourceAndParameterContract(t *testing.T) {
	a := orderMessage()
	b := orderMessage()
	b.Text = "Confirmed order {orderId}"
	if Revision(a) == Revision(b) {
		t.Fatal("source text revision not changed")
	}
	b = orderMessage()
	b.Parameters[0].Kind = ParameterInteger
	if Revision(a) == Revision(b) {
		t.Fatal("parameter contract revision not changed")
	}
	b = orderMessage()
	b.Parameters = append(b.Parameters, Parameter{Name: "user", Kind: ParameterText})
	b.Text += " by {user}"
	c := b
	c.Parameters = append([]Parameter(nil), b.Parameters...)
	c.Parameters[0], c.Parameters[1] = c.Parameters[1], c.Parameters[0]
	if Revision(b) != Revision(c) {
		t.Fatal("parameter ordering affected canonical revision")
	}
}
