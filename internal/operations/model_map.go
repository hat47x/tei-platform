package operations

import (
	"fmt"
	"sort"
	"strings"

	"tei/internal/ir"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// ModelMap is a deliberately partial, read-only navigation projection from
// the SAME loaded World and Source Index as ModelCatalog. It makes no claim
// that absent events, transitions, plugins or external providers do not exist.
// Renderer, model editor and generator decisions are separate qualifications.
type ModelMap struct {
	SchemaVersion    string         `json:"schemaVersion"`
	Effect           string         `json:"effect"`
	Catalog          ModelCatalog   `json:"catalog"`
	Links            []ModelMapLink `json:"links"`
	Coverage         string         `json:"coverage"`
	UnqualifiedRoles []string       `json:"unqualifiedRoles"`
	AuthorityLimits  []string       `json:"authorityLimits"`
}

// Map links identify the semantic subject by KIND and ID, not by rendered
// labels, source filenames, graph positions or the order of a list.
type ModelMapLink struct {
	From       sourceindex.Subject `json:"from"`
	To         sourceindex.Subject `json:"to"`
	Relation   string              `json:"relation"`
	Resolution string              `json:"resolution"`
}

// MapWorkspace uses one Loader result and its Source Index for the
// navigation candidate. The Loader/Index is not an atomic filesystem snapshot;
// ModelRevision is informative and cannot authorize edit or execute.
func MapWorkspace(root string) (ModelMap, error) {
	if strings.TrimSpace(root) == "" || root != strings.TrimSpace(root) {
		return ModelMap{}, fmt.Errorf("workspace path must be nonempty and exact")
	}
	loaded, err := workspace.Load(root)
	if err != nil {
		return ModelMap{}, &ValidationFailure{Stage: StageLoad, Err: err}
	}
	idx, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		return ModelMap{}, &ValidationFailure{Stage: StageSourceIndex, Err: err}
	}
	worldFP, sourceFP, err := workspaceProjectionFingerprints(loaded.World, idx)
	if err != nil {
		return ModelMap{}, err
	}
	catalog := CatalogWorld(loaded.World, idx, "", worldFP, sourceFP)
	identity := loaded.Identity()
	catalog.Workspace = &identity
	// CatalogWorld is a low-level projection and does not run validation.
	// MapWorkspace must carry the same verdict as CatalogWorkspace for this
	// loaded Workspace, without interpreting a zero-value OK as a failure.
	catalog.Validation = catalogValidation(loaded)
	return MapWorld(loaded.World, catalog), nil
}

// MapWorld projects typed navigation links from declared IR references only.
// In particular, a Relation edge has no effect on current Host authority.
func MapWorld(world ir.World, catalog ModelCatalog) ModelMap {
	declared := map[sourceindex.Subject]int{}
	for _, entry := range catalog.Entries {
		declared[entry.Subject]++
	}
	out := ModelMap{
		SchemaVersion: "tei.operations.model-map/v0",
		Effect:        "read-only",
		Catalog:       catalog,
		Links:         []ModelMapLink{},
		Coverage:      "partial-projection",
		// These roles can be expressed by extensions or other sources, but the
		// current native Catalog does not establish their required closure.
		UnqualifiedRoles: []string{
			"business-event-flow",
			"lifecycle-state-transition",
			"effect-receipt",
			"extension-hook-binding",
			"consumer-disclosure",
		},
		AuthorityLimits: []string{
			"navigation-is-not-complete-domain-coverage",
			"missing-declared-link-is-not-absent-business-relation",
			"derived-reference-does-not-authorize-effect",
			"pattern-query-binding-is-not-a-function-or-provider-read",
			"standard-custom-extension-qualification-is-not-observed",
		},
	}
	seen := map[ModelMapLink]bool{}
	add := func(kind string, from, to sourceindex.Subject) {
		if from.ID == "" || to.ID == "" {
			return
		}
		resolution := "resolved"
		switch {
		case declared[from] == 0 || declared[to] == 0:
			resolution = "unresolved"
		case declared[from] != 1 || declared[to] != 1:
			resolution = "ambiguous"
		}
		link := ModelMapLink{From: from, To: to, Relation: kind, Resolution: resolution}
		if !seen[link] {
			out.Links = append(out.Links, link)
			seen[link] = true
		}
	}
	subject := func(kind, id string) sourceindex.Subject {
		return sourceindex.Subject{Kind: kind, ID: id}
	}
	for _, e := range catalog.Entries {
		if e.Parent != nil {
			add("contains", *e.Parent, e.Subject)
		}
	}
	for _, ns := range world.Semantics {
		for _, data := range ns.Data {
			for _, attr := range data.Attributes {
				add("has-type", subject("semantic-attribute", string(attr.ID)),
					subject("semantic-type", string(attr.TypeRef)))
			}
			for _, rel := range data.Relations {
				add("refers-to", subject("semantic-relation", string(rel.ID)),
					subject("semantic-data", string(rel.Target)))
			}
		}
	}
	for _, app := range world.Applications {
		for _, pattern := range app.Patterns {
			p := subject("application-pattern", string(pattern.ID))
			add("uses-data", p, subject("semantic-data", string(pattern.Data)))
			add("uses-query", p, subject("query", string(pattern.Query)))
			add("uses-relation", p, subject("semantic-relation", string(pattern.Relation)))
			add("invokes-capability", p, subject("semantic-capability", string(pattern.Capability)))
			if pattern.Interaction != nil {
				for _, nav := range pattern.Interaction.Navigations {
					add("navigates-to",
						subject("application-pattern", string(nav.From)), p)
				}
				for _, cmd := range pattern.Interaction.Commands {
					add("interaction-invokes", p,
						subject("semantic-capability", string(cmd.Capability)))
				}
			}
		}
	}
	for _, q := range world.Queries {
		from := subject("query", string(q.ID))
		add("owned-by", from, subject("application", string(q.Application)))
		for _, column := range q.ResultColumns {
			add("maps-result-to", from, subject("semantic-attribute", string(column.Attribute)))
		}
	}
	sort.Slice(out.Links, func(i, j int) bool {
		a, b := out.Links[i], out.Links[j]
		for k, av := range []string{a.From.Kind, a.From.ID, a.Relation, a.To.Kind, a.To.ID, a.Resolution} {
			bv := []string{b.From.Kind, b.From.ID, b.Relation, b.To.Kind, b.To.ID, b.Resolution}[k]
			if av != bv {
				return av < bv
			}
		}
		return false
	})
	return out
}
