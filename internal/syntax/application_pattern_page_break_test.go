package syntax_test

import (
	"testing"

	"tei/internal/syntax"
)

func TestParseHeaderDetailApplicationPatternPageBreak(t *testing.T) {
	src := `pattern order_entry.order_lines header-detail {
    relation sales.order.lines
    page-break 3 repeat-header
}`
	got, err := syntax.NewParser(src).ParseApplicationPattern()
	if err != nil {
		t.Fatal(err)
	}
	if got.PageBreak == nil || got.PageBreak.Rows != 3 || !got.PageBreak.RepeatHeader {
		t.Fatalf("page-break=%#v", got.PageBreak)
	}
}
