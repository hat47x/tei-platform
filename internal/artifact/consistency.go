package artifact

import (
	"fmt"
	"strings"

	"tei/internal/operations"
)

// Finding is one inconsistency inside a derived artifact set. An empty result
// means the set is consistent; it does not mean the artifacts are correct.
type Finding struct {
	Code    string `json:"code"`
	Subject string `json:"subject"`
	Detail  string `json:"detail"`
}

const (
	// FindingRevisionMismatch: an artifact was derived from another input revision.
	FindingRevisionMismatch = "ARTIFACT_REVISION_MISMATCH"
	// FindingManifestEntryMissing: the manifest does not list an artifact.
	FindingManifestEntryMissing = "MANIFEST_ENTRY_MISSING"
	// FindingSurfaceGapMissing: a surface gap is not carried as unresolved.
	FindingSurfaceGapMissing = "SURFACE_GAP_UNRESOLVED_MISSING"
	// FindingCompleteWithUnresolved: the set claims completeness while an
	// artifact still has unresolved entries.
	FindingCompleteWithUnresolved = "COMPLETE_WITH_UNRESOLVED"
	// FindingUnknownRule: an artifact names a rule that does not belong to its kind.
	FindingUnknownRule = "ARTIFACT_RULE_UNKNOWN"
	// FindingContentIncomplete: an artifact of a kind lost a line its reader relies on.
	FindingContentIncomplete = "ARTIFACT_CONTENT_INCOMPLETE"
	// FindingOperationIncomplete: a derived operation lost its basis or its
	// autonomy class.
	FindingOperationIncomplete = "SURFACE_OPERATION_INCOMPLETE"
	// FindingDuplicateArtifact: the same artifact appears twice in a set.
	FindingDuplicateArtifact = "ARTIFACT_DUPLICATE"
	// FindingManifestEntryStale: the manifest lists an entry no artifact produces.
	FindingManifestEntryStale = "MANIFEST_ENTRY_STALE"
	// FindingManifestTampered: the manifest digest does not match its entries.
	FindingManifestTampered = "MANIFEST_DIGEST_MISMATCH"
)

// CheckSet verifies the internal consistency of a rendered set: every artifact
// shares the input revision, the manifest lists every artifact, every gap of the
// operation surface is carried as unresolved, and a set with unresolved entries
// is not marked complete. It adds no meaning and changes nothing; callers decide
// what to do with a finding.
func CheckSet(set WorkspaceSet) []Finding {
	var findings []Finding

	for _, artifact := range set.Artifacts {
		if artifact.InputRevision != set.InputRevision {
			findings = append(findings, Finding{
				Code:    FindingRevisionMismatch,
				Subject: artifact.Kind,
				Detail:  fmt.Sprintf("artifact was derived from %s, set is %s", artifact.InputRevision, set.InputRevision),
			})
		}
	}

	if set.Manifest.Digest != "" {
		if err := set.Manifest.Verify(); err != nil {
			findings = append(findings, Finding{
				Code:    FindingManifestTampered,
				Subject: "manifest",
				Detail:  err.Error(),
			})
		}
	}

	listed := map[string]bool{}
	for _, entry := range set.Manifest.Entries {
		listed[entry.Kind+"\x00"+entry.Digest] = true
	}
	produced := map[string]bool{}
	for _, artifact := range set.Artifacts {
		key := artifact.Kind + "\x00" + artifact.Digest
		if produced[key] {
			findings = append(findings, Finding{
				Code:    FindingDuplicateArtifact,
				Subject: artifact.Kind,
				Detail:  "the same artifact appears twice in the set",
			})
		}
		produced[key] = true
	}
	for _, entry := range set.Manifest.Entries {
		if !produced[entry.Kind+"\x00"+entry.Digest] {
			findings = append(findings, Finding{
				Code:    FindingManifestEntryStale,
				Subject: entry.Kind,
				Detail:  "the manifest lists an artifact the set does not contain",
			})
		}
	}
	for _, artifact := range set.Artifacts {
		if !listed[artifact.Kind+"\x00"+artifact.Digest] {
			findings = append(findings, Finding{
				Code:    FindingManifestEntryMissing,
				Subject: artifact.Kind,
				Detail:  "the manifest does not list this artifact",
			})
		}
	}

	for _, artifact := range set.Artifacts {
		if artifact.Kind != string(KindOperationSurface) {
			continue
		}
		carried := map[string]bool{}
		for _, unresolved := range artifact.Unresolved {
			carried[unresolved.Subject] = true
		}
		payload, err := decodeOperationSurface(artifact.Content)
		if err == nil {
			for _, operation := range payload.Operations {
				if operation.Basis == "" || !knownAutonomy(operation.Autonomy) {
					findings = append(findings, Finding{
						Code:    FindingOperationIncomplete,
						Subject: operation.ID,
						Detail:  fmt.Sprintf("basis=%q autonomy=%q", operation.Basis, operation.Autonomy),
					})
				}
			}
		}
		for _, gap := range surfaceGaps(artifact) {
			if !carried[gap.Target] {
				findings = append(findings, Finding{
					Code:    FindingSurfaceGapMissing,
					Subject: gap.Target,
					Detail:  fmt.Sprintf("gap %s/%s is not carried as unresolved", gap.Kind, gap.Reason),
				})
			}
		}
	}

	for _, artifact := range set.Artifacts {
		for _, required := range requiredLines(artifact.Kind) {
			if !strings.Contains(artifact.Content, required) {
				findings = append(findings, Finding{
					Code:    FindingContentIncomplete,
					Subject: artifact.Kind,
					Detail:  fmt.Sprintf("content lost %q", required),
				})
			}
		}
		expected, ok := ruleForKind(artifact.Kind)
		if !ok {
			continue
		}
		if artifact.Rule.ID != expected {
			findings = append(findings, Finding{
				Code:    FindingUnknownRule,
				Subject: artifact.Kind,
				Detail:  fmt.Sprintf("artifact names rule %s, expected %s", artifact.Rule.ID, expected),
			})
		}
	}

	if set.Manifest.Complete {
		for _, artifact := range set.Artifacts {
			if len(artifact.Unresolved) > 0 {
				findings = append(findings, Finding{
					Code:    FindingCompleteWithUnresolved,
					Subject: artifact.Kind,
					Detail:  fmt.Sprintf("the set is complete but %s has %d unresolved entries", artifact.Kind, len(artifact.Unresolved)),
				})
			}
		}
	}

	return findings
}

// surfaceGaps reads the gaps back from the surface artifact content. The
// artifact is the only thing a consumer has, so the check reads the same bytes.
func surfaceGaps(artifact Artifact) []operations.SurfaceGap {
	payload, err := decodeOperationSurface(artifact.Content)
	if err != nil {
		return nil
	}
	return payload.Gaps
}

// ruleForKind names the one rule that may produce each kind. An artifact that
// names another rule is not rejected, but the mismatch is reported: a typo in a
// rule identifier would otherwise hide which rules produced the set.
func ruleForKind(kind string) (string, bool) {
	switch Kind(kind) {
	case KindTableDefinition:
		return TableDefinitionRuleID, true
	case KindFunctionDefinition:
		return FunctionDefinitionRuleID, true
	case KindOperationSurface:
		return OperationSurfaceRuleID, true
	}
	return "", false
}

// requiredLines names the line each kind must keep. A reader relies on these
// lines, so losing one is reported even when the artifact still verifies.
func requiredLines(kind string) []string {
	switch Kind(kind) {
	case KindTableDefinition:
		return []string{"table ", "data "}
	case KindFunctionDefinition:
		return []string{"application "}
	case KindOperationSurface:
		return []string{operations.OperationSurfaceSchemaVersion}
	}
	return nil
}

// knownAutonomy lists the autonomy classes a surface may declare. An unknown
// value would be read as "no gate" by a consumer, so it is reported.
func knownAutonomy(value string) bool {
	switch value {
	case "unattended", "attended", "human-gated":
		return true
	}
	return false
}
