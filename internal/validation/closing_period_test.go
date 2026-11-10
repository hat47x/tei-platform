package validation

import (
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/syntax"
)

func closingPeriodWorld(t *testing.T, source string) ir.World {
	t.Helper()
	namespace, err := syntax.NewParser(source).ParseSemanticFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return ir.World{Semantics: []semantic.Namespace{namespace}}
}

func closingCodes(w ir.World) []string {
	var out []string
	for _, diagnostic := range Validate(w) {
		switch diagnostic.Code {
		case "PERIOD_CLOSING_DAY_RANGE", "PERIOD_CLOSING_REQUIRES_CALENDAR":
			out = append(out, diagnostic.Code)
		}
	}
	return out
}

// TestClosingDayParsesAndValidates checks that `closes on day N` is read and that a
// period with a closing day and a calendar produces no diagnostic.
func TestClosingDayParsesAndValidates(t *testing.T) {
	world := closingPeriodWorld(t, `namespace sales

data order {
    attribute order_date type std.date
}

calendar sales.fiscal {
    fiscal-year starts month 4
    week starts monday
}

period sales.month {
    data sales.order
    business-date sales.order.order_date
    calendar sales.fiscal
    closes on day 20
    states open closing closed
}
`)
	for _, namespace := range world.Semantics {
		if len(namespace.Periods) != 1 || namespace.Periods[0].ClosingDay != 20 {
			t.Fatalf("periods=%+v", namespace.Periods)
		}
	}
	if codes := closingCodes(world); len(codes) != 0 {
		t.Fatalf("codes=%v", codes)
	}
}

// TestClosingDayRequiresACalendar checks that a closing day without a calendar is
// reported: a closing day splits a month, so the month has to be declared.
func TestClosingDayRequiresACalendar(t *testing.T) {
	world := closingPeriodWorld(t, `namespace sales

data order {
    attribute order_date type std.date
}

period sales.month {
    data sales.order
    business-date sales.order.order_date
    closes on day 20
}
`)
	codes := closingCodes(world)
	if len(codes) != 1 || codes[0] != "PERIOD_CLOSING_REQUIRES_CALENDAR" {
		t.Fatalf("codes=%v", codes)
	}
}

// TestClosingDayRefusesADayThatDoesNotExistEveryMonth checks that days 29–31 are
// refused: February has no such day, so the boundary would not exist in every
// month.
func TestClosingDayRefusesADayThatDoesNotExistEveryMonth(t *testing.T) {
	for _, day := range []string{"29", "30", "31"} {
		world := closingPeriodWorld(t, `namespace sales

data order {
    attribute order_date type std.date
}

calendar sales.fiscal {
    fiscal-year starts month 4
    week starts monday
}

period sales.month {
    data sales.order
    business-date sales.order.order_date
    calendar sales.fiscal
    closes on day `+day+`
}
`)
		codes := closingCodes(world)
		if len(codes) != 1 || codes[0] != "PERIOD_CLOSING_DAY_RANGE" {
			t.Fatalf("day %s codes=%v", day, codes)
		}
	}
}
