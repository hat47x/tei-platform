package syntax

import (
	"errors"
	"fmt"
)

// LexError is a source-located lexer error. Start and End use byte offsets and
// the same 1-based line/column convention as Token positions.
type LexError struct {
	Start Position
	End   Position
	Msg   string
}

func (e *LexError) Error() string {
	return fmt.Sprintf("%d:%d: %s", e.Start.Line, e.Start.Column, e.Msg)
}

// ErrorRange returns the source range carried by a syntax error. Parse errors
// created before End was populated are treated as point ranges for compatibility.
func ErrorRange(err error) (Position, Position, bool) {
	var parseErr *ParseError
	if errors.As(err, &parseErr) {
		end := parseErr.End
		if end.Line == 0 || end.Column == 0 {
			end = parseErr.Pos
		}
		return parseErr.Pos, end, true
	}
	var lexErr *LexError
	if errors.As(err, &lexErr) {
		return lexErr.Start, lexErr.End, true
	}
	return Position{}, Position{}, false
}

// ErrorMessage returns the location-free message of a source-located syntax
// error. Callers can render the position independently from the message.
func ErrorMessage(err error) (string, bool) {
	var parseErr *ParseError
	if errors.As(err, &parseErr) {
		return parseErr.Msg, true
	}
	var lexErr *LexError
	if errors.As(err, &lexErr) {
		return lexErr.Msg, true
	}
	return "", false
}
