package webcomposition

import (
	"context"
	"errors"
	"fmt"
	"time"

	"tei/internal/security/legacyauthorization"
)

// NewActiveLegacySubjectLinks composes a trusted external subject mapping with
// the legacy account's current eligibility. A valid identity link and a role
// permission are not sufficient if the account has been disabled, expired, or
// locked. This reference policy uses the existing legacy store and does not
// duplicate the canonical security.authorize business permission rules.
//
// The caller owns the clock and the store's freshness: a cached fixture alone
// is not evidence of live revocation checks in a production deployment.
func NewActiveLegacySubjectLinks(
	lookup SubjectLinkLookup,
	store legacyauthorization.SubjectStore,
	now func() time.Time,
) (SubjectLinkLookup, error) {
	if lookup == nil || store == nil || now == nil {
		return nil, errors.New("active subject link requires mapping, store, and clock")
	}
	return func(ctx context.Context, source ExternalSubject) (SubjectLink, bool, error) {
		link, found, err := lookup(ctx, source)
		if err != nil {
			return SubjectLink{}, false, fmt.Errorf("subject link lookup failed: %w", err)
		}
		if !found {
			return SubjectLink{}, false, nil
		}
		if link.PrincipalID == "" || link.SubjectID == "" {
			return SubjectLink{}, false, errors.New("subject link has incomplete identities")
		}
		user, found, err := store.FindBySubject(ctx, link.SubjectID)
		if err != nil {
			return SubjectLink{}, false, fmt.Errorf("linked account lookup failed: %w", err)
		}
		if !found {
			return SubjectLink{}, false, nil
		}
		if user.SubjectID != link.SubjectID {
			return SubjectLink{}, false, errors.New("linked account identity mismatch")
		}
		current := now()
		if user.Disabled ||
			(!user.ValidFrom.IsZero() && current.Before(user.ValidFrom)) ||
			(!user.ValidUntil.IsZero() && !current.Before(user.ValidUntil)) ||
			(!user.LockedUntil.IsZero() && current.Before(user.LockedUntil)) {
			return SubjectLink{}, false, nil
		}
		return link, true, nil
	}, nil
}
