package effectcommit

import (
	"context"
	"errors"
)

// ErrResourceObservationUnavailable means the Provider cannot read the
// resource's current state, so a caller cannot turn a committed receipt into a
// resource observation. It is not a write failure.
var ErrResourceObservationUnavailable = errors.New("provider cannot observe the resource's current state")

// ResourceState is the current stored state of one resource, read through the
// Provider at the observation boundary. Found is false when the resource does
// not exist.
type ResourceState struct {
	Version uint64
	Payload []byte
	Found   bool
}

// ResourceReader is an optional Provider capability: it reads the resource's
// current state after a Commit has completed. It deliberately does not take a
// Grant: the observation is of an already-authorized committed write and the
// result is only compared by the caller, never returned to it.
//
// A Provider that does not implement ResourceReader cannot claim resource
// observation. This separates "the receipt exists" (Lookup) from "the stored
// value matches the committed value" (ReadResource).
type ResourceReader interface {
	ReadResource(ctx context.Context, scope Scope) (ResourceState, error)
}

// ReadResource reads the current resource state through the Provider. It fails
// with ErrResourceObservationUnavailable when the Provider does not implement
// ResourceReader.
func (r Runner) ReadResource(ctx context.Context, scope Scope) (ResourceState, error) {
	if r.Provider == nil {
		return ResourceState{}, ErrMissingProvider
	}
	reader, ok := r.Provider.(ResourceReader)
	if !ok {
		return ResourceState{}, ErrResourceObservationUnavailable
	}
	return reader.ReadResource(ctx, scope)
}

// ResourceReader is implemented by the Provider that can read the resource.
