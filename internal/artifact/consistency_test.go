package artifact

import (
	"path/filepath"
	"testing"
)

func negativeFixtureSet(t *testing.T) WorkspaceSet {
	t.Helper()
	set, err := RenderWorkspace(filepath.Join("..", "..", "reference", "standard-crud", "workspace"), "ledger")
	if err != nil {
		t.Fatalf("render workspace: %v", err)
	}
	return set
}

func TestCheckSetAcceptsTheNegativeFixture(t *testing.T) {
	set := negativeFixtureSet(t)
	if findings := CheckSet(set); len(findings) != 0 {
		t.Fatalf("a consistently rendered set must have no findings: %+v", findings)
	}
}

func TestCheckSetReportsADroppedUnresolvedGap(t *testing.T) {
	set := negativeFixtureSet(t)
	for i := range set.Artifacts {
		if set.Artifacts[i].Kind == string(KindOperationSurface) {
			set.Artifacts[i].Unresolved = nil
		}
	}

	findings := CheckSet(set)
	if len(findings) == 0 {
		t.Fatal("dropping the unresolved gaps must be reported")
	}
	for _, finding := range findings {
		if finding.Code != FindingSurfaceGapMissing || finding.Detail == "" {
			t.Fatalf("unexpected finding: %+v", finding)
		}
	}
}

func TestCheckSetReportsACompleteSetWithUnresolved(t *testing.T) {
	set := negativeFixtureSet(t)
	set.Manifest.Complete = true

	findings := CheckSet(set)
	if len(findings) == 0 {
		t.Fatal("a set marked complete while unresolved must be reported")
	}
	found := false
	for _, finding := range findings {
		if finding.Code == FindingCompleteWithUnresolved {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected %s: %+v", FindingCompleteWithUnresolved, findings)
	}
}

func TestCheckSetReportsAMissingManifestEntry(t *testing.T) {
	set := negativeFixtureSet(t)
	set.Manifest.Entries = set.Manifest.Entries[:1]

	findings := CheckSet(set)
	if len(findings) == 0 {
		t.Fatal("an artifact missing from the manifest must be reported")
	}
	for _, finding := range findings {
		if finding.Code != FindingManifestEntryMissing {
			t.Fatalf("unexpected finding: %+v", finding)
		}
	}
}
func TestCheckSetReportsAnUnknownRule(t *testing.T) {
	set := negativeFixtureSet(t)
	for i := range set.Artifacts {
		if set.Artifacts[i].Kind == string(KindTableDefinition) {
			set.Artifacts[i].Rule.ID = "tei.artifact.typo"
		}
	}

	findings := CheckSet(set)
	found := false
	for _, finding := range findings {
		if finding.Code == FindingUnknownRule {
			found = true
		}
	}
	if !found {
		t.Fatalf("an artifact naming another rule must be reported: %+v", findings)
	}
}
func TestCheckSetReportsMissingRequiredLines(t *testing.T) {
	set := negativeFixtureSet(t)
	for i := range set.Artifacts {
		if set.Artifacts[i].Kind == string(KindTableDefinition) {
			set.Artifacts[i].Content = "columns only\n"
		}
	}

	findings := CheckSet(set)
	found := false
	for _, finding := range findings {
		if finding.Code == FindingContentIncomplete {
			found = true
		}
	}
	if !found {
		t.Fatalf("an artifact that lost its table line must be reported: %+v", findings)
	}
}
