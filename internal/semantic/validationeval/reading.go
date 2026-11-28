package validationeval

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

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

// DateReading is one calendar date and its state. Value is a date exactly
// when State is ValuePresent, and the zero value otherwise.
type DateReading struct {
	State ValueState
	Value semantic.CalendarDate
}

// StatefulDateSource is implemented by a reader that can read calendar
// dates. A rule over dates is not evaluated with a reader that cannot: the
// rule is refused, it is not decided from another representation.
type StatefulDateSource interface {
	ReadDate(attribute core.CanonicalID) (DateReading, error)
}

// TextReading is one text value and its state. Value is set exactly when State
// is ValuePresent.
//
// It is a reading of its own rather than a number read from the same column,
// because the two ask different questions: a text value is compared as written,
// and a column that holds no number is not therefore holding no text.
type TextReading struct {
	State ValueState
	Value string
}

// StatefulTextSource is implemented by a reader that can read a text value. A
// rule about text is refused by a reader that cannot; it is not decided from a
// value of another type (section 8.4).
type StatefulTextSource interface {
	ReadText(attribute core.CanonicalID) (TextReading, error)
}

// TruthReading is one truth value and its state. Value is set exactly when
// State is ValuePresent.
//
// The zero value is not false: a truth nobody read is not the truth "false",
// and the state says which of the two it is.
type TruthReading struct {
	State ValueState
	Value bool
}

// StatefulTruthSource is implemented by a reader that can read a truth value. A
// rule about truth is refused by a reader that cannot; it is not decided from a
// value of another type (section 8.4).
type StatefulTruthSource interface {
	ReadTruth(attribute core.CanonicalID) (TruthReading, error)
}

// AnyReading is one value of any type and its state. Value is set exactly when
// State is ValuePresent.
//
// The value keeps the type the reader read it as, because the declaration says
// nothing about the type: an input declared `any` may be a number in one record
// and a text in another, and the value carries which one it was (section 8.4).
type AnyReading struct {
	State ValueState
	Value semantic.AnyValue
}

// StatefulAnySource is implemented by a reader that can read a value of any
// type. A rule about `any` is refused by a reader that cannot; it is not decided
// from a value of another type (section 8.4).
type StatefulAnySource interface {
	ReadAny(attribute core.CanonicalID) (AnyReading, error)
}

// PresenceReading is what a reader knows about whether one attribute of one
// record holds a value.
//
// It is deliberately not a typed value: a rule about presence is decided by
// whether a value is there, whatever its type. State carries the same three
// meanings as ValueState, so "the record holds no value" and "the attribute
// was not read" stay apart.
type PresenceReading struct {
	State ValueState
}

// StatefulPresenceSource is implemented by a reader that can read whether an
// attribute holds a value. A rule about presence is refused by a reader that
// cannot; it is not decided from a value of another type.
type StatefulPresenceSource interface {
	ReadPresence(attribute core.CanonicalID) (PresenceReading, error)
}

// UndecidedReason names why a rule could not be decided.
type UndecidedReason string

const (
	UndecidedValueNull             UndecidedReason = "value-null"
	UndecidedValueNotFetched       UndecidedReason = "value-not-fetched"
	UndecidedRelationKeyNotFetched UndecidedReason = "relation-key-not-fetched"
	UndecidedRelationPartiallyRead UndecidedReason = "relation-partially-read"
	UndecidedRelationKeyNull       UndecidedReason = "relation-key-null"
	// UndecidedReadAttemptUnknown is reported when a rule reads more than one
	// attribute and the reader cannot say that they came from the same Read
	// Attempt. Having the values is not the same as knowing they are one
	// snapshot (decision section 14.2 B-10), so the rule stays undecided.
	UndecidedReadAttemptUnknown UndecidedReason = "read-attempt-unknown"
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

// legacyAttempt names the single read the legacy Source interface describes.
// That interface answers for one record ("the value of this attribute of this
// record"), so every attribute it returns belongs to that one record's read.
// A reader that can report several attempts implements AttributedSource and
// its own attempt is used instead.
const legacyAttempt = "legacy-single-record"

func (l legacySource) ReadAttempt(core.CanonicalID) (string, bool) {
	return legacyAttempt, true
}

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

// contractInputs returns the inputs of an operator signature in order. A
// contract that states no signature is the single-input operator the surface
// has always had: its one input is named "value" and takes the type the
// contract states. This keeps one reading of a contract instead of two
// parallel fields.
func contractInputs(contract semantic.Operator) []semantic.OperatorInputDeclaration {
	if len(contract.Inputs) == 0 {
		return []semantic.OperatorInputDeclaration{{Name: "value", Type: contract.Input}}
	}
	inputs := make([]semantic.OperatorInputDeclaration, 0, len(contract.Inputs))
	for _, input := range contract.Inputs {
		if input.Name == "" {
			input.Name = "value"
		}
		inputs = append(inputs, input)
	}
	return inputs
}

// readAnySubject reads the subject attribute as a value of the type the reader
// read it as (section 8.4). The type travels with the value, so an
// implementation is never asked to guess which kind it holds.
//
// A value two types read equally is refused rather than resolved by picking
// one: the same ambiguity section 8.3 refuses in an example table, because a
// reader that resolves it would decide a rule from a type the author did not
// state.
func readAnySubject(rule semantic.Validation, source StatefulSource) (semantic.OperatorValue, *Undecided, error) {
	reader, ok := source.(StatefulAnySource)
	if !ok {
		return semantic.OperatorValue{}, nil, fmt.Errorf("semantic validation %s reads attribute %s as any type, but this reader cannot read a value of any type", rule.ID, rule.Attribute)
	}
	reading, err := reader.ReadAny(rule.Attribute)
	if err != nil {
		return semantic.OperatorValue{}, nil, err
	}
	switch reading.State {
	case ValuePresent:
		value := reading.Value
		return semantic.OperatorValue{Any: &value}, nil, nil
	case ValueNull:
		return semantic.OperatorValue{}, &Undecided{Reason: UndecidedValueNull, Attribute: rule.Attribute}, nil
	default:
		return semantic.OperatorValue{}, &Undecided{Reason: UndecidedValueNotFetched, Attribute: rule.Attribute}, nil
	}
}

// readAssignedInput reads one input value that was assigned by name in the rule.
// For an attribute reference it reads from the source; for a constant it parses
// the decimal text. The input's absent-as mapping is not yet carried in the
// validation, so absent values are treated as undecided.
//
// It returns (*Undecided, error). A reader error is returned as the error
// (with rule context), not turned into an Undecided, so the caller can
// distinguish "no data" from "the reader failed". A missing or unreadable
// assignment is neither: the caller refuses an unassigned input before asking.
// readSubjectInput reads the first input, which the rule subject supplies from
// the attribute the rule constrains. It exists so the subject goes through the
// same absent-mapping handling as every later input: the value is the attribute
// the rule names, and if that attribute certainly holds no value the
// declaration's `absent as <name>` decides how the absence reads.
//
// A subject attribute that was not read stays undecided, and in the legacy
// reader a nil number stays the error it has always been: a reader that did not
// ask, or cannot say, knows nothing about the value, so no mapping applies. The
// mapping is only for "the record holds no value", which the reader states.
func readSubjectInput(rule semantic.Validation, input semantic.OperatorInputDeclaration, source StatefulSource, absent func(string) (*big.Rat, bool)) (*big.Rat, *Undecided, error) {
	value, stopped, err := readNumber(rule, source)
	if err != nil {
		return nil, nil, err
	}
	if stopped == nil {
		return value, nil, nil
	}
	if stopped.Reason == UndecidedValueNull && input.AbsentAs != "" && absent != nil {
		if mapped, ok := absent(input.AbsentAs); ok {
			return new(big.Rat).Set(mapped), nil, nil
		}
	}
	return nil, stopped, nil
}

// readTextSubject reads the subject attribute as a text value, in the type the
// input declaration states (section 8.4). The state is carried next to the
// value, so "the record holds no text" and "nobody read the attribute" stay
// apart rather than both becoming the empty string.
func readTextSubject(rule semantic.Validation, source StatefulSource) (semantic.OperatorValue, *Undecided, error) {
	reader, ok := source.(StatefulTextSource)
	if !ok {
		return semantic.OperatorValue{}, nil, fmt.Errorf("semantic validation %s reads attribute %s as text, but this reader cannot read text", rule.ID, rule.Attribute)
	}
	reading, err := reader.ReadText(rule.Attribute)
	if err != nil {
		return semantic.OperatorValue{}, nil, err
	}
	switch reading.State {
	case ValuePresent:
		text := reading.Value
		return semantic.OperatorValue{Text: &text}, nil, nil
	case ValueNull:
		return semantic.OperatorValue{}, &Undecided{Reason: UndecidedValueNull, Attribute: rule.Attribute}, nil
	default:
		return semantic.OperatorValue{}, &Undecided{Reason: UndecidedValueNotFetched, Attribute: rule.Attribute}, nil
	}
}

// readTruthSubject reads the subject attribute as a truth value, in the type the
// input declaration states (section 8.4). A truth nobody read is not the truth
// false: the state says which of the two it is.
func readTruthSubject(rule semantic.Validation, source StatefulSource) (semantic.OperatorValue, *Undecided, error) {
	reader, ok := source.(StatefulTruthSource)
	if !ok {
		return semantic.OperatorValue{}, nil, fmt.Errorf("semantic validation %s reads attribute %s as truth, but this reader cannot read truth", rule.ID, rule.Attribute)
	}
	reading, err := reader.ReadTruth(rule.Attribute)
	if err != nil {
		return semantic.OperatorValue{}, nil, err
	}
	switch reading.State {
	case ValuePresent:
		truth := reading.Value
		return semantic.OperatorValue{Truth: &truth}, nil, nil
	case ValueNull:
		return semantic.OperatorValue{}, &Undecided{Reason: UndecidedValueNull, Attribute: rule.Attribute}, nil
	default:
		return semantic.OperatorValue{}, &Undecided{Reason: UndecidedValueNotFetched, Attribute: rule.Attribute}, nil
	}
}

func readAssignedInput(rule semantic.Validation, input semantic.OperatorInputDeclaration, values map[string]*big.Rat, source StatefulSource, absent func(string) (*big.Rat, bool)) (*Undecided, error) {
	assignment := findInputAssignment(rule, input.Name)
	if assignment == nil {
		return &Undecided{Reason: UndecidedValueNotFetched, Attribute: core.CanonicalID(input.Name)}, nil
	}
	if isAttributeReference(assignment.Value) {
		attr, err := core.ParseCanonicalID(assignment.Value)
		if err != nil {
			// Malformed reference: treat as not fetched.
			return &Undecided{Reason: UndecidedValueNotFetched, Attribute: core.CanonicalID(input.Name)}, nil
		}
		value, stopped, err := readNumberForAttribute(rule, attr, source)
		if err != nil {
			return nil, err
		}
		if stopped != nil {
			// The input held no value. If the rule named a mapping and this
			// implementation can read the absence that way, it is read as
			// that value; otherwise the absence keeps its own reason. An
			// implementation that cannot read the mapping must not be given
			// zero, and the absence must not become a satisfied input.
			//
			// Only a value the record certainly does not hold is read this way.
			// A value the reader never fetched is unknown, not absent, so a
			// mapping does not apply to it: rewriting it would decide a rule
			// from a value nobody read (decision section 5's table). The
			// row-scoped path was corrected for this and the per-record path
			// was not, so one rule answered differently depending on which path
			// read it (#4359).
			if stopped.Reason != UndecidedValueNull {
				return stopped, nil
			}
			if assignment.AbsentAs != "" && absent != nil {
				if mapped, ok := absent(assignment.AbsentAs); ok {
					values[input.Name] = new(big.Rat).Set(mapped)
					return nil, nil
				}
			}
			return stopped, nil
		}
		values[input.Name] = new(big.Rat).Set(value)
		return nil, nil
	}
	// Constant value.
	value, err := parseConstant(assignment.Value)
	if err != nil {
		return &Undecided{Reason: UndecidedValueNotFetched, Attribute: core.CanonicalID(input.Name)}, nil
	}
	values[input.Name] = value
	return nil, nil
}

// findInputAssignment returns the named input assignment from the rule, or nil.
func findInputAssignment(rule semantic.Validation, name string) *semantic.ValidationInputAssignment {
	for i := range rule.Inputs {
		if rule.Inputs[i].Name == name {
			return &rule.Inputs[i]
		}
	}
	return nil
}

// isAttributeReference reports whether text looks like a canonical attribute
// reference (contains a dot and parses as a canonical id).
func isAttributeReference(text string) bool {
	if !strings.Contains(text, ".") {
		return false
	}
	_, err := core.ParseCanonicalID(text)
	return err == nil
}

// parseConstant reads a decimal constant from text.
func parseConstant(text string) (*big.Rat, error) {
	if text == "" {
		return nil, fmt.Errorf("empty constant")
	}
	value, ok := new(big.Rat).SetString(text)
	if !ok {
		return nil, fmt.Errorf("%q is not a valid decimal constant", text)
	}
	return value, nil
}

// readNumberForAttribute reads one number for the given attribute of a rule.
func readNumberForAttribute(rule semantic.Validation, attribute core.CanonicalID, source StatefulSource) (*big.Rat, *Undecided, error) {
	reading, err := source.ReadNumber(attribute)
	if errors.Is(err, errLegacyNoNumber) {
		return nil, &Undecided{Reason: UndecidedValueNotFetched, Attribute: attribute}, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("semantic validation %s cannot evaluate attribute %s: %w", rule.ID, attribute, err)
	}
	switch reading.State {
	case ValuePresent:
		if reading.Value == nil {
			return nil, nil, fmt.Errorf("semantic validation %s: the reader reports attribute %s as present without a value", rule.ID, attribute)
		}
		return reading.Value, nil, nil
	case ValueNull, ValueNotFetched:
		if reading.Value != nil {
			return nil, nil, fmt.Errorf("semantic validation %s: the reader reports attribute %s as %s together with a value", rule.ID, attribute, reading.State)
		}
		reason := UndecidedValueNull
		if reading.State == ValueNotFetched {
			reason = UndecidedValueNotFetched
		}
		return nil, &Undecided{Reason: reason, Attribute: attribute}, nil
	default:
		return nil, nil, fmt.Errorf("semantic validation %s: the reader reports attribute %s in the unknown state %q", rule.ID, attribute, reading.State)
	}
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

	case rule.Kind == semantic.ValidationNotAfter:
		return decideNotAfter(rule, source)

	case rule.Kind == semantic.ValidationNoOverlap:
		return decideNoOverlap(rule, source)

	case rule.Kind == semantic.ValidationRequiredIf:
		return decideRequiredIf(rule, source)

	case rule.Kind == semantic.ValidationOperator && withOperators:
		implementation, err := selectOperatorImplementation(rule, operators)
		if err != nil {
			return Decision{}, err
		}
		// Find the declared operator to learn how many inputs to read and
		// where each one comes from.
		var declared []semantic.Operator
		for _, op := range operators.Declared {
			if op.ID == rule.Operator {
				declared = append(declared, op)
			}
		}
		if len(declared) != 1 {
			return Decision{}, fmt.Errorf("semantic validation %s uses operator %s, which is declared %d times", rule.ID, rule.Operator, len(declared))
		}
		contract := declared[0]
		inputs := contractInputs(contract)
		// A presence input asks whether an attribute holds a value, and not
		// which value it holds. It is decided by its own path, before the
		// value paths: reading it through ReadNumber was the mistake #4359
		// measured (a presence contract answered satisfied from the value 7),
		// and reading it as a value of any type would answer the same
		// question by accident.
		if declaresPresenceInput(inputs) {
			for _, input := range inputs {
				if input.OfEachRow {
					return decidePresenceRowScopedOperator(rule, contract, inputs, implementation, source)
				}
			}
			return decidePresenceOperator(rule, contract, inputs, implementation, source)
		}
		// A rule whose operator declares an input read over every row of a
		// relation is asked of those rows, not of one record (decision section
		// 15.1). Every such input of this rule is read in one call, so one
		// answer never mixes two reads (section 14.2 B-10).
		//
		// The row-scoped question is decided before the one-record one: a
		// column of days is not one record's day, and reading it through the
		// one-record reader would answer a question about the relation from one
		// record of it.
		for _, input := range inputs {
			if input.OfEachRow {
				return decideRowScopedOperator(rule, contract, inputs, implementation, source)
			}
		}
		// A contract may declare a calendar-day input. A day is read as a day,
		// never as a number: `2026-04-01` read as a decimal is not a value
		// anyone wrote, and the two reads ask different questions.
		if declaresDateInput(inputs) {
			return decideDateOperator(rule, contract, inputs, implementation, source)
		}
		// The rule subject supplies the first input, and only it. A rule that
		// also assigns that input by name binds it twice, so the instruction
		// is refused instead of read.
		for _, assignment := range rule.Inputs {
			if assignment.Name == inputs[0].Name {
				return Decision{}, fmt.Errorf("semantic validation %s assigns operator %s input %q, which the rule subject already supplies", rule.ID, rule.Operator, assignment.Name)
			}
		}
		// The subject is read in the type its declaration states. A number is
		// read as it always was; a text and a truth are read by the reader that
		// understands them, which is what section 8.4 asks for: the value an
		// implementation receives carries the type it was read as, so a rule
		// about text is never answered from a number the attribute was never
		// read as.
		//
		// A subject that holds no value stays undecided, whatever its type, so
		// an empty cell is not read as the empty text or as false.
		var typedSubject semantic.OperatorValue
		switch inputs[0].Type {
		case semantic.OperatorInputNumber:
			number, undecided, err := readSubjectInput(rule, inputs[0], source, implementation.Absent)
			if err != nil {
				return Decision{}, err
			}
			if undecided != nil {
				return Decision{Undecided: undecided}, nil
			}
			typedSubject = semantic.OperatorValue{Number: number}
		case semantic.OperatorInputText:
			value, undecided, err := readTextSubject(rule, source)
			if err != nil {
				return Decision{}, err
			}
			if undecided != nil {
				return Decision{Undecided: undecided}, nil
			}
			typedSubject = value
		case semantic.OperatorInputTruth:
			value, undecided, err := readTruthSubject(rule, source)
			if err != nil {
				return Decision{}, err
			}
			if undecided != nil {
				return Decision{Undecided: undecided}, nil
			}
			typedSubject = value
		case semantic.OperatorInputAny:
			value, undecided, err := readAnySubject(rule, source)
			if err != nil {
				return Decision{}, err
			}
			if undecided != nil {
				return Decision{Undecided: undecided}, nil
			}
			typedSubject = value
		default:
			// A type with no reader on this path is refused by name rather
			// than answered from the empty value: a subject nobody read is not
			// the zero of its type, and a rule decided that way would be
			// answered from a value it never held (section 8.4).
			return Decision{}, fmt.Errorf("semantic validation %s reads operator %s input %q as %s, but this reader reads an attribute as a number; the %s input has no reader on this path yet",
				rule.ID, rule.Operator, inputs[0].Name, inputs[0].Type, inputs[0].Type)
		}
		// `values` is what the numeric fields receive, so it holds the subject
		// only when the subject was read as a number. A subject read as another
		// type is carried in `typed`, which the Values field receives, and it
		// has no number here to put in this map.
		values := make(map[string]*big.Rat, len(inputs))
		if typedSubject.Number != nil {
			values[inputs[0].Name] = new(big.Rat).Set(typedSubject.Number)
		}
		// The subject and every attribute-assigned input must belong to one
		// Read Attempt. Having all three values is not the same as knowing they
		// are one snapshot (section 14.2 B-10), so the origin of each input is
		// collected while the values are, and compared before the operator is
		// asked.
		origins := []inputOrigin{dataOrigin(inputs[0].Name, string(rule.Attribute), source)}
		// A named input of a type other than number is read in that type here,
		// before the numeric read: reading it as a number first would report the
		// record as holding no value for a text nobody asked a number about.
		typedAssigned := make(map[string]semantic.OperatorValue, len(inputs))
		for _, input := range inputs[1:] {
			// Every input after the first must be assigned by name; one that
			// is not is not silently read as the subject's value or as zero.
			assignment := findInputAssignment(rule, input.Name)
			if assignment == nil {
				return Decision{}, fmt.Errorf("semantic validation %s does not assign operator %s input %q; every input after the first must be assigned", rule.ID, rule.Operator, input.Name)
			}
			if isAttributeReference(assignment.Value) {
				origins = append(origins, dataOrigin(input.Name, assignment.Value, source))
			} else {
				origins = append(origins, constantOrigin(input.Name))
			}
			if input.Type != semantic.OperatorInputNumber {
				value, stopped, readErr := readAssignedTypedInput(rule, input, source)
				if readErr != nil {
					return Decision{}, readErr
				}
				if stopped != nil {
					return Decision{Undecided: stopped}, nil
				}
				typedAssigned[input.Name] = value
				continue
			}
			got, rerr := readAssignedInput(rule, input, values, source, implementation.Absent)
			if rerr != nil {
				return Decision{}, rerr
			}
			if got != nil {
				return Decision{Undecided: got}, nil
			}
		}
		unknown, aerr := requireSameAttempt(rule, origins)
		if aerr != nil {
			return Decision{}, aerr
		}
		if unknown != nil {
			return Decision{Undecided: unknown}, nil
		}
		var satisfied bool
		if implementation.Values != nil {
			// An implementation that reads values of any type is asked about
			// the values as the declaration says they are read (section 8.4).
			// A value it does not receive is one the reader did not read, which
			// the mapping above already turned into an undecided reason, so the
			// values in hand are all present ones.
			typed := make(map[string]semantic.OperatorValue, len(inputs))
			for index, input := range inputs {
				// The subject was read in the type its declaration states, so
				// The subject was read in the type its declaration states, so
				// it is carried as that type. A named input is read the same
				// way: a text bound is a text, and a truth bound is a truth.
				// Reading one as a number would answer a question about a value
				// it never held (decision section 8.4).
				if index == 0 {
					typed[input.Name] = typedSubject
					continue
				}
				// A named input was read in the type its declaration states before
				// the numeric read, so the value read then is carried here. Reading
				// it as a number first would report the record as holding no value
				// for a text nobody asked a number about.
				if input.Type != semantic.OperatorInputNumber {
					if value, read := typedAssigned[input.Name]; read {
						typed[input.Name] = value
						continue
					}
				}
				typed[input.Name] = semantic.OperatorValue{Number: new(big.Rat).Set(values[input.Name])}
			}
			satisfied, err = implementation.Values(typed)
		} else if implementation.Numbers != nil {
			satisfied, err = implementation.Numbers(values)
		} else {
			// A single-input implementation receives the first input's value,
			// which may be stored under the input's declared name.
			satisfied, err = implementation.Number(new(big.Rat).Set(values[inputs[0].Name]))
		}
		if err != nil {
			return Decision{}, fmt.Errorf("operator %s could not decide semantic validation %s: %w", rule.Operator, rule.ID, err)
		}
		return decided(satisfiedResult(satisfied))

	default:
		return Decision{}, fmt.Errorf("kind %q of semantic validation %s has no corresponding Evaluator", rule.Kind, rule.ID)
	}
}

// decideRowScopedOperator asks a row-scoped operator's implementation about the
// states of every row of one relation and composes that answer with the read
// Coverage and the operator's refutation statement in the one place that decides
// it (decideRowScoped, row_scope.go).
//
// The first input's column is the rule's subject attribute. Every other input
// must be assigned a name, and that assignment is either an attribute (a column
// of the rows) or a constant. A constant is one value for the whole relation,
// which is what the design allows a rule to write for any input; every row of
// the relation therefore holds that value, and the operator sees one limit
// rather than a column that changes from row to row.
func decideRowScopedOperator(rule semantic.Validation, contract semantic.Operator, inputs []semantic.OperatorInputDeclaration, implementation OperatorImplementation, source StatefulSource) (Decision, error) {
	if rule.Relation == "" {
		return Decision{}, fmt.Errorf(
			"semantic validation %s uses operator %s, which reads every row, but the rule names no relation; write \"per <relation>\"",
			rule.ID, rule.Operator)
	}
	if implementation.NumberRows == nil && implementation.ValueRows == nil {
		return Decision{}, fmt.Errorf(
			"operator %s reads every row of a relation, but its selected implementation does not decide over rows",
			rule.Operator)
	}
	// A row-scoped operator whose inputs are of a type other than number is read
	// by the reader that understands that type over rows, not by the row number
	// reader (decision section 8.4). Reading the column as numbers would answer
	// a question about a value nobody read.
	if implementation.ValueRows != nil {
		return decideTypedRowScopedOperator(rule, contract, inputs, implementation, source)
	}
	reader, ok := source.(StatefulRowNumberSource)
	if !ok {
		return Decision{}, fmt.Errorf(
			"semantic validation %s reads every row of relation %s, but this reader cannot read the rows of a relation",
			rule.ID, rule.Relation)
	}

	rows := make(map[string][]NumberReading, len(inputs))
	var coverage Coverage
	var reason *Undecided
	// A constant states one value for the whole relation. Its column is as long
	// as the rows that were read, so the value is written into every row once
	// the row count is known.
	var constantValues map[string]*big.Rat
	var width []NumberReading
	for index, input := range inputs {
		attribute := rule.Attribute
		// A declaration's `absent as <mapping>` is a name the rule MAY ask for,
		// not a mapping every rule of that operator reads by default (decision
		// sections 3-5 and 8.1's table: "or なし（宣言のみ） → 一件では判定不能").
		// The per-record path already reads only the assignment's mapping for a
		// named input, so the row-scoped path reads the same thing.
		//
		// The subject is the one exception the design keeps for compatibility:
		// the rule supplies it from the attribute it constrains and never
		// assigns it by name, so the declaration is where its mapping is
		// stated (#5905, #5937).
		mapping := ""
		if index == 0 {
			mapping = input.AbsentAs
		}

		if index > 0 {
			assignment := findInputAssignment(rule, input.Name)
			if assignment == nil {
				return Decision{}, fmt.Errorf(
					"semantic validation %s does not assign operator %s input %q; every input after the first must be assigned",
					rule.ID, rule.Operator, input.Name)
			}
			if !isAttributeReference(assignment.Value) {
				value, err := parseConstant(assignment.Value)
				if err != nil {
					return Decision{}, fmt.Errorf(
						"semantic validation %s assigns operator %s input %q the value %q, which is neither an attribute nor a decimal: %w",
						rule.ID, rule.Operator, input.Name, assignment.Value, err)
				}
				if constantValues == nil {
					constantValues = make(map[string]*big.Rat, len(inputs))
				}
				// A mapping says how an absent value is read; a constant is
				// present in every row, so it cannot be absent and needs none.
				constantValues[input.Name] = value
				continue
			}
		}
		if index > 0 {
			assignment := findInputAssignment(rule, input.Name)
			named, err := core.ParseCanonicalID(assignment.Value)
			if err != nil {
				return Decision{}, fmt.Errorf(
					"semantic validation %s assigns operator %s input %q an unreadable attribute %q",
					rule.ID, rule.Operator, input.Name, assignment.Value)
			}
			attribute = named
			// The rule's assignment names the mapping it read the input under;
			// the declaration's own mapping applies only where the rule states
			// none for this input.
			if assignment.AbsentAs != "" {
				mapping = assignment.AbsentAs
			}
		}
		reading, err := reader.ReadRelationNumbers(rule.Relation, attribute)
		if err != nil {
			return Decision{}, fmt.Errorf(
				"semantic validation %s cannot read attribute %s over relation %s: %w",
				rule.ID, attribute, rule.Relation, err)
		}
		// Every input of this one rule is read from this one relation, so all of
		// them report the same coverage. Two coverages would mean two reads, and
		// one answer must not mix them.
		if coverage != "" && reading.Coverage != coverage {
			return Decision{}, fmt.Errorf(
				"semantic validation %s read relation %s with two coverages (%q and %q); one answer cannot mix two reads",
				rule.ID, rule.Relation, coverage, reading.Coverage)
		}
		coverage = reading.Coverage
		rows[input.Name] = withAbsentMapping(reading.Rows, attribute, mapping, implementation.Absent)
		if index == 0 {
			width = rows[input.Name]
		}
		// A row that holds no value, or was not read, keeps its own reason. The
		// first such row in read order is reported; the implementation is still
		// asked, so a counterexample found in another row is not thrown away.
		if reason == nil {
			reason = firstRowReason(attribute, rows[input.Name])
		}
	}
	for name, value := range constantValues {
		rows[name] = fillConstant(width, value)
	}

	answer, err := implementation.NumberRows(rows)
	if err != nil {
		return Decision{}, fmt.Errorf("operator %s could not decide semantic validation %s: %w", rule.Operator, rule.ID, err)
	}
	return decideRowScoped(rule, coverage, contract.RefutesOnSomeRows, answer, reason)
}

// withAbsentMapping rewrites the rows that hold no value under the mapping the
// rule named, when the implementation can read that mapping. A row whose value
// is replaced becomes present with the mapped value; a mapping this
// implementation cannot read leaves the row as it was, so the absence keeps its
// own reason instead of being read as zero.
func withAbsentMapping(rows []NumberReading, attribute core.CanonicalID, mapping string, absent func(string) (*big.Rat, bool)) []NumberReading {
	_ = attribute
	if mapping == "" || absent == nil {
		return rows
	}
	out := make([]NumberReading, len(rows))
	copy(out, rows)
	for i, row := range out {
		// Only a row that certainly holds no value is read under the mapping.
		// A row whose column the query did not return is not absent, it is
		// unknown: nothing is known about it, including whether it is empty.
		// Rewriting that row as the mapped value would decide a rule from a
		// value that was never read (decision section 5: an unread value is
		// undecided, a value that is absent is read under the mapping).
		if row.State != ValueNull {
			continue
		}
		if mapped, ok := absent(mapping); ok {
			out[i] = NumberReading{State: ValuePresent, Value: new(big.Rat).Set(mapped)}
		}
	}
	return out
}

// fillConstant returns one column of the relation in which every row holds the
// same value. A constant input states one value for the whole relation, so the
// operator sees that value once per row.
func fillConstant(width []NumberReading, value *big.Rat) []NumberReading {
	rows := make([]NumberReading, len(width))
	for i := range rows {
		rows[i] = NumberReading{State: ValuePresent, Value: new(big.Rat).Set(value)}
	}
	return rows
}

// firstRowReason reports the first row that holds no value or was not read, in
// read order. It is the reason a row-scoped answer may be undecided; a decided
// answer must not be reported together with it (decideRowScoped checks that).
func firstRowReason(attribute core.CanonicalID, rows []NumberReading) *Undecided {
	for _, row := range rows {
		switch row.State {
		case ValueNull:
			return &Undecided{Reason: UndecidedValueNull, Attribute: attribute}
		case ValueNotFetched:
			return &Undecided{Reason: UndecidedValueNotFetched, Attribute: attribute}
		}
	}
	return nil
}

// decideRelationUnique decides "no two rows share a key".
//
// Two present keys that are equal settle the rule as not satisfied whatever
// else is unknown: no unread row and no empty value can undo a duplicate that
// was seen. Without such a pair, the rule is satisfied only if every row was
// read, at most one row holds no value, and no key is unread beside another
// row.
//
// The rule does not say whether two rows without a value share a key. Read
// either way, one such row shares its key with no other row, so it does not
// keep the rule from being decided; two or more do.
func decideRelationUnique(rule semantic.Validation, source StatefulSource) (Decision, error) {
	reading, err := source.ReadRelationKeys(rule.Relation, rule.Attribute)
	if err != nil {
		return Decision{}, fmt.Errorf("semantic validation %s cannot evaluate relation %s / attribute %s: %w", rule.ID, rule.Relation, rule.Attribute, err)
	}
	if reading.Coverage != CoverageComplete && reading.Coverage != CoveragePartial {
		return Decision{}, fmt.Errorf("semantic validation %s: the reader reports relation %s with the unknown coverage %q", rule.ID, rule.Relation, reading.Coverage)
	}
	seen := make(map[string]struct{}, len(reading.Keys))
	duplicate, notFetched, nulls := false, false, 0
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
				nulls++
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
	case notFetched && len(reading.Keys) > 1:
		// A key that was not read can only collide with another row.
		return undecided(UndecidedRelationKeyNotFetched, rule.Relation, rule.Attribute)
	case reading.Coverage == CoveragePartial:
		return undecided(UndecidedRelationPartiallyRead, rule.Relation, rule.Attribute)
	case nulls > 1:
		// Whether rows without a value count as sharing a key is not stated
		// by the rule, so it is not guessed either way.
		return undecided(UndecidedRelationKeyNull, rule.Relation, rule.Attribute)
	default:
		return decided(ResultSatisfied)
	}
}

// readDate reads one date and refuses a reading that contradicts itself.
func readDate(rule semantic.Validation, source StatefulDateSource, attribute core.CanonicalID) (DateReading, error) {
	reading, err := source.ReadDate(attribute)
	if err != nil {
		return DateReading{}, fmt.Errorf("semantic validation %s cannot evaluate attribute %s: %w", rule.ID, attribute, err)
	}
	switch reading.State {
	case ValuePresent:
		if reading.Value.IsZero() {
			return DateReading{}, fmt.Errorf("semantic validation %s: the reader reports attribute %s as present without a value", rule.ID, attribute)
		}
	case ValueNull, ValueNotFetched:
		if !reading.Value.IsZero() {
			return DateReading{}, fmt.Errorf("semantic validation %s: the reader reports attribute %s as %s together with a value", rule.ID, attribute, reading.State)
		}
	default:
		return DateReading{}, fmt.Errorf("semantic validation %s: the reader reports attribute %s in the unknown state %q", rule.ID, attribute, reading.State)
	}
	return reading, nil
}

// readPresence reads whether one attribute holds a value and refuses a
// reading that contradicts itself.
func readPresence(rule semantic.Validation, source StatefulPresenceSource, attribute core.CanonicalID) (PresenceReading, error) {
	reading, err := source.ReadPresence(attribute)
	if err != nil {
		return PresenceReading{}, fmt.Errorf("semantic validation %s cannot read attribute %s: %w", rule.ID, attribute, err)
	}
	switch reading.State {
	case ValuePresent, ValueNull, ValueNotFetched:
		return reading, nil
	default:
		return PresenceReading{}, fmt.Errorf("semantic validation %s: the reader reports attribute %s in the unknown state %q", rule.ID, attribute, reading.State)
	}
}

// decideRequiredIf decides "Attribute holds a value whenever Compared does".
//
// The rule constrains presence, not the value: any value of Attribute
// satisfies it, including zero and an empty string. It speaks only about
// records that have Compared, so a record without one satisfies it whatever
// Attribute holds. Whether Attribute is required in every record is the
// attribute definition's business and is not inferred here.
//
// Both presence values are read before the rule is decided, so an unread part
// is never turned into a settled answer and a value the caller supplied is
// never left unread. The rule is decided only when the answer is the same
// whatever was not read; otherwise it is undecided, and what reading more
// could settle is reported before what it could not.
func decideRequiredIf(rule semantic.Validation, source StatefulSource) (Decision, error) {
	presence, readsPresence := source.(StatefulPresenceSource)
	if !readsPresence {
		return Decision{}, fmt.Errorf("semantic validation %s is about presence, which this reader cannot read", rule.ID)
	}
	if rule.Compared == "" {
		return Decision{}, fmt.Errorf("semantic validation %s names no attribute that makes %s required", rule.ID, rule.Attribute)
	}
	trigger, err := readPresence(rule, presence, rule.Compared)
	if err != nil {
		return Decision{}, err
	}
	required, err := readPresence(rule, presence, rule.Attribute)
	if err != nil {
		return Decision{}, err
	}

	// A record with a value for Attribute satisfies the rule whatever the
	// trigger turns out to be: no trigger value and a value are both fine.
	if required.State == ValuePresent {
		return decided(ResultSatisfied)
	}
	// A trigger the rule does not speak about satisfies it whatever Attribute
	// holds, including a value that was not read.
	if trigger.State == ValueNull {
		return decided(ResultSatisfied)
	}
	// The trigger speaks about this record and Attribute holds nothing: the
	// business rejection the rule states, settled even if the trigger was not
	// read, because either way the rule applies.
	if trigger.State == ValuePresent && required.State == ValueNull {
		return decided(ResultNotSatisfied)
	}
	// What reading more could settle is reported before what it could not.
	if trigger.State == ValueNotFetched {
		return Decision{Undecided: &Undecided{Reason: UndecidedValueNotFetched, Attribute: rule.Compared}}, nil
	}
	return Decision{Undecided: &Undecided{Reason: UndecidedValueNotFetched, Attribute: rule.Attribute}}, nil
}

// decideNotAfter decides "the date of Attribute is not after the date of
// Compared" for one record.
//
// With IfPresent, a record that holds no Compared date satisfies the rule
// whatever its other date is: the rule speaks only about records that have
// one. In every other case both dates are needed. A date that was not read
// leaves the rule undecided, and so does an empty date the rule does not
// speak about.
func decideNotAfter(rule semantic.Validation, source StatefulSource) (Decision, error) {
	dates, readsDates := source.(StatefulDateSource)
	if !readsDates {
		return Decision{}, fmt.Errorf("semantic validation %s compares calendar dates, which this reader cannot read", rule.ID)
	}
	if rule.Compared == "" {
		return Decision{}, fmt.Errorf("semantic validation %s names no attribute to compare with", rule.ID)
	}
	subject, err := readDate(rule, dates, rule.Attribute)
	if err != nil {
		return Decision{}, err
	}
	compared, err := readDate(rule, dates, rule.Compared)
	if err != nil {
		return Decision{}, err
	}
	switch {
	case compared.State == ValueNull && rule.IfPresent:
		return decided(ResultSatisfied)
	// What reading more could settle is reported before what it could not.
	case subject.State == ValueNotFetched:
		return undecided(UndecidedValueNotFetched, "", rule.Attribute)
	case compared.State == ValueNotFetched:
		return undecided(UndecidedValueNotFetched, "", rule.Compared)
	case subject.State == ValueNull:
		return undecided(UndecidedValueNull, "", rule.Attribute)
	case compared.State == ValueNull:
		return undecided(UndecidedValueNull, "", rule.Compared)
	}
	return decided(satisfiedResult(subject.Value.Compare(compared.Value) <= 0))
}
