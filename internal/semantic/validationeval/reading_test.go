package validationeval_test

import (
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// These tests check Decide against expected values that do not come from it.
//
// For a number, the expected values are written by hand. For a relation, the
// expected value is computed by a second, slower definition: write out every
// way the unread part could turn out, decide "no two rows share a key" for
// each, and call the rule decided only if all of them agree.

type statefulSource struct {
	number   validationeval.NumberReading
	relation validationeval.RelationReading
	err      error
	calls    []string
}

func (s *statefulSource) ReadNumber(attribute core.CanonicalID) (validationeval.NumberReading, error) {
	s.calls = append(s.calls, "ReadNumber("+string(attribute)+")")
	return s.number, s.err
}

func (s *statefulSource) ReadRelationKeys(relation, attribute core.CanonicalID) (validationeval.RelationReading, error) {
	s.calls = append(s.calls, "ReadRelationKeys("+string(relation)+","+string(attribute)+")")
	return s.relation, s.err
}

var quantityRule = semantic.Validation{
	ID: "sales.order_line.quantity_positive", Symbol: "quantity_positive",
	Kind: semantic.ValidationPositive, Attribute: "sales.order_line.quantity",
}

var lineNumberRule = semantic.Validation{
	ID: "sales.order.lines.line_number_unique", Symbol: "line_number_unique",
	Kind: semantic.ValidationRelationUnique, Attribute: "sales.order_line.line_number", Relation: "sales.order.lines",
}

func rat(text string) *big.Rat {
	value, ok := new(big.Rat).SetString(text)
	if !ok {
		panic(text)
	}
	return value
}

func TestANumberIsDecidedOnlyWhenItIsPresent(t *testing.T) {
	type want struct {
		result validationeval.Result
		reason validationeval.UndecidedReason
	}
	cases := []struct {
		name    string
		reading validationeval.NumberReading
		want    want
	}{
		{"present and positive", validationeval.NumberReading{State: validationeval.ValuePresent, Value: rat("2.5")}, want{result: validationeval.ResultSatisfied}},
		{"present and zero", validationeval.NumberReading{State: validationeval.ValuePresent, Value: rat("0")}, want{result: validationeval.ResultNotSatisfied}},
		{"present and negative", validationeval.NumberReading{State: validationeval.ValuePresent, Value: rat("-1")}, want{result: validationeval.ResultNotSatisfied}},
		{"the record holds no value", validationeval.NumberReading{State: validationeval.ValueNull}, want{reason: validationeval.UndecidedValueNull}},
		{"the attribute was not read", validationeval.NumberReading{State: validationeval.ValueNotFetched}, want{reason: validationeval.UndecidedValueNotFetched}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := &statefulSource{number: tc.reading}
			got, err := validationeval.Decide(quantityRule, source, validationeval.Operators{})
			if err != nil {
				t.Fatal(err)
			}
			if got.Result != tc.want.result {
				t.Fatalf("result=%q, want %q", got.Result, tc.want.result)
			}
			if tc.want.reason == "" {
				if got.Undecided != nil {
					t.Fatalf("a decided rule carries the reason %v", got.Undecided)
				}
			} else {
				wantUndecided := validationeval.Undecided{Reason: tc.want.reason, Attribute: quantityRule.Attribute}
				if got.Undecided == nil || *got.Undecided != wantUndecided {
					t.Fatalf("undecided=%v, want %v", got.Undecided, wantUndecided)
				}
			}
			if !reflect.DeepEqual(source.calls, []string{"ReadNumber(sales.order_line.quantity)"}) {
				t.Fatalf("reads=%v", source.calls)
			}
		})
	}
}

// The two reasons a number is missing are different facts. If they were one,
// a reader could not tell "ask the person to fill it in" from "read it first".
func TestTheTwoReasonsANumberIsMissingStayDifferent(t *testing.T) {
	null, err := validationeval.Decide(quantityRule, &statefulSource{number: validationeval.NumberReading{State: validationeval.ValueNull}}, validationeval.Operators{})
	if err != nil {
		t.Fatal(err)
	}
	notFetched, err := validationeval.Decide(quantityRule, &statefulSource{number: validationeval.NumberReading{State: validationeval.ValueNotFetched}}, validationeval.Operators{})
	if err != nil {
		t.Fatal(err)
	}
	if null.Undecided == nil || notFetched.Undecided == nil || null.Undecided.Reason == notFetched.Undecided.Reason {
		t.Fatalf("null=%v notFetched=%v", null.Undecided, notFetched.Undecided)
	}
	if !strings.Contains(null.Undecided.String(), "value-null") || !strings.Contains(null.Undecided.String(), "sales.order_line.quantity") {
		t.Fatalf("the reason does not name itself and the attribute: %s", null.Undecided)
	}
}

// A reading that contradicts itself is a broken reader. It is reported as an
// error, so it cannot pass as an undecided rule or as a decided one.
func TestAContradictoryReadingIsAnErrorNotADecision(t *testing.T) {
	cases := []struct {
		name    string
		reading validationeval.NumberReading
		mention string
	}{
		{"present without a value", validationeval.NumberReading{State: validationeval.ValuePresent}, "present without a value"},
		{"null with a value", validationeval.NumberReading{State: validationeval.ValueNull, Value: rat("3")}, "together with a value"},
		{"not fetched with a value", validationeval.NumberReading{State: validationeval.ValueNotFetched, Value: rat("3")}, "together with a value"},
		{"no state at all", validationeval.NumberReading{Value: rat("3")}, "unknown state"},
		{"a state nobody defined", validationeval.NumberReading{State: "absent"}, "unknown state"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := validationeval.Decide(quantityRule, &statefulSource{number: tc.reading}, validationeval.Operators{})
			if err == nil || !strings.Contains(err.Error(), tc.mention) {
				t.Fatalf("err=%v, want one mentioning %q", err, tc.mention)
			}
			if got.Result != "" || got.Undecided != nil {
				t.Fatalf("an error came with the decision %+v", got)
			}
		})
	}

	t.Run("a relation key that is null with a value", func(t *testing.T) {
		source := &statefulSource{relation: validationeval.RelationReading{Coverage: validationeval.CoverageComplete,
			Keys: []validationeval.KeyReading{{State: validationeval.ValueNull, Key: "1"}}}}
		if _, err := validationeval.Decide(lineNumberRule, source, validationeval.Operators{}); err == nil || !strings.Contains(err.Error(), "together with a value") {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("a relation key in no state", func(t *testing.T) {
		source := &statefulSource{relation: validationeval.RelationReading{Coverage: validationeval.CoverageComplete,
			Keys: []validationeval.KeyReading{{Key: "1"}}}}
		if _, err := validationeval.Decide(lineNumberRule, source, validationeval.Operators{}); err == nil || !strings.Contains(err.Error(), "unknown state") {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("a relation without a stated coverage", func(t *testing.T) {
		source := &statefulSource{relation: validationeval.RelationReading{
			Keys: []validationeval.KeyReading{{State: validationeval.ValuePresent, Key: "1"}}}}
		if _, err := validationeval.Decide(lineNumberRule, source, validationeval.Operators{}); err == nil || !strings.Contains(err.Error(), "unknown coverage") {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("a reader that fails", func(t *testing.T) {
		source := &statefulSource{err: fmt.Errorf("connection lost")}
		got, err := validationeval.Decide(quantityRule, source, validationeval.Operators{})
		if err == nil || !strings.Contains(err.Error(), "connection lost") || got.Undecided != nil || got.Result != "" {
			t.Fatalf("got=%+v err=%v", got, err)
		}
	})
	t.Run("no reader", func(t *testing.T) {
		if _, err := validationeval.Decide(quantityRule, nil, validationeval.Operators{}); err == nil {
			t.Fatal("a rule was decided without a reader")
		}
	})
}

// An operator is not asked about a value that is not there, whatever the
// operator would answer for zero.
func TestAnOperatorIsNotAskedAboutAMissingValue(t *testing.T) {
	for _, state := range []validationeval.ValueState{validationeval.ValueNull, validationeval.ValueNotFetched} {
		t.Run(string(state), func(t *testing.T) {
			implementation := &countingImplementation{}
			source := &statefulSource{number: validationeval.NumberReading{State: state}}
			got, err := validationeval.Decide(discountRule, source, validationeval.Operators{
				Declared:        []semantic.Operator{nonNegativeContract},
				Implementations: []validationeval.OperatorImplementation{implementation.nonNegative()},
			})
			if err != nil {
				t.Fatal(err)
			}
			if got.Result != "" || got.Undecided == nil || got.Undecided.Attribute != discountRule.Attribute {
				t.Fatalf("got=%+v", got)
			}
			if implementation.calls != 0 {
				t.Fatalf("the implementation was asked %d times about %v", implementation.calls, implementation.seen)
			}
		})
	}

	t.Run("a present value is passed as it is", func(t *testing.T) {
		implementation := &countingImplementation{}
		source := &statefulSource{number: validationeval.NumberReading{State: validationeval.ValuePresent, Value: rat("-0.01")}}
		got, err := validationeval.Decide(discountRule, source, validationeval.Operators{
			Declared:        []semantic.Operator{nonNegativeContract},
			Implementations: []validationeval.OperatorImplementation{implementation.nonNegative()},
		})
		if err != nil {
			t.Fatal(err)
		}
		if got.Result != validationeval.ResultNotSatisfied || got.Undecided != nil || !reflect.DeepEqual(implementation.seen, []string{"-1/100"}) {
			t.Fatalf("got=%+v seen=%v", got, implementation.seen)
		}
	})

	// An implementation that disagrees with the contract's example is refused
	// before the value is looked at, so a missing value cannot hide it.
	t.Run("a wrong implementation is refused even when the value is missing", func(t *testing.T) {
		contract := nonNegativeContract
		contract.Cases = []semantic.OperatorCase{{Value: "0", Satisfied: true}}
		nonZero := validationeval.OperatorImplementation{Operator: nonNegative, Input: semantic.OperatorInputNumber,
			Number: func(value *big.Rat) (bool, error) { return value.Sign() != 0, nil }}
		source := &statefulSource{number: validationeval.NumberReading{State: validationeval.ValueNull}}
		got, err := validationeval.Decide(discountRule, source, validationeval.Operators{
			Declared: []semantic.Operator{contract}, Implementations: []validationeval.OperatorImplementation{nonZero},
		})
		if err == nil || !strings.Contains(err.Error(), "TEI_OPERATOR_IMPLEMENTATION_MISMATCH") || got.Undecided != nil {
			t.Fatalf("got=%+v err=%v", got, err)
		}
		if len(source.calls) != 0 {
			t.Fatalf("real data was read for a refused implementation: %v", source.calls)
		}
	})

	t.Run("without operators an operator rule is not decided", func(t *testing.T) {
		source := &statefulSource{number: validationeval.NumberReading{State: validationeval.ValueNull}}
		got, err := validationeval.Decide(discountRule, source, validationeval.Operators{})
		if err == nil || got.Undecided != nil || got.Result != "" {
			t.Fatalf("got=%+v err=%v", got, err)
		}
	})
}

// --- relation: the slow definition -----------------------------------------

// oneRow is how a row is written in the cases below: a key, "NULL" for a row
// without a value, "?" for a row whose key was not read.
const (
	rowNull       = "NULL"
	rowNotFetched = "?"
)

func relationReading(rows []string, partial bool) validationeval.RelationReading {
	reading := validationeval.RelationReading{Coverage: validationeval.CoverageComplete}
	if partial {
		reading.Coverage = validationeval.CoveragePartial
	}
	for _, row := range rows {
		switch row {
		case rowNull:
			reading.Keys = append(reading.Keys, validationeval.KeyReading{State: validationeval.ValueNull})
		case rowNotFetched:
			reading.Keys = append(reading.Keys, validationeval.KeyReading{State: validationeval.ValueNotFetched})
		default:
			reading.Keys = append(reading.Keys, validationeval.KeyReading{State: validationeval.ValuePresent, Key: row})
		}
	}
	return reading
}

// possibleAnswers writes out every world that agrees with what was read and
// returns which answers "no two rows share a key" has among them.
//
// A world fixes three things the reading leaves open: the key of each row
// that was not read (any key, or no value), the rows that were not read at
// all when the reading is partial (none, one, or two more), and whether two
// rows without a value share a key (the rule does not say).
func possibleAnswers(rows []string, partial bool) (canBeUnique, canBeDuplicate bool) {
	// Every key a hidden value could take: each key that was seen, one key
	// nobody has per hidden slot, and "no value".
	hidden := 0
	alphabet := []string{rowNull}
	known := map[string]bool{}
	for _, row := range rows {
		if row == rowNotFetched {
			hidden++
		} else if row != rowNull && !known[row] {
			known[row] = true
			alphabet = append(alphabet, row)
		}
	}
	extras := []int{0}
	if partial {
		extras = []int{0, 1, 2}
	}
	for _, extra := range extras {
		slots := hidden + extra
		for i := 0; i < slots; i++ {
			alphabet = append(alphabet, fmt.Sprintf("fresh-%d-%d", extra, i))
		}
		assignment := make([]int, slots)
		for {
			world := make([]string, 0, len(rows)+extra)
			next := 0
			for _, row := range rows {
				if row == rowNotFetched {
					world = append(world, alphabet[assignment[next]])
					next++
				} else {
					world = append(world, row)
				}
			}
			for ; next < slots; next++ {
				world = append(world, alphabet[assignment[next]])
			}
			for _, emptyRowsShareAKey := range []bool{false, true} {
				if worldHasDuplicate(world, emptyRowsShareAKey) {
					canBeDuplicate = true
				} else {
					canBeUnique = true
				}
			}
			// next assignment
			i := 0
			for ; i < slots; i++ {
				assignment[i]++
				if assignment[i] < len(alphabet) {
					break
				}
				assignment[i] = 0
			}
			if i == slots {
				break
			}
		}
		alphabet = alphabet[:len(alphabet)-slots]
	}
	return canBeUnique, canBeDuplicate
}

func worldHasDuplicate(world []string, emptyRowsShareAKey bool) bool {
	for i := range world {
		for j := i + 1; j < len(world); j++ {
			if world[i] != world[j] {
				continue
			}
			if world[i] == rowNull && !emptyRowsShareAKey {
				continue
			}
			return true
		}
	}
	return false
}

func TestTheSlowDefinitionItself(t *testing.T) {
	// The slow definition is checked on cases whose answer is plain.
	for _, tc := range []struct {
		rows                   []string
		partial                bool
		canBeUnique, canBeDupe bool
	}{
		{[]string{"1", "2"}, false, true, false},
		{[]string{"1", "1"}, false, false, true},
		{[]string{"1", "1"}, true, false, true},
		{[]string{"1", "2"}, true, true, true},
		{[]string{"1", rowNotFetched}, false, true, true},
		{[]string{rowNull}, false, true, false},
		{[]string{rowNull, rowNull}, false, true, true},
		{nil, false, true, false},
		{nil, true, true, true},
	} {
		unique, duplicate := possibleAnswers(tc.rows, tc.partial)
		if unique != tc.canBeUnique || duplicate != tc.canBeDupe {
			t.Fatalf("rows=%v partial=%v: unique=%v duplicate=%v", tc.rows, tc.partial, unique, duplicate)
		}
	}
}

// Every reading of up to four rows over two keys, an empty row and an unread
// row, complete and partial: 682 readings. Decide must answer exactly when
// all worlds agree, and with their answer.
func TestUniqueIsDecidedExactlyWhenEveryWorldAgrees(t *testing.T) {
	symbols := []string{"1", "2", rowNull, rowNotFetched}
	var readings [][]string
	var build func(prefix []string)
	build = func(prefix []string) {
		readings = append(readings, append([]string(nil), prefix...))
		if len(prefix) == 4 {
			return
		}
		for _, symbol := range symbols {
			build(append(prefix, symbol))
		}
	}
	build(nil)
	if len(readings) != 341 {
		t.Fatalf("readings=%d", len(readings))
	}

	counts := map[string]int{}
	for _, rows := range readings {
		for _, partial := range []bool{false, true} {
			canBeUnique, canBeDuplicate := possibleAnswers(rows, partial)
			got, err := validationeval.Decide(lineNumberRule, &statefulSource{relation: relationReading(rows, partial)}, validationeval.Operators{})
			if err != nil {
				t.Fatalf("rows=%v partial=%v: %v", rows, partial, err)
			}
			switch {
			case canBeUnique && !canBeDuplicate:
				counts["satisfied"]++
				if got.Result != validationeval.ResultSatisfied || got.Undecided != nil {
					t.Fatalf("rows=%v partial=%v: every world is unique, got %+v", rows, partial, got)
				}
			case canBeDuplicate && !canBeUnique:
				counts["not-satisfied"]++
				if got.Result != validationeval.ResultNotSatisfied || got.Undecided != nil {
					t.Fatalf("rows=%v partial=%v: every world has a duplicate, got %+v", rows, partial, got)
				}
			default:
				counts["undecided"]++
				if got.Result != "" || got.Undecided == nil {
					t.Fatalf("rows=%v partial=%v: the worlds disagree, got %+v", rows, partial, got)
				}
				if got.Undecided.Relation != lineNumberRule.Relation || got.Undecided.Attribute != lineNumberRule.Attribute {
					t.Fatalf("rows=%v partial=%v: the reason names %+v", rows, partial, got.Undecided)
				}
			}
		}
	}
	// All three answers must really occur, or the comparison proves nothing.
	if counts["satisfied"] == 0 || counts["not-satisfied"] == 0 || counts["undecided"] == 0 {
		t.Fatalf("counts=%v", counts)
	}
	t.Logf("counts=%v", counts)
}

// Which reason is named is a choice of this design, so it is written by hand:
// what reading more could settle comes before what it could not.
func TestUniqueNamesTheReasonItStoppedAt(t *testing.T) {
	cases := []struct {
		name    string
		rows    []string
		partial bool
		result  validationeval.Result
		reason  validationeval.UndecidedReason
	}{
		{"all rows read, all keys differ", []string{"1", "2", "3"}, false, validationeval.ResultSatisfied, ""},
		{"no rows", nil, false, validationeval.ResultSatisfied, ""},
		{"one empty row among keys", []string{"1", rowNull, "2"}, false, validationeval.ResultSatisfied, ""},
		{"a duplicate that was seen", []string{"1", "2", "1"}, false, validationeval.ResultNotSatisfied, ""},
		{"a duplicate seen in a part", []string{"1", "1"}, true, validationeval.ResultNotSatisfied, ""},
		{"a duplicate seen beside unread and empty rows", []string{"1", rowNotFetched, rowNull, rowNull, "1"}, true, validationeval.ResultNotSatisfied, ""},
		{"a key that was not read", []string{"1", rowNotFetched}, false, "", validationeval.UndecidedRelationKeyNotFetched},
		{"only a part of the rows", []string{"1", "2"}, true, "", validationeval.UndecidedRelationPartiallyRead},
		{"no rows in the part that was read", nil, true, "", validationeval.UndecidedRelationPartiallyRead},
		{"two empty rows", []string{rowNull, "1", rowNull}, false, "", validationeval.UndecidedRelationKeyNull},
		{"the only row has an unread key", []string{rowNotFetched}, false, validationeval.ResultSatisfied, ""},
		{"the only row read has an unread key", []string{rowNotFetched}, true, "", validationeval.UndecidedRelationPartiallyRead},
		{"unread key before partial", []string{"1", rowNotFetched}, true, "", validationeval.UndecidedRelationKeyNotFetched},
		{"unread key before empty rows", []string{rowNull, rowNull, rowNotFetched}, false, "", validationeval.UndecidedRelationKeyNotFetched},
		{"partial before empty rows", []string{rowNull, rowNull}, true, "", validationeval.UndecidedRelationPartiallyRead},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := &statefulSource{relation: relationReading(tc.rows, tc.partial)}
			got, err := validationeval.Decide(lineNumberRule, source, validationeval.Operators{})
			if err != nil {
				t.Fatal(err)
			}
			var reason validationeval.UndecidedReason
			if got.Undecided != nil {
				reason = got.Undecided.Reason
			}
			if got.Result != tc.result || reason != tc.reason {
				t.Fatalf("result=%q reason=%q, want %q / %q", got.Result, reason, tc.result, tc.reason)
			}
			if !reflect.DeepEqual(source.calls, []string{"ReadRelationKeys(sales.order.lines,sales.order_line.line_number)"}) {
				t.Fatalf("reads=%v", source.calls)
			}
		})
	}
}

// The older entry cannot say why a value is missing. It keeps answering as it
// did: a result, or an error, never a reason.
func TestTheOlderEntryStillAnswersTheSame(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  validationeval.Result
	}{{"3", validationeval.ResultSatisfied}, {"0", validationeval.ResultNotSatisfied}, {"-2", validationeval.ResultNotSatisfied}} {
		older, err := validationeval.Evaluate(quantityRule, &operatorSource{value: rat(tc.value)})
		if err != nil {
			t.Fatal(err)
		}
		newer, err := validationeval.Decide(quantityRule, &statefulSource{number: validationeval.NumberReading{State: validationeval.ValuePresent, Value: rat(tc.value)}}, validationeval.Operators{})
		if err != nil {
			t.Fatal(err)
		}
		if older != tc.want || newer.Result != tc.want {
			t.Fatalf("value=%s older=%q newer=%q want=%q", tc.value, older, newer.Result, tc.want)
		}
	}
	if got, err := validationeval.Evaluate(quantityRule, &operatorSource{}); err == nil || got != "" ||
		!strings.Contains(err.Error(), "has no evaluable number for attribute sales.order_line.quantity") {
		t.Fatalf("got=%q err=%v", got, err)
	}
	// The entry without operators does not look for an operator at all.
	if got, err := validationeval.Evaluate(discountRule, &operatorSource{value: rat("1")}); err == nil || got != "" ||
		!strings.Contains(err.Error(), "has no corresponding Evaluator") {
		t.Fatalf("the entry without operators decided an operator rule: got=%q err=%v", got, err)
	}
	// The message names the kind first and the rule second, so a reader can
	// tell which rule needs an Evaluator for which kind.
	if _, err := validationeval.Evaluate(discountRule, &operatorSource{value: rat("1")}); err == nil ||
		!strings.Contains(err.Error(), `kind "operator" of semantic validation sales.order_line.discount_non_negative has no corresponding Evaluator`) {
		t.Fatalf("the message does not name the kind and the rule in order: %v", err)
	}
}
