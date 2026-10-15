// Package derivationprojection is a D10 read-only reference boundary.
// It fixes the minimal projection contract that lets an integrated modeller or
// a standard business screen show a derived item without confusing four
// states: the declared meaning, an unsaved preview, a runtime derived value
// and a stored confirmed fact.
//
// It does not build an editor, an ER editor, a reactive engine, a Command or
// transaction, and it never holds a second copy of a rule.
package derivationprojection

import (
	"errors"
	"fmt"
	"strings"

	"tei/internal/semantic/derivationref"
)

var (
	ErrInvalidField       = errors.New("projected field is invalid")
	ErrDerivedNotWritable = errors.New("a derived field cannot be written directly")
	ErrRuleNotDerived     = errors.New("a rule identity applies only to a derived field")
)

// SlotKind is what a model field represents in a functionality list.
type SlotKind string

const (
	PhysicalStored  SlotKind = "physical-stored"
	Derived         SlotKind = "derived"
	ConfirmedStored SlotKind = "confirmed-stored"
)

// Field is one projected model field. ID is the stable identity across a rule
// revision or a redeployment; a regenerated artifact must not present a new ID
// as a different business attribute. RuleID is an accepted rule identity, not
// a copy of the rule body.
type Field struct {
	ID     string
	Kind   SlotKind
	RuleID string
}

// Writable reports whether the field accepts a direct value write. A derived
// field never does, at the metadata level as well as in the UI.
func (f Field) Writable() bool {
	return f.Kind == PhysicalStored || f.Kind == ConfirmedStored
}

// Validate rejects a field that lacks a rule identity for a derived slot or
// that carries one for a stored slot.
func (f Field) Validate() error {
	if f.ID == "" || f.ID != strings.TrimSpace(f.ID) {
		return fmt.Errorf("%w: id", ErrInvalidField)
	}
	switch f.Kind {
	case Derived:
		if f.RuleID == "" {
			return fmt.Errorf("%w: derived field needs a rule identity", ErrInvalidField)
		}
	case PhysicalStored, ConfirmedStored:
		if f.RuleID != "" {
			return fmt.Errorf("%w: stored field carries a rule identity", ErrInvalidField)
		}
	default:
		return fmt.Errorf("%w: kind %q", ErrInvalidField, f.Kind)
	}
	return nil
}

// MeaningState is the semantic state of one projected value. Renderers receive
// this and never hold or evaluate a rule of their own.
type MeaningState string

const (
	Declared         MeaningState = "declared"
	Preview          MeaningState = "preview"
	Evaluated        MeaningState = "evaluated"
	Confirmed        MeaningState = "confirmed"
	Unavailable      MeaningState = "input-unavailable"
	EvaluationFailed MeaningState = "evaluation-failed"
	NotAuthorized    MeaningState = "authorization-denied"
)

// Value is one projected cell. Amount is set only for Preview, Evaluated and
// Confirmed; an unavailable, failed or denied cell carries no value, so a
// previous 600 is not shown as if it were current.
type Value struct {
	FieldID string       `json:"fieldId"`
	State   MeaningState `json:"state"`
	Amount  string       `json:"amount,omitempty"`
}

// ComputeFunc is one accepted rule body (D1). The projection refers to it by
// identity and never embeds it.
type ComputeFunc func(quantity, unitPrice derivationref.Input) (derivationref.Input, error)

// PreviewValue derives from unsaved inputs. A successful preview is not a save
// or a confirmation.
func PreviewValue(fieldID string, quantity, unitPrice derivationref.Input, rule ComputeFunc) Value {
	return derive(fieldID, Preview, quantity, unitPrice, rule)
}

// Evaluate derives the runtime value from the current inputs under the
// accepted rule. It does not touch the confirmed fact.
func Evaluate(fieldID string, quantity, unitPrice derivationref.Input, rule ComputeFunc) Value {
	return derive(fieldID, Evaluated, quantity, unitPrice, rule)
}

// Denied is the state when the current authorization rejects the derived value
// (owned by #1693). It carries no amount.
func Denied(fieldID string) Value {
	return Value{FieldID: fieldID, State: NotAuthorized}
}

func derive(fieldID string, state MeaningState, quantity, unitPrice derivationref.Input, rule ComputeFunc) Value {
	if rule == nil {
		return Value{FieldID: fieldID, State: EvaluationFailed}
	}
	amount, err := rule(quantity, unitPrice)
	if err != nil {
		if errors.Is(err, derivationref.ErrInput) {
			return Value{FieldID: fieldID, State: Unavailable}
		}
		return Value{FieldID: fieldID, State: EvaluationFailed}
	}
	if amount.State != derivationref.Present {
		return Value{FieldID: fieldID, State: Unavailable}
	}
	return Value{FieldID: fieldID, State: state, Amount: amount.Decimal}
}

// ConfirmedValue returns the stored business fact unchanged. stored is nil
// when the row is not confirmed. A recomputation never overwrites it.
func ConfirmedValue(fieldID string, stored *string) Value {
	if stored == nil {
		return Value{FieldID: fieldID, State: Declared}
	}
	return Value{FieldID: fieldID, State: Confirmed, Amount: *stored}
}

// WriteTarget is the application/persistence boundary for a value write, not
// the UI. A direct write to a derived field is rejected here, so hiding the
// editor is not the only guard.
func WriteTarget(field Field, amount string) error {
	if err := field.Validate(); err != nil {
		return err
	}
	if field.Kind == Derived {
		return fmt.Errorf("%w: %s", ErrDerivedNotWritable, field.ID)
	}
	return nil
}

// RuleEdit is a change to the declared rule identity, handled by the existing
// Proposal -> Acceptance -> Source flow (#1124/#267). It is a different type
// from a value write, so "change the rule" and "overwrite the computed cell"
// cannot be expressed as each other.
type RuleEdit struct {
	FieldID        string
	ProposedRuleID string
}

// ValidateRuleEdit rejects a rule edit on a non-derived field and an empty
// proposal.
func ValidateRuleEdit(field Field, edit RuleEdit) error {
	if err := field.Validate(); err != nil {
		return err
	}
	if field.Kind != Derived {
		return fmt.Errorf("%w: %s", ErrRuleNotDerived, field.ID)
	}
	if edit.FieldID != field.ID || strings.TrimSpace(edit.ProposedRuleID) == "" {
		return fmt.Errorf("%w: rule edit", ErrInvalidField)
	}
	return nil
}

// Reproject applies a new rule revision to a derived field, keeping the same
// field identity. A redeployment must not turn a product-identity change into
// a different business attribute ID.
func Reproject(field Field, ruleID string) (Field, error) {
	if err := field.Validate(); err != nil {
		return Field{}, err
	}
	if field.Kind != Derived {
		return Field{}, fmt.Errorf("%w: %s", ErrRuleNotDerived, field.ID)
	}
	if strings.TrimSpace(ruleID) == "" {
		return Field{}, fmt.Errorf("%w: rule identity", ErrInvalidField)
	}
	projected := field
	projected.RuleID = ruleID
	return projected, projected.Validate()
}

// Renderer projects a meaning state to a target-specific surface. It never
// holds or evaluates a rule; different renderers must agree on the meaning.
type Renderer interface {
	Render(value Value) string
}

// WebRenderer is a browser surface.
type WebRenderer struct{}

func (WebRenderer) Render(value Value) string {
	return "web|" + string(value.State) + "|" + value.Amount
}

// CLIRenderer is a command-line surface.
type CLIRenderer struct{}

func (CLIRenderer) Render(value Value) string {
	return "cli|" + string(value.State) + "|" + value.Amount
}
