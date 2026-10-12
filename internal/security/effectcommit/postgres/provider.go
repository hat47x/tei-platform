// Package postgres is a PostgreSQL-backed reference implementation of the
// authorized effect-commit boundary. Deployment must independently attest that
// the schema and authority mutation paths satisfy the transaction contract.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"tei/internal/security/effectcommit"
	"tei/internal/security/legacyauthorization"
)

var schemaName = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

// Provider requires one PostgreSQL database for authority, effects, and receipts.
// The schema is supplied by deployment, never taken from a user request.
type Provider struct {
	Pool   *pgxpool.Pool
	Schema string
}

func New(pool *pgxpool.Pool, schema string) (Provider, error) {
	if pool == nil || !schemaName.MatchString(schema) {
		return Provider{}, fmt.Errorf("PostgreSQLの接続または固定スキーマ名が不正です")
	}
	return Provider{Pool: pool, Schema: schema}, nil
}

func (p Provider) Begin(ctx context.Context) (effectcommit.Transaction, error) {
	if p.Pool == nil || !schemaName.MatchString(p.Schema) {
		return nil, effectcommit.ErrMissingProvider
	}
	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &transaction{tx: tx, schema: `"` + p.Schema + `"`}, nil
}

type transaction struct {
	tx     pgx.Tx
	schema string
}

func (t *transaction) table(name string) string { return t.schema + `."` + name + `"` }

// CheckAuthority acquires a row lock that is held until transaction completion.
// Every authority mutation must lock or UPDATE the same row, including indirect
// subject and role changes. A separate authority DB cannot satisfy this contract.
func (t *transaction) CheckAuthority(ctx context.Context, s effectcommit.Scope, e legacyauthorization.AuthorityEpochs) (bool, error) {
	if e.SubjectLink > math.MaxInt64 || e.Membership > math.MaxInt64 || e.Policy > math.MaxInt64 {
		return false, effectcommit.ErrInvalidRequest
	}
	q := "SELECT source, subject_link, membership, policy, enabled FROM " + t.table("authority") +
		" WHERE application=$1 AND capability=$2 AND subject=$3 AND action=$4 AND resource=$5 FOR UPDATE"
	var source string
	var link, membership, policy int64
	var enabled bool
	err := t.tx.QueryRow(ctx, q, s.Application, s.Capability, s.Subject, s.Action, s.Resource).
		Scan(&source, &link, &membership, &policy, &enabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if link <= 0 || membership <= 0 || policy <= 0 || source == "" {
		return false, fmt.Errorf("PostgreSQL認可正本の版が破損しています")
	}
	return enabled && source == e.Source && uint64(link) == e.SubjectLink &&
		uint64(membership) == e.Membership && uint64(policy) == e.Policy, nil
}

func (t *transaction) FindReceipt(ctx context.Context, operationID string) (effectcommit.Receipt, bool, error) {
	q := "SELECT application, capability, subject, action, resource, request_hash, resource_version FROM " +
		t.table("receipts") + " WHERE operation_id=$1"
	var rec effectcommit.Receipt
	var version int64
	err := t.tx.QueryRow(ctx, q, operationID).Scan(&rec.Scope.Application, &rec.Scope.Capability,
		&rec.Scope.Subject, &rec.Scope.Action, &rec.Scope.Resource, &rec.RequestHash, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		return effectcommit.Receipt{}, false, nil
	}
	if err != nil {
		return effectcommit.Receipt{}, false, err
	}
	if version <= 0 {
		return effectcommit.Receipt{}, false, fmt.Errorf("PostgreSQL確定記録の対象版が不正です")
	}
	rec.OperationID = operationID
	rec.ResourceVersion = uint64(version)
	return rec, true, nil
}

func (t *transaction) WriteResource(ctx context.Context, req effectcommit.Request) (uint64, error) {
	if req.ExpectedVersion >= math.MaxInt64 {
		return 0, effectcommit.ErrResourceConflict
	}
	q := "UPDATE " + t.table("objects") +
		" SET value=$3, version=version+1 WHERE application=$1 AND resource=$2 AND version=$4 RETURNING version"
	var next int64
	err := t.tx.QueryRow(ctx, q, req.Scope.Application, req.Scope.Resource,
		req.Payload, int64(req.ExpectedVersion)).Scan(&next)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, effectcommit.ErrResourceConflict
	}
	if err != nil {
		return 0, err
	}
	if next <= 0 {
		return 0, fmt.Errorf("PostgreSQLの対象版が不正です")
	}
	return uint64(next), nil
}

func (t *transaction) SaveReceipt(ctx context.Context, r effectcommit.Receipt) error {
	if r.ResourceVersion == 0 || r.ResourceVersion > math.MaxInt64 {
		return effectcommit.ErrInvalidRequest
	}
	q := "INSERT INTO " + t.table("receipts") +
		" (operation_id, application, capability, subject, action, resource, request_hash, resource_version)" +
		" VALUES ($1,$2,$3,$4,$5,$6,$7,$8)"
	_, err := t.tx.Exec(ctx, q, r.OperationID, r.Scope.Application, r.Scope.Capability,
		r.Scope.Subject, r.Scope.Action, r.Scope.Resource, r.RequestHash, int64(r.ResourceVersion))
	return err
}

func (t *transaction) Commit(ctx context.Context) error { return t.tx.Commit(ctx) }
func (t *transaction) Rollback() error { return t.tx.Rollback(context.Background()) }

var _ effectcommit.Provider = Provider{}
var _ effectcommit.Transaction = (*transaction)(nil)
