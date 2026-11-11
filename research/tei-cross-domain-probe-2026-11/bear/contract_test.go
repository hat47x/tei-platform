package bearprobe

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/validation"
	"tei/internal/workspace"
)

// The TEI Workspace and this probe are two declarations of the same boundary.
// If one drifts, this test fails instead of letting the probe check a boundary
// that no longer matches the model.
func TestWorkspaceEnumsMatchRuntimeContract(t *testing.T) {
	namespace := loadNamespace(t)
	cases := []struct {
		typeID string
		want   []string
	}{
		{"bear.intent", ValidIntents},
		{"bear.cognition_mode", ValidCognitionModes},
		{"bear.obstacle_mode", ValidObstacleModes},
		{"bear.recovery_state", ValidRecoveryStates},
		{"bear.decision_source", ValidDecisionSources},
		{"bear.observation_version", []string{ObservationVersion}},
	}
	for _, testCase := range cases {
		got := typeValues(t, namespace, testCase.typeID)
		want := sortedCopy(testCase.want)
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Fatalf("%s values = %v want %v", testCase.typeID, got, want)
		}
	}
}

func TestWorkspaceDeclaresScenarioValidationKinds(t *testing.T) {
	namespace := loadNamespace(t)
	kinds := map[string]semantic.ValidationKind{}
	for _, validation := range namespace.Validations {
		kinds[string(validation.ID)] = validation.Kind
	}
	if len(kinds) != 2 {
		t.Fatalf("validations = %v want 2", kinds)
	}
	for id, kind := range kinds {
		if kind != semantic.ValidationPositive {
			t.Fatalf("validation %s kind = %s want positive", id, kind)
		}
	}
}

func TestValidDecisionFixturePasses(t *testing.T) {
	decision := readDecision(t, "decision_valid.jsonl")
	if diagnostics := ValidateDecision(decision, 10.05); len(diagnostics) != 0 {
		t.Fatalf("valid decision produced diagnostics: %+v", diagnostics)
	}
	if weight := Weight(decision, 10.05); weight != 1 {
		t.Fatalf("weight at +50 ms = %v want 1", weight)
	}
	if weight := Weight(decision, 10.11); weight < 0.93 || weight > 0.94 {
		t.Fatalf("weight at +110 ms = %v want about 0.933", weight)
	}
	if weight := Weight(decision, 10.30); weight != 0 {
		t.Fatalf("weight at +300 ms = %v want 0", weight)
	}
}

func TestOutOfRangeDecisionFixtureIsRefused(t *testing.T) {
	decision := readDecision(t, "decision_out_of_range.jsonl")
	if !hasCode(ValidateDecision(decision, 10.05), "BEAR_DECISION_HEADING_OUT_OF_RANGE") {
		t.Fatal("heading 0.30 rad was accepted")
	}
}

func TestDecisionBoundsAreInclusiveAtTheDocumentedEdge(t *testing.T) {
	decision := readDecision(t, "decision_valid.jsonl")
	for _, edge := range []float64{-MaxHeadingBias, MaxHeadingBias} {
		decision.HeadingBiasRad = edge
		if diagnostics := ValidateDecision(decision, 10.05); len(diagnostics) != 0 {
			t.Fatalf("heading edge %v was refused: %+v", edge, diagnostics)
		}
	}
	decision.HeadingBiasRad = MaxHeadingBias + 0.0001
	if !hasCode(ValidateDecision(decision, 10.05), "BEAR_DECISION_HEADING_OUT_OF_RANGE") {
		t.Fatal("heading beyond the edge was accepted")
	}
}

func TestDecisionRefusesFutureExpiredAndInvertedValidity(t *testing.T) {
	decision := readDecision(t, "decision_valid.jsonl")
	if !hasCode(ValidateDecision(decision, 9.99), "BEAR_DECISION_FROM_THE_FUTURE") {
		t.Fatal("a decision issued after now was accepted")
	}
	if !hasCode(ValidateDecision(decision, 10.21), "BEAR_DECISION_EXPIRED") {
		t.Fatal("an expired decision was accepted")
	}
	decision.ValidUntilS = decision.TimestampS - 0.01
	if !hasCode(ValidateDecision(decision, 10.05), "BEAR_DECISION_VALIDITY_INVERTED") {
		t.Fatal("valid_until_s before timestamp_s was accepted")
	}
}

func TestEncodeWireMatchesTheDocumentedExample(t *testing.T) {
	decision := readDecision(t, "decision_valid.jsonl")
	decision.Intent = "maintain"
	wire, err := EncodeWire(decision)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if wire != "maintain|+045|+08|+12|10|88" {
		t.Fatalf("wire = %q want %q", wire, "maintain|+045|+08|+12|10|88")
	}

	decision.HeadingBiasRad = -0.005
	wire, err = EncodeWire(decision)
	if err != nil {
		t.Fatalf("encode negative heading: %v", err)
	}
	if !strings.HasPrefix(wire, "maintain|-005|") {
		t.Fatalf("wire = %q want a -005 heading field", wire)
	}
}

func TestEncodeWireRefusesOutOfRangeInsteadOfClamping(t *testing.T) {
	decision := readDecision(t, "decision_out_of_range.jsonl")
	if _, err := EncodeWire(decision); err == nil {
		t.Fatal("an out-of-range decision was encoded")
	}
}

func TestParseWireRoundTripsTheDocumentedLine(t *testing.T) {
	decision, err := ParseWire("maintain|+045|+08|+12|10|88", 1.0, 0.2)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if decision.Intent != "maintain" || decision.HeadingBiasRad != 0.045 || decision.PersistenceBias != 0.08 {
		t.Fatalf("parsed decision = %+v", decision)
	}
	if decision.Uncertainty != 0.10 || decision.Confidence != 0.88 {
		t.Fatalf("parsed uncertainty/confidence = %v/%v", decision.Uncertainty, decision.Confidence)
	}
	if decision.ValidUntilS != 1.2 || decision.Source != "typed_wire" {
		t.Fatalf("parsed validity = %v source = %q", decision.ValidUntilS, decision.Source)
	}
}

func TestParseWireRefusesMalformedLines(t *testing.T) {
	cases := []string{
		"maintain|+045|+08|+12|10",
		"stare|+045|+08|+12|10|88",
		"maintain|+04x|+08|+12|10|88",
		"maintain|+045|-08|+12|-10|88",
		"maintain|+045|+08|+12|10|88|extra",
	}
	for _, line := range cases {
		if _, err := ParseWire(line, 1.0, 0.2); err == nil {
			t.Fatalf("malformed line %q was accepted", line)
		}
	}
}

func TestParseWireClampsTTLLikeTheRuntime(t *testing.T) {
	decision, err := ParseWire("maintain|+000|+00|+00|00|80", 5.0, 2.0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if decision.ValidUntilS != 5.5 {
		t.Fatalf("valid_until_s = %v want 5.5 (clamped to 0.50 s)", decision.ValidUntilS)
	}
}

func TestProjectScenarioSmokeFixture(t *testing.T) {
	document := readFixture(t, "scenario_smoke.json")
	expectation, diagnostics, err := ProjectScenario(document, "smoke")
	if err != nil {
		t.Fatalf("project: %v", err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("smoke fixture produced diagnostics: %+v", diagnostics)
	}
	if expectation.ScenarioID != "smoke" || expectation.Seed != 7 || expectation.TotalTicks != 35 || expectation.MinSteps != 2 {
		t.Fatalf("expectation = %+v", expectation)
	}
	if !expectation.HasFinalPositionTolerance || expectation.FinalPositionTolerance != 0.001 {
		t.Fatalf("tolerance = %v declared %v", expectation.FinalPositionTolerance, expectation.HasFinalPositionTolerance)
	}
	if expectation.TrajectorySHA256 != "0fdcd61d750c68ccaf5ee9b4b9ae7756642cac9120ff1952528bb3ebed9bc167" {
		t.Fatalf("trajectory = %q", expectation.TrajectorySHA256)
	}
}

func TestProjectScenarioRefusesUnknownAndInvalidAssertions(t *testing.T) {
	cases := []struct {
		name     string
		mutate   func(map[string]json.RawMessage)
		wantCode string
	}{
		{
			name: "unknown assertion",
			mutate: func(assertions map[string]json.RawMessage) {
				assertions["unknown"] = json.RawMessage(`true`)
			},
			wantCode: "BEAR_SCENARIO_ASSERTION_UNKNOWN",
		},
		{
			name: "short trajectory hash",
			mutate: func(assertions map[string]json.RawMessage) {
				assertions["trajectory_sha256"] = json.RawMessage(`"abc"`)
			},
			wantCode: "BEAR_SCENARIO_TRAJECTORY_HASH_INVALID",
		},
		{
			name: "non-positive tolerance",
			mutate: func(assertions map[string]json.RawMessage) {
				assertions["final_position_tolerance"] = json.RawMessage(`0`)
			},
			wantCode: "BEAR_SCENARIO_TOLERANCE_INVALID",
		},
		{
			name: "empty stop reasons",
			mutate: func(assertions map[string]json.RawMessage) {
				assertions["stop_reasons"] = json.RawMessage(`[]`)
			},
			wantCode: "BEAR_SCENARIO_STOP_REASONS_INVALID",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			document := decodeScenario(t)
			testCase.mutate(document.Assertions)
			encoded, err := json.Marshal(document)
			if err != nil {
				t.Fatal(err)
			}
			_, diagnostics, err := ProjectScenario(encoded, "smoke")
			if err != nil {
				t.Fatalf("project: %v", err)
			}
			if !hasCode(diagnostics, testCase.wantCode) {
				t.Fatalf("diagnostics = %+v; %s was not reported", diagnostics, testCase.wantCode)
			}
		})
	}
}

func TestProjectScenarioRefusesUnknownTopLevelField(t *testing.T) {
	document := []byte(`{"schema":"bot-scenario-1","seed":1,"host":{"kind":"scripted"},"assertions":{"total_ticks":1},"unexpected":true}`)
	if _, _, err := ProjectScenario(document, "bad"); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown top-level field was not refused: %v", err)
	}
}

type rawScenario struct {
	Schema      string                     `json:"schema"`
	Seed        int                        `json:"seed"`
	StartHeight float64                    `json:"start_height"`
	Host        json.RawMessage            `json:"host"`
	Assertions  map[string]json.RawMessage `json:"assertions"`
}

func decodeScenario(t *testing.T) rawScenario {
	t.Helper()
	var scenario rawScenario
	if err := json.Unmarshal(readFixture(t, "scenario_smoke.json"), &scenario); err != nil {
		t.Fatalf("decode scenario: %v", err)
	}
	return scenario
}

func loadNamespace(t *testing.T) semantic.Namespace {
	t.Helper()
	loaded, err := workspace.Load(filepath.Join("workspace"))
	if err != nil {
		t.Fatalf("load Workspace: %v", err)
	}
	for _, diagnostic := range validation.Validate(loaded.World) {
		if diagnostic.Severity == validation.Error {
			t.Fatalf("Workspace validation error: %s %s", diagnostic.Code, diagnostic.Message)
		}
	}
	for _, namespace := range loaded.World.Semantics {
		if namespace.ID == "bear" {
			return namespace
		}
	}
	t.Fatal("semantic namespace bear is not declared")
	return semantic.Namespace{}
}

func typeValues(t *testing.T, namespace semantic.Namespace, id string) []string {
	t.Helper()
	for _, typ := range namespace.Types {
		if string(typ.ID) == id {
			return sortedCopy(typ.Values)
		}
	}
	t.Fatalf("semantic type %s is not declared", id)
	return nil
}

func sortedCopy(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

func readDecision(t *testing.T, name string) Decision {
	t.Helper()
	file, err := os.Open(filepath.Join("fixtures", name))
	if err != nil {
		t.Fatalf("open fixture %s: %v", name, err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		t.Fatalf("fixture %s has no line", name)
	}
	var decision Decision
	if err := json.Unmarshal(scanner.Bytes(), &decision); err != nil {
		t.Fatalf("decode decision %s: %v", name, err)
	}
	return decision
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("fixtures", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func hasCode(diagnostics []Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func BenchmarkEncodeWire(b *testing.B) {
	decision := Decision{
		EpisodeID: 1, RequestID: 2, TimestampS: 10, ValidUntilS: 10.2,
		Intent: "observe", HeadingBiasRad: 0.045, PersistenceBias: 0.08,
		BraceBias: 0.12, Uncertainty: 0.1, Confidence: 0.88, Source: "typed_wire",
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := EncodeWire(decision); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkProjectScenario(b *testing.B) {
	document, err := os.ReadFile(filepath.Join("fixtures", "scenario_smoke.json"))
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, diagnostics, err := ProjectScenario(document, "smoke"); err != nil || len(diagnostics) != 0 {
			b.Fatalf("project: %v %+v", err, diagnostics)
		}
	}
}

func BenchmarkWorkspaceLoadAndValidate(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		loaded, err := workspace.Load(filepath.Join("workspace"))
		if err != nil {
			b.Fatal(err)
		}
		for _, diagnostic := range validation.Validate(loaded.World) {
			if diagnostic.Severity == validation.Error {
				b.Fatalf("%s %s", diagnostic.Code, diagnostic.Message)
			}
		}
	}
}
