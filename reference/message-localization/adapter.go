// Package messagelocalization is a read-only reference for a std.localization
// plugin extending std.message. Its store/binding/policy declarations are
// candidates, not approved Workspace schema or runtime authorization.
package messagelocalization

import (
	"fmt"
	"sort"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/modelref"
	"tei/internal/plugin"
	"tei/internal/subject"
	resources "tei/reference/multilingual-resources"
	messages "tei/reference/standard-messages"
)

const StoreSchema = "tei.message-translation.v0"

// StoreDeclaration represents a plugin-defined logical storage contract.
// Host/Editor, not this package, owns physical application of that model.
type StoreDeclaration struct {
	Provider plugin.Identity `json:"provider"`
	ID       string          `json:"id"`
	Schema   string          `json:"schema"`
}

// ApplicationBinding enables one plugin store for one application/environment.
// Neither the mere presence of this struct nor a plugin signature grants it.
type ApplicationBinding struct {
	Environment core.CanonicalID `json:"environment"`
	Application core.CanonicalID `json:"application"`
	Provider    plugin.Identity  `json:"provider"`
	StoreID     string           `json:"storeId"`
}

type TranslationRecord struct {
	Application core.CanonicalID `json:"application"`
	StoreID     string           `json:"storeId"`
	Provider    plugin.Identity  `json:"provider"`
	Target      plugin.Identity  `json:"target"`
	MessageID   core.CanonicalID `json:"messageId"`
	Locale      string           `json:"locale"`
	Text        string           `json:"text"`
	BasedOn     string           `json:"basedOn"`
	Review      string           `json:"review"`
}

// Grant is supplied by a host-owned application/environment acceptance path.
type Grant struct {
	Environment core.CanonicalID `json:"environment"`
	Application core.CanonicalID `json:"application"`
	Provider    plugin.Identity  `json:"provider"`
	Target      plugin.Identity  `json:"target"`
	MessageID   core.CanonicalID `json:"messageId"`
	Locale      string           `json:"locale"`
}

// Override must name every defeated competing provider. No load-order tie-break.
type Override struct {
	Application core.CanonicalID `json:"application"`
	Target      plugin.Identity  `json:"target"`
	MessageID   core.CanonicalID `json:"messageId"`
	Locale      string           `json:"locale"`
	Winner      plugin.Identity  `json:"winner"`
	Replaces    plugin.Identity  `json:"replaces"`
}

type Policy struct {
	Mode                string   `json:"mode"` // "translated" (default) or "source-only"
	Locale              string   `json:"locale"`
	Fallback            []string `json:"fallback,omitempty"`
	AllowSourceFallback bool     `json:"allowSourceFallback,omitempty"`
}

type Request struct {
	Application core.CanonicalID `json:"application"`
	Message     messages.Ref     `json:"message"`
}

// ReviewInput is a review candidate, not an authorization token. Production
// host must authenticate the principal and independently recheck the accepted
// store bindings, source revision, plugin lock, grants and overrides.
type ReviewInput struct {
	Request         Request
	Policy          Policy
	SourceRevision  string
	LockFingerprint string
	Bindings        []ApplicationBinding
	Records         []TranslationRecord
	Grants          []Grant
	Overrides       []Override
}

type ReviewGate func(ReviewInput) error

type Input struct {
	Plugins     plugin.Catalog
	Environment environment.Environment
	Lock        plugin.Lock
	Messages    messages.Catalog
	Stores      []StoreDeclaration
	Bindings    []ApplicationBinding
	Records     []TranslationRecord
	Grants      []Grant
	Overrides   []Override
	Policy      Policy
	Request     Request
	Review      ReviewGate
}

type Output struct {
	Message         messages.Ref     `json:"message"`
	LockFingerprint string           `json:"lockFingerprint"`
	Resource        resources.Result `json:"resource"`
}

func chosen(set plugin.ResolvedSet, id plugin.Identity) bool {
	for _, item := range set.Plugins {
		if item == id {
			return true
		}
	}
	return false
}

func hasStandardMessage(set plugin.ResolvedSet) bool {
	for _, item := range set.Plugins {
		if item.ID == "std.message" {
			return true
		}
	}
	return false
}

func requirementForMessages(c plugin.Catalog, provider plugin.Identity) bool {
	for _, def := range c.Plugins {
		if def.ID != provider.ID || def.Version != provider.Version {
			continue
		}
		for _, req := range def.Requires {
			if req.Plugin == "std.message" && req.CompatibleMajor == 1 {
				return true
			}
		}
	}
	return false
}

func granted(in Input, rec TranslationRecord) bool {
	for _, g := range in.Grants {
		if g.Environment == in.Environment.ID && g.Application == in.Request.Application &&
			g.Provider == rec.Provider && g.Target == rec.Target && g.MessageID == rec.MessageID && g.Locale == rec.Locale {
			return true
		}
	}
	return false
}

func storeApplied(in Input, r TranslationRecord) bool {
	declarations := 0
	for _, store := range in.Stores {
		if store.Provider == r.Provider && store.ID == r.StoreID {
			if store.Schema != StoreSchema {
				return false
			}
			declarations++
		}
	}
	bindings := 0
	for _, binding := range in.Bindings {
		if binding.Environment == in.Environment.ID && binding.Application == in.Request.Application &&
			binding.Provider == r.Provider && binding.StoreID == r.StoreID {
			bindings++
		}
	}
	return declarations == 1 && bindings == 1
}

func displaces(in Input, winner, loser TranslationRecord) bool {
	for _, rule := range in.Overrides {
		if rule.Application == in.Request.Application && rule.Target == winner.Target &&
			rule.MessageID == winner.MessageID && rule.Locale == winner.Locale &&
			rule.Winner == winner.Provider && rule.Replaces == loser.Provider {
			return true
		}
	}
	return false
}

func selectRecords(in Input, def messages.Definition, resolved plugin.ResolvedSet) ([]resources.Translation, error) {
	byLocale := map[string][]TranslationRecord{}
	for _, rec := range in.Records {
		// A store of a different app, plugin or message cannot leak into this request.
		if rec.Application != in.Request.Application || rec.Target != def.Ref.Owner || rec.MessageID != def.Ref.ID {
			continue
		}
		if !chosen(resolved, rec.Provider) {
			return nil, fmt.Errorf("unselected-translation-provider")
		}
		if !requirementForMessages(in.Plugins, rec.Provider) {
			return nil, fmt.Errorf("missing-message-plugin-dependency")
		}
		if !storeApplied(in, rec) {
			return nil, fmt.Errorf("unapplied-plugin-store")
		}
		if !granted(in, rec) {
			return nil, fmt.Errorf("translation-not-granted")
		}
		if rec.Locale == def.SourceLocale {
			return nil, fmt.Errorf("source-override-unsupported")
		}
		if err := messages.ValidateText(def, rec.Text); err != nil {
			return nil, fmt.Errorf("invalid-translation-contract: %w", err)
		}
		for _, previous := range byLocale[rec.Locale] {
			if previous.Provider == rec.Provider {
				return nil, fmt.Errorf("duplicate-translation-provider")
			}
		}
		byLocale[rec.Locale] = append(byLocale[rec.Locale], rec)
	}
	locales := make([]string, 0, len(byLocale))
	for locale := range byLocale {
		locales = append(locales, locale)
	}
	sort.Strings(locales)
	translations := make([]resources.Translation, 0, len(locales))
	for _, locale := range locales {
		alternatives := byLocale[locale]
		selected := alternatives[0]
		if len(alternatives) > 1 {
			winners := []TranslationRecord{}
			for _, candidate := range alternatives {
				winner := true
				for _, other := range alternatives {
					if candidate.Provider != other.Provider && !displaces(in, candidate, other) {
						winner = false
					}
				}
				if winner {
					winners = append(winners, candidate)
				}
			}
			if len(winners) != 1 {
				return nil, fmt.Errorf("ambiguous-translation")
			}
			selected = winners[0]
		}
		translations = append(translations, resources.Translation{
			Locale: selected.Locale, Text: selected.Text, BasedOn: selected.BasedOn, Review: selected.Review,
		})
	}
	return translations, nil
}

// Resolve applies already-pinned plugins, message definitions, trusted review
// and E1's translation selection. It does NOT apply a DB schema or render MF2.
func Resolve(in Input) (Output, error) {
	deny := func(code string) (Output, error) { return Output{}, fmt.Errorf("message-localization: %s", code) }
	if in.Review == nil {
		return deny("host-review-required")
	}
	if in.Request.Application == "" {
		return deny("application-required")
	}
	resolved, err := in.Plugins.ResolveEnvironmentWithLock(in.Environment, in.Lock)
	if err != nil {
		return deny("plugin-lock: " + err.Error())
	}
	if !hasStandardMessage(resolved) {
		return deny("std-message-unselected")
	}
	if !chosen(resolved, in.Request.Message.Owner) {
		return deny("message-owner-unselected")
	}
	def, err := in.Messages.Lookup(in.Request.Message)
	if err != nil {
		return deny("message-catalog: " + err.Error())
	}
	fingerprint, err := plugin.EnvironmentLockFingerprint(in.Lock, in.Environment.ID)
	if err != nil {
		return deny("lock-fingerprint: " + err.Error())
	}
	if in.Policy.Mode != "" && in.Policy.Mode != "translated" && in.Policy.Mode != "source-only" {
		return deny("invalid-policy-mode")
	}
	key := modelref.Key{Scope: string(def.Ref.Owner.ID) + "@" + def.Ref.Owner.Version,
		Subject: subject.Subject{Kind: "message", ID: string(def.Ref.ID)}}
	resourceKey := resources.Key{Owner: string(def.Ref.Owner.ID), Target: key, Field: "message-body"}
	entry := resources.Entry{Key: resourceKey, Source: resources.Source{
		Locale: def.SourceLocale, Text: def.Text, Revision: messages.Revision(def)}, Critical: def.Critical}
	if in.Policy.Mode != "source-only" {
		translations, err := selectRecords(in, def, resolved)
		if err != nil {
			return deny(err.Error())
		}
		entry.Translations = translations
	}
	// A selected plugin set and matching digests do not authenticate policy.
	// The host must validate the exact candidate before releasing display text.
	if err := in.Review(ReviewInput{
		Request: in.Request, Policy: in.Policy, SourceRevision: entry.Source.Revision,
		LockFingerprint: fingerprint,
		Bindings:        append([]ApplicationBinding(nil), in.Bindings...),
		Records:         append([]TranslationRecord(nil), in.Records...),
		Grants:          append([]Grant(nil), in.Grants...), Overrides: append([]Override(nil), in.Overrides...),
	}); err != nil {
		return deny("host-review-denied: " + err.Error())
	}
	graph := modelref.Graph{Schema: "tei.modelref.v0", Nodes: []modelref.Node{{Key: key, DisplayName: def.Text}}}
	result := resources.Resolve(graph, []resources.Entry{entry}, resources.Request{
		Key: resourceKey, Locale: in.Policy.Locale, Fallback: in.Policy.Fallback,
		AllowSourceFallback: in.Policy.AllowSourceFallback,
	})
	if result.Diagnostic != "" {
		return deny(result.Diagnostic)
	}
	return Output{Message: in.Request.Message, LockFingerprint: fingerprint, Resource: result}, nil
}
