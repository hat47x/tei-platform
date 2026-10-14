//go:build !tei_repair_only

package main

import (
	"context"
	"fmt"

	webcomposition "tei/reference/web-composition/go-host"
)

// RunVerifiedSessionCommand is a separate entry from the legacy credential
// route. It requires the authenticated Principal that NewHost inserted into
// the request context after its HTTP protections have passed.
// A plain context, JSON subject field, or request header cannot impersonate it.
func (r *referenceCommandScenarioRunner) RunVerifiedSessionCommand(
	ctx context.Context,
	links webcomposition.SubjectLinkLookup,
	from string,
	rowIndex int,
) (commandScenarioResult, error) {
	result := commandScenarioResult{
		ScenarioID:    "order-entry.submit-order-session-web-slice",
		ApplicationID: "order_entry",
		Stage:         "identity-link",
		AuthorityLimits: []string{
			"session-identity-is-verified-by-web-host",
			"subject-link-must-be-trusted-and-current",
			"business-authorization-uses-canonical-capability",
			"synthetic-command-state-is-not-external-production-state",
		},
		Unresolved: []string{
			"actual-idp-and-session-provider-not-connected",
			"actual-functional-evidence-not-yet-acquired",
			"actual-external-command-observation-not-acquired",
		},
	}
	principal, ok := webcomposition.PrincipalFromContext(ctx)
	if !ok {
		return result, fmt.Errorf("session command requires a Host-verified Principal")
	}
	if links == nil {
		return result, fmt.Errorf("session command requires a trusted subject linker")
	}
	if from == "" || rowIndex < 0 {
		return result, fmt.Errorf("session command requires from and non-negative row")
	}
	link, found, err := links(ctx, webcomposition.ExternalSubject{
		ProviderID: principal.ProviderID,
		Issuer:     principal.Issuer,
		Subject:    principal.Subject,
	})
	if err != nil {
		return result, fmt.Errorf("session command cannot resolve trusted subject link: %w", err)
	}
	if !found {
		result.Stage = "identity-link-denied"
		return result, nil
	}
	if link.PrincipalID == "" || link.SubjectID == "" || link.PrincipalID != principal.PrincipalID {
		return result, fmt.Errorf("session command subject link does not match verified Principal")
	}
	// No credential is injected and legacy authentication is not misreported
	// as invoked. Recheck the live identity link at the effect boundary, after
	// Read, Validation and canonical authorization have completed.
	effectGuard := func(ctx context.Context) (bool, error) {
		current, stillLinked, err := links(ctx, webcomposition.ExternalSubject{
			ProviderID: principal.ProviderID,
			Issuer:     principal.Issuer,
			Subject:    principal.Subject,
		})
		if err != nil {
			return false, err
		}
		if !stillLinked {
			return false, nil
		}
		if current.PrincipalID == "" || current.SubjectID == "" ||
			current.PrincipalID != principal.PrincipalID || current != link {
			return false, fmt.Errorf("effect-time subject link differs from verified session identity")
		}
		return true, nil
	}
	return r.runAuthenticatedSubjectCommandWithEffectGuard(ctx, link.SubjectID, from, rowIndex, result, effectGuard)
}
