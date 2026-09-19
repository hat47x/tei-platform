package syntax

import (
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/query"
)

// Anchor is physical source metadata returned alongside a parsed domain value.
// It is intentionally owned by the syntax/authoring boundary rather than by
// application, database, query or other canonical domain packages.
type Anchor struct {
	Start Position
	End   Position
}

type Anchored[T any] struct {
	Value  T
	Anchor Anchor
}

func ParseApplicationAnchored(src string) (Anchored[application.Application], error) {
	value, err := NewParser(src).ParseApplication()
	if err != nil {
		return Anchored[application.Application]{}, err
	}
	anchor, err := DeclarationIdentifierAnchor(src, "application", string(value.ID))
	if err != nil {
		return Anchored[application.Application]{}, err
	}
	return Anchored[application.Application]{Value: value, Anchor: anchor}, nil
}

func ParseDatabaseAnchored(src string) (Anchored[database.Database], error) {
	value, err := NewParser(src).ParseDatabase()
	if err != nil {
		return Anchored[database.Database]{}, err
	}
	anchor, err := DeclarationIdentifierAnchor(src, "database", string(value.ID))
	if err != nil {
		return Anchored[database.Database]{}, err
	}
	return Anchored[database.Database]{Value: value, Anchor: anchor}, nil
}

func ParseQueryAnchored(src string, owner core.CanonicalID) (Anchored[query.EmbeddedSQL], error) {
	value, err := NewParser(src).ParseQuery(owner)
	if err != nil {
		return Anchored[query.EmbeddedSQL]{}, err
	}
	anchor, err := DeclarationIdentifierAnchor(src, "query", string(value.ID))
	if err != nil {
		return Anchored[query.EmbeddedSQL]{}, err
	}
	return Anchored[query.EmbeddedSQL]{Value: value, Anchor: anchor}, nil
}

// ContractQueryDeclarationAnchor returns the exact token range of a query
// name declared inside a top-level `contract <id> <version> { ... }` block.
// Unlike DeclarationIdentifierAnchor, which only accepts a keyword/ID pair at
// the very start of src, a Contract Query name is nested after the contract
// wrapper and any preceding read/write/query members. This walks past those
// members without re-parsing full query signatures (whose grammar has no
// nested braces, so a flat skip cannot be confused by an embedded raw block).
func ContractQueryDeclarationAnchor(src, queryName string) (Anchor, error) {
	lx := NewLexer(src)
	if _, err := expectLexerText(lx, "contract"); err != nil {
		return Anchor{}, err
	}
	if _, err := expectLexerIdent(lx, "contract identifier"); err != nil {
		return Anchor{}, err
	}
	if _, err := expectLexerIdent(lx, "contract version"); err != nil {
		return Anchor{}, err
	}
	if err := expectLexerKind(lx, TokenLBrace, "contract body"); err != nil {
		return Anchor{}, err
	}
	var pending *Token
	for {
		var t Token
		if pending != nil {
			t = *pending
			pending = nil
		} else {
			next, err := lx.Next()
			if err != nil {
				return Anchor{}, err
			}
			t = next
		}
		if t.Kind == TokenRBrace {
			return Anchor{}, fmt.Errorf("contract query %q not found", queryName)
		}
		if t.Kind == TokenEOF {
			return Anchor{}, fmt.Errorf("unterminated contract body")
		}
		if t.Kind != TokenIdent {
			return Anchor{}, fmt.Errorf("expected contract member at %d:%d, got %q", t.Start.Line, t.Start.Column, t.Text)
		}
		switch t.Text {
		case "read", "write":
			if _, err := expectLexerIdent(lx, "contract "+t.Text+" reference"); err != nil {
				return Anchor{}, err
			}
		case "query":
			name, err := expectLexerIdent(lx, "contract query name")
			if err != nil {
				return Anchor{}, err
			}
			next, err := lx.Next()
			if err != nil {
				return Anchor{}, err
			}
			if next.Kind == TokenLBrace {
				if err := skipContractQuerySignatureBody(lx); err != nil {
					return Anchor{}, err
				}
			} else {
				pending = &next
			}
			if name.Text == queryName {
				return tokenAnchor(name), nil
			}
		default:
			return Anchor{}, fmt.Errorf("expected read, write, or query at %d:%d, got %q", t.Start.Line, t.Start.Column, t.Text)
		}
	}
}

// skipContractQuerySignatureBody skips a Contract Query signature body
// (`{ (input <name> type <type> | result <type> <cardinality>)* }`) whose
// grammar, per parseContractQuerySignature, contains no nested braces.
func skipContractQuerySignatureBody(lx *Lexer) error {
	for {
		t, err := lx.Next()
		if err != nil {
			return err
		}
		if t.Kind == TokenRBrace {
			return nil
		}
		if t.Kind == TokenEOF {
			return fmt.Errorf("unterminated contract query signature body")
		}
		if t.Kind != TokenIdent || (t.Text != "input" && t.Text != "result") {
			return fmt.Errorf("expected contract query signature member at %d:%d, got %q", t.Start.Line, t.Start.Column, t.Text)
		}
		if t.Text == "input" {
			if _, err := expectLexerIdent(lx, "contract query input name"); err != nil {
				return err
			}
			if _, err := expectLexerText(lx, "type"); err != nil {
				return err
			}
			if _, err := expectLexerIdent(lx, "contract query input type"); err != nil {
				return err
			}
		} else {
			if _, err := expectLexerIdent(lx, "contract query result type"); err != nil {
				return err
			}
			if _, err := expectLexerIdent(lx, "contract query result cardinality"); err != nil {
				return err
			}
		}
	}
}

// DeclarationIdentifierAnchor returns the exact token range of a top-level
// declaration identifier. It only reads the declaration keyword and ID, so a
// later raw block (for example embedded SQL) cannot confuse source navigation.
func DeclarationIdentifierAnchor(src, keyword, expectedID string) (Anchor, error) {
	lx := NewLexer(src)
	first, err := lx.Next()
	if err != nil {
		return Anchor{}, err
	}
	if first.Kind != TokenIdent || first.Text != keyword {
		return Anchor{}, fmt.Errorf("expected declaration keyword %q, got %q at %d:%d", keyword, first.Text, first.Start.Line, first.Start.Column)
	}
	id, err := lx.Next()
	if err != nil {
		return Anchor{}, err
	}
	if id.Kind != TokenIdent {
		return Anchor{}, fmt.Errorf("expected %s identifier at %d:%d", keyword, id.Start.Line, id.Start.Column)
	}
	if id.Text != expectedID {
		return Anchor{}, fmt.Errorf("%s identifier %q does not match expected subject %q", keyword, id.Text, expectedID)
	}
	return Anchor{Start: id.Start, End: id.End}, nil
}
