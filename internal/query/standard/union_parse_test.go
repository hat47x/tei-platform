package standard

import (
	"testing"
)

func TestUnionParseIR(t *testing.T) {
	sql := "SELECT customer_id FROM orders WHERE status = 'OPEN' UNION ALL SELECT id FROM customers WHERE active = 1"
	q, diags := ParseSQLToStandard(sql, "")
	for _, d := range diags {
		if d.Severity == SeverityError {
			t.Fatalf("parse error: %s", d.Message)
		}
	}
	if q == nil {
		t.Fatal("expected non-nil query")
	}

	// Verify UNION node exists in IR
	var unionCount int
	var hasLeft, hasRight bool
	for _, n := range q.Nodes {
		if u, ok := n.(*Union); ok {
			unionCount++
			if u.NodeID == "" {
				t.Fatal("union node id is empty")
			}
			if u.Left == "" {
				t.Fatal("union left is empty")
			}
			if u.Right == "" {
				t.Fatal("union right is empty")
			}
			hasLeft = u.Left != ""
			hasRight = u.Right != ""
		}
	}
	if unionCount != 1 {
		t.Fatalf("expected 1 Union node, got %d", unionCount)
	}
	if !hasLeft || !hasRight {
		t.Fatalf("union missing left/right: left=%v right=%v", hasLeft, hasRight)
	}

	// Verify both branches have a Scan node
	var scanCount int
	for _, n := range q.Nodes {
		if _, ok := n.(*Scan); ok {
			scanCount++
		}
	}
	if scanCount != 2 {
		t.Fatalf("expected 2 Scan nodes for UNION branches, got %d", scanCount)
	}
}
