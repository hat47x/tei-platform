package ordermigrationdualdb

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

// Billing (P-TX-02). Every change of a billed customer field is written to
// omdd.billing_outbox in the same primary transaction (trigger). The sender
// posts each entry with its idempotency key. A timeout or a lost response
// makes the result unknown: the entry is marked "unknown" and later entries
// wait. ReconcileBilling asks the billing system whether the key was
// applied; only "not received" makes the entry pending again. Resending the
// same key is safe because the billing system applies a key once.

// BillingOptions bounds every call to the billing system.
type BillingOptions struct {
	URL     string
	Timeout time.Duration
}

// Billing outcomes recorded in billing_outbox.last_outcome.
const (
	BillingDelivered      = "delivered"
	BillingReplayed       = "delivered-replayed"
	BillingUnknown        = "unknown"
	BillingConfirmed      = "confirmed-by-query"
	BillingNotReceived    = "not-received-by-query"
	BillingRejected       = "rejected"
	BillingServerError    = "billing-error"
	billingProfilesPath   = "/billing-profiles"
	billingRequestsPrefix = "/billing-requests/"
)

// BillingResult reports one send or reconcile pass.
type BillingResult struct {
	Outstanding      int    `json:"outstanding"`
	Delivered        int    `json:"delivered"`
	Replayed         int    `json:"replayed"`
	Unknown          int    `json:"unknown"`
	Confirmed        int    `json:"confirmedByQuery"`
	NotReceived      int    `json:"notReceivedByQuery"`
	BlockedByUnknown bool   `json:"blockedByUnknown"`
	Error            string `json:"error,omitempty"`
}

type billingEntry struct {
	seq     int64
	key     string
	state   string
	payload []byte
}

func billingEntries(ctx context.Context, primary *pgx.Conn, states ...string) ([]billingEntry, error) {
	rows, err := primary.Query(ctx, `SELECT seq, idempotency_key, state, payload FROM omdd.billing_outbox WHERE state = ANY($1) ORDER BY seq`, states)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []billingEntry
	for rows.Next() {
		var e billingEntry
		if err := rows.Scan(&e.seq, &e.key, &e.state, &e.payload); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

type billingResponse struct {
	ApplyID  string `json:"applyId"`
	Replayed bool   `json:"replayed"`
}

func markBilling(ctx context.Context, primary *pgx.Conn, seq int64, state, outcome, errText, applyID, confirmedBy string) error {
	_, err := primary.Exec(ctx, `UPDATE omdd.billing_outbox SET state=$2, last_outcome=$3, last_error=NULLIF($4,''), apply_id=coalesce(NULLIF($5,''), apply_id),
		confirmed_by=coalesce(NULLIF($6,''), confirmed_by), attempts=attempts + CASE WHEN $3 IN ($7,$8) THEN 0 ELSE 1 END, updated_at=now() WHERE seq=$1`,
		seq, state, outcome, truncate(errText, 300), applyID, confirmedBy, BillingConfirmed, BillingNotReceived)
	return err
}

// SendBilling delivers pending billing entries in order. It stops at the
// first entry whose result is unknown, or at an unknown entry left by an
// earlier pass, so a later change of the same customer never overtakes it.
func SendBilling(ctx context.Context, primary *pgx.Conn, opt BillingOptions) (BillingResult, error) {
	var res BillingResult
	entries, err := billingEntries(ctx, primary, "pending", "unknown")
	if err != nil {
		return res, err
	}
	res.Outstanding = len(entries)
	client := &http.Client{Timeout: opt.Timeout}
	for _, e := range entries {
		if e.state == "unknown" {
			res.BlockedByUnknown = true
			return res, nil
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(opt.URL, "/")+billingProfilesPath, bytes.NewReader(e.payload))
		if err != nil {
			return res, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", e.key)
		resp, err := client.Do(req)
		if err != nil {
			// The request may have been applied. The idempotency key makes a
			// resend safe, but the entry is not counted as delivered until
			// the billing system confirms it.
			res.Unknown++
			res.Error = err.Error()
			return res, markBilling(ctx, primary, e.seq, "unknown", BillingUnknown, err.Error(), "", "")
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
		resp.Body.Close()
		switch {
		case resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated:
			var br billingResponse
			if err := json.Unmarshal(body, &br); err != nil {
				return res, fmt.Errorf("billing response for %s: %w", e.key, err)
			}
			outcome := BillingDelivered
			if br.Replayed {
				outcome = BillingReplayed
				res.Replayed++
			}
			res.Delivered++
			if err := markBilling(ctx, primary, e.seq, "delivered", outcome, "", br.ApplyID, "response"); err != nil {
				return res, err
			}
		case resp.StatusCode >= 500:
			res.Error = fmt.Sprintf("billing %d: %s", resp.StatusCode, body)
			return res, markBilling(ctx, primary, e.seq, "unknown", BillingUnknown, res.Error, "", "")
		default:
			res.Error = fmt.Sprintf("billing %d: %s", resp.StatusCode, body)
			return res, markBilling(ctx, primary, e.seq, "pending", BillingRejected, res.Error, "", "")
		}
	}
	return res, nil
}

// ReconcileBilling settles entries whose result is unknown by asking the
// billing system for the idempotency key.
func ReconcileBilling(ctx context.Context, primary *pgx.Conn, opt BillingOptions) (BillingResult, error) {
	var res BillingResult
	entries, err := billingEntries(ctx, primary, "unknown")
	if err != nil {
		return res, err
	}
	res.Outstanding = len(entries)
	client := &http.Client{Timeout: opt.Timeout}
	for _, e := range entries {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(opt.URL, "/")+billingRequestsPrefix+url.PathEscape(e.key), nil)
		if err != nil {
			return res, err
		}
		resp, err := client.Do(req)
		if err != nil {
			res.Unknown++
			res.Error = err.Error()
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
		resp.Body.Close()
		switch resp.StatusCode {
		case http.StatusOK:
			var br billingResponse
			if err := json.Unmarshal(body, &br); err != nil {
				return res, err
			}
			res.Confirmed++
			if err := markBilling(ctx, primary, e.seq, "delivered", BillingConfirmed, "", br.ApplyID, "query"); err != nil {
				return res, err
			}
		case http.StatusNotFound:
			res.NotReceived++
			if err := markBilling(ctx, primary, e.seq, "pending", BillingNotReceived, "", "", ""); err != nil {
				return res, err
			}
		default:
			res.Unknown++
			res.Error = fmt.Sprintf("billing query %d: %s", resp.StatusCode, body)
		}
	}
	return res, nil
}

// BillingMock is a stand-in for the external billing system. It applies a
// request at most once per idempotency key and can be told to hang after or
// before applying, so the caller's timeout fires either way.
type BillingMock struct {
	mu          sync.Mutex
	applied     map[string]billingRecord
	profiles    map[string]json.RawMessage
	requests    int
	applies     int
	replays     int
	hangAfter   int
	hangBefore  int
	hangFor     time.Duration
	applyPerKey map[string]int
}

type billingRecord struct {
	ApplyID string
	Body    string
}

// NewBillingMock returns an empty billing system.
func NewBillingMock() *BillingMock {
	return &BillingMock{applied: map[string]billingRecord{}, profiles: map[string]json.RawMessage{}, applyPerKey: map[string]int{}}
}

// HangAfterApply makes the next n posts apply and then not answer for d.
func (m *BillingMock) HangAfterApply(n int, d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hangAfter, m.hangFor = n, d
}

// HangBeforeApply makes the next n posts not answer for d and not apply.
func (m *BillingMock) HangBeforeApply(n int, d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hangBefore, m.hangFor = n, d
}

// BillingStats is what the billing system observed.
type BillingStats struct {
	Requests     int            `json:"requests"`
	Applies      int            `json:"applies"`
	Replays      int            `json:"replays"`
	DistinctKeys int            `json:"distinctKeys"`
	MaxPerKey    int            `json:"maxAppliesPerKey"`
	Profiles     map[string]any `json:"profiles"`
}

// Stats returns a copy of the counters.
func (m *BillingMock) Stats() BillingStats {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := BillingStats{Requests: m.requests, Applies: m.applies, Replays: m.replays, DistinctKeys: len(m.applied), Profiles: map[string]any{}}
	for _, n := range m.applyPerKey {
		if n > s.MaxPerKey {
			s.MaxPerKey = n
		}
	}
	for k, v := range m.profiles {
		var x any
		_ = json.Unmarshal(v, &x)
		s.Profiles[k] = x
	}
	return s
}

func (m *BillingMock) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	hang := func(d time.Duration) {
		select {
		case <-time.After(d):
		case <-r.Context().Done():
		}
	}
	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, billingRequestsPrefix):
		key, _ := url.PathUnescape(strings.TrimPrefix(r.URL.Path, billingRequestsPrefix))
		m.mu.Lock()
		rec, ok := m.applied[key]
		m.mu.Unlock()
		if !ok {
			http.Error(w, "not received", http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, billingResponse{ApplyID: rec.ApplyID})
	case r.Method == http.MethodPost && r.URL.Path == billingProfilesPath:
		key := r.Header.Get("Idempotency-Key")
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
		if err != nil || key == "" {
			http.Error(w, "idempotency key and body required", http.StatusBadRequest)
			return
		}
		var payload struct {
			CustomerCode string `json:"customerCode"`
		}
		if err := json.Unmarshal(body, &payload); err != nil || payload.CustomerCode == "" {
			http.Error(w, "customerCode required", http.StatusBadRequest)
			return
		}
		m.mu.Lock()
		m.requests++
		if m.hangBefore > 0 {
			m.hangBefore--
			d := m.hangFor
			m.mu.Unlock()
			hang(d)
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
			return
		}
		if rec, ok := m.applied[key]; ok {
			m.replays++
			m.mu.Unlock()
			if rec.Body != canonicalJSON(body) {
				http.Error(w, "idempotency key reused with a different body", http.StatusUnprocessableEntity)
				return
			}
			writeJSON(w, http.StatusOK, billingResponse{ApplyID: rec.ApplyID, Replayed: true})
			return
		}
		m.applies++
		m.applyPerKey[key]++
		rec := billingRecord{ApplyID: fmt.Sprintf("bill-%06d", m.applies), Body: canonicalJSON(body)}
		m.applied[key] = rec
		m.profiles[payload.CustomerCode] = json.RawMessage(body)
		hangAfter := m.hangAfter > 0
		if hangAfter {
			m.hangAfter--
		}
		d := m.hangFor
		m.mu.Unlock()
		if hangAfter {
			hang(d)
		}
		writeJSON(w, http.StatusCreated, billingResponse{ApplyID: rec.ApplyID})
	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

func canonicalJSON(b []byte) string {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return string(b)
	}
	out, _ := json.Marshal(v)
	return string(out)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// errBillingURL is returned by the CLI without -billing-url.
var errBillingURL = errors.New("-billing-url is required")
