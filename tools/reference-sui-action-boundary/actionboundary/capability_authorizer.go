package actionboundary

import (
	"context"
	"fmt"

	"tei/internal/security/legacyauthorization"
	"tei/internal/semantic"
	"tei/internal/semantic/capabilityinvoke"
)

// SubjectResolver obtains an authenticated subject from trusted runtime context.
// Implementations must not accept a subject ID supplied in the UI action payload.
type SubjectResolver interface {
	Subject(context.Context) (string, error)
}

// CapabilityAuthorizer invokes the existing authorization capability adapter.
// This reference bridge still needs a real session-backed SubjectResolver and
// Application-owned runtime binding before it is production authority.
type CapabilityAuthorizer struct {
	Subjects   SubjectResolver
	Capability semantic.Capability
	Adapter    capabilityinvoke.Adapter
}

func (a CapabilityAuthorizer) Authorize(ctx context.Context, appID, docID, actionID string) error {
	if a.Subjects == nil || a.Adapter == nil {
		return fmt.Errorf("%w: runtime authorization dependencies missing", ErrDenied)
	}
	if a.Capability.ID != legacyauthorization.AuthorizationCapability {
		return fmt.Errorf("%w: unexpected authorization capability", ErrDenied)
	}
	subject, err := a.Subjects.Subject(ctx)
	if err != nil || subject == "" {
		return fmt.Errorf("%w: authenticated subject unavailable", ErrDenied)
	}
	result, err := capabilityinvoke.Invoke(ctx, a.Capability, a.Adapter, legacyauthorization.Input{
		SubjectID: subject,
		Action:    "execute",
		Resource:  appID + "/" + docID + "/" + actionID,
	})
	if err != nil {
		return fmt.Errorf("%w: authorization invocation: %v", ErrDenied, err)
	}
	if result.Mode != capabilityinvoke.ResultOutcome || result.Outcome != "allowed" {
		return fmt.Errorf("%w: authorization outcome %s", ErrDenied, result.Outcome)
	}
	grant, ok := result.Payload.(legacyauthorization.AuthorizationGrant)
	if !ok || grant.SubjectID != subject || grant.Action != "execute" || grant.Resource != appID+"/"+docID+"/"+actionID {
		return fmt.Errorf("%w: authorization grant mismatch", ErrDenied)
	}
	return nil
}
