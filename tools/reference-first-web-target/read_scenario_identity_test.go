//go:build !tei_repair_only

package main

import (
	"strings"
	"testing"

	"tei/internal/core"
)

// Attributes in a page are only attributable to a read when the page carries
// that read's identity. A complete identity over the matching rows is accepted;
// an unnamed, mis-sized or misordered one is refused. The expectations are
// written from the contract, not copied from a rendered page.
func TestRequireAnswersBelongToRead(t *testing.T) {
	complete := readIdentityProjection{
		SelectedOrderKey: "1001",
		Environment:      "development",
		Pattern:          "order_entry.order_lines",
		Database:         "sales_core",
		Contract:         "sales_core.order_line",
		Query:            "find_by_order",
		InputOrderID:     "1001",
		RowCount:         2,
	}
	aligned := []rowAnswerProjection{{RowIndex: 0}, {RowIndex: 1}}

	for _, tc := range []struct {
		name     string
		identity readIdentityProjection
		rows     []rowAnswerProjection
		refuse   bool
		mention  string
	}{
		{name: "complete and aligned", identity: complete, rows: aligned},
		{name: "no order key", identity: withOrderKey(complete, ""), rows: aligned, refuse: true, mention: "no complete identity"},
		{name: "no environment", identity: func() readIdentityProjection { v := complete; v.Environment = ""; return v }(), rows: aligned, refuse: true, mention: "no complete identity"},
		{name: "no pattern", identity: func() readIdentityProjection { v := complete; v.Pattern = ""; return v }(), rows: aligned, refuse: true, mention: "no complete identity"},
		{name: "no database", identity: withDatabase(complete, ""), rows: aligned, refuse: true, mention: "no complete identity"},
		{name: "no query", identity: withQuery(complete, ""), rows: aligned, refuse: true, mention: "no complete identity"},
		{name: "no input", identity: func() readIdentityProjection { v := complete; v.InputOrderID = ""; return v }(), rows: aligned, refuse: true, mention: "no complete identity"},
		{name: "conflicting order and query input", identity: func() readIdentityProjection { v := complete; v.InputOrderID = "1002"; return v }(), rows: aligned, refuse: true, mention: "differs from query input"},
		{name: "row count disagrees", identity: withRowCount(complete, 3), rows: aligned, refuse: true, mention: "states 3 rows but 2"},
		{name: "positions reordered", identity: complete, rows: []rowAnswerProjection{{RowIndex: 1}, {RowIndex: 0}}, refuse: true, mention: "belongs to position"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := requireAnswersBelongToRead(tc.identity, tc.rows, nil)
			if tc.refuse {
				if err == nil {
					t.Fatalf("%s was accepted", tc.name)
				}
				if !strings.Contains(err.Error(), tc.mention) {
					t.Fatalf("refusal %q does not mention %q", err, tc.mention)
				}
				return
			}
			if err != nil {
				t.Fatalf("%s was refused: %v", tc.name, err)
			}
		})
	}
}

// Different declared read scopes must not share a key. An equal key only
// identifies the same scope; it does not identify a particular invocation or
// prove that the results came from the same database snapshot.
func TestReadIdentityKeySeparatesDeclaredScopes(t *testing.T) {
	base := readIdentityProjection{
		SelectedOrderKey: "1001", Environment: "development",
		Pattern: "order_entry.order_lines", Database: "sales_core",
		Contract: "sales_core.order_line", Query: "find_by_order",
		InputOrderID: "1001", RowCount: 2,
	}
	first := readIdentityKey(base, "development")
	for _, tc := range []struct {
		name     string
		identity readIdentityProjection
		env      core.CanonicalID
	}{
		{name: "other selected order", identity: withOrderKey(base, "1002"), env: "development"},
		{name: "other input", identity: func() readIdentityProjection { v := base; v.InputOrderID = "1002"; return v }(), env: "development"},
		{name: "other environment", identity: func() readIdentityProjection { v := base; v.Environment = "staging"; return v }(), env: "development"},
		{name: "other effective environment", identity: base, env: "staging"},
		{name: "other pattern", identity: func() readIdentityProjection { v := base; v.Pattern = "other"; return v }(), env: "development"},
		{name: "other database", identity: withDatabase(base, "other"), env: "development"},
		{name: "other contract", identity: func() readIdentityProjection { v := base; v.Contract = "other"; return v }(), env: "development"},
		{name: "other query", identity: withQuery(base, "other"), env: "development"},
		{name: "other row count", identity: withRowCount(base, 3), env: "development"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if other := readIdentityKey(tc.identity, tc.env); first == other {
				t.Fatalf("different scopes produced the same key: %q", other)
			}
		})
	}
	if first != readIdentityKey(base, "development") {
		t.Fatal("identical scope produced different keys")
	}
	// A separator inside a field must not alias a change to an adjacent field.
	mixed := base
	mixed.Pattern = "a|b"
	mixed.Database = "c"
	split := base
	split.Pattern = "a"
	split.Database = "b|c"
	if readIdentityKey(mixed, "development") == readIdentityKey(split, "development") {
		t.Fatal("scope fields collided across a separator")
	}
}

func withOrderKey(i readIdentityProjection, key string) readIdentityProjection {
	i.SelectedOrderKey = key
	return i
}

func withDatabase(i readIdentityProjection, database string) readIdentityProjection {
	i.Database = database
	return i
}

func withQuery(i readIdentityProjection, query string) readIdentityProjection {
	i.Query = query
	return i
}

func withRowCount(i readIdentityProjection, count int) readIdentityProjection {
	i.RowCount = count
	return i
}
