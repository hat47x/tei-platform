package webcomposition

import (
	"context"
	"errors"
	"testing"
	"time"

	"tei/internal/security/legacyauth"
)

type subjectUserStoreFunc func(context.Context, string) (legacyauth.User, bool, error)

func (f subjectUserStoreFunc) FindBySubject(ctx context.Context, subject string) (legacyauth.User, bool, error) {
	return f(ctx, subject)
}

func TestActiveLegacySubjectLinksRequireCurrentAccountEligibility(t *testing.T) {
	now := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	origin := ExternalSubject{ProviderID: "corporate", Issuer: "https://issuer.example.invalid/tenant1", Subject: "oidc-user"}
	link := SubjectLink{PrincipalID: "employee-1", SubjectID: "subject-order-editor"}
	valid := legacyauth.User{
		SubjectID: link.SubjectID,
		ValidFrom: now.Add(-time.Hour),
		ValidUntil: now.Add(time.Hour),
	}
	for _, tc := range []struct {
		name string
		user legacyauth.User
		found bool
		storeErr error
		linkFound bool
		linkErr error
		link SubjectLink
		wantFound bool
		wantError bool
		wantStoreCalls int
	}{
		{name: "active user", user: valid, found: true, linkFound: true, link: link, wantFound: true, wantStoreCalls: 1},
		{name: "disabled", user: func() legacyauth.User { u := valid; u.Disabled = true; return u }(), found: true, linkFound: true, link: link, wantStoreCalls: 1},
		{name: "locked", user: func() legacyauth.User { u := valid; u.LockedUntil = now.Add(time.Hour); return u }(), found: true, linkFound: true, link: link, wantStoreCalls: 1},
		{name: "not yet valid", user: func() legacyauth.User { u := valid; u.ValidFrom = now.Add(time.Second); return u }(), found: true, linkFound: true, link: link, wantStoreCalls: 1},
		{name: "expires at current instant", user: func() legacyauth.User { u := valid; u.ValidUntil = now; return u }(), found: true, linkFound: true, link: link, wantStoreCalls: 1},
		{name: "lock lifted at current instant", user: func() legacyauth.User { u := valid; u.LockedUntil = now; return u }(), found: true, linkFound: true, link: link, wantFound: true, wantStoreCalls: 1},
		{name: "linked user absent", linkFound: true, link: link, wantStoreCalls: 1},
		{name: "account store error", linkFound: true, link: link, storeErr: errors.New("backend down"), wantError: true, wantStoreCalls: 1},
		{name: "subject store identity mismatch", user: func() legacyauth.User { u := valid; u.SubjectID = "other"; return u }(), found: true, linkFound: true, link: link, wantError: true, wantStoreCalls: 1},
		{name: "incomplete mapping", user: valid, found: true, linkFound: true, link: SubjectLink{PrincipalID: "employee-1"}, wantError: true},
		{name: "mapping absent", wantStoreCalls: 0},
		{name: "mapping backend error", linkErr: errors.New("link down"), wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			storeCalls := 0
			lookup, err := NewActiveLegacySubjectLinks(
				func(_ context.Context, got ExternalSubject) (SubjectLink, bool, error) {
					if got != origin {
						t.Fatalf("wrong external source: %+v", got)
					}
					return tc.link, tc.linkFound, tc.linkErr
				},
				subjectUserStoreFunc(func(_ context.Context, id string) (legacyauth.User, bool, error) {
					storeCalls++
					if id != tc.link.SubjectID {
						t.Fatalf("wrong account lookup key %s", id)
					}
					return tc.user, tc.found, tc.storeErr
				}),
				func() time.Time { return now },
			)
			if err != nil {
				t.Fatal(err)
			}
			got, found, err := lookup(context.Background(), origin)
			if found != tc.wantFound || (err != nil) != tc.wantError || storeCalls != tc.wantStoreCalls {
				t.Fatalf("found=%v err=%v calls=%d want found=%v err=%v calls=%d", found, err, storeCalls, tc.wantFound, tc.wantError, tc.wantStoreCalls)
			}
			if tc.wantFound && got != link {
				t.Fatalf("active user link changed: %+v", got)
			}
			if !tc.wantFound && got != (SubjectLink{}) {
				t.Fatalf("rejected user link leaked: %+v", got)
			}
		})
	}
}

func TestActiveLegacySubjectLinksRequiresTrustedDependencies(t *testing.T) {
	lookup := SubjectLinkLookup(func(context.Context, ExternalSubject) (SubjectLink, bool, error) {
		return SubjectLink{}, false, nil
	})
	store := subjectUserStoreFunc(func(context.Context, string) (legacyauth.User, bool, error) {
		return legacyauth.User{}, false, nil
	})
	clock := func() time.Time { return time.Now() }
	for _, args := range []struct {
		links SubjectLinkLookup
		store subjectUserStoreFunc
		now func() time.Time
	}{
		{store: store, now: clock},
		{links: lookup, now: clock},
		{links: lookup, store: store},
	} {
		if _, err := NewActiveLegacySubjectLinks(args.links, args.store, args.now); err == nil {
			t.Fatal("incomplete subject eligibility guard was accepted")
		}
	}
}
