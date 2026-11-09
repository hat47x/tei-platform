package multilingualresources

import (
	"reflect"
	"testing"

	"tei/internal/modelref"
	"tei/internal/subject"
)

func example() (modelref.Graph, Entry, Request) {
	id := Key{Owner: "order-entry", Target: modelref.Key{Scope: "sales", Subject: subject.Subject{Kind: "function", ID: "sales.order.submit"}}, Field: "action-label"}
	graph := modelref.Graph{Schema: "tei.modelref.v0", Nodes: []modelref.Node{{Key: id.Target}}}
	entry := Entry{Key: id, Source: Source{Locale: "ja-JP", Text: "受注を確定", Revision: "source-12"}, Translations: []Translation{{Locale: "en-US", Text: "Confirm order", BasedOn: "source-12", Review: "confirmed"}, {Locale: "fr-FR", Text: "Confirmer la commande", BasedOn: "source-12", Review: "confirmed"}}}
	return graph, entry, Request{Key: id, Locale: "en-US"}
}

func TestDisplaySelectionKeepsIdentityAndSource(t *testing.T) {
	g, e, r := example()
	before := e.Key
	got := Resolve(g, []Entry{e}, r)
	if got.Diagnostic != "" || got.Text != "Confirm order" || got.Locale != "en-US" || got.Fallback || got.FromSource {
		t.Fatalf("wrong English display: %+v", got)
	}
	r.Locale = "ja-JP"
	got = Resolve(g, []Entry{e}, r)
	if got.Diagnostic != "" || got.Text != "受注を確定" || !got.FromSource {
		t.Fatalf("wrong original: %+v", got)
	}
	if !reflect.DeepEqual(e.Key, before) || e.Key.Target != g.Nodes[0].Key {
		t.Fatal("display resolution changed the semantic identity")
	}
}

func TestFallbackRequiresExplicitPreference(t *testing.T) {
	g, e, r := example()
	r.Locale = "en-GB"
	if got := Resolve(g, []Entry{e}, r); got.Diagnostic != "missing-translation" {
		t.Fatalf("unexpected implicit fallback: %+v", got)
	}
	r.Fallback = []string{"en-US"}
	if got := Resolve(g, []Entry{e}, r); got.Diagnostic != "" || got.Text != "Confirm order" || !got.Fallback {
		t.Fatalf("explicit fallback failed: %+v", got)
	}
	r.Fallback = nil
	r.AllowSourceFallback = true
	if got := Resolve(g, []Entry{e}, r); got.Diagnostic != "" || got.Locale != "ja-JP" || !got.FromSource || !got.Fallback {
		t.Fatalf("explicit source fallback failed: %+v", got)
	}
}

func TestStaleOrUnapprovedDoesNotFallThrough(t *testing.T) {
	for _, tc := range []struct{ name, basedOn, review, want string }{
		{"stale", "source-11", "confirmed", "stale-translation"},
		{"candidate", "source-12", "candidate", "unapproved-translation"},
		{"needs review", "source-12", "needs-review", "unapproved-translation"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, e, r := example()
			e.Translations[0].BasedOn = tc.basedOn
			e.Translations[0].Review = tc.review
			r.Fallback = []string{"fr-FR"}
			r.AllowSourceFallback = true
			if got := Resolve(g, []Entry{e}, r); got.Diagnostic != tc.want || got.Text != "" {
				t.Fatalf("must not silently fall through: %+v", got)
			}
		})
	}
}

func TestCriticalTextNeverFallsBack(t *testing.T) {
	g, e, r := example()
	e.Critical = true
	r.Locale = "en-GB"
	r.Fallback = []string{"en-US"}
	r.AllowSourceFallback = true
	if got := Resolve(g, []Entry{e}, r); got.Diagnostic != "missing-translation" {
		t.Fatalf("critical message fell back: %+v", got)
	}
	r.Locale = "en-US"
	r.Fallback = nil
	e.Translations[0].Review = "candidate"
	if got := Resolve(g, []Entry{e}, r); got.Diagnostic != "unapproved-translation" {
		t.Fatalf("critical draft displayed: %+v", got)
	}
	r.Locale = "ja-JP"
	if got := Resolve(g, []Entry{e}, r); got.Diagnostic != "" || !got.FromSource {
		t.Fatalf("original should remain usable: %+v", got)
	}
}

func TestDuplicateOrMalformedEntriesFailClosed(t *testing.T) {
	g, e, r := example()
	cases := []struct {
		name    string
		entries []Entry
		code    string
	}{
		{"duplicate owner", []Entry{e, e}, "duplicate-resource"},
		{"invalid source locale", func() []Entry { v := e; v.Source.Locale = "EN_us"; return []Entry{v} }(), "invalid-source"},
		{"duplicate translation locale", func() []Entry {
			v := e
			v.Translations = append(append([]Translation{}, e.Translations...), e.Translations[0])
			return []Entry{v}
		}(), "duplicate-language"},
		{"invalid translation status", func() []Entry {
			v := e
			v.Translations = append([]Translation(nil), e.Translations...)
			v.Translations[0].Review = "accepted"
			return []Entry{v}
		}(), "invalid-translation"},
		{"duplicate source language", func() []Entry {
			v := e
			v.Translations = append([]Translation(nil), e.Translations...)
			v.Translations[0].Locale = "ja-JP"
			return []Entry{v}
		}(), "duplicate-language"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ds := Validate(tc.entries)
			found := false
			for _, d := range ds {
				if d.Code == tc.code {
					found = true
				}
			}
			if !found {
				t.Fatalf("expected %s, got %+v", tc.code, ds)
			}
			if got := Resolve(g, tc.entries, r); got.Diagnostic != "invalid-catalog" {
				t.Fatalf("invalid catalog accepted: %+v", got)
			}
		})
	}
}

func TestModelBindingAndRequestConstraints(t *testing.T) {
	g, e, r := example()
	if got := Resolve(modelref.Graph{Schema: g.Schema}, []Entry{e}, r); got.Diagnostic != "unbound-subject" {
		t.Fatalf("missing semantic target accepted: %+v", got)
	}
	g.Schema = "unknown"
	if got := Resolve(g, []Entry{e}, r); got.Diagnostic != "unsupported-model" {
		t.Fatalf("unknown graph accepted: %+v", got)
	}
	g.Schema = "tei.modelref.v0"
	r.Key.Field = "different"
	if got := Resolve(g, []Entry{e}, r); got.Diagnostic != "missing-resource" {
		t.Fatalf("different display purpose reused: %+v", got)
	}
	r.Key = e.Key
	r.Locale = "en-US"
	r.Fallback = []string{"en-US"}
	if got := Resolve(g, []Entry{e}, r); got.Diagnostic != "invalid-request" {
		t.Fatalf("ambiguous fallback accepted: %+v", got)
	}
	r.Locale = "EN_us"
	r.Fallback = nil
	if got := Resolve(g, []Entry{e}, r); got.Diagnostic != "invalid-request" {
		t.Fatalf("invalid locale accepted: %+v", got)
	}
}

func TestUnrelatedEntriesDoNotOverride(t *testing.T) {
	g, e, r := example()
	other := e
	other.Key.Field = "notice"
	other.Source.Text = "保存"
	entries := []Entry{other, e}
	got := Resolve(g, entries, r)
	if got.Text != "Confirm order" || got.Diagnostic != "" {
		t.Fatalf("resource order changed resolution: %+v", got)
	}
	entries[0], entries[1] = entries[1], entries[0]
	got = Resolve(g, entries, r)
	if got.Text != "Confirm order" || got.Diagnostic != "" {
		t.Fatalf("reordered resources changed result: %+v", got)
	}
}

func TestSourceRevisionChangeInvalidatesPreviouslyConfirmedTranslation(t *testing.T) {
	g, e, r := example()
	if got := Resolve(g, []Entry{e}, r); got.Diagnostic != "" {
		t.Fatalf("expected confirmed baseline: %+v", got)
	}
	e.Source.Text = "受注を正式に確定"
	e.Source.Revision = "source-13"
	if got := Resolve(g, []Entry{e}, r); got.Diagnostic != "stale-translation" || got.Text != "" {
		t.Fatalf("revised source did not invalidate translation: %+v", got)
	}
	r.Locale = "ja-JP"
	if got := Resolve(g, []Entry{e}, r); got.Diagnostic != "" || got.Text != "受注を正式に確定" {
		t.Fatalf("new source text unavailable: %+v", got)
	}
}

func TestDifferentPluginOwnersRemainIndependent(t *testing.T) {
	g, e, r := example()
	other := e
	other.Key.Owner = "different-plugin"
	other.Translations = append([]Translation(nil), e.Translations...)
	other.Translations[0].Text = "Submit purchase"
	entries := []Entry{other, e}
	if got := Validate(entries); len(got) != 0 {
		t.Fatalf("two owners wrongly collide: %+v", got)
	}
	if got := Resolve(g, entries, r); got.Diagnostic != "" || got.Text != "Confirm order" {
		t.Fatalf("wrong owner selected: %+v", got)
	}
	r.Key.Owner = "different-plugin"
	if got := Resolve(g, entries, r); got.Diagnostic != "" || got.Text != "Submit purchase" {
		t.Fatalf("independent owner not selected: %+v", got)
	}
}
