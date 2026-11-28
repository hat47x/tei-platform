package validationeval_test

import (
	"math/big"
	"reflect"
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

func textValue(text string) *string { return &text }
func truthValue(truth bool) *bool   { return &truth }

func TestBuiltinImplementationsDecideByTheirDefinitions(t *testing.T) {
	// Expected values are the definitions written out for five numbers.
	singleInputWant := map[string]map[string]bool{
		"tei.number.non-negative": {"-3": false, "-1/100": false, "0": true, "1/100": true, "3": true},
		"tei.number.non-zero":     {"-3": true, "-1/100": true, "0": false, "1/100": true, "3": true},
	}
	// The row-scoped implementation compares two columns of the same rows.
	// The expected answers are read from its definition: every row at most
	// its own limit, one row above its own limit, a row whose value was not
	// read, and columns of different lengths (which are not the same rows).
	rowInputWant := map[string][]struct {
		name    string
		rows    map[string][]validationeval.NumberReading
		want    validationeval.Tri
		wantErr bool
	}{
		"tei.number.every-at-most": {
			{name: "every value at most its limit", rows: map[string][]validationeval.NumberReading{
				"value": presentRows("1", "0.5", "0"),
				"limit": presentRows("1", "1", "1"),
			}, want: validationeval.TriYes},
			{name: "one value above its own limit", rows: map[string][]validationeval.NumberReading{
				"value": presentRows("1", "2"),
				"limit": presentRows("1", "1"),
			}, want: validationeval.TriNo},
			{name: "a later limit hides an earlier one", rows: map[string][]validationeval.NumberReading{
				// The per-row comparison is what makes this TriNo: comparing
				// value 2 against the first limit (1) would also say TriNo,
				// but comparing it against the last limit (100) would not.
				"value": presentRows("1", "2"),
				"limit": presentRows("5", "1"),
			}, want: validationeval.TriNo},
			{name: "a row with no value", rows: map[string][]validationeval.NumberReading{
				"value": {{State: validationeval.ValueNull}},
				"limit": presentRows("1"),
			}, want: validationeval.TriMaybe},
			// A declaration may state `absent as zero`, and the rule may name
			// it. This implementation reads that mapping, so an absent value
			// is a value of zero rather than an invented one. Any mapping it
			// does not know leaves the row's own reason.
			{name: "zero is read under the mapping the rule names", rows: map[string][]validationeval.NumberReading{
				"value": presentRows("1", "0"),
				"limit": presentRows("1", "1"),
				// The rows are the same, so the answer is TriYes after the
				// absence would be replaced; the mapping itself is asserted
				// separately below.
			}, want: validationeval.TriYes},
			{name: "columns of different lengths", rows: map[string][]validationeval.NumberReading{
				"value": presentRows("1", "2"),
				"limit": presentRows("100"),
			}, wantErr: true},
			{name: "no value input", rows: map[string][]validationeval.NumberReading{
				"limit": presentRows("1"),
			}, wantErr: true},
			{name: "no limit input", rows: map[string][]validationeval.NumberReading{
				"value": presentRows("1"),
			}, wantErr: true},
		},
	}
	multiInputWant := map[string][]struct {
		name    string
		values  map[string]*big.Rat
		want    bool
		wantErr bool
	}{
		"tei.number.within": {
			{name: "in-range", values: map[string]*big.Rat{"value": big.NewRat(0, 1), "low": big.NewRat(0, 1), "high": big.NewRat(100, 1)}, want: true},
			{name: "at-low", values: map[string]*big.Rat{"value": big.NewRat(0, 1), "low": big.NewRat(0, 1), "high": big.NewRat(100, 1)}, want: true},
			{name: "at-high", values: map[string]*big.Rat{"value": big.NewRat(100, 1), "low": big.NewRat(0, 1), "high": big.NewRat(100, 1)}, want: true},
			{name: "below-low", values: map[string]*big.Rat{"value": big.NewRat(-1, 1), "low": big.NewRat(0, 1), "high": big.NewRat(100, 1)}, want: false},
			{name: "above-high", values: map[string]*big.Rat{"value": big.NewRat(101, 1), "low": big.NewRat(0, 1), "high": big.NewRat(100, 1)}, want: false},
			// low > high is a wrong range specification: it is reported, never
			// answered as not-satisfied, and the bounds are not swapped.
			{name: "low-greater-than-high", values: map[string]*big.Rat{"value": big.NewRat(50, 1), "low": big.NewRat(100, 1), "high": big.NewRat(0, 1)}, wantErr: true},
		},
	}
	// tei.day.not-before decides days through the one value shape section 8.4
	// fixes, so its expectations are the values as the reader hands them over
	// and the answer the contract states. A day is a day, so the values carry
	// the type they were read as.
	typedInputWant := map[string][]struct {
		name    string
		values  map[string]semantic.OperatorValue
		want    bool
		wantErr bool
	}{
		"tei.day.not-before": {
			{name: "the day is the deadline", values: dayValues("2026-04-01", "2026-04-01"), want: true},
			{name: "after the deadline", values: dayValues("2026-04-02", "2026-04-01"), want: true},
			{name: "before the deadline", values: dayValues("2026-03-31", "2026-04-01"), want: false},
			{name: "no value input", values: dayValues("", "2026-04-01"), wantErr: true},
			{name: "no deadline input", values: dayValues("2026-04-01", ""), wantErr: true},
		},
		"tei.text.same-as": {
			{name: "the same text", values: map[string]semantic.OperatorValue{"value": {Text: textValue("A-100")}, "other": {Text: textValue("A-100")}}, want: true},
			{name: "a different text", values: map[string]semantic.OperatorValue{"value": {Text: textValue("A-100")}, "other": {Text: textValue("B-200")}}, want: false},
			// The empty text is a value of length zero, so two of them are the
			// same text.
			{name: "the empty text", values: map[string]semantic.OperatorValue{"value": {Text: textValue("")}, "other": {Text: textValue("")}}, want: true},
			{name: "a text against an empty one", values: map[string]semantic.OperatorValue{"value": {Text: textValue("A-100")}, "other": {Text: textValue("")}}, want: false},
			{name: "no value input", values: map[string]semantic.OperatorValue{"other": {Text: textValue("A-100")}}, wantErr: true},
			{name: "no other input", values: map[string]semantic.OperatorValue{"value": {Text: textValue("A-100")}}, wantErr: true},
			// A value that was not read as a text is not one, and this
			// implementation says so rather than reading it as the nearest text.
			{name: "a value read as a number", values: map[string]semantic.OperatorValue{"value": {Number: big.NewRat(7, 1)}, "other": {Text: textValue("7")}}, wantErr: true},
		},
		"tei.any.same-as": {
			{name: "the same number", values: anyPair(anyValueOf(big.NewRat(100, 1)), anyValueOf(big.NewRat(100, 1))), want: true},
			{name: "a different number", values: anyPair(anyValueOf(big.NewRat(100, 1)), anyValueOf(big.NewRat(200, 1))), want: false},
			{name: "the same text", values: anyPair(anyValueOf("100"), anyValueOf("100")), want: true},
			{name: "a different text", values: anyPair(anyValueOf("100"), anyValueOf("200")), want: false},
			{name: "the same truth", values: anyPair(anyValueOf(true), anyValueOf(true)), want: true},
			{name: "a different truth", values: anyPair(anyValueOf(true), anyValueOf(false)), want: false},
			// A number and a text that spell the same characters are two values,
			// which is the distinction section 8.3 makes in an example table and
			// section 8.4 carries into the value an implementation receives.
			{name: "a number against the text of the same characters", values: anyPair(anyValueOf(big.NewRat(100, 1)), anyValueOf("100")), want: false},
			{name: "no value input", values: map[string]semantic.OperatorValue{"other": anyValueOf("A-100")}, wantErr: true},
			{name: "no other input", values: map[string]semantic.OperatorValue{"value": anyValueOf("A-100")}, wantErr: true},
			// A value that was not read as a value of any type is not one, and
			// this implementation says so rather than comparing it as the
			// nearest type.
			{name: "a value read as a number", values: map[string]semantic.OperatorValue{"value": {Number: big.NewRat(7, 1)}, "other": anyValueOf("7")}, wantErr: true},
		},
		"tei.truth.is-true": {
			{name: "true", values: map[string]semantic.OperatorValue{"value": {Truth: truthValue(true)}}, want: true},
			{name: "false", values: map[string]semantic.OperatorValue{"value": {Truth: truthValue(false)}}, want: false},
			{name: "no value input", values: map[string]semantic.OperatorValue{}, wantErr: true},
			// A value read as a number is not a truth, and this implementation
			// says so rather than reading 1 as true.
			{name: "a value read as a number", values: map[string]semantic.OperatorValue{"value": {Number: big.NewRat(1, 1)}}, wantErr: true},
		},
	}
	// tei.day.every-not-before decides days through the one value shape section
	// 8.4 fixes, over the rows of a relation. Its expectations are the rows as
	// the reader hands them over, each value carrying the type it was read as,
	// and the answer the contract states for them.
	typedRowInputWant := map[string][]struct {
		name    string
		rows    map[string][]validationeval.RowValueReading
		want    validationeval.Tri
		wantErr bool
	}{
		"tei.day.every-not-before": {
			{name: "every day not before its deadline", rows: dayColumns(
				[2]string{"2026-04-01", "2026-04-01"}, [2]string{"2026-04-02", "2026-04-01"},
			), want: validationeval.TriYes},
			{name: "one day before its own deadline", rows: dayColumns(
				[2]string{"2026-04-01", "2026-04-05"}, [2]string{"2026-03-31", "2026-04-01"},
			), want: validationeval.TriNo},
			{name: "a later deadline hides an earlier one", rows: dayColumns(
				// The per-row comparison is what makes this TriNo: comparing the
				// second day against the first deadline would also say TriNo, but
				// comparing it against the last one would not.
				[2]string{"2026-04-01", "2026-05-01"}, [2]string{"2026-03-31", "2026-04-01"},
			), want: validationeval.TriNo},
			{name: "a row that holds no value", rows: dayColumns(
				[2]string{semantic.RowCellNoValue, "2026-04-01"}, [2]string{"2026-04-02", "2026-04-01"},
			), want: validationeval.TriMaybe},
			{name: "a row nobody read", rows: dayColumns(
				[2]string{semantic.RowCellNotFetched, "2026-04-01"}, [2]string{"2026-04-02", "2026-04-01"},
			), want: validationeval.TriMaybe},
			// Section 8.1: the empty relation is one example. No row is before
			// its deadline, so the answer is the operator's own.
			{name: "no rows", rows: map[string][]validationeval.RowValueReading{
				"value": {}, "deadline": {},
			}, want: validationeval.TriYes},
			{name: "columns of different lengths", rows: map[string][]validationeval.RowValueReading{
				"value":    {dayCell("2026-04-01"), dayCell("2026-04-02")},
				"deadline": {dayCell("2026-04-01")},
			}, wantErr: true},
			{name: "no value input", rows: map[string][]validationeval.RowValueReading{
				"deadline": {dayCell("2026-04-01")},
			}, wantErr: true},
			{name: "no deadline input", rows: map[string][]validationeval.RowValueReading{
				"value": {dayCell("2026-04-01")},
			}, wantErr: true},
			{name: "a day nobody read is not a day", rows: map[string][]validationeval.RowValueReading{
				"value":    {{State: validationeval.ValuePresent, Value: semantic.OperatorValue{Number: big.NewRat(20260401, 1)}}},
				"deadline": {dayCell("2026-04-01")},
			}, wantErr: true},
		},
		// The row forms of the text, truth and any implementations compare the
		// rows the reader hands over. The expected answers are read from their
		// definitions: the row form answers about the rows as the one-record form
		// answers about one record, so the same counterexample settles it.
		"tei.text.every-same-as": {
			{name: "every row's texts are the same", rows: textColumns(
				[2]string{"\"A-100\"", "\"A-100\""}, [2]string{"\"B-2\"", "\"B-2\""},
			), want: validationeval.TriYes},
			{name: "one row's texts differ", rows: textColumns(
				[2]string{"\"A-100\"", "\"A-100\""}, [2]string{"\"B-2\"", "\"B-3\""},
			), want: validationeval.TriNo},
			{name: "a row nobody read", rows: textColumns(
				[2]string{"\"A-100\"", "\"A-100\""}, [2]string{semantic.RowCellNotFetched, "\"B-2\""},
			), want: validationeval.TriMaybe},
			{name: "columns of different lengths", rows: map[string][]validationeval.RowValueReading{
				"value": {textCell("\"A-100\""), textCell("\"B-2\"")},
				"other": {textCell("\"A-100\"")},
			}, wantErr: true},
			{name: "no other input", rows: map[string][]validationeval.RowValueReading{
				"value": {textCell("\"A-100\"")},
			}, wantErr: true},
		},
		"tei.truth.every-is-true": {
			{name: "every row holds true", rows: map[string][]validationeval.RowValueReading{
				"value": {truthCell("true"), truthCell("true")},
			}, want: validationeval.TriYes},
			{name: "one row holds false", rows: map[string][]validationeval.RowValueReading{
				"value": {truthCell("true"), truthCell("false")},
			}, want: validationeval.TriNo},
			{name: "a row nobody read", rows: map[string][]validationeval.RowValueReading{
				"value": {truthCell("true"), {State: validationeval.ValueNotFetched}},
			}, want: validationeval.TriMaybe},
			{name: "no value input", rows: map[string][]validationeval.RowValueReading{}, wantErr: true},
			{name: "a number is not a truth", rows: map[string][]validationeval.RowValueReading{
				"value": {{State: validationeval.ValuePresent, Value: semantic.OperatorValue{Number: big.NewRat(1, 1)}}},
			}, wantErr: true},
		},
		"tei.any.every-same-as": {
			{name: "every row's values are the same", rows: anyColumns(
				[2]string{"100", "100"}, [2]string{"\"100\"", "\"100\""},
			), want: validationeval.TriYes},
			{name: "a number and a text are not the same", rows: anyColumns(
				[2]string{"100", "\"100\""},
			), want: validationeval.TriNo},
			{name: "one row's values differ", rows: anyColumns(
				[2]string{"100", "100"}, [2]string{"200", "100"},
			), want: validationeval.TriNo},
			{name: "a row nobody read", rows: anyColumns(
				[2]string{"100", "100"}, [2]string{semantic.RowCellNotFetched, "100"},
			), want: validationeval.TriMaybe},
			{name: "no other input", rows: map[string][]validationeval.RowValueReading{
				"value": {anyCell("100")},
			}, wantErr: true},
		},
	}
	seen := map[string]bool{}
	for _, implementation := range validationeval.BuiltinImplementations() {
		if seen[implementation.ID] {
			t.Fatalf("built-in implementation %s is listed twice", implementation.ID)
		}
		seen[implementation.ID] = true
		if implementation.Number != nil {
			expected, known := singleInputWant[implementation.ID]
			if !known {
				t.Fatalf("built-in implementation %s has no expected values in this test", implementation.ID)
			}
			if implementation.Input != semantic.OperatorInputNumber {
				t.Fatalf("%s input=%q", implementation.ID, implementation.Input)
			}
			for text, satisfied := range expected {
				value, _ := new(big.Rat).SetString(text)
				got, err := implementation.Number(value)
				if err != nil || got != satisfied {
					t.Fatalf("%s(%s)=%v err=%v, want %v", implementation.ID, text, got, err, satisfied)
				}
			}
		} else if implementation.Numbers != nil {
			cases, known := multiInputWant[implementation.ID]
			if !known {
				t.Fatalf("built-in implementation %s has no expected values in this test", implementation.ID)
			}
			for _, tc := range cases {
				got, err := implementation.Numbers(tc.values)
				if tc.wantErr {
					if err == nil {
						t.Fatalf("%s(%s)=%v err=nil, want an error", implementation.ID, tc.name, got)
					}
					continue
				}
				if err != nil {
					t.Fatalf("%s(%s)=%v err=%v, want %v", implementation.ID, tc.name, got, err, tc.want)
				}
				if got != tc.want {
					t.Fatalf("%s(%s)=%v, want %v", implementation.ID, tc.name, got, tc.want)
				}
			}
		} else if implementation.NumberRows != nil {
			cases, known := rowInputWant[implementation.ID]
			if !known {
				t.Fatalf("built-in implementation %s has no expected values in this test", implementation.ID)
			}
			for _, tc := range cases {
				got, err := implementation.NumberRows(tc.rows)
				if tc.wantErr {
					if err == nil {
						t.Fatalf("%s(%s)=%d err=nil, want an error", implementation.ID, tc.name, int(got))
					}
					continue
				}
				if err != nil {
					t.Fatalf("%s(%s)=%d err=%v, want %d", implementation.ID, tc.name, int(got), err, int(tc.want))
				}
				if got != tc.want {
					t.Fatalf("%s(%s)=%d, want %d", implementation.ID, tc.name, int(got), int(tc.want))
				}
			}
		} else if implementation.ValueRows != nil {
			cases, known := typedRowInputWant[implementation.ID]
			if !known {
				t.Fatalf("built-in implementation %s has no expected values in this test", implementation.ID)
			}
			for _, tc := range cases {
				got, err := implementation.ValueRows(tc.rows)
				if tc.wantErr {
					if err == nil {
						t.Fatalf("%s(%s)=%d err=nil, want an error", implementation.ID, tc.name, int(got))
					}
					continue
				}
				if err != nil {
					t.Fatalf("%s(%s)=%d err=%v, want %d", implementation.ID, tc.name, int(got), err, int(tc.want))
				}
				if got != tc.want {
					t.Fatalf("%s(%s)=%d, want %d", implementation.ID, tc.name, int(got), int(tc.want))
				}
			}
		} else if implementation.Presence != nil {
			cases, known := presenceInputWant[implementation.ID]
			if !known {
				t.Fatalf("built-in implementation %s has no expected presence values in this test", implementation.ID)
			}
			for _, tc := range cases {
				got, err := implementation.Presence(tc.values)
				if tc.wantErr {
					if err == nil {
						t.Fatalf("%s(%s)=%v err=nil, want an error", implementation.ID, tc.name, got)
					}
					continue
				}
				if err != nil {
					t.Fatalf("%s(%s)=%v err=%v, want %v", implementation.ID, tc.name, got, err, tc.want)
				}
				if got != tc.want {
					t.Fatalf("%s(%s)=%v, want %v", implementation.ID, tc.name, got, tc.want)
				}
			}
		} else if implementation.Values != nil {
			cases, known := typedInputWant[implementation.ID]
			if !known {
				t.Fatalf("built-in implementation %s has no expected values in this test", implementation.ID)
			}
			for _, tc := range cases {
				got, err := implementation.Values(tc.values)
				if tc.wantErr {
					if err == nil {
						t.Fatalf("%s(%s)=%v err=nil, want an error", implementation.ID, tc.name, got)
					}
					continue
				}
				if err != nil {
					t.Fatalf("%s(%s)=%v err=%v, want %v", implementation.ID, tc.name, got, err, tc.want)
				}
				if got != tc.want {
					t.Fatalf("%s(%s)=%v, want %v", implementation.ID, tc.name, got, tc.want)
				}
			}
		} else {
			t.Fatalf("built-in implementation %s has no decision function", implementation.ID)
		}
	}
	if len(seen) != len(singleInputWant)+len(multiInputWant)+len(rowInputWant)+len(typedInputWant)+len(typedRowInputWant)+len(presenceInputWant) {
		t.Fatalf("built-in implementations=%v, want exactly %d", seen, len(singleInputWant)+len(multiInputWant)+len(rowInputWant)+len(typedInputWant)+len(typedRowInputWant)+len(presenceInputWant))
	}
}

// presenceInputWant is what each shipped presence implementation answers for the
// three states a reader reports. A presence rule is decided by whether a value
// is there, so the answer is read from the state and never from a value.
var presenceInputWant = map[string][]struct {
	name    string
	values  map[string]bool
	want    bool
	wantErr bool
}{
	"tei.presence.present": {
		{
			name:   "an attribute that holds a value",
			values: map[string]bool{"value": true},
			want:   true,
		},
		{
			name:   "an attribute that holds no value",
			values: map[string]bool{"value": false},
			want:   false,
		},
		{
			name:    "no value input",
			values:  map[string]bool{},
			wantErr: true,
		},
	},
}

// textColumns builds the two text columns a rule hands to an implementation:
// one row per pair, in the order written.
func textColumns(pairs ...[2]string) map[string][]validationeval.RowValueReading {
	rows := map[string][]validationeval.RowValueReading{}
	for _, pair := range pairs {
		rows["value"] = append(rows["value"], textCell(pair[0]))
		rows["other"] = append(rows["other"], textCell(pair[1]))
	}
	return rows
}

// anyColumns builds the two columns of an `any` input, one row per pair.
func anyColumns(pairs ...[2]string) map[string][]validationeval.RowValueReading {
	rows := map[string][]validationeval.RowValueReading{}
	for _, pair := range pairs {
		rows["value"] = append(rows["value"], anyCell(pair[0]))
		rows["other"] = append(rows["other"], anyCell(pair[1]))
	}
	return rows
}

// textCell reads one cell of a text column, in the writing section 8.3 fixes:
// quotes around the text, and the two words for "no value" and "not read".
func textCell(text string) validationeval.RowValueReading {
	switch text {
	case semantic.RowCellNoValue:
		return validationeval.RowValueReading{State: validationeval.ValueNull}
	case semantic.RowCellNotFetched:
		return validationeval.RowValueReading{State: validationeval.ValueNotFetched}
	}
	value, err := semantic.ParseOperatorTextValue(text)
	if err != nil {
		panic(err)
	}
	return validationeval.RowValueReading{
		State: validationeval.ValuePresent,
		Value: semantic.OperatorValue{Text: &value},
	}
}

// truthCell reads one cell of a truth column: the words true and false.
func truthCell(text string) validationeval.RowValueReading {
	value, err := semantic.ParseOperatorTruthValue(text)
	if err != nil {
		panic(err)
	}
	return validationeval.RowValueReading{
		State: validationeval.ValuePresent,
		Value: semantic.OperatorValue{Truth: &value},
	}
}

// anyCell reads one cell of an `any` column, where the quoting decides the type.
func anyCell(text string) validationeval.RowValueReading {
	switch text {
	case semantic.RowCellNoValue:
		return validationeval.RowValueReading{State: validationeval.ValueNull}
	case semantic.RowCellNotFetched:
		return validationeval.RowValueReading{State: validationeval.ValueNotFetched}
	}
	value, err := semantic.ParseOperatorInputValue(semantic.OperatorInputAny, text)
	if err != nil {
		panic(err)
	}
	return validationeval.RowValueReading{State: validationeval.ValuePresent, Value: value}
}

// dayColumns builds the two columns a rule about days hands to an
// implementation: one row per pair, in the order written.
func dayColumns(pairs ...[2]string) map[string][]validationeval.RowValueReading {
	rows := map[string][]validationeval.RowValueReading{}
	for _, pair := range pairs {
		rows["value"] = append(rows["value"], dayCell(pair[0]))
		rows["deadline"] = append(rows["deadline"], dayCell(pair[1]))
	}
	return rows
}

// dayCell reads one cell the way a reader hands it over: a day, a row that
// holds no value, or a row nobody read.
func dayCell(text string) validationeval.RowValueReading {
	switch text {
	case semantic.RowCellNoValue:
		return validationeval.RowValueReading{State: validationeval.ValueNull}
	case semantic.RowCellNotFetched:
		return validationeval.RowValueReading{State: validationeval.ValueNotFetched}
	}
	day, err := semantic.ParseCalendarDate(text)
	if err != nil {
		panic(err)
	}
	return validationeval.RowValueReading{State: validationeval.ValuePresent, Value: semantic.OperatorValue{Date: &day}}
}

// dayValues builds the values a rule about two days hands to an implementation.
// An empty text is a day nobody supplied, which the implementation refuses
// rather than reading as the zero day.
func dayValues(value, deadline string) map[string]semantic.OperatorValue {
	values := map[string]semantic.OperatorValue{}
	if value != "" {
		day, err := semantic.ParseCalendarDate(value)
		if err != nil {
			panic(err)
		}
		values["value"] = semantic.OperatorValue{Date: &day}
	}
	if deadline != "" {
		day, err := semantic.ParseCalendarDate(deadline)
		if err != nil {
			panic(err)
		}
		values["deadline"] = semantic.OperatorValue{Date: &day}
	}
	return values
}

func TestResolveImplementationsReturnsOnlyWhatWasSelected(t *testing.T) {
	var calls []string
	record := func(name string, result bool) validationeval.NamedImplementation {
		return validationeval.NamedImplementation{
			ID: name, Input: semantic.OperatorInputNumber,
			Number: func(*big.Rat) (bool, error) { calls = append(calls, name); return result, nil },
		}
	}
	a, b := record("example.a", true), record("example.b", false)
	selection := []validationeval.ImplementationSelection{{Operator: "sales.non_negative", Implementation: "example.b"}}

	for _, available := range [][]validationeval.NamedImplementation{{a, b}, {b, a}} {
		calls = nil
		var observed []validationeval.ImplementationCall
		resolved, err := validationeval.ResolveImplementations(selection, available, func(call validationeval.ImplementationCall) {
			observed = append(observed, call)
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(resolved) != 1 || resolved[0].Operator != "sales.non_negative" || resolved[0].Input != semantic.OperatorInputNumber {
			t.Fatalf("resolved=%+v", resolved)
		}
		if len(observed) != 0 {
			t.Fatalf("resolving reported a call before anything was decided: %+v", observed)
		}
		got, err := resolved[0].Number(big.NewRat(1, 1))
		if err != nil || got {
			t.Fatalf("selected B must answer false: %v %v", got, err)
		}
		if !reflect.DeepEqual(calls, []string{"example.b"}) {
			t.Fatalf("called %v, want only the selected implementation", calls)
		}
		wantCall := []validationeval.ImplementationCall{{Operator: "sales.non_negative", Implementation: "example.b"}}
		if !reflect.DeepEqual(observed, wantCall) {
			t.Fatalf("observed=%+v, want %+v", observed, wantCall)
		}
	}
}

func TestResolveImplementationsRefusals(t *testing.T) {
	one := validationeval.NamedImplementation{ID: "example.a", Input: semantic.OperatorInputNumber,
		Number: func(*big.Rat) (bool, error) { return true, nil }}
	cases := []struct {
		name      string
		selected  []validationeval.ImplementationSelection
		available []validationeval.NamedImplementation
		want      string
	}{
		{"name nobody carries",
			[]validationeval.ImplementationSelection{{Operator: "sales.op", Implementation: "example.missing"}},
			[]validationeval.NamedImplementation{one}, "TEI_OPERATOR_IMPLEMENTATION_UNAVAILABLE"},
		{"name carried twice",
			[]validationeval.ImplementationSelection{{Operator: "sales.op", Implementation: "example.a"}},
			[]validationeval.NamedImplementation{one, one}, "TEI_OPERATOR_IMPLEMENTATION_AMBIGUOUS"},
		{"operator selected twice",
			[]validationeval.ImplementationSelection{
				{Operator: "sales.op", Implementation: "example.a"}, {Operator: "sales.op", Implementation: "example.a"}},
			[]validationeval.NamedImplementation{one}, "TEI_OPERATOR_IMPLEMENTATION_AMBIGUOUS"},
		{"selection without an implementation",
			[]validationeval.ImplementationSelection{{Operator: "sales.op"}},
			[]validationeval.NamedImplementation{one}, "TEI_OPERATOR_IMPLEMENTATION_INVALID"},
		{"selection without an operator",
			[]validationeval.ImplementationSelection{{Implementation: "example.a"}},
			[]validationeval.NamedImplementation{one}, "TEI_OPERATOR_IMPLEMENTATION_INVALID"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resolved, err := validationeval.ResolveImplementations(tc.selected, tc.available, nil)
			if err == nil || !strings.Contains(err.Error(), tc.want) || resolved != nil {
				t.Fatalf("resolved=%+v err=%v, want %q", resolved, err, tc.want)
			}
		})
	}
	// The unavailable message names what can be selected instead.
	_, err := validationeval.ResolveImplementations(
		[]validationeval.ImplementationSelection{{Operator: "sales.op", Implementation: "example.missing"}},
		validationeval.BuiltinImplementations(), nil)
	if err == nil || !strings.Contains(err.Error(), "tei.number.non-negative, tei.number.non-zero") {
		t.Fatalf("message does not list the available implementations: %v", err)
	}
}

// presentRows builds the readings of one column whose rows all hold a value.
func presentRows(values ...string) []validationeval.NumberReading {
	out := make([]validationeval.NumberReading, 0, len(values))
	for _, text := range values {
		value, ok := new(big.Rat).SetString(text)
		if !ok {
			panic("not a number: " + text)
		}
		out = append(out, validationeval.NumberReading{State: validationeval.ValuePresent, Value: value})
	}
	return out
}

// within reads the same mapping every-at-most learned: an absent bound is
// zero only when the declaration states `absent as zero` and the rule names it.
func TestWithinReadsAbsentValuesUnderTheMappingItKnows(t *testing.T) {
	for _, available := range validationeval.BuiltinImplementations() {
		if available.ID != "tei.number.within" {
			continue
		}
		if available.Absent == nil {
			t.Fatal("within states no mapping reader")
		}
		zero, ok := available.Absent("zero")
		if !ok || zero.Sign() != 0 {
			t.Fatalf("zero=%v ok=%v, want the value zero", zero, ok)
		}
		if _, ok := available.Absent("open"); ok {
			t.Fatal("within read a mapping it does not state")
		}
	}
}

// An absence is read under the mapping the rule named, and a mapping the
// implementation does not know leaves the row's own reason.
func TestEveryAtMostReadsAbsentValuesUnderTheMappingItKnows(t *testing.T) {
	for _, available := range validationeval.BuiltinImplementations() {
		if available.ID != "tei.number.every-at-most" {
			continue
		}
		if available.Absent == nil {
			t.Fatal("every-at-most states no mapping reader")
		}
		zero, ok := available.Absent("zero")
		if !ok || zero.Sign() != 0 {
			t.Fatalf("zero=%v ok=%v, want the value zero", zero, ok)
		}
		// A mapping this implementation does not know is refused, so the
		// absence keeps its own reason rather than being read as zero.
		if _, ok := available.Absent("open"); ok {
			t.Fatal("every-at-most read a mapping it does not state")
		}

		// With a row whose value is absent, the mapping it states makes the
		// row a zero; without the mapping the answer stays undecided.
		rows := map[string][]validationeval.NumberReading{
			"value": {{State: validationeval.ValueNull}, {State: validationeval.ValuePresent, Value: big.NewRat(5, 1)}},
			"limit": presentRows("1", "10"),
		}
		got, err := available.NumberRows(rows)
		if err != nil {
			t.Fatalf("err=%v", err)
		}
		if got != validationeval.TriMaybe {
			t.Fatalf("an absent row answered %d; it must not be read as a value the implementation never received", int(got))
		}
	}
}

// anyPair builds the two inputs the shipped `tei.any.same-as` reads.
func anyPair(value, other semantic.OperatorValue) map[string]semantic.OperatorValue {
	return map[string]semantic.OperatorValue{"value": value, "other": other}
}
