package generationgraph

import (
	"strings"
	"testing"
)

// Contract mirrors the issue first Reference: `order_entry` 設定で既存の一覧・
// 受注明細・業務検証を再構成する経路。a pure listing operator shared by both
// phases, plus a generation-only persistence operator.
func sharedOperators(declared []string) []Operator {
	listing := Operator{
		Name:       "list-order-lines",
		Version:    "v1",
		Phases:     declared,
		InputType:  "order-key",
		OutputType: "order-line-set",
	}
	merge := Operator{
		Name:       "merge-persisted-settings",
		Version:    "v1",
		Effects:    []string{EffectPersistence},
		Phases:     []string{PhaseGeneration},
		InputType:  "order-line-set",
		OutputType: "order-line-set",
	}
	return []Operator{listing, merge}
}

func graphFixture(phase string) Graph {
	return Graph{
		Phase:     phase,
		Operators: sharedOperators([]string{PhaseGeneration, PhaseRuntime}),
		Steps: []Step{
			{ID: "req-order", Operator: "list-order-lines", Origin: "requirement:R1"},
			{ID: "spec-lines", Operator: "merge-persisted-settings", Inputs: []string{"req-order"}, Origin: "design:one-list"},
		},
	}
}

func TestPureOperatorIsSharedAcrossPhasesWithTheSameMeaning(t *testing.T) {
	generation := Decide(Input{Graph: graphOf(graphFixture(PhaseGeneration), sharedOperators([]string{PhaseGeneration, PhaseRuntime}))})
	if !generation.Clean || generation.Code != CodeCheckPassed {
		t.Fatalf("generation=%+v want CHECK_PASSED", generation)
	}
	runtimeFixture := graphFixture(PhaseRuntime)
	runtimeFixture.Steps = runtimeFixture.Steps[:1]
	runtime := Decide(Input{Graph: graphOf(runtimeFixture, sharedOperators([]string{PhaseGeneration, PhaseRuntime}))})
	if !runtime.Clean || runtime.Code != CodeCheckPassed {
		t.Fatalf("runtime=%+v want CHECK_PASSED", runtime)
	}
	report := Decide(Input{
		Mode:      "compare",
		Graph:     graphOf(graphFixture(PhaseGeneration), sharedOperators([]string{PhaseGeneration, PhaseRuntime})),
		CompareTo: graphOf(runtimeFixture, sharedOperators([]string{PhaseGeneration, PhaseRuntime})),
	})
	if report.Code != CodeMeaningShared {
		t.Fatalf("report=%+v want MEANING_SHARED", report)
	}
}

func graphOf(fixture Graph, operators []Operator) *Graph {
	graph := fixture
	graph.Operators = operators
	return &graph
}

func TestGenerationOnlyEffectNeverFlowsIntoRuntime(t *testing.T) {
	// The runtime graph must not reference merge-persisted-settings at all.
	restricted := graphFixture(PhaseRuntime)
	restricted.Steps = restricted.Steps[:1]
	report := Decide(Input{Graph: graphOf(restricted, sharedOperators([]string{PhaseGeneration, PhaseRuntime}))})
	if !report.Clean || report.Code != CodeCheckPassed {
		t.Fatalf("report=%+v want CHECK_PASSED", report)
	}
	// And if someone wires it into runtime anyway, the phase list blocks it.
	leaked := graphFixture(PhaseRuntime)
	report = Decide(Input{Graph: &leaked})
	if report.Clean || report.Code != CodePhaseNotAllowed {
		t.Fatalf("report=%+v want PHASE_NOT_ALLOWED", report)
	}
}

func TestEffectBearingOperatorInsideRuntimeGraphIsRefused(t *testing.T) {
	operators := sharedOperators([]string{PhaseGeneration, PhaseRuntime})
	operators[1].Effects = []string{EffectTime} // generation-only capability
	operators[1].Phases = []string{PhaseGeneration, PhaseRuntime}
	graph := graphFixture(PhaseRuntime)
	graph.Operators = operators
	if report := Decide(Input{Graph: &graph}); report.Clean || report.Code != CodeEffectInRuntime {
		t.Fatalf("report=%+v want EFFECT_IN_RUNTIME", report)
	}
}

func TestUnknownOperatorStepsAreRefused(t *testing.T) {
	graph := graphFixture(PhaseGeneration)
	graph.Steps[1].Operator = "gone-operator"
	if report := Decide(Input{Graph: &graph}); report.Clean || report.Code != CodeOperatorUnresolved {
		t.Fatalf("report=%+v want OPERATOR_UNRESOLVED", report)
	}
}

func TestDuplicateOperatorDeclarationsAreRefused(t *testing.T) {
	graph := graphFixture(PhaseGeneration)
	graph.Operators = append(graph.Operators, graph.Operators[0])
	if report := Decide(Input{Graph: &graph}); report.Clean || report.Code != CodeDuplicateOperator {
		t.Fatalf("report=%+v want DUPLICATE_OPERATOR", report)
	}
}

func TestTypedContractMismatchIsRefused(t *testing.T) {
	graph := graphFixture(PhaseGeneration)
	graph.Operators[1].InputType = "customer-list"
	if report := Decide(Input{Graph: &graph}); report.Clean || report.Code != CodeContractMismatch {
		t.Fatalf("report=%+v want OPERATOR_CONTRACT_MISMATCH", report)
	}
}

func TestDependencyCycleIsRefused(t *testing.T) {
	graph := graphFixture(PhaseGeneration)
	graph.Steps[0].Inputs = []string{"spec-lines"}
	graph.Operators[0].InputType = "" // drop the type to reach the cycle check
	if report := Decide(Input{Graph: &graph}); report.Clean || report.Code != CodeTransformCycle {
		t.Fatalf("report=%+v want TRANSFORM_CYCLE", report)
	}
}

func TestUnresolvedRequirementStaysUnresolvedDownstream(t *testing.T) {
	graph := graphFixture(PhaseGeneration)
	graph.Steps[0].Unresolved = true
	report := Decide(Input{Graph: &graph})
	if !report.Clean || report.Code != CodeCheckPassed {
		t.Fatalf("report=%+v want CHECK_PASSED with unresolved kept", report)
	}
	if len(report.Unresolved) != 2 ||
		!strings.Contains(report.Unresolved[0], "req-order") ||
		!strings.Contains(report.Unresolved[1], "spec-lines") {
		t.Fatalf("unresolved=%v want both steps propagated", report.Unresolved)
	}
}

func TestRuleVersionChangeForcesExplicitRecompute(t *testing.T) {
	graph := graphFixture(PhaseGeneration)
	report := Decide(Input{Graph: &graph, CurrentRuleVersion: "rules-v1", IncomingRuleVersion: "rules-v2"})
	if report.Clean || report.Code != CodeRecomputeRequired {
		t.Fatalf("report=%+v want RECOMPUTE_REQUIRED", report)
	}
	if report.Fingerprint == "" {
		t.Fatal("fingerprint must be pinned for the recompute")
	}
	stable := Decide(Input{Graph: &graph, CurrentRuleVersion: "rules-v1", IncomingRuleVersion: "rules-v1"})
	if !stable.Clean || stable.Code != CodeCheckPassed || stable.Fingerprint != report.Fingerprint {
		t.Fatalf("stable=%+v want the same structural fingerprint", stable)
	}
}

func TestChangedOperatorContractBreaksSharedMeaning(t *testing.T) {
	left := Decide(Input{
		Mode:      "compare",
		Graph:     graphOf(graphFixture(PhaseGeneration), sharedOperators([]string{PhaseGeneration, PhaseRuntime})),
		CompareTo: graphOf(graphFixture(PhaseRuntime), sharedOperators([]string{PhaseGeneration, PhaseRuntime})),
	})
	if left.Code != CodeMeaningShared {
		t.Fatalf("report=%+v want MEANING_SHARED before the contract change", left)
	}
	changed := sharedOperators([]string{PhaseGeneration, PhaseRuntime})
	changed[0].Version = "v2"
	report := Decide(Input{
		Mode:      "compare",
		Graph:     graphOf(graphFixture(PhaseGeneration), sharedOperators([]string{PhaseGeneration, PhaseRuntime})),
		CompareTo: graphOf(graphFixture(PhaseRuntime), changed),
	})
	if report.Clean || report.Code != CodeVersionChanged {
		t.Fatalf("report=%+v want OPERATOR_VERSION_CHANGED", report)
	}
}

func TestInvalidRequestsAreRefused(t *testing.T) {
	if report := Decide(Input{}); report.Code != CodeInvalidRequest {
		t.Fatalf("report=%+v want INVALID_REQUEST", report)
	}
	graph := graphFixture(PhaseGeneration)
	graph.Phase = "deployment"
	if report := Decide(Input{Graph: &graph}); report.Code != CodeInvalidRequest {
		t.Fatalf("report=%+v want INVALID_REQUEST", report)
	}
	if report := Decide(Input{Mode: "diff"}); report.Code != CodeInvalidRequest {
		t.Fatalf("report=%+v want INVALID_REQUEST", report)
	}
}
