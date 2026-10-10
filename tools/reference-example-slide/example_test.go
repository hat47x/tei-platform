package exampleslide

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func fixture(t *testing.T) Input {
	t.Helper()
	data, err := os.ReadFile("testdata/partial_inspection.json")
	if err != nil {
		t.Fatal(err)
	}
	var input Input
	if err := json.Unmarshal(data, &input); err != nil {
		t.Fatal(err)
	}
	return input
}

func TestBuildAndSVG(t *testing.T) {
	in := fixture(t)
	out, err := Build(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Slides) != 3 {
		t.Fatalf("expected three slides, got %d", len(out.Slides))
	}
	if len(out.Questions) != 1 || out.Questions[0].ID != "residual" {
		t.Fatal("unresolved question was lost")
	}
	if len(out.Slides[0].Tables[1].Records) != 0 {
		t.Fatal("receipt present before receipt stage")
	}
	if len(out.Slides[1].Tables[1].Records) != 1 {
		t.Fatal("receipt absent after receipt stage")
	}
	last := out.Slides[2]
	if len(last.Tables[2].Records) != 1 {
		t.Fatal("inspection absent from final stage")
	}
	if last.Tables[1].X != 315 || last.Tables[2].X != 610 {
		t.Fatal("authored positions were modified")
	}
	if len(last.Connections) != 2 {
		t.Fatalf("expected two reference edges, got %d", len(last.Connections))
	}
	receipt := last.Tables[1]
	if len(receipt.Fields) != 2 || receipt.Fields[0] != "number" || receipt.Fields[1] != "order" {
		t.Fatalf("diagram field overrides failed: %v", receipt.Fields)
	}
	svg := SVG(last)
	if !strings.Contains(svg, "x=\"315\"") || !strings.Contains(svg, "<path") {
		t.Fatal("SVG lost manual placement or references")
	}
	// Hiding the field changes only the presentation; not the relationship.
	in.Diagram.Placements[1].Show = nil
	withoutField, err := Build(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(withoutField.Slides[2].Connections) != 2 {
		t.Fatal("hidden foreign key removed a semantic relation")
	}
}

func TestFramesDoNotDefineExecutionOrder(t *testing.T) {
	in := fixture(t)
	in.Presentation.Frames[0], in.Presentation.Frames[2] =
		in.Presentation.Frames[2], in.Presentation.Frames[0]
	out, err := Build(in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Slides[0].Stage != "inspected" || len(out.Slides[0].Tables[2].Records) != 1 {
		t.Fatal("stage was incorrectly derived from presentation order")
	}
}

func TestValidationRejectsInvalidCases(t *testing.T) {
	t.Run("stale", func(t *testing.T) {
		in := fixture(t)
		in.Example.ModelRevision = "old"
		if _, err := Build(in); err == nil || !strings.Contains(err.Error(), "stale") {
			t.Fatalf("expected stale revision error, got %v", err)
		}
	})
	t.Run("broken reference", func(t *testing.T) {
		in := fixture(t)
		in.Example.Stages[0].Changes[0].Record.Values["order"] = json.RawMessage([]byte("\"missing\""))
		if _, err := Build(in); err == nil || !strings.Contains(err.Error(), "broken reference") {
			t.Fatalf("expected broken reference error, got %v", err)
		}
	})
	t.Run("invalid type", func(t *testing.T) {
		in := fixture(t)
		in.Example.Base[0].Values["quantity"] = json.RawMessage([]byte("\"ten\""))
		if _, err := Build(in); err == nil || !strings.Contains(err.Error(), "expected number") {
			t.Fatalf("expected numeric type error, got %v", err)
		}
	})
	t.Run("duplicate record", func(t *testing.T) {
		in := fixture(t)
		in.Example.Base = append(in.Example.Base, in.Example.Base[0])
		if _, err := Build(in); err == nil || !strings.Contains(err.Error(), "duplicate record") {
			t.Fatalf("expected duplicate ID error, got %v", err)
		}
	})
	t.Run("unknown stage", func(t *testing.T) {
		in := fixture(t)
		in.Presentation.Frames[0].Stage = "not-a-stage"
		if _, err := Build(in); err == nil || !strings.Contains(err.Error(), "unknown stage") {
			t.Fatalf("expected unknown stage error, got %v", err)
		}
	})
	t.Run("invalid field override", func(t *testing.T) {
		in := fixture(t)
		in.Diagram.Placements[1].Show = []string{"nonesuch"}
		if _, err := Build(in); err == nil || !strings.Contains(err.Error(), "display override") {
			t.Fatalf("expected invalid field override, got %v", err)
		}
	})
}

func TestProjectionDoesNotMutateSource(t *testing.T) {
	in := fixture(t)
	before, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Build(in); err != nil {
		t.Fatal(err)
	}
	after, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("projection modified authored source")
	}
}

func TestSVGEscapesLabels(t *testing.T) {
	slide := Slide{Tables: []ShownTable{{
		PlacementID: "box", Table: "a<&", X: 5, Y: 6,
		Fields: []string{"x<&"},
	}}}
	svg := SVG(slide)
	if strings.Contains(svg, "a<&") || !strings.Contains(svg, "a&lt;&amp;") {
		t.Fatal("SVG contains unescaped field or table label")
	}
}

func TestSVGIncludesAllRecordsAndReviewCounts(t *testing.T) {
	slide := Slide{
		ID: "reviewed", Title: "検収<&", Stage: "after<&",
		Tables: []ShownTable{{
			PlacementID: "box", Table: "items<&", X: 315, Y: 30,
			Fields: []string{"key"},
			Records: []ShownRecord{
				{ID: "first", Values: map[string]json.RawMessage{"key": json.RawMessage("\"alpha<&\"")}},
				{ID: "second", Values: map[string]json.RawMessage{"key": json.RawMessage("\"beta\"")}},
			},
		}},
		RelationFindings: []RelationFinding{{Result: "unresolved"}},
	}
	svg := SVG(slide)
	for _, present := range []string{
		"first", "second", "alpha&lt;&amp;", "beta",
		"検収&lt;&amp;", "items&lt;&amp;", "Unresolved: 1", "x=\"315\"",
	} {
		if !strings.Contains(svg, present) {
			t.Fatalf("SVG lost record or review context %q: %s", present, svg)
		}
	}
	for _, unsafe := range []string{"alpha<&", "items<&", "検収<&"} {
		if strings.Contains(svg, unsafe) {
			t.Fatalf("SVG contains unescaped label %q", unsafe)
		}
	}
}

func TestSVGShowsSchemaWhenNoExampleRecordExists(t *testing.T) {
	svg := SVG(Slide{Tables: []ShownTable{{
		PlacementID: "empty-box", Table: "empty", Fields: []string{"code", "quantity"},
	}}})
	for _, expected := range []string{"(no example records)", "code", "quantity"} {
		if !strings.Contains(svg, expected) {
			t.Fatalf("empty table lost schema field %q", expected)
		}
	}
}

func TestSVGEdgeLabelDoesNotInjectMarkup(t *testing.T) {
	svg := SVG(Slide{
		Tables: []ShownTable{
			{PlacementID: "parent", X: 0, Y: 0},
			{PlacementID: "child", X: 315, Y: 0},
		},
		Connections: []Connection{{
			Parent: "parent", Child: "child", RelationID: "edge\"<&",
		}},
	})
	if !strings.Contains(svg, "aria-label=\"edge&#34;&lt;&amp;\"") {
		t.Fatalf("reference edge label was not escaped: %s", svg)
	}
}
