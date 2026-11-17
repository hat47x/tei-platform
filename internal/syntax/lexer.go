package syntax

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type TokenKind int

const (
	TokenEOF TokenKind = iota
	TokenIdent
	TokenString
	TokenLBrace
	TokenRBrace
	TokenArrow
	TokenCaret
	// TokenColon is the clause separator of the rule authoring syntax
	// (tei.rule-authoring/v0). It was previously an invalid character, so no
	// accepted Source changes meaning; a Source that used ':' now stops at the
	// declaration that does not accept the colon instead of at the lexer.
	TokenColon
)

type Position struct {
	Offset int
	Line   int
	Column int
}

type Token struct {
	Kind  TokenKind
	Text  string
	Start Position
	End   Position
}

type Lexer struct {
	src    string
	off    int
	line   int
	column int
}

// NewLexer ignores one leading UTF-8 BOM as a prefix that is not part of the
// source text. Byte offsets stay anchored to src so callers can still slice or
// splice by Position.Offset; only the first visible character starts at
// column 1. A BOM anywhere else remains an invalid character.
func NewLexer(src string) *Lexer {
	l := &Lexer{src: src, line: 1, column: 1}
	if strings.HasPrefix(src, "\ufeff") {
		l.off = len("\ufeff")
	}
	return l
}

func (l *Lexer) Source() string { return l.src }
func (l *Lexer) Offset() int    { return l.off }

func (l *Lexer) Next() (Token, error) {
	l.skipSpaceAndComments()
	start := l.pos()
	if l.off >= len(l.src) {
		return Token{Kind: TokenEOF, Start: start, End: start}, nil
	}
	switch l.src[l.off] {
	case '{':
		l.advanceByte()
		return Token{Kind: TokenLBrace, Text: "{", Start: start, End: l.pos()}, nil
	case '}':
		l.advanceByte()
		return Token{Kind: TokenRBrace, Text: "}", Start: start, End: l.pos()}, nil
	case '^':
		l.advanceByte()
		return Token{Kind: TokenCaret, Text: "^", Start: start, End: l.pos()}, nil
	case ':':
		l.advanceByte()
		return Token{Kind: TokenColon, Text: ":", Start: start, End: l.pos()}, nil
	case '-':
		if l.off+1 < len(l.src) && l.src[l.off+1] == '>' {
			l.advanceByte()
			l.advanceByte()
			return Token{Kind: TokenArrow, Text: "->", Start: start, End: l.pos()}, nil
		}
	case '"':
		return l.scanString()
	}
	if isIdentByte(l.src[l.off]) {
		begin := l.off
		for l.off < len(l.src) && isIdentByte(l.src[l.off]) {
			l.advanceByte()
		}
		return Token{Kind: TokenIdent, Text: l.src[begin:l.off], Start: start, End: l.pos()}, nil
	}
	end := start
	end.Offset++
	end.Column++
	return Token{}, &LexError{Start: start, End: end, Msg: fmt.Sprintf("unexpected character %q", l.src[l.off])}
}

func (l *Lexer) ReadRawBlock() (string, Position, error) {
	// Called immediately after the opening '{' token has been consumed.
	start := l.pos()
	begin := l.off
	depth := 1
	var quote byte
	escaped := false
	lineComment := false
	blockComment := false
	for l.off < len(l.src) {
		c := l.src[l.off]
		next := byte(0)
		if l.off+1 < len(l.src) {
			next = l.src[l.off+1]
		}
		if lineComment {
			l.advanceByte()
			if c == '\n' {
				lineComment = false
			}
			continue
		}
		if blockComment {
			l.advanceByte()
			if c == '*' && next == '/' {
				l.advanceByte()
				blockComment = false
			}
			continue
		}
		if quote != 0 {
			l.advanceByte()
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '-' && next == '-' {
			l.advanceByte()
			l.advanceByte()
			lineComment = true
			continue
		}
		if c == '/' && next == '*' {
			l.advanceByte()
			l.advanceByte()
			blockComment = true
			continue
		}
		if c == '\'' || c == '"' || c == '`' {
			quote = c
			l.advanceByte()
			continue
		}
		switch c {
		case '{':
			depth++
			l.advanceByte()
		case '}':
			depth--
			if depth == 0 {
				body := l.src[begin:l.off]
				l.advanceByte()
				return body, start, nil
			}
			l.advanceByte()
		default:
			l.advanceByte()
		}
	}
	return "", start, &LexError{Start: start, End: l.pos(), Msg: "unterminated raw block"}
}

func (l *Lexer) scanString() (Token, error) {
	start := l.pos()
	l.advanceByte() // opening quote
	begin := l.off
	escaped := false
	for l.off < len(l.src) {
		c := l.src[l.off]
		if escaped {
			escaped = false
			l.advanceByte()
			continue
		}
		if c == '\\' {
			escaped = true
			l.advanceByte()
			continue
		}
		if c == '"' {
			text := l.src[begin:l.off]
			l.advanceByte()
			return Token{Kind: TokenString, Text: text, Start: start, End: l.pos()}, nil
		}
		l.advanceByte()
	}
	return Token{}, &LexError{Start: start, End: l.pos(), Msg: "unterminated string"}
}

func (l *Lexer) skipSpaceAndComments() {
	for l.off < len(l.src) {
		if unicode.IsSpace(rune(l.src[l.off])) {
			l.advanceByte()
			continue
		}
		if l.src[l.off] == '/' && l.off+1 < len(l.src) && l.src[l.off+1] == '/' {
			for l.off < len(l.src) && l.src[l.off] != '\n' {
				l.advanceByte()
			}
			continue
		}
		if l.src[l.off] == '#' {
			for l.off < len(l.src) && l.src[l.off] != '\n' {
				l.advanceByte()
			}
			continue
		}
		break
	}
}

// TryConsumeDecimalWidth reads an optional "(precision,scale)" that directly
// follows the last token the parser consumed, and reports whether one was
// there. Parentheses are deliberately NOT tokens: a Source that never writes a
// width keeps exactly today's token sequence, and the width is a suffix of
// one storage declaration rather than a new declaration form. A Source that
// writes something else after the type is left alone, so the caller can report
// its own error instead of this one.
func (l *Lexer) TryConsumeDecimalWidth() (precision, scale int, ok bool, err error) {
	l.skipSpaceAndComments()
	if l.off >= len(l.src) || l.src[l.off] != '(' {
		return 0, 0, false, nil
	}
	start := l.pos()
	l.advanceByte()
	precision, err = l.readDecimalPart()
	if err != nil {
		return 0, 0, false, err
	}
	if l.off >= len(l.src) || l.src[l.off] != ',' {
		return 0, 0, false, l.errAtPosition(start, `decimal width must be "(precision,scale)"`)
	}
	l.advanceByte()
	scale, err = l.readDecimalPart()
	if err != nil {
		return 0, 0, false, err
	}
	if l.off >= len(l.src) || l.src[l.off] != ')' {
		return 0, 0, false, l.errAtPosition(start, `decimal width must be "(precision,scale)"`)
	}
	l.advanceByte()
	return precision, scale, true, nil
}

// readDecimalPart reads one non-empty run of ASCII digits.
func (l *Lexer) readDecimalPart() (int, error) {
	begin := l.off
	for l.off < len(l.src) && l.src[l.off] >= '0' && l.src[l.off] <= '9' {
		l.advanceByte()
	}
	if l.off == begin {
		return 0, l.errAtPosition(l.pos(), "decimal width parts must be digits")
	}
	value, err := strconv.Atoi(l.src[begin:l.off])
	if err != nil {
		// Too many digits for an int: report the declaration as out of range
		// rather than leaking a strconv message.
		return 0, l.errAtPosition(l.pos(), "decimal width part is too large")
	}
	return value, nil
}

// errAtPosition reports an error at a position the lexer already recorded.
func (l *Lexer) errAtPosition(at Position, msg string) error {
	end := at
	end.Column++
	end.Offset++
	return &LexError{Start: at, End: end, Msg: msg}
}

// PeekOpenParen reports whether the next character, after whitespace and
// comments, is an opening parenthesis. The parser uses it to say that a width
// belongs only to a decimal, rather than reporting a generic suffix error.
func (l *Lexer) PeekOpenParen() bool {
	l.skipSpaceAndComments()
	return l.off < len(l.src) && l.src[l.off] == '('
}

func (l *Lexer) pos() Position { return Position{Offset: l.off, Line: l.line, Column: l.column} }
func (l *Lexer) advanceByte() {
	if l.off >= len(l.src) {
		return
	}
	if l.src[l.off] == '\n' {
		l.line++
		l.column = 1
	} else {
		l.column++
	}
	l.off++
}
func isIdentByte(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || b == '_' || b == '-' || b == '.' || b == '/' || b == '*'
}
