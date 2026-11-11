package standard

import (
	"strings"
	"testing"
)

func TestParameterRoundTrip(t *testing.T) {
	cases := []struct {
		name   string
		sql    string
		expect string
	}{
		{
			name:   "question_mark",
			sql:    "SELECT order_id FROM orders WHERE status = ?",
			expect: "SELECT order_id FROM orders WHERE (status = ?)",
		},
		{
			name:   "named_parameter",
			sql:    "SELECT order_id FROM orders WHERE status = :status",
			expect: "SELECT order_id FROM orders WHERE (status = :status)",
		},
		{
			name:   "parameter_in_expression",
			sql:    "SELECT order_id FROM orders WHERE amount > :min AND amount < ?",
			expect: "SELECT order_id FROM orders WHERE ((amount > :min) AND (amount < ?))",
		},
		{
			name:   "parameter_in_in_list",
			sql:    "SELECT order_id FROM orders WHERE customer_id IN (?, ?)",
			expect: "SELECT order_id FROM orders WHERE (customer_id IN (?, ?))",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q, diags := ParseSQLToStandard(tc.sql, "")
			for _, d := range diags {
				if d.Severity == SeverityError {
					t.Fatalf("parse error: %s", d.Message)
				}
			}

			generated, gdiags := StandardToSQL(q)
			for _, d := range gdiags {
				if d.Severity == SeverityError {
					t.Fatalf("generate error: %s", d.Message)
				}
			}

			if generated != tc.expect {
				t.Fatalf("SQL mismatch:\n  got:      %s\n  expected: %s", generated, tc.expect)
			}

			// Also verify parameters are preserved in the IR.
			found := findParameters(q)
			for _, p := range []string{"?", ":status", ":min"} {
				if strings.Contains(tc.sql, p) && !found[p] {
					t.Fatalf("parameter %q not found in IR", p)
				}
			}
		})
	}
}

func findParameters(q *StandardQuery) map[string]bool {
	found := map[string]bool{}
	if q == nil {
		return found
	}
	for _, n := range q.Nodes {
		collectParameters(n, found)
	}
	return found
}

func collectParameters(n Node, found map[string]bool) {
	switch v := n.(type) {
	case *Project:
		for _, item := range v.Items {
			scanExpr(item.Expr, found)
		}
	case *Filter:
		scanExpr(v.Condition, found)
	case *Aggregate:
		for _, item := range v.Items {
			scanExpr(item.Expr, found)
		}
		for _, g := range v.Grouping {
			scanExpr(g, found)
		}
	case *Sort:
		for _, k := range v.Keys {
			scanExpr(k.Expr, found)
		}
	case *Join:
		scanExpr(v.Condition, found)
	case *Subquery:
		for _, cn := range v.Query.Nodes {
			collectParameters(cn, found)
		}
	case *SubqueryScan:
		// no expr fields
	}
}

func scanExpr(e Expr, found map[string]bool) {
	if e == nil {
		return
	}
	switch v := e.(type) {
	case *Parameter:
		found[v.Name] = true
	case *BinaryOp:
		scanExpr(v.Left, found)
		scanExpr(v.Right, found)
	case *UnaryOp:
		scanExpr(v.Operand, found)
	case *FunctionCall:
		for _, a := range v.Args {
			scanExpr(a, found)
		}
	case *IsNull:
		scanExpr(v.Expr, found)
	case *InList:
		scanExpr(v.Expr, found)
		for _, it := range v.List {
			scanExpr(it, found)
		}
		if v.Subquery != nil {
			for _, cn := range v.Subquery.Nodes {
				collectParameters(cn, found)
			}
		}
	case *Between:
		scanExpr(v.Expr, found)
		scanExpr(v.Low, found)
		scanExpr(v.High, found)
	case *Exists:
		if v.Query != nil {
			for _, cn := range v.Query.Nodes {
				collectParameters(cn, found)
			}
		}
	case *Cast:
		scanExpr(v.Expr, found)
	case *Case:
		if v.Value != nil {
			scanExpr(v.Value, found)
		}
		for _, w := range v.Whens {
			scanExpr(w.Cond, found)
			scanExpr(w.Then, found)
		}
		if v.Else != nil {
			scanExpr(v.Else, found)
		}
	case *SubqueryExpr:
		if v.Query != nil {
			for _, cn := range v.Query.Nodes {
				collectParameters(cn, found)
			}
		}
	case *Column:
		// no parameters
	case *Literal:
		// no parameters
	}
}
