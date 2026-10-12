package multilingualoverlays

import (
	"reflect"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/modelref"
	"tei/internal/plugin"
	"tei/internal/subject"
	resources "tei/reference/multilingual-resources"
)

func fixture() (Input, resources.Key, plugin.Identity) {
	ordinary := plugin.Identity{ID: core.CanonicalID("std.order-entry"), Version: "1.2.0"}
	lang := plugin.Identity{ID: core.CanonicalID("std.lang-en"), Version: "1.0.0"}
	target := modelref.Key{Scope: "sales", Subject: subject.Subject{Kind: "function", ID: "sales.order.submit"}}
	key := resources.Key{Owner: string(ordinary.ID), Target: target, Field: "display-name"}
	input := Input{
		Resolved: plugin.ResolvedSet{
			Environment: core.CanonicalID("env.test"),
			Plugins:     []plugin.Identity{ordinary, lang},
		},
		Graph: modelref.Graph{Schema: "tei.modelref.v0", Nodes: []modelref.Node{
			{Key: target, DisplayName: "受注を確定"},
		}},
		Surfaces: []Surface{
			{Target: ordinary, Subject: target, Field: "display-name", SourceLocale: "ja-JP"},
		},
	}
	input.Packs = []Pack{
		{Provider: lang, Target: ordinary, Key: key, Locale: "en-US",
			Text: "Confirm order", BasedOn: Fingerprint("受注を確定"), Review: "confirmed"},
	}
	input.Grants = []Grant{{
		Environment: "env.test", Provider: lang, Target: ordinary, Key: key, Locale: "en-US",
	}}
	return input, key, ordinary
}

func TestTransparentLocalizationOfOrdinaryPlugin(t *testing.T) {
	in, key, original := fixture()
	before := in.Graph.Nodes[0]
	pluginsBefore := append([]plugin.Identity(nil), in.Resolved.Plugins...)
	projection, diag := Compose(in)
	if diag != "" {
		t.Fatal(diag)
	}
	for _, tc := range []struct{ locale, expected string }{
		{"ja-JP", "受注を確定"},
		{"en-US", "Confirm order"},
	} {
		got := projection.Resolve(Lookup{Target: original, Key: key, Locale: tc.locale})
		if got.Diagnostic != "" || got.Text != tc.expected || got.Locale != tc.locale {
			t.Fatalf("%s: %+v", tc.locale, got)
		}
	}
	if in.Graph.Nodes[0] != before || !reflect.DeepEqual(in.Resolved.Plugins, pluginsBefore) {
		t.Fatal("ordinary plugin or model was mutated")
	}
	if len(projection.entries) != 1 || projection.entries[0].Key.Target != key.Target {
		t.Fatal("semantic identity lost")
	}
}

func TestOverlayIsOptionalAndDoesNotChangeBasePresentation(t *testing.T) {
	in, key, original := fixture()
	in.Packs = nil
	in.Grants = nil
	projection, diag := Compose(in)
	if diag != "" {
		t.Fatal(diag)
	}
	if got := projection.Resolve(Lookup{Target: original, Key: key, Locale: "ja-JP"}); got.Text != "受注を確定" {
		t.Fatalf("nonlocalized plugin lost base display: %+v", got)
	}
	if got := projection.Resolve(Lookup{Target: original, Key: key, Locale: "en-US"}); got.Diagnostic != "missing-translation" {
		t.Fatalf("missing overlay implicitly translated: %+v", got)
	}
}

func TestSourceChangeInvalidatesOverlay(t *testing.T) {
	in, key, original := fixture()
	in.Graph.Nodes[0].DisplayName = "注文を確定"
	projection, diag := Compose(in)
	if diag != "" {
		t.Fatal(diag)
	}
	if got := projection.Resolve(Lookup{Target: original, Key: key, Locale: "en-US", Fallback: []string{"ja-JP"}, AllowSourceFallback: true}); got.Diagnostic != "stale-translation" {
		t.Fatalf("stale translation hidden by fallback: %+v", got)
	}
	if got := projection.Resolve(Lookup{Target: original, Key: key, Locale: "ja-JP"}); got.Text != "注文を確定" {
		t.Fatalf("base label did not follow model: %+v", got)
	}
	in.Packs[0].BasedOn = Fingerprint("注文を確定")
	projection, diag = Compose(in)
	if diag != "" || projection.Resolve(Lookup{Target: original, Key: key, Locale: "en-US"}).Diagnostic != "" {
		t.Fatalf("confirmed current translation rejected: %s", diag)
	}
}

func TestExactGrantProviderAndTargetVersionsAreRequired(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Input)
		want string
	}{
		{"missing grant", func(x *Input) { x.Grants = nil }, "overlay-not-permitted"},
		{"wrong environment", func(x *Input) { x.Grants[0].Environment = "env.other" }, "overlay-not-permitted"},
		{"unselected provider", func(x *Input) { x.Resolved.Plugins = x.Resolved.Plugins[:1] }, "unselected-locale-plugin"},
		{"different provider version", func(x *Input) { x.Packs[0].Provider.Version = "2.0.0" }, "unselected-locale-plugin"},
		{"changed ordinary plugin version", func(x *Input) { x.Packs[0].Target.Version = "1.3.0" }, "incompatible-plugin-version"},
		{"binding not selected", func(x *Input) { x.Surfaces[0].Target.Version = "2.0.0" }, "unselected-base-plugin"},
		{"unbound subject", func(x *Input) { x.Packs[0].Key.Target.Subject.ID = "unknown" }, "unbound-overlay-target"},
		{"non-display field", func(x *Input) { x.Surfaces[0].Field = "capability" }, "unsupported-display-field"},
		{"unauthorized native replacement", func(x *Input) { x.Packs[0].Locale = "ja-JP"; x.Grants[0].Locale = "ja-JP" }, "source-override-unsupported"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in, _, _ := fixture()
			tc.edit(&in)
			if _, diag := Compose(in); diag != tc.want {
				t.Fatalf("wanted %q, got %q", tc.want, diag)
			}
		})
	}
}

func TestConflictingOverlaysRequireExplicitCompleteOverride(t *testing.T) {
	in, key, ordinary := fixture()
	second := plugin.Identity{ID: core.CanonicalID("std.lang-en-special"), Version: "1.0.0"}
	in.Resolved.Plugins = append(in.Resolved.Plugins, second)
	newPack := in.Packs[0]
	newPack.Provider = second
	newPack.Text = "Finalize order"
	in.Packs = append(in.Packs, newPack)
	permit := in.Grants[0]
	permit.Provider = second
	in.Grants = append(in.Grants, permit)
	if _, diag := Compose(in); diag != "ambiguous-overlay" {
		t.Fatalf("ambiguous locale silently chosen: %s", diag)
	}
	in.Overrides = []Override{{Key: key, Locale: "en-US", Winner: second, Replaces: in.Packs[0].Provider}}
	projection, diag := Compose(in)
	if diag != "" {
		t.Fatal(diag)
	}
	got := projection.Resolve(Lookup{Target: ordinary, Key: key, Locale: "en-US"})
	if got.Text != "Finalize order" {
		t.Fatalf("explicit override ignored: %+v", got)
	}
	in.Packs[0], in.Packs[1] = in.Packs[1], in.Packs[0]
	in.Grants[0], in.Grants[1] = in.Grants[1], in.Grants[0]
	projection, diag = Compose(in)
	if diag != "" || projection.Resolve(Lookup{Target: ordinary, Key: key, Locale: "en-US"}).Text != "Finalize order" {
		t.Fatalf("input order affected winner: %s", diag)
	}
}

func TestCriticalResourceFailsClosedAndDisallowsFallback(t *testing.T) {
	in, key, original := fixture()
	in.Surfaces[0].Critical = true
	in.Packs[0].Review = "candidate"
	projection, diag := Compose(in)
	if diag != "" {
		t.Fatal(diag)
	}
	if got := projection.Resolve(Lookup{Target: original, Key: key, Locale: "en-US", Fallback: []string{"ja-JP"}, AllowSourceFallback: true}); got.Diagnostic != "unapproved-translation" {
		t.Fatalf("critical candidate should not fallback: %+v", got)
	}
	in.Packs = nil
	in.Grants = nil
	projection, diag = Compose(in)
	if diag != "" {
		t.Fatal(diag)
	}
	if got := projection.Resolve(Lookup{Target: original, Key: key, Locale: "en-US", Fallback: []string{"ja-JP"}, AllowSourceFallback: true}); got.Diagnostic != "missing-translation" {
		t.Fatalf("critical resource silently substituted: %+v", got)
	}
}

func TestDuplicateSurfacePackAndInvalidGraphFail(t *testing.T) {
	in, _, _ := fixture()
	in.Surfaces = append(in.Surfaces, in.Surfaces[0])
	if _, diag := Compose(in); diag != "duplicate-display-surface" {
		t.Fatalf("duplicate display surface: %s", diag)
	}
	in, _, _ = fixture()
	in.Packs = append(in.Packs, in.Packs[0])
	if _, diag := Compose(in); diag != "duplicate-overlay" {
		t.Fatalf("duplicate translation: %s", diag)
	}
	in, _, _ = fixture()
	in.Graph.Nodes = append(in.Graph.Nodes, in.Graph.Nodes[0])
	if _, diag := Compose(in); diag != "invalid-model-graph" {
		t.Fatalf("duplicated model node: %s", diag)
	}
}

func TestLookupOwnerAndFallbackAreExplicit(t *testing.T) {
	in, key, owner := fixture()
	projection, diag := Compose(in)
	if diag != "" {
		t.Fatal(diag)
	}
	other := owner
	other.Version = "2.0.0"
	if got := projection.Resolve(Lookup{Target: other, Key: key, Locale: "en-US"}); got.Diagnostic != "unbound-lookup-owner" {
		t.Fatalf("wrong owner was able to read: %+v", got)
	}
	if got := projection.Resolve(Lookup{Target: owner, Key: key, Locale: "en-GB"}); got.Diagnostic != "missing-translation" {
		t.Fatalf("implicit locale fallback: %+v", got)
	}
	if got := projection.Resolve(Lookup{Target: owner, Key: key, Locale: "en-GB", Fallback: []string{"en-US"}}); got.Text != "Confirm order" || !got.Fallback {
		t.Fatalf("explicit fallback rejected: %+v", got)
	}
}

func TestInvalidTranslationRemainsAnE1Diagnostic(t *testing.T) {
	in, key, owner := fixture()
	in.Packs[0].Review = "needs-review"
	projection, diag := Compose(in)
	if diag != "" {
		t.Fatal(diag)
	}
	got := projection.Resolve(Lookup{Target: owner, Key: key, Locale: "en-US"})
	if got.Diagnostic != "unapproved-translation" {
		t.Fatalf("review state lost: %+v", got)
	}
	in.Packs[0].Text = "  "
	if _, diag := Compose(in); !strings.Contains(diag, "invalid-resource-overlay") {
		t.Fatalf("invalid overlay accepted: %s", diag)
	}
}
