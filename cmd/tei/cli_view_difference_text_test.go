package main

import (
	"bytes"
	"strings"
	"testing"

	"tei/internal/applicationview"
	"tei/internal/command"
	"tei/internal/semantic"
)

// TestWriteReferenceRunTextShowsDifferences checks that a reader of the text
// output can see an unresolved difference without opening the operation log.
func TestWriteReferenceRunTextShowsDifferences(t *testing.T) {
	run := referenceCommandRun{
		Mode:      "reference",
		Plan:      applicationview.CommandPlan{Capability: "sales.delete_order", Label: "delete_order"},
		Effect:    &semantic.DataEffect{Kind: semantic.DataEffectDelete, Data: "sales.order"},
		Committed: true,
		Differences: []command.Difference{{
			Code:     command.DifferenceAdditionalScope,
			Declared: semantic.DataEffect{Kind: semantic.DataEffectDelete, Data: "sales.order"},
			Applied:  command.AppliedOperation{Kind: semantic.DataEffectDelete, Data: "sales.order_line"},
		}},
	}
	var out bytes.Buffer
	if code, err := writeReferenceRunText(&out, run); err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if !strings.Contains(out.String(), "difference: additional-scope (delete sales.order_line beyond delete sales.order)") {
		t.Fatalf("output=%s", out.String())
	}
}

// TestWriteReferenceRunTextOmitsDifferencesWhenNone checks that a run without
// a difference does not print a difference line.
func TestWriteReferenceRunTextOmitsDifferencesWhenNone(t *testing.T) {
	var out bytes.Buffer
	run := referenceCommandRun{
		Mode:      "reference",
		Plan:      applicationview.CommandPlan{Capability: "sales.delete_order", Label: "delete_order"},
		Committed: true,
	}
	if code, err := writeReferenceRunText(&out, run); err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if strings.Contains(out.String(), "difference:") {
		t.Fatalf("output=%s", out.String())
	}
}
