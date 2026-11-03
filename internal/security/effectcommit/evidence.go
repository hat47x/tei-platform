package effectcommit

import (
	"context"
	"errors"

	"tei/internal/security/legacyauthorization"
)

// Provenance labels where the evidence came from. It is supplied by the
// composition, never inferred by the Runner. Unapproved A2 reference proposal,
// not a ratified contract.
type Provenance string

const (
	ProvenanceUnspecified Provenance = "unspecified"
	ProvenanceSynthetic   Provenance = "synthetic"
	ProvenanceRealPG      Provenance = "real-pg"
)

// ErrorClass is a coarse class of an Execution Error. It never carries the
// underlying error text (which may contain DSNs, SQL or payload fragments).
type ErrorClass string

const (
	ErrorClassNone            ErrorClass = ""
	ErrorClassInvalidRequest  ErrorClass = "invalid-request"
	ErrorClassMissingProvider ErrorClass = "missing-provider"
	ErrorClassCommitUncertain ErrorClass = "commit-uncertain"
	ErrorClassContext         ErrorClass = "context"
	ErrorClassExecution       ErrorClass = "execution-error"
)

// Evidence is a secret-free record of one Execute/Lookup outcome (Issue #1174
// requirement 8). Unapproved A2 reference proposal, not a ratified contract.
// It deliberately has no payload, request hash, DSN, token or error-text field.
// For calls rejected before the request is validated, Scope, Epochs and
// OperationID are left empty because they are still untrusted input.
type Evidence struct {
	Operation      string // "execute" or "lookup"
	Scope          Scope
	Epochs         legacyauthorization.AuthorityEpochs
	OperationID    string
	Status         Status     // empty when ErrorClass is set
	ErrorClass     ErrorClass // empty when Status is set
	ReceiptVersion uint64     // receipt resource version for committed/replayed
	Provenance     Provenance
	Guarantee      GuaranteeLevel // Provider's declared level, or undeclared
}

// EvidenceSink receives Evidence. It has no error return: a sink failure must
// not change the effect outcome. The Runner recovers a sink panic for the same
// reason; a sink must still be non-blocking or bounded by its owner.
type EvidenceSink interface{ Record(Evidence) }

// AuditConfig is the optional audit wiring on Runner. The zero value (nil
// Sink) means no Evidence is built and behaviour is identical to a Runner
// without it.
type AuditConfig struct {
	Sink       EvidenceSink
	Provenance Provenance
}

// Execute runs the transactional order described on execute and, if an audit
// sink is configured, records exactly one Evidence for the call.
func (r Runner) Execute(ctx context.Context, req Request, grant legacyauthorization.AuthorizationGrant) (Result, error) {
	res, err := r.execute(ctx, req, grant)
	if r.Audit.Sink != nil {
		r.emit("execute", req, grant, res, err)
	}
	return res, err
}

// Lookup runs the lookup described on lookup and, if an audit sink is
// configured, records exactly one Evidence for the call.
func (r Runner) Lookup(ctx context.Context, req Request, grant legacyauthorization.AuthorizationGrant) (Result, error) {
	res, err := r.lookup(ctx, req, grant)
	if r.Audit.Sink != nil {
		r.emit("lookup", req, grant, res, err)
	}
	return res, err
}

func classify(err error) ErrorClass {
	var unc *CommitUncertainError
	switch {
	case err == nil:
		return ErrorClassNone
	case errors.As(err, &unc):
		return ErrorClassCommitUncertain
	case errors.Is(err, ErrInvalidRequest):
		return ErrorClassInvalidRequest
	case errors.Is(err, ErrMissingProvider):
		return ErrorClassMissingProvider
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return ErrorClassContext
	default:
		return ErrorClassExecution
	}
}

func (r Runner) emit(op string, req Request, grant legacyauthorization.AuthorizationGrant, res Result, err error) {
	prov := r.Audit.Provenance
	if prov == "" {
		prov = ProvenanceUnspecified
	}
	ev := Evidence{Operation: op, Provenance: prov, Guarantee: DeclaredGuarantee(r.Provider).Level}
	ev.ErrorClass = classify(err)
	if ev.ErrorClass == ErrorClassNone {
		ev.Status = res.Status
		if res.Status == StatusCommitted || res.Status == StatusReplayed {
			ev.ReceiptVersion = res.Receipt.ResourceVersion
		}
	}
	// Only a request that passes validation is trusted enough to be echoed.
	if validate(req, grant) == nil {
		ev.Scope = req.Scope
		ev.OperationID = req.OperationID
		ev.Epochs = *grant.Epochs
	}
	defer func() { _ = recover() }()
	r.Audit.Sink.Record(ev)
}
