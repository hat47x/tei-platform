package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

// The oracle is a fixed, checked-in contract-owner fixture independent of the
// candidate function. It is not an authenticated third-party test suite and
// cannot prove SQLite execution, Host acceptance, or production conformance.
type sqliteReadOracleCase struct {
	ID          string          `json:"id"`
	Table       string          `json:"table"`
	Field       string          `json:"field"`
	Value       json.RawMessage `json:"value"`
	ValueKind   string          `json:"valueKind,omitempty"`
	ExpectedSQL string          `json:"expectedSQL,omitempty"`
	WantError   bool            `json:"wantError,omitempty"`
}

type sqliteReadOracle struct {
	Schema string                 `json:"schema"`
	Cases  []sqliteReadOracleCase `json:"cases"`
}

type sqliteReadCandidate func(string, string, string, any) (string, []any, error)

func loadSQLiteReadOracle(t *testing.T) []sqliteReadOracleCase {
	t.Helper()
	raw, err := os.ReadFile("testdata/sqlite_read_binding_oracle_v0.json")
	if err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var oracle sqliteReadOracle
	// Top-level fixture metadata is deliberately not interpreted as a Provider grant.
	var doc struct {
		Schema        string                 `json:"schema"`
		State         string                 `json:"state"`
		UpstreamRules []string               `json:"upstreamRules"`
		Cases         []sqliteReadOracleCase `json:"cases"`
	}
	if err := decoder.Decode(&doc); err != nil {
		t.Fatal(err)
	}
	oracle.Schema, oracle.Cases = doc.Schema, doc.Cases
	if oracle.Schema != "tei.reference.sqlite.read-binding.oracle.v0" || len(oracle.Cases) != 14 {
		t.Fatalf("unexpected contract-owner oracle identity or case count: %q (%d)", oracle.Schema, len(oracle.Cases))
	}
	ids := map[string]bool{}
	for _, item := range oracle.Cases {
		if item.ID == "" || ids[item.ID] || len(item.Value) == 0 || (!item.WantError && item.ExpectedSQL == "") {
			t.Fatalf("invalid fixed oracle case: %+v", item)
		}
		ids[item.ID] = true
	}
	return oracle.Cases
}

func sqliteOracleValue(t *testing.T, row sqliteReadOracleCase) any {
	t.Helper()
	var v any
	d := json.NewDecoder(bytes.NewReader(row.Value))
	d.UseNumber()
	if err := d.Decode(&v); err != nil {
		t.Fatalf("invalid oracle value: %v", err)
	}
	return v
}

// This disposable reference candidate is deliberately not a TEI database
// Provider. Identifiers are quoted separately from the single bound value.
func sqliteReadCandidateReference(table, field, valueKind string, value any) (string, []any, error) {
	if valueKind == "int64" {
		raw, ok := value.(string)
		if !ok {
			return "", nil, fmt.Errorf("int64 value must be a decimal string")
		}
		number, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return "", nil, fmt.Errorf("unsupported int64: %w", err)
		}
		value = number
	} else if valueKind != "" {
		return "", nil, fmt.Errorf("unsupported value kind %q", valueKind)
	}
	switch value.(type) {
	case nil, string, int64:
	default:
		return "", nil, fmt.Errorf("unsupported parameter value")
	}
	quote := func(name string) (string, error) {
		if name == "" || strings.ContainsRune(name, 0) || !utf8.ValidString(name) {
			return "", fmt.Errorf("invalid identifier")
		}
		return "\"" + strings.ReplaceAll(name, "\"", "\"\"") + "\"", nil
	}
	tableSQL, err := quote(table)
	if err != nil {
		return "", nil, err
	}
	fieldSQL, err := quote(field)
	if err != nil {
		return "", nil, err
	}
	return "SELECT " + fieldSQL + " FROM " + tableSQL + " WHERE " + fieldSQL + " IS ?1", []any{value}, nil
}

func sqliteOracleAcceptsCandidate(t *testing.T, row sqliteReadOracleCase, candidate sqliteReadCandidate) bool {
	t.Helper()
	input := sqliteOracleValue(t, row)
	sql, args, err := candidate(row.Table, row.Field, row.ValueKind, input)
	if row.WantError {
		return err != nil
	}
	if err != nil || sql != row.ExpectedSQL || len(args) != 1 {
		return false
	}
	expected := input
	if row.ValueKind == "int64" {
		raw, ok := input.(string)
		if !ok {
			return false
		}
		number, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return false
		}
		expected = number
	}
	return reflect.DeepEqual(args[0], expected)
}

func TestSQLiteContractOnlyFixedOracleAcceptsDisposableCandidate(t *testing.T) {
	for _, row := range loadSQLiteReadOracle(t) {
		t.Run(row.ID, func(t *testing.T) {
			if !sqliteOracleAcceptsCandidate(t, row, sqliteReadCandidateReference) {
				t.Fatalf("reference candidate does not satisfy fixed oracle case %s", row.ID)
			}
		})
	}
}

func TestSQLiteContractOnlyFixedOracleRejectsMutatedCandidates(t *testing.T) {
	tests := []struct {
		name      string
		killedBy  string
		candidate sqliteReadCandidate
	}{
		{"value-interpolation", "untrusted-value-not-sql", func(table, field, kind string, value any) (string, []any, error) {
			sql, _, err := sqliteReadCandidateReference(table, field, kind, value)
			if err != nil {
				return "", nil, err
			}
			return strings.Replace(sql, "?1", fmt.Sprint(value), 1), nil, nil
		}},
		{"unquoted-identifiers", "reserved-words", func(table, field, kind string, value any) (string, []any, error) {
			_, args, err := sqliteReadCandidateReference(table, field, kind, value)
			if err != nil {
				return "", nil, err
			}
			return "SELECT " + field + " FROM " + table + " WHERE " + field + " IS ?1", args, nil
		}},
		{"null-to-empty-string", "preserve-null", func(table, field, kind string, value any) (string, []any, error) {
			sql, args, err := sqliteReadCandidateReference(table, field, kind, value)
			if value == nil && err == nil {
				args[0] = ""
			}
			return sql, args, err
		}},
		{"unsupported-treated-as-success", "unsupported-object", func(table, field, kind string, value any) (string, []any, error) {
			sql, args, err := sqliteReadCandidateReference(table, field, kind, value)
			if err != nil {
				return "SELECT 1", nil, nil
			}
			return sql, args, nil
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			killed := false
			for _, row := range loadSQLiteReadOracle(t) {
				if row.ID == tc.killedBy && !sqliteOracleAcceptsCandidate(t, row, tc.candidate) {
					killed = true
				}
			}
			if !killed {
				t.Fatalf("fixed oracle did not reject deliberately faulty candidate: %s", tc.name)
			}
		})
	}
}
