package multilingualintegration

import (
	"errors"
	"strings"
	"testing"

	"tei/internal/environment"
	"tei/internal/plugin"
	localization "tei/reference/message-localization"
	overlays "tei/reference/multilingual-overlays"
	resources "tei/reference/multilingual-resources"
)

// These tests compare separate reference projections. They do not simulate
// trusted Host grants, publisher authentication, or a production cutover.
func displayBundleForScene(t *testing.T, s orderScene) overlays.LocaleBundle {
	t.Helper()
	bundle, err := overlays.BuildLockedLocaleBundle(
		s.catalog, s.env, s.lock, s.graph, s.surfaces, s.packs, s.grants,
		nil, overlays.LocaleBundleRequest{
			Target:  s.lookup.Target,
			Keys:    []resources.Key{s.lookup.Key},
			Locales: []string{"en-US"},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return bundle
}

func TestCrossDomainPluginGenerationPinsDisplayButDoesNotAuthorizeRetiredGeneration(t *testing.T) {
	old := newOrderScene(t)
	bundle := displayBundleForScene(t, old)
	if err := overlays.CheckLocaleBundleAgainstLock(bundle, old.catalog, old.env, old.lock); err != nil {
		t.Fatalf("original pinned bundle was rejected: %v", err)
	}

	// The next definition is a second synthetic generation, not a live Host
	// cutover. The old artifact remains valid only under the OLD pinned lock.
	nextCatalog := plugin.Catalog{Plugins: append([]plugin.Definition(nil), old.catalog.Plugins...)}
	found := false
	for i := range nextCatalog.Plugins {
		if nextCatalog.Plugins[i].ID == old.lookup.Target.ID {
			nextCatalog.Plugins[i].Version = "1.3.0"
			found = true
		}
	}
	if !found {
		t.Fatal("ordinary plugin not found in reference catalog")
	}
	nextLock, err := plugin.BuildLock(nextCatalog, []environment.Environment{old.env})
	if err != nil {
		t.Fatal(err)
	}
	if err := overlays.CheckLocaleBundleAgainstLock(bundle, nextCatalog, old.env, nextLock); err == nil ||
		!strings.Contains(err.Error(), "environment-lock-mismatch") {
		t.Fatalf("old-generation bundle matched next-generation lock: %v", err)
	}

	// The request still identifies the previous plugin version; there must
	// be no implicit migration of old translation ownership into v1.3.0.
	_, err = overlays.BuildLockedLocaleBundle(
		nextCatalog, old.env, nextLock, old.graph, old.surfaces, old.packs,
		old.grants, nil, overlays.LocaleBundleRequest{
			Target:  old.lookup.Target,
			Keys:    []resources.Key{old.lookup.Key},
			Locales: []string{"en-US"},
		},
	)
	if err == nil || !strings.Contains(err.Error(), "unselected-target") {
		t.Fatalf("previous target was silently applied to next generation: %v", err)
	}
	if err := overlays.CheckLocaleBundleAgainstLock(bundle, old.catalog, old.env, old.lock); err != nil {
		t.Fatalf("old pinned inputs were invalidated by an independent fixture: %v", err)
	}
}

func TestCrossDomainBundleChecksAreNotCurrentGrantOrSourceAcceptance(t *testing.T) {
	s := newOrderScene(t)
	bundle := displayBundleForScene(t, s)

	// Removing a Grant rejects a fresh composition. A previously built
	// bundle, however, has no current-Grant parameter at verification time.
	s.grants = nil
	if _, err := s.display(); err == nil || !strings.Contains(err.Error(), "overlay-not-permitted") {
		t.Fatalf("fresh display unexpectedly accepted a missing grant: %v", err)
	}
	if err := overlays.CheckLocaleBundleAgainstLock(bundle, s.catalog, s.env, s.lock); err != nil {
		t.Fatalf("lock-only check unexpectedly observed a grant revocation: %v", err)
	}
	// A matching digest/lock is not permission to serve the cached bundle.
	// Only a separately trusted Host decision can determine current use.

	// A source label edit also rejects fresh composition, but the bundle
	// cannot compare itself with the current accepted source definition.
	s = newOrderScene(t)
	s.graph.Nodes[0].DisplayName = "受注を送信"
	if _, err := s.display(); err == nil || !strings.Contains(err.Error(), "stale-translation") {
		t.Fatalf("fresh display accepted a stale translation: %v", err)
	}
	if err := overlays.CheckLocaleBundleAgainstLock(bundle, s.catalog, s.env, s.lock); err != nil {
		t.Fatalf("lock-only check unexpectedly observed a source edit: %v", err)
	}
}

func TestCrossDomainMessageReviewRevocationDoesNotRewriteBusinessPublication(t *testing.T) {
	s := newOrderScene(t)
	s.message.Review = func(_ localization.ReviewInput) error { return errors.New("revoked") }
	if _, err := s.translatedMessage(); err == nil || !strings.Contains(err.Error(), "host-review-denied") {
		t.Fatalf("message was served after synthetic review denial: %v", err)
	}
	requireUnchangedOtherDomains(t, s, true, false, true)
}
