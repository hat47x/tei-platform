package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/database/persistence"
	"tei/internal/driver"
	"tei/internal/ir"
	"tei/internal/realization"
	"tei/internal/semantic"
)

// The new runtime path is opt-in: an ordinary view keeps its projection
// contract, and an explicit database view cannot quietly discard the input.
func TestDestinationRuntimeViewRequiresExplicitInputsAndConnection(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	t.Setenv("TEI_VIEW_PG_DSN", "")
	base := []string{
		"-workspace", root,
		"-application", "order_entry",
		"-pattern", "order_entry.order_lines",
		"-environment", "development",
		"-format", "json",
	}
	for _, tc := range []struct {
		name string
		args []string
		code string
	}{
		{name: "no input", code: "INVALID_ARGUMENT"},
		{name: "malformed input", args: []string{"-input", "1001"}, code: "INVALID_ARGUMENT"},
		{name: "wrong input", args: []string{"-input", "line_number=1"}, code: "INVALID_ARGUMENT"},
		{name: "missing database connection", args: []string{"-input", "order_id=1001"}, code: "VIEW_EXECUTION_FAILED"},
		{name: "duplicate input", args: []string{"-input", "order_id=1001", "-input", "order_id=1002"}, code: "INVALID_ARGUMENT"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			args := append(append([]string(nil), base...), tc.args...)
			exit, err := runViewCommand(args, &out)
			if err != nil || exit == 0 {
				t.Fatalf("exit=%d err=%v json=%s", exit, err, out.String())
			}
			var result struct {
				OK    bool `json:"ok"`
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(out.Bytes(), &result); err != nil {
				t.Fatalf("invalid JSON report: %v (%s)", err, out.String())
			}
			if result.OK || result.Error.Code != tc.code {
				t.Fatalf("report=%+v, want %s", result, tc.code)
			}
		})
	}
}

func TestDestinationRuntimeViewDoesNotSilentlyChangeStaticOrCommandViews(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "runtime without a pattern", args: []string{"-environment", "development"}},
		{name: "runtime with a command", args: []string{"-pattern", "order_entry.order_lines", "-environment", "development", "-command", "sales.order.submit"}},
		{name: "runtime with text format", args: []string{"-pattern", "order_entry.order_lines", "-environment", "development", "-format", "text"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			args := append([]string{"-workspace", root, "-application", "order_entry"}, tc.args...)
			exit, err := runViewCommand(args, &out)
			if err != nil || exit == 0 {
				t.Fatalf("exit=%d err=%v report=%s", exit, err, out.String())
			}
			var report struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(out.Bytes(), &report); err != nil {
				t.Fatal(err)
			}
			if report.Error.Code != "INVALID_ARGUMENT" {
				t.Fatalf("code=%s, want INVALID_ARGUMENT", report.Error.Code)
			}
		})
	}
}

// A numeric-looking code is still text when the declared column is text.
// Only a declared int64 column is parsed as an integer, and an invalid
// integer fails before calling PostgreSQL. The source attribute comes from
// the Contract Query binding, not from a hard-coded reference Workspace.
func TestEnvironmentInputTypeIsFromStorage(t *testing.T) {
	resolved := driver.ResolvedHeaderDetailContractQuery{
		Database: database.Database{ID: "fixture.db"},
		Relation: driver.ResolvedRelation{Source: semantic.Data{ID: "fixture.parent"}},
		Binding: realization.HeaderDetailContractQueryBinding{
			Inputs: []realization.HeaderDetailContractQueryInputBinding{
				{Input: "code", SourceAttribute: "fixture.parent.code"},
				{Input: "number", SourceAttribute: "fixture.parent.number"},
			},
		},
	}
	world := ir.World{Persistence: []persistence.Mapping{{
		Database: "fixture.db", SemanticData: "fixture.parent",
		Columns: []persistence.ColumnMapping{
			{Attribute: "fixture.parent.code", Storage: persistence.StorageVarchar},
			{Attribute: "fixture.parent.number", Storage: persistence.StorageInt64},
		},
	}}}
	source, err := environmentHeaderSource(world, resolved, map[string]string{
		"code": "0007", "number": "7",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(source.Rows) != 1 || len(source.Rows[0]) != 2 {
		t.Fatalf("unexpected source rows: %#v", source.Rows)
	}
	if got, ok := source.Rows[0][0].(string); !ok || got != "0007" {
		t.Fatalf("numeric-looking varchar was converted: %#v", source.Rows[0][0])
	}
	if got, ok := source.Rows[0][1].(int64); !ok || got != 7 {
		t.Fatalf("declared int64 was not parsed: %#v", source.Rows[0][1])
	}
	if _, err := environmentHeaderSource(world, resolved, map[string]string{
		"code": "0007", "number": "not-an-int64",
	}); err == nil {
		t.Fatal("invalid int64 input must fail before provider execution")
	}
	if _, err := environmentHeaderSource(world, resolved, map[string]string{
		"code": "0007", "number": "7", "unexpected": "8",
	}); err == nil {
		t.Fatal("undeclared extra input must be refused")
	}
}

// For a composite key, delimiter joining can make ["a/b","c"] equal
// ["a","b/c"]. A reversible encoding must keep them distinct. Single-part
// keys retain the existing simple representation for the first destination.
func TestEnvironmentCompositeRowKeysAreUnambiguous(t *testing.T) {
	attributes := []core.CanonicalID{"fixture.item.left", "fixture.item.right"}
	a, err := environmentRowKeyValue(map[core.CanonicalID]any{
		"fixture.item.left": "a/b", "fixture.item.right": "c",
	}, attributes)
	if err != nil {
		t.Fatal(err)
	}
	b, err := environmentRowKeyValue(map[core.CanonicalID]any{
		"fixture.item.left": "a", "fixture.item.right": "b/c",
	}, attributes)
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatalf("different composite identities collapsed: %q", a)
	}
	var decoded []string
	if err := json.Unmarshal([]byte(a), &decoded); err != nil {
		t.Fatalf("composite identity cannot be decoded: %v", err)
	}
	if len(decoded) != 2 || decoded[0] != "a/b" || decoded[1] != "c" {
		t.Fatalf("composite identity changed: %v", decoded)
	}
	single, err := environmentRowKeyValue(map[core.CanonicalID]any{
		"fixture.item.left": int64(7),
	}, attributes[:1])
	if err != nil || single != "7" {
		t.Fatalf("single-part identity changed: %q err=%v", single, err)
	}
}
