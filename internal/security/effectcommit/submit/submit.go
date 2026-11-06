// Package submit is a target-neutral caller helper for effectcommit.Runner.
// It turns one Intent plus an injected grant into an effectcommit.Request and
// classifies the Runner's answer into a Disposition, so that a Web target and a
// CLI target can share the same request building and outcome reading while each
// keeps its own user-facing wording.
//
// What this package does NOT do:
//   - It mints and validates no authority. The grant is supplied by an injected
//     GrantSource and is checked only by effectcommit.Runner and the Provider.
//   - The Disposition set is an unapproved A2 reference proposal (#1174). Whether
//     the production scheme separates declared denial, common non-execution and
//     adapter execution error this way is an Owner decision.
//   - Uncertain does not always mean "the write result is unknown":
//     effectcommit.Runner.Execute also returns CommitUncertainError from its
//     read-only replay branch (a receipt already existed and only the closing
//     Commit failed). Recover with Lookup in either case.
//   - NotRecorded (from Lookup) means only "no receipt in that database"; it is
//     not proof that no effect happened.
//
// The package imports only the standard library, effectcommit and
// legacyauthorization; an import-guard test enforces that.
package submit

import (
	"context"
	"errors"

	"tei/internal/security/effectcommit"
	"tei/internal/security/legacyauthorization"
)

// Runner has the same shape as effectcommit.Runner.
type Runner interface {
	Execute(context.Context, effectcommit.Request, legacyauthorization.AuthorizationGrant) (effectcommit.Result, error)
	Lookup(context.Context, effectcommit.Request, legacyauthorization.AuthorizationGrant) (effectcommit.Result, error)
}

var _ Runner = effectcommit.Runner{}

// GrantSource returns the grant handed to the Runner. It is caller-supplied;
// there is no default.
type GrantSource func(context.Context) (legacyauthorization.AuthorizationGrant, error)

var (
	// ErrGrantUnavailable is returned when the grant source is nil or fails. It
	// carries no text from the source.
	ErrGrantUnavailable = errors.New("submit: grant unavailable")
	// ErrInvalidIntent is returned before any Runner call when a spec or intent
	// field that this package checks is empty, or the Runner is nil.
	ErrInvalidIntent = errors.New("submit: invalid spec or intent")
	// ErrIntentRejected is returned before any Runner call when Spec.ValidateIntent
	// is set and refuses the Intent. It carries no text from the validator and
	// never the payload.
	ErrIntentRejected = errors.New("submit: intent rejected before execution")
)

// AcquireGrant calls src and fails closed: a nil source or any source error
// yields ErrGrantUnavailable and an empty grant.
func AcquireGrant(ctx context.Context, src GrantSource) (legacyauthorization.AuthorizationGrant, error) {
	if src == nil {
		return legacyauthorization.AuthorizationGrant{}, ErrGrantUnavailable
	}
	grant, err := src(ctx)
	if err != nil {
		return legacyauthorization.AuthorizationGrant{}, ErrGrantUnavailable
	}
	return grant, nil
}

// Spec names the target application and capability of the effect.
//
// ValidateIntent is an optional, target-neutral hook. The Target (the caller
// that composes the Intent) uses it to guarantee that the payload agrees with
// the OperationID and the grant's Resource before anything is executed (Owner
// provisional decision D11). When nil, behaviour is exactly as before. When set
// and it returns an error, RequestFromGrant, Caller.Execute and Caller.Lookup
// refuse with ErrIntentRejected and the Runner is never called. The hook
// receives a private copy of the Intent (the same bytes then go into the
// Request) and must not retain it; a panic counts as a rejection and its error
// text or panic value is discarded so that no payload content can leak.
type Spec struct {
	Application    string
	Capability     string
	ValidateIntent func(Intent) error
}

// Intent is one requested effect. Payload is passed through byte for byte and is
// never re-serialised here; the Runner's request hash covers exactly these bytes.
type Intent struct {
	OperationID     string
	ExpectedVersion uint64
	Payload         []byte
}

// RequestFromGrant builds the Request. Subject, Action and Resource are copied
// only from the grant. It refuses an empty application, capability or operation
// id, and an Intent that Spec.ValidateIntent (when set) rejects. It does not
// validate the grant; effectcommit.Runner does.
func RequestFromGrant(spec Spec, grant legacyauthorization.AuthorizationGrant, in Intent) (effectcommit.Request, error) {
	if spec.Application == "" || spec.Capability == "" || in.OperationID == "" {
		return effectcommit.Request{}, ErrInvalidIntent
	}
	// One private copy of the payload is made. The hook checks exactly this copy
	// and the Request carries exactly this copy, so the bytes that were checked
	// are the bytes that execute (no check/use gap) and the caller's slice is
	// never shared with the hook.
	checked := in
	checked.Payload = append([]byte(nil), in.Payload...)
	if spec.ValidateIntent != nil && !intentAccepted(spec.ValidateIntent, checked) {
		return effectcommit.Request{}, ErrIntentRejected
	}
	return effectcommit.Request{
		Scope: effectcommit.Scope{
			Application: spec.Application,
			Capability:  spec.Capability,
			Subject:     grant.SubjectID,
			Action:      grant.Action,
			Resource:    grant.Resource,
		},
		OperationID:     in.OperationID,
		ExpectedVersion: in.ExpectedVersion,
		Payload:         checked.Payload,
	}, nil
}

// intentAccepted runs the hook and reports whether it accepted. An error or a
// panic is a rejection; neither the error text nor the panic value is kept.
func intentAccepted(validate func(Intent) error, in Intent) (ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	return validate(in) == nil
}

// Disposition is the neutral reading of a Runner answer.
type Disposition string

const (
	// Committed: this call performed the write and the Commit was acknowledged.
	Committed Disposition = "committed"
	// Replayed: a matching receipt already existed; this call did not write.
	// It is deliberately separate from NotCommitted; each target decides what
	// it means to its user.
	Replayed Disposition = "replayed"
	// NotCommitted: the Runner declared Denied or Conflict. Status tells which.
	NotCommitted Disposition = "not-committed"
	// Uncertain: the Commit answer was lost (see the package comment).
	Uncertain Disposition = "uncertain"
	// Failed: any other error, or a status that is a contract violation here.
	Failed Disposition = "failed"
	// NotRecorded: Lookup only. No receipt in that database.
	NotRecorded Disposition = "not-recorded"
)

// Outcome carries the Disposition, the Runner Status (so a target can tell
// Denied from Conflict) and the OperationID when known. It carries no error text.
type Outcome struct {
	Disposition Disposition
	Status      effectcommit.Status
	OperationID string
}

// Classify reads the answer of Runner.Execute.
func Classify(result effectcommit.Result, err error) Outcome {
	if err != nil {
		var uncertain *effectcommit.CommitUncertainError
		if errors.As(err, &uncertain) {
			return Outcome{Disposition: Uncertain}
		}
		return Outcome{Disposition: Failed}
	}
	out := Outcome{Status: result.Status, OperationID: result.Receipt.OperationID}
	switch result.Status {
	case effectcommit.StatusCommitted:
		out.Disposition = Committed
	case effectcommit.StatusReplayed:
		out.Disposition = Replayed
	case effectcommit.StatusDenied, effectcommit.StatusConflict:
		out.Disposition = NotCommitted
	default:
		// NotRecorded or an unknown status from Execute is a contract violation.
		out.Disposition = Failed
	}
	return out
}

// ClassifyLookup reads the answer of Runner.Lookup. A raw error is Failed, never
// Uncertain: Lookup does not write.
func ClassifyLookup(result effectcommit.Result, err error) Outcome {
	if err != nil {
		return Outcome{Disposition: Failed}
	}
	out := Outcome{Status: result.Status, OperationID: result.Receipt.OperationID}
	switch result.Status {
	case effectcommit.StatusReplayed:
		out.Disposition = Replayed
	case effectcommit.StatusNotRecorded:
		out.Disposition = NotRecorded
	case effectcommit.StatusDenied, effectcommit.StatusConflict:
		out.Disposition = NotCommitted
	default:
		out.Disposition = Failed
	}
	return out
}

// Caller bundles a Runner, a GrantSource and a Spec.
type Caller struct {
	Runner Runner
	Grants GrantSource
	Spec   Spec
}

// refusalOutcome reads a refusal before any Runner call: a rejected Intent is
// NotCommitted (nothing ran), anything else is Failed. The error is still
// returned to the caller alongside.
func refusalOutcome(err error, in Intent) Outcome {
	if errors.Is(err, ErrIntentRejected) {
		return Outcome{Disposition: NotCommitted, OperationID: in.OperationID}
	}
	return Outcome{Disposition: Failed}
}

// Grant acquires the grant, failing closed.
func (c Caller) Grant(ctx context.Context) (legacyauthorization.AuthorizationGrant, error) {
	return AcquireGrant(ctx, c.Grants)
}

// Execute builds the request and runs it. The error is non-nil only for refusals
// before any Runner call (ErrInvalidIntent, or ErrIntentRejected with a
// NotCommitted Outcome).
func (c Caller) Execute(ctx context.Context, grant legacyauthorization.AuthorizationGrant, in Intent) (Outcome, error) {
	if c.Runner == nil {
		return Outcome{Disposition: Failed}, ErrInvalidIntent
	}
	req, err := RequestFromGrant(c.Spec, grant, in)
	if err != nil {
		return refusalOutcome(err, in), err
	}
	out := Classify(c.Runner.Execute(ctx, req, grant))
	if out.OperationID == "" {
		out.OperationID = in.OperationID
	}
	return out, nil
}

// Lookup rebuilds the same request and asks the Runner for its receipt.
func (c Caller) Lookup(ctx context.Context, grant legacyauthorization.AuthorizationGrant, in Intent) (Outcome, error) {
	if c.Runner == nil {
		return Outcome{Disposition: Failed}, ErrInvalidIntent
	}
	req, err := RequestFromGrant(c.Spec, grant, in)
	if err != nil {
		return refusalOutcome(err, in), err
	}
	out := ClassifyLookup(c.Runner.Lookup(ctx, req, grant))
	if out.OperationID == "" {
		out.OperationID = in.OperationID
	}
	return out, nil
}
