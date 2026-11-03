package syntax_test

import (
	"testing"

	"tei/internal/syntax"
)

func TestParseEffectRule(t *testing.T) {
	source := `namespace sales

data Order {
    attribute order_date type std.date
}

data Stock {
    attribute quantity type std.integer
}

capability sales.order.submit {
    input order sales.order
    outcome submitted sales.order
    effect create sales.order
}

capability sales.stock.decrement {
    input stock sales.stock
    outcome decremented sales.stock
    effect update sales.stock
}

effect-rule order.stock {
    on capability sales.order.submit
    stage after-commit
    effect update sales.stock
    via capability sales.stock.decrement
    on-failure compensate
}
`
	namespace, err := syntax.NewParser(source).ParseSemanticFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(namespace.EffectRules) != 1 {
		t.Fatalf("expected one effect rule: %+v", namespace.EffectRules)
	}
	rule := namespace.EffectRules[0]
	if rule.ID != "sales.order.stock" {
		t.Fatalf("unexpected id: %s", rule.ID)
	}
	if rule.OnCapability != "sales.order.submit" || rule.Stage != "after-commit" || rule.ViaCapability != "sales.stock.decrement" {
		t.Fatalf("unexpected rule: %+v", rule)
	}
	if rule.EffectKind != "update" || rule.EffectData != "sales.stock" || rule.OnFailure != "compensate" {
		t.Fatalf("unexpected effect: %+v", rule)
	}
}
