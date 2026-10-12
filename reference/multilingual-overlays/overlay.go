// Package multilingualoverlays composes optional localization packs over
// ordinary, already-resolved TEI plugins. This read-only reference changes
// neither the underlying graph nor plugin resolution/acceptance authority.
package multilingualoverlays

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"tei/internal/modelref"
	"tei/internal/plugin"
	resources "tei/reference/multilingual-resources"
)

// Surface is a trusted environment-side declaration. An ordinary plugin does
// not have to know about localization; its typed graph node provides the
// default display name. Only the display-name field is supported in O1.
type Surface struct {
	Target       plugin.Identity `json:"target"`
	Subject      modelref.Key    `json:"subject"`
	Field        string          `json:"field"`
	SourceLocale string          `json:"sourceLocale"`
	Critical     bool            `json:"critical,omitempty"`
}

// Pack is contributed by a separate, already-selected localization plugin.
// Target pins the exact ordinary plugin version. BasedOn is the original
// display text's SHA-256, not the semantic node version.
type Pack struct {
	Provider plugin.Identity `json:"provider"`
	Target   plugin.Identity `json:"target"`
	Key      resources.Key   `json:"key"`
	Locale   string          `json:"locale"`
	Text     string          `json:"text"`
	BasedOn  string          `json:"basedOn"`
	Review   string          `json:"review"`
}

// Grant must be supplied by the trusted environment/application policy, not
// self-declared by a localization pack. The target and provider versions are
// exact; a grant cannot change executable plugin capabilities.
type Grant struct {
	Environment string          `json:"environment"`
	Provider    plugin.Identity `json:"provider"`
	Target      plugin.Identity `json:"target"`
	Key         resources.Key   `json:"key"`
	Locale      string          `json:"locale"`
}

// Override explicitly displaces one already-permitted pack with another for
// one language and display purpose. This is not a numeric priority or load
// order; all displaced providers must be enumerated.
type Override struct {
	Key      resources.Key   `json:"key"`
	Locale   string          `json:"locale"`
	Winner   plugin.Identity `json:"winner"`
	Replaces plugin.Identity `json:"replaces"`
}

// Input is assembled after normal plugin and model resolution. It does not
// declare a new plugin dependency resolver, permission system or model owner.
type Input struct {
	Resolved  plugin.ResolvedSet `json:"resolved"`
	Graph     modelref.Graph     `json:"graph"`
	Surfaces  []Surface          `json:"surfaces"`
	Packs     []Pack             `json:"packs,omitempty"`
	Grants    []Grant            `json:"grants,omitempty"`
	Overrides []Override         `json:"overrides,omitempty"`
}

// Projection wraps existing E1 selection. Entries are display-only; callers
// must never interpret them as a change to canonical meaning or business data.
type Projection struct {
	graph   modelref.Graph
	entries []resources.Entry
	owners  map[resources.Key]plugin.Identity
}

type Lookup struct {
	Target              plugin.Identity
	Key                 resources.Key
	Locale              string
	Fallback            []string
	AllowSourceFallback bool
}

// Fingerprint tracks display text independently of business semantic version.
func Fingerprint(text string) string {
	sum := sha256.Sum256([]byte(text))
	return "sha256:" + hex.EncodeToString(sum[:])
}

type languageKey struct {
	Key    resources.Key
	Locale string
}

func resourceKey(s Surface) resources.Key {
	return resources.Key{Owner: string(s.Target.ID), Target: s.Subject, Field: s.Field}
}

func selected(resolved plugin.ResolvedSet, who plugin.Identity) bool {
	for _, p := range resolved.Plugins {
		if p == who {
			return true
		}
	}
	return false
}

func sortSurface(s []Surface) {
	sort.Slice(s, func(i, j int) bool {
		return fmt.Sprintf("%s@%s/%s/%s/%s/%s", s[i].Target.ID, s[i].Target.Version, s[i].Subject.Scope, s[i].Subject.Subject.Kind, s[i].Subject.Subject.ID, s[i].Field) <
			fmt.Sprintf("%s@%s/%s/%s/%s/%s", s[j].Target.ID, s[j].Target.Version, s[j].Subject.Scope, s[j].Subject.Subject.Kind, s[j].Subject.Subject.ID, s[j].Field)
	})
}

func sortPack(p []Pack) {
	sort.Slice(p, func(i, j int) bool {
		a, b := p[i], p[j]
		return fmt.Sprintf("%s/%s/%s/%s/%s@%s", a.Key.Owner, a.Key.Target.Scope, a.Key.Target.Subject.ID, a.Key.Field, a.Locale, a.Provider.ID) <
			fmt.Sprintf("%s/%s/%s/%s/%s@%s", b.Key.Owner, b.Key.Target.Scope, b.Key.Target.Subject.ID, b.Key.Field, b.Locale, b.Provider.ID)
	})
}

func permitted(input Input, p Pack) bool {
	for _, g := range input.Grants {
		if g.Environment == string(input.Resolved.Environment) && g.Provider == p.Provider &&
			g.Target == p.Target && g.Key == p.Key && g.Locale == p.Locale {
			return true
		}
	}
	return false
}

func displaces(input Input, winner, loser Pack) bool {
	for _, rule := range input.Overrides {
		if rule.Key == winner.Key && rule.Locale == winner.Locale &&
			rule.Winner == winner.Provider && rule.Replaces == loser.Provider {
			return true
		}
	}
	return false
}

func findNode(graph modelref.Graph, key modelref.Key) (modelref.Node, bool) {
	for _, node := range graph.Nodes {
		if node.Key == key {
			return node, true
		}
	}
	return modelref.Node{}, false
}

// Compose constructs a read-only E1 catalog after the ordinary plugin set has
// already been selected. A missing or ambiguous grant rejects the projection,
// rather than silently allowing a localization plugin to replace another.
func Compose(input Input) (Projection, string) {
	if input.Resolved.Environment == "" || input.Graph.Schema != "tei.modelref.v0" {
		return Projection{}, "invalid-composition-input"
	}
	if len(modelref.Validate(input.Graph, nil)) != 0 {
		return Projection{}, "invalid-model-graph"
	}
	pluginIDs := map[string]bool{}
	for _, p := range input.Resolved.Plugins {
		if p.ID == "" || p.Version == "" || pluginIDs[string(p.ID)] {
			return Projection{}, "ambiguous-plugin-set"
		}
		pluginIDs[string(p.ID)] = true
	}
	surfaces := append([]Surface(nil), input.Surfaces...)
	sortSurface(surfaces)
	entryByKey := make(map[resources.Key]resources.Entry)
	ownerByKey := make(map[resources.Key]plugin.Identity)
	for _, s := range surfaces {
		if !selected(input.Resolved, s.Target) {
			return Projection{}, "unselected-base-plugin"
		}
		if s.Field != "display-name" {
			return Projection{}, "unsupported-display-field"
		}
		key := resourceKey(s)
		if _, exists := entryByKey[key]; exists {
			return Projection{}, "duplicate-display-surface"
		}
		ref := modelref.Ref{Scope: s.Subject.Scope, Subject: s.Subject}
		found, code := modelref.Resolve(input.Graph.Nodes, ref)
		node, exists := findNode(input.Graph, s.Subject)
		if code != "" || !exists || found != s.Subject || strings.TrimSpace(node.DisplayName) == "" {
			return Projection{}, "unbound-display-surface"
		}
		entry := resources.Entry{
			Key: key,
			Source: resources.Source{Locale: s.SourceLocale, Text: node.DisplayName,
				Revision: Fingerprint(node.DisplayName)},
			Critical: s.Critical,
		}
		if len(resources.Validate([]resources.Entry{entry})) != 0 {
			return Projection{}, "invalid-display-surface"
		}
		entryByKey[key] = entry
		ownerByKey[key] = s.Target
	}
	packs := append([]Pack(nil), input.Packs...)
	sortPack(packs)
	groups := make(map[languageKey][]Pack)
	for _, p := range packs {
		if !selected(input.Resolved, p.Provider) {
			return Projection{}, "unselected-locale-plugin"
		}
		target, found := ownerByKey[p.Key]
		if !found {
			return Projection{}, "unbound-overlay-target"
		}
		if target != p.Target {
			return Projection{}, "incompatible-plugin-version"
		}
		entry := entryByKey[p.Key]
		if p.Locale == entry.Source.Locale {
			// O1 does not replace the original language's authoritative label.
			return Projection{}, "source-override-unsupported"
		}
		if !permitted(input, p) {
			return Projection{}, "overlay-not-permitted"
		}
		key := languageKey{Key: p.Key, Locale: p.Locale}
		for _, old := range groups[key] {
			if old.Provider == p.Provider {
				return Projection{}, "duplicate-overlay"
			}
		}
		groups[key] = append(groups[key], p)
	}
	groupKeys := make([]languageKey, 0, len(groups))
	for k := range groups {
		groupKeys = append(groupKeys, k)
	}
	sort.Slice(groupKeys, func(i, j int) bool {
		a, b := groupKeys[i], groupKeys[j]
		return fmt.Sprintf("%s/%s/%s/%s/%s", a.Key.Owner, a.Key.Target.Scope, a.Key.Target.Subject.ID, a.Key.Field, a.Locale) <
			fmt.Sprintf("%s/%s/%s/%s/%s", b.Key.Owner, b.Key.Target.Scope, b.Key.Target.Subject.ID, b.Key.Field, b.Locale)
	})
	for _, key := range groupKeys {
		choices := groups[key]
		selectedPack := choices[0]
		if len(choices) > 1 {
			var winners []Pack
			for _, candidate := range choices {
				wins := true
				for _, opponent := range choices {
					if candidate.Provider != opponent.Provider && !displaces(input, candidate, opponent) {
						wins = false
					}
				}
				if wins {
					winners = append(winners, candidate)
				}
			}
			if len(winners) != 1 {
				return Projection{}, "ambiguous-overlay"
			}
			selectedPack = winners[0]
		}
		entry := entryByKey[key.Key]
		entry.Translations = append(entry.Translations, resources.Translation{
			Locale: selectedPack.Locale, Text: selectedPack.Text,
			BasedOn: selectedPack.BasedOn, Review: selectedPack.Review,
		})
		entryByKey[key.Key] = entry
	}
	keys := make([]resources.Key, 0, len(entryByKey))
	for k := range entryByKey {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return fmt.Sprintf("%s/%s/%s/%s/%s", keys[i].Owner, keys[i].Target.Scope, keys[i].Target.Subject.Kind, keys[i].Target.Subject.ID, keys[i].Field) <
			fmt.Sprintf("%s/%s/%s/%s/%s", keys[j].Owner, keys[j].Target.Scope, keys[j].Target.Subject.Kind, keys[j].Target.Subject.ID, keys[j].Field)
	})
	var entries []resources.Entry
	for _, key := range keys {
		entry := entryByKey[key]
		sort.Slice(entry.Translations, func(i, j int) bool { return entry.Translations[i].Locale < entry.Translations[j].Locale })
		entries = append(entries, entry)
	}
	if len(resources.Validate(entries)) != 0 {
		return Projection{}, "invalid-resource-overlay"
	}
	return Projection{graph: input.Graph, entries: entries, owners: ownerByKey}, ""
}

// Resolve uses E1 for language selection, review status, fallback and source
// revision checks. The ordinary plugin's capability identity is unchanged.
func (p Projection) Resolve(request Lookup) resources.Result {
	if who, ok := p.owners[request.Key]; !ok || who != request.Target {
		return resources.Result{Diagnostic: "unbound-lookup-owner"}
	}
	return resources.Resolve(p.graph, p.entries, resources.Request{
		Key: request.Key, Locale: request.Locale,
		Fallback: request.Fallback, AllowSourceFallback: request.AllowSourceFallback,
	})
}
