package operations

import (
	"fmt"
	"sort"
	"strings"

	"tei/internal/ir"
	"tei/internal/sourceindex"
	"tei/internal/stdlib"
	"tei/internal/validation"
	"tei/internal/workspace"
)

// ModelCatalog is a read-only, target-neutral Modeler projection. It is a
// navigation aid, not a second canonical model, edit authority or acceptance.
type ModelCatalog struct {
	SchemaVersion   string `json:"schemaVersion"`
	Effect          string `json:"effect"`
	NamespaceFilter string `json:"namespaceFilter,omitempty"`
	// NamespaceMatched reports whether the requested namespace exists in the
	// loaded World. It is always true when no filter was requested. A caller
	// must not read an empty Entries list as "the namespace declares nothing":
	// a misspelled namespace and a declared namespace with no entries are
	// different answers.
	NamespaceMatched bool `json:"namespaceMatched"`
	// NamespaceMatchKind names what the filter selected: "all" (no filter),
	// "namespace", "application", "query", or "unknown". An application or
	// query ID is a real workspace subject, but it is not a semantic
	// namespace and therefore selects no entries through -namespace.
	NamespaceMatchKind     string `json:"namespaceMatchKind,omitempty"`
	WorldFingerprint       string `json:"worldFingerprint"`
	SourceIndexFingerprint string `json:"sourceIndexFingerprint"`
	// Workspace names the Workspace this catalog was read from (#3181). The
	// same Canonical ID may exist in several Workspaces with different meaning,
	// and worldFingerprint identifies content rather than the Workspace. It is
	// absent when the caller supplied a World directly (CatalogWorld).
	Workspace *workspace.WorkspaceIdentity `json:"workspace,omitempty"`
	// ModelRevision is the revision of the managed Source set, not of the
	// loaded World. An empty Workspace (no managed Source read) reports the
	// SHA-256 of the empty input while still listing the standard prelude,
	// so this value must not be read as "the model is empty" (#2822).
	ModelRevision string `json:"modelRevision"`
	// DeclarationCounts separates Workspace-authored declarations from the
	// standard semantic prelude, which is always present (#2822).
	DeclarationCounts DeclarationCounts   `json:"declarationCounts"`
	Entries           []ModelCatalogEntry `json:"entries"`
	// Validation reports whether this Workspace passes the same checks
	// `validate -workspace ... -json` runs. The catalog itself is a
	// navigation projection and still answers for a Workspace with findings,
	// so a caller that needs "is this model sound" must read this field
	// instead of inferring it from ok or the fingerprints (#3674).
	Validation      CatalogValidation `json:"validation"`
	AuthorityLimits []string          `json:"authorityLimits"`
}

// CatalogValidation is the validation verdict for the Workspace the catalog
// was read from. OK is true only when no diagnostic has error severity and no
// source-root finding is an error, which is the same rule validate uses.
// ErrorCount and WarningCount are counts, not the diagnostics themselves:
// `validate -json` returns the detail, and duplicating it here would create a
// second place to read the same findings.
type CatalogValidation struct {
	OK           bool `json:"ok"`
	ErrorCount   int  `json:"errorCount"`
	WarningCount int  `json:"warningCount"`
}

// stdlibNamespaceID is the in-process standard semantic namespace.
const stdlibNamespaceID = "std"

// isPreludeType reports whether id is one of the types the prelude itself
// declares. A std.* name the prelude does not declare belongs to whoever wrote
// it.
func isPreludeType(id string) bool {
	for _, typ := range stdlib.SemanticNamespace().Types {
		if string(typ.ID) == id {
			return true
		}
	}
	return false
}

// preludeSubject reports whether a catalog subject belongs to the in-process
// standard semantic prelude. The prelude is the std namespace and every
// subject whose ID is std or starts with std.; the Loader always adds it, so
// no source can own it. The check is by ID because that is the identity the
// prelude uses, not by Source Status (the prelude is never indexed).
func preludeSubject(s sourceindex.Subject) bool {
	// Judge by identity, not by the name. A Workspace may declare its own type
	// inside the std namespace (for example `type email from std.text`); that
	// declaration is the caller's, so it must not carry the prelude mark. The
	// contract defines prelude:true as "the in-process standard meaning, not
	// something the Workspace declared" (#3347, #3851).
	//
	// standardSemanticSubject is the rule that already answers this correctly
	// for duplicate detection; this reuses it so the two cannot disagree.
	if s.Kind == "semantic" {
		return s.ID == stdlibNamespaceID
	}
	return isPreludeType(s.ID)
}

type ModelCatalogEntry struct {
	Subject sourceindex.Subject `json:"subject"`
	// Prelude is true for the in-process standard semantic prelude (the std namespace
	// and its types). It is always loaded, so its presence does not mean the Workspace
	// declared anything (#2822). Without this flag a caller cannot tell a std entry from
	// a source-authored one without matching IDs (#3347).
	Prelude      bool                 `json:"prelude,omitempty"`
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
	catalog, _, err := catalogWorkspaceLoaded(root, namespace)
	return catalog, err
}

// catalogRead is one Workspace load. A caller that needs the catalog together
// with the World or the diagnostics (model locate, model overview) reads them
// from here instead of loading twice and risking two Source snapshots.
//
// catalogRead is one Workspace load with everything the catalog, location and
// overview projections derive from it. Diagnostics are the ones the catalog's
// validation verdict counted, so a roll-up and the verdict cannot disagree.
type catalogRead struct {
	Catalog            ModelCatalog
	World              ir.World
	Diagnostics        []validation.Diagnostic
	SourceRootFindings []workspace.WorkspaceSourceRootFinding
}

func loadCatalogRead(root string) (catalogRead, error) {
	if strings.TrimSpace(root) == "" || root != strings.TrimSpace(root) {
		return catalogRead{}, fmt.Errorf("workspace path must be nonempty and exact")
	}
	catalog, loaded, err := catalogWorkspaceLoaded(root, "")
	if err != nil {
		return catalogRead{}, err
	}
	return catalogRead{
		Catalog:            catalog,
		World:              loaded.World,
		Diagnostics:        validation.Validate(loaded.World),
		SourceRootFindings: loaded.SourceRootFindings,
	}, nil
}

func catalogWorkspaceLoaded(root, namespace string) (ModelCatalog, workspace.Result, error) {
	loaded, err := workspace.Load(root)
	if err != nil {
		return ModelCatalog{}, workspace.Result{}, &ValidationFailure{Stage: StageLoad, Err: err}
	}
	idx, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		return ModelCatalog{}, workspace.Result{}, &ValidationFailure{Stage: StageSourceIndex, Err: err}
	}
	worldFP, sourceFP, err := workspaceProjectionFingerprints(loaded.World, idx)
	if err != nil {
		return ModelCatalog{}, workspace.Result{}, err
	}
	catalog := CatalogWorld(loaded.World, idx, namespace, worldFP, sourceFP)
	identity := loaded.Identity()
	catalog.Workspace = &identity
	catalog.Validation = catalogValidation(loaded)
	return catalog, loaded, nil
}

// catalogValidation applies validate's rule to the same loaded Workspace, so a
// caller that only reads the catalog can tell a sound model from one with
// findings (#3674). It reports counts, not the findings: `validate -json`
// returns those, and a second copy would be two places to read one fact.
func catalogValidation(loaded workspace.Result) CatalogValidation {
	verdict := CatalogValidation{OK: true}
	for _, finding := range loaded.SourceRootFindings {
		switch finding.Severity {
		case "error":
			verdict.OK = false
			verdict.ErrorCount++
		case "warning":
			verdict.WarningCount++
		}
	}
	for _, diagnostic := range validation.Validate(loaded.World) {
		if diagnostic.Severity == validation.Error {
			verdict.OK = false
			verdict.ErrorCount++
			continue
		}
		verdict.WarningCount++
	}
	return verdict
}

// namespaceMatch reports whether a namespace filter names something in the
// World and what kind of subject it named. An empty filter matches everything.
func namespaceMatch(world ir.World, namespace string) (bool, string) {
	if namespace == "" {
		return true, "all"
	}
	for _, ns := range world.Semantics {
		if string(ns.ID) == namespace {
			return true, "namespace"
		}
	}
	for _, app := range world.Applications {
		if string(app.ID) == namespace {
			return false, "application"
		}
	}
	for _, query := range world.Queries {
		if string(query.ID) == namespace {
			return false, "query"
		}
	}
	return false, "unknown"
}

// CatalogError is a typed read failure that is not a Workspace load or Source
// Index failure. Each adapter maps it to the same string code, so a client can
// compare the CLI and MCP answers instead of reading two different failures
// (#3420).
type CatalogError struct {
	Code    string
	Message string
}

func (e *CatalogError) Error() string {
	if e == nil {
		return "catalog failure"
	}
	return e.Message
}

// CatalogCodeNamespaceNotFound is the code for a -namespace that names no
// semantic namespace in this Workspace.
const CatalogCodeNamespaceNotFound = "NAMESPACE_NOT_FOUND"

// NamespaceFilterError reports the filtered read as a failure. An empty entry
// list is then never read as "the namespace declares nothing": a misspelled
// namespace and a declared namespace with no entries stay different answers.
func (c ModelCatalog) NamespaceFilterError() *CatalogError {
	if c.NamespaceMatched {
		return nil
	}
	message := "namespace " + c.NamespaceFilter + " is not a semantic namespace in this workspace"
	switch c.NamespaceMatchKind {
	case "application":
		message = "application " + c.NamespaceFilter + " is not a semantic namespace; omit -namespace or use a namespace ID"
	case "query":
		message = "query " + c.NamespaceFilter + " is not a semantic namespace; omit -namespace or use a namespace ID"
	}
	return &CatalogError{Code: CatalogCodeNamespaceNotFound, Message: message}
}

// CatalogWorld is a transport-independent projection for CLI today and other
// targets later. It does not synthesize source anchors or writable locations.
func CatalogWorld(world ir.World, idx sourceindex.Index, namespace, worldFP, sourceFP string) ModelCatalog {
	refCount := map[sourceindex.Subject]int{}
	for _, ref := range idx.References {
		refCount[ref.Subject]++
	}
	matched, matchKind := namespaceMatch(world, namespace)
	out := ModelCatalog{
		SchemaVersion:          "tei.operations.model-catalog/v0",
		Effect:                 "read-only",
		NamespaceFilter:        namespace,
		NamespaceMatched:       matched,
		NamespaceMatchKind:     matchKind,
		WorldFingerprint:       worldFP,
		SourceIndexFingerprint: sourceFP,
		ModelRevision:          sourceindex.ModelRevision(idx),
		DeclarationCounts:      CountDeclarations(world),
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
		// The standard prelude is always loaded; marking it keeps a caller from
		// reading its presence as a Workspace declaration (#3347).
		out.Entries[i].Prelude = preludeSubject(out.Entries[i].Subject)
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
