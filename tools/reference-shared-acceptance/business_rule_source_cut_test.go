package main

import (
	"bytes"
	"strings"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/syntax"
)

const authoredBusinessRuleSource = `namespace sales

# The following comment must never be taken as the source anchor:
# candidate-key businessLine { field sales.order_line.ghost }
data OrderLine {
    attribute orderId type std.identifier
    attribute lineNumber type std.integer
    attribute itemCode type std.code
    candidate-key businessLine {
        field sales.order_line.order_id
        field sales.order_line.line_number
    }
    functional-dependency itemByLine {
        determinant sales.order_line.order_id
        determinant sales.order_line.line_number
        dependent sales.order_line.item_code
    }
}
data Customer {
    attribute customerId type std.identifier
    candidate-key customerIdentity {
        field sales.customer.customer_id
    }
}
`

func TestRuleSourceCutTargetsExactDataAndRuleAnchor(t *testing.T) {
	before := []byte(authoredBusinessRuleSource)
	keyRange, after, err := planBusinessRuleSourceCut(before, "candidate-key", "sales.order_line", "sales.order_line.business_line")
	if err != nil {
		t.Fatal(err)
	}
	if keyRange.Kind != "candidate-key" || keyRange.RuleID != "sales.order_line.business_line" ||
		keyRange.DataID != "sales.order_line" ||
		keyRange.BeforeFingerprint != sourceindex.Fingerprint(before) ||
		keyRange.AfterFingerprint != sourceindex.Fingerprint(after) {
		t.Fatalf("source cut evidence is not bound to exact before/after bytes: %+v", keyRange)
	}
	r := keyRange.DefinitionRange
	if string(before[r.Start.Offset:r.End.Offset]) != "businessLine" ||
		keyRange.RemovedRange.Start.Offset >= keyRange.RemovedRange.End.Offset ||
		string(before[keyRange.RemovedRange.Start.Offset:keyRange.RemovedRange.End.Offset]) != keyRange.RemovedText {
		t.Fatalf("wrong rule identifier / complete block range: %+v", keyRange)
	}
	if !strings.HasPrefix(keyRange.RemovedText, "    candidate-key businessLine {\n") ||
		!strings.HasSuffix(keyRange.RemovedText, "    }\n") {
		t.Fatalf("cut was not complete standalone rule: %q", keyRange.RemovedText)
	}
	if strings.Contains(string(after), "\n    candidate-key businessLine {") ||
		!strings.Contains(string(after), "    functional-dependency itemByLine {") ||
		!strings.Contains(string(after), "    candidate-key customerIdentity {") ||
		!strings.Contains(string(after), "# candidate-key businessLine { field sales.order_line.ghost }") {
		t.Fatalf("cut removed wrong Data/member or changed unrelated comments: %s", after)
	}
	reconstructed := append([]byte(nil), before[:keyRange.RemovedRange.Start.Offset]...)
	reconstructed = append(reconstructed, before[keyRange.RemovedRange.End.Offset:]...)
	if !bytes.Equal(reconstructed, after) || !bytes.Equal(before, []byte(authoredBusinessRuleSource)) {
		t.Fatal("cut modified unrelated bytes or its own input")
	}
	parsed, err := syntax.ParseSemanticFileAnchored(string(after))
	if err != nil || len(parsed.Value.Data) != 2 ||
		len(parsed.Value.Data[0].CandidateKeys) != 0 ||
		len(parsed.Value.Data[0].FunctionalDependencies) != 1 ||
		len(parsed.Value.Data[1].CandidateKeys) != 1 {
		t.Fatalf("cut left inconsistent Data or FD: %+v err=%v", parsed.Value.Data, err)
	}
	if _, _, err := planBusinessRuleSourceCut(after, "candidate-key", "sales.order_line", "sales.order_line.business_line"); err == nil {
		t.Fatal("second removal must reject missing rule")
	}
}

func TestRuleSourceCutPreservesCRLFAndNestedFD(t *testing.T) {
	before := []byte(strings.ReplaceAll(authoredBusinessRuleSource, "\n", "\r\n"))
	cut, after, err := planBusinessRuleSourceCut(before, "functional-dependency", "sales.order_line", "sales.order_line.item_by_line")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(after), "functional-dependency itemByLine {") ||
		!strings.Contains(string(after), "candidate-key businessLine {") ||
		!strings.HasSuffix(cut.RemovedText, "    }\r\n") ||
		!strings.Contains(string(after), "    }\r\n}\r\ndata Customer {") {
		t.Fatalf("CRLF or adjacent nested rules were not preserved: %+v / %q", cut, after)
	}
	if got := sourceindex.Fingerprint(after); got != cut.AfterFingerprint {
		t.Fatalf("wrong source fingerprint %s vs %s", got, cut.AfterFingerprint)
	}
}

func TestRuleSourceCutRejectsWrongIdentityAndAmbiguousOwner(t *testing.T) {
	tests := []struct {
		name, kind, data, id string
		source string
	}{
		{"unknown kind", "relation", "sales.order_line", "sales.order_line.business_line", authoredBusinessRuleSource},
		{"missing rule", "candidate-key", "sales.order_line", "sales.order_line.missing", authoredBusinessRuleSource},
		{"wrong kind", "functional-dependency", "sales.order_line", "sales.order_line.business_line", authoredBusinessRuleSource},
		{"wrong Data", "candidate-key", "sales.customer", "sales.order_line.business_line", authoredBusinessRuleSource},
		{"invalid Data ID", "candidate-key", "../sales", "sales.order_line.business_line", authoredBusinessRuleSource},
		{"invalid rule ID", "candidate-key", "sales.order_line", "sales.order_line.bad id", authoredBusinessRuleSource},
		{"broken rule body", "candidate-key", "sales.order_line", "sales.order_line.business_line",
			strings.Replace(authoredBusinessRuleSource, "field sales.order_line.line_number\n", "unknown sales.order_line.line_number\n", 1)},
		{"no rule block terminator", "candidate-key", "sales.order_line", "sales.order_line.business_line",
			strings.Replace(authoredBusinessRuleSource, "    }\n    functional-dependency", "    functional-dependency", 1)},
		{"inline rule declaration", "candidate-key", "sales.order_line", "sales.order_line.business_line",
			strings.Replace(authoredBusinessRuleSource, "    candidate-key businessLine {", "    attribute someCode type std.code candidate-key businessLine {", 1)},
		{"inline rule terminator", "candidate-key", "sales.order_line", "sales.order_line.business_line",
			strings.Replace(authoredBusinessRuleSource, "    }\n    functional-dependency", "    } attribute unused type std.code\n    functional-dependency", 1)},
		{"duplicate Data", "candidate-key", "sales.order_line", "sales.order_line.business_line",
			authoredBusinessRuleSource + "\ndata OrderLine {\n    attribute orderId type std.identifier\n    candidate-key businessLine { field sales.order_line.order_id }\n}\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			before := []byte(tc.source)
			copyBefore := bytes.Clone(before)
			cut, after, err := planBusinessRuleSourceCut(before, tc.kind, tc.data, tc.id)
			if err == nil || after != nil || cut != (BusinessRuleSourceCut{}) {
				t.Fatalf("unsafe cut was allowed: %+v after=%q err=%v", cut, after, err)
			}
			if !bytes.Equal(copyBefore, before) {
				t.Fatal("rejected cut changed source")
			}
		})
	}
}

func TestRuleSourceCutAllowsCommentsButRejectsTrailingCode(t *testing.T) {
	source := strings.Replace(authoredBusinessRuleSource,
		"    }\n    functional-dependency itemByLine",
		"    } # approved as a comment\n    functional-dependency itemByLine", 1)
	if _, _, err := planBusinessRuleSourceCut([]byte(source), "candidate-key", "sales.order_line", "sales.order_line.business_line"); err == nil {
		t.Fatal("cut must not silently discard an inline comment with assertion context")
	}
}
