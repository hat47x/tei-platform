// Package multilingualintegration contains cross-domain regression scenarios.
// This package is tests only: no new locale resolver, authorization or storage.
package multilingualintegration

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/modelref"
	"tei/internal/plugin"
	"tei/internal/subject"
	localization "tei/reference/message-localization"
	business "tei/reference/multilingual-business-data"
	overlays "tei/reference/multilingual-overlays"
	messages "tei/reference/standard-messages"
)

type orderScene struct {
	catalog  plugin.Catalog
	env      environment.Environment
	lock     plugin.Lock
	graph    modelref.Graph
	surfaces []overlays.Surface
	packs    []overlays.Pack
	grants   []overlays.Grant
	lookup   overlays.Lookup
	message  localization.Input
	snapshot business.Snapshot
	source   business.Key
	target   business.Key
}

func newOrderScene(t *testing.T) orderScene {
	t.Helper()
	c, env, lock, graph, surfaces, packs, grants, lookup, err := overlays.ReferenceOrderCase()
	if err != nil {
		t.Fatal(err)
	}

	standard := plugin.Identity{ID: "std.message", Version: "1.0.0"}
	owner := plugin.Identity{ID: "std.order_entry", Version: "1.2.0"}
	locale := plugin.Identity{ID: "std.localization_ja", Version: "1.0.0"}
	def := messages.Definition{
		Ref:          messages.Ref{Owner: owner, ID: core.CanonicalID("order.confirmed")},
		SourceLocale: "en-US", Text: "Order {orderId} confirmed",
		Format:       messages.FormatNamedV0,
		Parameters: []messages.Parameter{{Name: "orderId", Kind: messages.ParameterText}},
	}
	mc := plugin.Catalog{Plugins: []plugin.Definition{
		{ID: standard.ID, Version: standard.Version, Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}},
		{ID: owner.ID, Version: owner.Version, Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
			Requires: []plugin.Requirement{{Plugin: standard.ID, CompatibleMajor: 1}}},
		{ID: locale.ID, Version: locale.Version, Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
			Requires: []plugin.Requirement{{Plugin: standard.ID, CompatibleMajor: 1}}},
	}}
	me := environment.Environment{
		ID: "development",
		Plugins: []environment.ExtensionRequirement{
			{ID: owner.ID, CompatibleMajor: 1},
			{ID: locale.ID, CompatibleMajor: 1},
		},
	}
	ml, err := plugin.BuildLock(mc, []environment.Environment{me})
	if err != nil {
		t.Fatal(err)
	}
	app := core.CanonicalID("order_entry")
	message := localization.Input{
		Plugins: mc, Environment: me, Lock: ml,
		Messages: messages.Catalog{Definitions: []messages.Definition{def}},
		Stores:   []localization.StoreDeclaration{{Provider: locale, ID: "translations", Schema: localization.StoreSchema}},
		Bindings: []localization.ApplicationBinding{{
			Environment: me.ID, Application: app, Provider: locale, StoreID: "translations",
		}},
		Records: []localization.TranslationRecord{{
			Application: app, StoreID: "translations", Provider: locale, Target: owner,
			MessageID: def.Ref.ID, Locale: "ja-JP",
			Text:    "注文 {orderId} を確定しました",
			BasedOn: messages.Revision(def), Review: "confirmed",
		}},
		Grants: []localization.Grant{{
			Environment: me.ID, Application: app, Provider: locale,
			Target: owner, MessageID: def.Ref.ID, Locale: "ja-JP",
		}},
		Policy:  localization.Policy{Locale: "ja-JP"},
		Request: localization.Request{Application: app, Message: def.Ref},
		// Stub is NOT an actual Host acceptance, just a reference-test witness.
		Review: func(localization.ReviewInput) error { return nil },
	}

	field := modelref.Key{Scope: "sales", Subject: subject.Subject{Kind: "attribute", ID: "sales.order.note"}}
	source := business.Key{
		Tenant: "test-tenant", Record: "order:S-100", Field: field,
		Locale: "ja-JP", Edition: "sales",
	}
	target := source
	target.Locale = "en-US"
	sr := business.SourceRef{Key: source, Revision: "ja-1"}
	snapshot := business.Snapshot{
		Model: modelref.Graph{Schema: "tei.modelref.v0", Nodes: []modelref.Node{{Key: field}}},
		Revisions: []business.Revision{
			{Key: source, Revision: "ja-1", Text: "午前中に届けてください。", Origin: "original", Review: "approved"},
			{Key: target, Revision: "en-1", Text: "Please deliver in the morning.", Origin: "translation",
				BasedOn: &sr, Review: "approved"},
		},
		Publications: []business.Publication{
			{Key: source, Revision: "ja-1"},
			{Key: target, Revision: "en-1"},
		},
	}
	return orderScene{
		catalog: c, env: env, lock: lock, graph: graph,
		surfaces: surfaces, packs: packs, grants: grants, lookup: lookup,
		message: message, snapshot: snapshot, source: source, target: target,
	}
}

func (s orderScene) display() (overlays.DisplayManifest, error) {
	return overlays.BuildLockedDisplay(
		s.catalog, s.env, s.lock, s.graph, s.surfaces,
		s.packs, s.grants, nil, s.lookup,
	)
}

func (s orderScene) translatedMessage() (localization.Output, error) {
	return localization.Resolve(s.message)
}

func requireUnchangedOtherDomains(t *testing.T, s orderScene, wantDisplay, wantMessage, wantData bool) {
	t.Helper()
	if wantDisplay {
		got, err := s.display()
		if err != nil || got.Text != "Confirm order" {
			t.Fatalf("unrelated display changed: %+v %v", got, err)
		}
	}
	if wantMessage {
		got, err := s.translatedMessage()
		if err != nil || got.Resource.Text != "注文 {orderId} を確定しました" {
			t.Fatalf("unrelated message changed: %+v %v", got, err)
		}
	}
	if wantData {
		got := business.Select(s.snapshot, s.target)
		if got.Diagnostic != "" || got.Text != "Please deliver in the morning." {
			t.Fatalf("unrelated published business value changed: %+v", got)
		}
	}
}

// Three resources contribute to a single illustrative order scene, but they
// have distinct revision rules, identity scopes and publication authorities.
// No direct KouseiKanri <-> order_entry mapping is claimed by this fixture.
func TestCrossDomainSourceChangeImpactsOnlyItsOwnTranslation(t *testing.T) {
	t.Run("baseline", func(t *testing.T) {
		s := newOrderScene(t)
		requireUnchangedOtherDomains(t, s, true, true, true)
	})

	t.Run("display-label-change", func(t *testing.T) {
		s := newOrderScene(t)
		s.graph.Nodes[0].DisplayName = "受注処理を確定"
		if _, err := s.display(); err == nil || !strings.Contains(err.Error(), "stale-translation") {
			t.Fatalf("old label translation did not expire: %v", err)
		}
		requireUnchangedOtherDomains(t, s, false, true, true)
	})

	t.Run("message-wording-change", func(t *testing.T) {
		s := newOrderScene(t)
		s.message.Messages.Definitions[0].Text = "Confirmed order {orderId}"
		if _, err := s.translatedMessage(); err == nil || !strings.Contains(err.Error(), "stale-translation") {
			t.Fatalf("old message translation did not expire: %v", err)
		}
		requireUnchangedOtherDomains(t, s, true, false, true)
	})

	t.Run("message-placeholder-type-change", func(t *testing.T) {
		s := newOrderScene(t)
		s.message.Messages.Definitions[0].Parameters[0].Kind = messages.ParameterInteger
		if _, err := s.translatedMessage(); err == nil || !strings.Contains(err.Error(), "stale-translation") {
			t.Fatalf("old typed placeholder translation did not expire: %v", err)
		}
		requireUnchangedOtherDomains(t, s, true, false, true)
	})

	t.Run("business-original-new-publication", func(t *testing.T) {
		s := newOrderScene(t)
		s.snapshot.Revisions = append(s.snapshot.Revisions, business.Revision{
			Key: s.source, Revision: "ja-2", Text: "午後に届けてください。",
			Origin: "original", Review: "approved",
		})
		s.snapshot.Publications[0].Revision = "ja-2"
		if got := business.Select(s.snapshot, s.target); got.Diagnostic != "stale-translation" || got.Text != "" {
			t.Fatalf("old business translation remained published: %+v", got)
		}
		requireUnchangedOtherDomains(t, s, true, true, false)
	})
}

func TestCrossDomainCandidateAndIndependentEditionDoNotBecomeSharedApproval(t *testing.T) {
	s := newOrderScene(t)
	// Source-only candidates, Host grants and publication are distinct:
	// an unreviewed display Pack cannot turn into a live label.
	s.packs[0].Review = "candidate"
	if _, err := s.display(); err == nil || !strings.Contains(err.Error(), "unapproved-translation") {
		t.Fatalf("candidate display was treated as approved: %v", err)
	}
	s.message.Records[0].Review = "candidate"
	if _, err := s.translatedMessage(); err == nil || !strings.Contains(err.Error(), "unapproved-translation") {
		t.Fatalf("candidate message was treated as approved: %v", err)
	}
	// A new candidate business value is historical evidence, not publication.
	ref := business.SourceRef{Key: s.source, Revision: "ja-1"}
	s.snapshot.Revisions = append(s.snapshot.Revisions, business.Revision{
		Key: s.target, Revision: "en-2", Text: "Deliver tomorrow.",
		Origin: "translation", BasedOn: &ref, Review: "candidate",
	})
	if got := business.Select(s.snapshot, s.target); got.Diagnostic != "" || got.Revision != "en-1" {
		t.Fatalf("candidate automatically published: %+v", got)
	}
	// A separately authored English edition must NOT be invalidated by a
	// Japanese source change: business publication semantics differ from UI.
	s.snapshot.Revisions[1].Origin = "independent"
	s.snapshot.Revisions[1].BasedOn = nil
	s.snapshot.Revisions[1].Text = "Separately approved English terms."
	s.snapshot.Revisions = append(s.snapshot.Revisions, business.Revision{
		Key: s.source, Revision: "ja-2", Text: "変更後の日本語正文",
		Origin: "independent", Review: "approved",
	})
	s.snapshot.Publications[0].Revision = "ja-2"
	if got := business.Select(s.snapshot, s.target); got.Diagnostic != "" ||
		got.Text != "Separately approved English terms." {
		t.Fatalf("independent edition incorrectly expired: %+v", got)
	}
}
