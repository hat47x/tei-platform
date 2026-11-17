package numberingpostgres

import (
	"testing"
)

// TestQuoteIdentifierRefusesADoubleQuote checks the refusal the comment promises.
//
// The table name comes from host configuration. Escaping a double quote would make
// the identifier depend on quoting rules rather than on the name the host wrote, so
// the name is refused instead.
func TestQuoteIdentifierRefusesADoubleQuote(t *testing.T) {
	if got := quoteIdentifier(`bad"name`); got != "" {
		t.Fatalf("a name with a double quote was quoted anyway: %q", got)
	}
	if got := quoteIdentifier("plain"); got != `"plain"` {
		t.Fatalf("a plain name was not quoted: %q", got)
	}
}
