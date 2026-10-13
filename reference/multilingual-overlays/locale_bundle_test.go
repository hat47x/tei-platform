package multilingualoverlays

import (
	"reflect"
	"strings"
	"testing"

	"tei/internal/environment"
	"tei/internal/modelref"
	"tei/internal/plugin"
	"tei/internal/subject"
	resources "tei/reference/multilingual-resources"
)

func orderLocaleBundle(t *testing.T) (LocaleBundle, LocaleBundleRequest) {
	t.Helper()
	c, env, lock, graph, surfaces, packs, grants, lookup, err := ReferenceOrderCase()
	if err != nil {
		t.Fatal(err)
	}
	req := LocaleBundleRequest{
		Target:  lookup.Target,
		Keys:    []resources.Key{lookup.Key},
		Locales: []string{"en-US", "ja-JP"},
	}
	bundle, err := BuildLockedLocaleBundle(c, env, lock, graph, surfaces, packs, grants, nil, req)
	if err != nil {
		t.Fatal(err)
	}
	return bundle, req
}

func TestO3BuildLocaleBundleSelectsOnlyRequestedLocales(t *testing.T) {
	b, req := orderLocaleBundle(t)
	if b.Kind != LocaleBundleKind || len(b.Displays) != 2 ||
		b.Locales[0] != "en-US" || b.Locales[1] != "ja-JP" ||
		b.Displays[0].Text != "Confirm order" || b.Displays[0].FromSource ||
		b.Displays[1].Text != "受注を確定" || !b.Displays[1].FromSource ||
		b.Displays[0].Resource != req.Keys[0] || b.Displays[1].Resource != req.Keys[0] {
		t.Fatalf("incorrect locale projection: %+v", b)
	}
	if err := CheckLocaleBundleIntegrity(b); err != nil {
		t.Fatalf("generated bundle has invalid integrity: %v", err)
	}
	c, env, lock, graph, surfaces, packs, grants, _, err := ReferenceOrderCase()
	if err != nil {
		t.Fatal(err)
	}
	req.Locales = []string{"ja-JP"}
	sourceOnly, err := BuildLockedLocaleBundle(c, env, lock, graph, surfaces, nil, nil, nil, req)
	if err != nil || len(sourceOnly.Displays) != 1 || sourceOnly.Displays[0].Text != "受注を確定" {
		t.Fatalf("ordinary plugin should still work without locale pack: %+v: %v", sourceOnly, err)
	}
	if strings.Contains(sourceOnly.Displays[0].Text, "Confirm") {
		t.Fatal("unexpected translation bundled into source-only locale")
	}
}

func TestO3BundleIsIndependentOfInputOrder(t *testing.T) {
	c, env, lock, graph, surfaces, packs, grants, lookup, err := ReferenceOrderCase()
	if err != nil {
		t.Fatal(err)
	}
	// A second ordinary model field, without changing the plugin definition.
	second := modelref.Key{Scope: "sales", Subject: subject.Subject{Kind: "function", ID: "sales.order.cancel"}}
	graph.Nodes = append(graph.Nodes, modelref.Node{Key: second, DisplayName: "受注を取消"})
	surfaces = append(surfaces, Surface{Target: lookup.Target, Subject: second, Field: "display-name", SourceLocale: "ja-JP"})
	other := resources.Key{Owner: string(lookup.Target.ID), Target: second, Field: "display-name"}
	packs = append(packs, Pack{Provider: packs[0].Provider, Target: lookup.Target, Key: other,
		Locale: "en-US", Text: "Cancel order", BasedOn: Fingerprint("受注を取消"), Review: "confirmed"})
	grants = append(grants, Grant{Environment: string(env.ID), Provider: grants[0].Provider,
		Target: lookup.Target, Key: other, Locale: "en-US"})
	a, err := BuildLockedLocaleBundle(c, env, lock, graph, surfaces, packs, grants, nil, LocaleBundleRequest{
		Target: lookup.Target, Keys: []resources.Key{lookup.Key, other},
		Locales: []string{"ja-JP", "en-US"},
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := BuildLockedLocaleBundle(c, env, lock, graph,
		[]Surface{surfaces[1], surfaces[0]}, []Pack{packs[1], packs[0]},
		[]Grant{grants[1], grants[0]}, nil,
		LocaleBundleRequest{Target: lookup.Target, Keys: []resources.Key{other, lookup.Key},
			Locales: []string{"en-US", "ja-JP"}})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a, b) || a.Digest != b.Digest ||
		len(a.Displays) != 4 || a.Displays[0].Text != "Cancel order" ||
		a.Displays[1].Text != "Confirm order" ||
		a.Displays[2].Text != "受注を取消" ||
		a.Displays[3].Text != "受注を確定" {
		t.Fatalf("locale bundle depends on input order: a=%+v b=%+v", a, b)
	}
	if err := CheckLocaleBundleIntegrity(a); err != nil {
		t.Fatal(err)
	}
}

func TestO3BundleFailsClosedBeforeDistribution(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*plugin.Catalog, *environment.Environment, *plugin.Lock, *modelref.Graph, *[]Surface, *[]Pack, *[]Grant, *LocaleBundleRequest)
		want string
	}{
		{"missing translated locale", func(_ *plugin.Catalog, _ *environment.Environment, _ *plugin.Lock, _ *modelref.Graph, _ *[]Surface, _ *[]Pack, _ *[]Grant, r *LocaleBundleRequest) {
			r.Locales = []string{"fr-FR"}
		}, "missing-translation"},
		{"stale translation", func(_ *plugin.Catalog, _ *environment.Environment, _ *plugin.Lock, g *modelref.Graph, _ *[]Surface, _ *[]Pack, _ *[]Grant, _ *LocaleBundleRequest) {
			g.Nodes[0].DisplayName = "受注を送信"
		}, "stale-translation"},
		{"unreviewed translation", func(_ *plugin.Catalog, _ *environment.Environment, _ *plugin.Lock, _ *modelref.Graph, _ *[]Surface, p *[]Pack, _ *[]Grant, _ *LocaleBundleRequest) {
			(*p)[0].Review = "candidate"
		}, "unapproved-translation"},
		{"ungranted translation", func(_ *plugin.Catalog, _ *environment.Environment, _ *plugin.Lock, _ *modelref.Graph, _ *[]Surface, _ *[]Pack, g *[]Grant, _ *LocaleBundleRequest) {
			*g = nil
		}, "overlay-not-permitted"},
		{"duplicate locale", func(_ *plugin.Catalog, _ *environment.Environment, _ *plugin.Lock, _ *modelref.Graph, _ *[]Surface, _ *[]Pack, _ *[]Grant, r *LocaleBundleRequest) {
			r.Locales = []string{"en-US", "en-US"}
		}, "duplicate-or-empty-locale"},
		{"duplicate key", func(_ *plugin.Catalog, _ *environment.Environment, _ *plugin.Lock, _ *modelref.Graph, _ *[]Surface, _ *[]Pack, _ *[]Grant, r *LocaleBundleRequest) {
			r.Keys = append(r.Keys, r.Keys[0])
		}, "duplicate-resource"},
		{"foreign key owner", func(_ *plugin.Catalog, _ *environment.Environment, _ *plugin.Lock, _ *modelref.Graph, _ *[]Surface, _ *[]Pack, _ *[]Grant, r *LocaleBundleRequest) {
			r.Keys[0].Owner = "std.untrusted"
		}, "foreign-resource-owner"},
		{"unselected plugin", func(_ *plugin.Catalog, e *environment.Environment, l *plugin.Lock, _ *modelref.Graph, _ *[]Surface, _ *[]Pack, _ *[]Grant, r *LocaleBundleRequest) {
			r.Target = plugin.Identity{ID: "std.unselected", Version: "1.0.0"}
			_ = e
			_ = l
		}, "unselected-target"},
		{"definition mutated with same version", func(c *plugin.Catalog, _ *environment.Environment, _ *plugin.Lock, _ *modelref.Graph, _ *[]Surface, _ *[]Pack, _ *[]Grant, _ *LocaleBundleRequest) {
			c.Plugins[0].Runtime.Kind = plugin.RuntimeProcess
		}, "plugin-lock"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, env, lock, graph, surfaces, packs, grants, lookup, err := ReferenceOrderCase()
			if err != nil {
				t.Fatal(err)
			}
			r := LocaleBundleRequest{Target: lookup.Target, Keys: []resources.Key{lookup.Key},
				Locales: []string{"ja-JP", "en-US"}}
			tc.edit(&c, &env, &lock, &graph, &surfaces, &packs, &grants, &r)
			b, err := BuildLockedLocaleBundle(c, env, lock, graph, surfaces, packs, grants, nil, r)
			if err == nil || !strings.Contains(err.Error(), tc.want) || b.Kind != "" {
				t.Fatalf("expected %s: bundle=%+v err=%v", tc.want, b, err)
			}
		})
	}
}

func TestO3BundleDigestDetectsMutationButIsNotAuthority(t *testing.T) {
	b, _ := orderLocaleBundle(t)
	b.Displays[0].Text = "Forged label"
	if err := CheckLocaleBundleIntegrity(b); err == nil || !strings.Contains(err.Error(), "digest-mismatch") {
		t.Fatalf("modified text accepted: %v", err)
	}
	// An attacker can recompute a self-contained hash. Hence passing this
	// check cannot replace authenticated Host delivery or accepted grants.
	b.Digest, _ = localeBundleDigest(b)
	if err := CheckLocaleBundleIntegrity(b); err != nil {
		t.Fatal(err)
	}
	b.Locales = []string{"en-US", "en-US"}
	b.Digest, _ = localeBundleDigest(b)
	if err := CheckLocaleBundleIntegrity(b); err == nil || !strings.Contains(err.Error(), "invalid-locale-order") {
		t.Fatalf("duplicate locales accepted: %v", err)
	}
}

func TestO3BundleChecksCurrentEnvironmentLock(t *testing.T) {
	c, env, lock, graph, surfaces, packs, grants, req, err := ReferenceOrderCase()
	if err != nil {
		t.Fatal(err)
	}
	b, err := BuildLockedLocaleBundle(c, env, lock, graph, surfaces, packs, grants, nil,
		LocaleBundleRequest{Target: req.Target, Keys: []resources.Key{req.Key},
			Locales: []string{"en-US"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckLocaleBundleAgainstLock(b, c, env, lock); err != nil {
		t.Fatalf("current bundle rejected: %v", err)
	}
	b.LockFingerprint = strings.Repeat("0", 64)
	for i := range b.Displays {
		b.Displays[i].LockFingerprint = b.LockFingerprint
	}
	b.Digest, _ = localeBundleDigest(b)
	if err := CheckLocaleBundleAgainstLock(b, c, env, lock); err == nil ||
		!strings.Contains(err.Error(), "environment-lock-mismatch") {
		t.Fatalf("foreign lock accepted with recomputed self-digest: %v", err)
	}
	b, _ = orderLocaleBundle(t)
	b.Target.Version = "9.9.9"
	for i := range b.Displays {
		b.Displays[i].Target.Version = b.Target.Version
	}
	b.Digest, _ = localeBundleDigest(b)
	if err := CheckLocaleBundleAgainstLock(b, c, env, lock); err == nil ||
		!strings.Contains(err.Error(), "target-not-in-current-lock") {
		t.Fatalf("unselected target version accepted: %v", err)
	}
	b, _ = orderLocaleBundle(t)
	c.Plugins[0].Runtime.Kind = plugin.RuntimeProcess
	if err := CheckLocaleBundleAgainstLock(b, c, env, lock); err == nil ||
		!strings.Contains(err.Error(), "plugin-lock") {
		t.Fatalf("modified plugin definition accepted: %v", err)
	}
}
