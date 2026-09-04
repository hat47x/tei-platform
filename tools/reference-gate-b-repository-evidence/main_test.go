package main

import (
	"reflect"
	"testing"

	"tei/tools/internal/repositoryevidence"
)

func TestInitialRepositorySourceNeverClaimsReproducibleRevision(t *testing.T) {
	result := evidence{ReproducibleRevision: true}
	source := repositoryevidence.Source{
		RepositoryRevision: "abc123",
		WorkingTreeClean:   true,
	}

	recordInitialRepositorySource(&result, source)

	if result.RepositoryRevision != source.RepositoryRevision {
		t.Fatalf("repository revision=%q want=%q", result.RepositoryRevision, source.RepositoryRevision)
	}
	if !result.WorkingTreeClean {
		t.Fatal("開始時clean状態を記録できていません")
	}
	if result.ReproducibleRevision {
		t.Fatal("post-run確認前にreproducibleRevision=trueを記録しています")
	}
}

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
