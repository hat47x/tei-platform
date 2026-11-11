package okf

import (
	"path"
	"strings"
)

// Resolution names how a path-valued field or body link was interpreted. OKF v0.2
// section 6.2 allows an absolute URL, a bundle-relative path beginning with "/", or
// a relative path. The specification does not fix the base for the third form, and
// the upstream sample bundle resolves path-valued fields against the bundle root
// while body links use standard document-relative Markdown, so this reader tries
// both and records which one applied.
type Resolution string

const (
	// ResolutionDocumentRelative means the target resolved against the
	// containing document's directory.
	ResolutionDocumentRelative Resolution = "document-relative"
	// ResolutionBundleRoot means the target resolved against the bundle root only.
	// OKF v0.2 section 6.1 recommends this form for body links, and the upstream
	// sample bundle uses it for path-valued frontmatter fields.
	ResolutionBundleRoot Resolution = "bundle-root"
	// ResolutionAmbiguous means the target resolved against both bases, to
	// different files. The reader keeps both and does not choose.
	ResolutionAmbiguous Resolution = "ambiguous"
	// ResolutionAbsoluteURL means the target is a URL. The reader never fetches it;
	// the document is a reference, not a retrieval instruction.
	ResolutionAbsoluteURL Resolution = "absolute-url-not-fetched"
	// ResolutionScopeDescriptor means the target names a population or scope a
	// consumer cannot follow as a file.
	ResolutionScopeDescriptor Resolution = "scope-descriptor-not-fetched"
	// ResolutionUnresolved means the target matched neither base form.
	ResolutionUnresolved Resolution = "unresolved"
	// ResolutionInvalid means the target could not be interpreted at all.
	ResolutionInvalid Resolution = "invalid"
)

// resolvedPath is one path-valued field or link and what the reader learned about
// it. Resolved holds the bundle-relative path(s) that exist; a URL or scope
// descriptor leaves it empty and sets Resolution accordingly.
type resolvedPath struct {
	// Raw is the value exactly as written.
	Raw string `json:"raw"`
	// Kind names the field the value came from (a frontmatter key such as
	// "resource", "sources[].resource", "executor.resource", or "body-link").
	Kind string `json:"kind"`
	// Resolution says how the raw value was interpreted.
	Resolution Resolution `json:"resolution"`
	// Resolved lists bundle-relative paths that exist for this target. It has two
	// entries only when Resolution is ambiguous.
	Resolved []string `json:"resolved,omitempty"`
	// Line is the 1-based line in the file, when the reader can locate it.
	Line int `json:"line,omitempty"`
}

// isAbsoluteURL reports whether a value is a URL rather than a bundle path.
func isAbsoluteURL(raw string) bool {
	lower := strings.ToLower(strings.TrimSpace(raw))
	for _, prefix := range []string{"http://", "https://", "mailto:", "urn:", "ftp://"} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

// looksLikeScopeDescriptor reports whether a non-URL resources entry reads as a
// population or scope rather than a followable artifact. OKF v0.2 section 5.1
// permits this ("all queries in BigQuery project X"), and the K1 fixture uses it.
// The test is deliberately shallow: a value that carries spaces or sentence
// punctuation but is not a URL is called a scope descriptor rather than an error.
func looksLikeScopeDescriptor(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || isAbsoluteURL(trimmed) {
		return false
	}
	return strings.ContainsAny(trimmed, " \t\u3000")
}

// resolveTarget interprets one path-valued value from one containing document.
// existed reports whether a bundle-relative path is present in the bundle; the
// caller supplies it so this function stays pure and performs no I/O.
func resolveTarget(containingPath, raw, kind string, line int, existed func(bundleRelative string) bool) resolvedPath {
	out := resolvedPath{Raw: raw, Kind: kind, Line: line}
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		out.Resolution = ResolutionInvalid
		return out
	}
	if isAbsoluteURL(trimmed) {
		out.Resolution = ResolutionAbsoluteURL
		return out
	}

	if strings.HasPrefix(trimmed, "/") {
		// A bundle-root form has exactly one interpretation.
		candidate := cleanBundleCandidate(strings.TrimPrefix(trimmed, "/"))
		if candidate != "" && existed(candidate) {
			out.Resolution = ResolutionBundleRoot
			out.Resolved = []string{candidate}
			return out
		}
		if looksLikeScopeDescriptor(trimmed) {
			out.Resolution = ResolutionScopeDescriptor
			return out
		}
		out.Resolution = ResolutionUnresolved
		return out
	}

	documentCandidate := ""
	if base := path.Dir(containingPath); base != "." && base != "" {
		documentCandidate = cleanBundleCandidate(base + "/" + trimmed)
	} else {
		documentCandidate = cleanBundleCandidate(trimmed)
	}
	rootCandidate := cleanBundleCandidate(trimmed)

	documentExists := documentCandidate != "" && existed(documentCandidate)
	rootExists := rootCandidate != "" && rootCandidate != documentCandidate && existed(rootCandidate)

	switch {
	case documentExists && rootExists:
		out.Resolution = ResolutionAmbiguous
		out.Resolved = []string{documentCandidate, rootCandidate}
	case documentExists:
		out.Resolution = ResolutionDocumentRelative
		out.Resolved = []string{documentCandidate}
	case rootExists:
		out.Resolution = ResolutionBundleRoot
		out.Resolved = []string{rootCandidate}
	case looksLikeScopeDescriptor(trimmed):
		out.Resolution = ResolutionScopeDescriptor
	default:
		out.Resolution = ResolutionUnresolved
	}
	return out
}

// cleanBundleCandidate normalizes a bundle-relative candidate path. A path that
// escapes the bundle ("..", an absolute path, or a Windows drive prefix) yields ""
// so the caller reports it as unresolved rather than reading outside.
func cleanBundleCandidate(raw string) string {
	value := strings.ReplaceAll(strings.TrimSpace(raw), "\\", "/")
	if value == "" || strings.HasPrefix(value, "/") {
		return ""
	}
	if colon := strings.Index(value, ":"); colon >= 0 {
		return ""
	}
	cleaned := path.Clean(value)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return ""
	}
	return cleaned
}

// escapesBundle reports whether a raw target, interpreted from the containing
// document's directory, would leave the bundle root. Consumers of this reader treat
// that as a security-relevant observation, not as a format violation, because OKF
// itself does not define a sandbox.
//
// The check is relative to the containing document: "../data/order.md" from
// "functions/submit.md" stays inside the bundle, while "../../../etc/passwd" leaves
// it. A bundle-root absolute target is additionally rejected when it carries a
// drive prefix, which has no meaning in OKF.
func escapesBundle(containingPath, raw string) bool {
	value := strings.ReplaceAll(strings.TrimSpace(raw), "\\", "/")
	if value == "" || isAbsoluteURL(value) {
		return false
	}
	if colon := strings.Index(value, ":"); colon >= 0 {
		return true
	}
	if strings.HasPrefix(value, "/") {
		// A bundle-root target is relative to the root by definition; path.Clean
		// removes any leading "..", so nothing else can leave here.
		return false
	}
	joined := value
	if base := path.Dir(containingPath); base != "." && base != "" {
		joined = base + "/" + value
	}
	cleaned := path.Clean(joined)
	return cleaned == ".." || strings.HasPrefix(cleaned, "../")
}
