// Package sourceacceptance checks a Host's CURRENT Accepted Source witness
// against a Source Index made from the bytes read by the caller's Workspace.
// It does not create an Acceptance authority or grant command permission.
package sourceacceptance

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"tei/internal/sourceindex"
)

// Record is a read-only projection of a trusted Host's current Accepted Source
// decision. The Host owns provenance, revocation and authorization. Arbitrary
// user-provided JSON, a Decision string, or a caller-supplied revision is NOT
// an implementation of a trusted Reader.
type Record struct {
	ApplicationID     string                 `json:"applicationId"`
	DecisionRef       string                 `json:"decisionRef"`
	AuthorityRevision string                 `json:"authorityRevision"`
	ModelRevision     string                 `json:"modelRevision"`
	Artifacts         []sourceindex.Artifact `json:"artifacts"`
}

// ErrCurrentAcceptanceUnavailable is the public, deliberately opaque failure
// for a Host CURRENT acceptance read. Host errors may include credentials,
// private paths or issuer policy; never return them or wrap their error chain.
// The Host can keep its own diagnostic in private telemetry.
var ErrCurrentAcceptanceUnavailable = errors.New("SOURCE_ACCEPTANCE_CURRENT_UNAVAILABLE")

// Reader is implemented by a trusted Host. Implementations must check current
// acceptance and revocation on each call rather than trusting a stale receipt.
type Reader interface {
	ReadCurrentAcceptedSource(context.Context, string) (Record, error)
}

// VerifyCurrent refuses if the CURRENT Host witness is missing, malformed, or
// does not match the exact managed Source membership, path, and byte digest.
// The Source Index must be built from the same Workspace Load as the runtime.
// This check does not ensure a race-free multi-file Load or prevent source
// mutation after startup; see #1017. Nor does it grant command authorization.
func VerifyCurrent(ctx context.Context, reader Reader, applicationID string, loaded sourceindex.Index) (Record, error) {
	if reader == nil {
		return Record{}, errors.New("SOURCE_ACCEPTANCE_READER_REQUIRED")
	}
	if strings.TrimSpace(applicationID) == "" {
		return Record{}, errors.New("SOURCE_ACCEPTANCE_APPLICATION_REQUIRED")
	}
	if err := ctx.Err(); err != nil {
		return Record{}, fmt.Errorf("SOURCE_ACCEPTANCE_CURRENT_UNAVAILABLE: %w", err)
	}
	record, err := reader.ReadCurrentAcceptedSource(ctx, applicationID)
	if err != nil {
		// Never pass through the untrusted Reader's raw error or its
		// unwrap chain: startup CLI errors must not disclose Host secrets.
		return Record{}, ErrCurrentAcceptanceUnavailable
	}
	if err := ctx.Err(); err != nil {
		return Record{}, fmt.Errorf("SOURCE_ACCEPTANCE_CURRENT_UNAVAILABLE: %w", err)
	}
	if record.ApplicationID != applicationID {
		return Record{}, errors.New("SOURCE_ACCEPTANCE_APPLICATION_MISMATCH")
	}
	if strings.TrimSpace(record.DecisionRef) == "" || strings.TrimSpace(record.AuthorityRevision) == "" {
		return Record{}, errors.New("SOURCE_ACCEPTANCE_WITNESS_INVALID: missing Host decision provenance")
	}
	if !validFingerprint(record.ModelRevision) {
		return Record{}, errors.New("SOURCE_ACCEPTANCE_WITNESS_INVALID: invalid ModelRevision")
	}
	expected, err := artifactMap(record.Artifacts)
	if err != nil || len(expected) == 0 {
		return Record{}, fmt.Errorf("SOURCE_ACCEPTANCE_WITNESS_INVALID: managed source membership: %v", err)
	}
	if sourceindex.ModelRevision(sourceindex.Index{Artifacts: record.Artifacts}) != record.ModelRevision {
		return Record{}, errors.New("SOURCE_ACCEPTANCE_WITNESS_INVALID: revision does not match witnessed members")
	}
	actual, err := artifactMap(loaded.Artifacts)
	if err != nil || len(actual) == 0 {
		return Record{}, fmt.Errorf("SOURCE_INDEX_INVALID: loaded managed source membership: %v", err)
	}
	if len(actual) != len(expected) {
		return Record{}, errors.New("SOURCE_MEMBERSHIP_MISMATCH: managed file count differs")
	}
	// Stable failure selection is important for reproducible evidence.
	paths := make([]string, 0, len(expected))
	for path := range expected {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		got, exists := actual[path]
		if !exists {
			return Record{}, fmt.Errorf("SOURCE_MEMBERSHIP_MISMATCH: missing managed source %q", path)
		}
		if got != expected[path] {
			return Record{}, fmt.Errorf("SOURCE_FINGERPRINT_MISMATCH: managed source %q", path)
		}
	}
	if sourceindex.ModelRevision(loaded) != record.ModelRevision {
		return Record{}, errors.New("SOURCE_REVISION_MISMATCH: loaded Source and current Host acceptance differ")
	}
	// Avoid returning the Host Reader's mutable backing slice to a consumer.
	record.Artifacts = append([]sourceindex.Artifact(nil), record.Artifacts...)
	return record, nil
}

func artifactMap(artifacts []sourceindex.Artifact) (map[string]string, error) {
	byPath := make(map[string]string, len(artifacts))
	for _, artifact := range artifacts {
		normalized, err := sourceindex.NormalizeWorkspacePath(artifact.Path)
		if err != nil || normalized != artifact.Path {
			return nil, fmt.Errorf("noncanonical managed Source path %q", artifact.Path)
		}
		if !validFingerprint(artifact.Fingerprint) {
			return nil, fmt.Errorf("invalid fingerprint at %q", artifact.Path)
		}
		if _, exists := byPath[artifact.Path]; exists {
			return nil, fmt.Errorf("duplicate managed Source path %q", artifact.Path)
		}
		byPath[artifact.Path] = artifact.Fingerprint
	}
	return byPath, nil
}

func validFingerprint(value string) bool {
	if len(value) != 71 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	for _, c := range value[7:] {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}
