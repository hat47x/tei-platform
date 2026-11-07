// Package commandadapter connects a Command execution to the authorized
// effect-commit boundary. It does not reimplement authorization or receipts.
package commandadapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"tei/internal/command"
	"tei/internal/security/effectcommit"
	"tei/internal/security/legacyauthorization"
	"tei/internal/semantic"
)

// Executor implements command.Executor by delegating to effectcommit.Runner.
// OperationID, expected version, payload, and grant are fixed by the caller
// that owns the authorization decision.
type Executor struct {
	Runner          effectcommit.Runner
	Grant           legacyauthorization.AuthorizationGrant
	OperationID     string
	ExpectedVersion uint64
	Payload         []byte
	// Effect is the capability's declared effect, supplied by the caller that
	// resolved the Command plan. The effectcommit Runner returns only a receipt,
	// so without this the adapter cannot report which effect it committed.
	// A nil Effect keeps the adapter's previous receipt-only behaviour.
	Effect *semantic.DataEffect
	// Lookup is an optional independent read of the committed operation. When
	// set, Observe calls Runner.Lookup with the same request and grant. Lookup
	// re-checks authority and reads the receipt out of the same database
	// boundary; it does not read the resource's current value, so this is
	// receipt observation, not record observation.
	Lookup bool
}

// Execute calls the authorized effect-commit boundary and maps its status to a
// command result. Denial and resource conflict are execution errors here; the
// caller can distinguish them with errors.Is.
func (e Executor) Execute(ctx context.Context, request command.Request) (command.Result, error) {
	payload := e.Payload
	if payload == nil {
		encoded, err := json.Marshal(request.Inputs)
		if err != nil {
			return command.Result{}, err
		}
		payload = encoded
	}
	scope := effectcommit.Scope{
		Application: string(request.Application),
		Capability:  string(request.Capability),
		Subject:     e.Grant.SubjectID,
		Action:      e.Grant.Action,
		Resource:    e.Grant.Resource,
	}
	result, err := e.Runner.Execute(ctx, effectcommit.Request{
		Scope:           scope,
		OperationID:     e.OperationID,
		ExpectedVersion: e.ExpectedVersion,
		Payload:         payload,
	}, e.Grant)
	if err != nil {
		// A lost commit response means the effect may or may not be applied.
		// Preserve that as an unknown-effect execution error so the Command
		// lifecycle refuses a same-attempt replay and requires a takeover.
		var uncertain *effectcommit.CommitUncertainError
		if errors.As(err, &uncertain) {
			return command.Result{}, command.NewUnknownExecutionError("COMMAND_COMMIT_UNCERTAIN", uncertain.Error(), err)
		}
		return command.Result{}, err
	}
	switch result.Status {
	case effectcommit.StatusCommitted, effectcommit.StatusReplayed:
		// Report the declared effect only when the caller supplied it. A
		// Replayed receipt proves the same operation id and request hash were
		// recorded before; it does not prove the resource's current value, so do
		// not upgrade this to an independent read.
		return command.Result{Effect: e.Effect}, nil
	case effectcommit.StatusDenied:
		return command.Result{}, effectcommit.ErrBusinessDenied
	case effectcommit.StatusConflict:
		return command.Result{}, effectcommit.ErrResourceConflict
	default:
		return command.Result{}, fmt.Errorf("effect commit returned unsupported status %s", result.Status)
	}
}

// Observe performs the optional independent read that the Command boundary
// asks for after a successful Commit. It re-runs the same authorized receipt
// lookup with the same OperationID and payload. A replayed receipt proves the
// operation id and request hash were recorded; it does not prove the resource's
// current value, so the Command boundary must not upgrade this to "the
// declared effect is visible in the data".
func (e Executor) Observe(ctx context.Context, request command.Request, declared semantic.DataEffect) error {
	if !e.Lookup {
		return errors.New("effectcommit adapter has no lookup observation configured")
	}
	payload, err := e.payload(request)
	if err != nil {
		return err
	}
	scope := effectcommit.Scope{
		Application: string(request.Application),
		Capability:  string(request.Capability),
		Subject:     e.Grant.SubjectID,
		Action:      e.Grant.Action,
		Resource:    e.Grant.Resource,
	}
	result, err := e.Runner.Lookup(ctx, effectcommit.Request{
		Scope:           scope,
		OperationID:     e.OperationID,
		ExpectedVersion: e.ExpectedVersion,
		Payload:         payload,
	}, e.Grant)
	if err != nil {
		return err
	}
	if result.Status != effectcommit.StatusReplayed {
		return fmt.Errorf("effect commit lookup returned %s, not a replayed receipt", result.Status)
	}
	return nil
}

func (e Executor) payload(request command.Request) ([]byte, error) {
	if e.Payload != nil {
		return e.Payload, nil
	}
	return json.Marshal(request.Inputs)
}
