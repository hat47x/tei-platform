package operations

import (
	"fmt"
	"sort"

	"tei/internal/ir"
	"tei/internal/sourceindex"
	"tei/internal/validation"
	"tei/internal/workspace"
)

// ModelOverview is the table of contents of one Workspace's model: one row per
// top-level section (semantic namespace, application, query) with the shape of
// everything under it and the problems found anywhere under it (RQ-036,
// design §19.8 G5 and §19.8.2 item 1).
//
// A problem deep inside a section is counted on that section, so a person or
// an AI reading only the overview can see where to look without opening every
// detail. Totals cover every section, including the ones cut by the limit, so
// a shortened list never hides a problem. It is a read-only projection of the
// same load as model catalog; it is not a verdict, an acceptance or an edit
// authority.
type ModelOverview struct {
	SchemaVersion          string                       `json:"schemaVersion"`
	Effect                 string                       `json:"effect"`
	WorldFingerprint       string                       `json:"worldFingerprint"`
	SourceIndexFingerprint string                       `json:"sourceIndexFingerprint"`
	ModelRevision          string                       `json:"modelRevision"`
	Workspace              *workspace.WorkspaceIdentity `json:"workspace,omitempty"`
	Validation             CatalogValidation            `json:"validation"`
	Limit                  int                          `json:"limit"`
	SectionTotal           int                          `json:"sectionTotal"`
	SectionsOmitted        int                          `json:"sectionsOmitted"`
	// SectionScope says what a section can contain. A reader must not take
	// "sectionTotal: 8" to mean the workspace holds eight things: the sections
	// mirror the model catalog, and the catalog does not list every layer
	// (#4128).
	SectionScope ModelSectionScope      `json:"sectionScope"`
	Sections     []ModelOverviewSection `json:"sections"`
	// Totals sums Attention over every section, listed or omitted.
	Totals ModelAttention `json:"totals"`
	// Unattributed holds the findings that belong to no catalog element: a
	// diagnostic without a subject, a diagnostic about a kind the catalog does
	// not list (for example a period or a toolchain), and Source root findings.
	Unattributed    ModelUnattributed     `json:"unattributed"`
	Coverage        ModelLocationCoverage `json:"coverage"`
	AuthorityLimits []string              `json:"authorityLimits"`
}

// ModelOverviewSection is one top-level element and its whole subtree.
type ModelOverviewSection struct {
	Entry       ModelCatalogEntry `json:"entry"`
	Descendants int               `json:"descendants"`
	KindCounts  []ModelKindCount  `json:"kindCounts"`
	Attention   ModelAttention    `json:"attention"`
}

// ModelSectionScope states the rule that selects a section and which declared
// kinds therefore have none. Both lists are derived, so the statement cannot
// drift away from what the overview actually does.
type ModelSectionScope struct {
	// RootsAre names the selection rule for a section.
	RootsAre string `json:"rootsAre"`
	// CoveredKinds are the subject kinds the model catalog lists. A kind here
	// can appear inside a section.
	CoveredKinds []string `json:"coveredKinds"`
	// NotCoveredKinds are kinds this workspace declares that the catalog does
	// not list. They have no section and no attention count.
	NotCoveredKinds []string `json:"notCoveredKinds"`
	// ReadableWithoutSection are the kinds in NotCoveredKinds that `inspect`
	// can still resolve by name. The distinction matters: a designer who has
	// just declared a persistence mapping sees its kind in NotCoveredKinds and
	// cannot tell whether the declaration landed or the kind is simply never
	// sectioned. With this list they can read the artefact back immediately
	// (#4128).
	ReadableWithoutSection []string `json:"readableWithoutSection"`
}

// declaredLayerKinds names the World collections that hold author-written
// declarations, in report order. The catalog does not list several of them, and
// the overview must say so rather than let a section count imply otherwise.
var declaredLayerKinds = []struct {
	kind  string
	count func(ir.World) int
}{
	{"plugin", func(w ir.World) int { return len(w.Plugins) }},
	{"bundle", func(w ir.World) int { return len(w.Bundles) }},
	{"database", func(w ir.World) int { return len(w.Databases) }},
	{"persistence-mapping", func(w ir.World) int { return len(w.Persistence) }},
	{"migration", func(w ir.World) int { return len(w.Migrations) }},
	{"migration-provider", func(w ir.World) int { return len(w.MigrationProviders) }},
	{"application-release", func(w ir.World) int { return len(w.ApplicationReleases) }},
	{"database-release", func(w ir.World) int { return len(w.DatabaseReleases) }},
	{"solution-release", func(w ir.World) int { return len(w.SolutionReleases) }},
	{"environment", func(w ir.World) int { return len(w.Environments) }},
	{"custom-go-binding", func(w ir.World) int { return len(w.Bindings) }},
	{"header-detail-contract-query-binding", func(w ir.World) int { return len(w.HeaderDetailContractQueryBindings) }},
	{"contract-query-executor-binding", func(w ir.World) int { return len(w.ContractQueryExecutorBindings) }},
	{"contract-query-sql-realization", func(w ir.World) int { return len(w.ContractQuerySQLRealizations) }},
}

// sectionScope derives the scope statement from the catalog and the World.
func sectionScope(catalog ModelCatalog, world ir.World) ModelSectionScope {
	covered := map[string]bool{}
	var coveredKinds []string
	for _, entry := range catalog.Entries {
		if covered[entry.Subject.Kind] {
			continue
		}
		covered[entry.Subject.Kind] = true
		coveredKinds = append(coveredKinds, entry.Subject.Kind)
	}
	sort.Strings(coveredKinds)

	notCovered := []string{}
	for _, layer := range declaredLayerKinds {
		if covered[layer.kind] || layer.count(world) == 0 {
			continue
		}
		notCovered = append(notCovered, layer.kind)
	}
	// A not-covered kind is readable when its own operation accepts it. Reading
	// the same list the CLI advertises keeps the two answers together.
	readable := []string{}
	for _, kind := range notCovered {
		if supportedInspectionKind(kind) {
			readable = append(readable, kind)
		}
	}
	return ModelSectionScope{
		RootsAre:               "catalog-root-entries",
		CoveredKinds:           coveredKinds,
		NotCoveredKinds:        notCovered,
		ReadableWithoutSection: readable,
	}
}

// ModelAttention counts what needs a look in one subtree. Each counter is a
// different kind of problem and is kept apart; NeedsAttention is true when
// any counter is nonzero.
type ModelAttention struct {
	NeedsAttention bool `json:"needsAttention"`
	// ValidationErrors and ValidationWarnings are diagnostics from the same
	// validation run the catalog verdict counts, attributed by subject.
	ValidationErrors   int `json:"validationErrors"`
	ValidationWarnings int `json:"validationWarnings"`
	// UnresolvedReferences are declared references whose target ID names
	// nothing of the expected kind (the resolved:false links of model locate).
	UnresolvedReferences       int              `json:"unresolvedReferences"`
	UnresolvedReferencesByRole []ModelKindCount `json:"unresolvedReferencesByRole"`
	// AmbiguousElements are elements declared more than once.
	AmbiguousElements int `json:"ambiguousElements"`
	// WithoutSourceReference counts Workspace-declared (non-prelude) elements
	// the Source Index cannot point at. They can be read but not traced back
	// to a file position.
	WithoutSourceReference int `json:"withoutSourceReference"`
}

type ModelUnattributed struct {
	ValidationErrors   int              `json:"validationErrors"`
	ValidationWarnings int              `json:"validationWarnings"`
	DiagnosticsByKind  []ModelKindCount `json:"diagnosticsByKind"`
	SourceRootErrors   int              `json:"sourceRootErrors"`
	SourceRootWarnings int              `json:"sourceRootWarnings"`
}

const (
	OverviewDefaultLimit = 100
	OverviewMaxLimit     = 1000
)

// OverviewWorkspace loads the Workspace once and returns its overview.
func OverviewWorkspace(root string, limit int) (ModelOverview, error) {
	if limit == 0 {
		limit = OverviewDefaultLimit
	}
	if limit < 1 || limit > OverviewMaxLimit {
		return ModelOverview{}, &CatalogError{Code: LocationCodeInvalidLimit,
			Message: fmt.Sprintf("limit must be between 1 and %d", OverviewMaxLimit)}
	}
	read, err := loadCatalogRead(root)
	if err != nil {
		return ModelOverview{}, err
	}
	return OverviewFromRead(read.Catalog, read.World, read.Diagnostics, read.SourceRootFindings, limit), nil
}

// OverviewFromRead is the transport-independent projection over one load.
func OverviewFromRead(catalog ModelCatalog, world ir.World, diagnostics []validation.Diagnostic,
	rootFindings []workspace.WorkspaceSourceRootFinding, limit int) ModelOverview {
	tree := newCatalogTree(catalog, world, diagnostics)
	out := ModelOverview{
		SchemaVersion:          "tei.operations.model-overview/v0",
		Effect:                 "read-only",
		WorldFingerprint:       catalog.WorldFingerprint,
		SourceIndexFingerprint: catalog.SourceIndexFingerprint,
		ModelRevision:          catalog.ModelRevision,
		Workspace:              catalog.Workspace,
		Validation:             catalog.Validation,
		Limit:                  limit,
		SectionScope:           sectionScope(catalog, world),
		Sections:               []ModelOverviewSection{},
		Coverage: ModelLocationCoverage{
			Derived: []string{
				"validation-diagnostics-by-subject",
				"unresolved-declared-references",
				"ambiguous-declarations",
				"elements-without-source-reference",
			},
			NotDerived: []string{
				"customization-and-extension-slots",
				"business-event-flow-gaps",
				"lifecycle-state-transition-gaps",
				"implementation-binding",
				"consumer-disclosure-filter",
			},
		},
		AuthorityLimits: []string{
			"overview-is-a-derived-read-only-projection",
			"attention-counts-are-navigation-not-a-validation-verdict",
			"no-consumer-disclosure-is-applied-the-visibility-equals-model-catalog",
		},
	}
	var roots []ModelCatalogEntry
	for _, entry := range catalog.Entries {
		if entry.Parent == nil {
			roots = append(roots, entry)
		}
	}
	out.SectionTotal = len(roots)
	var totals []ModelAttention
	for i, root := range roots {
		attention := tree.subtreeAttention(root.Subject)
		totals = append(totals, attention)
		if i >= limit {
			continue
		}
		descendants := tree.descendants(root.Subject)
		counts := map[string]int{}
		for _, d := range descendants {
			counts[d.Kind]++
		}
		out.Sections = append(out.Sections, ModelOverviewSection{
			Entry:       root,
			Descendants: len(descendants),
			KindCounts:  sortedKindCounts(counts),
			Attention:   attention,
		})
	}
	out.SectionsOmitted = len(roots) - len(out.Sections)
	out.Totals = sumAttention(totals)

	unattributedKinds := map[string]int{}
	for _, diagnostic := range diagnostics {
		if diagnostic.Subject != nil && tree.known(*diagnostic.Subject) {
			continue
		}
		kind := "(none)"
		if diagnostic.Subject != nil {
			kind = diagnostic.Subject.Kind
		}
		unattributedKinds[kind]++
		if diagnostic.Severity == validation.Error {
			out.Unattributed.ValidationErrors++
		} else {
			out.Unattributed.ValidationWarnings++
		}
	}
	out.Unattributed.DiagnosticsByKind = sortedKindCounts(unattributedKinds)
	for _, finding := range rootFindings {
		switch finding.Severity {
		case "error":
			out.Unattributed.SourceRootErrors++
		case "warning":
			out.Unattributed.SourceRootWarnings++
		}
	}
	return out
}

// catalogTree indexes one catalog for subtree questions.
type catalogTree struct {
	entries    map[sourceindex.Subject]ModelCatalogEntry
	children   map[sourceindex.Subject][]sourceindex.Subject
	ambiguous  map[sourceindex.Subject]bool
	errors     map[sourceindex.Subject]int
	warnings   map[sourceindex.Subject]int
	unresolved map[sourceindex.Subject]map[string]int
}

func newCatalogTree(catalog ModelCatalog, world ir.World, diagnostics []validation.Diagnostic) catalogTree {
	t := catalogTree{
		entries:    map[sourceindex.Subject]ModelCatalogEntry{},
		children:   map[sourceindex.Subject][]sourceindex.Subject{},
		ambiguous:  map[sourceindex.Subject]bool{},
		errors:     map[sourceindex.Subject]int{},
		warnings:   map[sourceindex.Subject]int{},
		unresolved: map[sourceindex.Subject]map[string]int{},
	}
	for _, entry := range catalog.Entries {
		if _, seen := t.entries[entry.Subject]; seen {
			// A duplicated subject is counted once as an element and marked
			// ambiguous; its second occurrence must not double the subtree.
			t.ambiguous[entry.Subject] = true
			continue
		}
		t.entries[entry.Subject] = entry
		if entry.Ambiguous {
			t.ambiguous[entry.Subject] = true
		}
		if entry.Parent != nil {
			t.children[*entry.Parent] = append(t.children[*entry.Parent], entry.Subject)
		}
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Subject == nil || !t.known(*diagnostic.Subject) {
			continue
		}
		if diagnostic.Severity == validation.Error {
			t.errors[*diagnostic.Subject]++
		} else {
			t.warnings[*diagnostic.Subject]++
		}
	}
	for _, link := range declaredLinks(world, t.entries) {
		if link.Resolved {
			continue
		}
		if t.unresolved[link.From] == nil {
			t.unresolved[link.From] = map[string]int{}
		}
		t.unresolved[link.From][link.Role]++
	}
	return t
}

func (t catalogTree) known(s sourceindex.Subject) bool {
	_, ok := t.entries[s]
	return ok
}

// descendants lists every element under root, not including root.
func (t catalogTree) descendants(root sourceindex.Subject) []sourceindex.Subject {
	var out []sourceindex.Subject
	visited := map[sourceindex.Subject]bool{root: true}
	stack := append([]sourceindex.Subject{}, t.children[root]...)
	for len(stack) > 0 {
		next := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if visited[next] {
			continue
		}
		visited[next] = true
		out = append(out, next)
		stack = append(stack, t.children[next]...)
	}
	return out
}

// subtreeAttention counts the problems on root and everything under it.
func (t catalogTree) subtreeAttention(root sourceindex.Subject) ModelAttention {
	var a ModelAttention
	roles := map[string]int{}
	for _, s := range append([]sourceindex.Subject{root}, t.descendants(root)...) {
		a.ValidationErrors += t.errors[s]
		a.ValidationWarnings += t.warnings[s]
		for role, n := range t.unresolved[s] {
			roles[role] += n
			a.UnresolvedReferences += n
		}
		if t.ambiguous[s] {
			a.AmbiguousElements++
		}
		if entry := t.entries[s]; !entry.Prelude && entry.SourceStatus != "indexed" {
			a.WithoutSourceReference++
		}
	}
	a.UnresolvedReferencesByRole = sortedKindCounts(roles)
	a.NeedsAttention = a.ValidationErrors+a.ValidationWarnings+a.UnresolvedReferences+
		a.AmbiguousElements+a.WithoutSourceReference > 0
	return a
}

func sumAttention(values []ModelAttention) ModelAttention {
	var sum ModelAttention
	roles := map[string]int{}
	for _, v := range values {
		sum.ValidationErrors += v.ValidationErrors
		sum.ValidationWarnings += v.ValidationWarnings
		sum.UnresolvedReferences += v.UnresolvedReferences
		sum.AmbiguousElements += v.AmbiguousElements
		sum.WithoutSourceReference += v.WithoutSourceReference
		for _, kc := range v.UnresolvedReferencesByRole {
			roles[kc.Kind] += kc.Count
		}
		sum.NeedsAttention = sum.NeedsAttention || v.NeedsAttention
	}
	sum.UnresolvedReferencesByRole = sortedKindCounts(roles)
	return sum
}

func sortedKindCounts(counts map[string]int) []ModelKindCount {
	out := make([]ModelKindCount, 0, len(counts))
	for kind, count := range counts {
		out = append(out, ModelKindCount{Kind: kind, Count: count})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Kind < out[j].Kind })
	return out
}
