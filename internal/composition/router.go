// Package composition preserves the existing host-facing API while the
// generation lease and cutover implementation lives in core/generation.
package composition

import "tei/core/generation"

var (
	ErrUnbound = generation.ErrUnbound
	ErrStaleGeneration = generation.ErrStaleGeneration
	ErrGuardRequired = generation.ErrGuardRequired
)

type Binding = generation.Binding
type Router = generation.Router
type Lease = generation.Lease
type Retired = generation.Retired

func NewRouter(first Binding, scope *Scope) (*Router, error) {
	return generation.NewRouter(first, scope)
}
