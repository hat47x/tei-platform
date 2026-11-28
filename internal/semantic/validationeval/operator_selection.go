package validationeval

import (
	"fmt"
	"math/big"
	"sort"
	"strings"

	"tei/internal/core"
	"tei/internal/semantic"
)

// This file connects a declared operator to the implementation a composition
// selected for it.
//
// Three things stay separate. A document declares the operator (what it is
// called and what it reads). A composition, such as a Workspace manifest,
// selects an implementation for it by name. The running program carries
// implementations under those names. An operator rule is decided only when
// the selection names exactly one implementation the program carries.
//
// The implementations compiled into this binary are generic checks on one
// number. They are not a plugin registry; an implementation supplied by a
// plugin has to arrive through the existing plugin selection and lock, and
// then enters ResolveImplementations as one more NamedImplementation.

// NamedImplementation is an implementation the running program carries, under
// the name a composition selects it by.
type NamedImplementation struct {
	ID     string
	Input  semantic.OperatorInput
	Number func(value *big.Rat) (bool, error)
	// Numbers decides for a multi-input operator. Exactly one of Number and
	// Numbers is non-nil.
	Numbers func(values map[string]*big.Rat) (bool, error)
	// NumberRows decides for an operator whose input reads every row of a
	// relation. At most one of Number, Numbers, NumberRows, Values and
	// ValueRows is non-nil.
	NumberRows func(rows map[string][]NumberReading) (Tri, error)
	// ValueRows decides for an operator whose input reads every row of a
	// relation in a type other than number. It receives, per input name, the
	// rows that were read in the order they were read, each value carrying the
	// type it was read as (section 8.4). At most one of Number, Numbers,
	// NumberRows, Values and ValueRows is non-nil.
	ValueRows func(rows map[string][]RowValueReading) (Tri, error)
	// Values decides for an operator whose inputs are of any type. It receives
	// one value per input name, each carrying the type it was read as
	// (section 8.4), so a day is decided by the same field that decides a
	// number and a text.
	Values func(values map[string]semantic.OperatorValue) (bool, error)
	// Absent reads an absent value under the mapping the rule named, or
	// reports that this implementation cannot read that mapping. A shipped
	// implementation that states none leaves every absence undecided.
	Absent func(mapping string) (*big.Rat, bool)
}

// ImplementationSelection is one choice of a composition: this operator is
// decided by that implementation.
type ImplementationSelection struct {
	Operator       core.CanonicalID `json:"operator"`
	Implementation string           `json:"implementation"`
}

// ImplementationCall records that a selected implementation was actually
// asked to decide. Being selected and being called are different facts.
type ImplementationCall struct {
	Operator       core.CanonicalID
	Implementation string
}

// BuiltinImplementations lists the implementations compiled into this binary.
// It is a function so no caller can mutate a shared table.
func BuiltinImplementations() []NamedImplementation {
	return []NamedImplementation{
		{
			ID:    "tei.number.non-negative",
			Input: semantic.OperatorInputNumber,
			Number: func(value *big.Rat) (bool, error) {
				return value.Sign() >= 0, nil
			},
		},
		{
			ID:    "tei.number.non-zero",
			Input: semantic.OperatorInputNumber,
			Number: func(value *big.Rat) (bool, error) {
				return value.Sign() != 0, nil
			},
		},
		{
			// tei.number.within decides "low <= value <= high" by input name.
			// It reads inputs named value, low, and high; a declaration that
			// names them otherwise is not this operator and is reported.
			ID:    "tei.number.within",
			Input: semantic.OperatorInputNumber,
			Numbers: func(values map[string]*big.Rat) (bool, error) {
				v, ok := values["value"]
				if !ok {
					return false, fmt.Errorf("within reads an input named \"value\"; the rule supplied no value input")
				}
				low, ok := values["low"]
				if !ok {
					return false, fmt.Errorf("within reads an input named \"low\"; the rule supplied no low input")
				}
				high, ok := values["high"]
				if !ok {
					return false, fmt.Errorf("within reads an input named \"high\"; the rule supplied no high input")
				}
				// A range whose lower bound is above its upper bound is a
				// wrong range specification, not an out-of-range value. It is
				// reported and never answered as not-satisfied; low and high
				// are not swapped to make it look decidable.
				if low.Cmp(high) > 0 {
					return false, fmt.Errorf("within: low (%s) is greater than high (%s); the range specification is wrong", low.RatString(), high.RatString())
				}
				return v.Cmp(low) >= 0 && v.Cmp(high) <= 0, nil
			},
			// A bound that holds no value is read as zero only when the
			// declaration states `absent as zero` and the rule names it. Any
			// other mapping is refused, so an absence keeps its own reason
			// rather than being read as a value this implementation invented.
			Absent: func(mapping string) (*big.Rat, bool) {
				if mapping != "zero" {
					return nil, false
				}
				return new(big.Rat), true
			},
		},
		{
			// tei.number.every-at-most decides "every row's value is at most
			// its limit". Both inputs are columns of the relation's rows: a
			// row-scoped input reads a column, never one constant, so the
			// comparison is row against row and a single row above its own
			// limit is the counterexample the declaration's `refutes on some
			// rows` is about.
			//
			// A row whose value or limit was not read answers TriMaybe, so
			// the evaluator reports that row's own reason instead of reading
			// it as zero.
			ID:    "tei.number.every-at-most",
			Input: semantic.OperatorInputNumber,
			NumberRows: func(rows map[string][]NumberReading) (Tri, error) {
				values, ok := rows["value"]
				if !ok {
					return TriMaybe, fmt.Errorf("every-at-most reads an input named \"value\"; the rule supplied no value input")
				}
				limits, ok := rows["limit"]
				if !ok {
					return TriMaybe, fmt.Errorf("every-at-most reads an input named \"limit\"; the rule supplied no limit input")
				}
				if len(values) != len(limits) {
					// Both inputs are columns of the same rows, so they must
					// be the same length. Comparing a short column against a
					// long one would compare rows that do not belong together.
					return TriMaybe, fmt.Errorf("every-at-most reads %d value rows and %d limit rows; both are columns of the same rows", len(values), len(limits))
				}
				// A row that cannot be compared leaves the answer undecided
				// only when no other row already refutes the contract. A
				// witnessed counterexample is a counterexample however many
				// rows around it were not read, so the rows are all looked at
				// before a row that cannot be compared decides the answer:
				// stopping at the first such row made the answer depend on the
				// order the rows happened to be read in (decision section 8).
				// A row the reader reports as present without a value is a
				// contract violation, not a value that was absent. It is
				// checked for every row before any row is compared: a known
				// counterexample must not hide a violation the reader made
				// (decision section 8.1).
				for index := range values {
					if values[index].State == ValuePresent && values[index].Value == nil {
						return TriMaybe, fmt.Errorf("every-at-most reads row %d as present without a value", index)
					}
					if limits[index].State == ValuePresent && limits[index].Value == nil {
						return TriMaybe, fmt.Errorf("every-at-most reads row %d as present without a limit", index)
					}
				}
				unreadable := false
				for index := range values {
					value, limit := values[index], limits[index]
					if value.State != ValuePresent || limit.State != ValuePresent {
						unreadable = true
						continue
					}
					if value.Value.Cmp(limit.Value) > 0 {
						return TriNo, nil
					}
				}
				if unreadable {
					return TriMaybe, nil
				}
				return TriYes, nil
			},
			// A declaration may state how an absent value is read, and a rule
			// may name that mapping. This implementation reads the one
			// mapping it knows: `zero`, which the author states on the
			// declaration. An absence is then a value of zero rather than
			// something this implementation invented, and a mapping it does
			// not know leaves the row's own reason (decision section 5).
			Absent: func(mapping string) (*big.Rat, bool) {
				if mapping != "zero" {
					return nil, false
				}
				return new(big.Rat), true
			},
		},
		{
			// tei.day.not-before decides "the value's day is not before the
			// deadline's day". Both inputs are calendar days, and a day is
			// read as a day: the reader reports which day each attribute
			// holds, and this implementation compares the two. A day is never
			// read through the decimal reader, where `2026-04-01` would not be
			// a value anyone wrote.
			//
			// The two days are the same day when the rule's subject and its
			// deadline name the same attribute; the answer is then that the day
			// is not before itself, which holds.
			ID:    "tei.day.not-before",
			Input: semantic.OperatorInputDate,
			Values: func(values map[string]semantic.OperatorValue) (bool, error) {
				value, ok := values["value"]
				if !ok || value.Date == nil {
					return false, fmt.Errorf("not-before reads an input named \"value\" as a calendar day; the rule supplied no day")
				}
				deadline, ok := values["deadline"]
				if !ok || deadline.Date == nil {
					return false, fmt.Errorf("not-before reads an input named \"deadline\" as a calendar day; the rule supplied no day")
				}
				return value.Date.Compare(*deadline.Date) >= 0, nil
			},
		},
		{
			// tei.day.every-not-before decides "every row's day is not before
			// the deadline's day". It is the row form of tei.day.not-before:
			// the same two days, asked of every row of a relation instead of
			// of one record. Both inputs are calendar days, and a day is read
			// as a day: the reader reports which day each attribute holds, and
			// this implementation compares the two. A day is never read through
			// the decimal reader, where `2026-04-01` would not be a value
			// anyone wrote.
			//
			// The deadline may be a column of the rows or one constant for the
			// whole relation; either way every row is compared against its own
			// deadline, and one row before its own deadline is the
			// counterexample the declaration's `refutes on some rows` is about.
			//
			// A row whose day or deadline was not read, or holds no value,
			// answers TriMaybe, so the evaluator reports that row's own reason
			// instead of reading it as a day this implementation invented. The
			// rows are all looked at before such a row decides the answer: a
			// counterexample that was seen is a counterexample however many
			// rows around it were not read (decision section 8).
			ID:    "tei.day.every-not-before",
			Input: semantic.OperatorInputDate,
			ValueRows: func(rows map[string][]RowValueReading) (Tri, error) {
				values, ok := rows["value"]
				if !ok {
					return TriMaybe, fmt.Errorf("every-not-before reads an input named \"value\"; the rule supplied no value input")
				}
				deadlines, ok := rows["deadline"]
				if !ok {
					return TriMaybe, fmt.Errorf("every-not-before reads an input named \"deadline\"; the rule supplied no deadline input")
				}
				if len(values) != len(deadlines) {
					// Both inputs are columns of the same rows, so they must be
					// the same length. Comparing a short column against a long
					// one would compare rows that do not belong together.
					return TriMaybe, fmt.Errorf("every-not-before reads %d value rows and %d deadline rows; both are columns of the same rows", len(values), len(deadlines))
				}
				// A row the reader reports as present without a value is a
				// contract violation, not a value that was absent. It is checked
				// for every row before any row is compared: a known
				// counterexample must not hide a violation the reader made
				// (decision section 8.1).
				for index := range values {
					if values[index].State == ValuePresent && values[index].Value.Kind() != semantic.OperatorInputDate {
						return TriMaybe, fmt.Errorf("every-not-before reads row %d as present without a day", index)
					}
					if deadlines[index].State == ValuePresent && deadlines[index].Value.Kind() != semantic.OperatorInputDate {
						return TriMaybe, fmt.Errorf("every-not-before reads row %d as present without a deadline day", index)
					}
				}
				unreadable := false
				for index := range values {
					value, deadline := values[index], deadlines[index]
					if value.State != ValuePresent || deadline.State != ValuePresent {
						unreadable = true
						continue
					}
					if value.Value.Date.Compare(*deadline.Value.Date) < 0 {
						return TriNo, nil
					}
				}
				if unreadable {
					return TriMaybe, nil
				}
				return TriYes, nil
			},
		},
		{
			// tei.text.every-same-as decides "every row's text is exactly the
			// other text". It is the row form of tei.text.same-as: the same two
			// texts, asked of every row of a relation instead of of one record.
			// Both inputs are texts, and a text is read as a text: the reader
			// reports what each attribute holds, and this implementation
			// compares the two. A text is never read as a number, so the text
			// "7" and the number 7 are not the same value and are never
			// compared.
			//
			// One row whose two texts differ is the counterexample the
			// declaration's `refutes on some rows` is about. A row whose texts
			// were not read, or holds no text, answers TriMaybe, so the evaluator
			// reports that row's own reason instead of reading it as a text this
			// implementation invented. The rows are all looked at before such a
			// row decides the answer: a counterexample that was seen is a
			// counterexample however many rows around it were not read (decision
			// section 8).
			ID:    "tei.text.every-same-as",
			Input: semantic.OperatorInputText,
			ValueRows: func(rows map[string][]RowValueReading) (Tri, error) {
				values, ok := rows["value"]
				if !ok {
					return TriMaybe, fmt.Errorf("every-same-as reads an input named \"value\"; the rule supplied no value input")
				}
				others, ok := rows["other"]
				if !ok {
					return TriMaybe, fmt.Errorf("every-same-as reads an input named \"other\"; the rule supplied no other input")
				}
				if len(values) != len(others) {
					return TriMaybe, fmt.Errorf("every-same-as reads %d value rows and %d other rows; both are columns of the same rows", len(values), len(others))
				}
				// A row the reader reports as present without a text is a
				// contract violation, not a text that was absent. It is checked
				// for every row before any row is compared: a known
				// counterexample must not hide a violation the reader made
				// (decision section 8.1).
				for index := range values {
					if values[index].State == ValuePresent && values[index].Value.Kind() != semantic.OperatorInputText {
						return TriMaybe, fmt.Errorf("every-same-as reads row %d as present without a text", index)
					}
					if others[index].State == ValuePresent && others[index].Value.Kind() != semantic.OperatorInputText {
						return TriMaybe, fmt.Errorf("every-same-as reads row %d as present without an other text", index)
					}
				}
				unreadable := false
				for index := range values {
					value, other := values[index], others[index]
					if value.State != ValuePresent || other.State != ValuePresent {
						unreadable = true
						continue
					}
					if *value.Value.Text != *other.Value.Text {
						return TriNo, nil
					}
				}
				if unreadable {
					return TriMaybe, nil
				}
				return TriYes, nil
			},
		},
		{
			// tei.truth.every-is-true decides "every row's truth holds". It is
			// the row form of tei.truth.is-true: the same truth, asked of every
			// row of a relation instead of of one record. A truth is read as a
			// truth, never as the number 1, which would answer a different
			// question.
			//
			// One row that holds false is the counterexample the declaration's
			// `refutes on some rows` is about. A row whose truth was not read, or
			// holds no truth, answers TriMaybe, so the evaluator reports that
			// row's own reason.
			ID:    "tei.truth.every-is-true",
			Input: semantic.OperatorInputTruth,
			ValueRows: func(rows map[string][]RowValueReading) (Tri, error) {
				values, ok := rows["value"]
				if !ok {
					return TriMaybe, fmt.Errorf("every-is-true reads an input named \"value\"; the rule supplied no value input")
				}
				for index := range values {
					if values[index].State == ValuePresent && values[index].Value.Kind() != semantic.OperatorInputTruth {
						return TriMaybe, fmt.Errorf("every-is-true reads row %d as present without a truth", index)
					}
				}
				unreadable := false
				for index := range values {
					if values[index].State != ValuePresent {
						unreadable = true
						continue
					}
					if !*values[index].Value.Truth {
						return TriNo, nil
					}
				}
				if unreadable {
					return TriMaybe, nil
				}
				return TriYes, nil
			},
		},
		{
			// tei.any.every-same-as decides "every row's value is exactly the
			// other value". It is the row form of tei.any.same-as, and it keeps
			// the same rule about kinds: an `any` value carries the type it was
			// read as, so a number and a text that spell the same characters are
			// different values and are never the same.
			//
			// One row whose two values differ is the counterexample. A row whose
			// value was not read answers TriMaybe, so the evaluator reports that
			// row's own reason.
			ID:    "tei.any.every-same-as",
			Input: semantic.OperatorInputAny,
			ValueRows: func(rows map[string][]RowValueReading) (Tri, error) {
				values, ok := rows["value"]
				if !ok {
					return TriMaybe, fmt.Errorf("every-same-as reads an input named \"value\"; the rule supplied no value input")
				}
				others, ok := rows["other"]
				if !ok {
					return TriMaybe, fmt.Errorf("every-same-as reads an input named \"other\"; the rule supplied no other input")
				}
				if len(values) != len(others) {
					return TriMaybe, fmt.Errorf("every-same-as reads %d value rows and %d other rows; both are columns of the same rows", len(values), len(others))
				}
				for index := range values {
					if values[index].State == ValuePresent && values[index].Value.Kind() != semantic.OperatorInputAny {
						return TriMaybe, fmt.Errorf("every-same-as reads row %d as present without a value", index)
					}
					if others[index].State == ValuePresent && others[index].Value.Kind() != semantic.OperatorInputAny {
						return TriMaybe, fmt.Errorf("every-same-as reads row %d as present without an other value", index)
					}
				}
				unreadable := false
				for index := range values {
					value, other := values[index], others[index]
					if value.State != ValuePresent || other.State != ValuePresent {
						unreadable = true
						continue
					}
					same, err := sameAnyOperatorValue(value.Value, other.Value)
					if err != nil {
						return TriMaybe, err
					}
					if !same {
						return TriNo, nil
					}
				}
				if unreadable {
					return TriMaybe, nil
				}
				return TriYes, nil
			},
		},
		{
			// tei.text.same-as decides "the text holds exactly the other text".
			// Both inputs are texts, and a text is read as a text: the reader
			// reports what each attribute holds, and this implementation
			// compares the two. A text is never read as a number, so the text
			// "7" and the number 7 are not the same value and are never
			// compared.
			//
			// Two spellings of one text are one text, because the reader
			// reports the text and not its spelling. The empty text is a value
			// of length zero, so a record that holds it and a record that holds
			// no text are different states and only the first is asked about
			// here (decision section 8.3).
			ID:    "tei.text.same-as",
			Input: semantic.OperatorInputText,
			Values: func(values map[string]semantic.OperatorValue) (bool, error) {
				value, ok := values["value"]
				if !ok {
					return false, fmt.Errorf("same-as reads an input named \"value\"; the rule supplied no value input")
				}
				other, ok := values["other"]
				if !ok {
					return false, fmt.Errorf("same-as reads an input named \"other\"; the rule supplied no other input")
				}
				if value.Text == nil || other.Text == nil {
					return false, fmt.Errorf("same-as reads two texts; one of its inputs was not read as a text")
				}
				return *value.Text == *other.Text, nil
			},
		},
		{
			// tei.truth.is-true decides "the truth holds". A truth is read as a
			// truth: the reader reports whether the attribute holds true, and
			// this implementation answers that. A truth is never read as the
			// number 1, which would answer a different question.
			ID:    "tei.truth.is-true",
			Input: semantic.OperatorInputTruth,
			Values: func(values map[string]semantic.OperatorValue) (bool, error) {
				value, ok := values["value"]
				if !ok {
					return false, fmt.Errorf("is-true reads an input named \"value\"; the rule supplied no value input")
				}
				if value.Truth == nil {
					return false, fmt.Errorf("is-true reads a truth; its input was not read as a truth")
				}
				return *value.Truth, nil
			},
		},
		{
			// tei.any.same-as decides "the value holds exactly the other
			// value", for inputs declared `any`. A value of an `any` input
			// keeps the type it was read as, so the comparison reads the kind
			// from there: a number and a text that spell the same characters
			// are different values and are never the same.
			ID:    "tei.any.same-as",
			Input: semantic.OperatorInputAny,
			Values: func(values map[string]semantic.OperatorValue) (bool, error) {
				return anySameAs(values)
			},
		},
		{
			// tei.presence.holds decides "the record holds a value". It is the
			// presence question, not a question about a value: a number of 0
			// and an empty text are values, and only a record that holds none
			// answers false. The reader reports which of the two it is, so this
			// implementation never reads an attribute to decide it.
			ID:    "tei.presence.holds",
			Input: semantic.OperatorInputPresence,
			Values: func(values map[string]semantic.OperatorValue) (bool, error) {
				value, ok := values["value"]
				if !ok {
					return false, fmt.Errorf("holds reads an input named \"value\"; the rule supplied no value input")
				}
				if value.Presence == nil {
					return false, fmt.Errorf("holds reads whether a value is there; its input was not read as presence")
				}
				return *value.Presence, nil
			},
		},
		{
			// tei.presence.absent decides "the record holds no value", the
			// other half of the same question. It exists so a rule can be
			// written that requires an attribute to be empty, which `holds`
			// alone cannot state.
			ID:    "tei.presence.absent",
			Input: semantic.OperatorInputPresence,
			Values: func(values map[string]semantic.OperatorValue) (bool, error) {
				value, ok := values["value"]
				if !ok {
					return false, fmt.Errorf("absent reads an input named \"value\"; the rule supplied no value input")
				}
				if value.Presence == nil {
					return false, fmt.Errorf("absent reads whether a value is there; its input was not read as presence")
				}
				return !*value.Presence, nil
			},
		},
	}
}

// ResolveImplementations turns the selections of a composition into operator
// implementations. Every selection must name exactly one available
// implementation, and one operator may be selected only once. Nothing is
// chosen by position, and an implementation nobody selected is not returned.
//
// called, when not nil, receives one entry each time a returned implementation
// is asked to decide. It observes the call; it cannot change the result.
func ResolveImplementations(selected []ImplementationSelection, available []NamedImplementation, called func(ImplementationCall)) ([]OperatorImplementation, error) {
	seen := map[core.CanonicalID]bool{}
	out := make([]OperatorImplementation, 0, len(selected))
	for _, selection := range selected {
		if selection.Operator == "" || selection.Implementation == "" {
			return nil, fmt.Errorf("TEI_OPERATOR_IMPLEMENTATION_INVALID: a selection needs an operator and an implementation")
		}
		if seen[selection.Operator] {
			return nil, fmt.Errorf("TEI_OPERATOR_IMPLEMENTATION_AMBIGUOUS: operator %s has more than one selected implementation", selection.Operator)
		}
		seen[selection.Operator] = true

		var matches []NamedImplementation
		for _, candidate := range available {
			if candidate.ID == selection.Implementation {
				matches = append(matches, candidate)
			}
		}
		switch {
		case len(matches) == 0:
			return nil, fmt.Errorf("TEI_OPERATOR_IMPLEMENTATION_UNAVAILABLE: operator %s selects implementation %q, which this tei does not carry (available: %s)",
				selection.Operator, selection.Implementation, availableImplementationNames(available))
		case len(matches) > 1:
			return nil, fmt.Errorf("TEI_OPERATOR_IMPLEMENTATION_AMBIGUOUS: implementation %q exists %d times; exactly one must be available",
				selection.Implementation, len(matches))
		}
		match := matches[0]
		implementation := OperatorImplementation{Operator: selection.Operator, Input: match.Input}
		if match.Number != nil {
			call := ImplementationCall{Operator: selection.Operator, Implementation: match.ID}
			decide := match.Number
			implementation.Number = func(value *big.Rat) (bool, error) {
				if called != nil {
					called(call)
				}
				return decide(value)
			}
		}
		if match.Numbers != nil {
			call := ImplementationCall{Operator: selection.Operator, Implementation: match.ID}
			decide := match.Numbers
			implementation.Numbers = func(values map[string]*big.Rat) (bool, error) {
				if called != nil {
					called(call)
				}
				return decide(values)
			}
		}
		if match.NumberRows != nil {
			call := ImplementationCall{Operator: selection.Operator, Implementation: match.ID}
			decide := match.NumberRows
			implementation.NumberRows = func(rows map[string][]NumberReading) (Tri, error) {
				if called != nil {
					called(call)
				}
				return decide(rows)
			}
		}
		if match.Values != nil {
			// A value of any type is decided by the one field that carries the
			// type it was read as (section 8.4). A day, a number and a text are
			// all decided by it, so there is no second field a rule could be
			// answered from.
			call := ImplementationCall{Operator: selection.Operator, Implementation: match.ID}
			decide := match.Values
			implementation.Values = func(values map[string]semantic.OperatorValue) (bool, error) {
				if called != nil {
					called(call)
				}
				return decide(values)
			}
		}
		if match.ValueRows != nil {
			// A row of any type is decided by the one field that carries the
			// type it was read as (section 8.4), exactly as a value of one
			// record is: a day, a number and a text are all decided by it, so
			// there is no second field a rule could be answered from.
			call := ImplementationCall{Operator: selection.Operator, Implementation: match.ID}
			decide := match.ValueRows
			implementation.ValueRows = func(rows map[string][]RowValueReading) (Tri, error) {
				if called != nil {
					called(call)
				}
				return decide(rows)
			}
		}
		// A shipped implementation may know how to read an absent value under
		// one named mapping; the rule names which mapping it wants, so the
		// reader is carried as it is.
		implementation.Absent = match.Absent
		out = append(out, implementation)
	}
	return out, nil
}

func availableImplementationNames(available []NamedImplementation) string {
	seen := map[string]bool{}
	var names []string
	for _, candidate := range available {
		if !seen[candidate.ID] {
			seen[candidate.ID] = true
			names = append(names, candidate.ID)
		}
	}
	if len(names) == 0 {
		return "none"
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
