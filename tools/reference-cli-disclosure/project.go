// Package disclosureref pressures C2 projection boundaries. It is NOT a Policy Engine.
// A trusted Host supplies a decision from the existing authorization capability.
package disclosureref

import (
	"encoding/json"
	"slices"
)

const Version = "tei.reference.cli-disclosure/v0"

// Binding is verified by the Host, never self-declared in selector/LLM arguments.
type Binding struct {
	Principal, Caller, Workspace, Tenant, Channel, Destination, Purpose string
}
type Selector struct {
	Operation, Kind, ID, GraphScope, Direction, Mode string
}
type Request struct {
	Selector  Selector
	Binding   Binding
	Requested []string
	Previous  *Receipt // internal evidence; a production Host must authenticate it
	Now       int64    // trusted Host clock
}
type Raw struct {
	Kind, ID                     string
	Canonical                    json.RawMessage // all-or-nothing authorization
	Sources, Related, Unresolved []string
	Total                        int
	DataSnapshot, SourceProvider string
}

// Manifest stays inside the trusted Host. Authorization must examine ALL members
// of any facet before granting that facet for the specified destination.
type Manifest struct {
	CanonicalSubject, DataSnapshot, SourceProvider string
	Sources, Related, Unresolved                   []string
}
type Decision struct {
	Allowed, Current                            bool
	Bound                                       Binding
	Selector                                    Selector
	AuthorizedSnapshot                          string
	PolicyRevision, DecisionRef, ReusePartition string // Host-issued non-secret refs
	ExpiresAt                                   int64
	Sections                                    []string
	PartialAllowed, ExistenceSensitive          bool
}
type Authorizer interface {
	Decide(Selector, Binding, Manifest) (Decision, error)
}
type Receipt struct {
	Binding                                      Binding
	Selector                                     Selector
	PolicyRevision, ReusePartition, DataSnapshot string
}
type Public struct {
	Version      string          `json:"schemaVersion"`
	Status       string          `json:"status"`
	Completeness string          `json:"completeness,omitempty"`
	Kind         string          `json:"subjectKind,omitempty"`
	ID           string          `json:"subjectId,omitempty"`
	Canonical    json.RawMessage `json:"canonical,omitempty"`
	Sources      []string        `json:"sources,omitempty"`
	Related      []string        `json:"related,omitempty"`
	Unresolved   []string        `json:"unresolved,omitempty"`
	Total        *int            `json:"authorizedTotal,omitempty"`
}
type Audit struct {
	Binding                                     Binding
	Selector                                    Selector
	SourceProvider, PolicyRevision, DecisionRef string
	Outcome, Reason                             string // fixed non-secret internal codes
	Sections                                    []string
}

func deny(a Audit, reason, outcome string) (Public, Audit, *Receipt) {
	a.Reason, a.Outcome = reason, outcome
	return Public{Version: Version, Status: "unavailable"}, a, nil
}
func validBinding(b Binding) bool {
	return b.Principal != "" && b.Caller != "" && b.Workspace != "" &&
		b.Tenant != "" && b.Channel != "" && b.Destination != "" && b.Purpose != ""
}
func validSelector(s Selector) bool {
	if (s.Operation != "inspect" && s.Operation != "trace") || s.Kind == "" || s.ID == "" {
		return false
	}
	if s.Operation == "trace" && s.GraphScope == "" {
		return false
	}
	return s.Mode == "summary" || s.Mode == "full"
}
func validSections(ss []string) bool {
	if len(ss) == 0 {
		return false
	}
	seen := map[string]bool{}
	for _, s := range ss {
		if !slices.Contains([]string{"subject", "canonical", "source", "related", "unresolved", "count"}, s) || seen[s] {
			return false
		}
		seen[s] = true
	}
	return true
}

// C1 allows summary -> full for the same subject/direction; C2 still
// authorizes the new mode and requested fields afresh.
func sameContinuation(a, b Selector) bool {
	return a.Operation == b.Operation && a.Kind == b.Kind && a.ID == b.ID &&
		a.GraphScope == b.GraphScope && a.Direction == b.Direction
}

// Project constructs public data only from explicitly authorized facets.
// Raw C1 results, policy identifiers, secret error objects, and raw fingerprints
// must never be serialized into the public response.
func Project(q Request, raw Raw, authorization Authorizer) (Public, Audit, *Receipt) {
	a := Audit{Binding: q.Binding, Selector: q.Selector, SourceProvider: raw.SourceProvider}
	if authorization == nil || !validBinding(q.Binding) || !validSelector(q.Selector) || !validSections(q.Requested) {
		return deny(a, "invalid-host-or-selector", "error")
	}
	if raw.Kind != q.Selector.Kind || raw.ID != q.Selector.ID || raw.DataSnapshot == "" {
		return deny(a, "unavailable-subject-or-snapshot", "deny")
	}
	m := Manifest{
		CanonicalSubject: raw.Kind + ":" + raw.ID, DataSnapshot: raw.DataSnapshot, SourceProvider: raw.SourceProvider,
		Sources: slices.Clone(raw.Sources), Related: slices.Clone(raw.Related), Unresolved: slices.Clone(raw.Unresolved),
	}
	d, err := authorization.Decide(q.Selector, q.Binding, m)
	if err != nil {
		return deny(a, "authorization-failure", "error")
	}
	a.DecisionRef, a.PolicyRevision = d.DecisionRef, d.PolicyRevision
	if !d.Allowed {
		return deny(a, "authorization-denied", "deny")
	}
	if !d.Current || d.ExpiresAt <= q.Now || d.ExpiresAt <= 0 || d.PolicyRevision == "" ||
		d.DecisionRef == "" || d.ReusePartition == "" || d.AuthorizedSnapshot != raw.DataSnapshot ||
		d.Bound != q.Binding || d.Selector != q.Selector || !validSections(d.Sections) {
		return deny(a, "invalid-or-stale-decision", "stale")
	}
	if q.Previous != nil && (q.Previous.Binding != q.Binding ||
		!sameContinuation(q.Previous.Selector, q.Selector) ||
		q.Previous.PolicyRevision != d.PolicyRevision || q.Previous.ReusePartition != d.ReusePartition ||
		q.Previous.DataSnapshot != raw.DataSnapshot) {
		return deny(a, "stale-continuation", "stale")
	}
	allow := map[string]bool{}
	for _, s := range d.Sections {
		allow[s] = true
	}
	hidden := false
	for _, s := range q.Requested {
		if !allow[s] {
			hidden = true
		}
	}
	if !allow["subject"] || (hidden && (!d.PartialAllowed || d.ExistenceSensitive)) {
		return deny(a, "visibility-not-permitted", "deny")
	}
	p := Public{Version: Version, Status: "available", Kind: raw.Kind, ID: raw.ID, Completeness: "complete"}
	if hidden {
		p.Completeness = "partial"
	}
	for _, s := range q.Requested {
		if !allow[s] {
			continue
		}
		switch s {
		case "canonical":
			if !json.Valid(raw.Canonical) {
				return deny(a, "invalid-canonical", "error")
			}
			p.Canonical = append(json.RawMessage(nil), raw.Canonical...)
		case "source":
			p.Sources = slices.Clone(raw.Sources)
		case "related":
			p.Related = slices.Clone(raw.Related)
		case "unresolved":
			p.Unresolved = slices.Clone(raw.Unresolved)
		case "count":
			// An unfiltered global total cannot represent a partially disclosed view.
			if hidden {
				return deny(a, "global-total-not-partial-count", "deny")
			}
			n := raw.Total
			p.Total = &n
		}
	}
	a.Outcome, a.Reason, a.Sections = "allow", "authorized-projection", slices.Clone(q.Requested)
	rec := &Receipt{Binding: q.Binding, Selector: q.Selector, PolicyRevision: d.PolicyRevision,
		ReusePartition: d.ReusePartition, DataSnapshot: raw.DataSnapshot}
	return p, a, rec
}
