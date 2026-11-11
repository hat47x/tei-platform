package syntax

import (
	"strings"
	"testing"
)

// An entry that only exists inside a block, written after the block's closing
// brace, is not a stray declaration: the block is right and its brace is early.
// Round 64 saw only `unexpected trailing token "bind"`, which does not say where
// it belongs (#4169).
func TestBlockOnlyEntryAfterClosingBraceNamesTheBlock(t *testing.T) {
	src := "environment staging {\n    bundle tei.reference.providers ^1\n}\n\nbind order_entry.sales -> sales_core.dev\n"
	_, err := NewParser(src).ParseEnvironment()
	if err == nil {
		t.Fatal("expected a parse error")
	}
	message, ok := ErrorMessage(err)
	if !ok {
		t.Fatal("error has no message")
	}
	if !strings.Contains(message, "belongs inside one") {
		t.Errorf("message does not say the entry belongs inside a block: %s", message)
	}
	if !strings.Contains(message, "bind <application>.<alias> -> <database-instance>") {
		t.Errorf("message does not show the declaration form: %s", message)
	}
	if strings.Contains(message, "unexpected trailing token") {
		t.Errorf("message is still the generic one: %s", message)
	}
}

// The hint must cover the entries that only exist inside a block, and must not
// swallow an unknown token.
func TestBlockOnlyEntryHintCoversBlockEntriesOnly(t *testing.T) {
	for _, text := range []string{"bind", "resource", "grant", "lease-policy", "feature", "column", "primary-key", "index"} {
		if blockOnlyEntryHint(text) == "" {
			t.Errorf("%q has no hint although it only exists inside a block", text)
		}
	}
	for _, text := range []string{"frobnicate", "application", "environment", "", "migration"} {
		if blockOnlyEntryHint(text) != "" {
			t.Errorf("%q must keep its own message", text)
		}
	}
}

// A top-level declaration keyword keeps the misplaced-declaration message; the
// block hint must not take precedence over it.
func TestTopLevelKeywordStillGetsTheDeclarationMessage(t *testing.T) {
	src := "environment staging {\n    bundle tei.reference.providers ^1\n}\n\nmigration x {\n}\n"
	_, err := NewParser(src).ParseEnvironment()
	if err == nil {
		t.Fatal("expected a parse error")
	}
	message, ok := ErrorMessage(err)
	if !ok {
		t.Fatal("error has no message")
	}
	if !strings.Contains(message, "starts a migration") {
		t.Errorf("message is %q, want the misplaced-declaration wording", message)
	}
}
