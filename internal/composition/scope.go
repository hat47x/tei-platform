package composition

import "tei/core/effect"

// ErrScopeClosed retains the previous error identity for legacy consumers.
// Transient effect cleanup has one implementation, in core/effect.
var ErrScopeClosed = effect.ErrScopeClosed

// Scope is an alias rather than a wrapper or duplicate implementation.
// Existing Lifecycle and Router users keep exactly the same Go type.
type Scope = effect.Scope

// NewScope retains the previous construction API for legacy callers.
func NewScope() *Scope { return effect.NewScope() }
