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
