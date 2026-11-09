package orderassistoffline

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CreditProvider simulates the external credit service. It runs as its own
// process with its own DB role and schema. Reservations are deduplicated by
// the Idempotency-Key header; a configured fault commits the consumption (or
// the release) and then closes the connection without answering (P-TX-02).
type CreditProvider struct {
	Pool *pgxpool.Pool
}

// Reservation is the provider's view of one credit consumption.
type Reservation struct {
	ReservationID int64  `json:"reservationId"`
	CustomerCode  string `json:"customerCode"`
	AmountMinor   int64  `json:"amountMinor"`
	QuoteRef      string `json:"quoteRef"`
	Status        string `json:"status"` // consumed | released
	ReleaseCount  int    `json:"releaseCount"`
	Replayed      bool   `json:"replayed,omitempty"`
}

type reserveBody struct {
	CustomerCode string `json:"customerCode"`
	AmountMinor  int64  `json:"amountMinor"`
	QuoteRef     string `json:"quoteRef"`
}

// SeedAccount sets a customer's credit limit.
func (p *CreditProvider) SeedAccount(ctx context.Context, customer string, limit int64) error {
	_, err := p.Pool.Exec(ctx, `INSERT INTO oapo_credit.account VALUES ($1,$2,0)`, customer, limit)
	return err
}

func (p *CreditProvider) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /reservations", p.reserve)
	mux.HandleFunc("GET /reservations/{key}", p.inquire)
	mux.HandleFunc("POST /reservations/{key}/release", p.release)
	return mux
}

func scanReservation(row pgx.Row) (Reservation, error) {
	var r Reservation
	err := row.Scan(&r.ReservationID, &r.CustomerCode, &r.AmountMinor, &r.QuoteRef, &r.Status, &r.ReleaseCount)
	return r, err
}

const reservationColumns = `reservation_id, customer_code, amount_minor, quote_ref, status, release_count`

// takeFault consumes one configured fault for the operation, inside the
// transaction that commits the effect.
func takeFault(ctx context.Context, tx pgx.Tx, op string) (bool, error) {
	tag, err := tx.Exec(ctx, `UPDATE oapo_credit.fault SET remaining=remaining-1 WHERE op=$1 AND remaining>0`, op)
	return tag.RowsAffected() == 1, err
}

func logRequest(ctx context.Context, tx pgx.Tx, op, key, outcome string) error {
	_, err := tx.Exec(ctx, `INSERT INTO oapo_credit.request_log(op, idempotency_key, outcome) VALUES ($1,NULLIF($2,''),$3)`, op, key, outcome)
	return err
}

// dropConnection closes the client connection without writing a response.
func dropConnection(w http.ResponseWriter) {
	if hj, ok := w.(http.Hijacker); ok {
		if conn, _, err := hj.Hijack(); err == nil {
			_ = conn.Close()
			return
		}
	}
	panic(http.ErrAbortHandler)
}

func (p *CreditProvider) reserve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	key := r.Header.Get("Idempotency-Key")
	var body reserveBody
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil || body.CustomerCode == "" || body.AmountMinor <= 0 || body.QuoteRef == "" {
		writeJSON(w, 400, map[string]string{"code": "malformed"})
		return
	}
	var status int
	var out any
	var drop bool
	err := pgx.BeginFunc(ctx, p.Pool, func(tx pgx.Tx) error {
		var limit, used int64
		if err := tx.QueryRow(ctx, `SELECT limit_minor, used_minor FROM oapo_credit.account WHERE customer_code=$1 FOR UPDATE`,
			body.CustomerCode).Scan(&limit, &used); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				status, out = 404, map[string]string{"code": "unknown-customer"}
				return logRequest(ctx, tx, "reserve", key, "unknown-customer")
			}
			return err
		}
		if key != "" {
			existing, err := scanReservation(tx.QueryRow(ctx, `SELECT `+reservationColumns+` FROM oapo_credit.reservation WHERE idempotency_key=$1`, key))
			if err == nil {
				if existing.CustomerCode != body.CustomerCode || existing.AmountMinor != body.AmountMinor || existing.QuoteRef != body.QuoteRef {
					status, out = 422, map[string]string{"code": "idempotency-key-reused"}
					return logRequest(ctx, tx, "reserve", key, "key-mismatch")
				}
				existing.Replayed = true
				status, out = 200, existing
				var ferr error
				if drop, ferr = takeFault(ctx, tx, "reserve"); ferr != nil {
					return ferr
				}
				return logRequest(ctx, tx, "reserve", key, "replayed")
			} else if !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
		}
		if used+body.AmountMinor > limit {
			status, out = 409, map[string]string{"code": "limit-exceeded"}
			return logRequest(ctx, tx, "reserve", key, "limit-exceeded")
		}
		res, err := scanReservation(tx.QueryRow(ctx, `INSERT INTO oapo_credit.reservation(idempotency_key, customer_code, amount_minor, quote_ref, status)
			VALUES (NULLIF($1,''),$2,$3,$4,'consumed') RETURNING `+reservationColumns, key, body.CustomerCode, body.AmountMinor, body.QuoteRef))
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE oapo_credit.account SET used_minor=used_minor+$2 WHERE customer_code=$1`, body.CustomerCode, body.AmountMinor); err != nil {
			return err
		}
		status, out = 201, res
		if drop, err = takeFault(ctx, tx, "reserve"); err != nil {
			return err
		}
		return logRequest(ctx, tx, "reserve", key, "consumed")
	})
	if err != nil {
		writeJSON(w, 500, map[string]string{"code": "internal"})
		return
	}
	if drop {
		dropConnection(w)
		return
	}
	writeJSON(w, status, out)
}

func (p *CreditProvider) inquire(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	key := r.PathValue("key")
	res, err := scanReservation(p.Pool.QueryRow(ctx, `SELECT `+reservationColumns+` FROM oapo_credit.reservation WHERE idempotency_key=$1`, key))
	_ = pgx.BeginFunc(ctx, p.Pool, func(tx pgx.Tx) error {
		outcome := "found"
		if err != nil {
			outcome = "not-found"
		}
		return logRequest(ctx, tx, "inquire", key, outcome)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, 404, map[string]string{"code": "not-found"})
		return
	} else if err != nil {
		writeJSON(w, 500, map[string]string{"code": "internal"})
		return
	}
	writeJSON(w, 200, res)
}

func (p *CreditProvider) release(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	key := r.PathValue("key")
	var status int
	var out any
	var drop bool
	err := pgx.BeginFunc(ctx, p.Pool, func(tx pgx.Tx) error {
		res, err := scanReservation(tx.QueryRow(ctx, `SELECT `+reservationColumns+` FROM oapo_credit.reservation WHERE idempotency_key=$1 FOR UPDATE`, key))
		if errors.Is(err, pgx.ErrNoRows) {
			status, out = 404, map[string]string{"code": "not-found"}
			return logRequest(ctx, tx, "release", key, "not-found")
		} else if err != nil {
			return err
		}
		outcome := "replayed"
		if res.Status == "consumed" {
			if _, err := tx.Exec(ctx, `UPDATE oapo_credit.account SET used_minor=used_minor-$2 WHERE customer_code=$1`, res.CustomerCode, res.AmountMinor); err != nil {
				return err
			}
			if res, err = scanReservation(tx.QueryRow(ctx, `UPDATE oapo_credit.reservation SET status='released', release_count=release_count+1
				WHERE reservation_id=$1 RETURNING `+reservationColumns, res.ReservationID)); err != nil {
				return err
			}
			outcome = "released"
		} else {
			res.Replayed = true
		}
		status, out = 200, res
		if drop, err = takeFault(ctx, tx, "release"); err != nil {
			return err
		}
		return logRequest(ctx, tx, "release", key, outcome)
	})
	if err != nil {
		writeJSON(w, 500, map[string]string{"code": "internal"})
		return
	}
	if drop {
		dropConnection(w)
		return
	}
	writeJSON(w, status, out)
}

// Credit client errors.
var (
	// ErrOutcomeUnknown: the request was sent and no answer arrived. The
	// provider may or may not have applied it.
	ErrOutcomeUnknown = errors.New("credit outcome unknown")
	ErrCreditRefused  = errors.New("credit refused")
	ErrNotFound       = errors.New("reservation not found")
)

// CreditClient calls the provider. Keep-alive is disabled so the standard
// transport never resends a request on its own; every resend is the saga's.
type CreditClient struct {
	BaseURL string
	HTTP    *http.Client
	// UnsafeNoIdempotencyKey omits the key (negative control for P-TX-02).
	UnsafeNoIdempotencyKey bool
}

// NewCreditClient builds a client. The timeout is a safety bound only; the
// injected fault closes the connection, it does not stall it.
func NewCreditClient(base string) *CreditClient {
	return &CreditClient{BaseURL: base, HTTP: &http.Client{Timeout: 60 * time.Second,
		Transport: &http.Transport{DisableKeepAlives: true}}}
}

func (c *CreditClient) do(ctx context.Context, method, path, key string, body any) (Reservation, error) {
	var rd io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		rd = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, rd)
	if err != nil {
		return Reservation{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if key != "" && !c.UnsafeNoIdempotencyKey {
		req.Header.Set("Idempotency-Key", key)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return Reservation{}, ctx.Err()
		}
		return Reservation{}, fmt.Errorf("%w: %v", ErrOutcomeUnknown, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if err != nil {
		return Reservation{}, fmt.Errorf("%w: %v", ErrOutcomeUnknown, err)
	}
	switch {
	case resp.StatusCode == 404:
		return Reservation{}, ErrNotFound
	case resp.StatusCode == 409 || resp.StatusCode == 422:
		return Reservation{}, fmt.Errorf("%w: %s", ErrCreditRefused, raw)
	case resp.StatusCode >= 500:
		return Reservation{}, fmt.Errorf("%w: status %d", ErrOutcomeUnknown, resp.StatusCode)
	case resp.StatusCode >= 300:
		return Reservation{}, fmt.Errorf("credit provider: status %d: %s", resp.StatusCode, raw)
	}
	var res Reservation
	if err := json.Unmarshal(raw, &res); err != nil {
		return Reservation{}, fmt.Errorf("%w: %v", ErrOutcomeUnknown, err)
	}
	return res, nil
}

// Reserve consumes credit under the idempotency key.
func (c *CreditClient) Reserve(ctx context.Context, key, customer string, amount int64, quoteRef string) (Reservation, error) {
	return c.do(ctx, http.MethodPost, "/reservations", key, reserveBody{customer, amount, quoteRef})
}

// Inquire reads the provider's record for the key.
func (c *CreditClient) Inquire(ctx context.Context, key string) (Reservation, error) {
	return c.do(ctx, http.MethodGet, "/reservations/"+key, "", nil)
}

// Release returns the consumed credit; releasing twice is answered with the
// first release.
func (c *CreditClient) Release(ctx context.Context, key string) (Reservation, error) {
	return c.do(ctx, http.MethodPost, "/reservations/"+key+"/release", "", nil)
}
