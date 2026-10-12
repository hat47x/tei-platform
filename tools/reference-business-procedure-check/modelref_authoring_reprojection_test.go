package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/index"
	"tei/internal/modelref"
	"tei/internal/modelref/derivedbridge"
	"tei/internal/sourceindex"
	"tei/internal/subject"
	"tei/internal/workspace"
)

// businessAuthoringGraphForTest composes an explicitly NON-Canonical Business
// authoring fixture with the real Workspace's typed Data/Function projection.
// It is test scaffolding, not a new accepted Business model or Source writer.
func businessAuthoringGraphForTest(loaded workspace.Result, f fixture) (modelref.Graph, []modelref.Rule, error) {
	const scope = "order-entry-reference"
	projected := derivedbridge.FromIndex(index.Build(loaded.World), scope)
	graph, rules := projected.Graph, projected.Rules
	if f.Canonical || f.Procedure.IdentityStatus != referenceLocalStable {
		return modelref.Graph{}, nil, fmt.Errorf("not a non-Canonical reference authoring fixture")
	}
	key := func(kind, id string) modelref.Key {
		return modelref.Key{Scope: scope, Subject: subject.Subject{Kind: kind, ID: id}}
	}
	nodes := make(map[modelref.Key]bool, len(graph.Nodes))
	for _, node := range graph.Nodes {
		nodes[node.Key] = true
	}
	procedureKey := key("business-procedure-authoring", f.Procedure.ID)
	if nodes[procedureKey] {
		return modelref.Graph{}, nil, fmt.Errorf("duplicate authoring procedure %s", f.Procedure.ID)
	}
	nodes[procedureKey] = true
	graph.Nodes = append(graph.Nodes, modelref.Node{Key: procedureKey, DisplayName: f.Procedure.Purpose})
	rules = append(rules,
		modelref.Rule{Relation: "declares-step", FromKind: "business-procedure-authoring", ToKind: "business-step-authoring"},
		modelref.Rule{Relation: "uses-pattern", FromKind: "business-step-authoring", ToKind: "application-pattern"},
		modelref.Rule{Relation: "checks-validation", FromKind: "business-step-authoring", ToKind: "semantic-validation"},
	)
	// Only pairs confirmed by the existing checker can be bridged into
	// modelref. A declared, unrecognized Pattern must fail; no guessing.
	checked := evaluate(loaded.World, f)
	confirmed := make(map[string]bool)
	for _, row := range checked.BusinessActionPatternMeaningRows {
		confirmed[row.StepID+"\x00"+row.Pattern] = true
	}
	seenSteps := make(map[string]bool)
	for _, step := range f.Steps {
		if seenSteps[step.ID] {
			return modelref.Graph{}, nil, fmt.Errorf("duplicate step %q", step.ID)
		}
		seenSteps[step.ID] = true
		stepKey := key("business-step-authoring", f.Procedure.ID+"/"+step.ID)
		nodes[stepKey] = true
		graph.Nodes = append(graph.Nodes, modelref.Node{Key: stepKey, DisplayName: step.BusinessAction})
		graph.Edges = append(graph.Edges, modelref.Edge{
			From: procedureKey, To: modelref.Ref{Scope: scope, Subject: stepKey.Subject}, Relation: "declares-step",
		})
		for _, patternID := range step.ApplicationPatterns {
			patternKey := key("application-pattern", patternID)
			if !nodes[patternKey] || !confirmed[step.ID+"\x00"+patternID] {
				return modelref.Graph{}, nil, fmt.Errorf("unresolved Business-to-Function reference %s/%s", step.ID, patternID)
			}
			graph.Edges = append(graph.Edges, modelref.Edge{
				From: stepKey, To: modelref.Ref{Scope: scope, Subject: patternKey.Subject}, Relation: "uses-pattern",
			})
		}
		for _, validationID := range step.Validations {
			target := key("semantic-validation", validationID)
			if !nodes[target] {
				return modelref.Graph{}, nil, fmt.Errorf("unresolved Business-to-Data validation %s/%s", step.ID, validationID)
			}
			graph.Edges = append(graph.Edges, modelref.Edge{
				From: stepKey, To: modelref.Ref{Scope: scope, Subject: target.Subject}, Relation: "checks-validation",
			})
		}
	}
	if findings := modelref.Validate(graph, rules); len(findings) != 0 {
		return modelref.Graph{}, nil, fmt.Errorf("typed authoring projection is invalid: %+v", findings)
	}
	return graph, rules, nil
}

// This checks the actual Reference Workspace + registered Procedure Authoring
// source before and after a Business-only edit. The Procedure is accepted as
// authoring *source*, not as a Canonical Business Procedure in ir.World.
func TestIssue1120ThreeViewReferencesSurviveBusinessAuthoringEdit(t *testing.T) {
	root := prepareProcedureAuthoringWorkspace(t, repoPath())
	relative := "procedures/order-handling.json"
	beforeLoaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	before, err := acceptProcedureSource(root, relative, &beforeLoaded, "")
	if err != nil {
		t.Fatal(err)
	}
	beforeGraph, rules, err := businessAuthoringGraphForTest(beforeLoaded, before.Fixture)
	if err != nil {
		t.Fatal(err)
	}
	beforeIndex, err := workspace.BuildSourceIndex(root, beforeLoaded)
	if err != nil {
		t.Fatal(err)
	}
	beforeRefs := beforeIndex.ReferencesFor(before.Subject)
	if len(beforeRefs) != 1 || beforeRefs[0].Role != sourceindex.RoleAuthoring {
		t.Fatalf("Business authoring source was not indexed: %+v", beforeRefs)
	}

	// Business-only wording change: the source content changes but stable
	// Procedure/Step, Application Pattern and Data identities must not.
	edited := before.Fixture
	found := false
	for i := range edited.Steps {
		if edited.Steps[i].ID == "BP-02" {
			edited.Steps[i].BusinessAction = "confirm the selected order's customer before continuing"
			found = true
		}
	}
	if !found {
		t.Fatal("BP-02 not present in actual reference procedure")
	}
	bytes, err := json.MarshalIndent(edited, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(relative)), append(bytes, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	afterLoaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := acceptProcedureSource(root, relative, &afterLoaded, before.Artifact.Fingerprint); err == nil {
		t.Fatal("stale Business authoring source revision was silently accepted")
	}
	afterBytes, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
	if err != nil {
		t.Fatal(err)
	}
	current, err := sourceindex.NewArtifact(relative, afterBytes)
	if err != nil {
		t.Fatal(err)
	}
	after, err := acceptProcedureSource(root, relative, &afterLoaded, current.Fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	if before.Subject != after.Subject || before.Artifact.Fingerprint == after.Artifact.Fingerprint {
		t.Fatal("Business source identity changed, or wording revision was not detected")
	}
	if !reflect.DeepEqual(beforeLoaded.World, afterLoaded.World) {
		t.Fatal("non-Canonical Business edit changed canonical Data/Function World")
	}
	afterGraph, newRules, err := businessAuthoringGraphForTest(afterLoaded, after.Fixture)
	if err != nil {
		t.Fatal(err)
	}
	beforeRules := modelref.RuleFingerprint(rules)
	if beforeRules != modelref.RuleFingerprint(newRules) {
		t.Fatal("Business-only wording changed extension dependency rules")
	}
	review := modelref.ReviewImpactDiff(beforeGraph, afterGraph, newRules)
	if !review.GraphValid {
		t.Fatalf("typed graph changed invalidly: %+v", review.Diagnostics)
	}
	const scope = "order-entry-reference"
	key := func(kind, id string) modelref.Key {
		return modelref.Key{Scope: scope, Subject: subject.Subject{Kind: kind, ID: id}}
	}
	businessStep := key("business-step-authoring", "order-entry.order-handling/BP-02")
	changedStep, changedOther := false, false
	for _, change := range review.Changes {
		if change.Key == businessStep && change.Kind == "display-renamed" {
			changedStep = true
		} else {
			changedOther = true
		}
	}
	if !changedStep || changedOther {
		t.Fatalf("Business wording mutated typed identities or dependencies: %+v", review.Changes)
	}
	// The actual Derived Index contains the Application Pattern -> Relation
	// -> Data path; the *explicit* Procedure fixture links its BP-02 step to
	// that same Pattern. No application display label or source path is used
	// as a stand-in for the typed reference.
	data := key("semantic-data", "customer.customer")
	affected := modelref.ReviewImpact(beforeGraph,
		[]modelref.Change{{Key: data, Kind: "version-changed"}}, rules)
	if !affected.GraphValid {
		t.Fatalf("real Data-to-Business route is unresolved: %+v", affected.Diagnostics)
	}
	foundStep := false
	for _, consumer := range affected.Affected {
		if consumer.Consumer == businessStep {
			foundStep = true
		}
	}
	if !foundStep {
		t.Fatalf("real Data-to-Application-to-Business candidate missing: %+v", affected.Affected)
	}
	afterIndex, err := workspace.BuildSourceIndex(root, afterLoaded)
	if err != nil {
		t.Fatal(err)
	}
	if refs := afterIndex.ReferencesFor(after.Subject); !reflect.DeepEqual(beforeRefs, refs) {
		t.Fatalf("unchanged Business Source location drifted: before=%+v after=%+v", beforeRefs, refs)
	}
}

func TestIssue1120BusinessAuthoringDoesNotInventMissingPatternReference(t *testing.T) {
	root := prepareProcedureAuthoringWorkspace(t, repoPath())
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	accepted, err := acceptProcedureSource(root, "procedures/order-handling.json", &loaded, "")
	if err != nil {
		t.Fatal(err)
	}
	bad := accepted.Fixture
	for i := range bad.Steps {
		if bad.Steps[i].ID == "BP-02" {
			bad.Steps[i].ApplicationPatterns = []string{"order_entry.nonexistent_pattern"}
		}
	}
	if _, _, err := businessAuthoringGraphForTest(loaded, bad); err == nil {
		t.Fatal("unknown Application Pattern was fabricated into the typed graph")
	}
}
