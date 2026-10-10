package languagereference

import (
	"errors"
	"fmt"
)

var (
	ErrOperatorMissing   = errors.New("operator contract is unavailable")
	ErrOperatorAmbiguous = errors.New("operator contract is ambiguous")
	ErrOperatorInvalid   = errors.New("operator contract is incomplete")
	ErrEffectEscalation  = errors.New("operator composition escalates effects")
)

// OperatorContract is an E2 reference-only semantic declaration.
// Effect is deliberately restricted to pure: side effects require existing
// Capability authorization, not a new DSL execution shortcut.
type OperatorContract struct {
	ID          string
	Version     string
	Fingerprint string
	Effect      string
	InputType   string
	OutputType  string
}

// ResolveOperator never chooses a newer version or the first registered item.
// Conflicting fingerprints for the same identity and version are ambiguous,
// even if the requested fingerprint is present in the catalog.
func ResolveOperator(pin OperatorContract, catalog []OperatorContract) (OperatorContract, error) {
	if pin.ID == "" || pin.Version == "" || pin.Fingerprint == "" || pin.InputType == "" || pin.OutputType == "" || pin.Effect == "" {
		return OperatorContract{}, ErrOperatorInvalid
	}
	if pin.Effect != "pure" {
		return OperatorContract{}, fmt.Errorf("%w: %s", ErrEffectEscalation, pin.ID)
	}
	matches := 0
	var found OperatorContract
	for _, c := range catalog {
		if c.ID == pin.ID && c.Version == pin.Version {
			matches++
			found = c
		}
	}
	if matches == 0 || (matches == 1 && found.Fingerprint != pin.Fingerprint) {
		return OperatorContract{}, fmt.Errorf("%w: %s@%s", ErrOperatorMissing, pin.ID, pin.Version)
	}
	if matches > 1 {
		return OperatorContract{}, fmt.Errorf("%w: %s@%s", ErrOperatorAmbiguous, pin.ID, pin.Version)
	}
	if found != pin {
		return OperatorContract{}, fmt.Errorf("%w: incompatible signature for %s", ErrOperatorInvalid, pin.ID)
	}
	return found, nil
}
