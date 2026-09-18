package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

const supportedFixtureSchema = "sei.decision-continuity-comparison-fixture/v0"

type fixture struct {
	Schema              string                     `json:"schema"`
	Status              string                     `json:"status"`
	Issue               string                     `json:"issue"`
	FixtureID           string                     `json:"fixture_id"`
	Title               string                     `json:"title"`
	Purpose             []string                   `json:"purpose"`
	Scenario            scenario                   `json:"scenario"`
	Conditions          map[string]json.RawMessage `json:"conditions"`
	ReferenceState      referenceState             `json:"reference_state"`
	HumanReviewTemplate json.RawMessage            `json:"human_review_template"`
}

type scenario struct {
	InitialQuestion      string `json:"initial_question"`
	Phase1               []item `json:"phase_1"`
	InterruptionPoint    string `json:"interruption_point"`
	PostDecisionEvidence []item `json:"post_decision_evidence"`
}

type item struct {
	Role               string   `json:"role"`
	ID                 string   `json:"id"`
	Text               string   `json:"text"`
	BasisRefs          []string `json:"basis_refs,omitempty"`
	RecommendationRefs []string `json:"recommendation_refs,omitempty"`
	UnresolvedRefs     []string `json:"unresolved_refs,omitempty"`
}

type referenceState struct {
	Decision           string   `json:"decision"`
	Recommendation     string   `json:"recommendation"`
	HistoricalBasisIDs []string `json:"historical_basis_ids"`
	UnresolvedIDs      []string `json:"unresolved_ids"`
	PostDecisionIDs    []string `json:"post_decision_ids"`
	TemporalStopline   string   `json:"temporal_stopline"`
}

type resumeCondition struct {
	SurfaceText resumeOracle `json:"surface_text"`
}

type resumeOracle struct {
	Question             string        `json:"question"`
	Recommendation       oracleItem    `json:"recommendation"`
	Decision             oracleItem    `json:"decision"`
	HistoricalBasis      []string      `json:"historical_basis"`
	Unresolved           []string      `json:"unresolved"`
	ChangesSinceDecision []interface{} `json:"changes_since_decision"`
}

type oracleItem struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type projectedItem struct {
	Role string `json:"role"`
	ID   string `json:"id"`
	Text string `json:"text"`
}

type resumeProjection struct {
	CurrentQuestion      string          `json:"current_question"`
	Recommendations      []projectedItem `json:"recommendations"`
	Decision             projectedItem   `json:"decision"`
	HistoricalBasis      []projectedItem `json:"historical_basis"`
	Unresolved           []projectedItem `json:"unresolved"`
	ChangesSinceDecision []projectedItem `json:"changes_since_decision"`
}

type semanticAudit struct {
	RecommendationSeparated    bool `json:"recommendation_separated"`
	DecisionMatchesOracle       bool `json:"decision_matches_oracle"`
	RecommendationMatchesOracle bool `json:"recommendation_matches_oracle"`
	HistoricalBasisMatches      bool `json:"historical_basis_matches"`
	UnresolvedMatches           bool `json:"unresolved_matches"`
	PostDecisionMatches         bool `json:"post_decision_matches"`
	TemporalSeparation          bool `json:"temporal_separation"`
}

type report struct {
	FixtureID     string           `json:"fixture_id"`
	InitialResume resumeProjection `json:"initial_resume"`
	PostDecision  resumeProjection `json:"post_decision_review"`
	SemanticAudit semanticAudit    `json:"semantic_audit"`
}

func main() {
	fixturePath := flag.String(
		"fixture",
		"reference/sei-adoption/decision-continuity-counterfactual-001.json",
		"SEI Decision Continuity Reference fixture",
	)
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "positional arguments are not supported")
		os.Exit(1)
	}

	result, err := run(*fixturePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(path string) (report, error) {
	f, err := readFixture(path)
	if err != nil {
		return report{}, err
	}
	return projectFixture(f)
}

func readFixture(path string) (fixture, error) {
	file, err := os.Open(path)
	if err != nil {
		return fixture{}, fmt.Errorf("SEI Reference fixture %sを読み込めません: %w", path, err)
	}
	defer file.Close()

	f, err := decodeFixture(file)
	if err != nil {
		return fixture{}, fmt.Errorf("SEI Reference fixture %sが不正です: %w", path, err)
	}
	return f, nil
}

func decodeFixture(r io.Reader) (fixture, error) {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()

	var f fixture
	if err := decoder.Decode(&f); err != nil {
		return fixture{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return fixture{}, errors.New("JSON documentの後ろに追加valueがあります")
		}
		return fixture{}, err
	}

	if f.Schema != supportedFixtureSchema {
		return fixture{}, fmt.Errorf("unsupported fixture schema %q", f.Schema)
	}
	if f.Status != "evaluation-only" {
		return fixture{}, fmt.Errorf("fixture status %q is not evaluation-only", f.Status)
	}
	if strings.TrimSpace(f.FixtureID) == "" {
		return fixture{}, errors.New("fixture_id is required")
	}
	return f, nil
}

func projectFixture(f fixture) (report, error) {
	phaseIndex, err := indexItems(f.Scenario.Phase1, "phase_1")
	if err != nil {
		return report{}, err
	}
	postIndex, err := indexItems(f.Scenario.PostDecisionEvidence, "post_decision_evidence")
	if err != nil {
		return report{}, err
	}
	for id := range postIndex {
		if _, exists := phaseIndex[id]; exists {
			return report{}, fmt.Errorf("item id %q exists in both phase_1 and post_decision_evidence", id)
		}
	}

	decision, err := uniqueDecision(f.Scenario.Phase1)
	if err != nil {
		return report{}, err
	}
	recommendations, err := resolveRefs(decision.RecommendationRefs, phaseIndex, map[string]bool{"recommendation": true}, "recommendation_refs")
	if err != nil {
		return report{}, err
	}
	basis, err := resolveRefs(decision.BasisRefs, phaseIndex, map[string]bool{"observation": true}, "basis_refs")
	if err != nil {
		return report{}, err
	}
	unresolved, err := resolveRefs(decision.UnresolvedRefs, phaseIndex, map[string]bool{"unknown": true, "conflict": true}, "unresolved_refs")
	if err != nil {
		return report{}, err
	}

	initial := resumeProjection{
		CurrentQuestion: strings.TrimSpace(f.Scenario.InitialQuestion),
		Recommendations: toProjected(recommendations),
		Decision:        toProjectedItem(decision),
		HistoricalBasis: toProjected(basis),
		Unresolved:      toProjected(unresolved),
	}
	post := initial
	post.ChangesSinceDecision = toProjected(f.Scenario.PostDecisionEvidence)

	audit, err := auditProjection(f, initial, post)
	if err != nil {
		return report{}, err
	}

	return report{
		FixtureID:     f.FixtureID,
		InitialResume: initial,
		PostDecision:  post,
		SemanticAudit: audit,
	}, nil
}

func indexItems(items []item, label string) (map[string]item, error) {
	result := make(map[string]item, len(items))
	for _, value := range items {
		value.ID = strings.TrimSpace(value.ID)
		value.Role = strings.TrimSpace(value.Role)
		value.Text = strings.TrimSpace(value.Text)
		if value.ID == "" {
			return nil, fmt.Errorf("%s contains item without id", label)
		}
		if value.Text == "" {
			return nil, fmt.Errorf("%s item %q has empty text", label, value.ID)
		}
		if !knownRole(value.Role) {
			return nil, fmt.Errorf("%s item %q has unsupported role %q", label, value.ID, value.Role)
		}
		if _, exists := result[value.ID]; exists {
			return nil, fmt.Errorf("%s contains duplicate id %q", label, value.ID)
		}
		result[value.ID] = value
	}
	return result, nil
}

func knownRole(role string) bool {
	switch role {
	case "observation", "recommendation", "unknown", "conflict", "decision":
		return true
	default:
		return false
	}
}

func uniqueDecision(items []item) (item, error) {
	var decisions []item
	for _, value := range items {
		if value.Role == "decision" {
			decisions = append(decisions, value)
		}
	}
	if len(decisions) != 1 {
		return item{}, fmt.Errorf("phase_1 must contain exactly one decision; got %d", len(decisions))
	}
	return decisions[0], nil
}

func resolveRefs(refs []string, index map[string]item, allowedRoles map[string]bool, label string) ([]item, error) {
	result := make([]item, 0, len(refs))
	for _, ref := range refs {
		value, ok := index[ref]
		if !ok {
			return nil, fmt.Errorf("%s reference %q does not resolve within phase_1", label, ref)
		}
		if allowedRoles != nil && !allowedRoles[value.Role] {
			return nil, fmt.Errorf("%s reference %q resolves role %q", label, ref, value.Role)
		}
		result = append(result, value)
	}
	return result, nil
}

func toProjected(values []item) []projectedItem {
	result := make([]projectedItem, 0, len(values))
	for _, value := range values {
		result = append(result, toProjectedItem(value))
	}
	return result
}

func toProjectedItem(value item) projectedItem {
	return projectedItem{Role: value.Role, ID: value.ID, Text: value.Text}
}

func auditProjection(f fixture, initial, post resumeProjection) (semanticAudit, error) {
	raw, ok := f.Conditions["C_sei_resume"]
	if !ok {
		return semanticAudit{}, errors.New("conditions.C_sei_resume is required as Projection oracle")
	}
	var condition resumeCondition
	if err := json.Unmarshal(raw, &condition); err != nil {
		return semanticAudit{}, fmt.Errorf("conditions.C_sei_resume is invalid: %w", err)
	}

	audit := semanticAudit{
		RecommendationSeparated:    len(initial.Recommendations) > 0 && allRecommendationIDsDifferFromDecision(initial.Recommendations, initial.Decision.ID),
		DecisionMatchesOracle:       initial.Decision.ID == condition.SurfaceText.Decision.ID,
		RecommendationMatchesOracle: containsProjectedID(initial.Recommendations, condition.SurfaceText.Recommendation.ID),
		HistoricalBasisMatches:      equalStringSlices(ids(initial.HistoricalBasis), f.ReferenceState.HistoricalBasisIDs),
		UnresolvedMatches:           equalStringSlices(ids(initial.Unresolved), f.ReferenceState.UnresolvedIDs),
		PostDecisionMatches:         equalStringSlices(ids(post.ChangesSinceDecision), f.ReferenceState.PostDecisionIDs),
		TemporalSeparation: noOverlap(ids(initial.HistoricalBasis), ids(post.ChangesSinceDecision)) &&
			len(initial.ChangesSinceDecision) == 0 &&
			len(condition.SurfaceText.ChangesSinceDecision) == 0,
	}

	if !audit.RecommendationSeparated {
		return semanticAudit{}, errors.New("Recommendation and Decision are not kept separate")
	}
	if !audit.DecisionMatchesOracle {
		return semanticAudit{}, errors.New("derived Decision does not match C_sei_resume oracle")
	}
	if !audit.RecommendationMatchesOracle {
		return semanticAudit{}, errors.New("derived Recommendation does not match C_sei_resume oracle")
	}
	if !audit.HistoricalBasisMatches {
		return semanticAudit{}, errors.New("derived historical basis ids do not match reference_state")
	}
	if !audit.UnresolvedMatches {
		return semanticAudit{}, errors.New("derived unresolved ids do not match reference_state")
	}
	if !audit.PostDecisionMatches {
		return semanticAudit{}, errors.New("derived post-decision ids do not match reference_state")
	}
	if !audit.TemporalSeparation {
		return semanticAudit{}, errors.New("post-decision evidence leaked into historical basis")
	}
	return audit, nil
}

func ids(values []projectedItem) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.ID)
	}
	return result
}

func containsProjectedID(values []projectedItem, target string) bool {
	for _, value := range values {
		if value.ID == target {
			return true
		}
	}
	return false
}

func allRecommendationIDsDifferFromDecision(values []projectedItem, decisionID string) bool {
	for _, value := range values {
		if value.ID == decisionID {
			return false
		}
	}
	return true
}

func equalStringSlices(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func noOverlap(left, right []string) bool {
	seen := make(map[string]struct{}, len(left))
	for _, value := range left {
		seen[value] = struct{}{}
	}
	for _, value := range right {
		if _, ok := seen[value]; ok {
			return false
		}
	}
	return true
}
