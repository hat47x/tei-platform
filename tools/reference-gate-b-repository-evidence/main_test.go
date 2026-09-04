package main

import (
	"reflect"
	"testing"
)

func TestTargetedChecksKeepGateBMinimumEvidenceSeparate(t *testing.T) {
	got := targetedChecks()
	if len(got) != 2 {
		t.Fatalf("targeted checks=%d want=2", len(got))
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
