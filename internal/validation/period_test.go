package validation_test

import (
	"strings"
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/syntax"
	"tei/internal/validation"
)

func parseSemantic(t *testing.T, src string) semantic.Namespace {
	t.Helper()
	namespace, err := syntax.NewParser(src).ParseSemanticFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return namespace
}

func periodCodes(t *testing.T, src string) []string {
	t.Helper()
	diagnostics := validation.Validate(ir.World{Semantics: []semantic.Namespace{parseSemantic(t, src)}})
	var codes []string
	for _, diagnostic := range diagnostics {
		if strings.HasPrefix(diagnostic.Code, "PERIOD_") {
			codes = append(codes, diagnostic.Code)
		}
	}
	return codes
}

const periodBase = `namespace sales

data Order {
    attribute order_date type std.date
}

capability sales.period.close {
    input order sales.order
    outcome closed sales.order
}

capability sales.period.correct {
    input order sales.order
    outcome corrected sales.order
}

`

func TestValidPeriodHasNoPeriodDiagnostic(t *testing.T) {
	src := periodBase + `period sales.month {
    data sales.order
    business-date sales.order.order_date
    states open closing closed
    close-requires sales.period.close
    closed-rejects create update
    correction via sales.period.correct
}
`
	if codes := periodCodes(t, src); len(codes) != 0 {
		t.Fatalf("a valid period must not report diagnostics: %v", codes)
	}
}

func TestPeriodReportsEachMissingPart(t *testing.T) {
	src := periodBase + `period sales.month {
    data sales.unknown
    states open
    close-requires sales.period.missing
    closed-rejects create archive
}
`
	codes := periodCodes(t, src)
	want := []string{"PERIOD_UNKNOWN_DATA", "PERIOD_STATES_INCOMPLETE", "PERIOD_UNKNOWN_CAPABILITY", "PERIOD_UNKNOWN_CLOSED_REJECT"}
	for _, expected := range want {
		found := false
		for _, code := range codes {
			if code == expected {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected %s in %v", expected, codes)
		}
	}
}
