package validationeval

import (
	"errors"
	"fmt"
	"math/big"

	"tei/internal/core"
	"tei/internal/semantic"
)

// This file is the entry of evaluation for a reader that can say why a value
// is not there.
//
// "There is no value" has more than one meaning. The record may hold an empty
// value, or the reader may not have asked for it, or the reader may have read
// only part of the rows. A rule is decided only when its answer is the same
// whatever the unread part turns out to be. Otherwise the answer is neither
// satisfied nor not satisfied: it is undecided, with the reason.

// ValueState says what a reader knows about one value.
type ValueState string

const (
	// ValuePresent: the value was read and is there.
	ValuePresent ValueState = "present"
	// ValueNull: the record was read and holds no value for the attribute.
	ValueNull ValueState = "null"
	// ValueNotFetched: the reader did not read the attribute. Nothing is
	// known about it, including whether it is empty.
	ValueNotFetched ValueState = "not-fetched"
)

// Coverage says whether a reader read every row of a relation.
type Coverage string

const (
	CoverageComplete Coverage = "complete"
	// CoveragePartial: some rows were not read (a page, a limit, a filter the
	// rule did not ask for).
	CoveragePartial Coverage = "partial"
)

// NumberReading is one numeric value and its state. Value is set exactly when
// State is ValuePresent.
type NumberReading struct {
	State ValueState
	Value *big.Rat
}

// KeyReading is the comparison key of one row and its state. Key is set
// exactly when State is ValuePresent.
type KeyReading struct {
	State ValueState
	Key   string
}

// RelationReading is the keys of the rows that were read and whether those
// are all the rows.
type RelationReading struct {
	Coverage Coverage
	Keys     []KeyReading
}

// StatefulSource is a Source that reports the state of what it read instead
// of turning every absence into the same nil.
type StatefulSource interface {
	ReadNumber(attribute core.CanonicalID) (NumberReading, error)
	ReadRelationKeys(relation, attribute core.CanonicalID) (RelationReading, error)
}

// UndecidedReason names why a rule could not be decided.
type UndecidedReason string

const (
	UndecidedValueNull             UndecidedReason = "value-null"
	UndecidedValueNotFetched       UndecidedReason = "value-not-fetched"
	UndecidedRelationKeyNotFetched UndecidedReason = "relation-key-not-fetched"
	UndecidedRelationPartiallyRead UndecidedReason = "relation-partially-read"
	UndecidedRelationKeyNull       UndecidedReason = "relation-key-null"
)

// Undecided is the reason and the place a decision stopped at.
type Undecided struct {
	Reason    UndecidedReason
	Attribute core.CanonicalID
	// Relation is set for a rule over a relation.
	Relation core.CanonicalID
}

func (u Undecided) String() string {
	if u.Relation != "" {
		return fmt.Sprintf("%s (relation %s / attribute %s)", u.Reason, u.Relation, u.Attribute)
	}
	return fmt.Sprintf("%s (attribute %s)", u.Reason, u.Attribute)
}

// Decision is the outcome of Decide. Exactly one of Result and Undecided is
// set: a decided rule has a Result, an undecided rule has none.
type Decision struct {
	Result    Result
	Undecided *Undecided
}

// Decide evaluates rule against a reader that reports states.
//
// It returns a Result only when the answer does not depend on anything the
// reader did not read. An empty or unread value is never read as zero, and a
// partly read relation is never read as the whole relation. A reading that
// contradicts itself (a present value without a value, an unknown state) is
// an error of the reader, not an undecided rule.
func Decide(rule semantic.Validation, source StatefulSource, operators Operators) (Decision, error) {
	if source == nil {
		return Decision{}, fmt.Errorf("no Source is specified to evaluate semantic validation %s", rule.ID)
	}
	return decide(rule, source, operators, true)
}

var errLegacyNoNumber = errors.New("legacy source returned no number")

// legacySource reads a Source that cannot report states. Everything it
// returns is present and complete; a nil number has no known reason and stays
// the error it always was.
type legacySource struct{ source Source }

func (l legacySource) ReadNumber(attribute core.CanonicalID) (NumberReading, error) {
	value, err := l.source.Number(attribute)
	if err != nil {
		return NumberReading{}, err
	}
	if value == nil {
		return NumberReading{}, errLegacyNoNumber
	}
	return NumberReading{State: ValuePresent, Value: value}, nil
}

func (l legacySource) ReadRelationKeys(relation, attribute core.CanonicalID) (RelationReading, error) {
	keys, err := l.source.RelationKeys(relation, attribute)
	if err != nil {
		return RelationReading{}, err
	}
	out := RelationReading{Coverage: CoverageComplete, Keys: make([]KeyReading, 0, len(keys))}
	for _, key := range keys {
		out.Keys = append(out.Keys, KeyReading{State: ValuePresent, Key: key})
	}
	return out, nil
}

func decided(result Result) (Decision, error) { return Decision{Result: result}, nil }

func undecided(reason UndecidedReason, relation, attribute core.CanonicalID) (Decision, error) {
	return Decision{Undecided: &Undecided{Reason: reason, Relation: relation, Attribute: attribute}}, nil
}

func satisfiedResult(satisfied bool) Result {
	if satisfied {
		return ResultSatisfied
	}
	return ResultNotSatisfied
}

// readNumber returns the value of a present reading, or the reason there is
// none. Exactly one of the first two results is set when err is nil.
func readNumber(rule semantic.Validation, source StatefulSource) (*big.Rat, *Undecided, error) {
	reading, err := source.ReadNumber(rule.Attribute)
	if errors.Is(err, errLegacyNoNumber) {
		return nil, nil, fmt.Errorf("semantic validation %s has no evaluable number for attribute %s", rule.ID, rule.Attribute)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("semantic validation %s cannot evaluate attribute %s: %w", rule.ID, rule.Attribute, err)
	}
	switch reading.State {
	case ValuePresent:
		if reading.Value == nil {
			return nil, nil, fmt.Errorf("semantic validation %s: the reader reports attribute %s as present without a value", rule.ID, rule.Attribute)
		}
		return reading.Value, nil, nil
	case ValueNull, ValueNotFetched:
		if reading.Value != nil {
			return nil, nil, fmt.Errorf("semantic validation %s: the reader reports attribute %s as %s together with a value", rule.ID, rule.Attribute, reading.State)
		}
		reason := UndecidedValueNull
		if reading.State == ValueNotFetched {
			reason = UndecidedValueNotFetched
		}
		return nil, &Undecided{Reason: reason, Attribute: rule.Attribute}, nil
	default:
		return nil, nil, fmt.Errorf("semantic validation %s: the reader reports attribute %s in the unknown state %q", rule.ID, rule.Attribute, reading.State)
	}
}

func decide(rule semantic.Validation, source StatefulSource, operators Operators, withOperators bool) (Decision, error) {
	switch {
	case rule.Kind == semantic.ValidationPositive:
		value, stopped, err := readNumber(rule, source)
		if err != nil {
			return Decision{}, err
		}
		if stopped != nil {
			return Decision{Undecided: stopped}, nil
		}
		return decided(satisfiedResult(value.Sign() > 0))

	case rule.Kind == semantic.ValidationRelationUnique:
		return decideRelationUnique(rule, source)

	case rule.Kind == semantic.ValidationOperator && withOperators:
		implementation, err := selectOperatorImplementation(rule, operators)
		if err != nil {
			return Decision{}, err
		}
		value, stopped, err := readNumber(rule, source)
		if err != nil {
			return Decision{}, err
		}
		if stopped != nil {
			// No value is not zero. The implementation is not asked.
			return Decision{Undecided: stopped}, nil
		}
		satisfied, err := implementation.Number(new(big.Rat).Set(value))
		if err != nil {
			return Decision{}, fmt.Errorf("operator %s could not decide semantic validation %s: %w", rule.Operator, rule.ID, err)
		}
		return decided(satisfiedResult(satisfied))

	default:
		return Decision{}, fmt.Errorf("kind %q of semantic validation %s has no corresponding Evaluator", rule.ID, rule.Kind)
	}
}

// decideRelationUnique decides "no two rows share a key".
//
// Two present keys that are equal settle the rule as not satisfied whatever
// else is unknown: no unread row and no empty value can undo a duplicate that
// was seen. Without such a pair, the rule is satisfied only if every row was
// read and every key is present.
func decideRelationUnique(rule semantic.Validation, source StatefulSource) (Decision, error) {
	reading, err := source.ReadRelationKeys(rule.Relation, rule.Attribute)
	if err != nil {
		return Decision{}, fmt.Errorf("semantic validation %s cannot evaluate relation %s / attribute %s: %w", rule.ID, rule.Relation, rule.Attribute, err)
	}
	if reading.Coverage != CoverageComplete && reading.Coverage != CoveragePartial {
		return Decision{}, fmt.Errorf("semantic validation %s: the reader reports relation %s with the unknown coverage %q", rule.ID, rule.Relation, reading.Coverage)
	}
	seen := make(map[string]struct{}, len(reading.Keys))
	duplicate, notFetched, null := false, false, false
	for _, key := range reading.Keys {
		switch key.State {
		case ValuePresent:
			if _, exists := seen[key.Key]; exists {
				duplicate = true
			}
			seen[key.Key] = struct{}{}
		case ValueNull, ValueNotFetched:
			if key.Key != "" {
				return Decision{}, fmt.Errorf("semantic validation %s: the reader reports a key of relation %s as %s together with a value", rule.ID, rule.Relation, key.State)
			}
			if key.State == ValueNull {
				null = true
			} else {
				notFetched = true
			}
		default:
			return Decision{}, fmt.Errorf("semantic validation %s: the reader reports a key of relation %s in the unknown state %q", rule.ID, rule.Relation, key.State)
		}
	}
	switch {
	case duplicate:
		return decided(ResultNotSatisfied)
	// What reading more could settle is reported before what it could not.
	case notFetched:
		return undecided(UndecidedRelationKeyNotFetched, rule.Relation, rule.Attribute)
	case reading.Coverage == CoveragePartial:
		return undecided(UndecidedRelationPartiallyRead, rule.Relation, rule.Attribute)
	case null:
		// Whether "unique" also speaks about rows without a value is not
		// stated by the rule, so it is not guessed either way.
		return undecided(UndecidedRelationKeyNull, rule.Relation, rule.Attribute)
	default:
		return decided(ResultSatisfied)
	}
}
