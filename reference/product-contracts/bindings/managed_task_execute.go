package bindings

import (
	"context"
	"errors"

	"tei/internal/core"
	"tei/internal/semantic/capabilityinvoke"
)

// ManagedTaskExecute is not implemented; it never reports synthetic output as success.
type ManagedTaskExecute struct{}

func (ManagedTaskExecute) Invoke(_ context.Context, _ core.CanonicalID, _ any) (capabilityinvoke.BusinessResult, error) {
	return capabilityinvoke.BusinessResult{}, errors.New("managed task execution is not implemented")
}
