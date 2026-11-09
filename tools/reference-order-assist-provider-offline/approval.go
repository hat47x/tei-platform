package orderassistoffline

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
)

// Stage is one step of a discount approval policy.
type Stage struct {
	ID           string `json:"id"`
	Role         string `json:"role"`
	CreditBefore bool   `json:"creditBefore,omitempty"` // the external credit must be consumed before this stage decides
	SendbackTo   string `json:"sendbackTo,omitempty"`
}

// Policy is a versioned approval policy. A case is pinned to the version
// (and its digest) it was opened with (P-WF-02).
type Policy struct {
	Schema  string  `json:"schema"`
	Version string  `json:"version"`
	Stages  []Stage `json:"stages"`
	// SendbackInvalidates declares which approvals a sendback withdraws:
	// "target-and-later" (approvals at the sendback target and every later
	// stage) or "all".
	SendbackInvalidates string `json:"sendbackInvalidates"`
}

// Migration maps the stages of one policy version onto another.
type Migration struct {
	Schema   string            `json:"schema"`
	From     string            `json:"from"`
	To       string            `json:"to"`
	StageMap map[string]string `json:"stageMap"`    // pending stage -> landing stage
	Carry    map[string]string `json:"approvalMap"` // approved stage -> stage the approval counts for
}

const (
	policySchema    = "tei.reference.order-assist.approval-policy/candidate-v0"
	migrationSchema = "tei.reference.order-assist.policy-migration/candidate-v0"
)

func (p Policy) index(stage string) int {
	for i, s := range p.Stages {
		if s.ID == stage {
			return i
		}
	}
	return -1
}

// Validate checks the policy is internally consistent.
func (p Policy) Validate() error {
	if p.Schema != policySchema || p.Version == "" || len(p.Stages) == 0 {
		return fmt.Errorf("policy %q: schema, version or stages missing", p.Version)
	}
	if p.SendbackInvalidates != "target-and-later" && p.SendbackInvalidates != "all" {
		return fmt.Errorf("policy %q: undeclared sendback scope %q", p.Version, p.SendbackInvalidates)
	}
	for i, s := range p.Stages {
		if s.ID == "" || s.Role == "" || p.index(s.ID) != i {
			return fmt.Errorf("policy %q: stage %d id/role missing or duplicate", p.Version, i)
		}
		if s.SendbackTo != "" {
			if t := p.index(s.SendbackTo); t < 0 || t >= i {
				return fmt.Errorf("policy %q: stage %s sends back to a later or unknown stage", p.Version, s.ID)
			}
		}
	}
	return nil
}

// Validate checks the migration against both policies.
func (m Migration) Validate(from, to Policy) error {
	if m.Schema != migrationSchema || m.From != from.Version || m.To != to.Version {
		return fmt.Errorf("migration %s->%s does not match policies %s->%s", m.From, m.To, from.Version, to.Version)
	}
	for k, v := range m.StageMap {
		if from.index(k) < 0 || to.index(v) < 0 {
			return fmt.Errorf("migration stage map %s->%s names an unknown stage", k, v)
		}
	}
	for k, v := range m.Carry {
		if from.index(k) < 0 || to.index(v) < 0 {
			return fmt.Errorf("migration approval map %s->%s names an unknown stage", k, v)
		}
	}
	return nil
}

func policyDigest(p Policy) string {
	raw, _ := json.Marshal(p)
	c, _ := canonicalJSON(raw)
	return digestBytes(c)
}

// LoadPolicy reads and validates a policy file.
func LoadPolicy(path string) (Policy, error) {
	var p Policy
	raw, err := os.ReadFile(path)
	if err == nil {
		err = decodeStrict(raw, &p)
	}
	if err == nil {
		err = p.Validate()
	}
	return p, err
}

// LoadMigration reads a migration file.
func LoadMigration(path string) (Migration, error) {
	var m Migration
	raw, err := os.ReadFile(path)
	if err == nil {
		err = decodeStrict(raw, &m)
	}
	return m, err
}

func decodeStrict(raw []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

// Approval errors. Each refused attempt is also recorded as an event.
var (
	ErrStaleQuote          = errors.New("approval targets a quote digest that is not current")
	ErrQuoteNotValid       = errors.New("quote is not valid for approval")
	ErrWrongStage          = errors.New("case is not at this stage")
	ErrCaseClosed          = errors.New("case is closed")
	ErrNotAuthorized       = errors.New("actor holds neither the role nor a delegation for it")
	ErrDelegationOutOfTerm = errors.New("delegation does not cover the business date")
	ErrCreditNotReady      = errors.New("external credit is not consumed for this quote digest")
	ErrNoSendback          = errors.New("stage declares no sendback target")
	ErrUnmappedStage       = errors.New("case is at a stage the migration does not map")
)

// InstallPolicy stores a policy version; active makes it the one new cases use.
func (s *Server) InstallPolicy(ctx context.Context, p Policy, active bool) error {
	if err := p.Validate(); err != nil {
		return err
	}
	raw, _ := json.Marshal(p)
	return pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		if active {
			if _, err := tx.Exec(ctx, `UPDATE oapo.policy SET active_for_new_cases=false`); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(ctx, `INSERT INTO oapo.policy VALUES ($1,$2,$3,$4)`, p.Version, policyDigest(p), raw, active); err != nil {
			return err
		}
		_, err := recordEvent(ctx, tx, "policy-installed", "", "", map[string]any{"version": p.Version, "digest": policyDigest(p), "active": active})
		return err
	})
}

// AddPrincipal registers a person with a role.
func (s *Server) AddPrincipal(ctx context.Context, id, role string) error {
	_, err := s.Pool.Exec(ctx, `INSERT INTO oapo.principal VALUES ($1,$2)`, id, role)
	return err
}

// AddDelegation lets delegate act for delegator's role between two business dates.
func (s *Server) AddDelegation(ctx context.Context, delegator, delegate, role, from, to string) error {
	_, err := s.Pool.Exec(ctx, `INSERT INTO oapo.delegation(delegator, delegate, role, valid_from, valid_to) VALUES ($1,$2,$3,$4,$5)`,
		delegator, delegate, role, from, to)
	return err
}

// SetBusinessDate moves the business date. Delegation periods are judged on
// this date, not on any machine clock.
func (s *Server) SetBusinessDate(ctx context.Context, date string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE oapo.clock SET business_date=$1 WHERE id=1`, date)
	return err
}

func loadPolicyTx(ctx context.Context, tx pgx.Tx, version string) (Policy, string, error) {
	var raw []byte
	var digest string
	if err := tx.QueryRow(ctx, `SELECT definition, digest FROM oapo.policy WHERE version=$1`, version).Scan(&raw, &digest); err != nil {
		return Policy{}, "", err
	}
	var p Policy
	err := json.Unmarshal(raw, &p)
	return p, digest, err
}

// OpenCase opens an approval case for a valid quote under the active policy.
func (s *Server) OpenCase(ctx context.Context, caseID, quoteID string) error {
	return pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		var status, digest string
		if err := tx.QueryRow(ctx, `SELECT status, digest FROM oapo.quote WHERE quote_id=$1 FOR SHARE`, quoteID).Scan(&status, &digest); err != nil {
			return err
		}
		if status != "valid" {
			return fmt.Errorf("%w: %s", ErrQuoteNotValid, status)
		}
		var version string
		if err := tx.QueryRow(ctx, `SELECT version FROM oapo.policy WHERE active_for_new_cases`).Scan(&version); err != nil {
			return err
		}
		p, pd, err := loadPolicyTx(ctx, tx, version)
		if err != nil {
			return err
		}
		seq, err := recordEvent(ctx, tx, "case-opened", quoteID, caseID, map[string]any{"policy": version, "policyDigest": pd, "quoteDigest": digest})
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO oapo.approval_case(case_id, quote_id, quote_digest, policy_version, policy_digest, stage, status, opened_seq)
			VALUES ($1,$2,$3,$4,$5,$6,'pending',$7)`, caseID, quoteID, digest, version, pd, p.Stages[0].ID, seq)
		return err
	})
}

// Decision is one approver action. IfMatch is the quote digest the approver
// saw; it must still be the current digest (RFC 9110 conditional request).
type Decision struct {
	CaseID   string
	Stage    string
	Actor    string
	Decision string // approve | sendback | reject
	IfMatch  string
}

// ActionRecord is a stored approver action.
type ActionRecord struct {
	ActionID          int64  `json:"actionId"`
	PolicyVersion     string `json:"policyVersion"`
	Stage             string `json:"stage"`
	Decision          string `json:"decision"`
	Actor             string `json:"actor"`
	OnBehalfOf        string `json:"onBehalfOf,omitempty"`
	QuoteDigest       string `json:"quoteDigest"`
	BusinessDate      string `json:"businessDate"`
	Valid             bool   `json:"valid"`
	InvalidatedReason string `json:"invalidatedReason,omitempty"`
	MigratedFrom      string `json:"migratedFrom,omitempty"`
}

// CaseState is the current state of a case.
type CaseState struct {
	CaseID        string         `json:"caseId"`
	QuoteID       string         `json:"quoteId"`
	QuoteDigest   string         `json:"quoteDigest"`
	PolicyVersion string         `json:"policyVersion"`
	PolicyDigest  string         `json:"policyDigest"`
	Stage         string         `json:"stage"`
	Status        string         `json:"status"`
	Actions       []ActionRecord `json:"actions"`
}

// Case reads a case with its actions.
func (s *Server) Case(ctx context.Context, caseID string) (CaseState, error) {
	var c CaseState
	if err := s.Pool.QueryRow(ctx, `SELECT case_id, quote_id, quote_digest, policy_version, policy_digest, coalesce(stage,''), status
		FROM oapo.approval_case WHERE case_id=$1`, caseID).Scan(&c.CaseID, &c.QuoteID, &c.QuoteDigest, &c.PolicyVersion, &c.PolicyDigest, &c.Stage, &c.Status); err != nil {
		return c, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT action_id, policy_version, stage, decision, actor, coalesce(on_behalf_of,''), quote_digest,
		business_date::text, valid, coalesce(invalidated_reason,''), coalesce(migrated_from,'') FROM oapo.approval_action WHERE case_id=$1 ORDER BY action_id`, caseID)
	if err != nil {
		return c, err
	}
	c.Actions, err = pgx.CollectRows(rows, func(r pgx.CollectableRow) (ActionRecord, error) {
		var a ActionRecord
		err := r.Scan(&a.ActionID, &a.PolicyVersion, &a.Stage, &a.Decision, &a.Actor, &a.OnBehalfOf, &a.QuoteDigest, &a.BusinessDate,
			&a.Valid, &a.InvalidatedReason, &a.MigratedFrom)
		return a, err
	})
	return c, err
}

// Decide records an approver action. A refused attempt is committed as an
// event and the error is returned.
func (s *Server) Decide(ctx context.Context, d Decision) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	refuse := func(cause error, detail map[string]any) error {
		detail["refused"] = cause.Error()
		detail["decision"] = d.Decision
		detail["stage"] = d.Stage
		detail["actor"] = d.Actor
		detail["ifMatch"] = d.IfMatch
		if _, err := recordEvent(ctx, tx, "decision-refused", "", d.CaseID, detail); err != nil {
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		return cause
	}
	var c CaseState
	if err := tx.QueryRow(ctx, `SELECT quote_id, quote_digest, policy_version, coalesce(stage,''), status FROM oapo.approval_case WHERE case_id=$1 FOR UPDATE`,
		d.CaseID).Scan(&c.QuoteID, &c.QuoteDigest, &c.PolicyVersion, &c.Stage, &c.Status); err != nil {
		return err
	}
	if c.Status != "pending" {
		return refuse(ErrCaseClosed, map[string]any{"status": c.Status})
	}
	var qStatus, qDigest, bizDate string
	if err := tx.QueryRow(ctx, `SELECT status, digest FROM oapo.quote WHERE quote_id=$1 FOR SHARE`, c.QuoteID).Scan(&qStatus, &qDigest); err != nil {
		return err
	}
	if err := tx.QueryRow(ctx, `SELECT business_date::text FROM oapo.clock WHERE id=1`).Scan(&bizDate); err != nil {
		return err
	}
	if d.IfMatch != qDigest || c.QuoteDigest != qDigest {
		return refuse(ErrStaleQuote, map[string]any{"currentDigest": qDigest, "caseDigest": c.QuoteDigest})
	}
	if qStatus != "valid" {
		return refuse(ErrQuoteNotValid, map[string]any{"quoteStatus": qStatus})
	}
	p, _, err := loadPolicyTx(ctx, tx, c.PolicyVersion)
	if err != nil {
		return err
	}
	idx := p.index(d.Stage)
	if d.Stage != c.Stage || idx < 0 {
		return refuse(ErrWrongStage, map[string]any{"caseStage": c.Stage})
	}
	stage := p.Stages[idx]
	var role string
	if err := tx.QueryRow(ctx, `SELECT role FROM oapo.principal WHERE principal_id=$1`, d.Actor).Scan(&role); err != nil {
		return refuse(ErrNotAuthorized, map[string]any{})
	}
	var onBehalf *string
	var delegationID *int64
	if role != stage.Role {
		var inTerm, anyDelegation bool
		var who string
		var id int64
		err := tx.QueryRow(ctx, `SELECT delegation_id, delegator, $3::date BETWEEN valid_from AND valid_to FROM oapo.delegation
			WHERE delegate=$1 AND role=$2 ORDER BY ($3::date BETWEEN valid_from AND valid_to) DESC, delegation_id LIMIT 1`,
			d.Actor, stage.Role, bizDate).Scan(&id, &who, &inTerm)
		if err == nil {
			anyDelegation = true
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if !anyDelegation {
			return refuse(ErrNotAuthorized, map[string]any{"role": role, "stageRole": stage.Role})
		}
		if !inTerm {
			return refuse(ErrDelegationOutOfTerm, map[string]any{"businessDate": bizDate, "delegationId": id, "delegator": who})
		}
		onBehalf, delegationID = &who, &id
	}
	newStage, newStatus := c.Stage, "pending"
	switch d.Decision {
	case "approve":
		if stage.CreditBefore {
			var state, creditDigest string
			err := tx.QueryRow(ctx, `SELECT state, quote_digest FROM oapo.credit_step WHERE case_id=$1`, d.CaseID).Scan(&state, &creditDigest)
			if errors.Is(err, pgx.ErrNoRows) || (err == nil && (state != "consumed" || creditDigest != c.QuoteDigest)) {
				return refuse(ErrCreditNotReady, map[string]any{"creditState": state})
			} else if err != nil {
				return err
			}
		}
		if idx+1 < len(p.Stages) {
			newStage = p.Stages[idx+1].ID
		} else {
			newStatus = "approved"
		}
	case "sendback":
		if stage.SendbackTo == "" {
			return refuse(ErrNoSendback, map[string]any{})
		}
		target := p.index(stage.SendbackTo)
		var stages []string
		for i, st := range p.Stages {
			if p.SendbackInvalidates == "all" || i >= target {
				stages = append(stages, st.ID)
			}
		}
		if _, err := tx.Exec(ctx, `UPDATE oapo.approval_action SET valid=false, invalidated_reason='sendback-from-'||$2
			WHERE case_id=$1 AND valid AND decision='approve' AND stage = ANY($3)`, d.CaseID, d.Stage, stages); err != nil {
			return err
		}
		newStage = stage.SendbackTo
	case "reject":
		newStatus = "rejected"
	default:
		return fmt.Errorf("unknown decision %q", d.Decision)
	}
	seq, err := recordEvent(ctx, tx, "decision", c.QuoteID, d.CaseID, map[string]any{"decision": d.Decision, "stage": d.Stage,
		"actor": d.Actor, "onBehalfOf": onBehalf, "policy": c.PolicyVersion, "quoteDigest": c.QuoteDigest, "businessDate": bizDate})
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO oapo.approval_action(case_id, policy_version, stage, decision, actor, on_behalf_of, delegation_id,
		quote_digest, business_date, seq, valid) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,true)`,
		d.CaseID, c.PolicyVersion, d.Stage, d.Decision, d.Actor, onBehalf, delegationID, c.QuoteDigest, bizDate, seq); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE oapo.approval_case SET stage=$2, status=$3 WHERE case_id=$1`, d.CaseID, newStage, newStatus); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// invalidateApprovalsForQuote runs in the transaction that changed a quote's
// digest: every pending case on the quote loses its approvals and returns to
// its policy's first stage, now pinned to the new digest.
func invalidateApprovalsForQuote(ctx context.Context, tx pgx.Tx, quoteID, newDigest string) error {
	rows, err := tx.Query(ctx, `SELECT c.case_id, c.quote_digest, p.definition->'stages'->0->>'id' FROM oapo.approval_case c
		JOIN oapo.policy p ON p.version=c.policy_version WHERE c.quote_id=$1 AND c.status='pending' FOR UPDATE OF c`, quoteID)
	if err != nil {
		return err
	}
	type pending struct{ id, digest, first string }
	cases, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (pending, error) {
		var p pending
		err := r.Scan(&p.id, &p.digest, &p.first)
		return p, err
	})
	if err != nil {
		return err
	}
	for _, c := range cases {
		tag, err := tx.Exec(ctx, `UPDATE oapo.approval_action SET valid=false, invalidated_reason='quote-changed' WHERE case_id=$1 AND valid AND decision='approve'`, c.id)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE oapo.approval_case SET stage=$2, quote_digest=$3 WHERE case_id=$1`, c.id, c.first, newDigest); err != nil {
			return err
		}
		if _, err := recordEvent(ctx, tx, "approvals-invalidated", quoteID, c.id, map[string]any{"from": c.digest, "to": newDigest,
			"invalidated": tag.RowsAffected(), "restartAt": c.first}); err != nil {
			return err
		}
	}
	return nil
}

// MigrateCase moves a pending case to the migration's target policy. The
// case lands on the mapped stage; approvals listed in the approval map are
// carried for earlier stages, others are withdrawn. A case at an unmapped
// stage is refused and stays on its policy.
func (s *Server) MigrateCase(ctx context.Context, caseID string, m Migration) (string, error) {
	var landed string
	err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		var stage, version, status, quoteID string
		if err := tx.QueryRow(ctx, `SELECT coalesce(stage,''), policy_version, status, quote_id FROM oapo.approval_case WHERE case_id=$1 FOR UPDATE`,
			caseID).Scan(&stage, &version, &status, &quoteID); err != nil {
			return err
		}
		from, _, err := loadPolicyTx(ctx, tx, m.From)
		if err != nil {
			return err
		}
		to, toDigest, err := loadPolicyTx(ctx, tx, m.To)
		if err != nil {
			return err
		}
		if err := m.Validate(from, to); err != nil {
			return err
		}
		if status != "pending" || version != m.From {
			return fmt.Errorf("%w: case %s is %s on %s", ErrCaseClosed, caseID, status, version)
		}
		target, ok := m.StageMap[stage]
		if !ok {
			return fmt.Errorf("%w: %s", ErrUnmappedStage, stage)
		}
		landed = target
		landIdx := to.index(target)
		rows, err := tx.Query(ctx, `SELECT action_id, stage FROM oapo.approval_action WHERE case_id=$1 AND valid AND decision='approve'`, caseID)
		if err != nil {
			return err
		}
		type act struct {
			id    int64
			stage string
		}
		acts, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (act, error) {
			var a act
			return a, r.Scan(&a.id, &a.stage)
		})
		if err != nil {
			return err
		}
		carried := map[string]string{}
		for _, a := range acts {
			if dst, ok := m.Carry[a.stage]; ok && to.index(dst) < landIdx {
				if _, err := tx.Exec(ctx, `UPDATE oapo.approval_action SET stage=$2, policy_version=$3, migrated_from=$4 WHERE action_id=$1`,
					a.id, dst, to.Version, version+"#"+a.stage); err != nil {
					return err
				}
				carried[a.stage] = dst
			} else if _, err := tx.Exec(ctx, `UPDATE oapo.approval_action SET valid=false, invalidated_reason='not-carried-by-migration' WHERE action_id=$1`, a.id); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(ctx, `UPDATE oapo.approval_case SET policy_version=$2, policy_digest=$3, stage=$4,
			history = history || jsonb_build_array(jsonb_build_object('migratedFrom',$5::text,'fromStage',$6::text,'toStage',$4::text))
			WHERE case_id=$1`, caseID, to.Version, toDigest, target, version, stage); err != nil {
			return err
		}
		_, err = recordEvent(ctx, tx, "case-migrated", quoteID, caseID, map[string]any{"from": version, "to": to.Version,
			"fromStage": stage, "toStage": target, "carried": carried})
		return err
	})
	if errors.Is(err, ErrUnmappedStage) {
		_ = pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
			_, e := recordEvent(ctx, tx, "case-migration-refused", "", caseID, map[string]any{"reason": err.Error(), "migration": m.From + "->" + m.To})
			return e
		})
	}
	return landed, err
}
