package xeadsource

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseWithLimitsAcceptsEachBudgetAtItsExactBoundary(t *testing.T) {
	source := []byte(`<System><Table ID="T1"><TableField ID="F1"/></Table></System>`)
	got, err := ParseWithLimits(FormatXead, source, ParseLimits{
		MaxBytes: int64(len(source)), MaxDepth: 3, MaxElements: 3,
	})
	if err != nil {
		t.Fatalf("ParseWithLimits() error = %v", err)
	}
	if got.Root.Name != "System" || len(got.Root.Children) != 1 ||
		len(got.Root.Children[0].Children) != 1 {
		t.Fatalf("parsed document lost expected hierarchy: %#v", got.Root)
	}
}

func TestParseWithLimitsRejectsEachBudgetOverrun(t *testing.T) {
	source := []byte(`<System><Table ID="synthetic-secret"><TableField/></Table></System>`)
	cases := []struct {
		name   string
		limits ParseLimits
	}{
		{name: "bytes", limits: ParseLimits{MaxBytes: int64(len(source) - 1), MaxDepth: 3, MaxElements: 3}},
		{name: "depth", limits: ParseLimits{MaxBytes: int64(len(source)), MaxDepth: 2, MaxElements: 3}},
		{name: "elements", limits: ParseLimits{MaxBytes: int64(len(source)), MaxDepth: 3, MaxElements: 2}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseWithLimits(FormatXead, source, test.limits)
			if err != ErrSourceLimitExceeded {
				t.Fatalf("ParseWithLimits() error = %v, want ErrSourceLimitExceeded", err)
			}
			if !reflect.DeepEqual(got, Document{}) {
				t.Fatalf("limit error returned partial document: %#v", got)
			}
			if strings.Contains(err.Error(), "synthetic-secret") || strings.Contains(err.Error(), "Table") {
				t.Fatalf("limit error disclosed source data: %v", err)
			}
		})
	}
}

func TestParseWithLimitsRequiresAllPositiveBudgets(t *testing.T) {
	source := []byte(`<System/>`)
	cases := []ParseLimits{
		{MaxBytes: 0, MaxDepth: 1, MaxElements: 1},
		{MaxBytes: -1, MaxDepth: 1, MaxElements: 1},
		{MaxBytes: 1, MaxDepth: 0, MaxElements: 1},
		{MaxBytes: 1, MaxDepth: -1, MaxElements: 1},
		{MaxBytes: 1, MaxDepth: 1, MaxElements: 0},
		{MaxBytes: 1, MaxDepth: 1, MaxElements: -1},
	}
	for _, limits := range cases {
		if _, err := ParseWithLimits(FormatXead, source, limits); err != ErrInvalidParseLimits {
			t.Fatalf("ParseWithLimits(%+v) error = %v, want ErrInvalidParseLimits", limits, err)
		}
	}
}

func TestParseWithLimitsMatchesParseForGenerousBudgets(t *testing.T) {
	source := []byte(`<System Version="1.2"><Table ID="T1"><TableField ID="F1" DataTypeID="D1"/></Table><Unknown value="synthetic-private"> text </Unknown></System>`)
	want, err := Parse(FormatXead, source)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseWithLimits(FormatXead, source, ParseLimits{
		MaxBytes: int64(len(source)), MaxDepth: 8, MaxElements: 8,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseWithLimits() differs from Parse()\n got: %#v\nwant: %#v", got, want)
	}
}
