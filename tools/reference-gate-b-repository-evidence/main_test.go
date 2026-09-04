package main

import (
	"reflect"
	"testing"
)

func TestTargetedChecksKeepGateBMinimumEvidenceSeparate(t *testing.T) {
	got := targetedChecks()
	if len(got) != 3 {
		t.Fatalf("targeted checks=%d want=3", len(got))
	}

	wantScenario := commandSpec{
		Name: "final-ui-less-scenario",
		Args: []string{
			"test", "./internal/driver",
			"-run", "^TestOrderEntryGateBScenarioConnectsMajorBoundariesWithoutUI$",
			"-count=1",
		},
	}
	if !reflect.DeepEqual(got[0], wantScenario) {
		t.Fatalf("final Scenario commandがGate B runbookと一致しません: %#v", got[0])
	}

	wantExternalEvidence := commandSpec{
		Name: "external-evidence-intake",
		Args: []string{
			"test",
			"./internal/database/externalqueryevidence",
			"./tools/reference-external-query-evidence",
			"-count=1",
		},
	}
	if !reflect.DeepEqual(got[1], wantExternalEvidence) {
		t.Fatalf("external Evidence intake commandがGate B runbookと一致しません: %#v", got[1])
	}

	wantRepositoryEvidenceBoundary := commandSpec{
		Name: "repository-evidence-source-boundary",
		Args: []string{
			"test",
			"./tools/internal/repositoryevidence",
			"./tools/reference-gate-b-repository-evidence",
			"-count=1",
		},
	}
	if !reflect.DeepEqual(got[2], wantRepositoryEvidenceBoundary) {
		t.Fatalf("Repository Evidence source boundary commandがGate B targeted Evidenceと一致しません: %#v", got[2])
	}
}

func TestBroadCheckIsExplicitRepositoryWideTest(t *testing.T) {
	got := broadCheck()
	want := commandSpec{
		Name: "repository-wide-go-test",
		Args: []string{"test", "./..."},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("broad check=%#v want=%#v", got, want)
	}
}

func TestCommandDisplayDoesNotUseShellString(t *testing.T) {
	spec := commandSpec{Name: "example", Args: []string{"test", "./internal/driver"}}
	got := commandDisplay(spec)
	want := []string{"go", "test", "./internal/driver"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("command display=%#v want=%#v", got, want)
	}
}
