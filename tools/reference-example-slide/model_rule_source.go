package exampleslide

import (
	"bytes"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"

	"tei/internal/sourceindex"
)

const (
	ruleSourceHeader = "tei-model-rules v0"
	maxRuleSourceBytes = 256 * 1024
	maxRuleSourceEntries = 128
)

var ruleSourceTokens = regexp.MustCompile(`\S+`)

// ModelRuleSourceEntry locates an explicitly authored, unaccepted rule.
type ModelRuleSourceEntry struct {
	RuleID string `json:"ruleId"`
	Kind string `json:"kind"`
	DraftStatus string `json:"draftStatus"`
	SourceRange sourceindex.Range `json:"sourceRange"`
}

// ModelRuleSourceReview indexes proposed rule identities in a separate,
// non-canonical reference Source Index. It does not alter Workspace Source Index.
type ModelRuleSourceReview struct {
	Schema string `json:"schema"`
	DraftFingerprint string `json:"draftFingerprint"`
	Fingerprint string `json:"fingerprint"`
	SourceIndex sourceindex.Index `json:"sourceIndex"`
	Entries []ModelRuleSourceEntry `json:"entries"`
	Unresolved []string `json:"unresolved"`
}

// ReviewLoadedWorkspaceRuleSource always reconstructs the typed draft from
// current Workspace semantics and Source Index before examining authored rules.
// The supplied rule file is external, unaccepted input, NOT a parsed Workspace
// source snapshot and NOT evidence of approval.
func ReviewLoadedWorkspaceRuleSource(root string, input Input, hypotheses ModelHypotheses, intent ModelRuleDraftIntent, path string, content []byte) (ModelRuleSourceReview, error) {
	draft, err := DraftLoadedWorkspaceModelRules(root, input, hypotheses, intent)
	if err != nil {
		return ModelRuleSourceReview{}, err
	}
	return IndexModelRuleDraftSource(draft, path, content)
}

// IndexModelRuleDraftSource validates an untrusted proposal-side rule file,
// matching every line to exactly one typed declaration from a checked draft.
// It builds only reference-authoring locators; no semantic mutation is emitted.
func IndexModelRuleDraftSource(draft ModelRuleDeclarationDraft, sourcePath string, content []byte) (ModelRuleSourceReview, error) {
	var empty ModelRuleSourceReview
	if draft.Schema != "tei.reference.model-rule-declaration-draft/v0" ||
		!validModelEvidenceFingerprint(draft.Fingerprint) ||
		!validModelEvidenceFingerprint(draft.ModelReviewFingerprint) ||
		!validModelEvidenceFingerprint(draft.SourceAuditFingerprint) ||
		draft.Fingerprint != digestHypothesisJSON(ModelRuleDeclarationDraft{
			Schema: draft.Schema,
			CaseID: draft.CaseID,
			ModelRevision: draft.ModelRevision,
			ModelReviewFingerprint: draft.ModelReviewFingerprint,
			SourceAuditFingerprint: draft.SourceAuditFingerprint,
			Declarations: draft.Declarations,
		}) {
		return empty, fmt.Errorf("model rule draft schema or self-fingerprint mismatch")
	}
	if len(draft.Declarations) == 0 || len(draft.Declarations) > maxRuleSourceEntries {
		return empty, fmt.Errorf("expected 1 to %d model declarations", maxRuleSourceEntries)
	}
	normalized, err := sourceindex.NormalizeWorkspacePath(sourcePath)
	if err != nil || normalized != sourcePath {
		return empty, fmt.Errorf("rule source path must be normalized and relative")
	}
	if len(content) == 0 || len(content) > maxRuleSourceBytes || bytes.IndexByte(content, 0) >= 0 {
		return empty, fmt.Errorf("rule source is empty, oversized, or contains NUL")
	}
	artifact, err := sourceindex.NewArtifact(sourcePath, content)
	if err != nil {
		return empty, err
	}
	desired := make(map[string]ModelRuleDeclaration, len(draft.Declarations))
	for _, d := range draft.Declarations {
		if d.ProposedRuleID == "" || desired[d.ProposedRuleID].ProposedRuleID != "" {
			return empty, fmt.Errorf("duplicate or empty rule draft ID %q", d.ProposedRuleID)
		}
		desired[d.ProposedRuleID] = d
	}
	index := sourceindex.Index{
		Artifacts: []sourceindex.Artifact{artifact},
		References: make([]sourceindex.Reference, 0, len(desired)),
	}
	entries := make([]ModelRuleSourceEntry, 0, len(desired))
	first, seen := false, make(map[string]bool, len(desired))
	for at := 0; at < len(content); {
		end := bytes.IndexByte(content[at:], '\n')
		if end < 0 {
			end = len(content)
		} else {
			end += at
		}
		line := content[at:end]
		positions := ruleSourceTokens.FindAllIndex(line, -1)
		parts := make([]string, 0, len(positions))
		for _, span := range positions {
			parts = append(parts, string(line[span[0]:span[1]]))
		}
		if len(parts) != 0 && !strings.HasPrefix(parts[0], "#") {
			if !first {
				if len(parts) != 2 || strings.Join(parts, " ") != ruleSourceHeader {
					return empty, fmt.Errorf("rule source requires exact header %q", ruleSourceHeader)
				}
				first = true
			} else {
				if len(parts) < 4 || len(entries) >= maxRuleSourceEntries {
					return empty, fmt.Errorf("invalid rule declaration at byte %d", at)
				}
				id := parts[1]
				d, ok := desired[id]
				if !ok || seen[id] {
					return empty, fmt.Errorf("unexpected or duplicate source rule %q", id)
				}
				if parts[0] != d.Kind || parts[2] != "on" || parts[3] != d.Table {
					return empty, fmt.Errorf("rule %s kind or Data differs from reviewed draft", id)
				}
				switch d.Kind {
				case "candidate-key":
					if len(parts) != 6 || parts[4] != "fields" || !sameRuleFieldSet(parts[5], d.KeyFields) {
						return empty, fmt.Errorf("candidate key %s attributes differ from reviewed draft", id)
					}
				case "functional-dependency":
					if len(parts) != 8 || parts[4] != "determinants" || parts[6] != "dependents" ||
						!sameRuleFieldSet(parts[5], d.Determinants) || !sameRuleFieldSet(parts[7], d.Dependents) {
						return empty, fmt.Errorf("functional dependency %s attributes differ from reviewed draft", id)
					}
				default:
					return empty, fmt.Errorf("unsupported rule kind %q", d.Kind)
				}
				rng, err := sourceindex.RangeFromOffsets(content, at+positions[1][0], at+positions[1][1])
				if err != nil {
					return empty, err
				}
				index.References = append(index.References, sourceindex.Reference{
					Subject: sourceindex.Subject{Kind: "reference-" + d.Kind, ID: id},
					ArtifactPath: sourcePath,
					Range: rng,
					Role: sourceindex.RoleAuthoring,
				})
				entries = append(entries, ModelRuleSourceEntry{
					RuleID: id, Kind: d.Kind, DraftStatus: d.Status, SourceRange: rng,
				})
				seen[id] = true
			}
		}
		if end == len(content) {
			break
		}
		at = end + 1
	}
	if !first || len(seen) != len(desired) {
		return empty, fmt.Errorf("source declares %d of %d reviewed rules", len(seen), len(desired))
	}
	sort.Slice(index.References, func(i, j int) bool {
		return index.References[i].Subject.ID < index.References[j].Subject.ID
	})
	sort.Slice(entries, func(i, j int) bool { return entries[i].RuleID < entries[j].RuleID })
	read := func(p string) ([]byte, error) {
		if p != sourcePath {
			return nil, fmt.Errorf("source outside this unaccepted authoring artifact: %s", p)
		}
		return bytes.Clone(content), nil
	}
	if err := index.Validate(read); err != nil {
		return empty, fmt.Errorf("reference source index: %w", err)
	}
	for _, ref := range index.References {
		span := content[ref.Range.Start.Offset:ref.Range.End.Offset]
		if string(span) != ref.Subject.ID {
			return empty, fmt.Errorf("rule source reference does not point to the exact rule identity")
		}
	}
	out := ModelRuleSourceReview{
		Schema: "tei.reference.model-rule-source-review/v0",
		DraftFingerprint: draft.Fingerprint,
		SourceIndex: index,
		Entries: entries,
		Unresolved: []string{
			"rule file is unaccepted external authoring input, not a Workspace Semantic declaration",
			"business validity, actor authority, and Shared Acceptance are not established",
		},
	}
	out.Fingerprint = digestHypothesisJSON(out)
	return out, nil
}

func sameRuleFieldSet(raw string, expected []string) bool {
	fields := strings.Split(raw, ",")
	if len(fields) == 0 || len(fields) != len(expected) {
		return false
	}
	for _, field := range fields {
		if field == "" || strings.TrimSpace(field) != field {
			return false
		}
	}
	sort.Strings(fields)
	for i := 1; i < len(fields); i++ {
		if fields[i-1] == fields[i] {
			return false
		}
	}
	sortedExpected := slices.Clone(expected)
	sort.Strings(sortedExpected)
	return slices.Equal(fields, sortedExpected)
}
