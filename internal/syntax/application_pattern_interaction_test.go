package syntax_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/syntax"
)

func TestParseRecordApplicationPatternInteraction(t *testing.T) {
	src := `pattern order_entry.order_record record {
    data sales.order
    interaction {
        item sales.order.order_id
        item sales.order.customer_id lookup order_entry.customer_reference customer.customer.customer_id
        navigate from order_entry.recent_orders
    }
}`

	got, err := syntax.NewParser(src).ParseApplicationPattern()
	if err != nil {
		t.Fatalf("parse pattern: %v", err)
	}
	if got.Interaction == nil {
		t.Fatal("interactionが読み取られていません")
	}
	if len(got.Interaction.Items) != 2 {
		t.Fatalf("items=%#v", got.Interaction.Items)
	}
	if got.Interaction.Items[0].Attribute != core.CanonicalID("sales.order.order_id") || got.Interaction.Items[0].Lookup != nil {
		t.Fatalf("first item=%#v", got.Interaction.Items[0])
	}
	lookup := got.Interaction.Items[1].Lookup
	if lookup == nil || lookup.Pattern != core.CanonicalID("order_entry.customer_reference") || lookup.TargetAttribute != core.CanonicalID("customer.customer.customer_id") {
		t.Fatalf("lookup=%#v", lookup)
	}
	if len(got.Interaction.Navigations) != 1 || got.Interaction.Navigations[0].From != core.CanonicalID("order_entry.recent_orders") {
		t.Fatalf("navigations=%#v", got.Interaction.Navigations)
	}
}

func TestParseRecordApplicationPatternInteractionRejectsUnknownEntry(t *testing.T) {
	src := `pattern order_entry.order_record record {
    data sales.order
    interaction {
        column sales.order.order_id
    }
}`

	if _, err := syntax.NewParser(src).ParseApplicationPattern(); err == nil {
		t.Fatal("interaction内の未知の項目を拒否していません")
	}
}

func TestParseRecordApplicationPatternInteractionRejectsDuplicateBlock(t *testing.T) {
	src := `pattern order_entry.order_record record {
    data sales.order
    interaction {
        item sales.order.order_id
    }
    interaction {
        item sales.order.customer_id
    }
}`

	if _, err := syntax.NewParser(src).ParseApplicationPattern(); err == nil {
		t.Fatal("interactionの重複を拒否していません")
	}
}

func TestParseRecordApplicationPatternInteractionRejectsEmptyBlock(t *testing.T) {
	src := `pattern order_entry.order_record record {
    data sales.order
    interaction {
    }
}`

	if _, err := syntax.NewParser(src).ParseApplicationPattern(); err == nil {
		t.Fatal("空のinteractionを拒否していません")
	}
}

func TestParseRecordApplicationPatternInteractionRejectsLookupWithoutTarget(t *testing.T) {
	src := `pattern order_entry.order_record record {
    data sales.order
    interaction {
        item sales.order.customer_id lookup order_entry.customer_reference
    }
}`

	if _, err := syntax.NewParser(src).ParseApplicationPattern(); err == nil {
		t.Fatal("lookupのtarget attributeがない宣言を拒否していません")
	}
}
