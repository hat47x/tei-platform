package generatorselection

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
)

// #2585 PFC-09: duplicate keys and multiple top-level values. Duplicate keys
// keep Go's last-wins decoding (documented, not rejected). Trailing top-level
// values are rejected because the input would otherwise be ambiguous.
func TestRecordProjectionPFCParseBoundary(t *testing.T) {
	contractPath := filepath.Join("testdata", "pfc-record", "contract.json")
	out := filepath.Join(t.TempDir(), "candidate")
	if _, err := Generate(contractPath, "text-record-template", out); err != nil {
		t.Fatal(err)
	}
	binary := candidateBinaryPath(filepath.Join(t.TempDir(), "candidate"))
	if err := buildCandidate(context.Background(), out, binary); err != nil {
		t.Fatal(err)
	}

	t.Run("duplicate-key-keeps-last-value", func(t *testing.T) {
		raw := `{"text":"first","text":"last","count":1,"amount":0.5}`
		got, _, err := runCandidate(context.Background(), binary, json.RawMessage(raw), ExecutionOptions{})
		if err != nil {
			t.Fatal(err)
		}
		want := pfcExpected(raw)
		if got != want {
			t.Fatalf("got=%+v want=%+v", got, want)
		}
		if got.Text != `{"amount":0.5,"count":1,"text":"last"}` {
			t.Fatalf("duplicate key did not resolve to the last value: %q", got.Text)
		}
	})

	t.Run("trailing-top-level-value-is-rejected", func(t *testing.T) {
		raw := `{"text":"a","count":1,"amount":0.5}{"text":"b","count":2,"amount":0.6}`
		got, _, err := runCandidate(context.Background(), binary, json.RawMessage(raw), ExecutionOptions{})
		if err == nil {
			t.Fatalf("trailing top-level value was accepted: %+v", got)
		}
	})

	t.Run("trailing-garbage-is-rejected", func(t *testing.T) {
		raw := `{"text":"a","count":1,"amount":0.5} trailing`
		if got, _, err := runCandidate(context.Background(), binary, json.RawMessage(raw), ExecutionOptions{}); err == nil {
			t.Fatalf("trailing garbage was accepted: %+v", got)
		}
	})

	t.Run("trailing-whitespace-is-accepted", func(t *testing.T) {
		raw := `{"text":"a","count":1,"amount":0.5}`
		got, _, err := runCandidate(context.Background(), binary, json.RawMessage(raw+"\n\t "), ExecutionOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if want := pfcExpected(raw); got != want {
			t.Fatalf("got=%+v want=%+v", got, want)
		}
	})
}
