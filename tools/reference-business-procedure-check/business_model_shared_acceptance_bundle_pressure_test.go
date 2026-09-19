package main

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/workspace"
)

type businessAuthoringBundleProbe struct {
	Procedures []inlineO1FixtureProbe       `json:"procedures"`
	Handoffs   standaloneHandoffSourceProbe `json:"handoffs"`
}

func TestBusinessAuthoringBundleNormalizesMultipleSourceKindsIntoOneSemanticRevision(t *testing.T) {
	root := repoPath()
	bundle := currentBusinessAuthoringBundleProbe(t, root)
	model, err := deriveBusinessAuthoringBundleProbe(bundle)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	if err := validateD4O1NormalizedBusinessModelProbe(loaded.World, model); err != nil {
		t.Fatalf("multi-source Business bundleをcross-view validateできません: %v", err)
	}

	setRoot := filepath.Join(root, "reference", "business-system-design")
	set, fixtures := currentReferenceSetAndFixturesD4O1Probe(t, setRoot)
	current, err := deriveD4O1NormalizedBusinessModelProbe(set, fixtures, currentProtectedTargetsD4O1Probe())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(model, current) {
		t.Fatalf("inline Procedure + standalone Handoff bundleがcurrent normalized meaningと一致しません: bundle=%+v current=%+v", model, current)
	}
}

func TestBusinessAuthoringBundleSemanticRevisionChangesForAnyAcceptedBusinessSourceChange(t *testing.T) {
	root := repoPath()
	baselineBundle := currentBusinessAuthoringBundleProbe(t, root)
	baseline, err := deriveBusinessAuthoringBundleProbe(baselineBundle)
	if err != nil {
		t.Fatal(err)
	}
	baselineFingerprint := semanticFingerprintD4O1Probe(t, baseline)

	t.Run("procedure wording", func(t *testing.T) {
		bundle := currentBusinessAuthoringBundleProbe(t, root)
		changed := false
		for p := range bundle.Procedures {
			if bundle.Procedures[p].Procedure.ID != "order-entry.order-handling" {
				continue
			}
			for s := range bundle.Procedures[p].Steps {
				if bundle.Procedures[p].Steps[s].ID == "BP-04" {
					bundle.Procedures[p].Steps[s].BusinessAction = "decide whether unresolved surrounding business circumstances permit the order to proceed"
					changed = true
				}
			}
		}
		if !changed {
			t.Fatal("test setup: BP-04がありません")
		}
		model, err := deriveBusinessAuthoringBundleProbe(bundle)
		if err != nil {
			t.Fatal(err)
		}
		if semanticFingerprintD4O1Probe(t, model) == baselineFingerprint {
			t.Fatal("Procedure wording source changeをBusiness semantic revisionが見逃しました")
		}
	})

	t.Run("protected target", func(t *testing.T) {
		bundle := currentBusinessAuthoringBundleProbe(t, root)
		changed := false
		for p := range bundle.Procedures {
			if bundle.Procedures[p].Procedure.ID != "order-entry.order-handling" {
				continue
			}
			for s := range bundle.Procedures[p].Steps {
				if bundle.Procedures[p].Steps[s].ID == "BP-06" {
					bundle.Procedures[p].Steps[s].ProtectedApplicationPattern = "order_entry.order_record"
					changed = true
				}
			}
		}
		if !changed {
			t.Fatal("test setup: BP-06がありません")
		}
		model, err := deriveBusinessAuthoringBundleProbe(bundle)
		if err != nil {
			t.Fatal(err)
		}
		if semanticFingerprintD4O1Probe(t, model) == baselineFingerprint {
			t.Fatal("protected-target source changeをBusiness semantic revisionが見逃しました")
		}
	})

	t.Run("handoff condition", func(t *testing.T) {
		bundle := currentBusinessAuthoringBundleProbe(t, root)
		if len(bundle.Handoffs.Handoffs) == 0 {
			t.Fatal("test setup: Handoffがありません")
		}
		bundle.Handoffs.Handoffs = append([]referenceSetHandoff(nil), bundle.Handoffs.Handoffs...)
		bundle.Handoffs.Handoffs[0].Condition = "follow-up remains required because unresolved external work continues"
		model, err := deriveBusinessAuthoringBundleProbe(bundle)
		if err != nil {
			t.Fatal(err)
		}
		if semanticFingerprintD4O1Probe(t, model) == baselineFingerprint {
			t.Fatal("Handoff source changeをBusiness semantic revisionが見逃しました")
		}
	})
}

func TestBusinessAuthoringBundleRequiresCoordinatedStepAndHandoffRename(t *testing.T) {
	root := repoPath()
	baseline, err := deriveBusinessAuthoringBundleProbe(currentBusinessAuthoringBundleProbe(t, root))
	if err != nil {
		t.Fatal(err)
	}
	baselineFingerprint := semanticFingerprintD4O1Probe(t, baseline)

	t.Run("procedure-only rename fails closed", func(t *testing.T) {
		bundle := currentBusinessAuthoringBundleProbe(t, root)
		if !renameInlineO1StepProbe(&bundle, "order-entry.order-handling", "BP-09", "BP-09-renamed") {
			t.Fatal("test setup: BP-09がありません")
		}
		if _, err := deriveBusinessAuthoringBundleProbe(bundle); err == nil {
			t.Fatal("Procedure Stepだけのrenameでstale Handoff endpointを黙って受理しました")
		}
	})

	t.Run("handoff-only rename fails closed", func(t *testing.T) {
		bundle := currentBusinessAuthoringBundleProbe(t, root)
		if len(bundle.Handoffs.Handoffs) == 0 {
			t.Fatal("test setup: Handoffがありません")
		}
		bundle.Handoffs.Handoffs = append([]referenceSetHandoff(nil), bundle.Handoffs.Handoffs...)
		bundle.Handoffs.Handoffs[0].From.Step = "BP-09-renamed"
		if _, err := deriveBusinessAuthoringBundleProbe(bundle); err == nil {
			t.Fatal("Handoff endpointだけのrenameでmissing Procedure Stepを黙って受理しました")
		}
	})

	t.Run("coordinated rename forms one new semantic revision", func(t *testing.T) {
		bundle := currentBusinessAuthoringBundleProbe(t, root)
		if !renameInlineO1StepProbe(&bundle, "order-entry.order-handling", "BP-09", "BP-09-renamed") {
			t.Fatal("test setup: BP-09がありません")
		}
		if len(bundle.Handoffs.Handoffs) == 0 {
			t.Fatal("test setup: Handoffがありません")
		}
		bundle.Handoffs.Handoffs = append([]referenceSetHandoff(nil), bundle.Handoffs.Handoffs...)
		bundle.Handoffs.Handoffs[0].From.Step = "BP-09-renamed"
		model, err := deriveBusinessAuthoringBundleProbe(bundle)
		if err != nil {
			t.Fatalf("協調renameを一つのBusiness change-setとしてnormalizeできません: %v", err)
		}
		if semanticFingerprintD4O1Probe(t, model) == baselineFingerprint {
			t.Fatal("協調Step/Handoff renameを新しいBusiness semantic revisionとして観測できません")
		}
	})
}

func TestBusinessAuthoringBundleNormalizesProcedureSourceOrdering(t *testing.T) {
	root := repoPath()
	beforeBundle := currentBusinessAuthoringBundleProbe(t, root)
	before, err := deriveBusinessAuthoringBundleProbe(beforeBundle)
	if err != nil {
		t.Fatal(err)
	}

	afterBundle := currentBusinessAuthoringBundleProbe(t, root)
	for left, right := 0, len(afterBundle.Procedures)-1; left < right; left, right = left+1, right-1 {
		afterBundle.Procedures[left], afterBundle.Procedures[right] = afterBundle.Procedures[right], afterBundle.Procedures[left]
	}
	after, err := deriveBusinessAuthoringBundleProbe(afterBundle)
	if err != nil {
		t.Fatal(err)
	}
	if semanticFingerprintD4O1Probe(t, before) != semanticFingerprintD4O1Probe(t, after) {
		t.Fatal("Procedure source orderingだけでBusiness semantic revisionが変化しました")
	}
}

func TestBusinessAuthoringBundleDoesNotCarrySourceLocatorOrRuntimeFields(t *testing.T) {
	bundleType := reflect.TypeOf(businessAuthoringBundleProbe{})
	for _, forbidden := range []string{"Path", "Artifact", "Locator", "Revision", "ETag", "Runtime", "Workflow"} {
		if _, found := bundleType.FieldByName(forbidden); found {
			t.Fatalf("Business semantic bundleへsource/runtime field %qを混入しました", forbidden)
		}
	}
}

func currentBusinessAuthoringBundleProbe(t *testing.T, root string) businessAuthoringBundleProbe {
	t.Helper()
	primary, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	followUp, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-follow-up-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	set, err := readReferenceSet(filepath.Join(root, "reference", "business-system-design", "order-entry-procedure-reference-set.json"))
	if err != nil {
		t.Fatal(err)
	}

	return businessAuthoringBundleProbe{
		Procedures: []inlineO1FixtureProbe{
			inlineO1CandidateFromFixtureProbe(primary, map[string]string{"BP-06": "order_entry.submit_order"}),
			inlineO1CandidateFromFixtureProbe(followUp, nil),
		},
		Handoffs: standaloneHandoffSourceProbe{
			SchemaVersion: standaloneHandoffSchemaVersion,
			ArtifactRole:  standaloneHandoffArtifactRole,
			Canonical:     false,
			Handoffs:      append([]referenceSetHandoff(nil), set.Handoffs...),
		},
	}
}

func deriveBusinessAuthoringBundleProbe(bundle businessAuthoringBundleProbe) (d4O1NormalizedBusinessModelProbe, error) {
	fixtures := make(map[string]fixture, len(bundle.Procedures))
	targets := map[string][]string{}
	for _, source := range bundle.Procedures {
		base, perStep, err := inlineO1CandidateToFixtureProbe(source)
		if err != nil {
			return d4O1NormalizedBusinessModelProbe{}, err
		}
		if _, duplicate := fixtures[base.Procedure.ID]; duplicate {
			return d4O1NormalizedBusinessModelProbe{}, fmt.Errorf("duplicate Business Procedure source %q", base.Procedure.ID)
		}
		fixtures[base.Procedure.ID] = base
		for stepID, values := range perStep {
			targets[base.Procedure.ID+"/"+stepID] = append([]string(nil), values...)
		}
	}
	if _, err := deriveStandaloneHandoffProbes(bundle.Handoffs, fixtures); err != nil {
		return d4O1NormalizedBusinessModelProbe{}, err
	}
	set := referenceSet{Handoffs: append([]referenceSetHandoff(nil), bundle.Handoffs.Handoffs...)}
	return deriveD4O1NormalizedBusinessModelProbe(set, fixtures, targets)
}

func renameInlineO1StepProbe(bundle *businessAuthoringBundleProbe, procedureID, from, to string) bool {
	for p := range bundle.Procedures {
		if bundle.Procedures[p].Procedure.ID != procedureID {
			continue
		}
		for s := range bundle.Procedures[p].Steps {
			if bundle.Procedures[p].Steps[s].ID == from {
				bundle.Procedures[p].Steps[s].ID = to
				return true
			}
		}
	}
	return false
}
