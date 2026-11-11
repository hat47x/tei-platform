package standard

import (
	"testing"
)

func TestNodeSpans(t *testing.T) {
	q, diags := ParseSQLToStandard("SELECT id FROM users WHERE active = 1 ORDER BY id LIMIT 10", "")
	for _, d := range diags {
		if d.Severity == SeverityError {
			t.Fatalf("parse error: %s", d.Message)
		}
	}

	var project, scan, filter, sort, limit Node
	for i := range q.Nodes {
		switch q.Nodes[i].(type) {
		case *Project:
			project = q.Nodes[i]
		case *Scan:
			scan = q.Nodes[i]
		case *Filter:
			filter = q.Nodes[i]
		case *Sort:
			sort = q.Nodes[i]
		case *Limit:
			limit = q.Nodes[i]
		}
	}

	if project == nil {
		t.Fatal("expected Project node")
	}
	if scan == nil {
		t.Fatal("expected Scan node")
	}
	if filter == nil {
		t.Fatal("expected Filter node")
	}
	if sort == nil {
		t.Fatal("expected Sort node")
	}
	if limit == nil {
		t.Fatal("expected Limit node")
	}

	checks := []struct {
		name string
		span SourceSpan
	}{
		{"Project", project.(*Project).Span},
		{"Scan", scan.(*Scan).Span},
		{"Filter", filter.(*Filter).Span},
		{"Sort", sort.(*Sort).Span},
		{"Limit", limit.(*Limit).Span},
	}
	for _, c := range checks {
		if c.span.Start == 0 || c.span.End == 0 {
			t.Fatalf("%s span is empty: %+v", c.name, c.span)
		}
		if c.span.Start > c.span.End {
			t.Fatalf("%s span is inverted: %+v", c.name, c.span)
		}
	}
}
