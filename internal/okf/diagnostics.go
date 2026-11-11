// Package okf reads an Open Knowledge Format (OKF) v0.2 knowledge bundle in
// read-only form and reports its format conformance.
//
// The reader is deliberately narrow. It parses Markdown + YAML frontmatter,
// extracts the standard frontmatter families when present, and reports
// diagnostics. It does not:
//
//   - fetch any resource named by the bundle (URL, scope descriptor, or path),
//   - open, execute, or validate any file that `executor`, `attester`, or
//     `computation` points at,
//   - treat frontmatter values as TEI meaning, source acceptance, review
//     status, host grants, or runtime qualifications,
//   - write, normalize, or repair the document bytes.
//
// Format conformance (F0) and the TEI-side quality profile (F1) are separate
// verdicts. Unknown `type` values, unknown extra frontmatter keys, broken
// cross-links, and a missing `index.md` are all F0-conformant; they only
// produce observations.
package okf

// Severity separates a format violation from an observation. Only SeverityError
// makes a bundle non-conformant under OKF v0.2 §11.
type Severity string

const (
	// SeverityError is a format (F0) violation: the document or reserved file
	// does not follow the structure OKF v0.2 requires.
	SeverityError Severity = "error"
	// SeverityWarning is an F1 observation. The format is still conformant.
	SeverityWarning Severity = "warning"
	// SeverityInfo is informational; it records what the reader declined to do.
	SeverityInfo Severity = "info"
)

// F0 diagnostic codes: format conformance under OKF v0.2 §11.
const (
	CodeFrontmatterMissing    = "FRONTMATTER_MISSING"
	CodeFrontmatterUnterminat = "FRONTMATTER_UNTERMINATED"
	CodeYAMLInvalid           = "YAML_INVALID"
	CodeYAMLDuplicateKey      = "YAML_DUPLICATE_KEY"
	CodeYAMLMultiDocument     = "YAML_MULTI_DOCUMENT"
	CodeYAMLNotMapping        = "YAML_NOT_MAPPING"
	CodeTypeMissing           = "TYPE_MISSING"
	CodeTypeEmpty             = "TYPE_EMPTY"
	CodeTypeNotString         = "TYPE_NOT_STRING"
	CodeIndexFrontmatterRoot  = "INDEX_FRONTMATTER_NOT_ROOT"
	CodeFileNotUTF8           = "FILE_NOT_UTF8"
	CodeFileTooLarge          = "FILE_TOO_LARGE"
	CodeBundleTooLarge        = "BUNDLE_TOO_LARGE"
	CodeTooManyConcepts       = "TOO_MANY_CONCEPTS"
	CodePathEscapesBundle     = "PATH_ESCAPES_BUNDLE"
	CodeReadFailed            = "READ_FAILED"
)

// F1 observation codes. None of these changes the F0 verdict.
const (
	CodeLinkUnresolved              = "LINK_UNRESOLVED"
	CodeIndexMissing                = "INDEX_MISSING"
	CodeConceptNotIndexed           = "CONCEPT_NOT_INDEXED"
	CodeIndexEntryTargetMissing     = "INDEX_ENTRY_TARGET_MISSING"
	CodeIndexEntryTargetEscapes     = "INDEX_ENTRY_TARGET_ESCAPES"
	CodeIndexEntryTargetNotConcept  = "INDEX_ENTRY_TARGET_NOT_CONCEPT"
	CodeLogDateNotISO               = "LOG_DATE_NOT_ISO"
	CodeLogFrontmatterPresent       = "LOG_FRONTMATTER_PRESENT"
	CodeStatusAbsent                = "STATUS_ABSENT"
	CodeStatusUnknown               = "STATUS_UNKNOWN"
	CodeFootnoteWithoutSourceID     = "FOOTNOTE_WITHOUT_SOURCE_ID"
	CodeSourceIDWithoutFootnote     = "SOURCE_ID_WITHOUT_FOOTNOTE"
	CodeStaleAfterWithoutOffset     = "STALE_AFTER_WITHOUT_OFFSET"
	CodeStaleAfterInvalid           = "STALE_AFTER_INVALID"
	CodeGeneratedWithoutBy          = "GENERATED_WITHOUT_BY"
	CodeGeneratedWithoutAt          = "GENERATED_WITHOUT_AT"
	CodeVerifiedEntryNotMapping     = "VERIFIED_ENTRY_NOT_MAPPING"
	CodeVerifiedEntryWithoutBy      = "VERIFIED_ENTRY_WITHOUT_BY"
	CodeVerifiedEmptyMapping        = "VERIFIED_EMPTY_MAPPING"
	CodeLegacyTimestampPresent      = "LEGACY_TIMESTAMP_PRESENT"
	CodeLegacyCitationsSection      = "LEGACY_CITATIONS_SECTION_PRESENT"
	CodeRootIndexOkfVersionInvalid  = "ROOT_INDEX_OKF_VERSION_INVALID"
	CodeRootIndexOkfVersionMissing  = "ROOT_INDEX_OKF_VERSION_MISSING"
	CodeResolvedFromBundleRoot      = "RELATIVE_RESOLVED_FROM_BUNDLE_ROOT"
	CodePathAmbiguousBase           = "PATH_AMBIGUOUS_BASE"
	CodeURLNotFetched               = "URL_NOT_FETCHED"
	CodeScopeDescriptorNotFetched   = "SCOPE_DESCRIPTOR_NOT_FETCHED"
	CodeBOMPresent                  = "BOM_PRESENT"
	CodeCRLFLineEndings             = "CRLF_LINE_ENDINGS"
	CodeAttestedComputationDeclared = "ATTESTED_COMPUTATION_DECLARED_ONLY"
	CodeExecutorNotOpened           = "EXECUTOR_NOT_OPENED"
	CodeAttesterNotOpened           = "ATTESTER_NOT_OPENED"
	CodeComputationNotOpened        = "COMPUTATION_NOT_OPENED"
	CodeDuplicatedConceptID         = "DUPLICATED_CONCEPT_ID"
	CodeConceptIDNotASCII           = "CONCEPT_ID_NOT_ASCII"
	CodeSourceMissingResource       = "SOURCE_MISSING_RESOURCE"
	CodeSourceComponentNotDerivable = "SOURCE_COMPONENT_NOT_DERIVABLE"
	CodeLinksNotEnumerated          = "LINKS_NOT_ENUMERATED"
	CodeHistoryNotAssessed          = "HISTORY_NOT_ASSESSED"
)

// Diagnostic is one observation about a bundle, a document, or a single
// frontmatter field. Position is optional: it is the 1-based line inside the
// file when the reader can locate the problem.
type Diagnostic struct {
	Code     string   `json:"code"`
	Severity Severity `json:"severity"`
	// Path is the bundle-relative path of the file the diagnostic belongs to,
	// or "" for a bundle-level diagnostic.
	Path      string `json:"path,omitempty"`
	ConceptID string `json:"conceptId,omitempty"`
	Line      int    `json:"line,omitempty"`
	Message   string `json:"message"`
}

// IsFormatError reports whether the diagnostic makes the bundle non-conformant.
func (d Diagnostic) IsFormatError() bool { return d.Severity == SeverityError }
