//go:build !tei_repair_only

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"tei/internal/ir"
	"tei/internal/security/effectcommit"
	effectpostgres "tei/internal/security/effectcommit/postgres"
	"tei/internal/security/legacyauthorization"
)

// runnerOrderSubmitStore is an opt-in orderSubmitStore whose Submit goes
// through the formal effectcommit.Runner (authority check, receipt lookup,
// versioned write, receipt save, Commit in one Provider transaction) instead
// of the direct UPDATE + Commit of postgresOrderSubmitStore. It is selected
// only by injecting it as webHostWiring.submitStore; the shipped CLI does not
// use it and the direct store is neither replaced nor retired.
//
// It is NOT an Authority. The grant that the Runner checks is supplied by an
// injected, synthetic grantSource and is not the request's own business
// authorization grant. The Outcome-to-Stage mapping below is an unapproved A2
// reference proposal, not a production disposition mapping. A green run against
// a disposable PostgreSQL is a reference result, not proof of production
// atomicity or durability.
type runnerOrderSubmitStore struct {
	runner      submitRunner
	grantSource submitGrantSource
	schemaCheck func(context.Context) error
	status      orderSubmitStore
	closer      func() error
}

// submitRunner is the part of effectcommit.Runner this store uses. A fake can
// stand in for it in tests.
type submitRunner interface {
	Execute(context.Context, effectcommit.Request, legacyauthorization.AuthorizationGrant) (effectcommit.Result, error)
	Lookup(context.Context, effectcommit.Request, legacyauthorization.AuthorizationGrant) (effectcommit.Result, error)
}

var _ submitRunner = effectcommit.Runner{}

// submitGrantSource returns the grant handed to the Runner for one order. It is
// SYNTHETIC and caller-supplied: there is no default, the store never mints a
// grant, and a nil or failing source fails closed before Execute is reached.
type submitGrantSource func(ctx context.Context, orderID int64) (legacyauthorization.AuthorizationGrant, error)

const (
	runnerSubmitApplication = "order_entry"
	runnerSubmitCapability  = "order_entry.submit_order"
	// The resource version is state-derived and nominal: draft=0, submitted=1.
	// databases/sales-core has no version column; the status='draft' predicate in
	// the Owner routine is the only real guard.
	runnerSubmitExpectedVersion = 0
)

var errRunnerSubmitFailed = errors.New("order submit effect failed")

func newRunnerOrderSubmitStore(
	runner submitRunner,
	grantSource submitGrantSource,
	schemaCheck func(context.Context) error,
	status orderSubmitStore,
) (*runnerOrderSubmitStore, error) {
	if runner == nil || grantSource == nil || status == nil {
		return nil, errors.New("runner order submit store requires a runner, a grant source and a status reader")
	}
	return &runnerOrderSubmitStore{runner: runner, grantSource: grantSource, schemaCheck: schemaCheck, status: status}, nil
}

// newPostgresRunnerOrderSubmitStore opens a pgx pool from the write DSN for the
// effectcommit Provider (trusted Owner routine "apply_owner_effect" in the
// fixed schema, which the deployment must install) and keeps the existing
// separate read pool for the Observation. Errors never carry the DSN.
func newPostgresRunnerOrderSubmitStore(
	ctx context.Context,
	writeDSN, readDSN, effectSchema string,
	world ir.World,
	grantSource submitGrantSource,
) (*runnerOrderSubmitStore, error) {
	direct, err := newPostgresOrderSubmitStore(writeDSN, readDSN)
	if err != nil {
		return nil, errors.New("open order submit store failed")
	}
	pool, err := pgxpool.New(ctx, writeDSN)
	if err != nil {
		_ = direct.Close()
		return nil, errors.New("open effect commit pool failed")
	}
	provider, err := effectpostgres.NewWithTrustedOwnerRoutine(pool, effectSchema)
	if err != nil {
		pool.Close()
		_ = direct.Close()
		return nil, errors.New("effect commit provider configuration invalid")
	}
	check := func(c context.Context) error {
		_, err := direct.checkOrderSubmitSchema(c, world)
		return err
	}
	store, err := newRunnerOrderSubmitStore(effectcommit.Runner{Provider: provider}, grantSource, check, direct)
	if err != nil {
		pool.Close()
		_ = direct.Close()
		return nil, err
	}
	store.closer = func() error {
		pool.Close()
		return direct.Close()
	}
	return store, nil
}

func (s *runnerOrderSubmitStore) Close() error {
	if s.closer == nil {
		return nil
	}
	return s.closer()
}

func runnerSubmitRequest(grant legacyauthorization.AuthorizationGrant, orderID int64) (effectcommit.Request, error) {
	payload, err := json.Marshal(struct {
		OrderID int64  `json:"orderId"`
		To      string `json:"to"`
	}{orderID, "submitted"})
	if err != nil {
		return effectcommit.Request{}, err
	}
	return effectcommit.Request{
		Scope: effectcommit.Scope{
			Application: runnerSubmitApplication,
			Capability:  runnerSubmitCapability,
			Subject:     grant.SubjectID,
			Action:      grant.Action,
			Resource:    grant.Resource,
		},
		// Deterministic, so an unknown commit can be recovered by Lookup with the
		// same OperationID instead of a new one.
		OperationID:     fmt.Sprintf("order-submit/%d", orderID),
		ExpectedVersion: runnerSubmitExpectedVersion,
		Payload:         payload,
	}, nil
}

func (s *runnerOrderSubmitStore) grantFor(ctx context.Context, orderID int64) (legacyauthorization.AuthorizationGrant, error) {
	if s.grantSource == nil {
		return legacyauthorization.AuthorizationGrant{}, errRunnerSubmitFailed
	}
	grant, err := s.grantSource(ctx, orderID)
	if err != nil {
		return legacyauthorization.AuthorizationGrant{}, errRunnerSubmitFailed
	}
	return grant, nil
}

func (s *runnerOrderSubmitStore) Submit(ctx context.Context, orderID int64) error {
	grant, err := s.grantFor(ctx, orderID)
	if err != nil {
		return err
	}
	if s.schemaCheck != nil {
		if err := s.schemaCheck(ctx); err != nil {
			return errPhysicalSchemaNotReady
		}
	}
	req, err := runnerSubmitRequest(grant, orderID)
	if err != nil {
		return errRunnerSubmitFailed
	}
	result, err := s.runner.Execute(ctx, req, grant)
	return mapSubmitOutcome(result, err)
}

// LookupSubmit asks the Runner for the receipt of the deterministic OperationID
// of this order. It is not on orderSubmitStore and is used only in tests.
// NotRecorded means only "no record in the same DB".
func (s *runnerOrderSubmitStore) LookupSubmit(ctx context.Context, orderID int64) (effectcommit.Result, error) {
	grant, err := s.grantFor(ctx, orderID)
	if err != nil {
		return effectcommit.Result{}, err
	}
	req, err := runnerSubmitRequest(grant, orderID)
	if err != nil {
		return effectcommit.Result{}, errRunnerSubmitFailed
	}
	return s.runner.Lookup(ctx, req, grant)
}

func (s *runnerOrderSubmitStore) ReadStatus(ctx context.Context, orderID int64) (string, error) {
	return s.status.ReadStatus(ctx, orderID)
}

// mapSubmitOutcome is the only place Runner outcomes become the existing
// sentinels. No driver text, DSN or Runner detail is carried through.
func mapSubmitOutcome(result effectcommit.Result, err error) error {
	if err != nil {
		var uncertain *effectcommit.CommitUncertainError
		if errors.As(err, &uncertain) {
			return errOrderSubmitCommitUnconfirmed
		}
		return errRunnerSubmitFailed
	}
	switch result.Status {
	case effectcommit.StatusCommitted:
		return nil
	case effectcommit.StatusDenied, effectcommit.StatusConflict:
		return errOrderNotSubmittable
	case effectcommit.StatusReplayed:
		// This request did not perform the write; keep the pinned "second submit
		// of an already-submitted order is refused" behaviour.
		return errOrderNotSubmittable
	default:
		// NotRecorded or an unknown status is not a committed write.
		return errRunnerSubmitFailed
	}
}

var _ orderSubmitStore = (*runnerOrderSubmitStore)(nil)
