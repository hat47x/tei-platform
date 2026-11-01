// Package auxiliary models where design auxiliary information belongs: terminology,
// revision history, annotations, bookmarks and zones. It follows the boundary decided
// for #1808: an item is meaning-canonical when a change to it changes execution,
// validation, authorization or target lowering; derived when it can be recomputed from
// the meaning or from version management; user-local when it is a per-viewer preference;
// display-info when it only changes how a diagram or screen is shown; and an operational
// record otherwise.
//
// The package is read-only. It never rewrites a source, accepts a definition, or stores
// a derived fact. It demonstrates the two invariants that matter most for this issue:
// annotations are listed from structured declarations instead of string markers, and a
// change to display, operational or user-local information never moves the meaning version.
package auxiliary

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const Schema = "tei.reference.authoring-auxiliary/v0"

// Placement is where an auxiliary item belongs.
type Placement string

const (
	// MeaningCanonical holds items that change execution, validation, authorization
	// or target lowering. They pass through acceptance before they take effect.
	MeaningCanonical Placement = "meaning-canonical"
	// Derived holds items recomputed from the meaning or from version management.
	Derived Placement = "derived"
	// UserLocal holds per-viewer preferences that never belong in the source.
	UserLocal Placement = "user-local"
	// DisplayInfo holds diagram or screen presentation that must not move the
	// meaning version.
	DisplayInfo Placement = "display-info"
	// OperationalRecord holds human or generated work records that reference a
	// meaning subject without becoming part of the meaning version.
	OperationalRecord Placement = "operational-record"
)

// Candidate is one auxiliary item proposed for placement.
type Candidate struct {
	ID string `json:"id"`
	// Content is the canonical content of the item. It enters the meaning
	// version only when the item is meaning-canonical.
	Content string `json:"content,omitempty"`
	// AffectsExecution is true when changing the item changes execution,
	// validation, authorization or target lowering.
	AffectsExecution bool `json:"affectsExecution"`
	// RecomputableFromCanonical is true when the item can be recomputed from the
	// meaning or from version management.
	RecomputableFromCanonical bool `json:"recomputableFromCanonical"`
	// PerViewer is true when the item is a preference of one viewer.
	PerViewer bool `json:"perViewer"`
	// PresentationOnly is true when the item only changes how something is shown.
	PresentationOnly bool `json:"presentationOnly"`
}

// Decision is the placement of one candidate, with the reason that decided it.
type Decision struct {
	ID        string    `json:"id"`
	Placement Placement `json:"placement"`
	Reason    string    `json:"reason"`
}

// Classify answers the ordered questions from the design note and stops at the first
// that applies. AffectsExecution is checked before RecomputableFromCanonical on purpose:
// a recomputable item that still changes execution is part of the meaning.
func Classify(c Candidate) Decision {
	switch {
	case c.AffectsExecution:
		return Decision{c.ID, MeaningCanonical, "changes execution, validation, authorization or target lowering"}
	case c.RecomputableFromCanonical:
		return Decision{c.ID, Derived, "recomputed from the meaning or from version management; do not store"}
	case c.PerViewer:
		return Decision{c.ID, UserLocal, "per-viewer preference; keep out of the source"}
	case c.PresentationOnly:
		return Decision{c.ID, DisplayInfo, "presentation only; must not move the meaning version"}
	default:
		return Decision{c.ID, OperationalRecord, "human or generated work record; reference the meaning subject"}
	}
}

// Annotation is a structured note about a meaning subject.
type Annotation struct {
	ID      string `json:"id"`
	Subject string `json:"subject"`
	Kind    string `json:"kind"`
	State   string `json:"state"`
	Author  string `json:"author"`
	Text    string `json:"text"`
}

// Finding is a diagnostic. It never decides that two meanings are the same.
type Finding struct {
	Code    string `json:"code"`
	Subject string `json:"subject"`
}

const embeddedMarker = "@NOTE@"

// ListAnnotations returns the annotations in a stable order without scanning any text
// for a marker. A note whose text still carries the legacy embedded marker is reported
// so the caller migrates it to a structured declaration instead of trusting the text.
func ListAnnotations(items []Annotation) ([]Annotation, []Finding) {
	out := make([]Annotation, 0, len(items))
	findings := []Finding{}
	for _, a := range items {
		if a.Subject == "" {
			findings = append(findings, Finding{"annotation-without-subject", a.ID})
		}
		switch a.Kind {
		case "issue", "question", "note":
		default:
			findings = append(findings, Finding{"unknown-annotation-kind", a.ID})
		}
		switch a.State {
		case "open", "done":
		default:
			findings = append(findings, Finding{"unknown-annotation-state", a.ID})
		}
		if containsMarker(a.Text) {
			// The old tool wrote a list of notes by looking for a marker in a free
			// text field. That is a string convention, not a structural meaning.
			findings = append(findings, Finding{"embedded-marker-not-structured", a.ID})
		}
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	sortFindings(findings)
	return out, findings
}

func containsMarker(text string) bool {
	return strings.Contains(text, embeddedMarker)
}

// RevisionEvent is one revision recorded by version management. Its facts are derived
// and must not be retyped by hand.
type RevisionEvent struct {
	Revision string `json:"revision"`
	At       string `json:"at"`
	Subject  string `json:"subject"`
	Change   string `json:"change"`
}

// RevisionFact is a revision fact derived from version management.
type RevisionFact struct {
	Revision string `json:"revision"`
	At       string `json:"at"`
	Subject  string `json:"subject"`
	Change   string `json:"change"`
}

// RevisionReason is the human-written reason for a revision, kept as an operational
// record tied to the revision and its actor.
type RevisionReason struct {
	Revision string `json:"revision"`
	Author   string `json:"author"`
	Reason   string `json:"reason"`
}

// History separates derived facts from human reasons so that a derived fact is never
// stored twice.
type History struct {
	Facts   []RevisionFact   `json:"facts"`
	Reasons []RevisionReason `json:"reasons"`
}

// DeriveRevisions turns version-management events into facts. The facts are the only
// revision record; no one transcribes them.
func DeriveRevisions(events []RevisionEvent) []RevisionFact {
	facts := make([]RevisionFact, 0, len(events))
	for _, e := range events {
		facts = append(facts, RevisionFact{e.Revision, e.At, e.Subject, e.Change})
	}
	sort.Slice(facts, func(i, j int) bool { return facts[i].Revision < facts[j].Revision })
	return facts
}

// CombineRevisions attaches human reasons to derived facts. A reason for a revision that
// version management does not know, or a reason that merely repeats the derived change,
// becomes a finding instead of a second source of truth.
func CombineRevisions(facts []RevisionFact, reasons []RevisionReason) (History, []Finding) {
	known := map[string]bool{}
	change := map[string]string{}
	for _, f := range facts {
		known[f.Revision] = true
		change[f.Revision] = f.Change
	}
	findings := []Finding{}
	kept := make([]RevisionReason, 0, len(reasons))
	for _, r := range reasons {
		if !known[r.Revision] {
			findings = append(findings, Finding{"reason-without-revision", r.Revision})
			continue
		}
		if r.Reason != "" && r.Reason == change[r.Revision] {
			findings = append(findings, Finding{"duplicated-derived-fact", r.Revision})
			continue
		}
		kept = append(kept, r)
	}
	sort.Slice(kept, func(i, j int) bool {
		if kept[i].Revision != kept[j].Revision {
			return kept[i].Revision < kept[j].Revision
		}
		return kept[i].Author < kept[j].Author
	})
	history := History{Facts: facts}
	if history.Facts == nil {
		history.Facts = []RevisionFact{}
	}
	if kept == nil {
		kept = []RevisionReason{}
	}
	history.Reasons = kept
	sortFindings(findings)
	return history, findings
}

// Term is one glossary entry. The name index is derived from Symbol; the meaning
// definition is a meaning candidate only when it is accepted with a basis.
type Term struct {
	ID         string `json:"id"`
	Symbol     string `json:"symbol"`
	Definition string `json:"definition"`
	Accepted   bool   `json:"accepted"`
	Basis      string `json:"basis"`
}

// TermEntry is one resolved glossary row.
type TermEntry struct {
	ID         string `json:"id"`
	Symbol     string `json:"symbol"`
	Definition string `json:"definition"`
	Status     string `json:"status"`
}

// TermIndex builds the derived name index and keeps a definition as meaning only when
// it is accepted with a basis. A guessed definition is left unresolved; the package
// never fills a meaning from inference.
func TermIndex(terms []Term) ([]TermEntry, []Finding) {
	entries := make([]TermEntry, 0, len(terms))
	findings := []Finding{}
	for _, t := range terms {
		if t.Symbol == "" {
			findings = append(findings, Finding{"term-without-symbol", t.ID})
		}
		entry := TermEntry{ID: t.ID, Symbol: t.Symbol}
		switch {
		case t.Definition == "":
			entry.Status = "index-only"
		case !t.Accepted:
			entry.Status = "unresolved"
			findings = append(findings, Finding{"definition-not-accepted", t.ID})
		case t.Basis == "":
			entry.Status = "unresolved"
			findings = append(findings, Finding{"definition-without-basis", t.ID})
		default:
			entry.Definition = t.Definition
			entry.Status = "accepted"
		}
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
	sortFindings(findings)
	return entries, findings
}

// Input is one read-only analysis request.
type Input struct {
	Candidates      []Candidate      `json:"candidates"`
	Annotations     []Annotation     `json:"annotations"`
	RevisionEvents  []RevisionEvent  `json:"revisionEvents"`
	RevisionReasons []RevisionReason `json:"revisionReasons"`
	Terms           []Term           `json:"terms"`
	// Display, Operational and Local are examples of non-meaning items. They are
	// carried only to show that they never move the meaning version.
	Display     []string `json:"display"`
	Operational []string `json:"operational"`
	Local       []string `json:"local"`
}

// Report is the read-only result.
type Report struct {
	Schema         string       `json:"schema"`
	Placements     []Decision   `json:"placements"`
	Annotations    []Annotation `json:"annotations"`
	History        History      `json:"history"`
	Terms          []TermEntry  `json:"terms"`
	MeaningVersion string       `json:"meaningVersion"`
	Findings       []Finding    `json:"findings"`
}

// Analyze classifies the candidates, lists the annotations, separates derived revision
// facts from human reasons, builds the derived glossary index, and computes the meaning
// version. The meaning version covers only the meaning-canonical items and accepted term
// definitions; display, operational and user-local inputs are outside it.
func Analyze(in Input) Report {
	placements := make([]Decision, 0, len(in.Candidates))
	meaning := []string{}
	findings := []Finding{}
	for _, c := range in.Candidates {
		d := Classify(c)
		if d.Placement == MeaningCanonical {
			meaning = append(meaning, "candidate:"+c.ID+"\x00"+c.Content)
		}
		if c.AffectsExecution && (c.RecomputableFromCanonical || c.PerViewer || c.PresentationOnly) {
			// A claim that the item is derived, per-viewer, or presentation-only
			// cannot hold when the item changes execution. Execution wins, and the
			// contradiction is reported instead of silently accepting the claim.
			findings = append(findings, Finding{"claim-contradicts-execution", c.ID})
		}
		placements = append(placements, d)
	}
	sort.Slice(placements, func(i, j int) bool { return placements[i].ID < placements[j].ID })

	annotations, aFind := ListAnnotations(in.Annotations)

	facts := DeriveRevisions(in.RevisionEvents)
	history, hFind := CombineRevisions(facts, in.RevisionReasons)

	terms, tFind := TermIndex(in.Terms)
	for _, t := range terms {
		if t.Status == "accepted" {
			meaning = append(meaning, "term:"+t.ID+"\x00"+t.Definition)
		}
	}

	findings = append(findings, aFind...)
	findings = append(findings, hFind...)
	findings = append(findings, tFind...)
	sortFindings(findings)

	return Report{
		Schema:         Schema,
		Placements:     placements,
		Annotations:    annotations,
		History:        history,
		Terms:          terms,
		MeaningVersion: MeaningVersion(meaning),
		Findings:       findings,
	}
}

// MeaningVersion is a digest over only the meaning-canonical item IDs. Changing display,
// operational or user-local information leaves it unchanged.
func MeaningVersion(meaning []string) string {
	ids := append([]string(nil), meaning...)
	sort.Strings(ids)
	raw, err := json.Marshal(ids)
	if err != nil {
		// Marshalling a string slice cannot fail; keep the signature total.
		raw = []byte(fmt.Sprintf("%v", ids))
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(raw))
}

func sortFindings(findings []Finding) {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Code != findings[j].Code {
			return findings[i].Code < findings[j].Code
		}
		return findings[i].Subject < findings[j].Subject
	})
}
