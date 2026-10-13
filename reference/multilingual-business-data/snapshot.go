// Package multilingualbusinessdata is a read-only reference for language-specific
// business values. It does not own TEI acceptance, database writes or AI execution.
package multilingualbusinessdata

import (
	"sort"
	"strconv"
	"strings"

	"golang.org/x/text/language"
	"tei/internal/modelref"
)

// Key identifies a language variant of an existing business record field.
// The stable business record and model field are NOT translated.
type Key struct {
	Tenant  string       `json:"tenant"`
	Record  string       `json:"record"`
	Field   modelref.Key `json:"field"`
	Locale  string       `json:"locale"`
	Edition string       `json:"edition"`
}

type SourceRef struct {
	Key      Key    `json:"key"`
	Revision string `json:"revision"`
}

type Revision struct {
	Key      Key        `json:"key"`
	Revision string     `json:"revision"`
	Text     string     `json:"text"`
	Origin   string     `json:"origin"` // original, translation, independent
	BasedOn  *SourceRef `json:"basedOn,omitempty"`
	Review   string     `json:"review"` // candidate, approved
}

// Publication is an explicit pointer to a published revision, not an
// assumption that the numerically largest revision is authoritative.
type Publication struct {
	Key      Key    `json:"key"`
	Revision string `json:"revision"`
}

type Snapshot struct {
	Model        modelref.Graph `json:"model"`
	Revisions    []Revision     `json:"revisions"`
	Publications []Publication  `json:"publications"`
}

type Diagnostic struct {
	Code string `json:"code"`
	At   string `json:"at"`
}

type Result struct {
	Text       string `json:"text,omitempty"`
	Revision   string `json:"revision,omitempty"`
	Locale     string `json:"locale,omitempty"`
	Diagnostic string `json:"diagnostic,omitempty"`
}

type revisionKey struct {
	Key      Key
	Revision string
}

func validKey(k Key) bool {
	if strings.TrimSpace(k.Tenant) == "" || strings.TrimSpace(k.Record) == "" ||
		strings.TrimSpace(k.Edition) == "" || k.Field.Scope == "" ||
		k.Field.Subject.Kind != "attribute" || k.Field.Subject.ID == "" {
		return false
	}
	tag, err := language.Parse(k.Locale)
	return err == nil && k.Locale != "und" && tag.String() == k.Locale
}

func findRevision(s Snapshot, ref SourceRef) (Revision, bool) {
	for _, r := range s.Revisions {
		if r.Key == ref.Key && r.Revision == ref.Revision {
			return r, true
		}
	}
	return Revision{}, false
}

func findPublication(s Snapshot, k Key) (string, bool) {
	for _, p := range s.Publications {
		if p.Key == k {
			return p.Revision, true
		}
	}
	return "", false
}

// Validate examines structural integrity without granting read, publish or
// acceptance authority. Stale source relations remain valid history and are
// handled by Select and CheckProposal.
func Validate(s Snapshot) []Diagnostic {
	var out []Diagnostic
	if s.Model.Schema != "tei.modelref.v0" {
		out = append(out, Diagnostic{"unsupported-model", "model"})
	}
	seenRevision := map[revisionKey]bool{}
	for i, r := range s.Revisions {
		at := "revisions/" + strconv.Itoa(i)
		if !validKey(r.Key) || r.Revision == "" || strings.TrimSpace(r.Text) == "" ||
			(r.Review != "candidate" && r.Review != "approved") {
			out = append(out, Diagnostic{"invalid-revision", at})
		}
		if seenRevision[revisionKey{r.Key, r.Revision}] {
			out = append(out, Diagnostic{"duplicate-revision", at})
		}
		seenRevision[revisionKey{r.Key, r.Revision}] = true
		if r.Origin != "original" && r.Origin != "translation" && r.Origin != "independent" {
			out = append(out, Diagnostic{"invalid-origin", at})
		} else if r.Origin != "translation" && r.BasedOn != nil {
			out = append(out, Diagnostic{"unexpected-source", at})
		} else if r.Origin == "translation" {
			if r.BasedOn == nil || r.BasedOn.Revision == "" ||
				!validKey(r.BasedOn.Key) || r.BasedOn.Key.Tenant != r.Key.Tenant ||
				r.BasedOn.Key.Record != r.Key.Record ||
				r.BasedOn.Key.Field != r.Key.Field || r.BasedOn.Key.Edition != r.Key.Edition ||
				r.BasedOn.Key.Locale == r.Key.Locale {
				out = append(out, Diagnostic{"invalid-translation-source", at})
			}
		}
		if s.Model.Schema == "tei.modelref.v0" {
			ref := modelref.Ref{Scope: r.Key.Field.Scope, Subject: r.Key.Field.Subject}
			if k, code := modelref.Resolve(s.Model.Nodes, ref); code != "" || k != r.Key.Field {
				out = append(out, Diagnostic{"unbound-attribute", at})
			}
		}
	}
	for i, r := range s.Revisions {
		if r.Origin != "translation" || r.BasedOn == nil {
			continue
		}
		source, ok := findRevision(s, *r.BasedOn)
		if !ok {
			out = append(out, Diagnostic{"missing-source-revision", "revisions/" + strconv.Itoa(i)})
		} else if source.Origin == "translation" {
			// E1 only allows a direct original -> translation relation.
			// This avoids silently treating multi-hop stale translations as current.
			out = append(out, Diagnostic{"chained-translation-unsupported", "revisions/" + strconv.Itoa(i)})
		}
	}
	seenPublications := map[Key]bool{}
	for i, p := range s.Publications {
		at := "publications/" + strconv.Itoa(i)
		if !validKey(p.Key) || p.Revision == "" {
			out = append(out, Diagnostic{"invalid-publication", at})
		}
		if seenPublications[p.Key] {
			out = append(out, Diagnostic{"ambiguous-publication", at})
		}
		seenPublications[p.Key] = true
		r, ok := findRevision(s, SourceRef{Key: p.Key, Revision: p.Revision})
		if !ok {
			out = append(out, Diagnostic{"unbound-publication", at})
		} else if r.Review != "approved" {
			out = append(out, Diagnostic{"unapproved-publication", at})
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

// Select returns a published value only. No automatic source-language or
// regional fallback occurs; the caller already owns publication permissions.
func Select(s Snapshot, key Key) Result {
	if len(Validate(s)) != 0 {
		return Result{Diagnostic: "invalid-snapshot"}
	}
	if !validKey(key) {
		return Result{Diagnostic: "invalid-request"}
	}
	published, ok := findPublication(s, key)
	if !ok {
		return Result{Diagnostic: "missing-publication"}
	}
	r, _ := findRevision(s, SourceRef{Key: key, Revision: published})
	if r.Origin == "translation" {
		if r.BasedOn == nil {
			return Result{Diagnostic: "invalid-snapshot"}
		}
		current, exists := findPublication(s, r.BasedOn.Key)
		if !exists || current != r.BasedOn.Revision {
			return Result{Diagnostic: "stale-translation"}
		}
	}
	return Result{Text: r.Text, Revision: r.Revision, Locale: key.Locale}
}

// TranslationProposal is a candidate, not an accepted write.
type TranslationProposal struct {
	Target                 Key
	ExpectedTargetRevision string
	Source                 SourceRef
	Text                   string
}

// CheckProposal only checks source freshness and optimistic concurrency.
// It does not grant authority to approve, publish or execute a translation.
func CheckProposal(s Snapshot, p TranslationProposal) string {
	if len(Validate(s)) != 0 {
		return "invalid-snapshot"
	}
	if !validKey(p.Target) || !validKey(p.Source.Key) ||
		p.Source.Revision == "" || strings.TrimSpace(p.Text) == "" ||
		p.Target.Tenant != p.Source.Key.Tenant ||
		p.Target.Record != p.Source.Key.Record ||
		p.Target.Field != p.Source.Key.Field ||
		p.Target.Edition != p.Source.Key.Edition ||
		p.Target.Locale == p.Source.Key.Locale {
		return "invalid-proposal"
	}
	current, hasCurrent := findPublication(s, p.Target)
	if (hasCurrent && current != p.ExpectedTargetRevision) ||
		(!hasCurrent && p.ExpectedTargetRevision != "") {
		return "target-changed"
	}
	sourceCurrent, ok := findPublication(s, p.Source.Key)
	if !ok || sourceCurrent != p.Source.Revision {
		return "source-changed"
	}
	return "candidate-current"
}
