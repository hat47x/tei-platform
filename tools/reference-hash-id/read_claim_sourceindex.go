package hashid

import (
	"errors"
	"fmt"
	"strings"

	"tei/internal/sourceindex"
)

// IndexedReadDiagnostic states that a caller-supplied Source Index projection,
// selected artifact fingerprint, Source revision label, and caller-supplied
// bytes are internally consistent. It never claims the Source Index came
// from a trusted Reader, that the Consumer actually used these bytes, or that
// the source/subject/attempt was accepted.
type IndexedReadDiagnostic struct {
	Status                   string
	Authority                string
	SourceIndexRevision      string
	ArtifactPath             string
	ArtifactFingerprint      string
	SourceReaderVerified     bool
	QualifiedSubjectVerified bool
	SourceScopeVerified      bool
	ReadAttemptVerified      bool
	SourceAccepted           bool
	AllSourceBytesVerified   bool
}

// CompareDeclaredSourceIndexRead uses TEI's existing sourceindex package
// rather than reimplementing its path/fingerprint/revision algorithm.
//
// An index consists of declared artifact fingerprints, not an atomic multi-
// file snapshot and not evidence that a tool consumed these file bytes. The
// Source Index's ModelRevision is based on *all* indexed paths/fingerprints:
// a per-file content hash must never stand in for this revision. The claim's
// subject/scope/attempt are syntactically checked but cannot be verified by
// an Index.Artifacts entry, which has no such authority.
func CompareDeclaredSourceIndexRead(index sourceindex.Index, rawPath string, claim ReadClaim, consumerBytes []byte) (IndexedReadDiagnostic, error) {
	if err := validateReadClaim(claim); err != nil {
		return IndexedReadDiagnostic{}, fmt.Errorf("caller read claim: %w", err)
	}
	relative, err := sourceindex.NormalizeWorkspacePath(rawPath)
	if err != nil || relative != rawPath {
		return IndexedReadDiagnostic{}, errors.New("source path must be an exact canonical Workspace-relative path")
	}
	if len(index.Artifacts) == 0 {
		return IndexedReadDiagnostic{}, errors.New("source index has no managed artifacts")
	}
	seen := make(map[string]struct{}, len(index.Artifacts))
	var selected *sourceindex.Artifact
	for i := range index.Artifacts {
		a := &index.Artifacts[i]
		canonical, err := sourceindex.NormalizeWorkspacePath(a.Path)
		if err != nil || canonical != a.Path {
			return IndexedReadDiagnostic{}, fmt.Errorf("source index artifact %d has a noncanonical path", i)
		}
		if _, duplicate := seen[a.Path]; duplicate {
			return IndexedReadDiagnostic{}, fmt.Errorf("source index duplicates artifact path %q", a.Path)
		}
		seen[a.Path] = struct{}{}
		if !canonicalSourceFingerprint(a.Fingerprint) {
			return IndexedReadDiagnostic{}, fmt.Errorf("source index artifact %q has an invalid fingerprint", a.Path)
		}
		if a.Path == relative {
			selected = a
		}
	}
	if selected == nil {
		return IndexedReadDiagnostic{}, errors.New("requested source file is not in the declared index")
	}
	revision := sourceindex.ModelRevision(index)
	if claim.SourceRevision != revision {
		return IndexedReadDiagnostic{}, errors.New("Source Index ModelRevision differs from caller revision")
	}
	fingerprint := sourceindex.Fingerprint(consumerBytes)
	if selected.Fingerprint != fingerprint {
		return IndexedReadDiagnostic{}, errors.New("consumer bytes differ from indexed artifact fingerprint")
	}
	if claim.SourceSHA256 != strings.TrimPrefix(fingerprint, "sha256:") {
		return IndexedReadDiagnostic{}, errors.New("caller source SHA-256 differs from indexed consumer bytes")
	}
	return IndexedReadDiagnostic{
		Status:                   "caller-index-bytes-consistent-only",
		Authority:                "none",
		SourceIndexRevision:      revision,
		ArtifactPath:             selected.Path,
		ArtifactFingerprint:      selected.Fingerprint,
		SourceReaderVerified:     false,
		QualifiedSubjectVerified: false,
		SourceScopeVerified:      false,
		ReadAttemptVerified:      false,
		SourceAccepted:           false,
		AllSourceBytesVerified:   false,
	}, nil
}

func canonicalSourceFingerprint(v string) bool {
	if len(v) != len("sha256:")+64 || !strings.HasPrefix(v, "sha256:") {
		return false
	}
	for i := len("sha256:"); i < len(v); i++ {
		c := v[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}
