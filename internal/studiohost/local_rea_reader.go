package studiohost

import (
	"encoding/hex"
	"errors"
	"path"
	"strings"
	"time"

	"tei/internal/externalanalysis"
	"tei/internal/operations"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// LocalREABinding is configured by the trusted local Host, not supplied by a
// renderer or remote caller. File is a normalized Workspace-relative path.
// Digest and EvidenceID pin the saved Evidence bytes, not the current source.
type LocalREABinding struct {
	WorkspaceID string
	Selector    string
	File        string
	Version     string
	EvidenceID  string
	Digest      string
	Revision    string
	Tool        externalanalysis.Tool
}

type localREAKey struct {
	workspace string
	selector  string
}

// LocalREAReader is an unexported-to-Wails, Host-bound consumer of the existing
// R2 byte projection. It is NOT an authenticated remote service or new store.
type LocalREAReader struct {
	host     *ReadOnly
	bindings map[localREAKey]LocalREABinding
}

const localREAReadLimitBytes int64 = 4 << 20

func NewLocalREAReader(host *ReadOnly, configured []LocalREABinding) (*LocalREAReader, error) {
	if host == nil || len(configured) == 0 {
		return nil, errors.New("local REA reader needs an existing Host and bindings")
	}
	reader := &LocalREAReader{host: host, bindings: make(map[localREAKey]LocalREABinding, len(configured))}
	for _, binding := range configured {
		if binding.WorkspaceID == "" || binding.Selector == "" || strings.TrimSpace(binding.Selector) != binding.Selector ||
			binding.File == "" || strings.TrimSpace(binding.Version) == "" ||
			binding.EvidenceID == "" || binding.Digest == "" {
			return nil, errors.New("local REA binding is incomplete")
		}
		if _, denied := host.workspace(binding.WorkspaceID); denied != nil {
			return nil, errors.New("local REA binding refers to an unavailable Workspace")
		}
		normalized, err := sourceindex.NormalizeWorkspacePath(binding.File)
		if err != nil || normalized != binding.File {
			return nil, errors.New("local REA binding has an invalid relative file")
		}
		ext := strings.ToLower(path.Ext(normalized))
		if ext != ".json" && ext != ".jsonl" {
			return nil, errors.New("local REA binding requires saved JSON Evidence")
		}
		// A bound selector must not silently move to another Evidence when the
		// file is replaced. R2 also checks both pins against the received bytes.
		if !strings.HasPrefix(binding.Digest, "sha256:") || len(binding.Digest) != 71 {
			return nil, errors.New("local REA binding requires an exact sha256 digest")
		}
		digest, err := hex.DecodeString(strings.TrimPrefix(binding.Digest, "sha256:"))
		if err != nil || len(digest) != 32 || binding.Digest != strings.ToLower(binding.Digest) {
			return nil, errors.New("local REA binding requires a lowercase sha256 digest")
		}
		key := localREAKey{binding.WorkspaceID, binding.Selector}
		if _, exists := reader.bindings[key]; exists {
			return nil, errors.New("duplicate local REA selector in one Workspace")
		}
		reader.bindings[key] = binding
	}
	return reader, nil
}

// LocalREAReadResponse is for local Host-only use. It must not be registered as
// a Wails Server service or delivered remotely without a separate Disclosure
// check: the successful R2 view can contain private Source locations and IDs.
type LocalREAReadResponse struct {
	OK      bool                              `json:"ok"`
	Value   *operations.ExternalCandidateView `json:"value,omitempty"`
	Failure *Failure                          `json:"failure,omitempty"`
}

// Read selects only Host-configured Evidence, never a client-supplied file
// path. Root identity, traversal and bounded file reads reuse Studio's
// existing Workspace guards. This does not prove current Grant, source closure
// or external delivery; no remote transport is connected here.
func (r *LocalREAReader) Read(workspaceID, selector, observationID, relationID string, at time.Time) LocalREAReadResponse {
	if r == nil || r.host == nil {
		return LocalREAReadResponse{Failure: localREAReadFailure()}
	}
	root, denied := r.host.workspace(workspaceID)
	if denied != nil {
		return LocalREAReadResponse{Failure: denied}
	}
	binding, ok := r.bindings[localREAKey{workspaceID, selector}]
	if !ok {
		return LocalREAReadResponse{Failure: localREAReadFailure()}
	}
	raw, err := workspace.ReadSource(root, binding.File, localREAReadLimitBytes)
	if err != nil {
		// A missing, private or oversized Evidence must have the same public
		// failure, without filesystem paths, file-existence bits or sizes.
		return LocalREAReadResponse{Failure: localREAReadFailure()}
	}
	tool := binding.Tool
	tool.ResolvedPath = ""
	tool.AbsentReason = ""
	value, err := operations.ExternalCandidatesFromBytes(operations.ExternalCandidateRequest{
		ToolVersion:            binding.Version,
		ObservationID:          observationID,
		RelationID:             relationID,
		ExpectedEvidenceID:     binding.EvidenceID,
		ExpectedReceivedDigest: binding.Digest,
		ExpectedSourceRevision: binding.Revision,
		Tool:                   tool,
	}, raw, at)
	if err != nil {
		// R2 selection errors include external IDs and producer diagnostics.
		// Do not forward those details from a Host read boundary.
		return LocalREAReadResponse{Failure: localREAReadFailure()}
	}
	return LocalREAReadResponse{OK: true, Value: &value}
}

func localREAReadFailure() *Failure {
	return technicalFailure("OPERATION_FAILED", "external Evidence is unavailable")
}
