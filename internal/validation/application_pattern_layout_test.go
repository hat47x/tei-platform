package validation

import (
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
)

func layoutWorld() ir.World {
	w := applicationPatternInteractionWorld()
	pattern := interactionRecordPattern(&w)
	pattern.Layout = &application.Layout{Order: []core.CanonicalID{"sales.order.customer_id", "sales.order.order_id"}}
	return w
}

func TestValidateRecordLayout(t *testing.T) {
	for _, diagnostic := range validateApplicationPatterns(layoutWorld()) {
		t.Fatalf("unexpected diagnostic: %s %s", diagnostic.Code, diagnostic.Message)
	}
}

func TestValidateRecordLayoutRejectsUnknownAttribute(t *testing.T) {
	w := layoutWorld()
	interactionRecordPattern(&w).Layout.Order = []core.CanonicalID{"sales.order.missing"}
	if diagnosticByCode(validateApplicationPatterns(w), "UNKNOWN_APPLICATION_PATTERN_LAYOUT_ORDER") == nil {
		t.Fatal("unknown layout attribute was accepted")
	}
}

func TestValidateRecordLayoutRejectsDuplicate(t *testing.T) {
	w := layoutWorld()
	interactionRecordPattern(&w).Layout.Order = []core.CanonicalID{"sales.order.order_id", "sales.order.order_id"}
	if diagnosticByCode(validateApplicationPatterns(w), "APPLICATION_PATTERN_LAYOUT_ORDER_DUPLICATE") == nil {
		t.Fatal("duplicate layout order was accepted")
	}
}

func TestValidateRecordLayoutRejectsNonRecord(t *testing.T) {
	w := layoutWorld()
	interactionRecordPattern(&w).Kind = application.PatternBrowse
	if diagnosticByCode(validateApplicationPatterns(w), "APPLICATION_PATTERN_LAYOUT_NOT_ALLOWED") == nil {
		t.Fatal("layout on browse was accepted")
	}
}
