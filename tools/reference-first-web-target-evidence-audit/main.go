package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Provenance struct {
	HandoverBaselineID         string `json:"handoverBaselineId"`
	BaselineRepositoryRevision string `json:"baselineRepositoryRevision"`
	TargetBuildRevision        string `json:"targetBuildRevision"`
	TargetAdapter              string `json:"targetAdapter"`
	Host                       string `json:"host"`
}

type manifestArtifact struct {
	Provenance
	Stage         string                    `json:"stage"`
	ApplicationID string                    `json:"applicationId"`
	Patterns      []manifestPatternArtifact `json:"patterns"`
	Unresolved    []string                  `json:"unresolved"`
}

type manifestPatternArtifact struct {
	ID                    string              `json:"id"`
	Kind                  string              `json:"kind"`
	PresentationCandidate string              `json:"presentationCandidate"`
	Capability            string              `json:"capability,omitempty"`
	Datasource            *manifestDatasource `json:"datasource,omitempty"`
}

type manifestDatasource struct {
	Alias  string `json:"alias"`
	Target string `json:"target"`
	Access string `json:"access"`
}

type readEnvelope struct {
	Provenance
	Result readArtifact `json:"result"`
}

type readArtifact struct {
	ScenarioID      string                   `json:"scenarioId"`
	ApplicationID   string                   `json:"applicationId"`
	RuntimeBindings []runtimeBindingArtifact `json:"runtimeBindings"`
}

type runtimeBindingArtifact struct {
	Pattern            string `json:"pattern"`
	Environment        string `json:"environment,omitempty"`
	Instance           string `json:"instance,omitempty"`
	Provider           string `json:"provider,omitempty"`
	ExecutorCapability string `json:"executorCapability,omitempty"`
	ContractQuery      string `json:"contractQuery,omitempty"`
	Datasource         string `json:"datasource,omitempty"`
	Database           string `json:"database,omitempty"`
	Access             string `json:"access,omitempty"`
}

type commandEnvelope struct {
	Provenance
	Result commandArtifact `json:"result"`
}

type commandErrorEnvelope struct {
	Provenance
	Result         commandArtifact `json:"result"`
	ExecutionError string          `json:"executionError"`
}

type commandArtifact struct {
	ScenarioID     string                       `json:"scenarioId"`
	ApplicationID  string                       `json:"applicationId"`
	Stage          string                       `json:"stage"`
	Authentication *businessResultArtifact      `json:"authentication,omitempty"`
	Validations    []validationResultArtifact   `json:"validations,omitempty"`
	Authorization  *businessResultArtifact      `json:"authorization,omitempty"`
	Command        *businessResultArtifact      `json:"command,omitempty"`
	Observation    *observationArtifact         `json:"observation,omitempty"`
}

type validationResultArtifact struct {
	Validation string `json:"validation"`
	Result     string `json:"result"`
}

type businessResultArtifact struct {
	Capability string         `json:"capability"`
	Mode       string         `json:"mode"`
	Outcome    string         `json:"outcome,omitempty"`
	Payload    map[string]any `json:"payload,omitempty"`
}

type observationArtifact struct {
	ReadIdentity string `json:"readIdentity"`
	Nature       string `json:"nature"`
	Result       any    `json:"result"`
}

type runMetadata struct {
	RunIdentity                 string `json:"runIdentity"`
	TargetBuildRevision         string `json:"targetBuildRevision"`
	RepositoryPreRunRevision    string `json:"repositoryPreRunRevision"`
	RepositoryPreRunClean       bool   `json:"repositoryPreRunClean"`
	RepositoryPostRunRevision   string `json:"repositoryPostRunRevision"`
	RepositoryPostRunClean      bool   `json:"repositoryPostRunClean"`
	SourceQualificationPassed   bool   `json:"sourceQualificationPassed"`
}

type evidenceArtifactDigest struct {
	File   string `json:"file"`
	SHA256 string `json:"sha256"`
}

type evidenceCheck struct {
	Passed   bool     `json:"passed"`
	Checks   []string `json:"checks"`
	Artifacts []string `json:"artifacts"`
}

type visualEvidenceState struct {
	Status   string `json:"status"`
	Required bool   `json:"required"`
	Reason   string `json:"reason"`
}

type evidenceSummary struct {
	EvidenceNature              string                   `json:"evidenceNature"`
	RunIdentity                 string                   `json:"runIdentity"`
	HandoverBaselineID          string                   `json:"handoverBaselineId"`
	BaselineRepositoryRevision  string                   `json:"baselineRepositoryRevision"`
	TargetBuildRevision         string                   `json:"targetBuildRevision"`
	TargetAdapter               string                   `json:"targetAdapter"`
	Host                        string                   `json:"host"`
	SourceQualification         evidenceCheck            `json:"sourceQualification"`
	SemanticConformance         evidenceCheck            `json:"semanticConformance"`
	FunctionalEvidence          evidenceCheck            `json:"functionalEvidence"`
	VisualEvidence              visualEvidenceState      `json:"visualEvidence"`
	Artifacts                   []evidenceArtifactDigest `json:"artifacts"`
	Unresolved                  []string                 `json:"unresolved"`
	GateCReady                  bool                     `json:"gateCReady"`
	AuthorityLimits             []string                 `json:"authorityLimits"`
}

func main() {
	var (
		artifactDir = flag.String("artifact-dir", "", "Directory containing raw First Web Target artifacts")
		runID       = flag.String("run-id", "", "Gate C evidence run identity")
		baselineID  = flag.String("baseline-id", "", "Handover Baseline ID")
		baselineRev = flag.String("baseline-revision", "", "Baseline Repository revision R")
		targetRev   = flag.String("target-revision", "", "Target build revision T")
	)
	flag.Parse()

	if *artifactDir == "" || *runID == "" || *baselineID == "" || *baselineRev == "" || *targetRev == "" {
		fatalf("-artifact-dir, -run-id, -baseline-id, -baseline-revision and -target-revision are required")
	}
	summary, err := auditArtifacts(*artifactDir, *runID, Provenance{
		HandoverBaselineID:         *baselineID,
		BaselineRepositoryRevision: *baselineRev,
		TargetBuildRevision:        *targetRev,
		TargetAdapter:              "reference-first-web-target",
		Host:                       "go-net-http",
	})
	if err != nil {
		fatalf("%v", err)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(summary); err != nil {
		fatalf("encode evidence summary: %v", err)
	}
}

func auditArtifacts(dir, runID string, expected Provenance) (evidenceSummary, error) {
	var metadata runMetadata
	if err := readJSONArtifact(dir, "run-metadata.json", &metadata); err != nil {
		return evidenceSummary{}, err
	}
	if metadata.RunIdentity != runID ||
		metadata.TargetBuildRevision != expected.TargetBuildRevision ||
		metadata.RepositoryPreRunRevision != expected.TargetBuildRevision ||
		metadata.RepositoryPostRunRevision != expected.TargetBuildRevision ||
		!metadata.RepositoryPreRunClean ||
		!metadata.RepositoryPostRunClean ||
		!metadata.SourceQualificationPassed {
		return evidenceSummary{}, fmt.Errorf("run metadata does not prove one clean Target revision: %+v", metadata)
	}

	health, err := os.ReadFile(filepath.Join(dir, "health.txt"))
	if err != nil {
		return evidenceSummary{}, fmt.Errorf("read health.txt: %w", err)
	}
	if string(health) != "ok\n" {
		return evidenceSummary{}, fmt.Errorf("health.txt=%q, want %q", string(health), "ok\n")
	}
	serverLog, err := os.ReadFile(filepath.Join(dir, "server.log"))
	if err != nil {
		return evidenceSummary{}, fmt.Errorf("read server.log: %w", err)
	}
	for _, want := range []string{
		"TEI first Web Target projection:",
		"application=order_entry",
		"baseline=" + expected.HandoverBaselineID,
		"R=" + expected.BaselineRepositoryRevision,
		"T=" + expected.TargetBuildRevision,
	} {
		if !strings.Contains(string(serverLog), want) {
			return evidenceSummary{}, fmt.Errorf("server.log does not contain %q", want)
		}
	}

	var manifest manifestArtifact
	if err := readJSONArtifact(dir, "manifest.json", &manifest); err != nil {
		return evidenceSummary{}, err
	}
	if err := requireProvenance("manifest.json", manifest.Provenance, expected); err != nil {
		return evidenceSummary{}, err
	}
	if manifest.ApplicationID != "order_entry" {
		return evidenceSummary{}, fmt.Errorf("manifest Application=%q, want order_entry", manifest.ApplicationID)
	}
	if manifest.Stage != "standard-and-custom-ui-shared-boundary" {
		return evidenceSummary{}, fmt.Errorf("manifest stage=%q", manifest.Stage)
	}
	if err := requireManifestPatterns(manifest.Patterns); err != nil {
		return evidenceSummary{}, err
	}

	var read readEnvelope
	if err := readJSONArtifact(dir, "read.json", &read); err != nil {
		return evidenceSummary{}, err
	}
	if err := requireProvenance("read.json", read.Provenance, expected); err != nil {
		return evidenceSummary{}, err
	}
	if read.Result.ScenarioID != "order-entry.read-only-web-slice" || read.Result.ApplicationID != "order_entry" {
		return evidenceSummary{}, fmt.Errorf("read Scenario identity is unexpected: %+v", read.Result)
	}
	if err := requireRuntimeBindings(read.Result.RuntimeBindings); err != nil {
		return evidenceSummary{}, err
	}

	var success commandEnvelope
	if err := readJSONArtifact(dir, "submit-success.json", &success); err != nil {
		return evidenceSummary{}, err
	}
	if err := requireProvenance("submit-success.json", success.Provenance, expected); err != nil {
		return evidenceSummary{}, err
	}
	if err := requireSuccessfulCommand(success.Result); err != nil {
		return evidenceSummary{}, err
	}

	var authDenied commandEnvelope
	if err := readJSONArtifact(dir, "submit-authentication-denied.json", &authDenied); err != nil {
		return evidenceSummary{}, err
	}
	if err := requireProvenance("submit-authentication-denied.json", authDenied.Provenance, expected); err != nil {
		return evidenceSummary{}, err
	}
	if authDenied.Result.Stage != "authentication-denied" ||
		authDenied.Result.Authentication == nil ||
		authDenied.Result.Authentication.Outcome != "denied" ||
		authDenied.Result.Command != nil ||
		authDenied.Result.Observation != nil {
		return evidenceSummary{}, fmt.Errorf("authentication denial boundary is not preserved")
	}

	var authzDenied commandEnvelope
	if err := readJSONArtifact(dir, "submit-authorization-denied.json", &authzDenied); err != nil {
		return evidenceSummary{}, err
	}
	if err := requireProvenance("submit-authorization-denied.json", authzDenied.Provenance, expected); err != nil {
		return evidenceSummary{}, err
	}
	if authzDenied.Result.Stage != "authorization-denied" ||
		authzDenied.Result.Authorization == nil ||
		authzDenied.Result.Authorization.Outcome != "denied" ||
		authzDenied.Result.Command != nil ||
		authzDenied.Result.Observation != nil {
		return evidenceSummary{}, fmt.Errorf("authorization denial boundary is not preserved")
	}

	var validationNotSatisfied commandEnvelope
	if err := readJSONArtifact(dir, "submit-validation-not-satisfied.json", &validationNotSatisfied); err != nil {
		return evidenceSummary{}, err
	}
	if err := requireProvenance("submit-validation-not-satisfied.json", validationNotSatisfied.Provenance, expected); err != nil {
		return evidenceSummary{}, err
	}
	if validationNotSatisfied.Result.Stage != "validation-not-satisfied" ||
		len(validationNotSatisfied.Result.Validations) != 1 ||
		validationNotSatisfied.Result.Validations[0].Validation != "sales.order_line.quantity_positive" ||
		validationNotSatisfied.Result.Validations[0].Result != "not-satisfied" ||
		validationNotSatisfied.Result.Authorization != nil ||
		validationNotSatisfied.Result.Command != nil ||
		validationNotSatisfied.Result.Observation != nil {
		return evidenceSummary{}, fmt.Errorf("Validation not-satisfied boundary is not preserved")
	}

	var executionError commandErrorEnvelope
	if err := readJSONArtifact(dir, "submit-execution-error.json", &executionError); err != nil {
		return evidenceSummary{}, err
	}
	if err := requireProvenance("submit-execution-error.json", executionError.Provenance, expected); err != nil {
		return evidenceSummary{}, err
	}
	if executionError.Result.Stage != "command" ||
		executionError.Result.Authorization == nil ||
		executionError.Result.Authorization.Outcome != "allowed" ||
		executionError.Result.Command != nil ||
		executionError.Result.Observation != nil ||
		!strings.Contains(executionError.ExecutionError, "synthetic submit adapter unavailable for order 1003") {
		return evidenceSummary{}, fmt.Errorf("Command Execution Error boundary is not preserved")
	}

	customGET, err := os.ReadFile(filepath.Join(dir, "custom-submit-get.html"))
	if err != nil {
		return evidenceSummary{}, fmt.Errorf("read custom-submit-get.html: %w", err)
	}
	customPOST, err := os.ReadFile(filepath.Join(dir, "custom-submit-success.html"))
	if err != nil {
		return evidenceSummary{}, fmt.Errorf("read custom-submit-success.html: %w", err)
	}
	for _, check := range []struct {
		name    string
		content []byte
		want    []string
	}{
		{
			name:    "custom-submit-get.html",
			content: customGET,
			want: []string{
				"Custom Submit UI",
				`action="/custom-submit"`,
				expected.HandoverBaselineID,
				expected.BaselineRepositoryRevision,
				expected.TargetBuildRevision,
			},
		},
		{
			name:    "custom-submit-success.html",
			content: customPOST,
			want: []string{
				"order-entry.submit-order-web-slice",
				"sales.order.submit",
				"synthetic.order_state.read",
				"submitted",
			},
		},
	} {
		for _, want := range check.want {
			if !strings.Contains(string(check.content), want) {
				return evidenceSummary{}, fmt.Errorf("%s does not contain %q", check.name, want)
			}
		}
		for _, forbidden := range []string{
			"editor-secret",
			"viewer-secret",
			"credential-ref://",
		} {
			if strings.Contains(string(check.content), forbidden) {
				return evidenceSummary{}, fmt.Errorf("%s contains credential material %q", check.name, forbidden)
			}
		}
	}

	artifactNames := []string{
		"run-metadata.json",
		"source-tests.log",
		"server.log",
		"health.txt",
		"manifest.json",
		"read.json",
		"submit-success.json",
		"submit-authentication-denied.json",
		"submit-authorization-denied.json",
		"submit-validation-not-satisfied.json",
		"submit-execution-error.json",
		"custom-submit-get.html",
		"custom-submit-success.html",
	}
	if err := requireNoReferenceCredentialMaterial(dir, artifactNames); err != nil {
		return evidenceSummary{}, err
	}
	digests := make([]evidenceArtifactDigest, 0, len(artifactNames))
	for _, name := range artifactNames {
		digest, err := digestArtifact(dir, name)
		if err != nil {
			return evidenceSummary{}, err
		}
		digests = append(digests, digest)
	}

	return evidenceSummary{
		EvidenceNature:             "first-web-target-semantic-functional-evidence",
		RunIdentity:                runID,
		HandoverBaselineID:         expected.HandoverBaselineID,
		BaselineRepositoryRevision: expected.BaselineRepositoryRevision,
		TargetBuildRevision:        expected.TargetBuildRevision,
		TargetAdapter:              expected.TargetAdapter,
		Host:                       expected.Host,
		SourceQualification: evidenceCheck{
			Passed: true,
			Checks: []string{
				"source tests passed with fixed Go environment",
				"pre-run Repository was clean at targetBuildRevision",
				"post-run Repository remained clean at the same targetBuildRevision",
			},
			Artifacts: []string{"run-metadata.json", "source-tests.log", "server.log", "health.txt"},
		},
		SemanticConformance: evidenceCheck{
			Passed: true,
			Checks: []string{
				"B/R/T provenance exact across manifest and scenario artifacts",
				"order_entry five Pattern identities retained",
				"master datasource remains read-only",
				"command Pattern remains sales.order.submit",
				"reference/header-detail runtime bindings preserve explicit Environment/provider identities",
			},
			Artifacts: []string{"manifest.json", "read.json"},
		},
		FunctionalEvidence: evidenceCheck{
			Passed: true,
			Checks: []string{
				"read-only browse/record/reference/header-detail scenario completed",
				"authenticated submit scenario completed",
				"canonical validations satisfied before allowed command",
				"authentication denial remained Business Outcome",
				"authorization denial remained Business Outcome",
				"Canonical Validation not-satisfied stopped before Authorization and Command",
				"Command Invocation Execution Error remained distinct from Business Result",
				"Command Business Result remained distinct from separate Observation",
				"Custom UI rendered the same Command scenario result without credential material",
			},
			Artifacts: []string{
				"read.json",
				"submit-success.json",
				"submit-authentication-denied.json",
				"submit-authorization-denied.json",
				"submit-validation-not-satisfied.json",
				"submit-execution-error.json",
				"custom-submit-success.html",
			},
		},
		VisualEvidence: visualEvidenceState{
			Status:   "pending",
			Required: true,
			Reason:   "HTML/HTTP artifacts are not Visual Fidelity Evidence; capture screenshot or equivalent separately",
		},
		Artifacts: digests,
		Unresolved: []string{
			"visual-fidelity-evidence-not-acquired",
			"browse-environment-instance-binding-not-exposed-by-current-driver-api",
			"actual-external-command-observation-not-acquired",
		},
		GateCReady: false,
		AuthorityLimits: []string{
			"this-audit-does-not-declare-gate-c-ready",
			"semantic-functional-pass-does-not-substitute-visual-evidence",
			"html-presence-does-not-count-as-visual-fidelity",
			"evidence-artifacts-are-derived-not-canonical",
		},
	}, nil
}

func requireProvenance(name string, actual, expected Provenance) error {
	if actual != expected {
		return fmt.Errorf("%s provenance=%+v, want %+v", name, actual, expected)
	}
	return nil
}

func requireManifestPatterns(patterns []manifestPatternArtifact) error {
	byID := make(map[string]manifestPatternArtifact, len(patterns))
	for _, pattern := range patterns {
		if _, exists := byID[pattern.ID]; exists {
			return fmt.Errorf("manifest Pattern %s is duplicated", pattern.ID)
		}
		byID[pattern.ID] = pattern
	}
	expected := map[string]string{
		"order_entry.recent_orders":     "browse",
		"order_entry.order_record":      "record",
		"order_entry.customer_reference": "reference",
		"order_entry.order_lines":       "header-detail",
		"order_entry.submit_order":      "command",
	}
	if len(byID) != len(expected) {
		return fmt.Errorf("manifest Pattern count=%d, want=%d", len(byID), len(expected))
	}
	for id, kind := range expected {
		pattern, ok := byID[id]
		if !ok || pattern.Kind != kind {
			return fmt.Errorf("manifest Pattern %s=%+v, want kind %s", id, pattern, kind)
		}
	}
	reference := byID["order_entry.customer_reference"]
	if reference.Datasource == nil ||
		reference.Datasource.Alias != "master" ||
		reference.Datasource.Target != "master_data" ||
		reference.Datasource.Access != "read" {
		return fmt.Errorf("customer reference datasource boundary=%+v", reference.Datasource)
	}
	command := byID["order_entry.submit_order"]
	if command.Capability != "sales.order.submit" {
		return fmt.Errorf("submit Pattern Capability=%q", command.Capability)
	}
	return nil
}

func requireRuntimeBindings(bindings []runtimeBindingArtifact) error {
	byPattern := make(map[string]runtimeBindingArtifact, len(bindings))
	for _, binding := range bindings {
		byPattern[binding.Pattern] = binding
	}
	browse := byPattern["order_entry.recent_orders"]
	if browse.Datasource != "sales" || browse.Database != "sales_core" || browse.Access != "read-write" {
		return fmt.Errorf("browse binding=%+v", browse)
	}
	if browse.Environment != "" || browse.Instance != "" {
		return fmt.Errorf("browse Environment was inferred: %+v", browse)
	}
	customer := byPattern["order_entry.customer_reference"]
	if customer.Environment != "development" ||
		customer.Instance != "master_data.dev" ||
		customer.Provider != "oracle" ||
		customer.ExecutorCapability != "database.query.oracle" ||
		customer.ContractQuery != "master_data/master_data.customer/find_by_id" {
		return fmt.Errorf("customer runtime binding=%+v", customer)
	}
	lines := byPattern["order_entry.order_lines"]
	if lines.Environment != "development" ||
		lines.Instance != "sales_core.dev" ||
		lines.Provider != "postgres" ||
		lines.ExecutorCapability != "database.query.postgres" ||
		lines.ContractQuery != "sales_core/sales_core.order_line/find_by_order" {
		return fmt.Errorf("order-lines runtime binding=%+v", lines)
	}
	return nil
}

func requireSuccessfulCommand(result commandArtifact) error {
	if result.ScenarioID != "order-entry.submit-order-web-slice" ||
		result.ApplicationID != "order_entry" ||
		result.Stage != "completed" {
		return fmt.Errorf("successful command Scenario identity/stage=%+v", result)
	}
	if result.Authentication == nil ||
		result.Authentication.Capability != "security.authenticate" ||
		result.Authentication.Outcome != "authenticated" {
		return fmt.Errorf("successful command Authentication=%+v", result.Authentication)
	}
	if len(result.Validations) != 2 {
		return fmt.Errorf("successful command Validation count=%d", len(result.Validations))
	}
	for _, validation := range result.Validations {
		if validation.Result != "satisfied" {
			return fmt.Errorf("successful command Validation=%+v", validation)
		}
	}
	if result.Authorization == nil ||
		result.Authorization.Capability != "security.authorize" ||
		result.Authorization.Outcome != "allowed" {
		return fmt.Errorf("successful command Authorization=%+v", result.Authorization)
	}
	if result.Command == nil ||
		result.Command.Capability != "sales.order.submit" ||
		result.Command.Mode != "output" {
		return fmt.Errorf("successful command Business Result=%+v", result.Command)
	}
	if result.Observation == nil ||
		result.Observation.ReadIdentity != "synthetic.order_state.read" ||
		result.Observation.Nature != "synthetic" ||
		fmt.Sprint(result.Observation.Result) != "submitted" {
		return fmt.Errorf("successful command Observation=%+v", result.Observation)
	}
	return nil
}

func readJSONArtifact(dir, name string, target any) error {
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return fmt.Errorf("read %s: %w", name, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("decode %s: %w", name, err)
	}
	return nil
}

func requireNoReferenceCredentialMaterial(dir string, names []string) error {
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("read %s for credential scan: %w", name, err)
		}
		for _, forbidden := range []string{
			"editor-secret",
			"viewer-secret",
			"locked-secret",
			"disabled-secret",
			"credential-ref://legacy/",
		} {
			if strings.Contains(string(data), forbidden) {
				return fmt.Errorf("%s contains Reference credential material %q", name, forbidden)
			}
		}
	}
	return nil
}

func digestArtifact(dir, name string) (evidenceArtifactDigest, error) {
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return evidenceArtifactDigest{}, fmt.Errorf("read %s for digest: %w", name, err)
	}
	sum := sha256.Sum256(data)
	return evidenceArtifactDigest{
		File:   name,
		SHA256: hex.EncodeToString(sum[:]),
	}, nil
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\\n", args...)
	os.Exit(1)
}
