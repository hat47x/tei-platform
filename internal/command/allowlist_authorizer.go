package command

import (
	"context"
	"fmt"

	"tei/internal/core"
)

// AllowListAuthorizer permits only the listed capabilities. It is a reference
// authorizer for wiring and tests; it does not model subjects, resources, or
// permission grants.
type AllowListAuthorizer []core.CanonicalID

// Authorize rejects capabilities outside the allow list.
func (a AllowListAuthorizer) Authorize(_ context.Context, request Request) error {
	for _, allowed := range a {
		if allowed == request.Capability {
			return nil
		}
	}
	return fmt.Errorf("capability %s is not allowed by the reference authorizer", request.Capability)
}
