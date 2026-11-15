package syntax_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"tei/internal/syntax"
)

func TestRuleAuthoringMarkdownSpansSliceTheWholeDocument(t *testing.T) {
	variants := map[string]string{
		"lf":       teiMarkdownTwoRegions,
		"crlf":     strings.ReplaceAll(teiMarkdownTwoRegions, "\n", "\r\n"),
		"bom":      "\xef\xbb\xbf" + teiMarkdownTwoRegions,
		"bom+crlf": "\xef\xbb\xbf" + strings.ReplaceAll(teiMarkdownTwoRegions, "\n", "\r\n"),
	}
	for name, src := range variants {
		t.Run(name, func(t *testing.T) {
			regions, err := syntax.ParseRuleAuthoringMarkdown(src)
			if err != nil {
				t.Fatal(err)
			}
			if len(regions) != 2 || len(regions[0].Rules.File.Rules) != 1 || len(regions[1].Rules.File.Rules) != 1 {
				t.Fatalf("unexpected shape: %+v", regions)
			}
			positive := regions[0].Rules.File.Rules[0]
			unique := regions[1].Rules.File.Rules[0]

			// Every expected offset is located independently in the document. The
			// Japanese prose before the first region is multi-byte, so a span
			// counted in characters instead of bytes would miss.
			checks := []struct {
				what string
				span syntax.Anchor
				text string
				at   int
			}{
				{"positive id", positive.IDSpan, "sales.order_line.quantity_positive",
					uniqueOffset(t, src, "sales.order_line.quantity_positive")},
				{"positive subject", positive.SubjectSpan, "sales.order_line.quantity",
					uniqueOffset(t, src, "sales.order_line.quantity is")},
				{"positive predicate", positive.PredicateSpan, "positive",
					uniqueOffset(t, src, "sales.order_line.quantity is positive") + len("sales.order_line.quantity is ")},
				{"unique id", unique.IDSpan, "sales.order.lines.line_number_unique",
					uniqueOffset(t, src, "sales.order.lines.line_number_unique")},
				{"unique scope", unique.ScopeSpan, "sales.order.lines",
					uniqueOffset(t, src, "per sales.order.lines") + len("per ")},
			}
			for _, check := range checks {
				if check.span.Start.Offset != check.at {
					t.Errorf("%s starts at %d, want %d", check.what, check.span.Start.Offset, check.at)
					continue
				}
				if got := src[check.span.Start.Offset:check.span.End.Offset]; got != check.text {
					t.Errorf("%s slices %q, want %q", check.what, got, check.text)
				}
				wantLine := 1 + strings.Count(src[:check.at], "\n")
				wantColumn := check.at - strings.LastIndex(src[:check.at], "\n")
				if check.span.Start.Line != wantLine || check.span.Start.Column != wantColumn {
					t.Errorf("%s is at %d:%d, want %d:%d", check.what,
						check.span.Start.Line, check.span.Start.Column, wantLine, wantColumn)
				}
			}
		})
	}
}

// A rule must mean the same thing whether it is written in a standalone rule
// source or inside a Markdown region. The standalone parse is the reference.
func TestRuleAuthoringMarkdownMeansTheSameAsTheStandaloneSource(t *testing.T) {
	regions, err := syntax.ParseRuleAuthoringMarkdown(teiMarkdownTwoRegions)
	if err != nil {
		t.Fatal(err)
	}
	for i, region := range regions {
		start, end := region.Region.Content.Start.Offset, region.Region.Content.End.Offset
		standalone, err := syntax.ParseRuleAuthoringAnchored(teiMarkdownTwoRegions[start:end])
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(region.Rules.Value, standalone.Value) {
			t.Fatalf("region %d meaning differs:\n markdown=%+v\n standalone=%+v", i, region.Rules.Value, standalone.Value)
		}
		for j, rule := range region.Rules.File.Rules {
			if got, want := rule.IDSpan.Start.Offset, standalone.File.Rules[j].IDSpan.Start.Offset+start; got != want {
				t.Fatalf("region %d rule %d offset=%d, want standalone offset shifted by the region start %d", i, j, got, want)
			}
		}
	}
}

func TestRuleAuthoringMarkdownRegionsDoNotShareANamespace(t *testing.T) {
	src := "```tei\nnamespace sales\nrule sales.a: sales.x.y is positive\n```\n\n" +
		"```tei\nrule sales.b: sales.x.z is positive\n```\n"
	_, err := syntax.ParseRuleAuthoringMarkdown(src)
	var parseErr *syntax.ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("a region without its own namespace was accepted: %v", err)
	}
	if want := uniqueOffset(t, src, "rule sales.b"); parseErr.Pos.Offset != want {
		t.Fatalf("error offset=%d, want the second region's first token at %d", parseErr.Pos.Offset, want)
	}
}

func TestRuleAuthoringMarkdownRefusesOneIdentityInTwoRegions(t *testing.T) {
	src := "```tei\nnamespace sales\nrule sales.same: sales.x.y is positive\n```\n\n" +
		"```tei\nnamespace sales\nrule sales.same: sales.x.z is positive\n```\n"
	_, err := syntax.ParseRuleAuthoringMarkdown(src)
	var parseErr *syntax.ParseError
	if !errors.As(err, &parseErr) || !strings.Contains(parseErr.Msg, "declared more than once") {
		t.Fatalf("duplicate identity across regions was accepted: %v", err)
	}
	if want := strings.LastIndex(src, "sales.same"); parseErr.Pos.Offset != want {
		t.Fatalf("error offset=%d, want the second declaration at %d", parseErr.Pos.Offset, want)
	}
}

func TestRuleAuthoringMarkdownErrorsStayInsideTheirRegion(t *testing.T) {
	// The predicate is missing at the end of the region. The error must point
	// into the region, not at the end of the document after the prose.
	src := "```tei\nnamespace sales\nrule sales.a: sales.x.y is\n```\n\n後ろの説明文。\n"
	_, err := syntax.ParseRuleAuthoringMarkdown(src)
	var parseErr *syntax.ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("missing predicate was accepted: %v", err)
	}
	if limit := uniqueOffset(t, src, "```\n\n"); parseErr.Pos.Offset > limit {
		t.Fatalf("error offset=%d is outside the region that ends at %d", parseErr.Pos.Offset, limit)
	}
}
