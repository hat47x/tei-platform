package syntax_test

import (
	"testing"

	"tei/internal/syntax"
)

func TestSemanticPeriodParses(t *testing.T) {
	src := `namespace sales

data Order {
    attribute order_date type std.date
    attribute status type std.code
}

capability sales.period.close {
    input order sales.order
    outcome closed sales.order
}

period sales.month {
    data sales.order
    business-date sales.order.order_date
    states open closing closed
    close-requires sales.period.close
    closed-rejects create update
    correction via sales.period.correct
}
`
	namespace, err := syntax.NewParser(src).ParseSemanticFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(namespace.Periods) != 1 {
		t.Fatalf("expected one period: %+v", namespace.Periods)
	}
	period := namespace.Periods[0]
	if period.ID != "sales.month" || period.Data != "sales.order" || period.BusinessDate != "sales.order.order_date" {
		t.Fatalf("period identity lost: %+v", period)
	}
	if len(period.States) != 3 || period.States[0] != "open" || period.States[2] != "closed" {
		t.Fatalf("states lost: %+v", period.States)
	}
	if period.CloseRequires != "sales.period.close" || period.Correction != "sales.period.correct" {
		t.Fatalf("capability references lost: %+v", period)
	}
	if len(period.ClosedRejects) != 2 || period.ClosedRejects[0] != "create" {
		t.Fatalf("closed rejects lost: %+v", period.ClosedRejects)
	}
}
