package documentdraft

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"tei/internal/operations"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
)

// ReconcileReport compares a stored draft with the Workspace as it is now. It is
// read-only: it names what changed and what needs reconfirmation, and it never
// writes a Source, never updates a document and never overwrites author text.
type ReconcileReport struct {
	SchemaVersion       string                       `json:"schemaVersion"`
	Effect              string                       `json:"effect"`
	DraftID             string                       `json:"draftId"`
	ModelReference      ModelReferenceReconciliation `json:"modelReference"`
	Occurrences         []OccurrenceReconciliation   `json:"occurrences"`
	NeedsReconfirmation []string                     `json:"needsReconfirmation"`
	AuthorityLimits     []string                     `json:"authorityLimits"`
}

// ModelReferenceReconciliation reports whether the draft's model reference still
// describes the current read, and which of the three digests moved. A World
// change, a Source location change and a managed-Source byte change are three
// different facts and are reported separately.
type ModelReferenceReconciliation struct {
	Status   string   `json:"status"` // current | stale
	Diffs    []string `json:"diffs"`
	Observed ModelRef `json:"observed"`
}

// OccurrenceReconciliation is the verdict for one stored occurrence.
type OccurrenceReconciliation struct {
	OccurrenceID          string `json:"occurrenceId"`
	Status                string `json:"status"`
	Detail                string `json:"detail,omitempty"`
	RecordedContentDigest string `json:"recordedContentDigest"`
	CurrentContentDigest  string `json:"currentContentDigest,omitempty"`
}

// Occurrence statuses.
const (
	OccurrenceUnchanged         = "unchanged"
	OccurrenceContentChanged    = "content-changed"
	OccurrenceSubjectMissing    = "subject-missing"
	OccurrenceSubjectAmbiguous  = "subject-ambiguous"
	OccurrenceConditionsChanged = "generation-conditions-changed"
	OccurrenceUnknownRule       = "unknown-rule"
	OccurrenceUnsupportedKind   = "unsupported-subject-kind"
	// OccurrenceRemoved means the stored draft has an occurrence the current
	// projection no longer produces, because the model stopped deriving it.
	OccurrenceRemoved = "occurrence-removed"
	// OccurrenceAdded means the current projection produces an occurrence the
	// stored draft never had. It is reported so a reader learns a new fact
	// appeared, without treating it as a change to existing content.
	OccurrenceAdded = "occurrence-added"
)

// ModelReferenceStatus values.
const (
	ModelReferenceCurrent = "current"
	ModelReferenceStale   = "stale"
)

// ReconcileAuthorityLimits states what a reconciliation does not grant.
func ReconcileAuthorityLimits() []string {
	return []string{
		"reconcile-is-a-read-only-comparison",
		"reconcile-does-not-update-the-draft-or-the-document",
		"reconcile-does-not-overwrite-author-text",
		"unchanged-content-is-not-accepted-meaning",
		"model-change-requires-the-existing-change-plan-and-owner-acceptance-path",
	}
}

// Reconcile compares one stored draft with the current Workspace. The draft is
// validated first: a draft that does not carry this contract's schema, effect
// and occurrence shape is refused instead of partially compared.
func Reconcile(draft Draft, root string) (ReconcileReport, error) {
	if err := validateDraft(draft); err != nil {
		return ReconcileReport{}, err
	}
	if strings.TrimSpace(root) == "" || root != strings.TrimSpace(root) {
		return ReconcileReport{}, draftError(CodeInvalidArgument, "workspace path must be nonempty and exact")
	}
	if len(draft.Subjects) == 0 {
		return ReconcileReport{}, draftError(CodeInvalidDraft, "draft declares no subject")
	}
	// A document about an application (a CRUD matrix) is re-derived and then
	// compared occurrence by occurrence, because its rows come from the operation
	// surface rather than from one canonical declaration.
	if draft.DocumentKind == DocumentKindCRUDTable {
		return reconcileCRUDTable(draft, root)
	}
	read, err := readWorkspace(root, draft.Subjects[0].Subject)
	if err != nil {
		// A subject that no longer exists is a reconciliation answer, not a
		// failure of the comparison.
		var draftErr *DraftError
		if errors.As(err, &draftErr) && draftErr.Code == CodeSubjectNotFound {
			return ReconcileReport{
				SchemaVersion: "tei.document-draft.reconcile/v0",
				Effect:        EffectReadOnlyProjection,
				DraftID:       draft.DraftID,
				ModelReference: ModelReferenceReconciliation{
					Status: ModelReferenceStale,
					Diffs:  []string{"subject-missing"},
				},
				Occurrences:         allOccurrencesMissing(draft),
				NeedsReconfirmation: occurrenceIDs(draft),
				AuthorityLimits:     ReconcileAuthorityLimits(),
			}, nil
		}
		return ReconcileReport{}, err
	}
	inspection, err := read.inspect(draft.Subjects[0].Subject)
	if err != nil {
		return ReconcileReport{}, err
	}

	report := ReconcileReport{
		SchemaVersion: "tei.document-draft.reconcile/v0",
		Effect:        EffectReadOnlyProjection,
		DraftID:       draft.DraftID,
		ModelReference: ModelReferenceReconciliation{
			Observed: ModelRef{
				Workspace:              inspection.Workspace,
				WorldFingerprint:       inspection.WorldFingerprint,
				SourceIndexFingerprint: inspection.SourceIndexFingerprint,
				ModelRevision:          read.modelRevision,
			},
		},
		AuthorityLimits: ReconcileAuthorityLimits(),
	}
	report.ModelReference.Diffs = modelReferenceDiffs(draft.Model, report.ModelReference.Observed)
	if len(report.ModelReference.Diffs) == 0 {
		report.ModelReference.Status = ModelReferenceCurrent
	} else {
		report.ModelReference.Status = ModelReferenceStale
	}

	for _, occurrence := range draft.Occurrences {
		status, detail, currentDigest := reconcileOccurrence(read, occurrence)
		report.Occurrences = append(report.Occurrences, OccurrenceReconciliation{
			OccurrenceID:          occurrence.OccurrenceID,
			Status:                status,
			Detail:                detail,
			RecordedContentDigest: occurrence.ContentDigest,
			CurrentContentDigest:  currentDigest,
		})
		if status != OccurrenceUnchanged {
			report.NeedsReconfirmation = append(report.NeedsReconfirmation, occurrence.OccurrenceID)
		}
	}
	if report.NeedsReconfirmation == nil {
		report.NeedsReconfirmation = []string{}
	}
	return report, nil
}

// reconcileCRUDTable compares a stored CRUD draft with the projection the
// current Workspace produces. The comparison is read-only and never writes the
// draft, the document or a Source.
func reconcileCRUDTable(draft Draft, root string) (ReconcileReport, error) {
	application := ""
	if draft.Scope != nil {
		application = draft.Scope.Application
	}
	if application == "" {
		return ReconcileReport{}, draftError(CodeInvalidDraft,
			"a crud-table draft must record the application it was derived for")
	}
	current, err := buildCRUDDraft(Request{
		Root:         root,
		DocumentKind: DocumentKindCRUDTable,
		Application:  application,
		Title:        draft.Title,
	})
	if err != nil {
		var draftErr *DraftError
		if errors.As(err, &draftErr) &&
			(draftErr.Code == CodeApplicationNotFound || draftErr.Code == CodeSubjectNotFound) {
			return ReconcileReport{
				SchemaVersion: "tei.document-draft.reconcile/v0",
				Effect:        EffectReadOnlyProjection,
				DraftID:       draft.DraftID,
				ModelReference: ModelReferenceReconciliation{
					Status: ModelReferenceStale,
					Diffs:  []string{"application-missing"},
				},
				Occurrences:         allOccurrencesMissing(draft),
				NeedsReconfirmation: occurrenceIDs(draft),
				AuthorityLimits:     ReconcileAuthorityLimits(),
			}, nil
		}
		return ReconcileReport{}, err
	}

	report := ReconcileReport{
		SchemaVersion: "tei.document-draft.reconcile/v0",
		Effect:        EffectReadOnlyProjection,
		DraftID:       draft.DraftID,
		ModelReference: ModelReferenceReconciliation{
			Observed: current.Model,
			Diffs:    modelReferenceDiffs(draft.Model, current.Model),
		},
		AuthorityLimits: ReconcileAuthorityLimits(),
	}
	if len(report.ModelReference.Diffs) == 0 {
		report.ModelReference.Status = ModelReferenceCurrent
	} else {
		report.ModelReference.Status = ModelReferenceStale
	}

	currentByID := map[string]Occurrence{}
	for _, occurrence := range current.Occurrences {
		currentByID[occurrence.OccurrenceID] = occurrence
	}
	recordedIDs := map[string]bool{}
	for _, occurrence := range draft.Occurrences {
		recordedIDs[occurrence.OccurrenceID] = true
		regenerated, ok := currentByID[occurrence.OccurrenceID]
		if !ok {
			report.Occurrences = append(report.Occurrences, OccurrenceReconciliation{
				OccurrenceID:          occurrence.OccurrenceID,
				Status:                OccurrenceRemoved,
				Detail:                "the current projection no longer produces this occurrence",
				RecordedContentDigest: occurrence.ContentDigest,
			})
			report.NeedsReconfirmation = append(report.NeedsReconfirmation, occurrence.OccurrenceID)
			continue
		}
		status := OccurrenceUnchanged
		detail := ""
		switch {
		case regenerated.ContentDigest != occurrence.ContentDigest:
			status = OccurrenceContentChanged
			detail = "the regenerated occurrence differs from the stored one"
		case !sameDigests(occurrence.GenerationConditions.InputDigests,
			regenerated.GenerationConditions.InputDigests):
			status = OccurrenceConditionsChanged
			detail = "content is identical but the model reference it was read from moved"
		}
		report.Occurrences = append(report.Occurrences, OccurrenceReconciliation{
			OccurrenceID:          occurrence.OccurrenceID,
			Status:                status,
			Detail:                detail,
			RecordedContentDigest: occurrence.ContentDigest,
			CurrentContentDigest:  regenerated.ContentDigest,
		})
		if status != OccurrenceUnchanged {
			report.NeedsReconfirmation = append(report.NeedsReconfirmation, occurrence.OccurrenceID)
		}
	}
	// An occurrence the stored draft never had is reported too: a new fact
	// appeared, and a reader must see it rather than trust the stored list.
	for _, occurrence := range current.Occurrences {
		if recordedIDs[occurrence.OccurrenceID] {
			continue
		}
		report.Occurrences = append(report.Occurrences, OccurrenceReconciliation{
			OccurrenceID:         occurrence.OccurrenceID,
			Status:               OccurrenceAdded,
			Detail:               "the current projection produces this occurrence and the stored draft did not",
			CurrentContentDigest: occurrence.ContentDigest,
		})
		report.NeedsReconfirmation = append(report.NeedsReconfirmation, occurrence.OccurrenceID)
	}
	if report.NeedsReconfirmation == nil {
		report.NeedsReconfirmation = []string{}
	}
	return report, nil
}

func modelReferenceDiffs(recorded, observed ModelRef) []string {
	var diffs []string
	if recorded.WorldFingerprint != observed.WorldFingerprint {
		diffs = append(diffs, "worldFingerprint")
	}
	if recorded.SourceIndexFingerprint != observed.SourceIndexFingerprint {
		diffs = append(diffs, "sourceIndexFingerprint")
	}
	if recorded.ModelRevision != observed.ModelRevision {
		diffs = append(diffs, "modelRevision")
	}
	sort.Strings(diffs)
	return diffs
}

// reconcileOccurrence re-derives ONE occurrence from the current read using the
// rule the draft recorded. A rule this version does not know is reported as
// unknown rather than guessed.
func reconcileOccurrence(read workspaceRead, occurrence Occurrence) (status, detail, currentDigest string) {
	if occurrence.GenerationConditions.Generator != GeneratorName ||
		occurrence.GenerationConditions.GeneratorVersion != GeneratorVersion {
		return OccurrenceUnknownRule,
			"recorded generator " + occurrence.GenerationConditions.Generator + "/" +
				occurrence.GenerationConditions.GeneratorVersion + " is not this generator", ""
	}
	rule := occurrence.GenerationConditions.Rule
	if !knownRule(rule) {
		return OccurrenceUnknownRule, "rule " + rule + " is not known to this version", ""
	}
	if len(occurrence.SubjectRefs) == 0 {
		return OccurrenceSubjectMissing, "occurrence references no subject", ""
	}
	subject := occurrence.SubjectRefs[0]
	if !supportedKind(subject.Kind) {
		return OccurrenceUnsupportedKind, "subject kind " + subject.Kind + " is not projected by this version", ""
	}
	inspection, err := read.inspect(subject)
	if err != nil {
		var draftErr *DraftError
		if errors.As(err, &draftErr) {
			switch draftErr.Code {
			case CodeSubjectNotFound:
				return OccurrenceSubjectMissing, draftErr.Message, ""
			case CodeSubjectAmbiguous:
				return OccurrenceSubjectAmbiguous, draftErr.Message, ""
			}
		}
		return OccurrenceSubjectMissing, err.Error(), ""
	}

	digests := inputDigests(inspection, read)
	regenerated, err := regenerate(read, inspection, subject, occurrence, digests)
	if err != nil {
		return OccurrenceUnknownRule, err.Error(), ""
	}
	digest := contentDigest(regenerated)
	switch {
	case digest != occurrence.ContentDigest:
		return OccurrenceContentChanged, "regenerated content differs from the recorded draft", digest
	case !sameDigests(occurrence.GenerationConditions.InputDigests, digests):
		return OccurrenceConditionsChanged,
			"content is identical but the model reference it was read from moved", digest
	default:
		return OccurrenceUnchanged, "", digest
	}
}

// regenerate rebuilds one occurrence's content with the recorded rule. It uses
// the same generators Build uses, so an unchanged digest means the same bytes.
func regenerate(
	read workspaceRead, inspection operations.SubjectInspection,
	subject sourceindex.Subject, occurrence Occurrence, digests map[string]string,
) (OccurrenceContent, error) {
	switch occurrence.GenerationConditions.Rule {
	case "element-summary-v0":
		switch subject.Kind {
		case "semantic-data":
			var data semantic.Data
			if err := json.Unmarshal(inspection.Canonical, &data); err != nil {
				return OccurrenceContent{}, err
			}
			return OccurrenceContent{Text: dataExplanation(subject, data)}, nil
		case "semantic-type":
			var typ semantic.Type
			if err := json.Unmarshal(inspection.Canonical, &typ); err != nil {
				return OccurrenceContent{}, err
			}
			return OccurrenceContent{Text: typeExplanation(subject, typ)}, nil
		case "semantic-capability":
			var capability semantic.Capability
			if err := json.Unmarshal(inspection.Canonical, &capability); err != nil {
				return OccurrenceContent{}, err
			}
			return OccurrenceContent{Text: capabilityExplanation(subject, capability)}, nil
		}
	case "data-attributes-v0":
		var data semantic.Data
		if err := json.Unmarshal(inspection.Canonical, &data); err != nil {
			return OccurrenceContent{}, err
		}
		return OccurrenceContent{Table: attributeTable(data)}, nil
	case "type-shape-v0":
		var typ semantic.Type
		if err := json.Unmarshal(inspection.Canonical, &typ); err != nil {
			return OccurrenceContent{}, err
		}
		return OccurrenceContent{Table: typeTable(typ)}, nil
	case "capability-shape-v0":
		var capability semantic.Capability
		if err := json.Unmarshal(inspection.Canonical, &capability); err != nil {
			return OccurrenceContent{}, err
		}
		return OccurrenceContent{Table: capabilityTable(capability)}, nil
	case "relation-summary-v0":
		if occurrence.Content.Relation == nil {
			return OccurrenceContent{}, draftError(CodeInvalidDraft,
				"relation occurrence %s carries no relation content", occurrence.OccurrenceID)
		}
		relation, err := regenerateRelation(read, inspection, subject, occurrence.Content.Relation.Relation.ID)
		if err != nil {
			return OccurrenceContent{}, err
		}
		return OccurrenceContent{Relation: &relation}, nil
	case "source-citation-v0":
		if occurrence.Content.Citation == nil {
			return OccurrenceContent{}, draftError(CodeInvalidDraft,
				"citation occurrence %s carries no citation content", occurrence.OccurrenceID)
		}
		return OccurrenceContent{Citation: citationOf(inspection, sourceindex.Artifact{
			Path:        occurrence.Content.Citation.ArtifactPath,
			Fingerprint: occurrence.Content.Citation.Fingerprint,
		})}, nil
	}
	return OccurrenceContent{}, draftError(CodeInvalidDraft,
		"rule %q is not known to this version", occurrence.GenerationConditions.Rule)
}

// regenerateRelation re-reads the declared relation from the current World. A
// relation that is no longer declared is a change, not a silent drop.
func regenerateRelation(
	read workspaceRead, inspection operations.SubjectInspection,
	from sourceindex.Subject, relationID string,
) (RelationRef, error) {
	var data semantic.Data
	if err := json.Unmarshal(inspection.Canonical, &data); err != nil {
		return RelationRef{}, err
	}
	relation, _, err := resolveRelation(read, data, from, relationID)
	if err != nil {
		return RelationRef{}, err
	}
	return relation, nil
}

func knownRule(rule string) bool {
	switch rule {
	case "element-summary-v0", "data-attributes-v0", "type-shape-v0",
		"capability-shape-v0", "relation-summary-v0", "source-citation-v0":
		return true
	}
	return false
}

func supportedKind(kind string) bool {
	for _, supported := range SupportedElementKinds() {
		if kind == supported {
			return true
		}
	}
	return false
}

func sameDigests(recorded, current map[string]string) bool {
	if len(recorded) != len(current) {
		return false
	}
	for key, value := range recorded {
		other, ok := current[key]
		if !ok || other != value {
			return false
		}
	}
	return true
}

// validateDraft refuses a document that does not carry this contract. A partial
// or hand-edited draft must not be completed by guessing.
func validateDraft(draft Draft) error {
	if draft.SchemaVersion != SchemaVersion {
		return draftError(CodeInvalidDraft,
			"draft schemaVersion %q is not %q", draft.SchemaVersion, SchemaVersion)
	}
	if draft.Effect != EffectReadOnlyProjection {
		return draftError(CodeInvalidDraft,
			"draft effect %q is not %q", draft.Effect, EffectReadOnlyProjection)
	}
	if strings.TrimSpace(draft.DraftID) == "" {
		return draftError(CodeInvalidDraft, "draft has no draftId")
	}
	if strings.TrimSpace(draft.DocumentKind) == "" {
		return draftError(CodeInvalidDraft, "draft has no documentKind")
	}
	if len(draft.Occurrences) == 0 {
		return draftError(CodeInvalidDraft, "draft has no occurrence")
	}
	seen := map[string]bool{}
	for _, occurrence := range draft.Occurrences {
		if strings.TrimSpace(occurrence.OccurrenceID) == "" {
			return draftError(CodeInvalidDraft, "draft has an occurrence without an id")
		}
		if seen[occurrence.OccurrenceID] {
			return draftError(CodeInvalidDraft, "draft repeats occurrence id %q", occurrence.OccurrenceID)
		}
		seen[occurrence.OccurrenceID] = true
		if strings.TrimSpace(occurrence.Kind) == "" {
			return draftError(CodeInvalidDraft, "occurrence %q has no kind", occurrence.OccurrenceID)
		}
		if strings.TrimSpace(occurrence.GenerationConditions.Rule) == "" {
			return draftError(CodeInvalidDraft, "occurrence %q has no generation rule", occurrence.OccurrenceID)
		}
		if strings.TrimSpace(occurrence.ContentDigest) == "" {
			return draftError(CodeInvalidDraft, "occurrence %q has no content digest", occurrence.OccurrenceID)
		}
		if len(occurrence.SubjectRefs) == 0 {
			return draftError(CodeInvalidDraft, "occurrence %q references no subject", occurrence.OccurrenceID)
		}
	}
	return nil
}

func allOccurrencesMissing(draft Draft) []OccurrenceReconciliation {
	out := make([]OccurrenceReconciliation, 0, len(draft.Occurrences))
	for _, occurrence := range draft.Occurrences {
		out = append(out, OccurrenceReconciliation{
			OccurrenceID:          occurrence.OccurrenceID,
			Status:                OccurrenceSubjectMissing,
			Detail:                "the subject this occurrence was read from is no longer declared",
			RecordedContentDigest: occurrence.ContentDigest,
		})
	}
	return out
}

func occurrenceIDs(draft Draft) []string {
	out := make([]string, 0, len(draft.Occurrences))
	for _, occurrence := range draft.Occurrences {
		out = append(out, occurrence.OccurrenceID)
	}
	return out
}
