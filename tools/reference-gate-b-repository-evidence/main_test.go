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

func TestIntermediateSourceRejectsNewDirtyTreeAfterCleanStart(t *testing.T) {
	source := cleanSource()
	postRun := cleanPostRun()
	postRun.WorkingTreeClean = false

	if err := verifyIntermediateRepositorySource(source, postRun, false, "first-check"); err == nil {
		t.Fatal("verifyIntermediateRepositorySource() error=nil, want new dirty tree failure")
	}
}

func TestIntermediateSourceAllowsInitialDirtyTreeOnlyForAllowDirtyDiagnostic(t *testing.T) {
	source := cleanSource()
	source.WorkingTreeClean = false
	postRun := cleanPostRun()
	postRun.WorkingTreeClean = false

	if err := verifyIntermediateRepositorySource(source, postRun, true, "diagnostic-check"); err != nil {
		t.Fatalf("verifyIntermediateRepositorySource() error=%v, want allow-dirty diagnostic continuation", err)
	}
	if err := verifyIntermediateRepositorySource(source, postRun, false, "normal-check"); err == nil {
		t.Fatal("verifyIntermediateRepositorySource() error=nil for dirty normal Evidence run")
	}
}

func TestIntermediateSourceRejectsRevisionDriftEvenInAllowDirtyDiagnostic(t *testing.T) {
	source := cleanSource()
	source.WorkingTreeClean = false
	postRun := cleanPostRun()
	postRun.RepositoryRevision = "R2"
	postRun.RevisionStable = false
	postRun.WorkingTreeClean = false

	if err := verifyIntermediateRepositorySource(source, postRun, true, "diagnostic-check"); err == nil {
		t.Fatal("verifyIntermediateRepositorySource() error=nil, want revision drift failure")
	}
}

func TestIntermediateSourceRejectsUncheckedObservation(t *testing.T) {
	postRun := cleanPostRun()
	postRun.Checked = false

	if err := verifyIntermediateRepositorySource(cleanSource(), postRun, false, "first-check"); err == nil {
		t.Fatal("verifyIntermediateRepositorySource() error=nil, want unchecked source failure")
	}
}

func TestFinalizeKeepsReproducibleFalseOnTargetedFailure(t *testing.T) {
	result := evidence{}
	err := finalizeRepositoryEvidence(&result, cleanSource(), cleanPostRun(), true, false)
	if err == nil {
		t.Fatal("finalizeRepositoryEvidence() error=nil, want targeted failure")
	}
	if result.ReproducibleRevision {
		t.Fatal("reproducibleRevision=true on targeted failure")
	}
}

func TestFinalizeKeepsReproducibleFalseOnBroadFailure(t *testing.T) {
	result := evidence{}
	err := finalizeRepositoryEvidence(&result, cleanSource(), cleanPostRun(), false, true)
	if err == nil {
		t.Fatal("finalizeRepositoryEvidence() error=nil, want broad failure")
	}
	if result.ReproducibleRevision {
		t.Fatal("reproducibleRevision=true on broad failure")
	}
}

func TestFinalizeRejectsUncheckedPostRun(t *testing.T) {
	result := evidence{}
	postRun := cleanPostRun()
	postRun.Checked = false

	err := finalizeRepositoryEvidence(&result, cleanSource(), postRun, false, false)
	if err == nil {
		t.Fatal("finalizeRepositoryEvidence() error=nil, want unchecked post-run failure")
	}
	if result.ReproducibleRevision {
		t.Fatal("reproducibleRevision=true with unchecked post-run")
	}
}

func TestFinalizeMarksReproducibleOnlyAfterChecksPassAndSourceIsStable(t *testing.T) {
	result := evidence{}
	if err := finalizeRepositoryEvidence(&result, cleanSource(), cleanPostRun(), false, false); err != nil {
		t.Fatalf("finalizeRepositoryEvidence() error=%v", err)
	}
	if !result.ReproducibleRevision {
		t.Fatal("reproducibleRevision=false after successful checks and stable source")
	}
}

func cleanSource() repositoryevidence.Source {
	return repositoryevidence.Source{
		RepositoryRevision: "R",
		WorkingTreeClean:   true,
	}
}

func cleanPostRun() repositoryevidence.PostRun {
	return repositoryevidence.PostRun{
		Checked:            true,
		RepositoryRevision: "R",
		RevisionStable:     true,
		WorkingTreeClean:   true,
	}
}
