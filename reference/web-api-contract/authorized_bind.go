package webapicontract

import (
	"context"
	"net/url"
)

// AuthorizeBrowseRead binds a trusted, already-resolved Application Browse
// Pattern to the existing security.authorize operation contract, then derives
// a bounded read plan. Authorization uses the canonical Pattern identity,
// never a client-supplied resource or OData query option.
//
// The caller MUST obtain publication, per-field decisions and TrustedReadContext
// from trusted TEI services. A successful return is NOT permission to execute
// an arbitrary SQL Query: the real Provider must independently enforce row
// scope, projection, filter, order, limit and current authorization.
func AuthorizeBrowseRead(
	ctx context.Context,
	authorization AuthorizationInvoker,
	publication PublishedBrowse,
	values url.Values,
	trusted TrustedReadContext,
) (ReadPlan, error) {
	if publication.ApplicationID == "" || publication.PatternID == "" ||
		publication.QueryID == "" || publication.DataID == "" ||
		trusted.PrincipalID == "" {
		return ReadPlan{}, ErrRejected
	}
	if err := CheckExistingAuthorization(ctx, authorization,
		trusted.PrincipalID, "browse", string(publication.PatternID)); err != nil {
		return ReadPlan{}, ErrRejected
	}
	return publication.BindRead(values, trusted)
}

// AuthorizeCommandIntent similarly checks the existing Command Pattern's
// execute permission and binds a synthetic Command API request to its resolved
// Capability identity. The CommandSpec is trusted; writable-field policy,
// revision, idempotency and Capability input type MUST still be verified by the
// actual execution authority. This function performs no writes.
func AuthorizeCommandIntent(
	ctx context.Context,
	authorization AuthorizationInvoker,
	publication PublishedCommand,
	request CommandRequest,
	spec CommandSpec,
	trustedPrincipalID string,
) (CommandIntent, error) {
	if publication.ApplicationID == "" || publication.PatternID == "" ||
		publication.CapabilityID == "" || trustedPrincipalID == "" ||
		spec.ActionID != string(publication.CapabilityID) {
		return CommandIntent{}, ErrRejected
	}
	if err := CheckExistingAuthorization(ctx, authorization,
		trustedPrincipalID, "execute", string(publication.PatternID)); err != nil {
		return CommandIntent{}, ErrRejected
	}
	return BindCommand(request, spec, trustedPrincipalID)
}
