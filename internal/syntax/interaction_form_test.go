package syntax

import (
	"strings"
	"testing"

	"tei/internal/application"
)

// `interaction` is a block, but writing it as a line gave only
// `unexpected token "command"`. An author had to guess whether interaction is a
// line or a block (#4169).
func TestInteractionLineFormNamesTheBlock(t *testing.T) {
	src := "pattern returns.note record {\n    data sales.order\n    interaction item sales.order.order_id\n}\n"
	_, err := NewParser(src).ParseApplicationPattern()
	if err == nil {
		t.Fatal("expected a parse error")
	}
	message, ok := ErrorMessage(err)
	if !ok {
		t.Fatal("error has no message")
	}
	if !strings.Contains(message, "interaction is a block") {
		t.Errorf("message does not say interaction is a block: %s", message)
	}
	if !strings.Contains(message, "interaction { item") {
		t.Errorf("message does not show the block form: %s", message)
	}
	if strings.Contains(message, `unexpected token`) {
		t.Errorf("message is still the generic one: %s", message)
	}
}

// A command pattern cannot take an interaction at all, so the message must not
// suggest a block that would be rejected immediately. Offering it would cost
// another round trip.
func TestCommandPatternInteractionLineSaysItIsNotAccepted(t *testing.T) {
	src := "pattern returns.accept_return command {\n    capability sales.record_return\n    interaction command accept\n}\n"
	_, err := NewParser(src).ParseApplicationPattern()
	if err == nil {
		t.Fatal("expected a parse error")
	}
	message, ok := ErrorMessage(err)
	if !ok {
		t.Fatal("error has no message")
	}
	if !strings.Contains(message, "does not accept one") {
		t.Errorf("message does not say the command pattern rejects an interaction: %s", message)
	}
	if strings.Contains(message, "interaction { item") || strings.Contains(message, "interaction { command") {
		t.Errorf("message suggests an example that the contract would reject: %s", message)
	}
}

// The block forms keep their existing behaviour, including the contract that
// limits the first interaction to record patterns.
func TestInteractionBlockFormsAreUnchanged(t *testing.T) {
	rec, err := NewParser("pattern returns.note record {\n    data sales.order\n    interaction {\n        item sales.order.order_id\n    }\n}\n").ParseApplicationPattern()
	if err != nil {
		t.Fatalf("record pattern with an interaction block should parse: %v", err)
	}
	if rec.Interaction == nil || len(rec.Interaction.Items) == 0 {
		t.Errorf("interaction was not recorded: %+v", rec.Interaction)
	}

	// Parsing succeeds; the contract is enforced later, so this only pins that
	// the parser still accepts the shape it accepted before.
	if _, err := NewParser("pattern returns.accept_return command {\n    capability sales.record_return\n    interaction {\n        command accept\n    }\n}\n").ParseApplicationPattern(); err != nil {
		t.Fatalf("the parser should still accept the block form: %v", err)
	}
}

// The example must follow the pattern kind, so one message is not reused where
// it would be wrong.
func TestInteractionExampleFollowsThePatternKind(t *testing.T) {
	if !isRecordPatternKind(application.PatternRecord) {
		t.Error("record must accept an interaction")
	}
	for _, kind := range []application.PatternKind{application.PatternBrowse, application.PatternCommand, application.PatternReference, application.PatternHeaderDetail} {
		if isRecordPatternKind(kind) {
			t.Errorf("%q must not be treated as accepting an interaction", kind)
		}
	}
}
