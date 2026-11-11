package okf

// Source is one entry of the `sources` frontmatter family. Every field is
// optional except `resource`, so an entry with no resource is reported rather than
// dropped.
type Source struct {
	ID           string `json:"id,omitempty"`
	Resource     string `json:"resource"`
	ResourceKind string `json:"resourceKind"`
	Title        string `json:"title,omitempty"`
	Author       string `json:"author,omitempty"`
	LastModified string `json:"lastModified,omitempty"`
	UsageCount   string `json:"usageCount,omitempty"`
	// Resolution and Resolved come from resolveTarget. A URL leaves both empty and
	// reads "absolute-url-not-fetched".
	Resolution Resolution `json:"resolution"`
	Resolved   []string   `json:"resolved,omitempty"`
}

// Verification is one `verified` event, normalized so a bare `{by, at}` mapping
// and a one-element list look the same to a consumer (OKF v0.2 §5.2).
type Verification struct {
	By string `json:"by,omitempty"`
	At string `json:"at,omitempty"`
}

// Generation is the `generated` mapping: who wrote the current content and when.
type Generation struct {
	By string `json:"by,omitempty"`
	At string `json:"at,omitempty"`
}

// Parameter is one declared `parameters` entry of an Attested Computation.
type Parameter struct {
	Name     string `json:"name"`
	Type     string `json:"type,omitempty"`
	Required string `json:"required,omitempty"`
}

// ExecutorDecl is the `executor` contract of an Attested Computation. The reader
// records the declaration; it never opens the resource it names.
type ExecutorDecl struct {
	Resource   string     `json:"resource,omitempty"`
	Receipt    []string   `json:"receipt,omitempty"`
	Resolution Resolution `json:"resolution,omitempty"`
	Resolved   []string   `json:"resolved,omitempty"`
}

// PathRef is a `computation` or `attester` declaration. It is a reference the
// reader refuses to follow.
type PathRef struct {
	Resource   string     `json:"resource"`
	Resolution Resolution `json:"resolution"`
	Resolved   []string   `json:"resolved,omitempty"`
}

// Concept is one OKF concept document. Nothing here is a TEI statement: the fields
// describe what the document says about itself, which is the scope at which an
// exchange format can be trusted.
type Concept struct {
	ConceptID string `json:"conceptId"`
	Path      string `json:"path"`
	Digest    string `json:"artifactDigest"`

	Type        string `json:"type"`
	TypeClass   string `json:"typeClass"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Resource    string `json:"resource,omitempty"`
	// ResourceResolution and ResourceResolved describe how the concept's own
	// `resource` was interpreted.
	ResourceResolution Resolution `json:"resourceResolution,omitempty"`
	ResourceResolved   []string   `json:"resourceResolved,omitempty"`
	Tags               []string   `json:"tags,omitempty"`

	Status       string `json:"status"`
	StatusAbsent bool   `json:"statusAbsent"`

	StaleAfter string `json:"staleAfter,omitempty"`
	Stale      string `json:"stale"`

	Generated *Generation    `json:"generated,omitempty"`
	Verified  []Verification `json:"verified"`
	// TrustTierLabel is OKF v0.2 §5.3's advisory label. It is not TEI review,
	// acceptance, or identity verification.
	TrustTierLabel string `json:"trustTierLabel"`

	Runtime     string        `json:"runtime,omitempty"`
	Parameters  []Parameter   `json:"parameters,omitempty"`
	Computation *PathRef      `json:"computation,omitempty"`
	Executor    *ExecutorDecl `json:"executor,omitempty"`
	Attester    *PathRef      `json:"attester,omitempty"`
	// HasComputationSection reports a body `# Computation` heading. Its presence is
	// a document fact, not evidence that anything ran.
	HasComputationSection bool `json:"hasComputationSection,omitempty"`
	// FencedLanguageCount counts the code fences in the body, so a caller can see
	// that a computation-shaped block exists without the reader executing it.
	FencedLanguageCount int `json:"fencedLanguageCount"`

	Sources []Source `json:"sources"`

	UnknownKeys []string `json:"unknownFrontmatterKeys,omitempty"`

	// Links lists body Markdown links with their resolution. It is the reader's
	// structural scan, not a claim that every reference in the prose was found.
	Links []bodyLink `json:"links"`
	// Footnotes joins the body's footnote labels with `sources[].id`.
	Footnotes []bodyFootnote `json:"footnotes"`

	// FormatConformant is this concept's own F0 verdict.
	FormatConformant bool         `json:"formatConformant"`
	Diagnostics      []Diagnostic `json:"diagnostics"`
}

// IndexEntry is one item of an `index.md` listing.
type IndexEntry struct {
	Title       string     `json:"title"`
	Target      string     `json:"target"`
	Description string     `json:"description,omitempty"`
	Line        int        `json:"line,omitempty"`
	Resolution  Resolution `json:"resolution"`
	Resolved    []string   `json:"resolved,omitempty"`
}

// Index is one reserved `index.md`.
type Index struct {
	Path   string `json:"path"`
	Digest string `json:"artifactDigest"`
	IsRoot bool   `json:"isRoot"`
	// OkfVersion is the root index's declared version, verbatim.
	OkfVersion  string       `json:"okfVersion,omitempty"`
	Entries     []IndexEntry `json:"entries"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// LogEntry is one date-grouped block of a `log.md`.
type LogEntry struct {
	Date    string   `json:"date"`
	Entries []string `json:"entries"`
	Line    int      `json:"line,omitempty"`
}

// Log is one reserved `log.md`.
type Log struct {
	Path           string       `json:"path"`
	Digest         string       `json:"artifactDigest"`
	HasFrontmatter bool         `json:"hasFrontmatter"`
	Entries        []LogEntry   `json:"entries"`
	Diagnostics    []Diagnostic `json:"diagnostics"`
}
