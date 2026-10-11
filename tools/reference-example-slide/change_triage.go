package exampleslide

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// ChangeTriage distinguishes a proposed correction to the Example from a
// request to edit TEI's authoritative semantic model. Neither creates a Patch.
type ChangeTriage struct {
	Schema            string             `json:"schema"`
	CaseID            string             `json:"caseId"`
	ModelRevision     string             `json:"modelRevision"`
	AnswerFingerprint string             `json:"answerFingerprint"`
	Items             []ChangeTriageItem `json:"items"`
}

type ChangeTriageItem struct {
	Question       QuestionRef `json:"question"`
	Route          string      `json:"route"`
	Reason         string      `json:"reason"`
	NextValidation []string    `json:"nextValidation"`
	CandidateReady bool        `json:"candidateReady"`
}

// TriageAnsweredReview rebuilds the exact answer binding, then returns only
// advice on the next *review* gate. The current shared acceptance reference
// supports add-relation but no typed update-relation or add-relation-key.
// Existing RelationKey omissions therefore cannot be silently converted into
// an add-relation operation with a new ID.
func TriageAnsweredReview(input Input, proposed AnswerInput) (ChangeTriage, error) {
	handoff, err := BindAnswers(input, proposed)
	if err != nil {
		return ChangeTriage{}, err
	}
	review, err := Review(input)
	if err != nil {
		return ChangeTriage{}, err
	}
	issues := make(map[string]ReviewIssue, len(review.Issues))
	for _, issue := range review.Issues {
		issues[issue.ID] = issue
	}
	out := ChangeTriage{
		Schema: "tei.reference.example-change-triage/v0",
		CaseID: handoff.CaseID, ModelRevision: handoff.ModelRevision,
		AnswerFingerprint: handoff.Fingerprint,
	}
	for _, answer := range handoff.Answers {
		item := ChangeTriageItem{Question: answer.Question}
		switch answer.Disposition {
		case "clarify-example":
			item.Route = "example-review"
			item.Reason = "correction-applies-to-illustrative-case-not-canonical-source"
			item.NextValidation = []string{"review revised example", "recompute model-bound Review fingerprint"}
		case "retain-model":
			item.Route = "confirmation-review"
			item.Reason = "no-canonical-edit-proposed"
			item.NextValidation = []string{"validate evidence and applicable context", "check existing Confirmation basis"}
		case "investigate":
			item.Route = "investigation"
			item.Reason = "insufficient-determination"
			item.NextValidation = []string{"collect independent supporting evidence", "review applicable context"}
		case "propose-model-change":
			item.Route = "semantic-change-review"
			item.Reason = "cause-and-typed-delta-not-qualified"
			item.NextValidation = []string{
				"identify the actual business rule or model defect",
				"confirm subject and Source Index evidence",
				"obtain current workspace revision and supported typed operation",
				"preview through existing Shared Acceptance; do not auto-accept",
			}
			if issue, ok := issues[answer.Question.ID]; answer.Question.Kind == "derived" && ok {
				if issue.Reason == "relation-keys-not-declared" {
					item.Reason = "existing-relation-key-edit-not-supported-by-current-typed-operations"
					item.NextValidation = []string{
						"confirm whether the missing keys are intentional",
						"do not use add-relation to replace an existing relation",
						"design and qualify a dedicated typed edit before proposing a Patch",
					}
				} else if issue.Reason == "multiple-targets-for-one" {
					item.Reason = "model-versus-example-cause-not-determined"
					item.NextValidation = []string{
						"determine whether business cardinality, key, or example data is incorrect",
						"check existing typed operations before proposing a delta",
					}
				}
			}
		}
		// CandidateReady remains false for every answer. A free-text answer
		// and unverified evidence can never authorize a typed source change.
		out.Items = append(out.Items, item)
	}
	return out, nil
}

// TriageFingerprint permits a downstream reviewer to bind their *review*
// response to the stable triage content. It grants no decision authority.
func TriageFingerprint(triage ChangeTriage) string {
	content, _ := json.Marshal(triage)
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}
