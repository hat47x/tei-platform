package orderwebmember

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	webapicontract "tei/reference/web-api-contract"
	derivationcache "tei/tools/reference-derivation-cache"
)

var columnCast = map[string]string{
	"order_id": "bigint", "account_id": "bigint", "quantity": "bigint", "version": "bigint",
	"total_amount": "numeric", "cost": "numeric", "ordered_at": "timestamptz",
}

var sqlOperator = map[string]string{"eq": "=", "ne": "<>", "gt": ">", "ge": ">=", "lt": "<", "le": "<="}

var errBadRequest = errors.New("bad request")

// selectList renders the projected columns only; numeric columns are read as
// text so that amounts keep their decimal digits.
func selectList(p Projection, names []string) ([]string, error) {
	var cols []string
	for _, n := range names {
		c, ok := p.Column(n)
		if !ok {
			return nil, errBadRequest
		}
		if columnCast[c] == "numeric" {
			cols = append(cols, "o."+c+"::text")
		} else {
			cols = append(cols, "o."+c)
		}
	}
	return cols, nil
}

// memberQuery builds the member read. The row scope (the grant's accounts and
// its snapshot sequence) is applied before the client filter, order and limit.
func memberQuery(p Projection, plan webapicontract.ReadPlan, g Grant, orderID *int64, ignoreSnapshot bool, burn int) (string, []any, error) {
	cols, err := selectList(p, plan.Selected)
	if err != nil {
		return "", nil, err
	}
	args := []any{g.Accounts}
	where := []string{"o.account_id = ANY($1)"}
	if !ignoreSnapshot {
		args = append(args, g.Seq)
		where = append(where, fmt.Sprintf("o.created_seq <= $%d", len(args)))
	}
	if orderID != nil {
		args = append(args, *orderID)
		where = append(where, fmt.Sprintf("o.order_id = $%d", len(args)))
	}
	if f := plan.Filter; f != nil {
		col, ok := p.Column(f.FieldID)
		op, okOp := sqlOperator[f.Operator]
		if !ok || !okOp || f.Value == "null" {
			return "", nil, errBadRequest
		}
		v := f.Value
		if strings.HasPrefix(v, "'") {
			v = strings.ReplaceAll(v[1:len(v)-1], "''", "'")
		}
		args = append(args, v)
		cast := columnCast[col]
		if cast == "" {
			cast = "text"
		}
		where = append(where, fmt.Sprintf("o.%s %s $%d::%s", col, op, len(args), cast))
	}
	if burn > 0 {
		args = append(args, burn)
		where = append(where, fmt.Sprintf("(SELECT count(*) FROM generate_series(1, $%d)) >= 0", len(args)))
	}
	order := "o.ordered_at DESC, o.order_id DESC"
	if plan.Order != nil {
		col, ok := p.Column(plan.Order.FieldID)
		if !ok {
			return "", nil, errBadRequest
		}
		dir := "ASC"
		if plan.Order.Descending {
			dir = "DESC"
		}
		order = fmt.Sprintf("o.%s %s, o.order_id %s", col, dir, dir)
	}
	args = append(args, plan.Limit)
	q := fmt.Sprintf("SELECT %s FROM owdm.orders o WHERE %s ORDER BY %s LIMIT $%d",
		strings.Join(cols, ", "), strings.Join(where, " AND "), order, len(args))
	return q, args, nil
}

func scanProjected(rows pgx.Rows, names []string) ([]map[string]any, error) {
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return nil, err
		}
		m := make(map[string]any, len(names))
		for i, n := range names {
			if t, ok := vals[i].(time.Time); ok {
				m[n] = t.UTC().Format(time.RFC3339Nano)
			} else {
				m[n] = vals[i]
			}
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (u *unit) trusted(g Grant) webapicontract.TrustedReadContext {
	return webapicontract.TrustedReadContext{PrincipalID: g.PrincipalID,
		RowScopeID: fmt.Sprintf("accounts@seq%d", g.Seq), PolicyRevision: strconv.FormatInt(g.Version, 10)}
}

func (u *unit) readOrders(ctx context.Context, values url.Values, g Grant, orderID *int64) ([]map[string]any, webapicontract.ReadPlan, error) {
	plan, err := u.proj.BindRead(values, u.trusted(g))
	if err != nil {
		return nil, plan, errBadRequest
	}
	q, args, err := memberQuery(u.proj, plan, g, orderID, u.cfg.UnsafeIgnoreAuthzSnapshot, u.cfg.MemberQueryBurn)
	if err != nil {
		return nil, plan, err
	}
	rows, err := u.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, plan, err
	}
	out, err := scanProjected(rows, plan.Selected)
	return out, plan, err
}

func (u *unit) listOrders(w http.ResponseWriter, r *http.Request, g Grant) {
	out, _, err := u.readOrders(r.Context(), r.URL.Query(), g, nil)
	if errors.Is(err, errBadRequest) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "query-not-accepted"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unavailable"})
		return
	}
	if out == nil {
		out = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"orders": out})
}

// notFound is the one body for an order that does not exist and for an order
// outside the caller's scope, so the two cannot be told apart.
var notFound = map[string]string{"error": "not-found"}

func pathOrderID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	return id, err == nil && id > 0
}

func (u *unit) getOrder(w http.ResponseWriter, r *http.Request, g Grant) {
	id, ok := pathOrderID(r)
	if !ok {
		writeJSON(w, http.StatusNotFound, notFound)
		return
	}
	vals := url.Values{}
	if s := r.URL.Query().Get("$select"); s != "" {
		vals.Set("$select", s)
	}
	if len(r.URL.Query()) != len(vals) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "query-not-accepted"})
		return
	}
	out, _, err := u.readOrders(r.Context(), vals, g, &id)
	switch {
	case errors.Is(err, errBadRequest):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "query-not-accepted"})
	case err != nil:
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unavailable"})
	case len(out) == 0:
		writeJSON(w, http.StatusNotFound, notFound)
	default:
		writeJSON(w, http.StatusOK, out[0])
	}
}

// patchOrder accepts only the fields of the member projection's accepted
// list. The shape is checked before the DB is touched; ownership next (404),
// then the version (409).
func (u *unit) patchOrder(w http.ResponseWriter, r *http.Request, g Grant) {
	id, ok := pathOrderID(r)
	if !ok {
		writeJSON(w, http.StatusNotFound, notFound)
		return
	}
	var changes map[string]any
	if err := readBody(r, &changes); err != nil || len(changes) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "body-not-accepted"})
		return
	}
	ifMatch := strings.Trim(r.Header.Get("If-Match"), `"`)
	if ifMatch == "" {
		writeJSON(w, http.StatusPreconditionRequired, map[string]string{"error": "if-match-required"})
		return
	}
	if _, err := u.proj.BindUpdate("member.update-order", ifMatch, changes, g.PrincipalID); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "field-not-accepted"})
		return
	}
	expected, err := strconv.ParseInt(ifMatch, 10, 64)
	note, isString := changes["deliveryNote"].(string)
	if err != nil || (changes["deliveryNote"] != nil && !isString) || len(note) > 200 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "body-not-accepted"})
		return
	}
	var notePtr *string
	if isString {
		notePtr = &note
	}
	args := []any{notePtr, id, g.Accounts, expected}
	snap := ""
	if !u.cfg.UnsafeIgnoreAuthzSnapshot {
		args = append(args, g.Seq)
		snap = " AND created_seq <= $5"
	}
	var newVersion int64
	err = u.pool.QueryRow(r.Context(), `UPDATE owdm.orders SET delivery_note = $1, version = version + 1
		WHERE order_id = $2 AND account_id = ANY($3) AND version = $4`+snap+` RETURNING version`, args...).Scan(&newVersion)
	if err == nil {
		writeJSON(w, http.StatusOK, map[string]any{"orderId": id, "version": newVersion})
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unavailable"})
		return
	}
	var cur int64
	args = []any{id, g.Accounts}
	snap = ""
	if !u.cfg.UnsafeIgnoreAuthzSnapshot {
		args = append(args, g.Seq)
		snap = " AND created_seq <= $3"
	}
	if err := u.pool.QueryRow(r.Context(), `SELECT version FROM owdm.orders WHERE order_id = $1 AND account_id = ANY($2)`+snap, args...).Scan(&cur); err != nil {
		writeJSON(w, http.StatusNotFound, notFound)
		return
	}
	writeJSON(w, http.StatusConflict, map[string]any{"error": "version-conflict", "currentVersion": cur})
}

func (u *unit) loadStock(item string) func(context.Context) (StockValue, error) {
	return func(ctx context.Context) (StockValue, error) {
		v := StockValue{ItemCode: item}
		err := u.pool.QueryRow(ctx, `SELECT stock, version, clock_timestamp() FROM owdm.item WHERE item_code = $1`, item).
			Scan(&v.Stock, &v.Version, &v.AsOf)
		return v, err
	}
}

// StockDisplay is the member display of an item. AsOf is when the canonical
// row was read; Source says whether the value came from the cache.
type StockDisplay struct {
	StockValue
	Source         string `json:"source"`
	MaxStalenessMS int64  `json:"maxStalenessMs"`
}

func (u *unit) stockDisplay(w http.ResponseWriter, r *http.Request, g Grant) {
	item := r.PathValue("code")
	v, hit, err := u.stock.Get(r.Context(), item, u.loadStock(item))
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, notFound)
		return
	}
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unavailable"})
		return
	}
	src := "canonical"
	if hit {
		src = "cache"
	}
	v.AsOf = v.AsOf.UTC()
	writeJSON(w, http.StatusOK, StockDisplay{StockValue: v, Source: src, MaxStalenessMS: u.cfg.StockMaxAge.Milliseconds()})
}

// ReorderResult is the response of a reorder.
type ReorderResult struct {
	OrderID          int64  `json:"orderId,omitempty"`
	Error            string `json:"error,omitempty"`
	DecidedOn        string `json:"decidedOn"`
	CanonicalStock   int64  `json:"canonicalStock"`
	CanonicalVersion int64  `json:"canonicalVersion"`
	DisplayedVersion int64  `json:"displayedVersion"`
	DisplayReuse     string `json:"displayReuse"`
	StockVersion     int64  `json:"stockVersionAfter,omitempty"`
}

// reorder places a new order with the same item and quantity. The decision
// reads the canonical item row under a row lock; the version the member saw
// on the display is only compared (derivationcache.Reuse) and reported.
func (u *unit) reorder(w http.ResponseWriter, r *http.Request, g Grant) {
	id, ok := pathOrderID(r)
	if !ok {
		writeJSON(w, http.StatusNotFound, notFound)
		return
	}
	var body struct {
		DisplayedStockVersion json.Number `json:"displayedStockVersion"`
	}
	if err := readBody(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "body-not-accepted"})
		return
	}
	displayed, err := body.DisplayedStockVersion.Int64()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "body-not-accepted"})
		return
	}
	ctx := r.Context()
	tx, err := u.pool.Begin(ctx)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unavailable"})
		return
	}
	defer tx.Rollback(context.Background())
	var item, total string
	var qty int64
	if err := tx.QueryRow(ctx, `SELECT item_code, quantity, total_amount::text FROM owdm.orders
		WHERE order_id = $1 AND account_id = ANY($2) AND created_seq <= $3`, id, g.Accounts, g.Seq).Scan(&item, &qty, &total); err != nil {
		writeJSON(w, http.StatusNotFound, notFound)
		return
	}
	res := ReorderResult{DecidedOn: "canonical", DisplayedVersion: displayed}
	if err := tx.QueryRow(ctx, `SELECT stock, version FROM owdm.item WHERE item_code = $1 FOR UPDATE`, item).
		Scan(&res.CanonicalStock, &res.CanonicalVersion); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unavailable"})
		return
	}
	reuse := derivationcache.Reuse(derivationcache.Input{CacheRequested: true,
		Entry:   &derivationcache.Entry{RuleVersion: "reorder-v1", InputVersion: strconv.FormatInt(displayed, 10), ContextClass: "member-display", SourceTimePoint: "item"},
		Current: &derivationcache.Context{RuleVersion: "reorder-v1", InputVersion: strconv.FormatInt(res.CanonicalVersion, 10), ContextClass: "member-display", SourceTimePoint: "item"}})
	res.DisplayReuse = reuse.Code
	if res.CanonicalStock < qty {
		res.Error = "insufficient-stock"
		writeJSON(w, http.StatusConflict, res)
		return
	}
	var seq int64
	err = tx.QueryRow(ctx, `UPDATE owdm.item SET stock = stock - $2, version = version + 1 WHERE item_code = $1 RETURNING version`, item, qty).Scan(&res.StockVersion)
	if err == nil {
		_, err = tx.Exec(ctx, `INSERT INTO owdm.item_version_log VALUES ($1, $2, $3)`, item, res.StockVersion, res.CanonicalStock-qty)
	}
	if err == nil {
		err = tx.QueryRow(ctx, `UPDATE owdm.clock SET seq = seq + 1 WHERE id = 1 RETURNING seq`).Scan(&seq)
	}
	if err == nil {
		err = tx.QueryRow(ctx, `INSERT INTO owdm.orders (order_id, account_id, ordered_at, status, item_code, quantity, total_amount, version, created_seq)
			SELECT nextval('owdm.order_id_seq'), account_id, clock_timestamp(), 'pending', item_code, quantity, total_amount, 1, $2
			FROM owdm.orders WHERE order_id = $1 RETURNING order_id`, id, seq).Scan(&res.OrderID)
	}
	if err == nil {
		_, err = tx.Exec(ctx, `SELECT pg_notify('owdm_stock', $1)`, fmt.Sprintf("%s:%d", item, res.StockVersion))
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unavailable"})
		return
	}
	// Read-your-writes: the caller's next read takes a fresh grant (with a
	// sequence that includes the new order), and the display is reloaded.
	u.authz.InvalidatePrincipal(g.PrincipalID)
	u.stock.Forget(item)
	writeJSON(w, http.StatusCreated, res)
}
