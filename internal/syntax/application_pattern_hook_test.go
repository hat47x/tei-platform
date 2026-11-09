package syntax_test

import (
	"testing"

	"tei/internal/syntax"
)

func TestParseRecordApplicationPatternHook(t *testing.T) {
	src := `pattern order_entry.order_record record {
    data sales.order
    interaction {
        hook before-update {
            reads sales.order.order_id
            uses sales.submit
            produces sales.order.customer_id
            on-failure reject
        }
    }
}`
	got, err := syntax.NewParser(src).ParseApplicationPattern()
	if err != nil {
		t.Fatal(err)
	}
	if got.Interaction == nil || len(got.Interaction.Hooks) != 1 {
		t.Fatalf("hooks=%#v", got.Interaction)
	}
	hook := got.Interaction.Hooks[0]
	if hook.Phase != "before-update" || len(hook.Reads) != 1 || len(hook.Uses) != 1 || len(hook.Produces) != 1 || hook.Failure != "reject" {
		t.Fatalf("hook=%#v", hook)
	}
}

// TestParseRecordApplicationPatternHookScope checks that the scope is read and that
// a word outside the vocabulary is refused instead of reading as "no scope".
func TestParseRecordApplicationPatternHookScope(t *testing.T) {
	src := `pattern order_entry.order_record record {
    data sales.order
    interaction {
        hook before-update {
            scope record
            reads sales.order.order_id
        }
    }
}`
	got, err := syntax.NewParser(src).ParseApplicationPattern()
	if err != nil {
		t.Fatal(err)
	}
	if got.Interaction == nil || len(got.Interaction.Hooks) != 1 {
		t.Fatalf("hooks=%#v", got.Interaction)
	}
	if got.Interaction.Hooks[0].Scope != "record" {
		t.Fatalf("scope=%q", got.Interaction.Hooks[0].Scope)
	}

	bad := `pattern order_entry.order_record record {
    data sales.order
    interaction {
        hook before-update {
            scope column
        }
    }
}`
	if _, err := syntax.NewParser(bad).ParseApplicationPattern(); err == nil {
		t.Fatal("an unsupported scope was accepted")
	}
}
