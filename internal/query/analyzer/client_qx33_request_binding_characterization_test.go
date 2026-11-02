package analyzer

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"tei/internal/core"
)

// QX-33 characterizes the present wire protocol. A matching Query ID and
// protocolVersion do NOT bind the response to SQL source or schema bytes.
// The helper intentionally ignores Source and Schema, unlike a real worker.
func TestQX33SameQueryIdentityDoesNotBindSQLSourceOrSchema(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TEI_QX33_WORKER_HELPER", "1")
	original := Request{
		ProtocolVersion: 1,
		Operation:       "analyze",
		Query:           core.MustCanonicalID("qx32.order_query"),
		Source:          "SELECT o.id FROM orders o WHERE o.hidden_score > 50",
		Schema: &SchemaContext{Tables: []SchemaTable{
			{Name: "orders", Columns: []string{"id", "hidden_score"}},
		}},
	}
	cases := []struct {
		name   string
		mutate func(*Request)
	}{
		{"same-source", func(*Request) {}},
		{"changed-predicate", func(q *Request) {
			q.Source = "SELECT o.id FROM orders o WHERE o.hidden_score > 70"
		}},
		{"changed-referenced-table", func(q *Request) {
			q.Source = "SELECT o.id FROM other_orders o WHERE o.hidden_score > 50"
		}},
		{"changed-schema", func(q *Request) {
			q.Schema = &SchemaContext{Tables: []SchemaTable{
				{Name: "orders", Columns: []string{"id"}},
			}}
		}},
	}
	var baseline Result
	for index, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := original
			tc.mutate(&req)
			got, err := RunPythonWorker(context.Background(), exe,
				"-test.run=^TestQX33WorkerHelper$", req)
			if err != nil {
				t.Fatal(err)
			}
			if got.Query != req.Query || got.ProtocolVersion != req.ProtocolVersion ||
				got.Status != "ok" || got.Lowering != "full" {
				t.Fatalf("fixture should return a matching, full-coverage synthetic report: %+v", got)
			}
			if index == 0 {
				baseline = got
			} else if !reflect.DeepEqual(baseline, got) {
				t.Fatalf("the deliberately stale fixture report changed: baseline=%+v got=%+v", baseline, got)
			}
		})
	}
}

// This subprocess is a deterministic test helper, NOT SQLGlot and NOT a
// candidate production authorization or worker-attestation implementation.
func TestQX33WorkerHelper(t *testing.T) {
	if os.Getenv("TEI_QX33_WORKER_HELPER") != "1" {
		return
	}
	var req Request
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		os.Exit(2)
	}
	// Intentionally return the original SELECT's synthetic analysis for every
	// different incoming SQL/schema with the same Query and protocol version.
	reply := Result{
		ProtocolVersion: req.ProtocolVersion,
		Status:          "ok",
		Query:           req.Query,
		Lowering:        "full",
		Scopes:          []Scope{{ID: "scope:root", Kind: "root"}},
		ColumnRefs: []ColumnOccurrence{
			{ID: "col:id", ScopeID: "scope:root", Name: "id", Qualifier: "o"},
			{ID: "col:hidden", ScopeID: "scope:root", Name: "hidden_score", Qualifier: "o"},
		},
		OutputFields: []OutputField{{
			ID: "out:id", ScopeID: "scope:root", Name: "id",
			Expression: "o.id", Inputs: []string{"col:id"},
		}},
		Operations: []Operation{
			{ID: "source", Kind: "source", ScopeID: "scope:root", Source: "orders", Alias: "o"},
			{ID: "filter", Kind: "filter", ScopeID: "scope:root", Inputs: []string{"source"},
				Expressions: []string{"o.hidden_score > 50"},
			},
		},
	}
	if err := json.NewEncoder(os.Stdout).Encode(reply); err != nil {
		os.Exit(2)
	}
	os.Exit(0)
}
