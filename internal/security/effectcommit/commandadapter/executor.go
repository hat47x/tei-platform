// Package commandadapter connects a Command execution to the authorized
// effect-commit boundary. It does not reimplement authorization or receipts.
package commandadapter

import (
	"context"
	"encoding/json"
	"fmt"

	"tei/internal/command"
	"tei/internal/security/effectcommit"
	"tei/internal/security/legacyauthorization"
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
		return command.Result{}, err
	}
	switch result.Status {
	case effectcommit.StatusCommitted, effectcommit.StatusReplayed:
		return command.Result{}, nil
	case effectcommit.StatusDenied:
		return command.Result{}, effectcommit.ErrBusinessDenied
	case effectcommit.StatusConflict:
		return command.Result{}, effectcommit.ErrResourceConflict
	default:
		return command.Result{}, fmt.Errorf("effect commit returned unsupported status %s", result.Status)
	}
}
