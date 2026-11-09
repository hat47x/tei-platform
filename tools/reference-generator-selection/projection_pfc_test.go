package generatorselection

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// #2585 PFC-01..05: presence, optional text/bool, integer and decimal
// conformance for a record projection. The expected values come from an oracle
// written here, not from the generated implementation. Numeric expectations
// intentionally record the float64 semantics the JSON decoder uses.
func TestRecordProjectionPFCPresenceAndTypes(t *testing.T) {
	contractPath := filepath.Join("testdata", "pfc-record", "contract.json")
	out := filepath.Join(t.TempDir(), "candidate")
	if _, err := Generate(contractPath, "text-record-template", out); err != nil {
		t.Fatal(err)
	}
	binary := candidateBinaryPath(filepath.Join(t.TempDir(), "candidate"))
	if err := buildCandidate(context.Background(), out, binary); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		raw  string
	}{
		{name: "PFC-01-present", raw: `{"text":"hello","note":"n","flag":false,"count":1,"amount":0.5}`},
		{name: "PFC-01-missing-text", raw: `{"count":1,"amount":0.5}`},
		{name: "PFC-01-null-text", raw: `{"text":null,"count":1,"amount":0.5}`},
		{name: "PFC-01-empty-text", raw: `{"text":"","count":1,"amount":0.5}`},
		{name: "PFC-02-optional-text-missing", raw: `{"text":"x","count":1,"amount":0.5}`},
		{name: "PFC-02-optional-text-null", raw: `{"text":"x","note":null,"count":1,"amount":0.5}`},
		{name: "PFC-03-optional-bool-absent", raw: `{"text":"x","count":1,"amount":0.5}`},
		{name: "PFC-03-optional-bool-null", raw: `{"text":"x","flag":null,"count":1,"amount":0.5}`},
		{name: "PFC-03-optional-bool-false", raw: `{"text":"x","flag":false,"count":1,"amount":0.5}`},
		{name: "PFC-03-optional-bool-true", raw: `{"text":"x","flag":true,"count":1,"amount":0.5}`},
		{name: "PFC-04-integer-precision", raw: `{"text":"x","count":9007199254740993,"amount":0.5}`},
		{name: "PFC-05-decimal-long", raw: `{"text":"x","count":1,"amount":0.10000000000000001}`},
		{name: "PFC-05-decimal-tie", raw: `{"text":"x","count":1,"amount":1.005}`},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got, _, err := runCandidate(context.Background(), binary, json.RawMessage(testCase.raw), ExecutionOptions{})
			if err != nil {
				t.Fatal(err)
			}
			want := pfcExpected(testCase.raw)
			if got != want {
				t.Fatalf("got=%+v want=%+v", got, want)
			}
		})
	}

	// PFC-04: the integer lexeme 9007199254740993 does not survive float64.
	t.Run("PFC-04-loss-is-visible", func(t *testing.T) {
		got, _, err := runCandidate(context.Background(), binary,
			json.RawMessage(`{"text":"x","count":9007199254740993,"amount":0.5}`), ExecutionOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(got.Text, `"count":9007199254740992`) {
			t.Fatalf("float64-rounded integer is missing: %q", got.Text)
		}
		if strings.Contains(got.Text, "9007199254740993") {
			t.Fatalf("the original integer lexeme survived, contrary to the documented loss: %q", got.Text)
		}
	})

	// PFC-05: the decimal lexemes are emitted as float64 without scale.
	t.Run("PFC-05-scale-is-not-preserved", func(t *testing.T) {
		got, _, err := runCandidate(context.Background(), binary,
			json.RawMessage(`{"text":"x","count":1,"amount":0.10000000000000001}`), ExecutionOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(got.Text, `"amount":0.1`) {
			t.Fatalf("unexpected decimal rendering: %q", got.Text)
		}
	})
}

// pfcExpected is the independent oracle for the PFC record contract.
// Required fields are text, count and amount in that order; note and flag are
// optional. Numbers follow encoding/json's float64 decoding.
func pfcExpected(raw string) Result {
	var input map[string]any
	if err := json.Unmarshal([]byte(raw), &input); err != nil {
		return Result{State: "error"}
	}
	record := map[string]any{}
	var missing []string
	for _, name := range []string{"text", "count", "amount"} {
		value, ok := input[name]
		if !ok {
			missing = append(missing, name)
			continue
		}
		if value == nil {
			return Result{State: "unresolved"}
		}
		switch name {
		case "text":
			text, ok := value.(string)
			if !ok {
				return Result{State: "error"}
			}
			record[name] = text
		default:
			number, ok := value.(float64)
			if !ok {
				return Result{State: "error"}
			}
			record[name] = number
		}
	}
	if len(missing) > 0 {
		encoded, _ := json.Marshal(missing)
		return Result{State: "missing", Text: string(encoded)}
	}
	if value, ok := input["note"]; ok && value != nil {
		text, ok := value.(string)
		if !ok {
			return Result{State: "error"}
		}
		record["note"] = text
	}
	if value, ok := input["flag"]; ok && value != nil {
		flag, ok := value.(bool)
		if !ok {
			return Result{State: "error"}
		}
		record["flag"] = flag
	}
	encoded, _ := json.Marshal(record)
	return Result{State: "present", Text: string(encoded)}
}
