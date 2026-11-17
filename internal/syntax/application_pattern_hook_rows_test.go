package syntax_test

import (
	"testing"

	"tei/internal/syntax"
)

// hookWithRows is the declaration shape a row-scoped hook uses to name where the
// rows come from. A record pattern cannot declare a query of its own, so the
// source is the only place the row set is named.
func hookWithRows(rows string) string {
	return `pattern order_entry.order_record record {
    data sales.order
    interaction {
        hook before-update {
            scope row
            ` + rows + `
            reads sales.order.order_id
        }
    }
}`
}

// TestParseHookRowSource checks that the declared source is read as a query
// reference, so the supplier is the declaration's and not the host's guess.
func TestParseHookRowSource(t *testing.T) {
	got, err := syntax.NewParser(hookWithRows("rows from query order_entry.order_list")).ParseApplicationPattern()
	if err != nil {
		t.Fatal(err)
	}
	if got.Interaction == nil || len(got.Interaction.Hooks) != 1 {
		t.Fatalf("hooks=%#v", got.Interaction)
	}
	declared := got.Interaction.Hooks[0]
	if declared.Rows == nil {
		t.Fatal("the rows source was not read")
	}
	if declared.Rows.Query != "order_entry.order_list" {
		t.Fatalf("query=%q", declared.Rows.Query)
	}
	if declared.Scope != "row" {
		t.Fatalf("scope=%q", declared.Scope)
	}
}

// TestParseHookRowSourceRefusesARelationSource checks that a source kind this
// build supplies nothing for is refused at parse time: a word that parsed but had
// no supplier would be a declaration that can never run.
func TestParseHookRowSourceRefusesARelationSource(t *testing.T) {
	if _, err := syntax.NewParser(hookWithRows("rows from relation sales.order.customer")).ParseApplicationPattern(); err == nil {
		t.Fatal("a relation row source was accepted although no execution supplies it")
	}
}

// TestParseHookRowSourceRefusesABareIdentifier checks that the source kind cannot
// be omitted: a later second kind would otherwise silently change what an
// already-written line meant.
func TestParseHookRowSourceRefusesABareIdentifier(t *testing.T) {
	if _, err := syntax.NewParser(hookWithRows("rows order_entry.order_list")).ParseApplicationPattern(); err == nil {
		t.Fatal("a row source without its kind was accepted")
	}
}

// TestParseHookRowSourceRefusesADeclarationTwice checks that two sources are
// refused rather than letting the last one win silently.
func TestParseHookRowSourceRefusesADeclarationTwice(t *testing.T) {
	source := `pattern order_entry.order_record record {
    data sales.order
    interaction {
        hook before-update {
            scope row
            rows from query order_entry.order_list
            rows from query order_entry.other_list
        }
    }
}`
	if _, err := syntax.NewParser(source).ParseApplicationPattern(); err == nil {
		t.Fatal("two row sources in one hook were accepted")
	}
}
