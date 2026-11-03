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
	"tei/internal/security/effectcommit/submit"
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
	status      orderStatusReader
	closer      func() error
}

// orderStatusReader is the read-only half of orderSubmitStore. The Observation
// read uses it so that this store has no field whose type could also write.
type orderStatusReader interface {
	ReadStatus(ctx context.Context, orderID int64) (string, error)
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

// runnerStoreOption configures optional behaviour of the Postgres-backed store.
type runnerStoreOption func(*runnerStoreConfig)

type runnerStoreConfig struct{ audit effectcommit.AuditConfig }

// withRunnerAudit wires an opt-in Evidence sink into the Runner this store
// builds. The Provenance label is supplied by the COMPOSITION (real-pg only
// where the code really runs against PostgreSQL); the store never infers it.
// Without this option no Evidence is built. The shipped CLI does not use it.
func withRunnerAudit(a effectcommit.AuditConfig) runnerStoreOption {
	return func(c *runnerStoreConfig) { c.audit = a }
}

var errRunnerSubmitFailed = errors.New("order submit effect failed")

func newRunnerOrderSubmitStore(
	runner submitRunner,
	grantSource submitGrantSource,
	schemaCheck func(context.Context) error,
	status orderStatusReader,
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
//
// Optional runnerStoreOption values (for example withRunnerAudit) are additive;
// with none the store behaves exactly as before.
func newPostgresRunnerOrderSubmitStore(
	ctx context.Context,
	writeDSN, readDSN, effectSchema string,
	world ir.World,
	grantSource submitGrantSource,
	opts ...runnerStoreOption,
) (*runnerOrderSubmitStore, error) {
	var cfg runnerStoreConfig
	for _, o := range opts {
		if o != nil {
			o(&cfg)
		}
	}
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
	store, err := newRunnerOrderSubmitStore(effectcommit.Runner{Provider: provider, Audit: cfg.audit}, grantSource, check, direct)
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

// runnerSubmitSpec carries the shared intent check (D11): the Target guarantees
// the payload agrees with the OperationID before the Runner is reached.
var runnerSubmitSpec = submit.Spec{
	Application:    runnerSubmitApplication,
	Capability:     runnerSubmitCapability,
	ValidateIntent: checkOrderSubmitIntent,
}

// runnerSubmitIntent composes the Intent for one order.
func runnerSubmitIntent(orderID int64) (submit.Intent, error) {
	payload, err := json.Marshal(struct {
		OrderID int64  `json:"orderId"`
		To      string `json:"to"`
	}{orderID, "submitted"})
	if err != nil {
		return submit.Intent{}, err
	}
	return submit.Intent{
		// Deterministic, so an unknown commit can be recovered by Lookup with the
		// same OperationID instead of a new one.
		OperationID:     fmt.Sprintf("order-submit/%d", orderID),
		ExpectedVersion: runnerSubmitExpectedVersion,
		Payload:         payload,
	}, nil
}

func runnerSubmitRequest(grant legacyauthorization.AuthorizationGrant, orderID int64) (effectcommit.Request, error) {
	in, err := runnerSubmitIntent(orderID)
	if err != nil {
		return effectcommit.Request{}, err
	}
	return runnerSubmitRequestFromIntent(grant, in)
}

// runnerSubmitRequestFromIntent builds the Request and runs the shared intent
// check first; a mismatching Intent yields submit.ErrIntentRejected.
func runnerSubmitRequestFromIntent(grant legacyauthorization.AuthorizationGrant, in submit.Intent) (effectcommit.Request, error) {
	return submit.RequestFromGrant(runnerSubmitSpec, grant, in)
}

func (s *runnerOrderSubmitStore) grantFor(ctx context.Context, orderID int64) (legacyauthorization.AuthorizationGrant, error) {
	var src submit.GrantSource
	if s.grantSource != nil {
		src = func(c context.Context) (legacyauthorization.AuthorizationGrant, error) {
			return s.grantSource(c, orderID)
		}
	}
	grant, err := submit.AcquireGrant(ctx, src)
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
	if errors.Is(err, submit.ErrIntentRejected) {
		return errOrderNotSubmittable // refused before execution; the Runner is not called
	}
	if err != nil {
		return errRunnerSubmitFailed
	}
	result, err := s.runner.Execute(ctx, req, grant)
	return mapSubmitOutcome(result, err)
}

// LookupSubmit asks the Runner for the receipt of the deterministic OperationID
// of this order. It is not on orderSubmitStore and is used only in tests.
//
// Recovery after errOrderSubmitCommitUnconfirmed: a retry of Submit with the same
// OperationID is reported as errOrderNotSubmittable even when the earlier
// attempt did commit (the Runner answers Replayed). A caller must not retry;
// it recovers with LookupSubmit (the receipt) or ReadStatus (the row).
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
// sentinels. It is a thin wrapper over submit.Classify; the neutral layer keeps
// Replayed apart from NotCommitted and only this Web wrapper collapses them.
// No driver text, DSN or Runner detail is carried through.
func mapSubmitOutcome(result effectcommit.Result, err error) error {
	switch submit.Classify(result, err).Disposition {
	case submit.Committed:
		return nil
	case submit.NotCommitted:
		return errOrderNotSubmittable
	case submit.Replayed:
		// This request did not perform the write; keep the pinned "second submit
		// of an already-submitted order is refused" behaviour.
		return errOrderNotSubmittable
	case submit.Uncertain:
		return errOrderSubmitCommitUnconfirmed
	default:
		return errRunnerSubmitFailed
	}
}

var _ orderSubmitStore = (*runnerOrderSubmitStore)(nil)
