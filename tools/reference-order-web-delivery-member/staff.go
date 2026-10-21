package orderwebmember

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5"
)

func unavailable(w http.ResponseWriter) {
	writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "unavailable"})
}

// staffGetOrder reads one order with the staff projection (all fields).
func (u *unit) staffGetOrder(w http.ResponseWriter, r *http.Request, g Grant) {
	id, ok := pathOrderID(r)
	if !ok {
		writeJSON(w, http.StatusNotFound, notFound)
		return
	}
	plan, err := u.proj.BindRead(url.Values{}, u.trusted(g))
	if err != nil {
		unavailable(w)
		return
	}
	cols, err := selectList(u.proj, plan.Selected)
	if err != nil {
		unavailable(w)
		return
	}
	rows, err := u.pool.Query(r.Context(), "SELECT "+strings.Join(cols, ", ")+" FROM owdm.orders o WHERE o.order_id = $1", id)
	if err != nil {
		unavailable(w)
		return
	}
	out, err := scanProjected(rows, plan.Selected)
	if err != nil {
		unavailable(w)
		return
	}
	if len(out) == 0 {
		writeJSON(w, http.StatusNotFound, notFound)
		return
	}
	writeJSON(w, http.StatusOK, out[0])
}

// confirm is the internal confirm path whose latency the member surge must
// not push beyond the target.
func (u *unit) confirm(w http.ResponseWriter, r *http.Request, g Grant) {
	id, ok := pathOrderID(r)
	if !ok {
		writeJSON(w, http.StatusNotFound, notFound)
		return
	}
	var body struct {
		ExpectedVersion json.Number `json:"expectedVersion"`
	}
	if err := readBody(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "body-not-accepted"})
		return
	}
	expected, err := body.ExpectedVersion.Int64()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "body-not-accepted"})
		return
	}
	ctx := r.Context()
	tx, err := u.pool.Begin(ctx)
	if err != nil {
		unavailable(w)
		return
	}
	defer tx.Rollback(context.Background())
	var version int64
	err = tx.QueryRow(ctx, `UPDATE owdm.orders SET status = 'confirmed', version = version + 1, confirmed_by = $3, confirmed_at = clock_timestamp()
		WHERE order_id = $1 AND status = 'pending' AND version = $2 RETURNING version`, id, expected, g.PrincipalID).Scan(&version)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "not-pending-or-version-changed"})
		return
	}
	if err == nil {
		_, err = tx.Exec(ctx, `INSERT INTO owdm.confirm_audit (order_id, staff_id, version) VALUES ($1, $2, $3)`, id, g.PrincipalID, version)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		unavailable(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"orderId": id, "version": version, "status": "confirmed"})
}

// createOrder adds an order (used to add rows after a scope reduction).
func (u *unit) createOrder(w http.ResponseWriter, r *http.Request, g Grant) {
	var body struct {
		AccountID json.Number `json:"accountId"`
		ItemCode  string      `json:"itemCode"`
		Quantity  json.Number `json:"quantity"`
		Total     string      `json:"total"`
	}
	if err := readBody(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "body-not-accepted"})
		return
	}
	account, err1 := body.AccountID.Int64()
	qty, err2 := body.Quantity.Int64()
	if err1 != nil || err2 != nil || body.ItemCode == "" || body.Total == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "body-not-accepted"})
		return
	}
	ctx := r.Context()
	tx, err := u.pool.Begin(ctx)
	if err != nil {
		unavailable(w)
		return
	}
	defer tx.Rollback(context.Background())
	var seq, id int64
	err = tx.QueryRow(ctx, `UPDATE owdm.clock SET seq = seq + 1 WHERE id = 1 RETURNING seq`).Scan(&seq)
	if err == nil {
		err = tx.QueryRow(ctx, `INSERT INTO owdm.orders VALUES (nextval('owdm.order_id_seq'), $1, clock_timestamp(), 'pending', $2, $3, $4::numeric, NULL, 1, $5,
			`+SentinelCost+`, '`+SentinelCredit+`', '`+SentinelMemo+`' || currval('owdm.order_id_seq'), NULL, NULL) RETURNING order_id`,
			account, body.ItemCode, qty, body.Total, seq).Scan(&id)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		unavailable(w)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"orderId": id, "createdSeq": seq})
}

// revoke deactivates a principal. The change and its NOTIFY commit together.
func (u *unit) revoke(w http.ResponseWriter, r *http.Request, g Grant) {
	principal := r.PathValue("id")
	ctx := r.Context()
	tx, err := u.pool.Begin(ctx)
	if err != nil {
		unavailable(w)
		return
	}
	defer tx.Rollback(context.Background())
	var version int64
	err = tx.QueryRow(ctx, `UPDATE owdm.principal SET active = false, authz_version = authz_version + 1 WHERE principal_id = $1 RETURNING authz_version`, principal).Scan(&version)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, notFound)
		return
	}
	if err == nil {
		_, err = tx.Exec(ctx, `SELECT pg_notify('owdm_authz', $1)`, principal)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		unavailable(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"principalId": principal, "authzVersion": version, "active": false})
}

// removeScope removes one account from a principal's scope. The clock moves
// in the same transaction, so rows added afterwards have a higher sequence
// than any grant read before the removal.
func (u *unit) removeScope(w http.ResponseWriter, r *http.Request, g Grant) {
	principal := r.PathValue("id")
	var body struct {
		AccountID json.Number `json:"accountId"`
	}
	if err := readBody(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "body-not-accepted"})
		return
	}
	account, err := body.AccountID.Int64()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "body-not-accepted"})
		return
	}
	ctx := r.Context()
	tx, err := u.pool.Begin(ctx)
	if err != nil {
		unavailable(w)
		return
	}
	defer tx.Rollback(context.Background())
	tag, err := tx.Exec(ctx, `DELETE FROM owdm.principal_scope WHERE principal_id = $1 AND account_id = $2`, principal, account)
	if err == nil && tag.RowsAffected() != 1 {
		writeJSON(w, http.StatusNotFound, notFound)
		return
	}
	var version, seq int64
	if err == nil {
		err = tx.QueryRow(ctx, `UPDATE owdm.principal SET authz_version = authz_version + 1 WHERE principal_id = $1 RETURNING authz_version`, principal).Scan(&version)
	}
	if err == nil {
		err = tx.QueryRow(ctx, `UPDATE owdm.clock SET seq = seq + 1 WHERE id = 1 RETURNING seq`).Scan(&seq)
	}
	if err == nil {
		_, err = tx.Exec(ctx, `SELECT pg_notify('owdm_authz', $1)`, principal)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		unavailable(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"principalId": principal, "authzVersion": version, "seq": seq})
}

// adjustStock changes the canonical stock (the writer of the cache tests).
func (u *unit) adjustStock(w http.ResponseWriter, r *http.Request, g Grant) {
	item := r.PathValue("code")
	var body struct {
		Delta json.Number `json:"delta"`
	}
	if err := readBody(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "body-not-accepted"})
		return
	}
	delta, err := body.Delta.Int64()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "body-not-accepted"})
		return
	}
	ctx := r.Context()
	tx, err := u.pool.Begin(ctx)
	if err != nil {
		unavailable(w)
		return
	}
	defer tx.Rollback(context.Background())
	var stock, version int64
	err = tx.QueryRow(ctx, `UPDATE owdm.item SET stock = stock + $2, version = version + 1 WHERE item_code = $1 RETURNING stock, version`, item, delta).Scan(&stock, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, notFound)
		return
	}
	if err == nil {
		_, err = tx.Exec(ctx, `INSERT INTO owdm.item_version_log VALUES ($1, $2, $3)`, item, version, stock)
	}
	if err == nil {
		_, err = tx.Exec(ctx, `SELECT pg_notify('owdm_stock', $1)`, fmt.Sprintf("%s:%d", item, version))
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "stock-not-changed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"itemCode": item, "stock": stock, "version": version})
}
