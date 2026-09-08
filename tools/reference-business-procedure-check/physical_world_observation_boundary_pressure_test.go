package main

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"tei/internal/driver"
)

// physicalWorldCompletionPressure is a test-only probe for second-corpus pressure.
// It is not a production Business Procedure / Observation schema.
type physicalWorldCompletionPressure struct {
	BusinessFact      string
	CommandAccepted   bool
	ObservationResult *driver.ObservationReadExecution
}

func TestSecondCorpusPhysicalObservationProbeDoesNotSmuggleDataAuthorityOrHandoffShape(t *testing.T) {
	probeType := reflect.TypeOf(physicalWorldCompletionPressure{})
	want := []string{"BusinessFact", "CommandAccepted", "ObservationResult"}
	if got := fieldNames(probeType); !reflect.DeepEqual(got, want) {
		t.Fatalf("physical-world Observation pressureが未証明meaningへ膨張しました: got=%v want=%v", got, want)
	}
	for _, forbidden := range []string{
		"Data", "Relation", "Database", "Datasource", "Ownership", "ApplicationPattern",
		"From", "To", "Procedure", "Step", "Condition", "Handoff",
	} {
		if _, found := probeType.FieldByName(forbidden); found {
			t.Fatalf("physical-world Observation pressureへ未証明のData authority / Handoff field %qを混入しました", forbidden)
		}
	}
}

func physicalWorldCompletionDiagnostics(stepID string, pressure physicalWorldCompletionPressure) []diagnostic {
	if !pressure.CommandAccepted {
		return nil
	}

	var result report
	if pressure.ObservationResult == nil {
		result.Diagnostics = append(result.Diagnostics, diagnostic{
			Code:     "BUSINESS_PHYSICAL_COMPLETION_REQUIRES_OBSERVATION",
			Severity: severityArchitecturePolicy,
			Step:     stepID,
			Message:  fmt.Sprintf("%s cannot be established from command acceptance without explicit observation", pressure.BusinessFact),
		})
		sortDiagnostics(result.Diagnostics)
		return result.Diagnostics
	}

	observed, ok := pressure.ObservationResult.Result.(bool)
	if !ok || !observed {
		result.Diagnostics = append(result.Diagnostics, diagnostic{
			Code:     "BUSINESS_PHYSICAL_COMPLETION_NOT_OBSERVED",
			Severity: severityArchitecturePolicy,
			Step:     stepID,
			Message:  fmt.Sprintf("%s has not been positively observed", pressure.BusinessFact),
		})
	}
	sortDiagnostics(result.Diagnostics)
	return result.Diagnostics
}

func TestSecondCorpusCommandAcceptanceDoesNotEstablishPhysicalCompletion(t *testing.T) {
	facts := []string{
		"material physically received",
		"manufacturing physically completed",
		"finished goods physically present",
		"shipment physically completed",
	}

	for _, fact := range facts {
		t.Run(fact, func(t *testing.T) {
			pressure := physicalWorldCompletionPressure{
				BusinessFact:    fact,
				CommandAccepted: true,
			}
			diagnostics := physicalWorldCompletionDiagnostics("SECOND-CORPUS", pressure)
			if err := checkExpectedDiagnostics(diagnostics, "BUSINESS_PHYSICAL_COMPLETION_REQUIRES_OBSERVATION"); err != nil {
				t.Fatalf("command acceptanceからphysical completionを推論しました: %v\n%+v", err, diagnostics)
			}
		})
	}
}

func TestSecondCorpusExplicitObservationStillSeparatesReadSuccessFromPhysicalCompletion(t *testing.T) {
	execution, err := driver.ExecuteObservationRead(
		context.Background(),
		"second-corpus.physical-state.read",
		driver.ObservationEvidenceExternalObserved,
		func(context.Context) (any, error) {
			return false, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	pressure := physicalWorldCompletionPressure{
		BusinessFact:      "manufacturing physically completed",
		CommandAccepted:   true,
		ObservationResult: &execution,
	}
	diagnostics := physicalWorldCompletionDiagnostics("SECOND-CORPUS", pressure)
	if err := checkExpectedDiagnostics(diagnostics, "BUSINESS_PHYSICAL_COMPLETION_NOT_OBSERVED"); err != nil {
		t.Fatalf("Observation read成功をphysical completionとして扱いました: %v\n%+v", err, diagnostics)
	}
}

func TestSecondCorpusPositiveExternalObservationCanEstablishTheObservedFactWithoutCreatingCanonicalObservationIdentity(t *testing.T) {
	execution, err := driver.ExecuteObservationRead(
		context.Background(),
		"second-corpus.physical-state.read",
		driver.ObservationEvidenceExternalObserved,
		func(context.Context) (any, error) {
			return true, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	pressure := physicalWorldCompletionPressure{
		BusinessFact:      "finished goods physically present",
		CommandAccepted:   true,
		ObservationResult: &execution,
	}
	if diagnostics := physicalWorldCompletionDiagnostics("SECOND-CORPUS", pressure); len(diagnostics) != 0 {
		t.Fatalf("positive external observationを保持できません: %+v", diagnostics)
	}
	if execution.Nature != driver.ObservationEvidenceExternalObserved {
		t.Fatalf("physical-world pressureのEvidence nature=%q", execution.Nature)
	}
	if execution.ReadIdentity != "second-corpus.physical-state.read" {
		t.Fatalf("runtime read locator=%q", execution.ReadIdentity)
	}
}

func TestSecondCorpusObservationExecutionErrorDoesNotBecomePhysicalCompletion(t *testing.T) {
	execution, err := driver.ExecuteObservationRead(
		context.Background(),
		"second-corpus.physical-state.read",
		driver.ObservationEvidenceExternalObserved,
		func(context.Context) (any, error) {
			return nil, fmt.Errorf("external state unavailable")
		},
	)
	if err == nil {
		t.Fatal("Observation Execution Errorが失われています")
	}
	if execution.Result != nil {
		t.Fatalf("Observation失敗時にphysical completion resultを生成しました: %+v", execution)
	}
	if execution.ReadIdentity == "" || execution.Nature != driver.ObservationEvidenceExternalObserved {
		t.Fatalf("Observation失敗時のruntime Evidence locatorが失われました: %+v", execution)
	}
}
