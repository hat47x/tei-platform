package validation

import (
	"testing"

	"tei/internal/application"
)

func TestValidatePrintTemplateOnRecord(t *testing.T) {
	w := applicationPatternInteractionWorld()
	interactionRecordPattern(&w).PrintTemplate = "a4-v1"
	for _, diagnostic := range validateApplicationPatterns(w) {
		if diagnostic.Code == "APPLICATION_PATTERN_PRINT_TEMPLATE_NOT_ALLOWED" {
			t.Fatalf("record print-template was rejected: %s", diagnostic.Message)
		}
	}
}

func TestValidatePrintTemplateRejectsCommand(t *testing.T) {
	w := applicationPatternInteractionWorld()
	w.Applications[0].Patterns = append(w.Applications[0].Patterns, application.Pattern{
		ID:            "order_entry.submit_order",
		Kind:          application.PatternCommand,
		Capability:    "sales.submit",
		PrintTemplate: "a4-v1",
	})
	if diagnosticByCode(validateApplicationPatterns(w), "APPLICATION_PATTERN_PRINT_TEMPLATE_NOT_ALLOWED") == nil {
		t.Fatal("command print-template was accepted")
	}
}
