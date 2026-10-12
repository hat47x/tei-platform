package exampleslide

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const answerSchema = "tei.reference.example-review-answer/v0"

// AnswerInput is an untrusted answer to an exact derived ReviewPacket. It does
// not contain an accepted decision, a Semantic Patch, or evidence of authority.
type AnswerInput struct {
	Schema            string        `json:"schema"`
	CaseID            string        `json:"caseId"`
	ModelRevision     string        `json:"modelRevision"`
	ReviewFingerprint string        `json:"reviewFingerprint"`
	Answers           []AnswerEntry `json:"answers"`
}

type QuestionRef struct {
	Kind string `json:"kind"` // authored | derived
	ID   string `json:"id"`
}

type AnswerActor struct {
	Kind string `json:"kind"` // claimed human | ai (NOT authenticated)
	ID   string `json:"id"`
}

// EvidencePointer is a claim about where support may be found. Its presence
// does not mean the source, applicability, or fingerprint was verified.
type EvidencePointer struct {
	Kind        string `json:"kind"`
	Ref         string `json:"ref"`
	Fingerprint string `json:"fingerprint,omitempty"`
}

type AnswerEntry struct {
	Question    QuestionRef       `json:"question"`
	Response    string            `json:"response"`
	Disposition string            `json:"disposition"`
	Context     string            `json:"context"`
	ClaimedBy   AnswerActor       `json:"claimedBy"`
	Evidence    []EvidencePointer `json:"evidence,omitempty"`
}

// ConfirmationRoute is a *suggested* mapping into the existing
// reference-shared-acceptance Confirmation mechanism, not a ConfirmationInput
// and not authorization to call confirm-record. The workspace must resolve
// and fingerprint the suggested canonical basis separately.
type ConfirmationRoute struct {
	IdentityKind    string `json:"identityKind"`
	IdentitySubject string `json:"identitySubject"`
	BasisKind       string `json:"basisKind"`
	BasisID         string `json:"basisId"`
}

type BoundAnswer struct {
	Question          QuestionRef        `json:"question"`
	QuestionText      string             `json:"questionText"`
	// A snapshot of the validated derived issue, not a claim by the answerer.
	// Authored questions have no derived record provenance.
	SourceIssue       *ReviewIssue       `json:"sourceIssue,omitempty"`
	Response          string             `json:"response"`
	Disposition       string             `json:"disposition"`
	Context           string             `json:"context"`
	ClaimedBy         AnswerActor        `json:"claimedBy"`
	Evidence          []EvidencePointer  `json:"evidence,omitempty"`
	Status            string             `json:"status"` // always "unverified"
	NextStep          string             `json:"nextStep"`
	ConfirmationRoute *ConfirmationRoute `json:"confirmationRoute,omitempty"`
}

type AnswerHandoff struct {
	Schema            string        `json:"schema"`
	CaseID            string        `json:"caseId"`
	ModelRevision     string        `json:"modelRevision"`
	ReviewFingerprint string        `json:"reviewFingerprint"`
	Answers           []BoundAnswer `json:"answers"`
	Fingerprint       string        `json:"fingerprint"`
}

// BindAnswers derives an unverified, reviewable handoff for another component.
// It *rebuilds* the review and binds each answer to that exact immutable review.
// No Decision, ConfirmationRecord or authoritative source is created or changed.
func BindAnswers(input Input, answers AnswerInput) (AnswerHandoff, error) {
	var empty AnswerHandoff
	review, err := Review(input)
	if err != nil {
		return empty, err
	}
	if answers.Schema != answerSchema {
		return empty, fmt.Errorf("unsupported answer schema %q", answers.Schema)
	}
	if answers.CaseID != review.CaseID ||
		answers.ModelRevision != review.ModelRevision ||
		answers.ReviewFingerprint != review.Fingerprint {
		return empty, fmt.Errorf("stale or mismatched review answer binding")
	}
	if len(answers.Answers) == 0 {
		return empty, fmt.Errorf("no review answers")
	}
	derived := make(map[string]ReviewIssue, len(review.Issues))
	for _, issue := range review.Issues {
		derived[issue.ID] = issue
	}
	authored := make(map[string]Question, len(review.Authored))
	for _, q := range review.Authored {
		if q.ID == "" || strings.TrimSpace(q.Text) == "" {
			return empty, fmt.Errorf("invalid authored question")
		}
		if _, found := authored[q.ID]; found {
			return empty, fmt.Errorf("duplicate authored question %q", q.ID)
		}
		authored[q.ID] = q
	}
	out := AnswerHandoff{
		Schema: "tei.reference.example-review-handoff/v0",
		CaseID: review.CaseID, ModelRevision: review.ModelRevision,
		ReviewFingerprint: review.Fingerprint,
	}
	seen := map[string]bool{}
	for _, answer := range answers.Answers {
		key := answer.Question.Kind + "\x00" + answer.Question.ID
		if seen[key] {
			return empty, fmt.Errorf("duplicate answer for %s:%s", answer.Question.Kind, answer.Question.ID)
		}
		seen[key] = true
		response := strings.TrimSpace(answer.Response)
		context := strings.TrimSpace(answer.Context)
		actorID := strings.TrimSpace(answer.ClaimedBy.ID)
		if response == "" || context == "" || actorID == "" {
			return empty, fmt.Errorf("%s: response, context and actor identity are required", key)
		}
		if answer.ClaimedBy.Kind != "human" && answer.ClaimedBy.Kind != "ai" {
			return empty, fmt.Errorf("%s: unsupported claimed actor", key)
		}
		switch answer.Disposition {
		case "clarify-example", "propose-model-change", "retain-model", "investigate":
		default:
			return empty, fmt.Errorf("%s: unsupported disposition %q", key, answer.Disposition)
		}
		var questionText string
		var route *ConfirmationRoute
		var sourceIssue *ReviewIssue
		switch answer.Question.Kind {
		case "authored":
			q, found := authored[answer.Question.ID]
			if !found {
				return empty, fmt.Errorf("unknown authored question %q", answer.Question.ID)
			}
			questionText = q.Text
		case "derived":
			issue, found := derived[answer.Question.ID]
			if !found {
				return empty, fmt.Errorf("unknown or stale derived question %q", answer.Question.ID)
			}
			questionText = issue.Question
			// Copy the actual issue supplied by the rebuilt Review, including
			// its exact and possible target IDs. Never trust answer-supplied
			// record IDs and never infer these from the answer text.
			issueCopy := issue
			issueCopy.FrameIDs = append([]string(nil), issue.FrameIDs...)
			issueCopy.MatchedTargetIDs = append([]string(nil), issue.MatchedTargetIDs...)
			issueCopy.PossibleTargetIDs = append([]string(nil), issue.PossibleTargetIDs...)
			issueCopy.EvidenceNeeded = append([]string(nil), issue.EvidenceNeeded...)
			sourceIssue = &issueCopy
			if issue.RelationID != "" {
				// The existing Confirmation history checks Source Index
				// subjects, not the illustrative Example content. Its subject
				// must therefore change when the Example or Catalog changes,
				// even if issue.ID and the semantic source stay identical.
				route = &ConfirmationRoute{
					IdentityKind:    "example-review",
					IdentitySubject: scopedConfirmationSubject(issue, review),
					BasisKind:       "semantic-relation",
					BasisID:         issue.RelationID,
				}
			}
		default:
			return empty, fmt.Errorf("unsupported question kind %q", answer.Question.Kind)
		}
		evidence := make([]EvidencePointer, 0, len(answer.Evidence))
		evidenceIDs := map[string]bool{}
		for _, e := range answer.Evidence {
			e.Ref = strings.TrimSpace(e.Ref)
			e.Fingerprint = strings.TrimSpace(e.Fingerprint)
			if e.Kind != "source-index" && e.Kind != "document" && e.Kind != "observation" {
				return empty, fmt.Errorf("%s: unknown evidence kind %q", key, e.Kind)
			}
			if e.Ref == "" {
				return empty, fmt.Errorf("%s: evidence reference is required", key)
			}
			id := e.Kind + "\x00" + e.Ref
			if evidenceIDs[id] {
				return empty, fmt.Errorf("%s: duplicate evidence reference", key)
			}
			evidenceIDs[id] = true
			evidence = append(evidence, e)
		}
		sort.Slice(evidence, func(i, j int) bool {
			if evidence[i].Kind != evidence[j].Kind {
				return evidence[i].Kind < evidence[j].Kind
			}
			return evidence[i].Ref < evidence[j].Ref
		})
		next := "human-review-required"
		switch {
		case len(evidence) == 0:
			next = "evidence-needed"
		case answer.ClaimedBy.Kind == "ai":
			next = "human-review-required"
		case answer.Disposition == "investigate":
			next = "investigation-needed"
		case answer.Disposition == "propose-model-change":
			next = "evaluate-design-change"
		default:
			next = "confirmation-review-needed"
		}
		out.Answers = append(out.Answers, BoundAnswer{
			Question: answer.Question, QuestionText: questionText,
			SourceIssue: sourceIssue,
			Response: response, Context: context,
			Disposition: answer.Disposition,
			ClaimedBy:   AnswerActor{Kind: answer.ClaimedBy.Kind, ID: actorID},
			Evidence:    evidence, Status: "unverified",
			NextStep: next, ConfirmationRoute: route,
		})
	}
	sort.Slice(out.Answers, func(i, j int) bool {
		a, b := out.Answers[i].Question, out.Answers[j].Question
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.ID < b.ID
	})
	encoded, _ := json.Marshal(out) // Fingerprint is still empty.
	digest := sha256.Sum256(encoded)
	out.Fingerprint = "sha256:" + hex.EncodeToString(digest[:])
	return out, nil
}

// scopedConfirmationSubject supplements existing Confirmation's canonical
// basis fingerprint with the exact, immutable illustrative Case context.
// Returning the raw issue ID would suppress fresh review when the model
// sources are unchanged but the example record values have changed.
func scopedConfirmationSubject(issue ReviewIssue, review ReviewPacket) string {
	fields, _ := json.Marshal([]string{
		review.CaseID, review.ModelRevision,
		review.ExampleFingerprint, review.CatalogFingerprint,
		issue.ID, issue.Question,
	})
	sum := sha256.Sum256(fields)
	return "example-" + hex.EncodeToString(sum[:])
}
