//go:build !tei_repair_only

package main

import (
	"strings"
	"testing"
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
		{name: "no database", identity: withDatabase(complete, ""), rows: aligned, refuse: true, mention: "no complete identity"},
		{name: "no query", identity: withQuery(complete, ""), rows: aligned, refuse: true, mention: "no complete identity"},
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

// Two reads of the same shape are different reads: the key that separates them
// is the identity, so a page cannot be shown as if the answers came from the
// read whose rows it displays.
func TestReadIdentityKeySeparatesTwoReadsOfTheSameShape(t *testing.T) {
	base := readIdentityProjection{
		SelectedOrderKey: "1001", Database: "sales_core",
		Contract: "sales_core.order_line", Query: "find_by_order",
		InputOrderID: "1001", RowCount: 2,
	}
	first := readIdentityKey(base, "development")
	otherOrder := readIdentityKey(withOrderKey(base, "1002"), "development")
	if first == otherOrder {
		t.Fatalf("different orders produced the same identity key: %q", first)
	}
	otherRows := readIdentityKey(withRowCount(base, 3), "development")
	if first == otherRows {
		t.Fatal("a different row count produced the same identity key")
	}
	if first != readIdentityKey(base, "development") {
		t.Fatal("the same read produced two different keys")
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
