package documentdraft

import "strings"

// ReprojectionOption is what a consumer may do about one occurrence after the
// model moved. It is a *permission list*, not an instruction: nothing here
// applies itself, and no option overwrites author text.
//
// The vocabulary exists so a consumer does not have to improvise. Before this,
// a consumer received a list of occurrence IDs needing reconfirmation and had to
// decide on its own whether regenerating was allowed — which is exactly the
// decision that must not be made silently.
type ReprojectionOption string

const (
	// OptionKeepAuthor says the author's text stands and reprojection must not
	// touch it. This is the answer for every occurrence the author edited.
	OptionKeepAuthor ReprojectionOption = "keep-author"
	// OptionOfferRegeneration says the model moved and the author did not edit
	// this occurrence, so a consumer MAY offer a regenerated version. Offering
	// is not applying: the author still accepts or refuses.
	OptionOfferRegeneration ReprojectionOption = "offer-regeneration"
	// OptionHumanDecision says both the model and the author's text moved. No
	// automatic choice exists, and none is made here.
	OptionHumanDecision ReprojectionOption = "human-decision"
	// OptionReportRemoved says the model no longer derives this occurrence. The
	// document keeps whatever the author wrote; nothing is deleted.
	OptionReportRemoved ReprojectionOption = "report-removed"
	// OptionReportAdded says the model now derives an occurrence the stored
	// draft did not have. It is offered as an addition, never written over
	// existing text.
	OptionReportAdded ReprojectionOption = "report-added"
)

// ReprojectionProposal is the per-occurrence answer to "what may the consumer do
// now". It is derived from a ReconcileReport plus the author-edit facts the
// consumer already holds, and it performs no write.
type ReprojectionProposal struct {
	SchemaVersion   string                       `json:"schemaVersion"`
	Effect          string                       `json:"effect"`
	DraftID         string                       `json:"draftId"`
	ModelReference  ModelReferenceReconciliation `json:"modelReference"`
	Items           []ReprojectionItem           `json:"items"`
	AuthorityLimits []string                     `json:"authorityLimits"`
}

// ReprojectionItem is one occurrence's option and the reason for it.
type ReprojectionItem struct {
	OccurrenceID          string             `json:"occurrenceId"`
	Status                string             `json:"status"`
	Option                ReprojectionOption `json:"option"`
	Reason                string             `json:"reason"`
	RecordedContentDigest string             `json:"recordedContentDigest"`
	CurrentContentDigest  string             `json:"currentContentDigest,omitempty"`
}

// ReprojectionAuthorityLimits is the fixed statement of what a proposal is not.
func ReprojectionAuthorityLimits() []string {
	return []string{
		"reprojection-never-overwrites-author-text",
		"proposal-is-a-permission-list-not-an-instruction",
		"no-automatic-merge-between-author-text-and-model",
		"offering-a-regeneration-is-not-applying-one",
		"occurrence-id-is-snapshot-local-not-a-persistent-document-id",
	}
}

// AuthorEditFact is what the consumer knows about one occurrence in the document
// it holds. TEI does not read documents, so the consumer supplies this; an
// unknown value is reported as such rather than assumed untouched.
type AuthorEditFact struct {
	OccurrenceID string
	// Edited reports whether the author changed the generated block. A false
	// value from a consumer that cannot tell is a different fact from a false
	// value it can, so the caller says which it is.
	Edited bool
	// Known reports whether the consumer can tell. When false, Edited is not
	// evidence and the item is escalated to a human decision.
	Known bool
}

// ProposeReprojection turns a reconciliation into a permission list. It reads
// nothing and writes nothing: the report and the author-edit facts are inputs,
// and the output only names options.
func ProposeReprojection(report ReconcileReport, authorEdits map[string]AuthorEditFact) ReprojectionProposal {
	proposal := ReprojectionProposal{
		SchemaVersion:   "tei.document-draft.reprojection-proposal/v0",
		Effect:          EffectReadOnlyProjection,
		DraftID:         report.DraftID,
		ModelReference:  report.ModelReference,
		Items:           []ReprojectionItem{},
		AuthorityLimits: ReprojectionAuthorityLimits(),
	}
	for _, occurrence := range report.Occurrences {
		proposal.Items = append(proposal.Items, proposeItem(occurrence, authorEdits[occurrence.OccurrenceID]))
	}
	return proposal
}

// proposeItem names one occurrence's option. The rule is the policy: an
// author-edited occurrence is never offered for regeneration, because
// regenerating it would discard the author's text.
func proposeItem(occurrence OccurrenceReconciliation, fact AuthorEditFact) ReprojectionItem {
	item := ReprojectionItem{
		OccurrenceID:          occurrence.OccurrenceID,
		Status:                occurrence.Status,
		RecordedContentDigest: occurrence.RecordedContentDigest,
		CurrentContentDigest:  occurrence.CurrentContentDigest,
	}
	switch occurrence.Status {
	case OccurrenceUnchanged:
		item.Option = OptionKeepAuthor
		item.Reason = "the regenerated occurrence is identical to the stored one"
		return item
	case OccurrenceRemoved:
		item.Option = OptionReportRemoved
		item.Reason = "the model no longer derives this occurrence; the document keeps the author's text"
		return item
	case OccurrenceAdded:
		item.Option = OptionReportAdded
		item.Reason = "the model now derives this occurrence; it is offered as an addition"
		return item
	}
	// The model moved. Whether a regeneration may be offered depends entirely on
	// whether the author's text moved too, and on whether the consumer can tell.
	if !fact.Known {
		item.Option = OptionHumanDecision
		item.Reason = "the consumer cannot tell whether the author edited this occurrence"
		return item
	}
	if fact.Edited {
		item.Option = OptionKeepAuthor
		item.Reason = "the author edited this occurrence; reprojection must not touch it"
		return item
	}
	item.Option = OptionOfferRegeneration
	item.Reason = "the model moved and the author did not edit this occurrence"
	return item
}

// ProposeReprojectionFromDraft is the convenience entry: it reconciles the draft
// and immediately derives the proposal. The reconciliation is still read-only,
// so this writes nothing either.
func ProposeReprojectionFromDraft(draft Draft, root string,
	authorEdits map[string]AuthorEditFact,
) (ReprojectionProposal, error) {
	report, err := Reconcile(draft, root)
	if err != nil {
		return ReprojectionProposal{}, err
	}
	return ProposeReprojection(report, authorEdits), nil
}

// OptionIsValid reports whether this is one of the five options. The set is
// closed so a consumer cannot invent a sixth permission.
func OptionIsValid(option ReprojectionOption) bool {
	switch option {
	case OptionKeepAuthor, OptionOfferRegeneration, OptionHumanDecision,
		OptionReportRemoved, OptionReportAdded:
		return true
	}
	return false
}

// AllReprojectionOptions lists the closed set, so a consumer can assert it
// covers everything it implements.
func AllReprojectionOptions() []ReprojectionOption {
	return []ReprojectionOption{
		OptionKeepAuthor, OptionOfferRegeneration, OptionHumanDecision,
		OptionReportRemoved, OptionReportAdded,
	}
}

// optionOf trims and validates an option string. An unknown value is an error
// rather than a default, because a typo must not silently become a permission.
func optionOf(value string) (ReprojectionOption, error) {
	option := ReprojectionOption(strings.TrimSpace(value))
	if !OptionIsValid(option) {
		return "", draftError(CodeInvalidArgument,
			"reprojection option %q is not one of %v", value, AllReprojectionOptions())
	}
	return option, nil
}
