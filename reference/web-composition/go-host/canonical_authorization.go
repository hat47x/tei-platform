package webcomposition

import (
	"context"
	"errors"
	"fmt"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/security/legacyauthorization"
	"tei/internal/semantic/capabilityinvoke"
)

// ExternalSubject identifies a verified identity at its authenticating source.
// A display name, email, and the application PrincipalID are deliberately not
// keys for an account link.
type ExternalSubject struct {
	ProviderID string
	Issuer     string
	Subject    string
}

// SubjectLink ties the externally verified identity to both the session's
// application Principal and the existing business authorization subject.
type SubjectLink struct {
	PrincipalID string
	SubjectID   string
}

// SubjectLinkLookup is supplied by a trusted Host-owned adapter. It must
// validate the current link and account lifecycle, including revocation.
// It must not infer links from user-provided names or email.
// Unknown or revoked links return found=false; infrastructure failures return error.
type SubjectLinkLookup func(context.Context, ExternalSubject) (link SubjectLink, found bool, err error)

// NewCanonicalActionAuthorizer adapts the existing security.authorize
// Capability to NewHost's mandatory authorization callback. It does not
// implement a second permission evaluator, discover providers or change the
// legacy credential-based RunCommand entry.
func NewCanonicalActionAuthorizer(
	resolver driver.Resolver,
	applicationID, commandPatternID core.CanonicalID,
	links SubjectLinkLookup,
	adapters driver.BindingAdapterResolver,
) (func(context.Context, Principal, string) (bool, error), error) {
	if applicationID == "" || commandPatternID == "" || links == nil || adapters == nil {
		return nil, errors.New("canonical action authorization requires application, command, subject links and adapters")
	}
	// Fail at composition time if the Action or its authorization Binding is
	// not uniquely present in the application's accepted canonical model.
	if _, err := resolver.ResolveCommandBinding(applicationID, commandPatternID); err != nil {
		return nil, fmt.Errorf("command binding is unavailable: %w", err)
	}
	if _, err := resolver.ResolveCapabilityBinding(applicationID, legacyauthorization.AuthorizationCapability); err != nil {
		return nil, fmt.Errorf("authorization binding is unavailable: %w", err)
	}
	return func(ctx context.Context, principal Principal, requestedAction string) (bool, error) {
		if requestedAction != string(commandPatternID) {
			return false, errors.New("authorization action differs from configured command")
		}
		if principal.ProviderID == "" || principal.Issuer == "" || principal.Subject == "" || principal.PrincipalID == "" {
			return false, errors.New("session verifier returned incomplete identity")
		}
		link, found, err := links(ctx, ExternalSubject{
			ProviderID: principal.ProviderID,
			Issuer: principal.Issuer,
			Subject: principal.Subject,
		})
		if err != nil {
			return false, fmt.Errorf("trusted subject mapping unavailable: %w", err)
		}
		if !found {
			return false, nil // no inferred account, no fallback
		}
		if link.PrincipalID == "" || link.SubjectID == "" || link.PrincipalID != principal.PrincipalID {
			return false, errors.New("subject link does not match verified application principal")
		}
		invocation, err := resolver.InvokeCapability(
			ctx,
			applicationID,
			legacyauthorization.AuthorizationCapability,
			adapters,
			legacyauthorization.Input{
				SubjectID: link.SubjectID,
				Action:    "execute",
				Resource:  string(commandPatternID),
			},
		)
		if err != nil {
			return false, err // Execution Error, never a business denial
		}
		if invocation.Result.Mode != capabilityinvoke.ResultOutcome {
			return false, errors.New("authorization Capability returned an unexpected result mode")
		}
		switch invocation.Result.Outcome {
		case "allowed":
			grant, ok := invocation.Result.Payload.(legacyauthorization.AuthorizationGrant)
			if !ok || grant.SubjectID != link.SubjectID || grant.Action != "execute" || grant.Resource != string(commandPatternID) {
				return false, errors.New("authorization Capability returned a mismatched grant")
			}
			return true, nil
		case "denied":
			return false, nil
		default:
			return false, fmt.Errorf("authorization Capability returned an unexpected outcome %q", invocation.Result.Outcome)
		}
	}, nil
}
