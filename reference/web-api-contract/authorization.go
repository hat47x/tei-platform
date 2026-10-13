package webapicontract

import (
	"context"
	"errors"

	"tei/internal/core"
	"tei/internal/security/legacyauthorization"
	"tei/internal/semantic/capabilityinvoke"
)

// Distinguish authorization denials from integration failures. The Web Host
// must not render backend failure details into an HTTP response.
var (
	ErrAuthorizationUnavailable = errors.New("authorization execution unavailable")
	ErrAuthorizationContract    = errors.New("authorization outcome contract violation")
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
	if trustedSubjectID == "" || action == "" || resource == "" {
		return ErrRejected
	}
	if invoker == nil || ctx == nil {
		return ErrAuthorizationUnavailable
	}
	outcome, err := invoker.Invoke(ctx, legacyauthorization.AuthorizationCapability,
		legacyauthorization.Input{
			SubjectID: trustedSubjectID,
			Action:    action,
			Resource:  resource,
		})
	if err != nil {
		return ErrAuthorizationUnavailable
	}
	if outcome.Capability != legacyauthorization.AuthorizationCapability ||
		outcome.Mode != capabilityinvoke.ResultOutcome {
		return ErrAuthorizationContract
	}
	if outcome.Outcome == "denied" {
		return ErrRejected
	}
	if outcome.Outcome != "allowed" {
		return ErrAuthorizationContract
	}
	grant, ok := outcome.Payload.(legacyauthorization.AuthorizationGrant)
	if !ok || grant.SubjectID != trustedSubjectID ||
		grant.Action != action || grant.Resource != resource {
		return ErrAuthorizationContract
	}
	return nil
}
