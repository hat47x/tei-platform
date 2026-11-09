package syntax

import (
	"fmt"
	"strings"

	"tei/internal/core"
	"tei/internal/semantic"
)

type SubjectAnchor struct {
	Kind   string
	ID     string
	Anchor Anchor
}

type SemanticAnchored struct {
	Value   semantic.Namespace
	Anchors []SubjectAnchor
}

// ParseSemanticFileAnchored parses the canonical semantic value with the
// existing parser, then builds a source-side anchor map from the same syntax.
// Physical positions are kept out of semantic.Namespace and its members.
func ParseSemanticFileAnchored(src string) (SemanticAnchored, error) {
	value, err := NewParser(src).ParseSemanticFile()
	if err != nil {
		return SemanticAnchored{}, err
	}
	anchors, err := semanticSubjectAnchors(src, value.ID)
	if err != nil {
		return SemanticAnchored{}, err
	}
	return SemanticAnchored{Value: value, Anchors: anchors}, nil
}

func semanticSubjectAnchors(src string, namespace core.CanonicalID) ([]SubjectAnchor, error) {
	lx := NewLexer(src)
	// Skip the optional semantic format header so the anchor scan starts at the
	// same declaration as the value parser.
	if first, err := peekLexer(lx); err != nil {
		return nil, err
	} else if first.Kind == TokenIdent && first.Text == "formatVersion" {
		if _, err := lx.Next(); err != nil {
			return nil, err
		}
		if _, err := expectLexerIdent(lx, "semantic format version"); err != nil {
			return nil, err
		}
	}
	if _, err := expectLexerText(lx, "namespace"); err != nil {
		return nil, err
	}
	nsTok, err := expectLexerIdent(lx, "semantic namespace")
	if err != nil {
		return nil, err
	}
	if nsTok.Text != string(namespace) {
		return nil, fmt.Errorf("semantic namespace %q does not match parsed namespace %q", nsTok.Text, namespace)
	}
	anchors := []SubjectAnchor{{Kind: "semantic", ID: string(namespace), Anchor: tokenAnchor(nsTok)}}

	for {
		t, err := lx.Next()
		if err != nil {
			return nil, err
		}
		if t.Kind == TokenEOF {
			return anchors, nil
		}
		if t.Kind != TokenIdent {
			return nil, fmt.Errorf("expected semantic declaration at %d:%d, got %q", t.Start.Line, t.Start.Column, t.Text)
		}
		switch t.Text {
		case "type":
			sym, err := expectLexerIdent(lx, "semantic type symbol")
			if err != nil {
				return nil, err
			}
			id, err := semanticChildID(namespace, sym.Text)
			if err != nil {
				return nil, err
			}
			anchors = append(anchors, SubjectAnchor{Kind: "semantic-type", ID: string(id), Anchor: tokenAnchor(sym)})
			if _, err := expectLexerText(lx, "from"); err != nil {
				return nil, err
			}
			if _, err := expectLexerIdent(lx, "semantic type underlying type"); err != nil {
				return nil, err
			}
			// A value-restricted alias ends with `values { "a" "b" }`.
			// The anchor scan only needs the subject identity, so skip the
			// restriction instead of failing the whole Source Index build
			// (the parser reads it; the anchor scanner must not stop here).
			if err := skipSemanticTypeValues(lx); err != nil {
				return nil, err
			}
		case "data":
			sym, err := expectLexerIdent(lx, "semantic data symbol")
			if err != nil {
				return nil, err
			}
			id, err := semanticChildID(namespace, sym.Text)
			if err != nil {
				return nil, err
			}
			anchors = append(anchors, SubjectAnchor{Kind: "semantic-data", ID: string(id), Anchor: tokenAnchor(sym)})
			if err := expectLexerKind(lx, TokenLBrace, "semantic data body"); err != nil {
				return nil, err
			}
			members, err := semanticDataAnchors(lx, id)
			if err != nil {
				return nil, err
			}
			anchors = append(anchors, members...)
		case "capability":
			sym, err := expectLexerIdent(lx, "semantic capability symbol")
			if err != nil {
				return nil, err
			}
			id, err := semanticCapabilityID(namespace, sym.Text)
			if err != nil {
				return nil, err
			}
			anchors = append(anchors, SubjectAnchor{Kind: "semantic-capability", ID: string(id), Anchor: tokenAnchor(sym)})
			if err := expectLexerKind(lx, TokenLBrace, "semantic capability body"); err != nil {
				return nil, err
			}
			if err := skipSemanticCapabilityBody(lx); err != nil {
				return nil, err
			}
		case "validation":
			sym, err := expectLexerIdent(lx, "semantic validation identity")
			if err != nil {
				return nil, err
			}
			id, err := semanticValidationID(namespace, sym.Text)
			if err != nil {
				return nil, err
			}
			anchors = append(anchors, SubjectAnchor{Kind: "semantic-validation", ID: string(id), Anchor: tokenAnchor(sym)})
			if _, err := expectLexerIdent(lx, "semantic validation kind"); err != nil {
				return nil, err
			}
			if err := expectLexerKind(lx, TokenLBrace, "semantic validation body"); err != nil {
				return nil, err
			}
			if err := skipSemanticValidationBody(lx); err != nil {
				return nil, err
			}
		case "period":
			sym, err := expectLexerIdent(lx, "semantic period identity")
			if err != nil {
				return nil, err
			}
			id, err := semanticPeriodID(namespace, sym.Text)
			if err != nil {
				return nil, err
			}
			anchors = append(anchors, SubjectAnchor{Kind: "semantic-period", ID: string(id), Anchor: tokenAnchor(sym)})
			if err := expectLexerKind(lx, TokenLBrace, "semantic period body"); err != nil {
				return nil, err
			}
			if err := skipSemanticPeriodBody(lx); err != nil {
				return nil, err
			}
		case "calendar":
			sym, err := expectLexerIdent(lx, "semantic calendar identity")
			if err != nil {
				return nil, err
			}
			id, err := semanticPeriodID(namespace, sym.Text)
			if err != nil {
				return nil, err
			}
			anchors = append(anchors, SubjectAnchor{Kind: "semantic-calendar", ID: string(id), Anchor: tokenAnchor(sym)})
			if err := expectLexerKind(lx, TokenLBrace, "semantic calendar body"); err != nil {
				return nil, err
			}
			if err := skipSemanticPeriodBody(lx); err != nil {
				return nil, err
			}
		case "effect-rule":
			sym, err := expectLexerIdent(lx, "semantic effect-rule identity")
			if err != nil {
				return nil, err
			}
			id, err := semanticPeriodID(namespace, sym.Text)
			if err != nil {
				return nil, err
			}
			anchors = append(anchors, SubjectAnchor{Kind: "semantic-effect-rule", ID: string(id), Anchor: tokenAnchor(sym)})
			if err := expectLexerKind(lx, TokenLBrace, "semantic effect-rule body"); err != nil {
				return nil, err
			}
			if err := skipSemanticPeriodBody(lx); err != nil {
				return nil, err
			}
		case "run-plan":
			sym, err := expectLexerIdent(lx, "semantic run-plan identity")
			if err != nil {
				return nil, err
			}
			id, err := semanticPeriodID(namespace, sym.Text)
			if err != nil {
				return nil, err
			}
			anchors = append(anchors, SubjectAnchor{Kind: "semantic-run-plan", ID: string(id), Anchor: tokenAnchor(sym)})
			if err := expectLexerKind(lx, TokenLBrace, "semantic run-plan body"); err != nil {
				return nil, err
			}
			if err := skipSemanticPeriodBody(lx); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unexpected semantic declaration %q at %d:%d", t.Text, t.Start.Line, t.Start.Column)
		}
	}
}

// skipSemanticTypeValues consumes an optional `values { ... }` restriction
// after a type's underlying type. It reads nothing else: the next token
// belongs to the enclosing namespace or body.
func skipSemanticTypeValues(lx *Lexer) error {
	t, err := peekLexer(lx)
	if err != nil {
		return err
	}
	if t.Kind != TokenIdent || t.Text != "values" {
		return nil
	}
	if _, err := lx.Next(); err != nil {
		return err
	}
	if err := expectLexerKind(lx, TokenLBrace, "semantic type values body"); err != nil {
		return err
	}
	for {
		t, err := lx.Next()
		if err != nil {
			return err
		}
		switch t.Kind {
		case TokenRBrace:
			return nil
		case TokenString:
			// one declared value
		default:
			return fmt.Errorf("unexpected token %q at %d:%d in semantic type values", t.Text, t.Start.Line, t.Start.Column)
		}
	}
}

func semanticDataAnchors(lx *Lexer, dataID core.CanonicalID) ([]SubjectAnchor, error) {
	var anchors []SubjectAnchor
	var pending *Token
	for {
		var t Token
		if pending != nil {
			t = *pending
			pending = nil
		} else {
			next, err := lx.Next()
			if err != nil {
				return nil, err
			}
			t = next
		}
		if t.Kind == TokenRBrace {
			return anchors, nil
		}
		if t.Kind == TokenEOF {
			return nil, fmt.Errorf("unterminated semantic data body")
		}
		if t.Kind != TokenIdent {
			return nil, fmt.Errorf("expected semantic data member at %d:%d, got %q", t.Start.Line, t.Start.Column, t.Text)
		}
		switch t.Text {
		case "attribute":
			sym, err := expectLexerIdent(lx, "semantic attribute symbol")
			if err != nil {
				return nil, err
			}
			id, err := semanticChildID(dataID, sym.Text)
			if err != nil {
				return nil, err
			}
			anchors = append(anchors, SubjectAnchor{Kind: "semantic-attribute", ID: string(id), Anchor: tokenAnchor(sym)})
			if _, err := expectLexerText(lx, "type"); err != nil {
				return nil, err
			}
			typeTok, err := expectLexerIdent(lx, "semantic attribute type")
			if err != nil {
				return nil, err
			}
			// The type reference token is anchored under its own kind so a typed edit
			// can replace exactly this token. Kind "semantic-attribute" stays the symbol.
			anchors = append(anchors, SubjectAnchor{Kind: "semantic-attribute-type", ID: string(id), Anchor: tokenAnchor(typeTok)})
			optionAnchors, err := semanticAttributeOptionAnchors(lx, id)
			if err != nil {
				return nil, err
			}
			anchors = append(anchors, optionAnchors...)
		case "relation":
			sym, err := expectLexerIdent(lx, "semantic relation symbol")
			if err != nil {
				return nil, err
			}
			id, err := semanticChildID(dataID, sym.Text)
			if err != nil {
				return nil, err
			}
			anchors = append(anchors, SubjectAnchor{Kind: "semantic-relation", ID: string(id), Anchor: tokenAnchor(sym)})
			if _, err := expectLexerText(lx, "to"); err != nil {
				return nil, err
			}
			if _, err := expectLexerIdent(lx, "semantic relation target"); err != nil {
				return nil, err
			}
			if _, err := expectLexerIdent(lx, "semantic relation cardinality"); err != nil {
				return nil, err
			}
			modifier, err := peekLexer(lx)
			if err != nil {
				return nil, err
			}
			if modifier.Kind == TokenIdent && modifier.Text == "on-delete" {
				_, _ = lx.Next()
				if _, err := expectLexerIdent(lx, "semantic relation on-delete policy"); err != nil {
					return nil, err
				}
			}
			next, err := skipSemanticRelationKeys(lx)
			if err != nil {
				return nil, err
			}
			pending = &next
		case "candidate-key", "functional-dependency":
			sym, err := expectLexerIdent(lx, "semantic business rule symbol")
			if err != nil {
				return nil, err
			}
			id, err := semanticChildID(dataID, sym.Text)
			if err != nil {
				return nil, err
			}
			kind := "semantic-" + t.Text
			anchors = append(anchors, SubjectAnchor{Kind: kind, ID: string(id), Anchor: tokenAnchor(sym)})
			if err := expectLexerKind(lx, TokenLBrace, "semantic business rule block"); err != nil {
				return nil, err
			}
			if err := skipSemanticBusinessRuleBody(lx, t.Text); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unexpected semantic data member %q at %d:%d", t.Text, t.Start.Line, t.Start.Column)
		}
	}
}

// skipSemanticRelationKeysは、relation宣言直後に続く任意個のkey対応（構文はparseSemanticRelationKeysと同じ）を
// Subject Anchorを増やさずに読み飛ばす。key対応はrelation自身の複合キーを表す修飾であり、
// 新しいanchor付きsubjectを導入しないため、attribute/relation/capability/validationと同じ扱いにしない。
// 呼び出し側はこの関数が返す次のsemantic data memberトークンをそのまま処理へ回す。
func skipSemanticRelationKeys(lx *Lexer) (Token, error) {
	for {
		t, err := lx.Next()
		if err != nil {
			return Token{}, err
		}
		if t.Kind != TokenIdent || t.Text != "key" {
			return t, nil
		}
		if _, err := expectLexerIdent(lx, "semantic relation key source attribute"); err != nil {
			return Token{}, err
		}
		if err := expectLexerKind(lx, TokenArrow, "semantic relation key arrow"); err != nil {
			return Token{}, err
		}
		if _, err := expectLexerIdent(lx, "semantic relation key target attribute"); err != nil {
			return Token{}, err
		}
	}
}

func skipSemanticCapabilityBody(lx *Lexer) error {
	for {
		t, err := lx.Next()
		if err != nil {
			return err
		}
		if t.Kind == TokenRBrace {
			return nil
		}
		if t.Kind == TokenEOF {
			return fmt.Errorf("unterminated semantic capability body")
		}
		if t.Kind != TokenIdent {
			return fmt.Errorf("expected semantic capability member at %d:%d, got %q", t.Start.Line, t.Start.Column, t.Text)
		}
		switch t.Text {
		case "input":
			if _, err := expectLexerIdent(lx, "semantic capability input name"); err != nil {
				return err
			}
			if _, err := expectLexerIdent(lx, "semantic capability input type"); err != nil {
				return err
			}
		case "output":
			if _, err := expectLexerIdent(lx, "semantic capability output type"); err != nil {
				return err
			}
		case "outcome":
			if _, err := expectLexerIdent(lx, "semantic capability outcome name"); err != nil {
				return err
			}
			if _, err := expectLexerIdent(lx, "semantic capability outcome payload type"); err != nil {
				return err
			}
		case "effect":
			if _, err := expectLexerIdent(lx, "semantic capability effect kind"); err != nil {
				return err
			}
			if _, err := expectLexerIdent(lx, "semantic capability effect data type"); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unexpected semantic capability member %q at %d:%d", t.Text, t.Start.Line, t.Start.Column)
		}
	}
}

func skipSemanticValidationBody(lx *Lexer) error {
	for {
		t, err := lx.Next()
		if err != nil {
			return err
		}
		if t.Kind == TokenRBrace {
			return nil
		}
		if t.Kind == TokenEOF {
			return fmt.Errorf("unterminated semantic validation body")
		}
		if t.Kind != TokenIdent || (t.Text != "attribute" && t.Text != "relation") {
			return fmt.Errorf("unexpected semantic validation member %q at %d:%d", t.Text, t.Start.Line, t.Start.Column)
		}
		if _, err := expectLexerIdent(lx, "semantic validation reference"); err != nil {
			return err
		}
	}
}

func semanticValidationID(namespace core.CanonicalID, symbol string) (core.CanonicalID, error) {
	if strings.Contains(symbol, ".") {
		id, err := core.ParseCanonicalID(symbol)
		if err != nil {
			return "", err
		}
		if !strings.HasPrefix(string(id), string(namespace)+".") {
			return "", fmt.Errorf("semantic validation %s does not belong to namespace %s", id, namespace)
		}
		return id, nil
	}
	return semanticChildID(namespace, symbol)
}

func semanticChildID(parent core.CanonicalID, symbol string) (core.CanonicalID, error) {
	segment, err := core.SymbolToSegment(symbol)
	if err != nil {
		return "", err
	}
	return core.ParseCanonicalID(string(parent) + "." + segment)
}

func semanticCapabilityID(namespace core.CanonicalID, symbol string) (core.CanonicalID, error) {
	if strings.Contains(symbol, ".") {
		id, err := core.ParseCanonicalID(symbol)
		if err != nil {
			return "", err
		}
		if !strings.HasPrefix(string(id), string(namespace)+".") {
			return "", fmt.Errorf("semantic capability %s does not belong to namespace %s", id, namespace)
		}
		return id, nil
	}
	return semanticChildID(namespace, symbol)
}

func tokenAnchor(t Token) Anchor {
	return Anchor{Start: t.Start, End: t.End}
}

func peekLexer(lx *Lexer) (Token, error) {
	off, line, column := lx.off, lx.line, lx.column
	t, err := lx.Next()
	lx.off, lx.line, lx.column = off, line, column
	return t, err
}

func expectLexerText(lx *Lexer, text string) (Token, error) {
	t, err := lx.Next()
	if err != nil {
		return Token{}, err
	}
	if t.Kind != TokenIdent || t.Text != text {
		return Token{}, fmt.Errorf("expected %q at %d:%d, got %q", text, t.Start.Line, t.Start.Column, t.Text)
	}
	return t, nil
}

// semanticAttributeOptionAnchors consumes attribute modifiers while retaining
// exact Source positions for declared Role values. These source-side anchors
// do not grant edit/meaning/Host authority; Writer safety remains separate.
func semanticAttributeOptionAnchors(lx *Lexer, id core.CanonicalID) ([]SubjectAnchor, error) {
	var anchors []SubjectAnchor
	for {
		next, err := peekLexer(lx)
		if err != nil {
			return nil, err
		}
		if next.Kind != TokenIdent {
			return anchors, nil
		}
		switch next.Text {
		case "description", "display", "value":
			_, _ = lx.Next()
			if err := expectLexerKind(lx, TokenString, "semantic attribute "+next.Text); err != nil {
				return nil, err
			}
		case "required", "immutable":
			_, _ = lx.Next()
		case "role":
			_, _ = lx.Next()
			roleTok, err := expectLexerIdent(lx, "semantic attribute role")
			if err != nil {
				return nil, err
			}
			anchors = append(anchors, SubjectAnchor{Kind: "semantic-attribute-role", ID: string(id), Anchor: tokenAnchor(roleTok)})
		default:
			return anchors, nil
		}
	}
}

func expectLexerIdent(lx *Lexer, description string) (Token, error) {
	t, err := lx.Next()
	if err != nil {
		return Token{}, err
	}
	if t.Kind != TokenIdent {
		return Token{}, fmt.Errorf("expected %s at %d:%d, got %q", description, t.Start.Line, t.Start.Column, t.Text)
	}
	return t, nil
}

func expectLexerKind(lx *Lexer, kind TokenKind, description string) error {
	t, err := lx.Next()
	if err != nil {
		return err
	}
	if t.Kind != kind {
		return fmt.Errorf("expected %s at %d:%d, got %q", description, t.Start.Line, t.Start.Column, t.Text)
	}
	return nil
}

// semanticPeriodID resolves a period identity the same way capabilities do.
func semanticPeriodID(namespace core.CanonicalID, symbol string) (core.CanonicalID, error) {
	if strings.Contains(symbol, ".") {
		id, err := core.ParseCanonicalID(symbol)
		if err != nil {
			return "", err
		}
		if !strings.HasPrefix(string(id), string(namespace)+".") {
			return "", fmt.Errorf("period canonical id must belong to the declared semantic namespace")
		}
		return id, nil
	}
	seg, err := core.SymbolToSegment(symbol)
	if err != nil {
		return "", err
	}
	return core.ParseCanonicalID(string(namespace) + "." + seg)
}

// skipSemanticPeriodBody skips the period body. Members are validated by the
// parser; the anchor pass only needs the identity that follows the keyword.
func skipSemanticPeriodBody(lx *Lexer) error {
	for {
		t, err := lx.Next()
		if err != nil {
			return err
		}
		if t.Kind == TokenRBrace {
			return nil
		}
		if t.Kind == TokenEOF {
			return fmt.Errorf("unterminated semantic period body")
		}
	}
}
