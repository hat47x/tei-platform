// Package multilingualformat is an E2a cross-target formatting probe.
// It intentionally does not parse or implement Unicode MessageFormat 2.
package multilingualformat

import (
	"fmt"

	"golang.org/x/text/feature/plural"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// MaxCount is restricted to values exactly representable by both 32-bit Go
// integer arguments and the JavaScript number type.
const MaxCount int64 = 2147483647

// Result contains only the independently observable, localized operations.
// A caller must not treat it as a formatted MessageFormat 2 message.
type Result struct {
	Category string `json:"category"`
	Number   string `json:"number"`
}

var supported = map[string]bool{
	"en-US": true,
	"ja-JP": true,
	"de-DE": true,
}

func category(form plural.Form) string {
	switch form {
	case plural.Zero:
		return "zero"
	case plural.One:
		return "one"
	case plural.Two:
		return "two"
	case plural.Few:
		return "few"
	case plural.Many:
		return "many"
	default:
		return "other"
	}
}

// Probe compares Go's pinned x/text plural and integer formatting facilities
// with the matching JavaScript Intl operations in probe.mjs. Locale coverage
// is explicit: this is not a universal language fallback or translator.
func Probe(locale string, count int64) (Result, error) {
	if !supported[locale] {
		return Result{}, fmt.Errorf("unsupported-locale: %q", locale)
	}
	if count < 0 || count > MaxCount {
		return Result{}, fmt.Errorf("invalid-count: %d", count)
	}
	tag, err := language.Parse(locale)
	if err != nil || tag.String() != locale {
		return Result{}, fmt.Errorf("invalid-locale: %q", locale)
	}
	form := plural.Cardinal.MatchPlural(tag, int(count), 0, 0, 0, 0)
	return Result{
		Category: category(form),
		Number: message.NewPrinter(tag).Sprintf("%d", count),
	}, nil
}
