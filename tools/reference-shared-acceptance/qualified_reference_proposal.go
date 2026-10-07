package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"tei/internal/application"
	"tei/internal/knowledgecontext"
)

const qualifiedReferenceReuseSchema = "tei.reference.qualified-reference-reuse/v0"

// QualifiedReferenceReuseは、qualifiedな既存reference Patternから共有query選択だけを再利用し、
// destination-ownedなPattern identity / owner / Relation / datasource / placementを明示する。
// Contract / Query / input mappingはcaller入力にせず、current Canonicalのqualified originから取得する。
type QualifiedReferenceReuse struct {
	Schema      string `json:"schema"`
	PatternID   string `json:"patternId"`
	Application string `json:"application"`
	Relation    string `json:"relation"`
	Datasource  string `json:"datasource"`
	Placement   string `json:"placement"`
}

func decodeQualifiedReferenceReuse(data []byte) (QualifiedReferenceReuse, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var request QualifiedReferenceReuse
	if err := decoder.Decode(&request); err != nil {
		return request, fmt.Errorf("qualified reference reuse requestをJSONとして読めません: %w", err)
	}
	if _, err := decoder.Token(); err != io.EOF {
		return request, fmt.Errorf("qualified reference reuse requestのJSONの後ろに余分な内容があります")
	}
	request.Schema = strings.TrimSpace(request.Schema)
	request.PatternID = strings.TrimSpace(request.PatternID)
	request.Application = strings.TrimSpace(request.Application)
	request.Relation = strings.TrimSpace(request.Relation)
	request.Datasource = strings.TrimSpace(request.Datasource)
	request.Placement = strings.TrimSpace(request.Placement)
	if request.Schema != qualifiedReferenceReuseSchema {
		return request, fmt.Errorf("qualified reference reuse request schema %q は未対応です", request.Schema)
	}
	if request.PatternID == "" ||
		request.Application == "" ||
		request.Relation == "" ||
		request.Datasource == "" ||
		request.Placement == "" {
		return request, fmt.Errorf("qualified reference reuse requestにはpatternId / application / relation / datasource / placementが必要です")
	}
	return request, nil
}

// proposeQualifiedReferencePatternは、探索・qualification済みPatternを暗黙copyせず、
// current qualificationを再確認したうえで共有Contract Query選択だけをdestination Patternへ写す。
// qualification Evidenceはcandidateへ添付し、Shared Acceptance Previewが再度current Sourceへ照合できる形を保つ。
func proposeQualifiedReferencePattern(
	root string,
	candidateData []byte,
	qualificationData []byte,
	requestData []byte,
) (Candidate, error) {
	candidate, err := decodeCandidate(candidateData)
	if err != nil {
		return Candidate{}, err
	}
	qualification, err := decodeQualificationResult(qualificationData)
	if err != nil {
		return Candidate{}, err
	}
	request, err := decodeQualifiedReferenceReuse(requestData)
	if err != nil {
		return Candidate{}, err
	}
	if qualification.Status != knowledgecontext.QualificationApplicable {
		return Candidate{}, fmt.Errorf("qualification status %q はapplicableではありません", qualification.Status)
	}
	if qualification.Candidate.Kind != "application-pattern" || strings.TrimSpace(qualification.Candidate.ID) == "" {
		return Candidate{}, fmt.Errorf(
			"qualified reference reuseにはapplication-pattern candidateが必要です: kind=%q id=%q",
			qualification.Candidate.Kind,
			qualification.Candidate.ID,
		)
	}
	requirements, err := knowledgecontext.NormalizeQualificationRequirements(qualification.Requirements)
	if err != nil {
		return Candidate{}, err
	}
	if requirements.ExpectedKind != "application-pattern" {
		return Candidate{}, fmt.Errorf(
			"qualified reference reuseにはexpectedKind=application-patternが必要です: %q",
			requirements.ExpectedKind,
		)
	}

	snapshot, err := takeSnapshot(root)
	if err != nil {
		return Candidate{}, err
	}
	currentContext := knowledgecontext.Build(snapshot.loaded.World, snapshot.index, qualification.Candidate)
	current, err := knowledgecontext.QualifyWithSources(currentContext, snapshot.index, requirements)
	if err != nil {
		return Candidate{}, err
	}
	if current.Status != knowledgecontext.QualificationApplicable {
		return Candidate{}, fmt.Errorf(
			"qualificationはcurrent workspaceではapplicableではありません: status=%s reasons=%v",
			current.Status,
			current.Reasons,
		)
	}
	if !sameQualificationForProposal(qualification, current) {
		return Candidate{}, fmt.Errorf("qualificationはcurrent workspaceのexact Evidenceと一致しません。knowledge-qualifyを再実行してください")
	}

	origin, owner, err := findReferencePattern(snapshot.loaded.World.Applications, qualification.Candidate.ID)
	if err != nil {
		return Candidate{}, err
	}
	if requirements.Application != "" && owner != requirements.Application {
		return Candidate{}, fmt.Errorf(
			"qualification application scope %q とorigin Pattern owner %q が一致しません",
			requirements.Application,
			owner,
		)
	}
	if origin.ContractQuery == nil {
		return Candidate{}, fmt.Errorf("qualified origin Pattern %s にContract Queryがありません", origin.ID)
	}
	requiredContractQuery := string(origin.ContractQuery.Contract) + "/" + string(origin.ContractQuery.Query)
	if !requiresQualifiedContractQuery(requirements, requiredContractQuery) {
		return Candidate{}, fmt.Errorf(
			"qualified origin Pattern %s のContract Query %s がrequiredDependenciesで明示されていません",
			origin.ID,
			requiredContractQuery,
		)
	}

	query := ContractQuerySpec{
		Contract: string(origin.ContractQuery.Contract),
		Query:    string(origin.ContractQuery.Query),
	}
	for _, input := range origin.ContractQuery.Inputs {
		query.Inputs = append(query.Inputs, QueryInputSpec{
			Attribute: string(input.TargetAttribute),
			Input:     string(input.Input),
		})
	}
	candidate.Delta = append(candidate.Delta, Operation{
		Op:        opAddReferencePattern,
		Placement: request.Placement,
		Pattern: &PatternSpec{
			ID:            request.PatternID,
			Application:   request.Application,
			Relation:      request.Relation,
			Datasource:    request.Datasource,
			ContractQuery: query,
		},
	})
	normalized, _, err := normalizeCandidate(candidate)
	if err != nil {
		return Candidate{}, err
	}
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return Candidate{}, err
	}
	return attachQualificationEvidence(encoded, current)
}

func sameQualificationForProposal(recorded, current knowledgecontext.QualificationResult) bool {
	if recorded.Status != current.Status || recorded.Candidate != current.Candidate {
		return false
	}
	recordedRequirements, err := knowledgecontext.NormalizeQualificationRequirements(recorded.Requirements)
	if err != nil {
		return false
	}
	currentRequirements, err := knowledgecontext.NormalizeQualificationRequirements(current.Requirements)
	if err != nil {
		return false
	}
	recordedJSON, _ := json.Marshal(recordedRequirements)
	currentJSON, _ := json.Marshal(currentRequirements)
	if !bytes.Equal(recordedJSON, currentJSON) {
		return false
	}
	return sameQualificationEvidence(recorded.Evidence, current.Evidence) &&
		sameQualificationEvidence(recorded.DependencyEvidence, current.DependencyEvidence)
}

func requiresQualifiedContractQuery(requirements knowledgecontext.QualificationRequirements, id string) bool {
	for _, dependency := range requirements.RequiredDependencies {
		if dependency.Direction == "outgoing" &&
			dependency.Relation == "contract-query" &&
			dependency.Kind == "database-contract-query" &&
			dependency.ID == id {
			return true
		}
	}
	return false
}

func findReferencePattern(applications []application.Application, id string) (application.Pattern, string, error) {
	var found application.Pattern
	var owner string
	count := 0
	for _, app := range applications {
		for _, pattern := range app.Patterns {
			if string(pattern.ID) != id {
				continue
			}
			count++
			found = pattern
			owner = string(app.ID)
		}
	}
	if count == 0 {
		return application.Pattern{}, "", fmt.Errorf("qualified origin Pattern %s がcurrent Canonical Worldにありません", id)
	}
	if count != 1 {
		return application.Pattern{}, "", fmt.Errorf("qualified origin Pattern %s がcurrent Canonical Worldで一意ではありません", id)
	}
	if found.Kind != application.PatternReference {
		return application.Pattern{}, "", fmt.Errorf(
			"qualified origin Pattern %s はreference Patternではありません: kind=%s",
			id,
			found.Kind,
		)
	}
	return found, owner, nil
}
