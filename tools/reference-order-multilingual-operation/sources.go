package ordermultilingualoperation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"tei/internal/core"
	"tei/internal/modelref"
	"tei/internal/plugin"
	localization "tei/reference/message-localization"
	overlays "tei/reference/multilingual-overlays"
	resources "tei/reference/multilingual-resources"
	messages "tei/reference/standard-messages"
)

// MessageSource is a source message without its owner: the owner is the
// unified owner plugin.
type MessageSource struct {
	ID           string               `json:"id"`
	SourceLocale string               `json:"sourceLocale"`
	Text         string               `json:"text"`
	Format       string               `json:"format"`
	Parameters   []messages.Parameter `json:"parameters,omitempty"`
}

// DisplaySource is the Japanese source label of a typed Workspace node. The
// workspace syntax only has ASCII identifiers (Symbol), so the label is
// declared here and bound to the Derived Index key.
type DisplaySource struct {
	Kind string `json:"kind"` // data or capability
	ID   string `json:"id"`
	Text string `json:"text"`
}

// SourceRelease is one release of the source declarations. A source revision
// is one change of this file; its digest is the release digest.
type SourceRelease struct {
	Release  string          `json:"release"`
	Messages []MessageSource `json:"messages"`
	Displays []DisplaySource `json:"displays"`
	Digest   string          `json:"-"`
}

// Translation is one translation a language plugin provides.
type Translation struct {
	Provider string            `json:"provider"`
	Kind     string            `json:"kind"` // message or display
	Target   string            `json:"target"`
	Locale   string            `json:"locale"`
	Release  string            `json:"release"`
	Text     string            `json:"text"`
	Plural   map[string]string `json:"plural,omitempty"`
}

func digestBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func digestJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return digestBytes(b)
}

// LoadRelease reads a source release; the digest covers the file bytes.
func LoadRelease(path string) (SourceRelease, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return SourceRelease{}, err
	}
	var r SourceRelease
	if err := json.Unmarshal(raw, &r); err != nil {
		return SourceRelease{}, fmt.Errorf("%s: %w", path, err)
	}
	r.Digest = digestBytes(raw)
	return r, nil
}

// LoadTranslations reads the translations the language plugins provide.
func LoadTranslations(path string) ([]Translation, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Translations []Translation `json:"translations"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	return doc.Translations, nil
}

// Definition converts a message source into the std.message definition of
// the unified owner.
func (u Unified) Definition(m MessageSource) messages.Definition {
	return messages.Definition{
		Ref:          messages.Ref{Owner: u.Owner, ID: core.CanonicalID(m.ID)},
		SourceLocale: m.SourceLocale, Text: m.Text, Format: m.Format,
		Parameters: append([]messages.Parameter(nil), m.Parameters...),
	}
}

// Catalog builds the std.message catalog of the release.
func (u Unified) Catalog(r SourceRelease) messages.Catalog {
	var c messages.Catalog
	for _, m := range r.Messages {
		c.Definitions = append(c.Definitions, u.Definition(m))
	}
	return c
}

// MessageRef is the reference of a message of the unified owner.
func (u Unified) MessageRef(id string) messages.Ref {
	return messages.Ref{Owner: u.Owner, ID: core.CanonicalID(id)}
}

// DisplayTarget is "kind:id", the matrix and translation form of a display.
func DisplayTarget(d DisplaySource) string { return d.Kind + ":" + d.ID }

// ParseDisplayTarget splits "kind:id".
func ParseDisplayTarget(t string) (string, string) {
	kind, id, _ := strings.Cut(t, ":")
	return kind, id
}

// ResourceKey is the overlay resource key of a display target.
func (u Unified) ResourceKey(target string) resources.Key {
	kind, id := ParseDisplayTarget(target)
	return resources.Key{Owner: string(u.Owner.ID), Target: u.SubjectKey(kind, id), Field: u.DisplayField}
}

// Graph builds the typed display graph of the release: only Derived Index
// nodes of the copy, labelled with the release's Japanese source text.
func (u Unified) Graph(ws Workspace, r SourceRelease) (modelref.Graph, error) {
	g := modelref.Graph{Schema: "tei.modelref.v0"}
	for _, d := range r.Displays {
		key := u.SubjectKey(d.Kind, d.ID)
		if !ws.HasNode(key) {
			return modelref.Graph{}, fmt.Errorf("display target %s is not in the Derived Index of the workspace", DisplayTarget(d))
		}
		g.Nodes = append(g.Nodes, modelref.Node{Key: key, DisplayName: d.Text})
	}
	return g, nil
}

// Surfaces are the environment side display surfaces of the owner.
func (u Unified) Surfaces(r SourceRelease) []overlays.Surface {
	var out []overlays.Surface
	for _, d := range r.Displays {
		out = append(out, overlays.Surface{Target: u.Owner, Subject: u.SubjectKey(d.Kind, d.ID),
			Field: u.DisplayField, SourceLocale: u.SourceLocale})
	}
	return out
}

// DisplayKeys are the resource keys of the release's displays, sorted.
func (u Unified) DisplayKeys(r SourceRelease) []resources.Key {
	var out []resources.Key
	for _, d := range r.Displays {
		out = append(out, u.ResourceKey(DisplayTarget(d)))
	}
	return out
}

func (r SourceRelease) message(id string) (MessageSource, bool) {
	for _, m := range r.Messages {
		if m.ID == id {
			return m, true
		}
	}
	return MessageSource{}, false
}

func (r SourceRelease) display(target string) (DisplaySource, bool) {
	for _, d := range r.Displays {
		if DisplayTarget(d) == target {
			return d, true
		}
	}
	return DisplaySource{}, false
}

// NativeRevision is the source revision each asset derives on its own: the
// definition-wide SHA-256 for std.message, the display text SHA-256 for the
// overlay. The two rules differ (PlatformGaps); the through record keeps both
// beside the release digest.
func (u Unified) NativeRevision(r SourceRelease, kind, target string) (string, error) {
	if kind == "message" {
		m, ok := r.message(target)
		if !ok {
			return "", fmt.Errorf("release %s has no message %s", r.Release, target)
		}
		return messages.Revision(u.Definition(m)), nil
	}
	d, ok := r.display(target)
	if !ok {
		return "", fmt.Errorf("release %s has no display %s", r.Release, target)
	}
	return overlays.Fingerprint(d.Text), nil
}

// Confirmed is the set of translations confirmed against source releases.
// Records of a superseded release stay in History; Current is what the
// language plugins serve.
type Confirmed struct {
	Current []ConfirmedTranslation `json:"current"`
	History []ConfirmedTranslation `json:"history"`
}

// ConfirmedTranslation is a translation with the asset native BasedOn of the
// release it was confirmed against.
type ConfirmedTranslation struct {
	Translation
	BasedOn string `json:"basedOn"`
}

// Confirm confirms the translations of a release against that release. A
// later confirmation of the same (provider, kind, target, locale) moves the
// earlier one into History; it is never deleted.
func (u Unified) Confirm(c Confirmed, r SourceRelease, all []Translation) (Confirmed, error) {
	out := Confirmed{History: append([]ConfirmedTranslation(nil), c.History...)}
	replaced := map[string]bool{}
	var fresh []ConfirmedTranslation
	for _, t := range all {
		if t.Release != r.Release {
			continue
		}
		kind := "message"
		if t.Kind == "display" {
			kind = "display"
		}
		based, err := u.NativeRevision(r, kind, t.Target)
		if err != nil {
			return Confirmed{}, err
		}
		fresh = append(fresh, ConfirmedTranslation{Translation: t, BasedOn: based})
		replaced[t.slot()] = true
	}
	for _, old := range c.Current {
		if replaced[old.slot()] {
			out.History = append(out.History, old)
		} else {
			out.Current = append(out.Current, old)
		}
	}
	out.Current = append(out.Current, fresh...)
	sort.Slice(out.Current, func(i, j int) bool { return out.Current[i].slot() < out.Current[j].slot() })
	return out, nil
}

func (t Translation) slot() string {
	return t.Kind + "|" + t.Target + "|" + t.Locale + "|" + t.Provider
}

// MessageRecords are the std.localization records of the confirmed message
// translations whose provider is in providers (nil = all).
func (u Unified) MessageRecords(c Confirmed, keep func(ConfirmedTranslation) bool) []localization.TranslationRecord {
	var out []localization.TranslationRecord
	for _, t := range c.Current {
		if t.Kind != "message" || (keep != nil && !keep(t)) {
			continue
		}
		out = append(out, localization.TranslationRecord{
			Application: u.Application, StoreID: u.Store, Provider: ProviderBy[t.Provider], Target: u.Owner,
			MessageID: core.CanonicalID(t.Target), Locale: t.Locale, Text: t.Text, BasedOn: t.BasedOn, Review: "confirmed",
		})
	}
	return out
}

// Packs are the overlay packs of the confirmed display translations.
func (u Unified) Packs(c Confirmed, keep func(ConfirmedTranslation) bool) []overlays.Pack {
	var out []overlays.Pack
	for _, t := range c.Current {
		if t.Kind != "display" || (keep != nil && !keep(t)) {
			continue
		}
		out = append(out, overlays.Pack{Provider: ProviderBy[t.Provider], Target: u.Owner, Key: u.ResourceKey(t.Target),
			Locale: t.Locale, Text: t.Text, BasedOn: t.BasedOn, Review: "confirmed"})
	}
	return out
}

// Grants is the host side grant set (both asset types) for one environment.
type Grants struct {
	Messages []localization.Grant `json:"messages"`
	Displays []overlays.Grant     `json:"displays"`
}

// GrantAll grants every confirmed translation in the unified environment.
func (u Unified) GrantAll(c Confirmed) Grants {
	var g Grants
	seen := map[string]bool{}
	for _, t := range c.Current {
		if seen[t.slot()] {
			continue
		}
		seen[t.slot()] = true
		p := ProviderBy[t.Provider]
		if t.Kind == "message" {
			g.Messages = append(g.Messages, localization.Grant{Environment: u.Environment, Application: u.Application,
				Provider: p, Target: u.Owner, MessageID: core.CanonicalID(t.Target), Locale: t.Locale})
		} else {
			g.Displays = append(g.Displays, overlays.Grant{Environment: string(u.Environment), Provider: p,
				Target: u.Owner, Key: u.ResourceKey(t.Target), Locale: t.Locale})
		}
	}
	return g
}

// Without returns the grants minus those of one locale.
func (g Grants) Without(locale string) Grants {
	var out Grants
	for _, m := range g.Messages {
		if m.Locale != locale {
			out.Messages = append(out.Messages, m)
		}
	}
	for _, d := range g.Displays {
		if d.Locale != locale {
			out.Displays = append(out.Displays, d)
		}
	}
	return out
}

// StoresAndBindings declares one translation store per language plugin and
// binds it to the unified application and environment.
func (u Unified) StoresAndBindings(providers []plugin.Identity) ([]localization.StoreDeclaration, []localization.ApplicationBinding) {
	var stores []localization.StoreDeclaration
	var bindings []localization.ApplicationBinding
	for _, p := range providers {
		stores = append(stores, localization.StoreDeclaration{Provider: p, ID: u.Store, Schema: localization.StoreSchema})
		bindings = append(bindings, localization.ApplicationBinding{Environment: u.Environment, Application: u.Application,
			Provider: p, StoreID: u.Store})
	}
	return stores, bindings
}
