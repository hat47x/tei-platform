package syntax_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/syntax"
)

const ruleAuthoringWithNotAfter = `namespace organization

rule organization.membership.period_in_order:
  organization.membership.starts_on is not after organization.membership.ends_on if present

rule organization.leave.period_in_order:
  organization.leave.first_day is not after organization.leave.last_day

rule organization.membership.allocation_positive:
  organization.membership.allocation is positive
`

func TestRuleAuthoringReadsNotAfter(t *testing.T) {
	src := ruleAuthoringWithNotAfter
	anchored, err := syntax.ParseRuleAuthoringAnchored(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(anchored.Issues) != 0 {
		t.Fatalf("conversion issues: %v", anchored.Issues)
	}
	want := []semantic.Validation{
		{ID: "organization.membership.period_in_order", Symbol: "period_in_order", Kind: semantic.ValidationNotAfter,
			Attribute: "organization.membership.starts_on", Compared: "organization.membership.ends_on", IfPresent: true},
		// Without `if present` nothing is assumed about an empty date.
		{ID: "organization.leave.period_in_order", Symbol: "period_in_order", Kind: semantic.ValidationNotAfter,
			Attribute: "organization.leave.first_day", Compared: "organization.leave.last_day"},
		{ID: "organization.membership.allocation_positive", Symbol: "allocation_positive", Kind: semantic.ValidationPositive,
			Attribute: "organization.membership.allocation"},
	}
	if !reflect.DeepEqual(anchored.Value.Validations, want) {
		t.Fatalf("validations=%+v\nwant=%+v", anchored.Value.Validations, want)
	}

	// The declaration ends where its last authored word ends.
	first := anchored.File.Rules[0]
	wantEnd := strings.Index(src, "if present") + len("if present")
	if first.DeclarationSpan.End.Offset != wantEnd || first.Compared == nil ||
		first.Compared.Anchor.Start.Offset != strings.Index(src, "organization.membership.ends_on") {
		t.Fatalf("declaration end=%d compared=%+v, want end %d", first.DeclarationSpan.End.Offset, first.Compared, wantEnd)
	}
	second := anchored.File.Rules[1]
	if second.DeclarationSpan.End.Offset != strings.Index(src, "organization.leave.last_day")+len("organization.leave.last_day") || second.IfPresent {
		t.Fatalf("second=%+v", second)
	}
}

func TestRuleAuthoringRefusesAnUnclearNotAfter(t *testing.T) {
	const head = "namespace organization\n\nrule organization.membership.period_in_order:\n  organization.membership.starts_on is "
	cases := []struct {
		name, tail string
		// at occurs once in the source; the error starts there. A word in
		// the wrong place is reported as itself; a missing word is reported
		// at the word before it.
		at, mention string
	}{
		{"not without after", "not before organization.membership.ends_on\n", "before", `expected "after" after "not"`},
		{"not at the end", "not\n", "not\n", `expected "after" after "not"`},
		{"nothing to compare with", "not after\n", "after\n", "expected the attribute to compare with"},
		{"the next rule where the attribute should be", "not after\n\nrule organization.x.y:\n  organization.x.z is positive\n", "rule organization.x.y", "expected the attribute to compare with"},
		{"if without an attribute", "not after if present\n", "if present", "expected the attribute to compare with"},
		{"a condition other than present", "not after organization.membership.ends_on if known\n", "known", `expected "present" after "if"`},
		{"if at the end", "not after organization.membership.ends_on if\n", "if\n", `expected "present" after "if"`},
		{"a scope", "not after organization.membership.ends_on per organization.department.members\n", "organization.department.members", "compares two dates of one record and accepts no scope"},
		{"a second comparison", "not after organization.membership.ends_on organization.membership.left_on\n", "organization.membership.left_on", "unexpected"},
		{"the words joined", "not-after organization.membership.ends_on\n", "not-after", "unknown rule predicate"},
		{"after alone", "after organization.membership.ends_on\n", "after", "unknown rule predicate"},
		{"an unqualified attribute", "not after endsOn\n", "endsOn", "is not a qualified canonical id"},
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
				t.Fatalf("error at offset %d, want %d (%q)", offset, strings.Index(src, tc.at), tc.at)
			}
			if len(anchored.Value.Validations) != 0 {
				t.Fatalf("a refused rule produced meaning: %+v", anchored.Value.Validations)
			}
		})
	}

	t.Run("not and after cannot be the subject", func(t *testing.T) {
		for _, word := range []string{"not", "after"} {
			src := "namespace organization\n\nrule organization.x.y:\n  " + word + " is positive\n"
			if _, err := syntax.ParseRuleAuthoringAnchored(src); err == nil || !strings.Contains(err.Error(), "cannot be a rule subject") {
				t.Fatalf("%s: err=%v", word, err)
			}
		}
	})
}

// Formatting has no layout for a comparison yet. It refuses the source; it
// does not drop the second attribute.
func TestRuleFormatRefusesNotAfter(t *testing.T) {
	if _, err := syntax.FormatRuleAuthoring(ruleAuthoringWithNotAfter); !errors.Is(err, syntax.ErrRuleFormatUnsupported) {
		t.Fatalf("err=%v", err)
	}
	if kind, ok := syntax.RuleValidationKind(syntax.RulePredicateNotAfter); !ok || kind != semantic.ValidationNotAfter {
		t.Fatalf("kind=%q ok=%v", kind, ok)
	}
}
