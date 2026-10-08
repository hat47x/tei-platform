// Package languagereference is an E1 reference probe, not TEI's production
// language registry. It checks that an already accepted language selection
// chooses exactly one provider independent of registration order.
package languagereference

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidSelection = errors.New("language selection is incomplete")
	ErrProviderMissing = errors.New("locked language provider is unavailable")
	ErrProviderAmbiguous = errors.New("locked language provider is ambiguous")
)

// Selection is an exact, previously accepted choice. The language contract is
// distinct from the implementation supplying the parser. Dependency solving,
// acceptance, and build provenance are outside this reference.
type Selection struct {
	LanguageID                 string
	ContractVersion            string
	ContractFingerprint        string
	ImplementationID           string
	ImplementationFingerprint string
}

func (s Selection) valid() bool {
	return s.LanguageID != "" && s.ContractVersion != "" &&
		s.ContractFingerprint != "" && s.ImplementationID != "" &&
		s.ImplementationFingerprint != ""
}

// Provider supplies a language parser for a specific candidate type. A parser
// returns a candidate; the returned value is not automatically accepted as
// Canonical meaning. Provider identity does not grant semantic authority.
type Provider[T any] struct {
	Selection Selection
	Parse     func(source string) (T, error)
}

// Resolve requires an exact language and implementation pin. It deliberately
// never selects the latest provider, or one found first in registry order.
func Resolve[T any](pin Selection, providers []Provider[T]) (Provider[T], error) {
	var zero Provider[T]
	if !pin.valid() {
		return zero, ErrInvalidSelection
	}
	matches := 0
	var found Provider[T]
	for _, p := range providers {
		if p.Selection != pin {
			continue
		}
		matches++
		found = p
	}
	if matches == 0 {
		return zero, fmt.Errorf("%w: %s@%s (%s)", ErrProviderMissing, pin.LanguageID, pin.ContractVersion, pin.ImplementationID)
	}
	if matches > 1 {
		return zero, fmt.Errorf("%w: %s@%s (%s)", ErrProviderAmbiguous, pin.LanguageID, pin.ContractVersion, pin.ImplementationID)
	}
	if found.Parse == nil {
		return zero, fmt.Errorf("%w: %s has no parser", ErrProviderMissing, pin.ImplementationID)
	}
	return found, nil
}

// ParseSelected returns an unaccepted candidate from the selected parser.
// Source identity and path are intentionally absent from Selection; callers
// retain source spans, validate semantics, and use the existing acceptance
// boundary before making authoritative changes.
func ParseSelected[T any](pin Selection, providers []Provider[T], source string) (T, error) {
	var zero T
	provider, err := Resolve(pin, providers)
	if err != nil {
		return zero, err
	}
	result, err := provider.Parse(source)
	if err != nil {
		return zero, fmt.Errorf("parse via %s: %w", pin.ImplementationID, err)
	}
	return result, nil
}
