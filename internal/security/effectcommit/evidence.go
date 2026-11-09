package effectcommit

import (
	"context"
	"errors"
	"unicode"

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
	// ProvenanceProvisionalGrant marks Evidence whose effect Grant came from the
	// development-only provisional issuer (provisional owner decision D8,
	// internal/security/provisionaleffectgrant). It is a mark of reduced trust:
	// such Evidence must never be counted for G2. The composition supplies it;
	// existing values are unchanged.
	ProvenanceProvisionalGrant Provenance = "provisional-grant"
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
//
// Limits a reader must know: a request that PASSES validation has its Scope
// fields and OperationID copied verbatim into Evidence, so Evidence is only as
// secret-free as the identifiers the caller chooses; keep secrets out of
// identifiers. A panic inside the Provider itself (for example in Begin)
// escapes Execute exactly as it did before this type existed, and then no
// Evidence is recorded for that call.
type Evidence struct {
	Operation      string // "execute" or "lookup"
	Scope          Scope
	Epochs         legacyauthorization.AuthorityEpochs
	OperationID    string
	Status         Status     // empty when ErrorClass is set
	ErrorClass     ErrorClass // empty when Status is set
	ReceiptVersion uint64     // receipt resource version for committed/replayed
	Provenance     Provenance
	Guarantee      GuaranteeLevel // the Provider's SELF-DECLARED level (not verified), or undeclared
	// GuaranteeReason is the Provider's fixed, secret-free reason text for its
	// declaration (for example "provisional D10: ..."), empty when undeclared.
	GuaranteeReason string
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

// MaxGuaranteeReasonRunes bounds the reason text copied into Evidence.
const MaxGuaranteeReasonRunes = 200

// sanitizeReason drops control characters (newlines, NUL, escapes) and
// truncates to MaxGuaranteeReasonRunes so a Provider cannot inject log lines or
// bulk text through the declaration. It does not detect secrets: a reason must
// still be fixed, secret-free text.
func sanitizeReason(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if unicode.IsControl(r) {
			continue
		}
		out = append(out, r)
		if len(out) == MaxGuaranteeReasonRunes {
			break
		}
	}
	return string(out)
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
	// Everything in emit runs AFTER the effect has finished (possibly committed),
	// so nothing in it may change what the caller sees: not the sink, not the
	// Provider's declaration, not building the Evidence. Recover from all of it.
	// If building the Evidence itself panics, no Evidence is recorded (the
	// outcome is preserved at the price of exactly-once).
	defer func() { _ = recover() }()
	prov := r.Audit.Provenance
	if prov == "" {
		prov = ProvenanceUnspecified
	}
	declared := DeclaredGuarantee(r.Provider)
	ev := Evidence{Operation: op, Provenance: prov, Guarantee: declared.Level, GuaranteeReason: sanitizeReason(declared.Reason)}
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
	r.Audit.Sink.Record(ev)
}
