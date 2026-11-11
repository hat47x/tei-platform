// Package interfaceadapter is a synthetic cross-transport harness for Issue
// #1158 conditions 4 and 5. It compares the same typed Command operation through
// an in-process adapter, a real net/http adapter, and a local IPC adapter.
//
// It is NOT production authentication, session storage, CSRF protection,
// WebView, Wails, or a desktop packaging proof. The HTTP session verifier and
// the IPC principal are fixed fixtures; neither is an accepted identity source.
package interfaceadapter

import (
	"context"
	"fmt"

	"tei/internal/command"
	"tei/internal/core"
	"tei/internal/failure"
	"tei/internal/ir"
)

// Principal is the verified identity handed to an adapter by its host. It is
// never decoded from an HTTP body, query string or IPC message.
type Principal struct {
	Subject string
	Role    string
}

// Request is the typed operation input shared by the adapters.
type Request struct {
	Application string            `json:"application"`
	Pattern     string            `json:"pattern"`
	Capability  string            `json:"capability"`
	Inputs      map[string]string `json:"inputs,omitempty"`
	// Revision is the expected workspace revision. An empty value is not a
	// wildcard: it is compared like any other value.
	Revision string `json:"revision"`
}

// Failure separates a typed transport/business rejection from a result.
type Failure struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// FailureCode returns the machine-readable failure code.
func (f Failure) FailureCode() string { return f.Code }

// FailureMessage returns the human-readable failure message.
func (f Failure) FailureMessage() string { return f.Message }

// FailureCause returns nil because the adapter Failure does not wrap an
// underlying error.
func (f Failure) FailureCause() error { return nil }

// Kind classifies the business meaning of the failure.
func (f Failure) Kind() failure.Kind {
	switch f.Code {
	case "UNAUTHENTICATED", "FORBIDDEN", "STALE_REVISION":
		return failure.BusinessRejection
	default:
		return failure.TechnicalFailure
	}
}

// Retryable reports whether the caller may safely retry the same operation
// identity after this failure.
func (f Failure) Retryable() failure.RetryPolicy {
	switch f.Code {
	case "UNAUTHENTICATED", "FORBIDDEN", "STALE_REVISION":
		return failure.NotRetryable
	default:
		return failure.NotRetryable
	}
}

// HasEffectUnknown reports whether the failure may have left partial effects
// applied. Adapter transport failures are clean refusals with no side effects,
// so this is always false.
func (f Failure) HasEffectUnknown() bool { return false }

// Result is the typed operation result shared by the adapters.
type Result struct {
	Outcome  string   `json:"outcome,omitempty"`
	Revision string   `json:"revision,omitempty"`
	Failure  *Failure `json:"failure,omitempty"`
}

// OK reports whether the operation produced a business outcome.
func (r Result) OK() bool { return r.Failure == nil }

// Core is the transport-independent Command boundary. The adapters differ only
// in how they deliver a verified Principal and a Request to this function.
type Core struct {
	World    ir.World
	Revision string
	Invoker  command.Invoker
}

// NewCore builds a reference Command boundary. It uses the existing
// ReferenceExecutor and MemoryUnitOfWork, so it has no durable effect.
func NewCore(world ir.World, revision string) Core {
	return Core{
		World:    world,
		Revision: revision,
		Invoker: command.Invoker{
			World:      world,
			Executor:   command.ReferenceExecutor{World: world},
			UnitOfWork: &command.MemoryUnitOfWork{},
			Authorizer: command.ReferenceAuthorizer{},
		},
	}
}

// Invoke is the single operation implementation. All transports call it with a
// Principal their host already verified.
func (c Core) Invoke(ctx context.Context, principal Principal, request Request) Result {
	if principal.Subject == "" {
		return Result{Failure: &Failure{Code: "UNAUTHENTICATED", Message: "no verified principal"}}
	}
	if principal.Role != "editor" {
		return Result{Failure: &Failure{Code: "FORBIDDEN", Message: "the principal may not execute this action"}}
	}
	if request.Revision != c.Revision {
		return Result{Failure: &Failure{Code: "STALE_REVISION", Message: fmt.Sprintf("expected revision %q, got %q", c.Revision, request.Revision)}}
	}
	result, err := c.Invoker.Invoke(ctx, command.Request{
		Application: core.CanonicalID(request.Application),
		Pattern:     core.CanonicalID(request.Pattern),
		Capability:  core.CanonicalID(request.Capability),
		Inputs:      request.Inputs,
	})
	if err != nil {
		return Result{Failure: &Failure{Code: "COMMAND_FAILED", Message: err.Error()}}
	}
	outcome := ""
	if result.Outcome != nil {
		outcome = string(result.Outcome.Name)
	}
	return Result{Outcome: outcome, Revision: c.Revision}
}

// InProcessAdapter calls Core directly, as a Go host would.
type InProcessAdapter struct {
	Core Core
}

// Execute runs the operation without serialization.
func (a InProcessAdapter) Execute(ctx context.Context, principal Principal, request Request) Result {
	return a.Core.Invoke(ctx, principal, request)
}
