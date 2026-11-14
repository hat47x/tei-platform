package sqlite

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"tei/internal/sourceindex"
)

// These independently authored input observations describe the seven-row
// oracle. They are NOT parsed from Markdown by the production TEI Reader.
// Keeping the raw JSON value distinguishes absent from explicit null.
type ctkWindowInputRow struct {
	ObsID string          `json:"obs_id"`
	Group string          `json:"grp"`
	Seq   int64           `json:"seq"`
	Value json.RawMessage `json:"value"`
}

type ctkWindowInputWitness struct {
	Group string
	Seq   int64
	Value sql.NullInt64
}

func validateCTKWindowInputRows(rows []ctkWindowInputRow) (map[string]ctkWindowInputWitness, error) {
	if len(rows) != 7 {
		return nil, fmt.Errorf("incomplete input observation closure: got %d, want 7", len(rows))
	}
	out := make(map[string]ctkWindowInputWitness, len(rows))
	for _, r := range rows {
		if r.ObsID == "" || r.Group == "" || r.Seq <= 0 {
			return nil, fmt.Errorf("invalid input observation identity %q", r.ObsID)
		}
		if _, found := out[r.ObsID]; found {
			return nil, fmt.Errorf("duplicate input observation ID %q", r.ObsID)
		}
		if len(r.Value) == 0 {
			return nil, fmt.Errorf("missing input value field for %s, not explicit null", r.ObsID)
		}
		var value sql.NullInt64
		if bytes.Equal(r.Value, []byte("null")) {
			if r.ObsID != "B2" {
				return nil, fmt.Errorf("unexpected input null at %s", r.ObsID)
			}
		} else {
			if r.ObsID == "B2" {
				return nil, fmt.Errorf("B2 lost its explicit null value")
			}
			if err := json.Unmarshal(r.Value, &value.Int64); err != nil {
				return nil, fmt.Errorf("invalid exact integer at %s: %w", r.ObsID, err)
			}
			value.Valid = true
		}
		out[r.ObsID] = ctkWindowInputWitness{Group: r.Group, Seq: r.Seq, Value: value}
	}
	if _, ok := out["B2"]; !ok {
		return nil, fmt.Errorf("B2 explicit-null observation is missing")
	}
	return out, nil
}

func ctkWindowInputSourceLiteral(r ctkWindowInputRow) (string, error) {
	if len(r.Value) == 0 {
		return "", fmt.Errorf("missing value for %q", r.ObsID)
	}
	val := "NULL"
	if !bytes.Equal(r.Value, []byte("null")) {
		var i int64
		if err := json.Unmarshal(r.Value, &i); err != nil {
			return "", err
		}
		val = strconv.FormatInt(i, 10)
	}
	return fmt.Sprintf("| %s | %s | %d | %s |", r.ObsID, r.Group, r.Seq, val), nil
}

// Textual presence with exact byte spans is a fixture-integrity witness, not
// a Markdown table parse, selected Role, or accepted Window semantics.
func TestCTKWindowInputObservationByteWitnesses(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "testdata", "tei-md-window")
	data, err := os.ReadFile(filepath.Join(root, "independent-expected.candidate.json"))
	if err != nil {
		t.Fatal(err)
	}
	var oracle struct {
		InputObservations []ctkWindowInputRow   `json:"inputObservations"`
		Cases             []ctkWindowOracleCase `json:"cases"`
	}
	if err := json.Unmarshal(data, &oracle); err != nil {
		t.Fatal(err)
	}
	if _, err := validateCTKWindowInputRows(oracle.InputObservations); err != nil {
		t.Fatal(err)
	}
	if err := checkCTKWindowCaseClosure(oracle.Cases); err != nil {
		t.Fatalf("cannot search unqualified Markdown candidate Source: %v", err)
	}
	for _, c := range oracle.Cases {
		t.Run(c.File, func(t *testing.T) {
			src, err := os.ReadFile(filepath.Join(root, c.File))
			if err != nil {
				t.Fatal(err)
			}
			for _, r := range oracle.InputObservations {
				literal, err := ctkWindowInputSourceLiteral(r)
				if err != nil {
					t.Fatal(err)
				}
				at := bytes.Index(src, []byte(literal))
				if at < 0 || bytes.Count(src, []byte(literal)) != 1 {
					t.Fatalf("expected one raw observation line for %s in %s", r.ObsID, c.File)
				}
				rng, err := sourceindex.RangeFromOffsets(src, at, at+len(literal))
				if err != nil {
					t.Fatal(err)
				}
				if err := rng.Validate(src); err != nil {
					t.Fatal(err)
				}
				if string(src[rng.Start.Offset:rng.End.Offset]) != literal {
					t.Fatalf("source range does not identify the exact observation %s", r.ObsID)
				}
			}
			// This negative fixture deliberately has another row with ID A3.
			// Its presence is not accepted as a second semantic observation.
			const extra = "| A3 | A | 3 | 35 |"
			extraCount := bytes.Count(src, []byte(extra))
			if c.File == "duplicate-observation-id.tei.md" && extraCount != 1 ||
				c.File != "duplicate-observation-id.tei.md" && extraCount != 0 {
				t.Fatalf("unexpected duplicate-ID raw witness in %s: %d", c.File, extraCount)
			}
		})
	}
}

func TestCTKWindowInputRowWitnessRejectsMissingNullAndDuplicate(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "testdata", "tei-md-window")
	data, err := os.ReadFile(filepath.Join(root, "independent-expected.candidate.json"))
	if err != nil {
		t.Fatal(err)
	}
	var oracle struct {
		InputObservations []ctkWindowInputRow `json:"inputObservations"`
	}
	if err := json.Unmarshal(data, &oracle); err != nil {
		t.Fatal(err)
	}
	if _, err := validateCTKWindowInputRows(oracle.InputObservations); err != nil {
		t.Fatalf("pinned input observations rejected: %v", err)
	}
	mutate := func(id string, value json.RawMessage) []ctkWindowInputRow {
		t.Helper()
		cp := append([]ctkWindowInputRow(nil), oracle.InputObservations...)
		for i := range cp {
			if cp[i].ObsID == id {
				cp[i].Value = value
				return cp
			}
		}
		t.Fatalf("missing source observation %s", id)
		return nil
	}
	for _, scenario := range []struct {
		name string
		rows []ctkWindowInputRow
	}{
		{"missing B2 value is not explicit null", mutate("B2", nil)},
		{"B2 null changed to numeric zero", mutate("B2", json.RawMessage("0"))},
		{"A3 nonnull changed to null", mutate("A3", json.RawMessage("null"))},
		{"inexact fractional value", mutate("A3", json.RawMessage("3.5"))},
		{"removed input row", oracle.InputObservations[:6]},
		{"duplicate input row", append(append([]ctkWindowInputRow(nil), oracle.InputObservations...), oracle.InputObservations[0])},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			if _, err := validateCTKWindowInputRows(scenario.rows); err == nil {
				t.Fatal("unreviewed input observation change was accepted")
			}
		})
	}
}
