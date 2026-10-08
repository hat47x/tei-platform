// Package actionboundary is a reference-only commit boundary for native UI actions.
// It does not implement TEI runtime authorization or SUI DocumentV1 persistence.
package actionboundary

import (
 "context"
 "errors"
 "fmt"
)

var (
 ErrInvalid = errors.New("invalid action")
 ErrConflict = errors.New("stale revision")
 ErrDenied = errors.New("action denied")
)

// Intent is emitted only when the UI commits an interaction, never on preview.
type Intent struct {
 ApplicationID string
 DocumentID string
 ActionID string
 ExpectedRevision string
 Payload any
}

// Store is application-owned; the host never interprets its document schema.
// Commit must atomically compare the revision and persist the mutation.
type Store interface {
 Commit(context.Context, Intent) (string, error)
}

// Authorizer must check the caller's runtime authority, not UI claims.
type Authorizer interface {
 Authorize(context.Context, string, string, string) error
}

type Binding struct {
 ApplicationID string
 ActionID string
 Store Store
}

type Host struct {
 Bindings []Binding
 Authorizer Authorizer
 // CommandGate, when supplied, validates the action against TEI Canonical.
 // It does not replace runtime authorization.
 CommandGate *CanonicalCommandGate
}

func (h Host) Commit(ctx context.Context, in Intent) (string, error) {
 if in.ApplicationID == "" || in.DocumentID == "" || in.ActionID == "" || in.ExpectedRevision == "" {
  return "", ErrInvalid
 }
 if h.Authorizer == nil { return "", fmt.Errorf("%w: authorizer unavailable", ErrDenied) }
 var selected *Binding
 for i := range h.Bindings {
  b := &h.Bindings[i]
  if b.ApplicationID != in.ApplicationID || b.ActionID != in.ActionID { continue }
  if selected != nil { return "", fmt.Errorf("%w: ambiguous binding", ErrInvalid) }
  selected = b
 }
 if selected == nil || selected.Store == nil { return "", fmt.Errorf("%w: unbound application action", ErrDenied) }
 if h.CommandGate != nil {
  if err := h.CommandGate.Check(ctx,in); err != nil { return "",err }
 }
 if err := h.Authorizer.Authorize(ctx, in.ApplicationID, in.DocumentID, in.ActionID); err != nil {
  return "", fmt.Errorf("%w: %v", ErrDenied, err)
 }
 return selected.Store.Commit(ctx, in)
}
