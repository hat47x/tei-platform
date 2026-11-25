// Package cognitionhandoff converts one cognition proposal into a Shared
// Acceptance change candidate. It writes a proposal artifact; it does not
// preview, review, accept, or apply it.
package cognitionhandoff

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

const (
	ProposalSchema  = "tei.reference.cognition-proposal/v0"
	CandidateSchema = "tei.reference.change-candidate/v0"
)

type ProposalCandidate struct {
	ID          string         `json:"id"`
	Kind        string         `json:"kind"`
	Subject     string         `json:"subject"`
	Payload     map[string]any `json:"payload"`
	Basis       []string       `json:"basis"`
	Assumptions []string       `json:"assumptions,omitempty"`
}

type Proposal struct {
	Schema     string              `json:"schema"`
	State      string              `json:"state"`
	Candidates []ProposalCandidate `json:"candidates"`
}

type Origin struct {
	Kind  string            `json:"kind"`
	Actor string            `json:"actor,omitempty"`
	Notes map[string]string `json:"notes,omitempty"`
}

// ProposalBinding is the currentness binding a cognition receipt records: the
// subject, its revision, and the judgement basis in force. #5808 and #5809 both
// require that a candidate produced from an older revision or an older criteria
// set never reaches the current selection, so the handoff carries the binding
// into the change candidate instead of leaving a consumer to reconstruct it.
type ProposalBinding struct {
	Subject  string `json:"subject"`
	Revision string `json:"revision"`
	Basis    string `json:"basis,omitempty"`
}

type Candidate struct {
	Schema            string           `json:"schema"`
	ID                string           `json:"id"`
	BaseModelRevision string           `json:"baseModelRevision"`
	Origin            Origin           `json:"origin"`
	Intent            string           `json:"intent"`
	Delta             []Operation      `json:"delta"`
	Binding           *ProposalBinding `json:"binding,omitempty"`
}

// Operation is one delta entry a candidate asks the acceptance path to add.
type Operation struct {
	Op        string `json:"op"`
	Namespace string `json:"namespace"`
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Attribute string `json:"attribute"`
}

var segmentPattern = regexp.MustCompile(`[^a-z0-9]+`)

func canonicalSegment(value string) string {
	parts := segmentPattern.Split(strings.ToLower(strings.TrimSpace(value)), -1)
	var cleaned []string
	for _, part := range parts {
		if part != "" {
			cleaned = append(cleaned, part)
		}
	}
	if len(cleaned) == 0 {
		return "candidate"
	}
	joined := strings.Join(cleaned, "_")
	if joined[0] < 'a' || joined[0] > 'z' {
		joined = "c_" + joined
	}
	return joined
}

func Handoff(proposalPath, workspaceDir, providerID, model, version, actor, candidateID string) (Candidate, error) {
	return HandoffWithBinding(proposalPath, workspaceDir, providerID, model, version, actor, candidateID, nil)
}

// HandoffWithBinding is Handoff with the proposal's currentness binding carried
// into the change candidate. A nil binding keeps the pre-#5808 behaviour: the
// candidate is produced but states no revision, and RequireCurrentAtHandoff
// then refuses to treat it as current.
func HandoffWithBinding(proposalPath, workspaceDir, providerID, model, version, actor, candidateID string, binding *ProposalBinding) (Candidate, error) {
	raw, err := os.ReadFile(proposalPath)
	if err != nil {
		return Candidate{}, err
	}
	var proposal Proposal
	if err := json.Unmarshal(raw, &proposal); err != nil {
		return Candidate{}, fmt.Errorf("%s: %w", proposalPath, err)
	}
	if proposal.Schema != ProposalSchema {
		return Candidate{}, fmt.Errorf("proposal schema %q is not %q", proposal.Schema, ProposalSchema)
	}
	if proposal.State != "proposed" {
		return Candidate{}, fmt.Errorf("proposal state %q is not proposed", proposal.State)
	}
	if len(proposal.Candidates) == 0 {
		return Candidate{}, fmt.Errorf("proposal has no candidates")
	}
	var selected *ProposalCandidate
	for i := range proposal.Candidates {
		candidate := &proposal.Candidates[i]
		if candidateID != "" && candidate.ID != candidateID {
			continue
		}
		selected = candidate
		break
	}
	if selected == nil {
		return Candidate{}, fmt.Errorf("candidate %q is not in the proposal", candidateID)
	}
	if selected.Kind != "validation-candidate" {
		return Candidate{}, fmt.Errorf("candidate kind %q is not validation-candidate", selected.Kind)
	}
	if !strings.Contains(selected.Subject, ".") {
		return Candidate{}, fmt.Errorf("candidate subject %q is not a namespaced attribute", selected.Subject)
	}
	expression, _ := selected.Payload["expression"].(string)
	if !strings.Contains(expression, ">=") || !strings.Contains(expression, "0") {
		return Candidate{}, fmt.Errorf("candidate expression %q is not a supported positive validation", expression)
	}
	namespace := strings.SplitN(selected.Subject, ".", 2)[0]
	base, err := workspace.Load(workspaceDir)
	if err != nil {
		return Candidate{}, fmt.Errorf("load Workspace: %w", err)
	}
	index, err := workspace.BuildSourceIndex(workspaceDir, base)
	if err != nil {
		return Candidate{}, fmt.Errorf("build Source Index: %w", err)
	}
	actorValue := strings.TrimSpace(actor)
	if actorValue == "" {
		actorValue = strings.TrimSpace(providerID + "/" + model + "@" + version)
	}
	candidate := Candidate{
		Schema: CandidateSchema, ID: namespace + "." + canonicalSegment(selected.ID),
		BaseModelRevision: sourceindex.ModelRevision(index),
		Origin: Origin{
			Kind: "ai", Actor: actorValue,
			Notes: map[string]string{
				"proposalId": selected.ID, "basis": strings.Join(selected.Basis, ","),
			},
		},
		Intent: "add positive validation for " + selected.Subject + " from cognition proposal " + selected.ID,
		Delta: []Operation{{
			Op: "add-validation", Namespace: namespace,
			ID: namespace + "." + canonicalSegment(selected.ID), Kind: "positive", Attribute: selected.Subject,
		}},
	}
	if binding != nil {
		candidate.Binding = binding
	}
	return candidate, nil
}

func Write(candidate Candidate, path string) error {
	raw, err := json.MarshalIndent(candidate, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}
