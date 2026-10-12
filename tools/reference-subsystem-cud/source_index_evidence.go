package main

import (
	"bytes"
	"encoding/json"
	"sort"

	"tei/internal/sourceindex"
)

// IndexedProcedureEvidence is a read-only Source Index location, not a
// Shared Acceptance decision or a declaration of operational authority.
type IndexedProcedureEvidence struct {
	Status         string               `json:"status"`
	Subject        sourceindex.Subject  `json:"subject"`
	Reference      sourceindex.Reference `json:"reference"`
	Artifact       sourceindex.Artifact `json:"artifact"`
	AuthorityLimit string               `json:"authorityLimit"`
}

// InspectIndexedProcedureSnapshot checks a candidate against the exact
// Procedure bytes AND an existing Source Index. The caller supplies both the
// index and its source reader; this method cannot authenticate their origin.
// All indexed source files are read once per invocation to avoid mixing two
// revisions during index validation and Procedure byte comparison.
func InspectIndexedProcedureSnapshot(worldBytes, procedureBytes, factsBytes []byte, idx sourceindex.Index, procedurePath string, read func(string) ([]byte, error)) (Report, error) {
	report, err := InspectBoundProcedureSnapshot(worldBytes, procedureBytes, factsBytes)
	if err != nil {
		return report, err
	}
	unresolved := func(code, explanation string) (Report, error) {
		report.Diagnostics = append(report.Diagnostics, Diagnostic{
			Code: code, Severity: "unresolved", Subject: "procedure",
			Explanation: explanation,
		})
		sortDiagnostics(&report)
		report.Status = "unresolved"
		return report, nil
	}
	if read == nil {
		return unresolved("PROCEDURE_SOURCE_READER_UNAVAILABLE", "Source Index requires an exact-revision source reader")
	}
	path, err := sourceindex.NormalizeWorkspacePath(procedurePath)
	if err != nil || path != procedurePath {
		return unresolved("PROCEDURE_SOURCE_PATH_INVALID", "Procedure path must be normalized and workspace-relative")
	}

	// Freeze the source snapshot before checking the index and artifact,
	// without trusting the reader to return identical bytes on each call.
	contents := make(map[string][]byte, len(idx.Artifacts))
	readOnce := func(path string) ([]byte, error) {
		if content, found := contents[path]; found {
			return content, nil
		}
		content, err := read(path)
		if err != nil {
			return nil, err
		}
		contents[path] = bytes.Clone(content)
		return contents[path], nil
	}
	if err := idx.Validate(readOnce); err != nil {
		return unresolved("PROCEDURE_SOURCE_INDEX_INVALID", err.Error())
	}

	content, ok := contents[path]
	if !ok {
		return unresolved("PROCEDURE_SOURCE_UNINDEXED", "Procedure source is not present in the validated Source Index")
	}
	if !bytes.Equal(content, procedureBytes) {
		return unresolved("PROCEDURE_SOURCE_BYTES_MISMATCH", "indexed Procedure source bytes differ from the supplied candidate-bound bytes")
	}
	var fixture struct {
		Procedure struct {
			ID string `json:"id"`
		} `json:"procedure"`
	}
	if err := json.Unmarshal(procedureBytes, &fixture); err != nil || fixture.Procedure.ID == "" {
		return unresolved("PROCEDURE_SOURCE_SUBJECT_UNKNOWN", "Procedure source has no unambiguous procedure.id")
	}
	subject := sourceindex.Subject{Kind: "business-procedure-authoring", ID: fixture.Procedure.ID}
	var selected []sourceindex.Reference
	for _, ref := range idx.ReferencesFor(subject) {
		if ref.Role == sourceindex.RoleAuthoring || ref.Role == sourceindex.RoleDefinition {
			selected = append(selected, ref)
		}
	}
	if len(selected) != 1 || selected[0].Role != sourceindex.RoleAuthoring || selected[0].ArtifactPath != path {
		return unresolved("PROCEDURE_SOURCE_ANCHOR_UNRESOLVED", "exactly one matching authoring reference is required, without a competing definition")
	}
	var artifact sourceindex.Artifact
	for _, entry := range idx.Artifacts {
		if entry.Path == path {
			artifact = entry
			break
		}
	}
	if artifact.Fingerprint != sourceindex.Fingerprint(procedureBytes) {
		return unresolved("PROCEDURE_SOURCE_FINGERPRINT_MISMATCH", "indexed Procedure artifact fingerprint differs from the supplied bytes")
	}
	report.IndexedProcedure = &IndexedProcedureEvidence{
		Status: "located", Subject: subject, Reference: selected[0],
		Artifact: artifact, AuthorityLimit: "source-index-location-is-not-shared-acceptance-or-cud-authority",
	}
	return report, nil
}

// Shared by the Source Index extension only, which appends at most one
// additional diagnostic. The base inspector already sorts its own findings.
func sortDiagnostics(report *Report) {
	sort.Slice(report.Diagnostics, func(i, j int) bool {
		a, b := report.Diagnostics[i], report.Diagnostics[j]
		for k, value := range []string{a.Code, a.Subject, a.Source, a.Explanation} {
			other := []string{b.Code, b.Subject, b.Source, b.Explanation}[k]
			if value != other {
				return value < other
			}
		}
		return false
	})
}
