package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"tei/internal/application"
	"tei/internal/database"
	"tei/internal/ir"
	"tei/internal/workspace"
)

const (
	fixtureSchemaVersion       = "business-procedure-reference-v0"
	probeSchemaVersion         = "business-procedure-access-probe-v0"
	fixtureArtifactRole        = "reference-authoring-fixture"
	severitySemanticError      = "semantic-error"
	severityArchitecturePolicy = "architecture-policy"
)

type fixture struct {
	SchemaVersion string    `json:"schemaVersion"`
	ArtifactRole  string    `json:"artifactRole"`
	Canonical     bool      `json:"canonical"`
	Application   string    `json:"application,omitempty"`
	Procedure     procedure `json:"procedure"`
	Steps         []step    `json:"steps"`
}

type procedure struct {
	ID             string       `json:"id"`
	IdentityStatus string       `json:"identityStatus"`
	Purpose        string       `json:"purpose"`
	Trigger        candidateRef `json:"trigger"`
	Actor          candidateRef `json:"actor"`
}

type candidateRef struct {
	Status string `json:"status"`
	Text   string `json:"text,omitempty"`
	Label  string `json:"label,omitempty"`
}

type step struct {
	ID                  string   `json:"id"`
	BusinessAction      string   `json:"businessAction"`
	Realization         string   `json:"realization"`
	ApplicationPatterns []string `json:"applicationPatterns,omitempty"`
	Data                []string `json:"data,omitempty"`
	Relations           []string `json:"relations,omitempty"`
	Validations         []string `json:"validations,omitempty"`
	Capabilities        []string `json:"capabilities,omitempty"`
}

type accessIntent struct {
	Required database.Access `json:"required"`
}

type accessProbe struct {
	SchemaVersion      string       `json:"schemaVersion"`
	Procedure          string       `json:"procedure"`
	Step               string       `json:"step"`
	ApplicationPattern string       `json:"applicationPattern"`
	RequestedAccess    accessIntent `json:"requestedAccess"`
	Reason             string       `json:"reason"`
}

type diagnostic struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Step     string `json:"step,omitempty"`
	Message  string `json:"message"`
}

type functionRow struct {
	StepID                string   `json:"step"`
	BusinessAction        string   `json:"businessAction"`
	ApplicationPatterns   []string `json:"applicationPatterns,omitempty"`
	NoApplicationFunction bool     `json:"noApplicationFunction"`
}

type dataRow struct {
	StepID       string   `json:"step"`
	Data         []string `json:"data,omitempty"`
	Relations    []string `json:"relations,omitempty"`
	Validations  []string `json:"validations,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
}

type patternMeaningRow struct {
	StepID     string `json:"step"`
	Pattern    string `json:"pattern"`
	Data       string `json:"data,omitempty"`
	Relation   string `json:"relation,omitempty"`
	Query      string `json:"query,omitempty"`
	Capability string `json:"capability,omitempty"`
	Datasource string `json:"datasource,omitempty"`
}

type report struct {
	Procedure                        string                     `json:"procedure"`
	Application                      string                     `json:"application,omitempty"`
	CanonicalProcedureIdentityUsed   bool                       `json:"canonicalProcedureIdentityUsed"`
	BusinessActionFunctionRows       []functionRow              `json:"businessActionFunctionRows"`
	BusinessActionPatternMeaningRows []patternMeaningRow        `json:"businessActionPatternMeaningRows"`
	ExplicitBusinessMeaningRows      []dataRow                  `json:"explicitBusinessMeaningRows"`
	Diagnostics                      []diagnostic               `json:"diagnostics"`
	ValidationScopeReview            []validationScopeReviewRow `json:"validationScopeReview,omitempty"`
}

func main() {
	workspacePath := flag.String("workspace", "reference/workspace", "Reference Workspace directory")
	fixturePath := flag.String("fixture", "reference/business-system-design/order-entry-business-procedure.json", "Business Procedure Reference fixture")
	probePath := flag.String("probe", "", "optional experiment access probe")
	expectDiagnostic := flag.String("expect-diagnostic", "", "expected blocking diagnostic code for an intentional defect probe")
	reviewValidationScope := flag.Bool("review-validation-scope", false, "Application全体のValidation宣言とBusiness Procedureの明示参照を読み取り専用で比較する")
	flag.Parse()

	result, err := runWithOptions(*workspacePath, *fixturePath, *probePath, *reviewValidationScope)
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

	if err := checkExpectedDiagnostics(result.Diagnostics, *expectDiagnostic); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(workspacePath, fixturePath, probePath string) (report, error) {
	return runWithOptions(workspacePath, fixturePath, probePath, false)
}

func runWithOptions(workspacePath, fixturePath, probePath string, reviewValidationScope bool) (report, error) {
	f, err := readFixture(fixturePath)
	if err != nil {
		return report{}, err
	}
	loaded, err := workspace.Load(workspacePath)
	if err != nil {
		return report{}, fmt.Errorf("Reference Workspaceを読み込めません: %w", err)
	}

	result := evaluate(loaded.World, f)
	if reviewValidationScope {
		result.ValidationScopeReview = compareValidationScopes(loaded.World, f)
		for i := range result.ValidationScopeReview {
			result.ValidationScopeReview[i].ProcedureSource = fixturePath
		}
		if validationScopeSourcesNeeded(result.ValidationScopeReview) {
			sources, err := workspace.BuildSourceIndex(workspacePath, loaded)
			if err != nil {
				return report{}, fmt.Errorf("Validation比較の根拠Sourceを確認できません: %w", err)
			}
			attachValidationScopeSources(result.ValidationScopeReview, sources)
		}
	}
	if strings.TrimSpace(probePath) != "" {
		probe, err := readProbe(probePath)
		if err != nil {
			return report{}, err
		}
		applyAccessProbe(&result, loaded.World, f, probe)
	}
	return result, nil
}

func readFixture(path string) (fixture, error) {
	file, err := os.Open(path)
	if err != nil {
		return fixture{}, fmt.Errorf("Business Procedure fixture %sを読み込めません: %w", path, err)
	}
	defer file.Close()
	f, err := decodeFixture(file)
	if err != nil {
		return fixture{}, fmt.Errorf("Business Procedure fixture %sが不正です: %w", path, err)
	}
	return f, nil
}

func decodeFixture(r io.Reader) (fixture, error) {
	var f fixture
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&f); err != nil {
		return fixture{}, err
	}
	if err := requireEOF(decoder); err != nil {
		return fixture{}, err
	}
	if f.SchemaVersion != fixtureSchemaVersion {
		return fixture{}, fmt.Errorf("unsupported schemaVersion %q; expected %q", f.SchemaVersion, fixtureSchemaVersion)
	}
	if f.ArtifactRole != fixtureArtifactRole {
		return fixture{}, fmt.Errorf("unsupported artifactRole %q; expected %q", f.ArtifactRole, fixtureArtifactRole)
	}
	return f, nil
}

func readProbe(path string) (accessProbe, error) {
	file, err := os.Open(path)
	if err != nil {
		return accessProbe{}, fmt.Errorf("Business Procedure probe %sを読み込めません: %w", path, err)
	}
	defer file.Close()
	var p accessProbe
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&p); err != nil {
		return accessProbe{}, fmt.Errorf("Business Procedure probe %sが不正です: %w", path, err)
	}
	if err := requireEOF(decoder); err != nil {
		return accessProbe{}, fmt.Errorf("Business Procedure probe %sが不正です: %w", path, err)
	}
	if p.SchemaVersion != probeSchemaVersion {
		return accessProbe{}, fmt.Errorf("Business Procedure probe %sのschemaVersion %qは未対応です; expected %q", path, p.SchemaVersion, probeSchemaVersion)
	}
	return p, nil
}

func requireEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); err == io.EOF {
		return nil
	} else if err != nil {
		return err
	}
	return fmt.Errorf("JSON documentの後ろに追加valueがあります")
}

func evaluate(world ir.World, f fixture) report {
	result := report{
		Procedure:                      f.Procedure.ID,
		Application:                    f.Application,
		CanonicalProcedureIdentityUsed: f.Canonical,
	}

	if f.Canonical {
		result.Diagnostics = append(result.Diagnostics, diagnostic{
			Code:     "BUSINESS_PROCEDURE_EXPERIMENT_BECAME_CANONICAL",
			Severity: severityArchitecturePolicy,
			Message:  "このReference実験はProcedureのCanonical化を前提にしてはいけません",
		})
	}
	if strings.TrimSpace(f.Procedure.ID) == "" {
		result.Diagnostics = append(result.Diagnostics, diagnostic{Code: "BUSINESS_PROCEDURE_ID_MISSING", Severity: severitySemanticError, Message: "experiment-local Procedure IDがありません"})
	}

	// fixture.Application is optional Reference/reporting context. It is not a
	// Business Procedure owner and is not an authority for Pattern policy.
	// Cross-view semantics must resolve from the explicit references below.
	patterns := patternCounts(world)
	data, relations, validations, capabilities := semanticCounts(world)
	stepIDs := map[string]struct{}{}

	for _, s := range f.Steps {
		if _, exists := stepIDs[s.ID]; exists {
			result.Diagnostics = append(result.Diagnostics, diagnostic{Code: "BUSINESS_PROCEDURE_STEP_DUPLICATE", Severity: severitySemanticError, Step: s.ID, Message: "Business Procedure step IDが重複しています"})
		} else {
			stepIDs[s.ID] = struct{}{}
		}

		result.BusinessActionFunctionRows = append(result.BusinessActionFunctionRows, functionRow{
			StepID:                s.ID,
			BusinessAction:        s.BusinessAction,
			ApplicationPatterns:   append([]string(nil), s.ApplicationPatterns...),
			NoApplicationFunction: len(s.ApplicationPatterns) == 0,
		})
		result.ExplicitBusinessMeaningRows = append(result.ExplicitBusinessMeaningRows, dataRow{
			StepID:       s.ID,
			Data:         append([]string(nil), s.Data...),
			Relations:    append([]string(nil), s.Relations...),
			Validations:  append([]string(nil), s.Validations...),
			Capabilities: append([]string(nil), s.Capabilities...),
		})

		for _, ref := range s.ApplicationPatterns {
			checkReference(&result, s.ID, "APPLICATION_PATTERN", ref, patterns[ref])
			if pattern, ok := findPattern(world, ref); ok {
				result.BusinessActionPatternMeaningRows = append(result.BusinessActionPatternMeaningRows, patternMeaningRow{
					StepID:     s.ID,
					Pattern:    ref,
					Data:       string(pattern.Data),
					Relation:   string(pattern.Relation),
					Query:      string(pattern.Query),
					Capability: string(pattern.Capability),
					Datasource: string(pattern.Datasource),
				})
			}
		}
		for _, ref := range s.Data {
			checkReference(&result, s.ID, "DATA", ref, data[ref])
		}
		for _, ref := range s.Relations {
			checkReference(&result, s.ID, "RELATION", ref, relations[ref])
		}
		for _, ref := range s.Validations {
			checkReference(&result, s.ID, "VALIDATION", ref, validations[ref])
		}
		for _, ref := range s.Capabilities {
			checkReference(&result, s.ID, "CAPABILITY", ref, capabilities[ref])
		}
	}

	sortDiagnostics(result.Diagnostics)
	return result
}

func findApplication(world ir.World, id string) (application.Application, bool) {
	for _, app := range world.Applications {
		if string(app.ID) == id {
			return app, true
		}
	}
	return application.Application{}, false
}

func patternCounts(world ir.World) map[string]int {
	counts := map[string]int{}
	for _, app := range world.Applications {
		for _, pattern := range app.Patterns {
			counts[string(pattern.ID)]++
		}
	}
	return counts
}

func findPattern(world ir.World, id string) (application.Pattern, bool) {
	_, pattern, ok := findPatternOwner(world, id)
	return pattern, ok
}

func findPatternOwner(world ir.World, id string) (application.Application, application.Pattern, bool) {
	var foundApp application.Application
	var foundPattern application.Pattern
	count := 0
	for _, app := range world.Applications {
		for _, pattern := range app.Patterns {
			if string(pattern.ID) == id {
				foundApp = app
				foundPattern = pattern
				count++
			}
		}
	}
	return foundApp, foundPattern, count == 1
}

func semanticCounts(world ir.World) (map[string]int, map[string]int, map[string]int, map[string]int) {
	data := map[string]int{}
	relations := map[string]int{}
	validations := map[string]int{}
	capabilities := map[string]int{}
	for _, ns := range world.Semantics {
		for _, d := range ns.Data {
			data[string(d.ID)]++
			for _, relation := range d.Relations {
				relations[string(relation.ID)]++
			}
		}
		for _, validation := range ns.Validations {
			validations[string(validation.ID)]++
		}
		for _, capability := range ns.Capabilities {
			capabilities[string(capability.ID)]++
		}
	}
	return data, relations, validations, capabilities
}

func checkReference(result *report, stepID, kind, ref string, count int) {
	if count == 1 {
		return
	}
	code := "BUSINESS_" + kind + "_REF_NOT_FOUND"
	message := fmt.Sprintf("%s reference %qがnormalized Worldでexactに1件解決できません（count=%d）", kind, ref, count)
	if count > 1 {
		code = "BUSINESS_" + kind + "_REF_AMBIGUOUS"
	}
	result.Diagnostics = append(result.Diagnostics, diagnostic{Code: code, Severity: severitySemanticError, Step: stepID, Message: message})
}

func checkAccessIntent(result *report, app application.Application, s step, pattern application.Pattern, intent accessIntent) {
	if !intent.Required.Valid() {
		result.Diagnostics = append(result.Diagnostics, diagnostic{Code: "BUSINESS_ACCESS_INTENT_INVALID", Severity: severitySemanticError, Step: s.ID, Message: fmt.Sprintf("requested access %qは不正です", intent.Required)})
		return
	}

	datasource := string(pattern.Datasource)
	if strings.TrimSpace(datasource) == "" {
		result.Diagnostics = append(result.Diagnostics, diagnostic{
			Code:     "BUSINESS_FUNCTION_DATASOURCE_TRACE_MISSING",
			Severity: severityArchitecturePolicy,
			Step:     s.ID,
			Message:  fmt.Sprintf("Application Pattern %sからdatasourceをtraceできません", pattern.ID),
		})
		return
	}

	requirement, found := findDatasource(app, datasource)
	if !found {
		result.Diagnostics = append(result.Diagnostics, diagnostic{Code: "BUSINESS_DATASOURCE_NOT_FOUND", Severity: severitySemanticError, Step: s.ID, Message: fmt.Sprintf("Application %sにPattern %sが参照するdatasource %qがありません", app.ID, pattern.ID, datasource)})
		return
	}

	if !allowsAccess(requirement.Access, intent.Required) {
		result.Diagnostics = append(result.Diagnostics, diagnostic{
			Code:     "BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT",
			Severity: severityArchitecturePolicy,
			Step:     s.ID,
			Message: fmt.Sprintf(
				"Application %s Pattern %sから導出したdatasource %sのaccepted accessは%sですが、Business/Function probeは%sを要求しています。Projectionだけでaccessを拡張してはいけません",
				app.ID,
				pattern.ID,
				datasource,
				requirement.Access,
				intent.Required,
			),
		})
	}
}

func findDatasource(app application.Application, alias string) (application.DatabaseRequirement, bool) {
	for _, requirement := range app.Databases {
		if string(requirement.Alias) == alias {
			return requirement, true
		}
	}
	return application.DatabaseRequirement{}, false
}

func allowsAccess(granted, required database.Access) bool {
	switch required {
	case database.AccessRead:
		return granted == database.AccessRead || granted == database.AccessReadWrite
	case database.AccessWrite:
		return granted == database.AccessWrite || granted == database.AccessReadWrite
	case database.AccessReadWrite:
		return granted == database.AccessReadWrite
	default:
		return false
	}
}

func applyAccessProbe(result *report, world ir.World, f fixture, probe accessProbe) {
	if probe.Procedure != f.Procedure.ID {
		result.Diagnostics = append(result.Diagnostics, diagnostic{Code: "BUSINESS_PROBE_PROCEDURE_MISMATCH", Severity: severitySemanticError, Message: fmt.Sprintf("probe procedure %qはfixture procedure %qと一致しません", probe.Procedure, f.Procedure.ID)})
		sortDiagnostics(result.Diagnostics)
		return
	}
	for _, s := range f.Steps {
		if s.ID != probe.Step {
			continue
		}
		if !contains(s.ApplicationPatterns, probe.ApplicationPattern) {
			result.Diagnostics = append(result.Diagnostics, diagnostic{Code: "BUSINESS_PROBE_PATTERN_MISMATCH", Severity: severitySemanticError, Step: s.ID, Message: fmt.Sprintf("probe pattern %qはstepのexplicit referenceではありません", probe.ApplicationPattern)})
			sortDiagnostics(result.Diagnostics)
			return
		}
		app, pattern, ok := findPatternOwner(world, probe.ApplicationPattern)
		if !ok {
			result.Diagnostics = append(result.Diagnostics, diagnostic{Code: "BUSINESS_APPLICATION_PATTERN_REF_NOT_FOUND", Severity: severitySemanticError, Step: s.ID, Message: fmt.Sprintf("probe pattern %qをnormalized Worldでexactに1件解決できません", probe.ApplicationPattern)})
			sortDiagnostics(result.Diagnostics)
			return
		}
		checkAccessIntent(result, app, s, pattern, probe.RequestedAccess)
		sortDiagnostics(result.Diagnostics)
		return
	}
	result.Diagnostics = append(result.Diagnostics, diagnostic{Code: "BUSINESS_PROBE_STEP_NOT_FOUND", Severity: severitySemanticError, Step: probe.Step, Message: "probe対象stepがfixtureにありません"})
	sortDiagnostics(result.Diagnostics)
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func sortDiagnostics(values []diagnostic) {
	sort.SliceStable(values, func(i, j int) bool {
		if values[i].Step != values[j].Step {
			return values[i].Step < values[j].Step
		}
		return values[i].Code < values[j].Code
	})
}

func checkExpectedDiagnostics(values []diagnostic, expected string) error {
	var blocking []string
	for _, item := range values {
		if item.Severity == severitySemanticError || item.Severity == severityArchitecturePolicy {
			blocking = append(blocking, item.Code)
		}
	}

	if strings.TrimSpace(expected) == "" {
		if len(blocking) == 0 {
			return nil
		}
		return fmt.Errorf("blocking cross-view diagnosticがあります: %s", strings.Join(blocking, ", "))
	}

	found := false
	var unexpected []string
	for _, code := range blocking {
		if code == expected {
			found = true
			continue
		}
		unexpected = append(unexpected, code)
	}
	if !found {
		return fmt.Errorf("expected diagnostic %sが再現されませんでした; blocking=%s", expected, strings.Join(blocking, ", "))
	}
	if len(unexpected) > 0 {
		return fmt.Errorf("expected diagnostic %s以外のblocking diagnosticがあります: %s", expected, strings.Join(unexpected, ", "))
	}
	return nil
}
