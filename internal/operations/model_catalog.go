package operations

import (
	"fmt"
	"sort"
	"strings"

	"tei/internal/ir"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// ModelCatalog is a read-only, target-neutral Modeler projection. It is a
// navigation aid, not a second canonical model, edit authority or acceptance.
type ModelCatalog struct {
	SchemaVersion          string              `json:"schemaVersion"`
	Effect                 string              `json:"effect"`
	NamespaceFilter        string              `json:"namespaceFilter,omitempty"`
	WorldFingerprint       string              `json:"worldFingerprint"`
	SourceIndexFingerprint string              `json:"sourceIndexFingerprint"`
	ModelRevision          string              `json:"modelRevision"`
	Entries                []ModelCatalogEntry `json:"entries"`
	AuthorityLimits        []string            `json:"authorityLimits"`
}

type ModelCatalogEntry struct {
	Subject      sourceindex.Subject  `json:"subject"`
	Parent       *sourceindex.Subject `json:"parent,omitempty"`
	Symbol       string               `json:"symbol,omitempty"`
	Detail       string               `json:"detail,omitempty"`
	SourceStatus string               `json:"sourceStatus"`
	Ambiguous    bool                 `json:"ambiguous,omitempty"`
}

// CatalogWorkspace reads the normalized TEI-native workspace through the
// existing loader and Source Index. XEA[D]/XEAF are not parsed or accepted here.
func CatalogWorkspace(root, namespace string) (ModelCatalog, error) {
	if strings.TrimSpace(root) == "" || root != strings.TrimSpace(root) {
		return ModelCatalog{}, fmt.Errorf("workspace path must be nonempty and exact")
	}
	if namespace != strings.TrimSpace(namespace) {
		return ModelCatalog{}, fmt.Errorf("namespace filter must be an exact ID")
	}
	loaded, err := workspace.Load(root)
	if err != nil {
		return ModelCatalog{}, &ValidationFailure{Stage: StageLoad, Err: err}
	}
	idx, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		return ModelCatalog{}, &ValidationFailure{Stage: StageSourceIndex, Err: err}
	}
	worldFP, sourceFP, err := workspaceProjectionFingerprints(loaded.World, idx)
	if err != nil {
		return ModelCatalog{}, err
	}
	return CatalogWorld(loaded.World, idx, namespace, worldFP, sourceFP), nil
}

// CatalogWorld is a transport-independent projection for CLI today and other
// targets later. It does not synthesize source anchors or writable locations.
func CatalogWorld(world ir.World, idx sourceindex.Index, namespace, worldFP, sourceFP string) ModelCatalog {
	refCount := map[sourceindex.Subject]int{}
	for _, ref := range idx.References {
		refCount[ref.Subject]++
	}
	out := ModelCatalog{
		SchemaVersion:          "tei.operations.model-catalog/v0",
		Effect:                 "read-only",
		NamespaceFilter:        namespace,
		WorldFingerprint:       worldFP,
		SourceIndexFingerprint: sourceFP,
		ModelRevision:          sourceindex.ModelRevision(idx),
		Entries:                []ModelCatalogEntry{},
		AuthorityLimits: []string{
			"catalog-is-a-derived-read-only-projection",
			"source-references-are-navigation-not-edit-or-acceptance-authority",
			"model-revision-does-not-authorize-source-writes",
			"external-xead-xeaf-sources-are-not-canonical-tei-models",
		},
	}
	seen := map[sourceindex.Subject]int{}
	include := true
	add := func(kind, id, symbol, detail string, parent *sourceindex.Subject) {
		if id == "" {
			return
		}
		subject := sourceindex.Subject{Kind: kind, ID: id}
		seen[subject]++
		if !include {
			return
		}
		status := "not-indexed"
		if refCount[subject] > 0 {
			status = "indexed"
		}
		out.Entries = append(out.Entries, ModelCatalogEntry{
			Subject: subject, Parent: parent, Symbol: symbol,
			Detail: detail, SourceStatus: status,
		})
	}
	parentOf := func(kind, id string) *sourceindex.Subject {
		p := sourceindex.Subject{Kind: kind, ID: id}
		return &p
	}
	for _, ns := range world.Semantics {
		include = namespace == "" || string(ns.ID) == namespace
		nsID := string(ns.ID)
		add("semantic", nsID, "", "", nil)
		np := parentOf("semantic", nsID)
		for _, typ := range ns.Types {
			add("semantic-type", string(typ.ID), typ.Symbol, string(typ.Kind), np)
		}
		for _, data := range ns.Data {
			dataID := string(data.ID)
			add("semantic-data", dataID, data.Symbol, "", np)
			dp := parentOf("semantic-data", dataID)
			for _, attr := range data.Attributes {
				add("semantic-attribute", string(attr.ID), attr.Symbol, string(attr.TypeRef), dp)
			}
			for _, rel := range data.Relations {
				add("semantic-relation", string(rel.ID), rel.Symbol, string(rel.Target), dp)
			}
			for _, key := range data.CandidateKeys {
				add("semantic-candidate-key", string(key.ID), key.Symbol, "", dp)
			}
			for _, dep := range data.FunctionalDependencies {
				add("semantic-functional-dependency", string(dep.ID), dep.Symbol, "", dp)
			}
		}
		for _, cap := range ns.Capabilities {
			add("semantic-capability", string(cap.ID), cap.Symbol, "", np)
		}
		for _, validation := range ns.Validations {
			add("semantic-validation", string(validation.ID), validation.Symbol, string(validation.Kind), np)
		}
	}
	// Applications are workspace-level and not members of one semantic
	// namespace; exclude them when a namespace filter is requested.
	include = namespace == ""
	for _, app := range world.Applications {
		appID := string(app.ID)
		add("application", appID, "", "", nil)
		ap := parentOf("application", appID)
		for _, pattern := range app.Patterns {
			add("application-pattern", string(pattern.ID), "", string(pattern.Kind), ap)
		}
	}
	for _, query := range world.Queries {
		add("query", string(query.ID), "", "", nil)
	}
	for i := range out.Entries {
		if seen[out.Entries[i].Subject] > 1 {
			out.Entries[i].Ambiguous = true
		}
	}
	sort.SliceStable(out.Entries, func(i, j int) bool {
		a, b := out.Entries[i], out.Entries[j]
		if a.Subject.Kind != b.Subject.Kind {
			return a.Subject.Kind < b.Subject.Kind
		}
		if a.Subject.ID != b.Subject.ID {
			return a.Subject.ID < b.Subject.ID
		}
		if a.Symbol != b.Symbol {
			return a.Symbol < b.Symbol
		}
		return a.Detail < b.Detail
	})
	return out
}
