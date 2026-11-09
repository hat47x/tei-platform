// Package generationgraph is a read-only, synthetic reference for #1063
// (DSLの生成時・実行時二用途と案件別変換工程). It checks one phase-annotated
// transform graph: the same pure Operator may be shared by the generation
// phase and the runtime phase with the same meaning, an Operator that carries
// IO, time, random, network or persistence effects may run only where it is
// declared and never flows into runtime unconditionally, operator contracts
// hook together by type, the dependency subgraph stays acyclic, unresolved
// requirements stay unresolved through propagation, and a rule version change
// forces an explicit recompute instead of silently reusing old output.
//
// It is not a production generator, a second Plugin Resolver, a second Build
// Lock or an accepted business-meaning register. It executes nothing: the
// fingerprint pins the graph structure and operator contracts, not computed
// values, and generation output is never promoted to accepted business
// meaning.
package generationgraph

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Phases of the transform work.
const (
	PhaseAuthoring  = "authoring"
	PhaseValidation = "validation"
	PhaseGeneration = "generation"
	PhaseRuntime    = "runtime"
)

// Example effect declarations of an Operator; any non-empty set makes the
// Operator non-pure for this reference.
const (
	EffectIO          = "io"
	EffectTime        = "time"
	EffectRandom      = "random"
	EffectNetwork     = "network"
	EffectPersistence = "persistence"
)

// Result codes.
const (
	CodeCheckPassed        = "CHECK_PASSED"
	CodeMeaningShared      = "MEANING_SHARED"
	CodeMeaningChanged     = "MEANING_CHANGED"
	CodeVersionChanged     = "OPERATOR_VERSION_CHANGED"
	CodeEffectInRuntime    = "EFFECT_IN_RUNTIME"
	CodePhaseNotAllowed    = "PHASE_NOT_ALLOWED"
	CodeOperatorUnresolved = "OPERATOR_UNRESOLVED"
	CodeDuplicateOperator  = "DUPLICATE_OPERATOR"
	CodeContractMismatch   = "OPERATOR_CONTRACT_MISMATCH"
	CodeTransformCycle     = "TRANSFORM_CYCLE"
	CodeRecomputeRequired  = "RECOMPUTE_REQUIRED"
	CodeInvalidRequest     = "INVALID_REQUEST"
)

// Operator is one declared meaning operation. A pure Operator (empty Effects)
// may be shared between the generation phase and the runtime phase.
type Operator struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	// Effects declare generation-only physical capabilities. Empty means pure.
	Effects []string `json:"effects,omitempty"`
	// Phases lists where the Operator may run.
	Phases []string `json:"phases"`
	// InputType and OutputType give the operator contract. An empty type means
	// the reference does not check that side.
	InputType  string `json:"inputType,omitempty"`
	OutputType string `json:"outputType,omitempty"`
}

// Step is one node of the transform graph. A step is either a source
// (requirement, design candidate or setting) or a produced node.
type Step struct {
	ID       string `json:"id"`
	Operator string `json:"operator"`
	// Inputs reference other step IDs or external source inputs.
	Inputs []string `json:"inputs,omitempty"`
	// Origin records what the step derives from, for example
	// "requirement:R1" or "setting:list-columns".
	Origin string `json:"origin,omitempty"`
	// Unresolved marks a step whose requirement is not settled. It stays
	// unresolved and propagates downstream.
	Unresolved bool `json:"unresolved,omitempty"`
}

// Graph is one phase-annotated transform graph.
type Graph struct {
	Phase     string     `json:"phase"`
	Operators []Operator `json:"operators"`
	Steps     []Step     `json:"steps"`
}

// Input is the whole synthetic request.
type Input struct {
	Graph *Graph `json:"graph"`
	// Mode is "graph" (default) or "compare".
	Mode string `json:"mode,omitempty"`
	// CompareTo is the graph of the other phase when Mode is "compare".
	CompareTo *Graph `json:"compareTo,omitempty"`
	// CurrentRuleVersion is the rule bundle version the current outputs were
	// produced with.
	CurrentRuleVersion string `json:"currentRuleVersion,omitempty"`
	// IncomingRuleVersion is the rule bundle version being applied now.
	IncomingRuleVersion string `json:"incomingRuleVersion,omitempty"`
}

// Report is the read-only result.
type Report struct {
	Code string `json:"code"`
	// Clean is true when the graph passed every structural check.
	Clean       bool     `json:"clean,omitempty"`
	Fingerprint string   `json:"fingerprint,omitempty"`
	Trace       []string `json:"trace,omitempty"`
	// Unresolved lists every step (after downstream propagation) that is not
	// settled.
	Unresolved []string `json:"unresolved,omitempty"`
	Message    string   `json:"message,omitempty"`
}

func reject(code, message string) Report {
	return Report{Code: code, Message: message}
}

func findOperator(name string, operators []Operator) (Operator, bool) {
	for _, operator := range operators {
		if operator.Name == name {
			return operator, true
		}
	}
	return Operator{}, false
}

func containsPhase(phases []string, phase string) bool {
	for _, candidate := range phases {
		if candidate == phase {
			return true
		}
	}
	return false
}

func canonicalPhases(phases []string) []string {
	list := append([]string{}, phases...)
	sort.Strings(list)
	return list
}

func fingerprintOf(graph Graph) string {
	operators := append([]Operator{}, graph.Operators...)
	sort.Slice(operators, func(i, j int) bool {
		if operators[i].Name != operators[j].Name {
			return operators[i].Name < operators[j].Name
		}
		return operators[i].Version < operators[j].Version
	})
	for i := range operators {
		operators[i].Phases = canonicalPhases(operators[i].Phases)
	}
	steps := append([]Step{}, graph.Steps...)
	sort.Slice(steps, func(i, j int) bool {
		return steps[i].ID < steps[j].ID
	})
	canonical := struct {
		Phase     string     `json:"phase"`
		Operators []Operator `json:"operators"`
		Steps     []Step     `json:"steps"`
	}{strings.TrimSpace(graph.Phase), operators, steps}
	encoded, err := json.Marshal(canonical)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// hasCycle reports a dependency cycle over the step subgraph using Kahn's
// algorithm with a deterministic order.
func hasCycle(steps []Step) bool {
	ids := map[string]bool{}
	for _, step := range steps {
		ids[step.ID] = true
	}
	indegree := map[string]int{}
	edges := map[string][]string{}
	for _, step := range steps {
		for _, input := range step.Inputs {
			if ids[input] {
				edges[input] = append(edges[input], step.ID)
				indegree[step.ID]++
			}
		}
	}
	var ready []string
	for id := range ids {
		if indegree[id] == 0 {
			ready = append(ready, id)
		}
	}
	sort.Strings(ready)
	seen := 0
	for len(ready) > 0 {
		id := ready[0]
		ready = ready[1:]
		seen++
		for _, follow := range edges[id] {
			indegree[follow]--
			if indegree[follow] == 0 {
				ready = append(ready, follow)
			}
		}
		sort.Strings(ready)
	}
	return seen != len(ids)
}

// sameContract compares the meaning-bearing fields of two operators.
func sameContract(left, right Operator) (bool, string) {
	if left.Version != right.Version {
		return false, "version"
	}
	if !equalStrings(canonicalPhases(left.Effects), canonicalPhases(right.Effects)) {
		return false, "effects"
	}
	if !equalStrings(canonicalPhases(left.Phases), canonicalPhases(right.Phases)) {
		return false, "phases"
	}
	if left.InputType != right.InputType {
		return false, "inputType"
	}
	if left.OutputType != right.OutputType {
		return false, "outputType"
	}
	return true, ""
}

func equalStrings(left, right []string) bool {
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

// Decide checks one request in "graph" or "compare" mode. It never executes
// an operator and never writes anything.
func Decide(in Input) Report {
	switch strings.TrimSpace(in.Mode) {
	case "", "graph":
		return decideGraph(in)
	case "compare":
		return decideCompare(in)
	default:
		return reject(CodeInvalidRequest, fmt.Sprintf("未知の mode %q。graph か compare を指定。", in.Mode))
	}
}

func decideCompare(in Input) Report {
	if in.Graph == nil || in.CompareTo == nil {
		return reject(CodeInvalidRequest, "compare では graph と compareTo の双方が必要。")
	}
	first := operatorMap(in.Graph.Operators)
	second := operatorMap(in.CompareTo.Operators)
	for _, name := range sortedKeys(first) {
		other, ok := second[name]
		if !ok {
			return reject(CodeMeaningChanged, fmt.Sprintf("段階の片側にだけ Operator %q がある。", name))
		}
		same, field := sameContract(first[name], other)
		if !same {
			if field == "version" {
				return reject(CodeVersionChanged,
					fmt.Sprintf("Operator %q の版が %q != %q。同じ契約として扱わない。", name, first[name].Version, other.Version))
			}
			return reject(CodeMeaningChanged,
				fmt.Sprintf("Operator %q の %s が二つの段階で異なる。同じ意味では共有しない。", name, field))
		}
	}
	for _, name := range sortedKeys(second) {
		if _, ok := first[name]; !ok {
			return reject(CodeMeaningChanged, fmt.Sprintf("段階の片側にだけ Operator %q がある。", name))
		}
	}
	return Report{Code: CodeMeaningShared,
		Message: "同一の Operator 契約が両段階で一致する。同一fixtureでの意味共有の前提を保持する。"}
}

func operatorMap(operators []Operator) map[string]Operator {
	set := make(map[string]Operator, len(operators))
	for _, operator := range operators {
		set[operator.Name] = operator
	}
	return set
}

func sortedKeys(set map[string]Operator) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func decideGraph(in Input) Report {
	if in.Graph == nil {
		return reject(CodeInvalidRequest, "graph が未指定。")
	}
	phase := strings.TrimSpace(in.Graph.Phase)
	switch phase {
	case PhaseAuthoring, PhaseValidation, PhaseGeneration, PhaseRuntime:
	default:
		return reject(CodeInvalidRequest, fmt.Sprintf("未知の phase %q。", in.Graph.Phase))
	}
	if len(in.Graph.Operators) == 0 && len(in.Graph.Steps) == 0 {
		return reject(CodeInvalidRequest, "operators と steps がともに空。")
	}

	seen := map[string]bool{}
	for _, operator := range in.Graph.Operators {
		name := strings.TrimSpace(operator.Name)
		if name == "" {
			return reject(CodeInvalidRequest, "Operator 名が未指定。")
		}
		if seen[name] {
			return reject(CodeDuplicateOperator, fmt.Sprintf("Operator %q が重複する。同名の別実装を許さない。", name))
		}
		seen[name] = true
	}

	produced := map[string]Step{}
	for _, step := range in.Graph.Steps {
		id := strings.TrimSpace(step.ID)
		if id == "" {
			return reject(CodeInvalidRequest, "step の id が未指定。")
		}
		if _, duplicated := produced[id]; duplicated {
			return reject(CodeInvalidRequest, fmt.Sprintf("step id %q が重複する。", id))
		}
		produced[id] = step
	}

	// Phase membership and effect discipline.
	for _, step := range in.Graph.Steps {
		operator, ok := findOperator(step.Operator, in.Graph.Operators)
		if !ok {
			return reject(CodeOperatorUnresolved, fmt.Sprintf("step %q が未宣言の Operator %q を参照。", step.ID, step.Operator))
		}
		if !containsPhase(operator.Phases, phase) {
			return reject(CodePhaseNotAllowed,
				fmt.Sprintf("Operator %q は phase %q を許可しない。", operator.Name, phase))
		}
		if phase == PhaseRuntime && len(operator.Effects) != 0 {
			return reject(CodeEffectInRuntime,
				fmt.Sprintf("Operator %q の Effect %v は実行時に無条件に漏れない。", operator.Name, operator.Effects))
		}
	}

	// Typed contract between a consuming operator and a producing step.
	outputType := map[string]string{}
	for id, step := range produced {
		if operator, ok := findOperator(step.Operator, in.Graph.Operators); ok {
			outputType[id] = operator.OutputType
		}
	}
	for _, step := range in.Graph.Steps {
		operator, ok := findOperator(step.Operator, in.Graph.Operators)
		if !ok || strings.TrimSpace(operator.InputType) == "" {
			continue
		}
		for _, input := range step.Inputs {
			producerType, isStep := outputType[input]
			if !isStep {
				continue // external source input: the type is not checked here
			}
			if producerType != "" && producerType != operator.InputType {
				return reject(CodeContractMismatch,
					fmt.Sprintf("step %q は入力 %q の型 %q を要求するが、出力は %q。", step.ID, input, operator.InputType, producerType))
			}
		}
	}

	if hasCycle(in.Graph.Steps) {
		return reject(CodeTransformCycle, "生成stepの依存に循環がある。決定的な変換にならない。")
	}

	// Trace and unresolved propagation.
	unresolved := map[string]bool{}
	for _, step := range in.Graph.Steps {
		if step.Unresolved {
			unresolved[step.ID] = true
		}
	}
	for _, id := range sortedStepIDs(produced) {
		if !unresolved[id] {
			continue
		}
		for _, downstream := range downstreamOf(id, in.Graph.Steps, produced) {
			unresolved[downstream] = true
		}
	}
	trace := make([]string, 0, len(in.Graph.Steps))
	for _, id := range sortedStepIDs(produced) {
		step := produced[id]
		line := id + " <- " + step.Operator
		if step.Origin != "" {
			line += " @ " + step.Origin
		}
		if unresolved[id] {
			line += " (unresolved)"
		}
		trace = append(trace, line)
	}
	sort.Strings(trace)

	unresolvedList := make([]string, 0, len(unresolved))
	for id := range unresolved {
		unresolvedList = append(unresolvedList, id)
	}
	sort.Strings(unresolvedList)

	fingerprint := fingerprintOf(*in.Graph)
	if strings.TrimSpace(in.CurrentRuleVersion) != strings.TrimSpace(in.IncomingRuleVersion) {
		return Report{Code: CodeRecomputeRequired, Fingerprint: fingerprint,
			Trace: trace, Unresolved: unresolvedList,
			Message: fmt.Sprintf("規則版が %s から %s へ変わる。影響のない生成物を黙って維持せず、依存に応じて再計算する。", in.CurrentRuleVersion, in.IncomingRuleVersion)}
	}

	message := fmt.Sprintf("graph の構造検査に合格。fingerprint は構造と契約の固定。抵触は受理入口 #267 へ。規則版 %s。", in.CurrentRuleVersion)
	if len(unresolvedList) != 0 {
		message = fmt.Sprintf("graph の構造検査に合格。未確定要件 %d 件は解決せず保持する。規則版 %s。", len(unresolvedList), in.CurrentRuleVersion)
	}
	return Report{Code: CodeCheckPassed, Clean: true, Fingerprint: fingerprint,
		Trace: trace, Unresolved: unresolvedList, Message: message}
}

func sortedStepIDs(steps map[string]Step) []string {
	ids := make([]string, 0, len(steps))
	for id := range steps {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// downstreamOf lists every step that transitively depends on the given id.
func downstreamOf(id string, steps []Step, produced map[string]Step) []string {
	reverse := map[string][]string{} // input id -> step ids consuming it
	for _, step := range steps {
		for _, input := range step.Inputs {
			if _, isStep := produced[input]; isStep {
				reverse[input] = append(reverse[input], step.ID)
			}
		}
	}
	var queue []string
	queue = append(queue, reverse[id]...)
	result := map[string]bool{}
	for len(queue) > 0 {
		next := queue[0]
		queue = queue[1:]
		if result[next] {
			continue
		}
		result[next] = true
		queue = append(queue, reverse[next]...)
	}
	ordered := make([]string, 0, len(result))
	for key := range result {
		ordered = append(ordered, key)
	}
	sort.Strings(ordered)
	return ordered
}
