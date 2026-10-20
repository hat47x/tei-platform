package ordermigrationdualdb

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// tableSet names the order/line tables and an optional fixed filter column
// (plan_digest for the staged copy).
type tableSet struct {
	orders, lines string
	digestFilter  bool
}

var (
	legacyTables  = tableSet{orders: "omdd.legacy_order", lines: "omdd.legacy_order_line"}
	stageTables   = tableSet{orders: "omdd.stage_order", lines: "omdd.stage_order_line", digestFilter: true}
	primaryTables = tableSet{orders: "omdd.order_backlog", lines: "omdd.order_backlog_line"}
)

// keyRange restricts loading to one chunk; zero value loads everything.
type keyRange struct{ from, to, exact string }

func chunkRange(c Chunk) keyRange { return keyRange{from: c.FromKey, to: c.ToKeyExclusive} }

// loadOrders reads orders and lines ordered by key in byte order.
func loadOrders(ctx context.Context, q querier, ts tableSet, planDigest string, r keyRange) ([]Order, error) {
	where, args := "TRUE", []any{}
	add := func(cond string, v any) {
		args = append(args, v)
		where += fmt.Sprintf(" AND "+cond, len(args))
	}
	if ts.digestFilter {
		add("o.plan_digest = $%d", planDigest)
	}
	if r.exact != "" {
		add("o.order_no = $%d", r.exact)
	}
	if r.from != "" {
		add(`o.order_no COLLATE "C" >= $%d`, r.from)
	}
	if r.to != "" {
		add(`o.order_no COLLATE "C" < $%d`, r.to)
	}
	rows, err := q.Query(ctx, fmt.Sprintf(`SELECT o.order_no, o.customer_code, to_char(o.ordered_on,'YYYY-MM-DD'), o.status
		FROM %s o WHERE %s ORDER BY o.order_no COLLATE "C"`, ts.orders, where), args...)
	if err != nil {
		return nil, err
	}
	var orders []Order
	index := map[string]int{}
	for rows.Next() {
		var o Order
		if err := rows.Scan(&o.OrderNo, &o.CustomerCode, &o.OrderedOn, &o.Status); err != nil {
			rows.Close()
			return nil, err
		}
		index[o.OrderNo] = len(orders)
		orders = append(orders, o)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	join := "l.order_no = o.order_no"
	if ts.digestFilter {
		join += " AND l.plan_digest = o.plan_digest"
	}
	rows, err = q.Query(ctx, fmt.Sprintf(`SELECT l.order_no, l.line_no, l.item_code, l.qty, l.unit_price::text, l.amount::text
		FROM %s l JOIN %s o ON %s WHERE %s ORDER BY l.order_no COLLATE "C", l.line_no`, ts.lines, ts.orders, join, where), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		var l Line
		if err := rows.Scan(&key, &l.LineNo, &l.ItemCode, &l.Qty, &l.UnitPrice, &l.Amount); err != nil {
			return nil, err
		}
		i, ok := index[key]
		if !ok {
			return nil, fmt.Errorf("line for unknown order %q", key)
		}
		orders[i].Lines = append(orders[i].Lines, l)
	}
	return orders, rows.Err()
}
