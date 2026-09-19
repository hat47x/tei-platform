package main

import (
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

type canonicalBusinessHandoffProbe struct {
	From      canonicalBusinessHandoffEndpointProbe `json:"from"`
	To        canonicalBusinessHandoffEndpointProbe `json:"to"`
	Condition string                                `json:"condition"`
}

type canonicalBusinessHandoffEndpointProbe struct {
	Procedure string `json:"procedure"`
	Step      string `json:"step"`
}

func TestCanonicalBusinessHandoffProbeRemainsThinSemanticRelation(t *testing.T) {
	handoffType := reflect.TypeOf(canonicalBusinessHandoffProbe{})
	want := []string{"From", "To", "Condition"}
	if got := fieldNames(handoffType); !reflect.DeepEqual(got, want) {
		t.Fatalf("handoff relationへruntime orchestration detailが侵入しました: got=%v want=%v", got, want)
	}
	endpointType := reflect.TypeOf(canonicalBusinessHandoffEndpointProbe{})
	endpointWant := []string{"Procedure", "Step"}
	if got := fieldNames(endpointType); !reflect.DeepEqual(got, endpointWant) {
		t.Fatalf("handoff endpointがsource locator / runtime identityへ膨張しました: got=%v want=%v", got, endpointWant)
	}
}

func TestCanonicalBusinessHandoffProbeDerivesFromReferenceSetAndResolvesEndpoints(t *testing.T) {
	root := filepath.Join(repoPath(), "reference", "business-system-design")
	set, err := readReferenceSet(filepath.Join(root, "order-entry-procedure-reference-set.json"))
	if err != nil {
		t.Fatal(err)
	}
	fixtures, err := resolveReferenceSet(root, set)
	if err != nil {
		t.Fatal(err)
	}
	handoffs, err := deriveCanonicalBusinessHandoffProbes(set, fixtures)
	if err != nil {
		t.Fatal(err)
	}
	if len(handoffs) != 1 {
		t.Fatalf("handoff count=%d; want 1", len(handoffs))
	}
	got := handoffs[0]
	want := canonicalBusinessHandoffProbe{
		From: canonicalBusinessHandoffEndpointProbe{Procedure: "order-entry.order-handling", Step: "BP-09"},
		To: canonicalBusinessHandoffEndpointProbe{Procedure: "order-entry.follow-up", Step: "FU-00"},
		Condition: "follow-up is required after the earlier order interaction",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected canonical handoff: got=%+v want=%+v", got, want)
	}
}

func TestCanonicalBusinessHandoffProbeIsIndependentFromArtifactLocator(t *testing.T) {
	root := filepath.Join(repoPath(), "reference", "business-system-design")
	set, err := readReferenceSet(filepath.Join(root, "order-entry-procedure-reference-set.json"))
	if err != nil {
		t.Fatal(err)
	}
	fixtures, err := resolveReferenceSet(root, set)
	if err != nil {
		t.Fatal(err)
	}
	before, err := deriveCanonicalBusinessHandoffProbes(set, fixtures)
	if err != nil {
		t.Fatal(err)
	}

	moved := set
	moved.Procedures = append([]referenceSetProcedure(nil), set.Procedures...)
	for i := range moved.Procedures {
		moved.Procedures[i].Artifact = "moved/" + moved.Procedures[i].Artifact
	}
	after, err := deriveCanonicalBusinessHandoffProbes(moved, fixtures)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("artifact locator変更でBusiness handoff meaningが変化しました: before=%+v after=%+v", before, after)
	}
}

func TestCanonicalBusinessHandoffProbeSeesConditionChangeWithoutChangingProcedureBodies(t *testing.T) {
	root := filepath.Join(repoPath(), "reference", "business-system-design")
	set, err := readReferenceSet(filepath.Join(root, "order-entry-procedure-reference-set.json"))
	if err != nil {
		t.Fatal(err)
	}
	fixtures, err := resolveReferenceSet(root, set)
	if err != nil {
		t.Fatal(err)
	}
	before, err := deriveCanonicalBusinessHandoffProbes(set, fixtures)
	if err != nil {
		t.Fatal(err)
	}

	changed := set
	changed.Handoffs = append([]referenceSetHandoff(nil), set.Handoffs...)
	changed.Handoffs[0].Condition = "follow-up is required because unresolved surrounding work remains"
	after, err := deriveCanonicalBusinessHandoffProbes(changed, fixtures)
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(before, after) {
		t.Fatal("Business handoff condition changeをsemantic relation差として観測できません")
	}

	primaryBefore, err := deriveCanonicalBusinessProcedureMeaningProbe(fixtures["order-entry.order-handling"])
	if err != nil {
		t.Fatal(err)
	}
	primaryAfter, err := deriveCanonicalBusinessProcedureMeaningProbe(fixtures["order-entry.order-handling"])
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(primaryBefore, primaryAfter) {
		t.Fatal("handoff condition changeだけでProcedure body meaningまで変化しました")
	}
}

func TestCanonicalBusinessHandoffProbeKeepsSameEndpointsWithDifferentConditionsDistinct(t *testing.T) {
	root := filepath.Join(repoPath(), "reference", "business-system-design")
	set, err := readReferenceSet(filepath.Join(root, "order-entry-procedure-reference-set.json"))
	if err != nil {
		t.Fatal(err)
	}
	fixtures, err := resolveReferenceSet(root, set)
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Handoffs) == 0 {
		t.Fatal("test setup: Handoffがありません")
	}

	second := set.Handoffs[0]
	second.Condition = "follow-up remains required because unresolved external work continues"
	expanded := set
	expanded.Handoffs = append(append([]referenceSetHandoff(nil), set.Handoffs...), second)

	handoffs, err := deriveCanonicalBusinessHandoffProbes(expanded, fixtures)
	if err != nil {
		t.Fatalf("same endpoints / different conditionをendpoint-only identityでrejectしました: %v", err)
	}
	if len(handoffs) != 2 {
		t.Fatalf("handoff count=%d; want 2", len(handoffs))
	}

	conditions := map[string]int{}
	for _, handoff := range handoffs {
		conditions[handoff.Condition]++
	}
	for _, condition := range []string{set.Handoffs[0].Condition, second.Condition} {
		if conditions[condition] != 1 {
			t.Fatalf("distinct H1/R3 condition meaningをexactly once保持できません: condition=%q got=%v", condition, conditions)
		}
	}
}

func TestCanonicalBusinessHandoffProbeRejectsMissingEndpoint(t *testing.T) {
	root := filepath.Join(repoPath(), "reference", "business-system-design")
	set, err := readReferenceSet(filepath.Join(root, "order-entry-procedure-reference-set.json"))
	if err != nil {
		t.Fatal(err)
	}
	fixtures, err := resolveReferenceSet(root, set)
	if err != nil {
		t.Fatal(err)
	}
	broken := set
	broken.Handoffs = append([]referenceSetHandoff(nil), set.Handoffs...)
	broken.Handoffs[0].To.Step = "FU-MISSING"
	if _, err := deriveCanonicalBusinessHandoffProbes(broken, fixtures); err == nil {
		t.Fatal("存在しないProcedure stepへのhandoffを受理しました")
	}
}

func TestCanonicalBusinessHandoffProbeRejectsDuplicateRelation(t *testing.T) {
	root := filepath.Join(repoPath(), "reference", "business-system-design")
	set, err := readReferenceSet(filepath.Join(root, "order-entry-procedure-reference-set.json"))
	if err != nil {
		t.Fatal(err)
	}
	fixtures, err := resolveReferenceSet(root, set)
	if err != nil {
		t.Fatal(err)
	}
	duplicated := set
	duplicated.Handoffs = append(append([]referenceSetHandoff(nil), set.Handoffs...), set.Handoffs[0])
	if _, err := deriveCanonicalBusinessHandoffProbes(duplicated, fixtures); err == nil {
		t.Fatal("同一Business handoff relationを重複受理しました")
	}
}

func deriveCanonicalBusinessHandoffProbes(set referenceSet, fixtures map[string]fixture) ([]canonicalBusinessHandoffProbe, error) {
	if err := validateReferenceSetHandoffs(set, fixtures); err != nil {
		return nil, err
	}
	result := make([]canonicalBusinessHandoffProbe, 0, len(set.Handoffs))
	seen := map[string]struct{}{}
	for _, item := range set.Handoffs {
		probe := canonicalBusinessHandoffProbe{
			From: canonicalBusinessHandoffEndpointProbe{Procedure: item.From.Procedure, Step: item.From.Step},
			To: canonicalBusinessHandoffEndpointProbe{Procedure: item.To.Procedure, Step: item.To.Step},
			Condition: item.Condition,
		}
		key := probe.From.Procedure + "/" + probe.From.Step + "->" + probe.To.Procedure + "/" + probe.To.Step + ":" + probe.Condition
		if _, duplicate := seen[key]; duplicate {
			return nil, errProcedureMeaning("duplicate Business handoff relation " + key)
		}
		seen[key] = struct{}{}
		result = append(result, probe)
	}
	sort.Slice(result, func(i, j int) bool {
		left := result[i].From.Procedure + "/" + result[i].From.Step + "->" + result[i].To.Procedure + "/" + result[i].To.Step + ":" + result[i].Condition
		right := result[j].From.Procedure + "/" + result[j].From.Step + "->" + result[j].To.Procedure + "/" + result[j].To.Step + ":" + result[j].Condition
		return left < right
	})
	return result, nil
}
