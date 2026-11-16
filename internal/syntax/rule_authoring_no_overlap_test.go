package syntax_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/syntax"
)

const ruleAuthoringWithNoOverlap = `namespace organization

rule organization.department.members.periods_apart:
  organization.membership.starts_on until organization.membership.ends_on or open
    is not overlapping per organization.department.members
    for each organization.membership.person_id

rule organization.item.prices.periods_apart:
  organization.price.valid_from until organization.price.valid_to is not overlapping per organization.item.prices

rule organization.membership.period_in_order:
  organization.membership.starts_on is not after organization.membership.ends_on if present
`

func TestRuleAuthoringReadsNotOverlapping(t *testing.T) {
	src := ruleAuthoringWithNoOverlap
	anchored, err := syntax.ParseRuleAuthoringAnchored(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(anchored.Issues) != 0 {
		t.Fatalf("conversion issues: %v", anchored.Issues)
	}
	want := []semantic.Validation{
		{ID: "organization.department.members.periods_apart", Symbol: "periods_apart", Kind: semantic.ValidationNoOverlap,
			Attribute: "organization.membership.starts_on", Compared: "organization.membership.ends_on",
			Relation: "organization.department.members", GroupBy: "organization.membership.person_id", OpenEnded: true},
		// Neither `or open` nor a grouping is assumed when it is not written.
		{ID: "organization.item.prices.periods_apart", Symbol: "periods_apart", Kind: semantic.ValidationNoOverlap,
			Attribute: "organization.price.valid_from", Compared: "organization.price.valid_to", Relation: "organization.item.prices"},
		{ID: "organization.membership.period_in_order", Symbol: "period_in_order", Kind: semantic.ValidationNotAfter,
			Attribute: "organization.membership.starts_on", Compared: "organization.membership.ends_on", IfPresent: true},
	}
	if !reflect.DeepEqual(anchored.Value.Validations, want) {
		t.Fatalf("validations=%+v\nwant=%+v", anchored.Value.Validations, want)
	}

	first := anchored.File.Rules[0]
	wantEnd := strings.Index(src, "organization.membership.person_id") + len("organization.membership.person_id")
	if first.DeclarationSpan.End.Offset != wantEnd || first.PeriodEnd == nil || first.GroupBy == nil ||
		first.PeriodEnd.Anchor.Start.Offset != strings.Index(src, "organization.membership.ends_on or open") ||
		first.GroupBy.Anchor.Start.Offset != strings.Index(src, "organization.membership.person_id") {
		t.Fatalf("first=%+v, want declaration end %d", first, wantEnd)
	}
	second := anchored.File.Rules[1]
	if second.DeclarationSpan.End.Offset != strings.Index(src, "organization.item.prices\n")+len("organization.item.prices") ||
		second.OpenEnded || second.GroupBy != nil {
		t.Fatalf("second=%+v", second)
	}
}

func TestRuleAuthoringRefusesAnUnclearNotOverlapping(t *testing.T) {
	const head = "namespace organization\n\nrule organization.department.members.periods_apart:\n  organization.membership.starts_on "
	cases := []struct {
		name, tail string
		// at occurs once in the source; the error starts there.
		at, mention string
	}{
		{"no relation", "until organization.membership.ends_on is not overlapping\n", "not overlapping", `requires "per <relation>"`},
		{"no period", "is not overlapping per organization.department.members\n", "organization.membership.starts_on", `write the subject as "<start> until <end>"`},
		{"until without an attribute", "until is not overlapping per organization.department.members\n", "is not", "expected the attribute that ends the period"},
		{"until at the end", "until\n", "until", "expected the attribute that ends the period"},
		{"or without open", "until organization.membership.ends_on or later is not overlapping per organization.department.members\n", "later", `expected "open" after "or"`},
		{"or open before the end", "until or open is not overlapping per organization.department.members\n", "or open", "expected the attribute that ends the period"},
		{"for without each", "until organization.membership.ends_on is not overlapping per organization.department.members for organization.membership.person_id\n", "organization.membership.person_id", `expected "each" after "for"`},
		{"for each without an attribute", "until organization.membership.ends_on is not overlapping per organization.department.members for each\n", "each", "expected the attribute to group rows by"},
		{"a period with another predicate", "until organization.membership.ends_on is positive\n", "organization.membership.ends_on", `is the subject of "not overlapping" only`},
		{"a period with not after", "until organization.membership.ends_on is not after organization.membership.left_on\n", "organization.membership.ends_on", `is the subject of "not overlapping" only`},
		{"a period with unique", "until organization.membership.ends_on is unique per organization.department.members\n", "organization.membership.ends_on", `is the subject of "not overlapping" only`},
		{"a grouping on unique", "is unique per organization.department.members for each organization.membership.person_id\n", "for each", "unexpected"},
		{"the words joined", "until organization.membership.ends_on is not-overlapping per organization.department.members\n", "not-overlapping", "unknown rule predicate"},
		{"overlapping alone", "until organization.membership.ends_on is overlapping per organization.department.members\n", "overlapping", "unknown rule predicate"},
		{"an unqualified end", "until endsOn is not overlapping per organization.department.members\n", "endsOn", "is not a qualified canonical id"},
		{"an unqualified grouping", "until organization.membership.ends_on is not overlapping per organization.department.members for each personId\n", "personId", "is not a qualified canonical id"},
		{"something after the grouping", "until organization.membership.ends_on is not overlapping per organization.department.members for each organization.membership.person_id if present\n", "if present", "unexpected"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := head + tc.tail
			anchored, err := syntax.ParseRuleAuthoringAnchored(src)
			var message string
			var offset int
			var parseError *syntax.ParseError
			switch {
			case errors.As(err, &parseError):
				message, offset = parseError.Msg, parseError.Pos.Offset
			case err == nil && len(anchored.Issues) == 1:
				message, offset = anchored.Issues[0].Msg, anchored.Issues[0].Span.Start.Offset
			default:
				t.Fatalf("err=%v issues=%v validations=%+v", err, anchored.Issues, anchored.Value.Validations)
			}
			if !strings.Contains(message, tc.mention) {
				t.Fatalf("message=%q, want it to mention %q", message, tc.mention)
			}
			if strings.Count(src, tc.at) != 1 || offset != strings.Index(src, tc.at) {
				t.Fatalf("error at offset %d, want %d (%q occurs %d times)", offset, strings.Index(src, tc.at), tc.at, strings.Count(src, tc.at))
			}
			if len(anchored.Value.Validations) != 0 {
				t.Fatalf("a refused rule produced meaning: %+v", anchored.Value.Validations)
			}
		})
	}

	t.Run("until and overlapping cannot be the subject", func(t *testing.T) {
		for _, word := range []string{"until", "overlapping"} {
			src := "namespace organization\n\nrule organization.x.y:\n  " + word + " is positive\n"
			if _, err := syntax.ParseRuleAuthoringAnchored(src); err == nil || !strings.Contains(err.Error(), "cannot be a rule subject") {
				t.Fatalf("%s: err=%v", word, err)
			}
		}
	})
}

func TestRuleFormatRefusesNotOverlapping(t *testing.T) {
	if _, err := syntax.FormatRuleAuthoring(ruleAuthoringWithNoOverlap); !errors.Is(err, syntax.ErrRuleFormatUnsupported) {
		t.Fatalf("err=%v", err)
	}
	// A source whose only rule is about periods is refused for the period
	// itself, not because of another rule beside it.
	onlyPeriods := "namespace organization\n\nrule organization.item.prices.periods_apart:\n" +
		"  organization.price.valid_from until organization.price.valid_to is not overlapping per organization.item.prices\n"
	if out, err := syntax.FormatRuleAuthoring(onlyPeriods); !errors.Is(err, syntax.ErrRuleFormatUnsupported) || !strings.Contains(err.Error(), "periods") {
		t.Fatalf("out=%q err=%v", out, err)
	}
	if kind, ok := syntax.RuleValidationKind(syntax.RulePredicateNotOverlapping); !ok || kind != semantic.ValidationNoOverlap {
		t.Fatalf("kind=%q ok=%v", kind, ok)
	}
}
