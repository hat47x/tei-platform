// Package diag holds the diagnostic record that code generators return
// (#5771). One Record says what went wrong, where, and who can fix it. Its
// JSON form is described by tei.codegen.diagnostic/draft.1, which
// internal/codegen/record embeds.
//
// The package defines the record only. It assigns no codes: each generator
// keeps its own and documents them. Other diagnostic types in the repository
// are not rewritten to this one; diag/fromworkspace converts the two that
// carry source positions.
package diag

import (
	"errors"
	"fmt"
	"regexp"

	"tei/internal/core"
)

// Severity values.
const (
	SeverityError   = "error"
	SeverityWarning = "warning"
	SeverityInfo    = "info"
)

// Fixer values name who can remove the cause of a diagnostic.
const (
	// FixerDesigner changes the design: the rule, its attribute or its type.
	FixerDesigner = "designer"
	// FixerDeveloper changes what the developer controls in the generated
	// code, such as the package name or the Go names it is given.
	FixerDeveloper = "developer"
	// FixerProvider changes the generator itself.
	FixerProvider = "provider"
)

// Record is one diagnostic.
type Record struct {
	// Code is an upper-case identifier, optionally behind a lower-case
	// namespace and a slash: NAME_COLLISION, goemit/NAME_COLLISION.
	Code     string `json:"code"`
	Severity string `json:"severity"`
	// File is the path of the file the diagnostic points into, when there
	// is one. It is the path the producer wrote or read, with '/' separators.
	File string `json:"file,omitempty"`
	// Range is a span of File in UTF-8 byte offsets.
	Range *Range `json:"range,omitempty"`
	// Line and Column are 1-based and give the start of the span for people.
	// Column counts bytes, like the offsets. Zero means unknown.
	Line   int `json:"line,omitempty"`
	Column int `json:"column,omitempty"`
	// Element is the design element the diagnostic is about.
	Element core.CanonicalID `json:"element,omitempty"`
	Message string           `json:"message"`
	// Related are other places that take part, such as the second
	// declaration of a duplicate.
	Related []Related `json:"related,omitempty"`
	// Fixer is one of the Fixer values, or empty when unknown.
	Fixer string `json:"fixer,omitempty"`
	// Provider names the generator or plugin that produced the record.
	Provider string `json:"provider,omitempty"`
}

// Range is a half-open span [Start, End) of UTF-8 byte offsets.
type Range struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// Related is another place a diagnostic refers to. Role says how it takes
// part, for example "definition" or "conflict".
type Related struct {
	File    string           `json:"file,omitempty"`
	Range   *Range           `json:"range,omitempty"`
	Element core.CanonicalID `json:"element,omitempty"`
	Role    string           `json:"role"`
}

// CodePattern is the form of Record.Code.
var CodePattern = regexp.MustCompile(`^([a-z0-9]+(\.[a-z0-9-]+)*/)?[A-Z][A-Z0-9_]*$`)

// Validate reports the first way r breaks the record's rules, or nil.
func (r Record) Validate() error {
	if !CodePattern.MatchString(r.Code) {
		return fmt.Errorf("diagnostic code %q does not match %s", r.Code, CodePattern)
	}
	switch r.Severity {
	case SeverityError, SeverityWarning, SeverityInfo:
	default:
		return fmt.Errorf("diagnostic %s: severity %q is not error, warning or info", r.Code, r.Severity)
	}
	switch r.Fixer {
	case "", FixerDesigner, FixerDeveloper, FixerProvider:
	default:
		return fmt.Errorf("diagnostic %s: fixer %q is not designer, developer or provider", r.Code, r.Fixer)
	}
	if err := r.Range.validate(); err != nil {
		return fmt.Errorf("diagnostic %s: %w", r.Code, err)
	}
	if r.Line < 0 || r.Column < 0 {
		return fmt.Errorf("diagnostic %s: line %d and column %d must not be negative", r.Code, r.Line, r.Column)
	}
	if r.Message == "" {
		return fmt.Errorf("diagnostic %s: message is empty", r.Code)
	}
	for i, related := range r.Related {
		if related.Role == "" {
			return fmt.Errorf("diagnostic %s: related[%d] has no role", r.Code, i)
		}
		if err := related.Range.validate(); err != nil {
			return fmt.Errorf("diagnostic %s: related[%d]: %w", r.Code, i, err)
		}
	}
	return nil
}

func (r *Range) validate() error {
	if r == nil {
		return nil
	}
	if r.Start < 0 || r.End < r.Start {
		return errors.New("range must satisfy 0 <= start <= end")
	}
	return nil
}
