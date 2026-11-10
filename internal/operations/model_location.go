package operations

import (
	"fmt"
	"sort"
	"strings"

	"tei/internal/ir"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// ModelLocation answers "where is this element in the whole model" for one
// exact subject: its path from the top, its children and siblings, and the
// declared references that enter and leave it (RQ-036, design §19.8 G1/G3).
//
// It is derived from the same World and Source Index as ModelCatalog and
// carries the same fingerprints, so a person reading a diagram and an AI
// reading this text can confirm they look at one model version (MAP-21). It is
// a navigation aid: it neither accepts meaning nor grants edit authority, and
// it does not yet derive event flow, lifecycle or implementation bindings; the
// NotDerived list says so instead of letting an empty list read as "none".
type ModelLocation struct {
	SchemaVersion          string                       `json:"schemaVersion"`
	Effect                 string                       `json:"effect"`
	WorldFingerprint       string                       `json:"worldFingerprint"`
	SourceIndexFingerprint string                       `json:"sourceIndexFingerprint"`
	ModelRevision          string                       `json:"modelRevision"`
	Workspace              *workspace.WorkspaceIdentity `json:"workspace,omitempty"`
	Validation             CatalogValidation            `json:"validation"`
	Subject                ModelCatalogEntry            `json:"subject"`
	Path                   []ModelCatalogEntry          `json:"path"`
	Children               ModelLocationSlice           `json:"children"`
	Siblings               ModelLocationSlice           `json:"siblings"`
	Outgoing               []ModelLocationLink          `json:"outgoing"`
	Incoming               []ModelLocationLink          `json:"incoming"`
	Limit                  int                          `json:"limit"`
	// Attention counts the problems on this element and everything under it.
	// It is set by LocateWorkspace, which has the diagnostics; LocateInCatalog
	// alone leaves it nil rather than reporting zero problems it did not count.
	Attention       *ModelAttention       `json:"attention,omitempty"`
	Coverage        ModelLocationCoverage `json:"coverage"`
	AuthorityLimits []string              `json:"authorityLimits"`
}

// ModelLocationSlice is a bounded list. Total is the full count, Entries holds
// at most Limit items in catalog order, and Omitted is Total-len(Entries), so
// a truncated answer is never read as the whole set. KindCounts gives the
// shape of the full set even when Entries is cut.
type ModelLocationSlice struct {
	Total      int                 `json:"total"`
	Omitted    int                 `json:"omitted"`
	KindCounts []ModelKindCount    `json:"kindCounts"`
	Entries    []ModelCatalogEntry `json:"entries"`
}

type ModelKindCount struct {
	Kind  string `json:"kind"`
	Count int    `json:"count"`
}

// ModelLocationLink is one declared reference between two subjects. Role names
// what the reference means, because a relation target, an attribute type and a
// pattern's data are different relations even when they look like one line in
// a diagram. Resolved is false when the target ID names nothing in this World;
// such a link is reported, not dropped.
type ModelLocationLink struct {
	Role     string              `json:"role"`
	From     sourceindex.Subject `json:"from"`
	To       sourceindex.Subject `json:"to"`
	Resolved bool                `json:"resolved"`
}

// ModelLocationCoverage states which relation families this answer derives
// and which it does not. A caller must not read an empty Incoming list as "no
// function uses this" for a family listed in NotDerived.
type ModelLocationCoverage struct {
	Derived    []string `json:"derived"`
	NotDerived []string `json:"notDerived"`
}

const (
	LocationCodeSubjectNotFound  = "SUBJECT_NOT_FOUND"
	LocationCodeSubjectAmbiguous = "SUBJECT_AMBIGUOUS"
	LocationCodeInvalidLimit     = "INVALID_LIMIT"

	// LocationDefaultLimit bounds each list so the answer stays readable for a
	// person and fits an AI's context; LocationMaxLimit stops a caller from
	// asking for an unbounded dump through this command.
	LocationDefaultLimit = 50
	LocationMaxLimit     = 500
)

// Link roles. They are the declared references the World already carries.
const (
	LinkRoleRelationTarget    = "relation-target"
	LinkRoleAttributeType     = "attribute-type"
	LinkRolePatternData       = "pattern-data"
	LinkRolePatternRelation   = "pattern-relation"
	LinkRolePatternQuery      = "pattern-query"
	LinkRolePatternCapability = "pattern-capability"
)

// LocateWorkspace loads the Workspace once and answers the location of one
// subject. The catalog and the links are read from that single load, so the
// answer cannot mix two Source snapshots.
func LocateWorkspace(root string, subject sourceindex.Subject, limit int) (ModelLocation, error) {
	if limit == 0 {
		limit = LocationDefaultLimit
	}
	if limit < 1 || limit > LocationMaxLimit {
		return ModelLocation{}, &CatalogError{Code: LocationCodeInvalidLimit,
			Message: fmt.Sprintf("limit must be between 1 and %d", LocationMaxLimit)}
	}
	if strings.TrimSpace(subject.Kind) == "" || strings.TrimSpace(subject.ID) == "" ||
		subject.Kind != strings.TrimSpace(subject.Kind) || subject.ID != strings.TrimSpace(subject.ID) {
		return ModelLocation{}, fmt.Errorf("subject kind and id must be nonempty and exact")
	}
	read, err := loadCatalogRead(root)
	if err != nil {
		return ModelLocation{}, err
	}
	location, failure := LocateInCatalog(read.Catalog, read.World, subject, limit)
	if failure != nil {
		return ModelLocation{}, failure
	}
	// The problems under this element, counted the same way model overview
	// counts a section (§19.8 G5), so the detail and the overview agree.
	attention := newCatalogTree(read.Catalog, read.World, read.Diagnostics).subtreeAttention(subject)
	location.Attention = &attention
	return location, nil
}

// LocateInCatalog is the transport-independent projection. It reads only the
// catalog entries and the declared references in world; it never searches
// Source text or guesses a missing parent.
func LocateInCatalog(catalog ModelCatalog, world ir.World, subject sourceindex.Subject, limit int) (ModelLocation, *CatalogError) {
	if limit < 1 || limit > LocationMaxLimit {
		return ModelLocation{}, &CatalogError{Code: LocationCodeInvalidLimit,
			Message: fmt.Sprintf("limit must be between 1 and %d", LocationMaxLimit)}
	}
	var matches []ModelCatalogEntry
	byKey := map[sourceindex.Subject]ModelCatalogEntry{}
	for _, entry := range catalog.Entries {
		if _, seen := byKey[entry.Subject]; !seen {
			byKey[entry.Subject] = entry
		}
		if entry.Subject == subject {
			matches = append(matches, entry)
		}
	}
	switch {
	case len(matches) == 0:
		return ModelLocation{}, &CatalogError{Code: LocationCodeSubjectNotFound,
			Message: fmt.Sprintf("%s %s is not an element of this workspace's model catalog", subject.Kind, subject.ID)}
	case len(matches) > 1 || matches[0].Ambiguous:
		// The same Qualified ID declared twice has no single position. Picking
		// the first would show one of two places as "the" location.
		return ModelLocation{}, &CatalogError{Code: LocationCodeSubjectAmbiguous,
			Message: fmt.Sprintf("%s %s is declared more than once; its position is ambiguous", subject.Kind, subject.ID)}
	}
	self := matches[0]

	out := ModelLocation{
		SchemaVersion:          "tei.operations.model-location/v0",
		Effect:                 "read-only",
		WorldFingerprint:       catalog.WorldFingerprint,
		SourceIndexFingerprint: catalog.SourceIndexFingerprint,
		ModelRevision:          catalog.ModelRevision,
		Validation:             catalog.Validation,
		Subject:                self,
		Path:                   []ModelCatalogEntry{},
		Outgoing:               []ModelLocationLink{},
		Incoming:               []ModelLocationLink{},
		Limit:                  limit,
		Coverage: ModelLocationCoverage{
			Derived: []string{
				"containment",
				LinkRoleRelationTarget, LinkRoleAttributeType,
				LinkRolePatternData, LinkRolePatternRelation, LinkRolePatternQuery, LinkRolePatternCapability,
			},
			NotDerived: []string{
				"business-event-flow",
				"lifecycle-state-transition",
				"query-data-dependency",
				"implementation-binding",
				"test-and-evidence-binding",
			},
		},
		AuthorityLimits: []string{
			"location-is-a-derived-read-only-projection",
			"location-is-navigation-not-semantic-acceptance-or-edit-authority",
			"empty-lists-cover-only-the-derived-families",
		},
	}
	out.Workspace = catalog.Workspace

	// Path: walk parents from the top. A parent missing from the catalog ends
	// the walk; it is not invented. A cycle cannot occur in catalog parents,
	// but the guard keeps a malformed input from looping.
	visited := map[sourceindex.Subject]bool{self.Subject: true}
	for parent := self.Parent; parent != nil; {
		if visited[*parent] {
			break
		}
		visited[*parent] = true
		entry, ok := byKey[*parent]
		if !ok {
			break
		}
		out.Path = append([]ModelCatalogEntry{entry}, out.Path...)
		parent = entry.Parent
	}

	var children, siblings []ModelCatalogEntry
	for _, entry := range catalog.Entries {
		if entry.Parent != nil && *entry.Parent == self.Subject {
			children = append(children, entry)
		}
		if entry.Subject == self.Subject {
			continue
		}
		if sameParent(entry.Parent, self.Parent) && (self.Parent != nil || entry.Subject.Kind == self.Subject.Kind) {
			// Top-level elements have no parent; their siblings are the other
			// top-level elements of the same kind, not every root in the World.
			siblings = append(siblings, entry)
		}
	}
	out.Children = boundedSlice(children, limit)
	out.Siblings = boundedSlice(siblings, limit)

	for _, link := range declaredLinks(world, byKey) {
		if link.From == self.Subject {
			out.Outgoing = append(out.Outgoing, link)
		}
		if link.To == self.Subject {
			out.Incoming = append(out.Incoming, link)
		}
	}
	sortLinks(out.Outgoing)
	sortLinks(out.Incoming)
	return out, nil
}

func sameParent(a, b *sourceindex.Subject) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func boundedSlice(entries []ModelCatalogEntry, limit int) ModelLocationSlice {
	counts := map[string]int{}
	for _, entry := range entries {
		counts[entry.Subject.Kind]++
	}
	kinds := make([]ModelKindCount, 0, len(counts))
	for kind, count := range counts {
		kinds = append(kinds, ModelKindCount{Kind: kind, Count: count})
	}
	sort.Slice(kinds, func(i, j int) bool { return kinds[i].Kind < kinds[j].Kind })
	kept := entries
	if len(kept) > limit {
		kept = kept[:limit]
	}
	return ModelLocationSlice{
		Total:      len(entries),
		Omitted:    len(entries) - len(kept),
		KindCounts: kinds,
		Entries:    append([]ModelCatalogEntry{}, kept...),
	}
}

// declaredLinks lists the references the World declares between catalog
// subjects. The target kind comes from the field that holds the reference, so
// a relation target is looked up as Data and a pattern's query as a Query; an
// ID that does not exist under that kind is reported with Resolved=false.
func declaredLinks(world ir.World, byKey map[sourceindex.Subject]ModelCatalogEntry) []ModelLocationLink {
	var links []ModelLocationLink
	add := func(role string, from sourceindex.Subject, toKind, toID string) {
		if toID == "" {
			return
		}
		to := sourceindex.Subject{Kind: toKind, ID: toID}
		_, resolved := byKey[to]
		links = append(links, ModelLocationLink{Role: role, From: from, To: to, Resolved: resolved})
	}
	for _, ns := range world.Semantics {
		for _, data := range ns.Data {
			for _, attr := range data.Attributes {
				add(LinkRoleAttributeType, sourceindex.Subject{Kind: "semantic-attribute", ID: string(attr.ID)},
					"semantic-type", string(attr.TypeRef))
			}
			for _, rel := range data.Relations {
				add(LinkRoleRelationTarget, sourceindex.Subject{Kind: "semantic-relation", ID: string(rel.ID)},
					"semantic-data", string(rel.Target))
			}
		}
	}
	for _, app := range world.Applications {
		for _, pattern := range app.Patterns {
			from := sourceindex.Subject{Kind: "application-pattern", ID: string(pattern.ID)}
			add(LinkRolePatternData, from, "semantic-data", string(pattern.Data))
			add(LinkRolePatternRelation, from, "semantic-relation", string(pattern.Relation))
			add(LinkRolePatternQuery, from, "query", string(pattern.Query))
			add(LinkRolePatternCapability, from, "semantic-capability", string(pattern.Capability))
		}
	}
	return links
}

func sortLinks(links []ModelLocationLink) {
	sort.SliceStable(links, func(i, j int) bool {
		a, b := links[i], links[j]
		if a.Role != b.Role {
			return a.Role < b.Role
		}
		if a.From != b.From {
			return a.From.Kind+"\x00"+a.From.ID < b.From.Kind+"\x00"+b.From.ID
		}
		return a.To.Kind+"\x00"+a.To.ID < b.To.Kind+"\x00"+b.To.ID
	})
}
