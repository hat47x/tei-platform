package orderassistoffline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Saga drives the credit step of a case and the handoff to confirmation
// (P-TX-02, P-TX-03). Every state is written to the order DB before the
// provider is called, so a stopped worker resumes from the DB. The provider
// call itself and the recording of its answer are not atomic; that gap is
// closed by the idempotency key and by inquiry, not by a transaction.
type Saga struct {
	Pool   *pgxpool.Pool
	Credit *CreditClient
	Hooks  SagaHooks
}

// SagaHooks are fault-injection points. A hook may terminate the process.
type SagaHooks struct {
	AfterReserveResponse func() // the provider answered; nothing recorded yet
	BeforeReleaseSend    func() // release-pending is recorded; nothing sent yet
	AfterReleaseResponse func() // the provider answered the release; nothing recorded yet
}

// MaxAutomaticReserveAttempts is how many times the saga sends a reservation
// (the first send and automatic resends, all with the same idempotency key)
// before it resolves an unknown outcome by inquiry.
const MaxAutomaticReserveAttempts = 2

// SagaStep is one step the saga took.
type SagaStep struct {
	Action string `json:"action"`
	State  string `json:"state"`
	Detail string `json:"detail,omitempty"`
}

// CreditKey is the idempotency key for a case's credit consumption. It binds
// the key to the quote digest the case was approved against.
func CreditKey(caseID, quoteDigest string) string {
	return "oapo-credit." + caseID + "." + strings.TrimPrefix(quoteDigest, "sha256:")[:32]
}

type sagaView struct {
	status, stage, policy, quoteID, digest string
	creditBefore                           bool
	credit                                 *creditRow
	handoff                                bool
	customer                               string
	net                                    int64
}

type creditRow struct {
	key, state, digest string
	reservation        *int64
	amount             int64
	attempts           int
}

func (s *Saga) load(ctx context.Context, caseID string) (sagaView, error) {
	var v sagaView
	err := s.Pool.QueryRow(ctx, `SELECT c.status, coalesce(c.stage,''), c.policy_version, c.quote_id, c.quote_digest,
		coalesce((SELECT bool_or((st->>'creditBefore')::boolean) FROM jsonb_array_elements(p.definition->'stages') st WHERE st->>'id'=c.stage), false),
		q.doc->>'customerCode', coalesce((q.revalidation->'judgment'->'result'->>'netMinor')::bigint, 0),
		EXISTS (SELECT 1 FROM oapo.handoff h WHERE h.case_id=c.case_id)
		FROM oapo.approval_case c JOIN oapo.policy p ON p.version=c.policy_version JOIN oapo.quote q ON q.quote_id=c.quote_id
		WHERE c.case_id=$1`, caseID).Scan(&v.status, &v.stage, &v.policy, &v.quoteID, &v.digest, &v.creditBefore, &v.customer, &v.net, &v.handoff)
	if err != nil {
		return v, err
	}
	var cr creditRow
	err = s.Pool.QueryRow(ctx, `SELECT idempotency_key, state, quote_digest, reservation_id, amount_minor, attempts FROM oapo.credit_step WHERE case_id=$1`, caseID).
		Scan(&cr.key, &cr.state, &cr.digest, &cr.reservation, &cr.amount, &cr.attempts)
	if err == nil {
		v.credit = &cr
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return v, err
	}
	return v, nil
}

func (s *Saga) setCredit(ctx context.Context, caseID, state, note string, reservation *int64, attempt, releaseAttempt int) error {
	return pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE oapo.credit_step SET state=$2, reservation_id=coalesce($3, reservation_id),
			attempts=attempts+$4, release_attempts=release_attempts+$5,
			history = history || jsonb_build_array(jsonb_build_object('state',$2::text,'note',$6::text)) WHERE case_id=$1`,
			caseID, state, reservation, attempt, releaseAttempt, note)
		if err != nil {
			return err
		}
		_, err = recordEvent(ctx, tx, "credit-"+state, "", caseID, map[string]any{"note": note, "reservationId": reservation})
		return err
	})
}

// Run advances the case's credit step and handoff until nothing more can be
// done without a person's decision. It is safe to run again at any point.
func (s *Saga) Run(ctx context.Context, caseID string) ([]SagaStep, error) {
	var steps []SagaStep
	for i := 0; i < 20; i++ {
		v, err := s.load(ctx, caseID)
		if err != nil {
			return steps, err
		}
		step, progressed, err := s.advance(ctx, caseID, v)
		if step.Action != "" {
			steps = append(steps, step)
		}
		if err != nil || !progressed {
			return steps, err
		}
	}
	return steps, fmt.Errorf("saga for %s did not settle", caseID)
}

func (s *Saga) advance(ctx context.Context, caseID string, v sagaView) (SagaStep, bool, error) {
	c := v.credit
	switch {
	case v.status == "pending" && v.creditBefore && c == nil:
		if v.net <= 0 {
			return SagaStep{}, false, fmt.Errorf("case %s: quote has no positive net amount", caseID)
		}
		key := CreditKey(caseID, v.digest)
		err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, `INSERT INTO oapo.credit_step(case_id, quote_digest, idempotency_key, customer_code, amount_minor, state, history)
				VALUES ($1,$2,$3,$4,$5,'reserve-pending', jsonb_build_array(jsonb_build_object('state','reserve-pending')))`,
				caseID, v.digest, key, v.customer, v.net); err != nil {
				return err
			}
			_, err := recordEvent(ctx, tx, "credit-reserve-pending", v.quoteID, caseID, map[string]any{"key": key, "amount": v.net, "quoteDigest": v.digest})
			return err
		})
		return SagaStep{"record-reserve-intent", "reserve-pending", key}, err == nil, err

	case c != nil && (c.state == "reserve-pending" || c.state == "unknown"):
		if c.state == "unknown" && c.attempts >= MaxAutomaticReserveAttempts {
			// The answers were lost: stop resending and ask the provider.
			res, err := s.Credit.Inquire(ctx, c.key)
			switch {
			case err == nil:
				return SagaStep{"inquire", "consumed", "resolved by inquiry"}, true,
					s.setCredit(ctx, caseID, "consumed", "resolved-by-inquiry", &res.ReservationID, 0, 0)
			case errors.Is(err, ErrNotFound):
				// Not applied: send again with the same key below.
			default:
				return SagaStep{"inquire", "unknown", err.Error()}, false, err
			}
		}
		res, err := s.Credit.Reserve(ctx, c.key, v.customer, c.amount, c.digest)
		if s.Hooks.AfterReserveResponse != nil {
			s.Hooks.AfterReserveResponse()
		}
		switch {
		case err == nil:
			note := "answered"
			if res.Replayed {
				note = "answered-replayed"
			}
			return SagaStep{"reserve", "consumed", note}, true, s.setCredit(ctx, caseID, "consumed", note, &res.ReservationID, 1, 0)
		case errors.Is(err, ErrOutcomeUnknown):
			return SagaStep{"reserve", "unknown", err.Error()}, true, s.setCredit(ctx, caseID, "unknown", "no-answer", nil, 1, 0)
		case errors.Is(err, ErrCreditRefused):
			return SagaStep{"reserve", "refused", err.Error()}, false, s.setCredit(ctx, caseID, "refused", err.Error(), nil, 1, 0)
		default:
			return SagaStep{"reserve", c.state, err.Error()}, false, err
		}

	case c != nil && v.status == "rejected" && c.state == "consumed":
		return SagaStep{"record-release-intent", "release-pending", ""}, true, s.setCredit(ctx, caseID, "release-pending", "case-rejected", nil, 0, 0)

	case c != nil && c.state == "release-pending":
		if s.Hooks.BeforeReleaseSend != nil {
			s.Hooks.BeforeReleaseSend()
		}
		res, err := s.Credit.Release(ctx, c.key)
		if s.Hooks.AfterReleaseResponse != nil {
			s.Hooks.AfterReleaseResponse()
		}
		switch {
		case err == nil:
			note := "answered"
			if res.Replayed {
				note = "answered-replayed"
			}
			return SagaStep{"release", "released", note}, true, s.setCredit(ctx, caseID, "released", note, nil, 0, 1)
		case errors.Is(err, ErrOutcomeUnknown):
			// Count the attempt, then ask the provider.
			if err := s.setCredit(ctx, caseID, "release-pending", "no-answer", nil, 0, 1); err != nil {
				return SagaStep{}, false, err
			}
			res, ierr := s.Credit.Inquire(ctx, c.key)
			if ierr == nil && res.Status == "released" {
				return SagaStep{"inquire", "released", "resolved by inquiry"}, true, s.setCredit(ctx, caseID, "released", "resolved-by-inquiry", nil, 0, 0)
			}
			return SagaStep{"release", "release-pending", err.Error()}, true, nil
		default:
			return SagaStep{"release", c.state, err.Error()}, false, err
		}

	case v.status == "approved" && !v.handoff && c != nil && c.state == "consumed" && c.digest == v.digest:
		return SagaStep{"handoff", "ready-for-confirmation", ""}, true, s.writeHandoff(ctx, caseID, v, "ready-for-confirmation")

	case v.status == "rejected" && !v.handoff && (c == nil || c.state == "released" || c.state == "refused"):
		return SagaStep{"handoff", "withdrawn", ""}, true, s.writeHandoff(ctx, caseID, v, "withdrawn")
	}
	return SagaStep{}, false, nil
}

func (s *Saga) writeHandoff(ctx context.Context, caseID string, v sagaView, outcome string) error {
	return pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT stage, actor, coalesce(on_behalf_of,''), quote_digest, policy_version FROM oapo.approval_action
			WHERE case_id=$1 AND valid ORDER BY action_id`, caseID)
		if err != nil {
			return err
		}
		approvals, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (map[string]string, error) {
			var stage, actor, behalf, digest, policy string
			err := r.Scan(&stage, &actor, &behalf, &digest, &policy)
			return map[string]string{"stage": stage, "actor": actor, "onBehalfOf": behalf, "quoteDigest": digest, "policy": policy}, err
		})
		if err != nil {
			return err
		}
		var reservation *int64
		state := "none"
		if v.credit != nil {
			reservation, state = v.credit.reservation, v.credit.state
		}
		detail, _ := json.Marshal(map[string]any{"approvals": approvals, "creditKey": func() string {
			if v.credit != nil {
				return v.credit.key
			}
			return ""
		}()})
		seq, err := recordEvent(ctx, tx, "handoff-"+outcome, v.quoteID, caseID, map[string]any{"quoteDigest": v.digest})
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO oapo.handoff VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			caseID, v.quoteID, v.digest, v.policy, outcome, reservation, state, detail, seq)
		return err
	})
}

// Convergence classifies a case's end state across the order side and the
// provider: all-complete, all-compensated, or intermediate.
func Convergence(ctx context.Context, order *pgxpool.Pool, credit *CreditClient, caseID string) (string, error) {
	var status, handoff, digest string
	var cstate, key *string
	err := order.QueryRow(ctx, `SELECT c.status, c.quote_digest, coalesce(h.outcome,''), cs.state, cs.idempotency_key
		FROM oapo.approval_case c LEFT JOIN oapo.handoff h ON h.case_id=c.case_id LEFT JOIN oapo.credit_step cs ON cs.case_id=c.case_id
		WHERE c.case_id=$1`, caseID).Scan(&status, &digest, &handoff, &cstate, &key)
	if err != nil {
		return "", err
	}
	provider := "none"
	if key != nil {
		res, err := credit.Inquire(ctx, *key)
		switch {
		case err == nil:
			provider = res.Status
		case errors.Is(err, ErrNotFound):
		default:
			return "", err
		}
	}
	local := "none"
	if cstate != nil {
		local = *cstate
	}
	switch {
	case status == "approved" && handoff == "ready-for-confirmation" && local == "consumed" && provider == "consumed":
		return "all-complete", nil
	case status == "rejected" && handoff == "withdrawn" && (local == "released" && provider == "released" || local == "none" && provider == "none"):
		return "all-compensated", nil
	}
	return fmt.Sprintf("intermediate(case=%s handoff=%s credit=%s provider=%s)", status, handoff, local, provider), nil
}
