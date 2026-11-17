package syntax

import (
	"strings"
	"testing"

	"tei/internal/database/persistence"
)

const decimalFence = `persist sales_core.order_line {
    database sales_core
    semantic sales.order_line
    table public.order_lines
    column sales.order_line.quantity -> quantity storage %s required
    primary-key sales.order_line.order_line_id
}`

// A declared width is captured on the column, and a bare decimal keeps
// carrying no width at all. The two cases are asserted together so a change
// that makes "no width" look like "width 0" fails here.
func TestParsePersistenceCapturesOptionalDecimalWidth(t *testing.T) {
	withWidth, err := NewParser(fmtFence("decimal(12,3)")).ParsePersistence()
	if err != nil {
		t.Fatalf("decimal(12,3) did not parse: %v", err)
	}
	if got := withWidth.Columns[0].Precision; got == nil || *got != 12 {
		t.Fatalf("precision=%v, want 12", got)
	}
	if got := withWidth.Columns[0].Scale; got == nil || *got != 3 {
		t.Fatalf("scale=%v, want 3", got)
	}

	bare, err := NewParser(fmtFence("decimal")).ParsePersistence()
	if err != nil {
		t.Fatalf("bare decimal did not parse: %v", err)
	}
	if bare.Columns[0].Precision != nil || bare.Columns[0].Scale != nil {
		t.Fatalf("bare decimal gained a width: precision=%v scale=%v",
			bare.Columns[0].Precision, bare.Columns[0].Scale)
	}
	if _, _, ok := bare.Columns[0].DecimalWidth(); ok {
		t.Fatal("bare decimal reported a width")
	}
}

// A width on a non-decimal type, a scale above the precision, and a malformed
// suffix are all parse errors. Silently ignoring them would let a Source claim
// a width the database never receives.
func TestParsePersistenceRejectsInvalidDecimalWidth(t *testing.T) {
	for _, tc := range []struct {
		name   string
		suffix string
		want   string
	}{
		{"width on varchar", "varchar(12)", "cannot carry decimal precision"},
		{"scale above precision", "decimal(3,12)", "decimal scale 12 is outside 0..3"},
		{"zero precision", "decimal(0,0)", "decimal precision 0 is outside 1..38"},
		{"precision above bound", "decimal(999,1)", "decimal precision 999 is outside 1..38"},
		{"missing scale", "decimal(12)", "decimal width must be"},
		{"missing close", "decimal(12,3", "decimal width must be"},
		{"non-digit", "decimal(12,x)", "decimal width parts must be digits"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewParser(fmtFence(tc.suffix)).ParsePersistence()
			if err == nil {
				t.Fatalf("%q was accepted", tc.suffix)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("%q error=%v, want it to mention %q", tc.suffix, err, tc.want)
			}
		})
	}
}

func fmtFence(suffix string) string {
	return strings.Replace(decimalFence, "%s", suffix, 1)
}

// The JSON of an existing Source must not change: no width means no keys.
func TestColumnMappingJSONWithoutWidthOmitsWidthKeys(t *testing.T) {
	m, err := NewParser(fmtFence("decimal")).ParsePersistence()
	if err != nil {
		t.Fatal(err)
	}
	if got := m.Columns[0].Precision; got != nil {
		t.Fatalf("omitted width serialized as %v", got)
	}
	if m.Columns[0].Storage != persistence.StorageDecimal {
		t.Fatalf("storage=%q", m.Columns[0].Storage)
	}
}
