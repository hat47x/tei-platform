// Package multilingualresources contains a read-only E1 reference for display
// resources linked to an existing typed TEI model reference. It neither
// resolves plugin providers nor accepts changes to authoritative meaning.
package multilingualresources

import (
	"sort"
	"strconv"
	"strings"

	"golang.org/x/text/language"
	"tei/internal/modelref"
)

// Key identifies one display purpose owned by a logical plugin. It contains
// the existing modelref.Key rather than introducing another subject identity.
type Key struct {
	Owner  string       `json:"owner"`
	Target modelref.Key `json:"target"`
	Field  string       `json:"field"`
}

type Source struct {
	Locale   string `json:"locale"`
	Text     string `json:"text"`
	Revision string `json:"revision"`
}

type Translation struct {
	Locale  string `json:"locale"`
	Text    string `json:"text"`
	BasedOn string `json:"basedOn"`
	Review  string `json:"review"` // candidate, confirmed, needs-review
}

type Entry struct {
	Key          Key           `json:"key"`
	Source       Source        `json:"source"`
	Translations []Translation `json:"translations,omitempty"`
	Critical     bool          `json:"critical,omitempty"`
}

type Diagnostic struct {
	Code string `json:"code"`
	At   string `json:"at"`
}

// Request contains the caller's already selected preferences. The existing
// plugin selection and configuration layers own how these were determined.
type Request struct {
	Key                 Key
	Locale              string
	Fallback            []string // explicit, ordered; no implicit language-parent fallback
	AllowSourceFallback bool
}

type Result struct {
	Text           string `json:"text,omitempty"`
	Locale         string `json:"locale,omitempty"`
	SourceRevision string `json:"sourceRevision,omitempty"`
	FromSource     bool   `json:"fromSource,omitempty"`
	Fallback       bool   `json:"fallback,omitempty"`
	Diagnostic     string `json:"diagnostic,omitempty"`
}

func validLocale(value string) bool {
	tag, err := language.Parse(value)
	return err == nil && tag.String() == value && value != "und"
}

// Validate finds competing owners and translations without using registration
// order to pick a winner. Source and translation revisions stay separate.
func Validate(entries []Entry) []Diagnostic {
	var out []Diagnostic
	seen := make(map[Key]bool)
	for i, e := range entries {
		at := "entries/" + strconv.Itoa(i)
		if strings.TrimSpace(e.Key.Owner) == "" || strings.TrimSpace(e.Key.Field) == "" ||
			strings.TrimSpace(e.Key.Target.Scope) == "" || strings.TrimSpace(e.Key.Target.Subject.Kind) == "" ||
			strings.TrimSpace(e.Key.Target.Subject.ID) == "" {
			out = append(out, Diagnostic{"invalid-key", at})
		}
		if seen[e.Key] {
			out = append(out, Diagnostic{"duplicate-resource", at})
		}
		seen[e.Key] = true
		if !validLocale(e.Source.Locale) || strings.TrimSpace(e.Source.Revision) == "" ||
			strings.TrimSpace(e.Source.Text) == "" {
			out = append(out, Diagnostic{"invalid-source", at})
		}
		locales := map[string]bool{e.Source.Locale: true}
		for j, t := range e.Translations {
			target := at + "/translations/" + strconv.Itoa(j)
			if !validLocale(t.Locale) || strings.TrimSpace(t.Text) == "" ||
				strings.TrimSpace(t.BasedOn) == "" || (t.Review != "candidate" && t.Review != "confirmed" && t.Review != "needs-review") {
				out = append(out, Diagnostic{"invalid-translation", target})
			}
			if locales[t.Locale] {
				out = append(out, Diagnostic{"duplicate-language", target})
			}
			locales[t.Locale] = true
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].At != out[j].At {
			return out[i].At < out[j].At
		}
		return out[i].Code < out[j].Code
	})
	return out
}

// Resolve performs read-only display selection after the caller has selected
// the plugin, resource set and effective locale. Invalid catalogs fail closed.
// An existing but stale/unapproved exact translation blocks fallback.
func Resolve(graph modelref.Graph, entries []Entry, req Request) Result {
	if len(Validate(entries)) != 0 {
		return Result{Diagnostic: "invalid-catalog"}
	}
	if !validLocale(req.Locale) || strings.TrimSpace(req.Key.Owner) == "" || strings.TrimSpace(req.Key.Field) == "" {
		return Result{Diagnostic: "invalid-request"}
	}
	localeSeen := map[string]bool{req.Locale: true}
	for _, locale := range req.Fallback {
		if !validLocale(locale) || localeSeen[locale] {
			return Result{Diagnostic: "invalid-request"}
		}
		localeSeen[locale] = true
	}
	if graph.Schema != "tei.modelref.v0" {
		return Result{Diagnostic: "unsupported-model"}
	}
	target, issue := modelref.Resolve(graph.Nodes, modelref.Ref{Scope: req.Key.Target.Scope, Subject: req.Key.Target.Subject})
	if issue != "" || target != req.Key.Target {
		return Result{Diagnostic: "unbound-subject"}
	}
	var entry *Entry
	for i := range entries {
		if entries[i].Key == req.Key {
			entry = &entries[i]
			break
		}
	}
	if entry == nil {
		return Result{Diagnostic: "missing-resource"}
	}
	locales := []string{req.Locale}
	if !entry.Critical {
		locales = append(locales, req.Fallback...)
	}
	for i, locale := range locales {
		if locale == entry.Source.Locale {
			if i == 0 || req.AllowSourceFallback {
				return Result{Text: entry.Source.Text, Locale: locale, SourceRevision: entry.Source.Revision, FromSource: true, Fallback: i != 0}
			}
			continue
		}
		for _, t := range entry.Translations {
			if t.Locale != locale {
				continue
			}
			if t.BasedOn != entry.Source.Revision {
				return Result{Diagnostic: "stale-translation"}
			}
			if t.Review != "confirmed" {
				return Result{Diagnostic: "unapproved-translation"}
			}
			return Result{Text: t.Text, Locale: locale, SourceRevision: entry.Source.Revision, Fallback: i != 0}
		}
	}
	if !entry.Critical && req.AllowSourceFallback && !localeSeen[entry.Source.Locale] {
		return Result{Text: entry.Source.Text, Locale: entry.Source.Locale, SourceRevision: entry.Source.Revision, FromSource: true, Fallback: true}
	}
	return Result{Diagnostic: "missing-translation"}
}
