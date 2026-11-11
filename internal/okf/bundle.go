package okf

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"
	"unicode/utf8"
)

// Default limits. They bound what one read may allocate; an oversized file is
// rejected with a diagnostic rather than truncated, because a silent truncation
// would report a digest of bytes that are not the author's.
const (
	DefaultMaxFileBytes   = 1 << 20  // 1 MiB per document
	DefaultMaxBundleBytes = 16 << 20 // 16 MiB per bundle
	DefaultMaxConcepts    = 512
	// maxScanDepth bounds the walk so a symlink loop or a deeply nested tree
	// cannot run without end. It is a walk guard, not a format rule.
	maxScanDepth = 64
)

// ErrInvalidRoot reports a bundle root that cannot be opened as a directory.
var ErrInvalidRoot = errors.New("okf: bundle root is not a readable directory")

// Options tunes what a read is allowed to consume. Zero values take the
// documented defaults; there is no unbounded mode.
type Options struct {
	MaxFileBytes   int64 `json:"maxFileBytes"`
	MaxBundleBytes int64 `json:"maxBundleBytes"`
	MaxConcepts    int   `json:"maxConcepts"`
}

func (o Options) normalized() Options {
	out := o
	if out.MaxFileBytes <= 0 {
		out.MaxFileBytes = DefaultMaxFileBytes
	}
	if out.MaxBundleBytes <= 0 {
		out.MaxBundleBytes = DefaultMaxBundleBytes
	}
	if out.MaxConcepts <= 0 {
		out.MaxConcepts = DefaultMaxConcepts
	}
	return out
}

// Revision identifies the exact bytes read: every concept, index and log file by
// path and digest. It is a byte-set identity, not a semantic version and not an
// acceptance record.
type Revision struct {
	Algorithm string `json:"algorithm"`
	Value     string `json:"value"`
}

// Format is the bundle-level F0 verdict plus the diagnostics behind it.
type Format struct {
	Conformant       bool         `json:"conformant"`
	Concepts         int          `json:"concepts"`
	Indexes          int          `json:"indexes"`
	Logs             int          `json:"logs"`
	FormatErrors     []Diagnostic `json:"formatErrors"`
	ObservationCount int          `json:"observationCount"`
}

// NotEvaluated names the questions this read did not answer. The list is part of
// the output so a consumer cannot read a conformant bundle as an accepted one.
type NotEvaluated struct {
	Items []string `json:"items"`
}

// Bundle is the complete result of one read-only pass over a directory tree.
type Bundle struct {
	// Root is the directory the read was given. It is echoed, not resolved, so a
	// caller does not mistake it for a canonical path.
	Root string `json:"root"`
	// OkfVersion is the version this reader implements, not a claim about what
	// the bundle declares.
	OkfVersion string `json:"okfVersion"`
	// Revision identifies the bytes read.
	Revision Revision `json:"bundleRevision"`

	Concepts []Concept `json:"concepts"`
	Indexes  []Index   `json:"indexes"`
	Logs     []Log     `json:"logs"`

	// UnresolvedReferences lists path-valued fields and links that matched no
	// file. OKF v0.2 §6.1 requires tolerating these; listing them is how a
	// consumer tells knowledge-still-unwritten from a broken bundle.
	UnresolvedReferences []resolvedPath `json:"unresolvedReferences"`

	Format Format `json:"format"`
	// Diagnostics aggregates bundle-level findings plus every file's findings, so
	// one array answers "what did the read observe" without re-walking.
	Diagnostics  []Diagnostic `json:"diagnostics"`
	NotEvaluated NotEvaluated `json:"notEvaluated"`
	Options      Options      `json:"options"`
}

// notEvaluatedItems is fixed text: it states what a format read cannot answer.
var notEvaluatedItems = []string{
	"TEI meaning, role, or trait acceptance",
	"TEI domain conformance (F2)",
	"required source closure for any consumer purpose",
	"business owner acceptance or decision",
	"host grant, capability, or runtime qualification",
	"external resource fetch or retrieval success",
	"attested computation execution, receipt, or verdict",
	"document write, repair, or normalization",
	"concept history, review history, or A-B-A detection",
	"SUI tree, island, or occurrence recoverability",
	"source identity across moves or a second location",
	"independent oracle qualification of this reader",
}

type rawDoc struct {
	path     string
	document *parsedDocument
	body     bodyScan
	digest   string
}

// Read walks a bundle and returns everything the reader could establish. It
// performs no network access, starts no process, loads no plugin, and writes
// nothing. A format error does not stop the walk: the remaining documents are
// still reported, because a writer needs the raw bytes even when the format is
// wrong.
func Read(root string, opts Options) (*Bundle, error) {
	options := opts.normalized()
	rootHandle, err := os.OpenRoot(root)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidRoot, err)
	}
	defer rootHandle.Close()
	info, err := rootHandle.Stat(".")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidRoot, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%w: %s is not a directory", ErrInvalidRoot, root)
	}

	bundle := &Bundle{
		Root:         root,
		OkfVersion:   "0.2",
		Options:      options,
		Concepts:     []Concept{},
		Indexes:      []Index{},
		Logs:         []Log{},
		Diagnostics:  []Diagnostic{},
		NotEvaluated: NotEvaluated{Items: append([]string{}, notEvaluatedItems...)},
	}

	paths, allFiles, diags := walkBundle(rootHandle)
	bundle.Diagnostics = append(bundle.Diagnostics, diags...)

	var docs []rawDoc
	var totalBytes int64
	conceptsSeen := 0

	for _, rel := range paths {
		raw, err := readCapped(rootHandle, rel, options.MaxFileBytes)
		if err != nil {
			code := CodeReadFailed
			message := "the file could not be read: " + err.Error()
			var tooLarge *fileTooLargeError
			if errors.As(err, &tooLarge) {
				code = CodeFileTooLarge
				message = fmt.Sprintf("the file is larger than the %d byte limit; the reader refuses to truncate it", options.MaxFileBytes)
			}
			bundle.Diagnostics = append(bundle.Diagnostics, Diagnostic{
				Code: code, Severity: SeverityError, Path: rel, Message: message,
			})
			continue
		}
		totalBytes += int64(len(raw))
		if totalBytes > options.MaxBundleBytes {
			bundle.Diagnostics = append(bundle.Diagnostics, Diagnostic{
				Code:     CodeBundleTooLarge,
				Severity: SeverityError,
				Message:  fmt.Sprintf("the bundle exceeds the %d byte limit; the walk stopped at %s", options.MaxBundleBytes, rel),
			})
			break
		}
		if !utf8.Valid(raw) {
			bundle.Diagnostics = append(bundle.Diagnostics, Diagnostic{
				Code:     CodeFileNotUTF8,
				Severity: SeverityError,
				Path:     rel,
				Message:  "the file is not valid UTF-8; OKF v0.2 documents are UTF-8 Markdown",
			})
			continue
		}

		if !isReservedName(path.Base(rel)) {
			conceptsSeen++
			if conceptsSeen > options.MaxConcepts {
				bundle.Diagnostics = append(bundle.Diagnostics, Diagnostic{
					Code:     CodeTooManyConcepts,
					Severity: SeverityError,
					Message:  fmt.Sprintf("the bundle declares more than %d concepts; the walk stopped before %s", options.MaxConcepts, rel),
				})
				break
			}
		}

		conceptID := conceptIDFor(rel)
		doc := parseDocument(rel, conceptID, raw)
		docs = append(docs, rawDoc{
			path:     rel,
			document: doc,
			body:     scanBody(doc.Body, doc.BodyStartLine),
			digest:   digestOf(raw),
		})
	}

	existing := make(map[string]bool, len(docs))
	for _, doc := range docs {
		existing[doc.path] = true
	}
	// A path-valued field may name a file the reader does not parse: an executor
	// script, an attester module, an image. Presence on disk is what makes such a
	// reference resolvable, so the walk contributes every file it saw, while only
	// the .md files are read as documents.
	for _, rel := range allFiles {
		existing[rel] = true
	}

	var revisionMaterial strings.Builder
	fmt.Fprintf(&revisionMaterial, "okf:%s\n", bundle.OkfVersion)
	conceptIDs := map[string]string{}

	for _, doc := range docs {
		base := path.Base(doc.path)
		switch {
		case base == "index.md":
			index := buildIndex(doc)
			resolveIndexEntries(&index, func(candidate string) bool { return existing[candidate] })
			bundle.Indexes = append(bundle.Indexes, index)
			bundle.Diagnostics = append(bundle.Diagnostics, index.Diagnostics...)
			for _, entry := range index.Entries {
				if entry.Resolution == ResolutionUnresolved || entry.Resolution == ResolutionAmbiguous || entry.Resolution == ResolutionInvalid {
					bundle.UnresolvedReferences = append(bundle.UnresolvedReferences, resolvedPath{
						Raw: entry.Target, Kind: "index-entry", Line: entry.Line,
						Resolution: entry.Resolution, Resolved: entry.Resolved,
					})
				}
			}
		case base == "log.md":
			log := buildLog(doc)
			bundle.Logs = append(bundle.Logs, log)
			bundle.Diagnostics = append(bundle.Diagnostics, log.Diagnostics...)
		default:
			concept, refs := buildConcept(doc, existing)
			if previous, ok := conceptIDs[concept.ConceptID]; ok {
				concept.Diagnostics = append(concept.Diagnostics, Diagnostic{
					Code:      CodeDuplicatedConceptID,
					Severity:  SeverityError,
					Path:      concept.Path,
					ConceptID: concept.ConceptID,
					Message:   "concept ID " + quoteKey(concept.ConceptID) + " is already produced by " + previous + "; OKF concept IDs are bundle-relative paths",
				})
				concept.FormatConformant = false
			} else {
				conceptIDs[concept.ConceptID] = concept.Path
			}
			// Body links are resolved only now, when the whole file set is
			// known, so a link to a later file is not reported as broken.
			for _, link := range concept.Links {
				ref := resolveTarget(concept.Path, link.Target, "body-link", link.Line, func(candidate string) bool {
					return existing[candidate]
				})
				if isUnresolved(ref.Resolution) {
					bundle.UnresolvedReferences = append(bundle.UnresolvedReferences, ref)
					concept.Diagnostics = append(concept.Diagnostics, unresolvedDiagnostic(
						concept.Path, concept.ConceptID, "body link", link.Target, ref))
				}
			}
			concept.FormatConformant = !hasFormatError(concept.Diagnostics)
			bundle.Concepts = append(bundle.Concepts, concept)
			bundle.Diagnostics = append(bundle.Diagnostics, concept.Diagnostics...)
			for _, ref := range refs {
				if isUnresolved(ref.Resolution) {
					bundle.UnresolvedReferences = append(bundle.UnresolvedReferences, ref)
				}
			}
		}
		fmt.Fprintf(&revisionMaterial, "%s\x00%s\n", doc.path, doc.digest)
	}

	revisionSum := sha256.Sum256([]byte(revisionMaterial.String()))
	bundle.Revision = Revision{Algorithm: "sha256", Value: "sha256:" + hex.EncodeToString(revisionSum[:])}

	bundle.Diagnostics = append(bundle.Diagnostics, structuralObservations(bundle)...)
	bundle.Format = finalizeFormat(bundle)
	sortDiagnostics(bundle.Diagnostics)
	for i := range bundle.Concepts {
		sortDiagnostics(bundle.Concepts[i].Diagnostics)
	}
	for i := range bundle.Indexes {
		sortDiagnostics(bundle.Indexes[i].Diagnostics)
	}
	for i := range bundle.Logs {
		sortDiagnostics(bundle.Logs[i].Diagnostics)
	}
	sortUnresolved(bundle.UnresolvedReferences)
	return bundle, nil
}

func isUnresolved(r Resolution) bool {
	switch r {
	case ResolutionUnresolved, ResolutionAmbiguous, ResolutionInvalid:
		return true
	}
	return false
}

// hasFormatError reports whether any diagnostic is an F0 violation.
func hasFormatError(diags []Diagnostic) bool {
	for _, diag := range diags {
		if diag.IsFormatError() {
			return true
		}
	}
	return false
}

// finalizeFormat derives the bundle verdict from every diagnostic collected.
func finalizeFormat(bundle *Bundle) Format {
	format := Format{
		Concepts: len(bundle.Concepts),
		Indexes:  len(bundle.Indexes),
		Logs:     len(bundle.Logs),
	}
	for _, diag := range bundle.Diagnostics {
		if diag.IsFormatError() {
			format.FormatErrors = append(format.FormatErrors, diag)
		} else {
			format.ObservationCount++
		}
	}
	format.Conformant = len(format.FormatErrors) == 0
	return format
}

// structuralObservations reports the two F1 observations that only a whole-bundle
// view can make: no root index, and concepts no index mentions.
func structuralObservations(bundle *Bundle) []Diagnostic {
	var diags []Diagnostic
	hasRootIndex := false
	for _, index := range bundle.Indexes {
		if index.IsRoot {
			hasRootIndex = true
		}
	}
	if !hasRootIndex {
		diags = append(diags, Diagnostic{
			Code:     CodeIndexMissing,
			Severity: SeverityWarning,
			Message:  "the bundle has no root index.md; OKF v0.2 §8 makes it optional and a consumer may synthesize one",
		})
	}

	indexed := map[string]bool{}
	for _, index := range bundle.Indexes {
		for _, entry := range index.Entries {
			for _, resolved := range entry.Resolved {
				indexed[resolved] = true
			}
		}
	}
	for _, concept := range bundle.Concepts {
		if !indexed[concept.Path] {
			diags = append(diags, Diagnostic{
				Code:      CodeConceptNotIndexed,
				Severity:  SeverityInfo,
				Path:      concept.Path,
				ConceptID: concept.ConceptID,
				Message:   "no index.md entry lists this concept; progressive disclosure is incomplete",
			})
		}
	}
	for _, log := range bundle.Logs {
		if !indexed[log.Path] {
			diags = append(diags, Diagnostic{
				Code:     CodeConceptNotIndexed,
				Severity: SeverityInfo,
				Path:     log.Path,
				Message:  "no index.md entry lists this log file",
			})
		}
	}
	return diags
}

// conceptIDFor derives the OKF concept ID: the bundle-relative path with the .md
// suffix removed. It is an identifier, not a TEI meaning or document ID.
func conceptIDFor(rel string) string {
	return strings.TrimSuffix(rel, path.Ext(rel))
}

func digestOf(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// walkBundle lists the bundle's files. markdown holds the .md files the reader
// parses as documents; allFiles holds every regular file, so a path-valued field
// that names an executor script or an attester module is resolvable even though the
// reader never reads its content.
func walkBundle(root *os.Root) (markdown []string, allFiles []string, diags []Diagnostic) {
	// Walking through the root handle keeps directory listing inside the same
	// traversal boundary as the file reads.
	err := fs.WalkDir(root.FS(), ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			diags = append(diags, Diagnostic{
				Code:     CodeReadFailed,
				Severity: SeverityError,
				Path:     name,
				Message:  "the directory could not be walked: " + walkErr.Error(),
			})
			return fs.SkipDir
		}
		if entry.IsDir() {
			if name == "." {
				return nil
			}
			depth := strings.Count(name, "/") + 1
			if depth > maxScanDepth {
				diags = append(diags, Diagnostic{
					Code:     CodeReadFailed,
					Severity: SeverityWarning,
					Path:     name,
					Message:  fmt.Sprintf("the walk stopped below %d levels; deeper directories were not read", maxScanDepth),
				})
				return fs.SkipDir
			}
			return nil
		}
		rel := strings.ReplaceAll(name, "\\", "/")
		allFiles = append(allFiles, rel)
		if strings.HasSuffix(strings.ToLower(rel), ".md") {
			markdown = append(markdown, rel)
		}
		return nil
	})
	if err != nil {
		diags = append(diags, Diagnostic{
			Code:     CodeReadFailed,
			Severity: SeverityError,
			Message:  "the bundle walk failed: " + err.Error(),
		})
	}
	sort.Strings(markdown)
	sort.Strings(allFiles)
	return markdown, allFiles, diags
}

// fileTooLargeError distinguishes a size refusal so the caller can name it.
type fileTooLargeError struct{ limit int64 }

func (e *fileTooLargeError) Error() string {
	return fmt.Sprintf("file exceeds %d bytes", e.limit)
}

// readCapped reads one bundle-relative file through the root boundary and refuses
// anything larger than the limit instead of truncating it.
func readCapped(root *os.Root, rel string, limit int64) ([]byte, error) {
	f, err := root.Open(rel)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file", rel)
	}
	if info.Size() > limit {
		return nil, &fileTooLargeError{limit: limit}
	}
	// Read one byte past the limit so a file that grew between Stat and Read is
	// still refused rather than silently cut.
	buf, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(buf)) > limit {
		return nil, &fileTooLargeError{limit: limit}
	}
	return buf, nil
}

// sortDiagnostics orders findings so two reads of the same bytes produce the same
// output: by path, then line, then code, then message.
func sortDiagnostics(diags []Diagnostic) {
	sort.SliceStable(diags, func(i, j int) bool {
		if diags[i].Path != diags[j].Path {
			return diags[i].Path < diags[j].Path
		}
		if diags[i].Line != diags[j].Line {
			return diags[i].Line < diags[j].Line
		}
		if diags[i].Code != diags[j].Code {
			return diags[i].Code < diags[j].Code
		}
		return diags[i].Message < diags[j].Message
	})
}

func sortUnresolved(refs []resolvedPath) {
	sort.SliceStable(refs, func(i, j int) bool {
		if refs[i].Kind != refs[j].Kind {
			return refs[i].Kind < refs[j].Kind
		}
		if refs[i].Raw != refs[j].Raw {
			return refs[i].Raw < refs[j].Raw
		}
		return refs[i].Line < refs[j].Line
	})
}

// isReservedName reports the two filenames OKF v0.2 §3.1 reserves.
func isReservedName(base string) bool {
	return base == "index.md" || base == "log.md"
}
