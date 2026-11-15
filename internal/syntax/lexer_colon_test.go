package syntax

import (
	"strings"
	"testing"
)

// The colon token was added for the rule surface. These cases pin that the
// lexer change cannot alter how an existing Source reads: a colon was previously
// an invalid character everywhere, so no accepted Source could contain one
// outside a string or a raw block.

func TestLexerReadsColonAsItsOwnToken(t *testing.T) {
	lx := NewLexer("rule sales.q: sales.order_line.quantity is positive")
	var tokens []Token
	for {
		tok, err := lx.Next()
		if err != nil {
			t.Fatalf("lex: %v", err)
		}
		if tok.Kind == TokenEOF {
			break
		}
		tokens = append(tokens, tok)
	}
	colonAt := -1
	for i, tok := range tokens {
		if tok.Kind == TokenColon {
			colonAt = i
			if tok.Text != ":" {
				t.Fatalf("colon text=%q", tok.Text)
			}
			// A colon written without surrounding spaces must still be its own
			// token, not glued to the preceding identifier.
			if tokens[i-1].Kind != TokenIdent || tokens[i-1].Text != "sales.q" {
				t.Fatalf("token before colon=%+v", tokens[i-1])
			}
			if tokens[i+1].Kind != TokenIdent || tokens[i+1].Text != "sales.order_line.quantity" {
				t.Fatalf("token after colon=%+v", tokens[i+1])
			}
		}
	}
	if colonAt < 0 {
		t.Fatal("no colon token")
	}
}

// A colon inside a string is part of the string; it is not a separator.
func TestLexerKeepsColonInsideAString(t *testing.T) {
	lx := NewLexer(`description "sha256:abc"`)
	if _, err := lx.Next(); err != nil {
		t.Fatal(err)
	}
	tok, err := lx.Next()
	if err != nil {
		t.Fatal(err)
	}
	if tok.Kind != TokenString || tok.Text != "sha256:abc" {
		t.Fatalf("string token=%+v", tok)
	}
}

// A colon inside a raw block stays in the block: SQL and JSON carry them and
// must not be split.
func TestLexerKeepsColonInsideRawBlock(t *testing.T) {
	src := "sql postgres { SELECT '{\"a\": 1}'::jsonb, x::int FROM t }"
	lx := NewLexer(src)
	var open Token
	for {
		tok, err := lx.Next()
		if err != nil {
			t.Fatalf("lex: %v", err)
		}
		if tok.Kind == TokenLBrace {
			open = tok
			break
		}
	}
	body, _, err := lx.ReadRawBlock()
	if err != nil {
		t.Fatalf("raw block: %v", err)
	}
	if !strings.Contains(body, "::jsonb") || !strings.Contains(body, "x::int") {
		t.Fatalf("raw block lost a colon: %q", body)
	}
	_ = open
}

// An existing declaration that never wrote a colon keeps its exact message: the
// change must not move an error from the parser to the lexer or the reverse.
func TestExistingGrammarWithoutColonIsUnchanged(t *testing.T) {
	src := "application order_entry { unknown }"
	_, err := NewParser(src).ParseApplication()
	if err == nil {
		t.Fatal("expected a parse error")
	}
	// The error must still be a ParseError (not a LexError) at the offending
	// word, which is what the existing tests pin.
	if !isParseError(err) {
		t.Fatalf("error kind changed: %T %v", err, err)
	}
	if _, _, ok := ErrorRange(err); !ok {
		t.Fatalf("no range: %v", err)
	}
}

// A colon in an existing grammar is reported by the parser at the colon, not by
// the lexer. The complaint names the token rather than "unexpected character".
func TestColonInExistingGrammarIsAParseError(t *testing.T) {
	src := "application order_entry { database sales: master }"
	_, err := NewParser(src).ParseApplication()
	if err == nil {
		t.Fatal("expected a parse error")
	}
	if !isParseError(err) {
		t.Fatalf("lexer-level failure for a colon: %T %v", err, err)
	}
	start, end, ok := ErrorRange(err)
	if !ok {
		t.Fatalf("no range: %v", err)
	}
	if got := src[start.Offset:end.Offset]; got != ":" {
		t.Fatalf("range text=%q", got)
	}
}

func isParseError(err error) bool {
	for err != nil {
		if _, ok := err.(*ParseError); ok {
			return true
		}
		if u, ok := err.(interface{ Unwrap() error }); ok {
			err = u.Unwrap()
			continue
		}
		return false
	}
	return false
}
