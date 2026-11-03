// Package studioplugindatalifecycle is a read-only reference for Studio source
// editing and explicit data deletion (#2047). It keeps a raw-source save apart
// from semantic acceptance and execution permission, refuses to overwrite an
// external edit, deletes plugin data only when the target area is confirmed and
// explicitly selected, and preserves bytes outside the deleted or converted
// scope.
//
// It is not a Studio editor, a Host file writer, a resolver, an acceptance
// engine, or an accepted API. It never writes a file; every result is a
// synthetic proposal over the given input. The real Host save, the real screen
// and user-value checks are outside this reference.
package studioplugindatalifecycle

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// Schema identifies the save, delete, and conversion records.
const Schema = "tei.reference.studio-plugin-data-lifecycle/v0"

// Plugin availability. A missing, disabled or version-incompatible plugin
// never blocks a raw-source save.
const (
	PluginAvailable          = "available"
	PluginMissing            = "missing"
	PluginDisabled           = "disabled"
	PluginVersionUnsupported = "version-unsupported"
)

// EditRequest is one raw-source edit. Base and current identify what the editor
// started from and what is on disk now; Original and Edited are raw text.
type EditRequest struct {
	Path               string `json:"path"`
	BaseRevision       string `json:"baseRevision"`
	BaseFingerprint    string `json:"baseFingerprint"`
	CurrentRevision    string `json:"currentRevision"`
	CurrentFingerprint string `json:"currentFingerprint"`
	Original           string `json:"original"`
	Edited             string `json:"edited"`
	// Protected lists raw snippets that must survive. They stand for bytes
	// outside the edited range: unknown attributes, unknown child elements, and
	// extensions inside a known child.
	Protected            []string `json:"protected,omitempty"`
	PluginAvailability   string   `json:"pluginAvailability"`
	SyntaxValid          bool     `json:"syntaxValid"`
	UnresolvedReferences []string `json:"unresolvedReferences,omitempty"`
	// Accepted and Executable are the existing acceptance and execution facts.
	// Saving never changes them.
	Accepted   bool `json:"accepted"`
	Executable bool `json:"executable"`
}

// SaveResult is the raw-source save and the separate diagnosis.
type SaveResult struct {
	Schema            string   `json:"schema"`
	Path              string   `json:"path"`
	Saved             bool     `json:"saved"`
	Bytes             string   `json:"bytes,omitempty"`
	Reasons           []string `json:"reasons,omitempty"`
	Diagnosis         []string `json:"diagnosis,omitempty"`
	Accepted          bool     `json:"accepted"`
	AcceptanceChanged bool     `json:"acceptanceChanged"`
	Executable        bool     `json:"executable"`
	SourcePreserved   bool     `json:"sourcePreserved"`
	ProtectedMissing  []string `json:"protectedMissing,omitempty"`
}

func sortedUnique(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	compacted := out[:0]
	for _, value := range out {
		if value == "" {
			continue
		}
		if len(compacted) == 0 || compacted[len(compacted)-1] != value {
			compacted = append(compacted, value)
		}
	}
	return compacted
}

func fingerprint(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// Fingerprint returns the digest used for source bytes.
func Fingerprint(value string) string { return fingerprint(value) }

func validateEdit(req EditRequest) error {
	if strings.TrimSpace(req.Path) == "" {
		return fmt.Errorf("path is required")
	}
	if strings.TrimSpace(req.BaseRevision) == "" || strings.TrimSpace(req.CurrentRevision) == "" {
		return fmt.Errorf("baseRevision and currentRevision are required")
	}
	if strings.TrimSpace(req.BaseFingerprint) == "" || strings.TrimSpace(req.CurrentFingerprint) == "" {
		return fmt.Errorf("baseFingerprint and currentFingerprint are required")
	}
	switch req.PluginAvailability {
	case "", PluginAvailable, PluginMissing, PluginDisabled, PluginVersionUnsupported:
	default:
		return fmt.Errorf("unknown pluginAvailability %q", req.PluginAvailability)
	}
	return nil
}

// Save accepts a raw-source save even when the plugin is absent, disabled,
// version-incompatible or the text no longer parses. It refuses when the
// editor's base does not match the current revision or bytes (an external
// edit), and refuses when the edit removed protected content. It never turns a
// save into acceptance or execution.
func Save(req EditRequest) (SaveResult, error) {
	if err := validateEdit(req); err != nil {
		return SaveResult{}, err
	}
	result := SaveResult{Schema: Schema, Path: req.Path, Accepted: req.Accepted}
	if req.CurrentRevision != req.BaseRevision {
		result.Reasons = append(result.Reasons, "stale-revision")
		return finishSave(result), nil
	}
	if req.CurrentFingerprint != req.BaseFingerprint {
		result.Reasons = append(result.Reasons, "source-bytes-mismatch")
		return finishSave(result), nil
	}
	var missing []string
	for _, snippet := range req.Protected {
		if !strings.Contains(req.Edited, snippet) {
			missing = append(missing, snippet)
		}
	}
	if len(missing) > 0 {
		result.ProtectedMissing = sortedUnique(missing)
		result.Reasons = append(result.Reasons, "protected-content-lost")
		return finishSave(result), nil
	}

	result.Saved = true
	result.Bytes = req.Edited
	result.SourcePreserved = true
	availability := req.PluginAvailability
	if availability == "" {
		availability = PluginAvailable
	}
	if availability != PluginAvailable {
		result.Diagnosis = append(result.Diagnosis, "plugin-unavailable:"+availability)
	}
	if !req.SyntaxValid {
		result.Diagnosis = append(result.Diagnosis, "syntax-invalid")
	}
	for _, reference := range req.UnresolvedReferences {
		result.Diagnosis = append(result.Diagnosis, "unknown-reference:"+reference)
	}
	result.Executable = req.Executable && availability == PluginAvailable &&
		req.SyntaxValid && len(req.UnresolvedReferences) == 0
	if !result.Executable && !contains(result.Diagnosis, "execution-denied") {
		result.Diagnosis = append(result.Diagnosis, "execution-denied")
	}
	// Acceptance is an existing fact. A save neither grants nor revokes it.
	result.AcceptanceChanged = false
	return finishSave(result), nil
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func finishSave(result SaveResult) SaveResult {
	result.Reasons = sortedUnique(result.Reasons)
	result.Diagnosis = sortedUnique(result.Diagnosis)
	result.ProtectedMissing = sortedUnique(result.ProtectedMissing)
	return result
}
