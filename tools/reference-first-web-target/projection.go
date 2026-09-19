package main

import (
	"fmt"
	"sort"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/ir"
)

type projectionConfig struct {
	HandoverBaselineID         string
	BaselineRepositoryRevision string
	TargetBuildRevision        string
	ApplicationID              core.CanonicalID
}

type webTargetProjection struct {
	HandoverBaselineID         string                 `json:"handoverBaselineId"`
	BaselineRepositoryRevision string                 `json:"baselineRepositoryRevision"`
	TargetBuildRevision        string                 `json:"targetBuildRevision"`
	TargetAdapter              string                 `json:"targetAdapter"`
	Host                       string                 `json:"host"`
	Stage                      string                 `json:"stage"`
	ApplicationID              string                 `json:"applicationId"`
	SemanticUses               []string               `json:"semanticUses,omitempty"`
	Validations                []string               `json:"validations,omitempty"`
	Patterns                   []webPatternProjection `json:"patterns"`
	Unresolved                 []string               `json:"unresolved"`
	AuthorityLimits            []string               `json:"authorityLimits"`
}

type webPatternProjection struct {
	ID                    string                      `json:"id"`
	Kind                  string                      `json:"kind"`
	PresentationCandidate string                      `json:"presentationCandidate"`
	Data                  string                      `json:"data,omitempty"`
	Relation              *webRelationProjection      `json:"relation,omitempty"`
	Query                 string                      `json:"query,omitempty"`
	Datasource            *webDatasourceProjection    `json:"datasource,omitempty"`
	Capability            string                      `json:"capability,omitempty"`
	ContractQuery         *webContractQueryProjection `json:"contractQuery,omitempty"`
}

type webRelationProjection struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
}

type webDatasourceProjection struct {
	Alias  string `json:"alias"`
	Target string `json:"target"`
	Access string `json:"access"`
}

type webContractQueryProjection struct {
	Contract          string                         `json:"contract"`
	ContractVersion   string                         `json:"contractVersion"`
	Query             string                         `json:"query"`
	Inputs            []webContractQueryInputMapping `json:"inputs,omitempty"`
	Result            string                         `json:"result,omitempty"`
	ResultCardinality string                         `json:"resultCardinality,omitempty"`
}

type webContractQueryInputMapping struct {
	SemanticAttribute string `json:"semanticAttribute"`
	QueryInput        string `json:"queryInput"`
	QueryInputType    string `json:"queryInputType,omitempty"`
}

func buildWebTargetProjection(world ir.World, cfg projectionConfig) (webTargetProjection, error) {
	if cfg.HandoverBaselineID == "" {
		return webTargetProjection{}, fmt.Errorf("handover baseline ID is required")
	}
	if cfg.BaselineRepositoryRevision == "" {
		return webTargetProjection{}, fmt.Errorf("baseline repository revision is required")
	}
	if cfg.TargetBuildRevision == "" {
		return webTargetProjection{}, fmt.Errorf("target build revision is required")
	}
	if cfg.ApplicationID == "" {
		return webTargetProjection{}, fmt.Errorf("application ID is required")
	}

	app, err := projectionApplicationByID(world, cfg.ApplicationID)
	if err != nil {
		return webTargetProjection{}, err
	}
	resolver := driver.NewResolver(world)

	result := webTargetProjection{
		HandoverBaselineID:         cfg.HandoverBaselineID,
		BaselineRepositoryRevision: cfg.BaselineRepositoryRevision,
		TargetBuildRevision:        cfg.TargetBuildRevision,
		TargetAdapter:              "reference-first-web-target",
		Host:                       "go-net-http",
		Stage:                      "standard-and-custom-ui-shared-boundary",
		ApplicationID:              string(app.ID),
		SemanticUses:               canonicalIDsToStrings(app.SemanticUses),
		Validations:                canonicalIDsToStrings(app.Validations),
		Unresolved: []string{
			"browse-environment-instance-binding-not-exposed-by-current-driver-api",
			"actual-functional-evidence-not-yet-acquired",
			"visual-evidence-not-yet-acquired",
			"actual-external-command-observation-not-acquired",
		},
		AuthorityLimits: []string{
			"projection-is-derived-not-canonical",
			"presentation-candidate-is-target-local",
			"web-target-does-not-author-validation-or-authorization",
			"business-result-execution-error-observation-remain-distinct",
			"target-build-revision-is-recorded-by-evidence-not-baseline",
			"custom-ui-shares-command-scenario-boundary",
		},
	}

	for _, pattern := range app.Patterns {
		projected, err := projectPattern(resolver, app.ID, pattern)
		if err != nil {
			return webTargetProjection{}, err
		}
		result.Patterns = append(result.Patterns, projected)
	}
	sort.Slice(result.Patterns, func(i, j int) bool {
		return result.Patterns[i].ID < result.Patterns[j].ID
	})
	return result, nil
}

func projectPattern(resolver driver.Resolver, applicationID core.CanonicalID, pattern application.Pattern) (webPatternProjection, error) {
	resolved, err := resolver.ResolvePattern(applicationID, pattern.ID)
	if err != nil {
		return webPatternProjection{}, fmt.Errorf("resolve Pattern %s: %w", pattern.ID, err)
	}

	result := webPatternProjection{
		ID:                    string(resolved.Pattern.ID),
		Kind:                  string(resolved.Pattern.Kind),
		PresentationCandidate: presentationCandidateFor(resolved.Pattern.Kind),
	}
	if resolved.Data != nil {
		result.Data = string(resolved.Data.ID)
	}
	if resolved.Relation != nil {
		result.Relation = &webRelationProjection{
			ID:     string(resolved.Relation.Relation.ID),
			Source: string(resolved.Relation.Source.ID),
			Target: string(resolved.Relation.Target.ID),
		}
	}
	if resolved.Query != nil {
		result.Query = string(resolved.Query.ID)
	}
	if resolved.Datasource != nil {
		result.Datasource = projectDatasource(*resolved.Datasource)
	}
	if resolved.Capability != nil {
		result.Capability = string(resolved.Capability.ID)
	}

	switch resolved.Pattern.Kind {
	case application.PatternReference:
		exact, err := resolver.ResolveReferenceContractQuery(applicationID, pattern.ID)
		if err != nil {
			return webPatternProjection{}, fmt.Errorf("resolve reference Contract Query for %s: %w", pattern.ID, err)
		}
		result.ContractQuery = projectReferenceContractQuery(exact)
	case application.PatternHeaderDetail:
		exact, err := resolver.ResolveHeaderDetailContractQuery(applicationID, pattern.ID)
		if err != nil {
			return webPatternProjection{}, fmt.Errorf("resolve header-detail Contract Query for %s: %w", pattern.ID, err)
		}
		result.Datasource = projectDatasource(exact.Datasource)
		result.ContractQuery = projectHeaderDetailContractQuery(exact)
	case application.PatternCommand:
		if _, err := resolver.ResolveCommandBinding(applicationID, pattern.ID); err != nil {
			return webPatternProjection{}, fmt.Errorf("resolve command Binding for %s: %w", pattern.ID, err)
		}
	}
	return result, nil
}

func projectDatasource(source application.DatabaseRequirement) *webDatasourceProjection {
	return &webDatasourceProjection{
		Alias:  string(source.Alias),
		Target: string(source.TargetDatabase),
		Access: string(source.Access),
	}
}

func projectReferenceContractQuery(source driver.ResolvedReferenceContractQuery) *webContractQueryProjection {
	inputTypes := make(map[string]string, len(source.Query.Inputs))
	for _, input := range source.Query.Inputs {
		inputTypes[string(input.Name)] = string(input.Type)
	}
	inputs := make([]webContractQueryInputMapping, 0, len(source.Invocation.Inputs))
	for _, binding := range source.Invocation.Inputs {
		inputs = append(inputs, webContractQueryInputMapping{
			SemanticAttribute: string(binding.TargetAttribute),
			QueryInput:        string(binding.Input),
			QueryInputType:    inputTypes[string(binding.Input)],
		})
	}
	return &webContractQueryProjection{
		Contract:          string(source.Contract.ID),
		ContractVersion:   fmt.Sprintf("%d.%d", source.Contract.Version.Major, source.Contract.Version.Minor),
		Query:             source.Query.Name,
		Inputs:            inputs,
		Result:            string(source.Query.Result),
		ResultCardinality: string(source.Query.ResultCardinality),
	}
}

func projectHeaderDetailContractQuery(source driver.ResolvedHeaderDetailContractQuery) *webContractQueryProjection {
	inputTypes := make(map[string]string, len(source.Query.Inputs))
	for _, input := range source.Query.Inputs {
		inputTypes[string(input.Name)] = string(input.Type)
	}
	inputs := make([]webContractQueryInputMapping, 0, len(source.Binding.Inputs))
	for _, binding := range source.Binding.Inputs {
		inputs = append(inputs, webContractQueryInputMapping{
			SemanticAttribute: string(binding.SourceAttribute),
			QueryInput:        string(binding.Input),
			QueryInputType:    inputTypes[string(binding.Input)],
		})
	}
	return &webContractQueryProjection{
		Contract:          string(source.Contract.ID),
		ContractVersion:   fmt.Sprintf("%d.%d", source.Contract.Version.Major, source.Contract.Version.Minor),
		Query:             source.Query.Name,
		Inputs:            inputs,
		Result:            string(source.Query.Result),
		ResultCardinality: string(source.Query.ResultCardinality),
	}
}

func presentationCandidateFor(kind application.PatternKind) string {
	switch kind {
	case application.PatternBrowse:
		return "collection"
	case application.PatternRecord:
		return "item"
	case application.PatternReference:
		return "relation-selection"
	case application.PatternHeaderDetail:
		return "detail-collection"
	case application.PatternCommand:
		return "command-invocation"
	default:
		return "unresolved"
	}
}

func projectionApplicationByID(world ir.World, id core.CanonicalID) (application.Application, error) {
	var found application.Application
	count := 0
	for _, candidate := range world.Applications {
		if candidate.ID != id {
			continue
		}
		found = candidate
		count++
	}
	if count == 0 {
		return application.Application{}, fmt.Errorf("Application %s not found", id)
	}
	if count > 1 {
		return application.Application{}, fmt.Errorf("Application %s is ambiguous: %d definitions", id, count)
	}
	return found, nil
}

func canonicalIDsToStrings(ids []core.CanonicalID) []string {
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		result = append(result, string(id))
	}
	sort.Strings(result)
	return result
}
