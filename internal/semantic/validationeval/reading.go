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

// readAssignedInput reads one input value that was assigned by name in the rule.
// For an attribute reference it reads from the source; for a constant it parses
// the decimal text. The input's absent-as mapping is not yet carried in the
// validation, so absent values are treated as undecided.
//
// It returns (*Undecided, error). A reader error is returned as the error
// (with rule context), not turned into an Undecided, so the caller can
// distinguish "no data" from "the reader failed". A missing or unreadable
// assignment is neither: the caller refuses an unassigned input before asking.
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
		// The rule subject supplies the first input, and only it. A rule that
		// also assigns that input by name binds it twice, so the instruction
		// is refused instead of read.
		for _, assignment := range rule.Inputs {
			if assignment.Name == inputs[0].Name {
				return Decision{}, fmt.Errorf("semantic validation %s assigns operator %s input %q, which the rule subject already supplies", rule.ID, rule.Operator, assignment.Name)
			}
		}
		values := make(map[string]*big.Rat, len(inputs))
		// The first input comes from the attribute the rule constrains.
		value, undecided, err := readNumber(rule, source)
		if err != nil {
			return Decision{}, err
		}
		if undecided != nil {
			return Decision{Undecided: undecided}, nil
		}
		values[inputs[0].Name] = new(big.Rat).Set(value)
		// The subject and every attribute-assigned input must belong to one
		// Read Attempt. Having all three values is not the same as knowing they
		// are one snapshot (section 14.2 B-10), so the origin of each input is
		// collected while the values are, and compared before the operator is
		// asked.
		origins := []inputOrigin{dataOrigin(inputs[0].Name, string(rule.Attribute), source)}
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
		if implementation.Numbers != nil {
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
