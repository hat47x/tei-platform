package webapicontract

import (
	"context"

	"tei/internal/core"
	"tei/internal/security/legacyauthorization"
	"tei/internal/semantic/capabilityinvoke"
)

// AuthorizationInvoker is the existing security.authorize Capability contract.
// It is intentionally not a Web-specific role or user store.
type AuthorizationInvoker interface {
	Invoke(context.Context, core.CanonicalID, any) (capabilityinvoke.BusinessResult, error)
}

// CheckExistingAuthorization checks a trusted server-side subject and exact
// operation/resource pair using the existing TEI authorization adapter.
// It does not invent field-level grants, prove row-scope enforcement or
// authorize the eventual transaction commit.
func CheckExistingAuthorization(
	ctx context.Context,
	invoker AuthorizationInvoker,
	trustedSubjectID, action, resource string,
) error {
	if invoker == nil || trustedSubjectID == "" || action == "" || resource == "" {
		return ErrRejected
	}
	outcome, err := invoker.Invoke(ctx, legacyauthorization.AuthorizationCapability,
		legacyauthorization.Input{
			SubjectID: trustedSubjectID,
			Action:    action,
			Resource:  resource,
		})
	if err != nil {
		return ErrRejected
	}
	if outcome.Capability != legacyauthorization.AuthorizationCapability ||
		outcome.Mode != capabilityinvoke.ResultOutcome ||
		outcome.Outcome != "allowed" {
		return ErrRejected
	}
	grant, ok := outcome.Payload.(legacyauthorization.AuthorizationGrant)
	if !ok || grant.SubjectID != trustedSubjectID ||
		grant.Action != action || grant.Resource != resource {
		return ErrRejected
	}
	return nil
}
