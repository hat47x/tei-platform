package workbenchfixture

import (
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"
)

// This is a source-static contract test. It never connects to PostgreSQL,
// runs the product query, selects a plugin, or proves a Function Data Context.
// The independently authored expectation catches a mismatch between the
// existing generic sales_order fixture and the actual public.orders query.
func TestProductRecentOrdersFixtureSourceContract(t *testing.T) {
	seed, err := os.ReadFile("recent_orders_product_seed.sql")
	if err != nil {
		t.Fatal(err)
	}
	query, err := os.ReadFile("../../reference/workspace/applications/order-entry/queries/recent_orders.sql")
	if err != nil {
		t.Fatal(err)
	}
	decl, err := os.ReadFile("../../reference/workspace/applications/order-entry/queries/recent_orders.tei")
	if err != nil {
		t.Fatal(err)
	}
	app, err := os.ReadFile("../../reference/workspace/applications/order-entry/application.tei")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"CREATE TABLE public.orders (",
		"order_id TEXT PRIMARY KEY",
		"customer_id TEXT NOT NULL",
		"ordered_at TIMESTAMPTZ NOT NULL",
		"status TEXT NOT NULL",
		"total_amount NUMERIC(12,2) NOT NULL",
		"INSERT INTO public.orders (order_id, customer_id, ordered_at, status, total_amount)",
	} {
		if !strings.Contains(string(seed), required) {
			t.Errorf("synthetic product seed lacks %q", required)
		}
	}
	for _, required := range []string{
		"o.order_id", "o.customer_id", "o.ordered_at",
		"o.status", "o.total_amount",
		"FROM public.orders AS o",
		"WHERE o.ordered_at >= :from",
		"ORDER BY o.ordered_at DESC",
	} {
		if !strings.Contains(string(query), required) {
			t.Errorf("actual product query changed or lost %q", required)
		}
	}
	for _, required := range []string{
		"query order_entry.recent_orders",
		"input from std.datetime",
		"datasource sales",
		"sql postgres file \"recent_orders.sql\"",
		"column order_id -> sales.order.order_id",
		"column customer_id -> sales.order.customer_id",
		"column ordered_at -> sales.order.ordered_at",
		"column status -> sales.order.status",
		"column total_amount -> sales.order.total",
	} {
		if !strings.Contains(string(decl), required) {
			t.Errorf("query result declaration changed or lost %q", required)
		}
	}
	if !strings.Contains(string(app), "database sales {") ||
		!strings.Contains(string(app), "target sales_core") {
		t.Error("application datasource alias/target mapping changed")
	}

	// Constrain the fixture's three independently authored physical rows.
	// This is NOT a SQL parser or a PostgreSQL SELECT execution.
	rowRE := regexp.MustCompile(`(?m)^\s*\('([^']+)', '([^']+)', '([^']+)', '([^']+)', ([0-9]+\.[0-9]{2})\)[,;]\s*$`)
	matches := rowRE.FindAllStringSubmatch(string(seed), -1)
	if len(matches) != 3 {
		t.Fatalf("want exactly 3 physical seed rows, got %d", len(matches))
	}
	type row struct {
		id, customer, status, amount string
		orderedAt                    time.Time
	}
	rows := make([]row, 0, len(matches))
	for _, m := range matches {
		ts, err := time.Parse(time.RFC3339, m[3])
		if err != nil {
			t.Fatalf("invalid timestamp in %s: %v", m[1], err)
		}
		rows = append(rows, row{id: m[1], customer: m[2], orderedAt: ts, status: m[4], amount: m[5]})
	}
	want := []row{
		{id: "O-001", customer: "C-101", orderedAt: time.Date(2026, 11, 7, 10, 0, 0, 0, time.UTC), status: "accepted", amount: "120.50"},
		{id: "O-002", customer: "C-101", orderedAt: time.Date(2026, 11, 7, 10, 5, 0, 0, time.UTC), status: "pending", amount: "0.00"},
		{id: "O-003", customer: "C-202", orderedAt: time.Date(2026, 11, 7, 10, 10, 0, 0, time.UTC), status: "accepted", amount: "5.25"},
	}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("physical seed input drifted: got %#v, want %#v", rows, want)
	}
	for _, tc := range []struct {
		from string
		ids  []string
	}{
		{"2026-11-07T09:00:00Z", []string{"O-003", "O-002", "O-001"}},
		{"2026-11-07T10:05:00Z", []string{"O-003", "O-002"}},
		{"2026-11-07T10:10:01Z", []string{}},
	} {
		from, err := time.Parse(time.RFC3339, tc.from)
		if err != nil {
			t.Fatal(err)
		}
		var selected []row
		for _, r := range rows {
			if !r.orderedAt.Before(from) {
				selected = append(selected, r)
			}
		}
		sort.Slice(selected, func(i, j int) bool {
			return selected[i].orderedAt.After(selected[j].orderedAt)
		})
		got := make([]string, 0, len(selected))
		for _, r := range selected {
			got = append(got, r.id)
		}
		if !reflect.DeepEqual(got, tc.ids) {
			t.Errorf("source-only expectation for :from=%s: IDs %v, want %v", tc.from, got, tc.ids)
		}
	}
}
