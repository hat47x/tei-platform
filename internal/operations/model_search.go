package operations

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// ModelSearch answers "where does this text appear in the Workspace's Sources,
// and which element declares it" (RQ-036, design §19.8). It exists because the
// catalog lists every element but searches no text: a caller who knew a name
// fragment had to page through all entries, guess the subject kind, and call
// `inspect` per candidate, and still learned nothing about a mention inside a
// declaration body. A match without a location is not usable by an agent that
// must then navigate there, so every hit carries the evidence for it: the
// subject, the Source artifact, the line and the line's text.
//
// It is derived from the same load as ModelCatalog and carries the same
// fingerprints, so a hit can be tied to one model version. It is a navigation
// aid: it accepts no meaning, grants no edit authority and changes no Source.
type ModelSearch struct {
	SchemaVersion string `json:"schemaVersion"`
	Effect        string `json:"effect"`
	// SnapshotFingerprint binds WorldFingerprint and SourceIndexFingerprint to
	// one value a caller can compare with the same field from `model catalog`,
	// `model locate`, `model overview` and `inspect`.
	SnapshotFingerprint    string                       `json:"snapshotFingerprint"`
	WorldFingerprint       string                       `json:"worldFingerprint"`
	SourceIndexFingerprint string                       `json:"sourceIndexFingerprint"`
	ModelRevision          string                       `json:"modelRevision"`
	Workspace              *workspace.WorkspaceIdentity `json:"workspace,omitempty"`
	Validation             CatalogValidation            `json:"validation"`
	Query                  ModelSearchQuery             `json:"query"`
	// Total is the number of matches found in the searched scope and Omitted is
	// Total-len(Hits), so a page cut by -limit is never read as the whole
	// answer. Offset is the page this answer actually starts at and
	// RequestedOffset keeps the caller's value when it was past the last match,
	// so a clamped page is observable instead of looking like a normal one
	// (#1177). NextOffset names where the next page starts, and is absent on
	// the last one.
	Total           int              `json:"total"`
	Omitted         int              `json:"omitted"`
	Offset          int              `json:"offset"`
	RequestedOffset int              `json:"requestedOffset,omitempty"`
	Limit           int              `json:"limit"`
	Hits            []ModelSearchHit `json:"hits"`
	KindCounts      []ModelKindCount `json:"kindCounts"`
	NextOffset      *int             `json:"nextOffset,omitempty"`
	// Truncated reports that this answer is partial because the search stopped
	// at its documented work bound (SearchMaxFiles or SearchMaxBytes) instead of
	// reading every Source. It is not the paging cut, which Omitted and
	// NextOffset already describe: a caller that sees truncated must not read
	// the hits, or an empty hit list, as the whole Workspace's answer.
	Truncated       bool                `json:"truncated"`
	Coverage        ModelSearchCoverage `json:"coverage"`
	AuthorityLimits []string            `json:"authorityLimits"`
}

// ModelSearchQuery is the request as it was applied, echoed so a reader can
// tell which term and which kind produced these hits.
type ModelSearchQuery struct {
	// Term is matched literally: the same characters, case-insensitively,
	// anywhere inside a Source line. It is required unless Kind is given;
	// without it every line of the requested kind is listed.
	Term string `json:"term,omitempty"`
	// Kind restricts the search to one subject kind from
	// operations.InspectionSubjectKinds. Empty means every kind.
	Kind string `json:"kind,omitempty"`
}

// ModelSearchHit is one matching line. Subject is the element the line is
// attributed to, ArtifactPath is the workspace-relative Source, Line is the
// 1-based line and Text is that line as written, without its line terminator.
// The location is what makes the hit evidence rather than an assertion.
type ModelSearchHit struct {
	Subject      sourceindex.Subject `json:"subject"`
	ArtifactPath string              `json:"artifactPath"`
	Line         int                 `json:"line"`
	Text         string              `json:"text"`
}

// ModelSearchCoverage states which text this search derives and which it does
// not. A caller must not read an empty hit list as "this text appears nowhere"
// for a family listed in NotDerived.
type ModelSearchCoverage struct {
	Derived    []string `json:"derived"`
	NotDerived []string `json:"notDerived"`
}

const (
	// SearchCodeInvalidArgument is the code for a query the projection cannot
	// apply: neither a term nor a kind, or a kind outside the closed set.
	SearchCodeInvalidArgument = "INVALID_ARGUMENT"
	// SearchCodeInvalidLimit is the code for a limit outside the accepted range.
	SearchCodeInvalidLimit = "INVALID_LIMIT"
	// SearchCodeInvalidOffset is the code for a negative offset.
	SearchCodeInvalidOffset = "INVALID_OFFSET"
	// SearchCodeSourceUnreadable is the code for a Source the index binds a
	// reference to but that cannot be read back.
	SearchCodeSourceUnreadable = "SOURCE_READ_FAILED"
	// SearchCodeSourceChanged is the code for a Source whose bytes no longer
	// match the fingerprint the index recorded. Answering would report a
	// location the model revision does not describe, so the read fails closed.
	SearchCodeSourceChanged = "SOURCE_ARTIFACT_CHANGED"

	// SearchDefaultLimit bounds a page so the answer stays readable for a person
	// and fits an AI's context; SearchMaxLimit stops a caller from asking for an
	// unbounded page through this command.
	SearchDefaultLimit = 50
	SearchMaxLimit     = 500

	// SearchMaxFiles and SearchMaxBytes bound the work one search performs, so
	// a caller cannot turn a read-only command into an unbounded scan of a
	// large Workspace. Sources are walked in path order, so a search that stops
	// at either bound answers a stable prefix and reports truncated: true. Both
	// bounds are far above a hand-authored Workspace; a Workspace large enough
	// to reach them is searched through narrower -kind queries instead.
	SearchMaxFiles = 200
	SearchMaxBytes = 32 << 20 // 32 MiB of Source text
)

// SearchQuery is the transport-independent form of one search request.
type SearchQuery struct {
	// Term is required unless Kind is given. It must be nonempty and without
	// surrounding whitespace, like every other exact input of this surface.
	Term string
	// Kind restricts the search to one subject kind. Empty means every kind.
	Kind string
}

// SearchBounds bounds the work one search performs. A zero field means the
// documented default, so an in-process caller that wants a smaller scan (a
// test, or a host with its own budget) can pass its own without the defaults
// disappearing.
type SearchBounds struct {
	MaxFiles int
	MaxBytes int
}

func (b SearchBounds) withDefaults() SearchBounds {
	if b.MaxFiles <= 0 {
		b.MaxFiles = SearchMaxFiles
	}
	if b.MaxBytes <= 0 {
		b.MaxBytes = SearchMaxBytes
	}
	return b
}

// SearchWorkspace loads the Workspace once and answers where the query text
// appears. The catalog, the Source Index and the Source bytes are read from
// that single load, so a hit cannot mix two Source snapshots.
func SearchWorkspace(root string, query SearchQuery, limit, offset int) (ModelSearch, error) {
	return searchWorkspace(root, query, limit, offset, SearchBounds{})
}

func searchWorkspace(root string, query SearchQuery, limit, offset int, bounds SearchBounds) (ModelSearch, error) {
	if limit == 0 {
		limit = SearchDefaultLimit
	}
	if limit < 1 || limit > SearchMaxLimit {
		return ModelSearch{}, &CatalogError{Code: SearchCodeInvalidLimit,
			Message: fmt.Sprintf("limit must be between 1 and %d", SearchMaxLimit)}
	}
	if offset < 0 {
		return ModelSearch{}, &CatalogError{Code: SearchCodeInvalidOffset,
			Message: "offset must not be negative"}
	}
	if err := query.validate(); err != nil {
		return ModelSearch{}, &CatalogError{Code: SearchCodeInvalidArgument, Message: err.Error()}
	}
	read, err := loadSearchRead(root)
	if err != nil {
		return ModelSearch{}, err
	}
	result, failure := SearchInSources(read.Catalog, read.Index, read.Sources, query, limit, offset, bounds)
	if failure != nil {
		return ModelSearch{}, failure
	}
	return result, nil
}

// validate applies the exactness rule the CLI also enforces, so a caller that
// reaches the projection directly gets the same answer.
func (q SearchQuery) validate() error {
	if strings.TrimSpace(q.Term) == "" && strings.TrimSpace(q.Kind) == "" {
		return fmt.Errorf("search requires -term, or -kind to list one kind's sources")
	}
	if q.Term != "" && q.Term != strings.TrimSpace(q.Term) {
		return fmt.Errorf("search term must not have surrounding whitespace")
	}
	if q.Kind != "" && !SupportedInspectionKind(q.Kind) {
		return fmt.Errorf("subject kind %s is not one of the kinds this surface resolves", q.Kind)
	}
	return nil
}

// searchRead is one Workspace load with everything the search derives from it:
// the catalog projection, the Source Index and the indexed Source bytes.
type searchRead struct {
	Catalog ModelCatalog
	Index   sourceindex.Index
	Sources map[string][]byte
}

func loadSearchRead(root string) (searchRead, error) {
	if strings.TrimSpace(root) == "" || root != strings.TrimSpace(root) {
		return searchRead{}, fmt.Errorf("workspace path must be nonempty and exact")
	}
	catalog, _, index, err := catalogWorkspaceLoadedWithIndex(root, "")
	if err != nil {
		return searchRead{}, err
	}
	sources, err := readIndexedSources(root, index)
	if err != nil {
		return searchRead{}, err
	}
	return searchRead{Catalog: catalog, Index: index, Sources: sources}, nil
}

// readIndexedSources reads the bytes of every Source the index binds a
// reference to. A hit's artifactPath, line and text are statements about those
// exact bytes, so SearchInSources verifies each Source it searches against the
// fingerprint the index recorded and refuses one that changed: answering from
// different bytes would report a location the model revision does not describe.
func readIndexedSources(root string, index sourceindex.Index) (map[string][]byte, error) {
	needed := map[string]bool{}
	for _, ref := range index.References {
		needed[ref.ArtifactPath] = true
	}
	out := make(map[string][]byte, len(needed))
	for _, artifact := range index.Artifacts {
		if !needed[artifact.Path] {
			continue
		}
		// NormalizeWorkspacePath already refuses an absolute path, a drive
		// letter and any escape from the workspace, so this join stays inside
		// the Workspace the caller named.
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(artifact.Path)))
		if err != nil {
			return nil, &CatalogError{Code: SearchCodeSourceUnreadable,
				Message: fmt.Sprintf("read source %s: %v", artifact.Path, err)}
		}
		out[artifact.Path] = content
	}
	return out, nil
}

// SearchInSources is the transport-independent projection over one load. The
// subjects it searches come from the catalog traversal (ModelCatalog's
// entries), so this is not a second World walk and not a second subject index:
// the catalog says which elements exist, the Source Index says where each one
// is declared, and the Source bytes are the text searched.
func SearchInSources(catalog ModelCatalog, index sourceindex.Index, sources map[string][]byte,
	query SearchQuery, limit, offset int, bounds SearchBounds) (ModelSearch, *CatalogError) {
	if limit < 1 || limit > SearchMaxLimit {
		return ModelSearch{}, &CatalogError{Code: SearchCodeInvalidLimit,
			Message: fmt.Sprintf("limit must be between 1 and %d", SearchMaxLimit)}
	}
	if offset < 0 {
		return ModelSearch{}, &CatalogError{Code: SearchCodeInvalidOffset,
			Message: "offset must not be negative"}
	}
	if err := query.validate(); err != nil {
		return ModelSearch{}, &CatalogError{Code: SearchCodeInvalidArgument, Message: err.Error()}
	}
	bounds = bounds.withDefaults()
	out := ModelSearch{
		SchemaVersion:          "tei.operations.model-search/v0",
		Effect:                 "read-only",
		SnapshotFingerprint:    InspectionSnapshotFingerprint(catalog.WorldFingerprint, catalog.SourceIndexFingerprint),
		WorldFingerprint:       catalog.WorldFingerprint,
		SourceIndexFingerprint: catalog.SourceIndexFingerprint,
		ModelRevision:          catalog.ModelRevision,
		Workspace:              catalog.Workspace,
		Validation:             catalog.Validation,
		Query: ModelSearchQuery{
			Term: query.Term,
			Kind: query.Kind,
		},
		Hits:     []ModelSearchHit{},
		Limit:    limit,
		Coverage: searchCoverage(),
		AuthorityLimits: []string{
			"search-is-a-derived-read-only-projection",
			"a-line-is-attributed-to-the-nearest-preceding-declaration-among-the-searched-kinds",
			"matches-are-navigation-not-semantic-acceptance-or-edit-authority",
			"truncated-means-the-work-bound-stopped-the-search-early",
		},
	}
	// The subjects come from the catalog traversal; the index only locates
	// them. A subject the catalog does not list (a persistence index, a
	// contract query) is therefore not searched: its kind is not one `inspect`
	// resolves, so a hit for it would be a dead end.
	subjects := map[sourceindex.Subject]bool{}
	for _, entry := range catalog.Entries {
		if query.Kind != "" && entry.Subject.Kind != query.Kind {
			continue
		}
		subjects[entry.Subject] = true
	}
	byArtifact := map[string][]sourceindex.Reference{}
	for _, ref := range index.References {
		if !subjects[ref.Subject] {
			continue
		}
		byArtifact[ref.ArtifactPath] = append(byArtifact[ref.ArtifactPath], ref)
	}
	paths := make([]string, 0, len(byArtifact))
	for path := range byArtifact {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	// The bytes are the evidence, so each Source is checked against the
	// fingerprint the index recorded before any line of it is answered.
	fingerprints := map[string]string{}
	for _, artifact := range index.Artifacts {
		fingerprints[artifact.Path] = artifact.Fingerprint
	}

	term := strings.ToLower(query.Term)
	var matches []ModelSearchHit
	counts := map[string]int{}
	scanned := 0
	scannedBytes := 0
	for _, path := range paths {
		// Path order makes the scan a stable prefix, so a search stopped by the
		// work bound answers the same Sources every time.
		if scanned >= bounds.MaxFiles || scannedBytes >= bounds.MaxBytes {
			out.Truncated = true
			break
		}
		content, ok := sources[path]
		if !ok {
			return ModelSearch{}, &CatalogError{Code: SearchCodeSourceUnreadable,
				Message: fmt.Sprintf("source %s is indexed but was not read", path)}
		}
		if got := sourceindex.Fingerprint(content); got != fingerprints[path] {
			return ModelSearch{}, &CatalogError{Code: SearchCodeSourceChanged,
				Message: fmt.Sprintf("source %s changed after this read indexed it; read the workspace again", path)}
		}
		scanned++
		scannedBytes += len(content)
		owners := attributeLines(content, byArtifact[path])
		lines := sourceLines(content)
		for i, text := range lines {
			line := i + 1
			subject := owners[line]
			if subject == nil {
				continue
			}
			if term == "" {
				// Without a term the answer is the kind's Sources, so a blank
				// line is noise rather than a hit.
				if strings.TrimSpace(text) == "" {
					continue
				}
			} else if !strings.Contains(strings.ToLower(text), term) {
				continue
			}
			matches = append(matches, ModelSearchHit{
				Subject: *subject, ArtifactPath: path, Line: line, Text: text,
			})
			counts[subject.Kind]++
		}
	}
	out.Total = len(matches)
	out.KindCounts = sortedKindCounts(counts)
	// A requested offset past the last match is clamped, and the caller's value
	// is kept so the clamped page is observable (#1177).
	if offset > out.Total {
		out.RequestedOffset = offset
		offset = out.Total
	}
	out.Offset = offset
	end := offset + limit
	if end > out.Total {
		end = out.Total
	}
	out.Hits = append(out.Hits, matches[offset:end]...)
	out.Omitted = out.Total - len(out.Hits)
	if end < out.Total {
		next := end
		out.NextOffset = &next
	}
	return out, nil
}

// attributeLines binds each line of one Source to the subject whose indexed
// declaration it belongs to.
//
// The Source Index binds a subject to its declaration identifier, which for the
// native grammar is a single line. A line that carries no identifier of its own
// -- a relation key, a capability input, a closing brace -- therefore belongs to
// the nearest preceding declaration among the declarations being searched: with
// no kind filter that is the most specific declaration on the preceding lines,
// and with a kind filter it is the nearest declaration of that kind, so a data
// search still sees a mention inside the data element's own block. A line before
// the first declaration of the file belongs to nothing and is not searched; the
// rule is stated in authorityLimits rather than left for a reader to infer.
// Where several declarations start on one line, the latest start wins, so a
// whole-Source contribution never shadows the declaration inside it.
func attributeLines(content []byte, refs []sourceindex.Reference) []*sourceindex.Subject {
	lines := sourceLines(content)
	starts := map[int]sourceindex.Reference{}
	for _, ref := range refs {
		line := ref.Range.Start.Line
		if line < 1 || line > len(lines) {
			continue
		}
		if current, ok := starts[line]; !ok || ref.Range.Start.Offset > current.Range.Start.Offset {
			starts[line] = ref
		}
	}
	// owners is 1-based: owners[0] is unused so a line number indexes directly,
	// and a nil entry names a line no declaration owns.
	owners := make([]*sourceindex.Subject, len(lines)+1)
	var current *sourceindex.Subject
	for line := 1; line <= len(lines); line++ {
		if ref, ok := starts[line]; ok {
			subject := ref.Subject
			current = &subject
		}
		if current != nil {
			owner := *current
			owners[line] = &owner
		}
	}
	return owners
}

// sourceLines splits one Source into its lines as written, without the line
// terminator. A CRLF checkout keeps its content identical to an LF one for
// matching purposes, so a trailing carriage return is dropped.
func sourceLines(content []byte) []string {
	text := strings.ReplaceAll(string(content), "\r\n", "\n")
	return strings.Split(text, "\n")
}

// searchCoverage names the text this search derives and the text it does not.
// The search covers the managed Sources the catalog's subjects are declared in;
// a Source with no indexed declaration, and the lines before the first
// declaration of a Source, are outside it and are listed rather than implied.
func searchCoverage() ModelSearchCoverage {
	return ModelSearchCoverage{
		Derived: []string{
			"managed-source-text-of-catalog-subjects",
		},
		NotDerived: []string{
			"sources-without-an-indexed-declaration",
			"lines-before-the-first-declaration-of-a-source",
			"sources-beyond-the-work-bound",
			"acceptance-and-edit-authority",
		},
	}
}
