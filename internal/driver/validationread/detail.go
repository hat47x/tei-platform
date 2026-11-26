// Package validationread reads rows a Contract Query returned as the input of
// business rules, and says for every value whether it is there and, if not,
// why.
//
// The rows come from the product read path: an Executor ran a Database
// Contract Query and mapped each result column to a semantic attribute. This
// package adds no query, no mapping and no default. It turns what was
// observed into the states the evaluator asks for:
//
//   - an attribute the row holds with a value is present;
//   - an attribute the row holds as nil (SQL NULL) is null;
//   - an attribute the row does not hold was not fetched. The query did not
//     return it, so nothing is known about it, including whether it is empty.
//
// Whether the rows are all the rows of the relation cannot be seen in the
// rows. The caller states it. When nothing is stated, a rule over all the
// rows is refused; the rows are never taken to be complete by default.
package validationread

import (
	"fmt"
	"math/big"
	"sync/atomic"
	"time"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// valueKind is how a value of an attribute is read, decided by the declared
// type of the attribute and never by the Go type a driver happened to return.
type valueKind int

const (
	kindOther valueKind = iota
	kindNumber
	kindDate
)

// Detail is the rows of one relation for one source record.
type Detail struct {
	relation semantic.Relation
	coverage validationeval.Coverage
	rows     []driver.ContractQueryRow
	kinds    map[core.CanonicalID]valueKind
	// attempt identifies the read these rows came from. Every attribute of
	// every row was read in this one attempt, so a rule that reads two
	// attributes of one row reads them from this same attempt. It is opaque:
	// callers compare it for equality only (decision section 14.2 B-10).
	attempt string
}

// FromHeaderDetail reads the detail rows of one header-detail execution: the
// rows of the Pattern's relation for the header record the execution was
// prepared from.
//
// coverage is the caller's statement about those rows. CoverageComplete says
// the query returned every row of the relation for that header;
// CoveragePartial says it may not have (a page, a filter). The empty value
// says nothing, and a rule that needs to know is then refused.
func FromHeaderDetail(semantics []semantic.Namespace, execution driver.HeaderDetailContractQueryExecution, coverage validationeval.Coverage) (*Detail, error) {
	relation := execution.Resolution.Relation
	if relation.Relation.ID == "" || relation.Target.ID == "" {
		return nil, fmt.Errorf("the header-detail execution names no relation; its rows cannot be read as the rows of one")
	}
	if execution.Resolution.Query.Result != relation.Target.ID {
		return nil, fmt.Errorf("Contract Query %s returns %s, but the rows of relation %s are %s",
			execution.Resolution.Query.Name, execution.Resolution.Query.Result, relation.Relation.ID, relation.Target.ID)
	}
	return NewDetail(semantics, relation.Relation, relation.Target, execution.Result.Rows, coverage)
}

// NewDetail reads rows as the rows of relation, whose records are target.
//
// The rows are one read attempt: they were returned by one execution of one
// Contract Query. The attempt is recorded so a rule that reads more than one
// attribute can be refused when those attributes did not come from one read.
func NewDetail(semantics []semantic.Namespace, relation semantic.Relation, target semantic.Data, rows []driver.ContractQueryRow, coverage validationeval.Coverage) (*Detail, error) {
	switch coverage {
	case "", validationeval.CoverageComplete, validationeval.CoveragePartial:
	default:
		return nil, fmt.Errorf("unknown coverage %q; state complete or partial, or state nothing", coverage)
	}
	if relation.Target != target.ID {
		return nil, fmt.Errorf("relation %s leads to %s, not to %s", relation.ID, relation.Target, target.ID)
	}
	types := map[core.CanonicalID]semantic.Type{}
	for _, namespace := range semantics {
		for _, typ := range namespace.Types {
			types[typ.ID] = typ
		}
	}
	kinds := make(map[core.CanonicalID]valueKind, len(target.Attributes))
	for _, attribute := range target.Attributes {
		switch {
		case semantic.IsNumberType(types, attribute.TypeRef):
			kinds[attribute.ID] = kindNumber
		case semantic.IsCalendarDateType(types, attribute.TypeRef):
			kinds[attribute.ID] = kindDate
		default:
			kinds[attribute.ID] = kindOther
		}
	}
	for index, row := range rows {
		for attribute := range row.Attributes {
			if _, known := kinds[attribute]; !known {
				return nil, fmt.Errorf("row %d holds %s, which is not an item of %s", index, attribute, target.ID)
			}
		}
	}
	return &Detail{
		relation: relation,
		coverage: coverage,
		rows:     rows,
		kinds:    kinds,
		attempt:  readAttemptIdentity(),
	}, nil
}

// Len is the number of rows.
func (d *Detail) Len() int { return len(d.rows) }

// Row is the reader of one row, for a rule about one record.
func (d *Detail) Row(index int) (Row, error) {
	if index < 0 || index >= len(d.rows) {
		return Row{}, fmt.Errorf("row %d does not exist; the relation %s was read with %d rows", index, d.relation.ID, len(d.rows))
	}
	return Row{detail: d, index: index}, nil
}

func (d *Detail) kindOf(attribute core.CanonicalID) (valueKind, error) {
	kind, known := d.kinds[attribute]
	if !known {
		return 0, fmt.Errorf("%s is not an item of %s, the rows of relation %s", attribute, d.relation.Target, d.relation.ID)
	}
	return kind, nil
}

func (d *Detail) checkRelation(relation core.CanonicalID) error {
	if relation != d.relation.ID {
		return fmt.Errorf("these are the rows of relation %s, not of %s", d.relation.ID, relation)
	}
	return nil
}

// number reads one numeric value. Exact representations are accepted and
// kept exact. A binary floating point value is refused: the decimal the
// record holds cannot be recovered from it.
func number(value any) (*big.Rat, error) {
	switch v := value.(type) {
	case *big.Rat:
		if v == nil {
			return nil, fmt.Errorf("a nil *big.Rat is not a number")
		}
		return new(big.Rat).Set(v), nil
	case int64:
		return new(big.Rat).SetInt64(v), nil
	case int32:
		return new(big.Rat).SetInt64(int64(v)), nil
	case int:
		return new(big.Rat).SetInt64(int64(v)), nil
	case string:
		return semantic.ParseOperatorCaseValue(v)
	case []byte:
		return semantic.ParseOperatorCaseValue(string(v))
	case float64, float32:
		return nil, fmt.Errorf("a binary floating point value (%v) is not an exact decimal; return the column as text or as an integer", v)
	}
	return nil, fmt.Errorf("a %T is not a number", value)
}

// date reads one calendar date. A time.Time is accepted only when it has no
// time of day: a moment is not a day until a zone is chosen, and none is
// chosen here.
func date(value any) (semantic.CalendarDate, error) {
	switch v := value.(type) {
	case semantic.CalendarDate:
		if v.IsZero() {
			return semantic.CalendarDate{}, fmt.Errorf("the zero calendar date is not a date")
		}
		return v, nil
	case string:
		return semantic.ParseCalendarDate(v)
	case []byte:
		return semantic.ParseCalendarDate(string(v))
	case time.Time:
		if v.Hour() != 0 || v.Minute() != 0 || v.Second() != 0 || v.Nanosecond() != 0 {
			return semantic.CalendarDate{}, fmt.Errorf("%s has a time of day; a calendar date has none", v.Format(time.RFC3339Nano))
		}
		return semantic.ParseCalendarDate(v.Format("2006-01-02"))
	}
	return semantic.CalendarDate{}, fmt.Errorf("a %T is not a calendar date", value)
}

// text reads the value of an attribute as text. A driver may hand back the
// bytes of a column or its text; both are the same text, so neither spelling is
// preferred over the other. A value that is not text at all is refused rather
// than printed, because a rule about text must not be answered from a
// representation the reader invented (section 8.4).
func text(value any) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	}
	return "", fmt.Errorf("a %T is not a text value", value)
}

// truth reads the value of an attribute as a truth. A driver hands back a bool
// for a boolean column, and a text or its bytes for one that spells the value:
// the two spellings the reader accepts are `true` and `false`, and anything else
// is refused rather than read as false, because a rule about truth must not be
// answered from a representation the reader invented (section 8.4).
func truth(value any) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		return parseTruth(v)
	case []byte:
		return parseTruth(string(v))
	}
	return false, fmt.Errorf("a %T is not a truth value", value)
}

// parseTruth reads the two spellings of a truth. An empty text is not false: a
// column that holds no value is a value that is null, which the caller says with
// the state rather than with a value.
func parseTruth(text string) (bool, error) {
	switch text {
	case "true":
		return true, nil
	case "false":
		return false, nil
	}
	return false, fmt.Errorf("%q is not written true or false", text)
}

// anyValue reads a value of an attribute as the type the driver handed it back
// in. The type is what the driver returned, not what the value looks like: a
// driver's `int64` is a number and its `bool` is a truth, so no guessing is
// involved and the value keeps the type it arrived with (section 8.4).
//
// A text is the one case where the driver's own type does not decide, because a
// driver may hand back the text of a column for any of the types. A text that
// reads as a number or as a truth is refused rather than resolved by picking
// one, which is the same refusal section 8.3 makes for an example cell.
func anyValue(value any) (semantic.AnyValue, error) {
	switch v := value.(type) {
	case *big.Rat:
		return semantic.AnyValue{Type: semantic.OperatorInputNumber, Number: v}, nil
	case semantic.CalendarDate:
		if v.IsZero() {
			return semantic.AnyValue{}, fmt.Errorf("the zero calendar date is not a date")
		}
		day := v
		return semantic.AnyValue{Type: semantic.OperatorInputDate, Date: &day}, nil
	case bool:
		return semantic.AnyValue{Type: semantic.OperatorInputTruth, Truth: &v}, nil
	case string, []byte:
		spelled, err := text(value)
		if err != nil {
			return semantic.AnyValue{}, err
		}
		if _, err := semantic.ParseOperatorCaseValue(spelled); err == nil {
			return semantic.AnyValue{}, fmt.Errorf("%q is read as both a text and a number; a value of an `any` input says its type with quotes", spelled)
		}
		if _, err := semantic.ParseOperatorTruthValue(spelled); err == nil {
			return semantic.AnyValue{}, fmt.Errorf("%q is read as both a text and a truth; a value of an `any` input says its type with quotes", spelled)
		}
		return semantic.AnyValue{Type: semantic.OperatorInputText, Text: &spelled}, nil
	}
	return semantic.AnyValue{}, fmt.Errorf("a %T is not a value of any type this program reads", value)
}

// key is what two values of one attribute are compared by. Numbers are
// compared as numbers and dates as dates, so two spellings of one value are
// one key. Anything else is compared as observed, and its Go type is part of
// the key: the text "1" and the integer 1 are not the same value.
func key(kind valueKind, value any) (string, error) {
	switch kind {
	case kindNumber:
		n, err := number(value)
		if err != nil {
			return "", err
		}
		return "number:" + n.RatString(), nil
	case kindDate:
		d, err := date(value)
		if err != nil {
			return "", err
		}
		return "date:" + d.String(), nil
	}
	switch v := value.(type) {
	case string:
		return "text:" + v, nil
	case []byte:
		return "text:" + string(v), nil
	case int64:
		return fmt.Sprintf("integer:%d", v), nil
	case int32:
		return fmt.Sprintf("integer:%d", v), nil
	case int:
		return fmt.Sprintf("integer:%d", v), nil
	case bool:
		return fmt.Sprintf("boolean:%t", v), nil
	}
	return "", fmt.Errorf("a %T cannot be compared for equality here", value)
}

func (d *Detail) readNumber(index int, attribute core.CanonicalID) (validationeval.NumberReading, error) {
	kind, err := d.kindOf(attribute)
	if err != nil {
		return validationeval.NumberReading{}, err
	}
	if kind != kindNumber {
		return validationeval.NumberReading{}, fmt.Errorf("%s is not declared as a number", attribute)
	}
	value, held := d.rows[index].Attributes[attribute]
	switch {
	case !held:
		return validationeval.NumberReading{State: validationeval.ValueNotFetched}, nil
	case value == nil:
		return validationeval.NumberReading{State: validationeval.ValueNull}, nil
	}
	n, err := number(value)
	if err != nil {
		return validationeval.NumberReading{}, fmt.Errorf("row %d, %s: %w", index, attribute, err)
	}
	return validationeval.NumberReading{State: validationeval.ValuePresent, Value: n}, nil
}

func (d *Detail) readDate(index int, attribute core.CanonicalID) (validationeval.DateReading, error) {
	kind, err := d.kindOf(attribute)
	if err != nil {
		return validationeval.DateReading{}, err
	}
	if kind != kindDate {
		return validationeval.DateReading{}, fmt.Errorf("%s is not declared as a calendar date", attribute)
	}
	value, held := d.rows[index].Attributes[attribute]
	switch {
	case !held:
		return validationeval.DateReading{State: validationeval.ValueNotFetched}, nil
	case value == nil:
		return validationeval.DateReading{State: validationeval.ValueNull}, nil
	}
	day, err := date(value)
	if err != nil {
		return validationeval.DateReading{}, fmt.Errorf("row %d, %s: %w", index, attribute, err)
	}
	return validationeval.DateReading{State: validationeval.ValuePresent, Value: day}, nil
}

// readText reads the attribute as a text value. The kind check is the one
// readDate makes: an attribute declared as a calendar date is not read as text,
// because the two ask different questions of the same column (section 7).
func (d *Detail) readText(index int, attribute core.CanonicalID) (validationeval.TextReading, error) {
	kind, err := d.kindOf(attribute)
	if err != nil {
		return validationeval.TextReading{}, err
	}
	if kind == kindDate {
		return validationeval.TextReading{}, fmt.Errorf("%s is declared as a calendar date, which is not read as text", attribute)
	}
	value, held := d.rows[index].Attributes[attribute]
	switch {
	case !held:
		return validationeval.TextReading{State: validationeval.ValueNotFetched}, nil
	case value == nil:
		return validationeval.TextReading{State: validationeval.ValueNull}, nil
	}
	spelled, err := text(value)
	if err != nil {
		return validationeval.TextReading{}, fmt.Errorf("row %d, %s: %w", index, attribute, err)
	}
	return validationeval.TextReading{State: validationeval.ValuePresent, Value: spelled}, nil
}

// readTruth reads the attribute as a truth value. A truth nobody read is not the
// truth false, so the state is carried next to the value.
func (d *Detail) readTruth(index int, attribute core.CanonicalID) (validationeval.TruthReading, error) {
	kind, err := d.kindOf(attribute)
	if err != nil {
		return validationeval.TruthReading{}, err
	}
	if kind == kindDate {
		return validationeval.TruthReading{}, fmt.Errorf("%s is declared as a calendar date, which is not read as truth", attribute)
	}
	value, held := d.rows[index].Attributes[attribute]
	switch {
	case !held:
		return validationeval.TruthReading{State: validationeval.ValueNotFetched}, nil
	case value == nil:
		return validationeval.TruthReading{State: validationeval.ValueNull}, nil
	}
	held2, err := truth(value)
	if err != nil {
		return validationeval.TruthReading{}, fmt.Errorf("row %d, %s: %w", index, attribute, err)
	}
	return validationeval.TruthReading{State: validationeval.ValuePresent, Value: held2}, nil
}

// readAny reads the attribute as a value of the type the driver handed it back
// in. A calendar date asked for as any type is still a calendar date: the kind
// check the other reads make keeps the two questions apart (section 7).
func (d *Detail) readAny(index int, attribute core.CanonicalID) (validationeval.AnyReading, error) {
	kind, err := d.kindOf(attribute)
	if err != nil {
		return validationeval.AnyReading{}, err
	}
	if kind == kindDate {
		return validationeval.AnyReading{}, fmt.Errorf("%s is declared as a calendar date, which is not read as a value of any type", attribute)
	}
	value, held := d.rows[index].Attributes[attribute]
	switch {
	case !held:
		return validationeval.AnyReading{State: validationeval.ValueNotFetched}, nil
	case value == nil:
		return validationeval.AnyReading{State: validationeval.ValueNull}, nil
	}
	any, err := anyValue(value)
	if err != nil {
		return validationeval.AnyReading{}, fmt.Errorf("row %d, %s: %w", index, attribute, err)
	}
	return validationeval.AnyReading{State: validationeval.ValuePresent, Value: any}, nil
}

func (d *Detail) readKey(index int, attribute core.CanonicalID) (validationeval.KeyReading, error) {
	kind, err := d.kindOf(attribute)
	if err != nil {
		return validationeval.KeyReading{}, err
	}
	value, held := d.rows[index].Attributes[attribute]
	switch {
	case !held:
		return validationeval.KeyReading{State: validationeval.ValueNotFetched}, nil
	case value == nil:
		return validationeval.KeyReading{State: validationeval.ValueNull}, nil
	}
	text, err := key(kind, value)
	if err != nil {
		return validationeval.KeyReading{}, fmt.Errorf("row %d, %s: %w", index, attribute, err)
	}
	return validationeval.KeyReading{State: validationeval.ValuePresent, Key: text}, nil
}

// ReadNumber refuses: a Detail is all the rows, and a rule about one record
// is asked through Row.
func (d *Detail) ReadNumber(attribute core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{}, fmt.Errorf("%s was asked of all the rows of relation %s; a rule about one record reads one row", attribute, d.relation.ID)
}

// ReadDate refuses for the same reason as ReadNumber.
func (d *Detail) ReadDate(attribute core.CanonicalID) (validationeval.DateReading, error) {
	return validationeval.DateReading{}, fmt.Errorf("%s was asked of all the rows of relation %s; a rule about one record reads one row", attribute, d.relation.ID)
}

// ReadRelationKeys reads the value of attribute in every row.
// ReadAttempt reports the read these rows came from. Every attribute of every
// row was read in this one execution, so two attributes of one row share it.
func (d *Detail) ReadAttempt(core.CanonicalID) (string, bool) {
	if d.attempt == "" {
		return "", false
	}
	return d.attempt, true
}

func (d *Detail) ReadRelationKeys(relation, attribute core.CanonicalID) (validationeval.RelationReading, error) {
	if err := d.checkRelation(relation); err != nil {
		return validationeval.RelationReading{}, err
	}
	reading := validationeval.RelationReading{Coverage: d.coverage, Keys: make([]validationeval.KeyReading, 0, len(d.rows))}
	for index := range d.rows {
		keyReading, err := d.readKey(index, attribute)
		if err != nil {
			return validationeval.RelationReading{}, err
		}
		reading.Keys = append(reading.Keys, keyReading)
	}
	return reading, nil
}

// ReadRelationNumbers reads one numeric attribute in every row of the relation,
// in the order the rows were read, together with whether those rows are all the
// rows. It is the row-scoped reader decision section 15.1 fixes: one call is one
// reading, so every row-scoped input of one rule comes from this same attempt.
func (d *Detail) ReadRelationNumbers(relation, attribute core.CanonicalID) (validationeval.NumberRowsReading, error) {
	if err := d.checkRelation(relation); err != nil {
		return validationeval.NumberRowsReading{}, err
	}
	reading := validationeval.NumberRowsReading{
		Coverage: d.coverage,
		Rows:     make([]validationeval.NumberReading, 0, len(d.rows)),
	}
	for index := range d.rows {
		row, err := d.readNumber(index, attribute)
		if err != nil {
			// A row this reader cannot turn into a reading is the reader's
			// error, not a value that was absent: it is returned rather than
			// recorded as "no value", which would answer a question about the
			// data with a statement about the read.
			return validationeval.NumberRowsReading{}, err
		}
		reading.Rows = append(reading.Rows, row)
	}
	return reading, nil
}

// ReadRelationValues reads one attribute in every row of the relation as a value
// of the type the value itself states, in the order the rows were read, together
// with whether those rows are all the rows.
//
// It is the typed counterpart of ReadRelationNumbers (decision section 8.4): a
// calendar date is read as a day, and a value of any other type is read as the
// type the driver handed it back in, so the kind the reader saw survives the trip
// to the implementation. A number is read as a number here too, so a rule that
// declares a typed input over every row is answered from the same read a
// one-record rule would be.
func (d *Detail) ReadRelationValues(relation, attribute core.CanonicalID) (validationeval.ValueRowsReading, error) {
	if err := d.checkRelation(relation); err != nil {
		return validationeval.ValueRowsReading{}, err
	}
	reading := validationeval.ValueRowsReading{
		Coverage: d.coverage,
		Rows:     make([]validationeval.RowValue, 0, len(d.rows)),
	}
	for index := range d.rows {
		row, err := d.readRowValue(index, attribute)
		if err != nil {
			// A row this reader cannot turn into a reading is the reader's
			// error, not a value that was absent: it is returned rather than
			// recorded as "no value", which would answer a question about the
			// data with a statement about the read.
			return validationeval.ValueRowsReading{}, err
		}
		reading.Rows = append(reading.Rows, row)
	}
	return reading, nil
}

// readRowValue reads one attribute of one row as the value of the type it
// states. The state is carried next to the value, so a row that holds no value
// and a row the query did not return stay apart rather than both becoming the
// zero of their type (decision section 5).
func (d *Detail) readRowValue(index int, attribute core.CanonicalID) (validationeval.RowValue, error) {
	if _, err := d.kindOf(attribute); err != nil {
		return validationeval.RowValue{}, err
	}
	value, held := d.rows[index].Attributes[attribute]
	switch {
	case !held:
		return validationeval.RowValue{State: validationeval.ValueNotFetched}, nil
	case value == nil:
		return validationeval.RowValue{State: validationeval.ValueNull}, nil
	}
	any, err := anyValue(value)
	if err != nil {
		return validationeval.RowValue{}, fmt.Errorf("row %d, %s: %w", index, attribute, err)
	}
	return validationeval.RowValue{State: validationeval.ValuePresent, Value: semantic.OperatorValue{Any: &any}}, nil
}

// ReadRelationPeriods reads the period of every row and, when group is not
// empty, the value rows are grouped by.
func (d *Detail) ReadRelationPeriods(relation, start, end, group core.CanonicalID) (validationeval.PeriodRowsReading, error) {
	if err := d.checkRelation(relation); err != nil {
		return validationeval.PeriodRowsReading{}, err
	}
	reading := validationeval.PeriodRowsReading{Coverage: d.coverage, Rows: make([]validationeval.PeriodRow, 0, len(d.rows))}
	for index := range d.rows {
		var row validationeval.PeriodRow
		var err error
		if row.Start, err = d.readDate(index, start); err != nil {
			return validationeval.PeriodRowsReading{}, err
		}
		if row.End, err = d.readDate(index, end); err != nil {
			return validationeval.PeriodRowsReading{}, err
		}
		if group != "" {
			if row.Group, err = d.readKey(index, group); err != nil {
				return validationeval.PeriodRowsReading{}, err
			}
		}
		reading.Rows = append(reading.Rows, row)
	}
	return reading, nil
}

// Row reads one row of a Detail.
type Row struct {
	detail *Detail
	index  int
}

// ReadAttempt reports the attempt the row was read in. One row of a Detail is
// one row of the same read, so every attribute of it shares that attempt.
func (r Row) ReadAttempt(attribute core.CanonicalID) (string, bool) {
	return validationeval.AttributedSource(r.detail).ReadAttempt(attribute)
}

func (r Row) ReadNumber(attribute core.CanonicalID) (validationeval.NumberReading, error) {
	return r.detail.readNumber(r.index, attribute)
}

func (r Row) ReadDate(attribute core.CanonicalID) (validationeval.DateReading, error) {
	return r.detail.readDate(r.index, attribute)
}

// ReadRelationNumbers refuses for the same reason as ReadRelationKeys.
func (r Row) ReadRelationNumbers(relation, attribute core.CanonicalID) (validationeval.NumberRowsReading, error) {
	return validationeval.NumberRowsReading{}, fmt.Errorf(
		"one row is not the rows of relation %s; a row-scoped input needs the reader of the whole relation", relation)
}

// ReadText and ReadTruth are the reader side of the two typed inputs: a rule
// whose operator declares a text or a truth input is answered from the value of
// that type, never from the number of the same column (section 8.4).
func (r Row) ReadText(attribute core.CanonicalID) (validationeval.TextReading, error) {
	return r.detail.readText(r.index, attribute)
}

func (r Row) ReadTruth(attribute core.CanonicalID) (validationeval.TruthReading, error) {
	return r.detail.readTruth(r.index, attribute)
}

// ReadAny reads the attribute as a value of the type the driver handed it back
// in, which is the type the value carries to the implementation (section 8.4).
func (r Row) ReadAny(attribute core.CanonicalID) (validationeval.AnyReading, error) {
	return r.detail.readAny(r.index, attribute)
}

// ReadRelationKeys refuses: one row is not the rows of a relation.
func (r Row) ReadRelationKeys(relation, attribute core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, fmt.Errorf("relation %s was asked of one row; a rule about all the rows reads the Detail", relation)
}

// readAttemptIdentity names one read of these rows. The value is opaque and is
// compared for equality only.
//
// It is derived from the read itself, not from the rows. Hashing the rows made
// the identity depend on Go map iteration order: twelve construction calls over
// the SAME rows produced three distinct identities, because range over
// row.Attributes is not ordered. An identity that changes between two views of
// one read cannot answer the question it exists for, "did these two values come
// from the same read?" (#4359, decision section 14.2 B-10).
//
// The attempt is therefore numbered per Detail from a process-wide counter. Two
// Details never share a number, so a value taken from another read is a stated
// mismatch rather than a coincidence; and one Detail keeps its number, so
// comparing two inputs of one decision is stable.
func readAttemptIdentity() string {
	return fmt.Sprintf("read:%d", atomic.AddUint64(&readAttemptCounter, 1))
}

var readAttemptCounter uint64
