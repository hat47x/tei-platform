package syntax

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"tei/internal/core"
)

// This file turns one typed change of a rule into one byte-range replacement
// of the authored document.
//
// The change is stated against meaning (which rule, which attribute), and the
// replacement is located by the span the parse recorded for that rule. It is
// never located by searching for text: the same attribute name may occur in
// prose, in a quoted example, and in another rule.
//
// Planning and applying are separate. A plan names the exact document it was
// made for, and applying it to any other bytes is refused. Nothing here writes
// a file, formats a document, or accepts meaning.

// TextEdit replaces one byte range of one exact document.
type TextEdit struct {
	// BaseFingerprint identifies the document the edit was planned on, in the
	// same form as a Source Index artifact fingerprint.
	BaseFingerprint string
	// Start and End are byte offsets into that document.
	Start int
	End   int
	// Old is the text the range held when the edit was planned.
	Old string
	// New is the replacement.
	New string
}

// TextFingerprint identifies document bytes. It matches the Source Index
// artifact fingerprint of the same bytes.
func TextFingerprint(src string) string {
	sum := sha256.Sum256([]byte(src))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// PlanRuleSubjectEdit plans the change "rule ruleID constrains attribute" for
// a TEI Markdown source. It reads the document once and changes nothing.
func PlanRuleSubjectEdit(src string, ruleID, attribute core.CanonicalID) (TextEdit, error) {
	if _, err := core.ParseCanonicalID(string(attribute)); err != nil {
		return TextEdit{}, fmt.Errorf("attribute %q: %w", attribute, err)
	}
	if !strings.Contains(string(attribute), ".") {
		return TextEdit{}, fmt.Errorf("attribute %q is not a qualified canonical id", attribute)
	}
	regions, err := ParseRuleAuthoringMarkdown(src)
	if err != nil {
		return TextEdit{}, err
	}
	for _, region := range regions {
		for _, rule := range region.Rules.File.Rules {
			if rule.ID != ruleID {
				continue
			}
			span := rule.SubjectSpan
			old := src[span.Start.Offset:span.End.Offset]
			if old == string(attribute) {
				return TextEdit{}, fmt.Errorf("rule %s already constrains %s", ruleID, attribute)
			}
			return TextEdit{
				BaseFingerprint: TextFingerprint(src),
				Start:           span.Start.Offset,
				End:             span.End.Offset,
				Old:             old,
				New:             string(attribute),
			}, nil
		}
	}
	return TextEdit{}, fmt.Errorf("rule %s is not declared in this source", ruleID)
}

// ApplyTextEdit returns src with the edit applied. It refuses a document other
// than the one the edit was planned on, even when the range still happens to
// hold the same text, and it changes no byte outside the range.
func ApplyTextEdit(src string, edit TextEdit) (string, error) {
	if got := TextFingerprint(src); got != edit.BaseFingerprint {
		return "", fmt.Errorf("stale edit: it was planned on %s, but the document is now %s; plan it again", edit.BaseFingerprint, got)
	}
	if edit.Start < 0 || edit.End < edit.Start || edit.End > len(src) {
		return "", fmt.Errorf("edit range [%d,%d) is outside the document", edit.Start, edit.End)
	}
	if src[edit.Start:edit.End] != edit.Old {
		return "", fmt.Errorf("edit range [%d,%d) no longer holds %q", edit.Start, edit.End, edit.Old)
	}
	return src[:edit.Start] + edit.New + src[edit.End:], nil
}
