package artifact

import (
	"encoding/json"
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
	foundGap := false
	for _, finding := range findings {
		switch finding.Code {
		case FindingSurfaceGapMissing:
			foundGap = true
			if finding.Detail == "" {
				t.Fatalf("missing gap detail: %+v", finding)
			}
		case FindingArtifactTampered:
			// This mutation changes the unresolved list without rehashing.
		default:
			t.Fatalf("unexpected finding: %+v", finding)
		}
	}
	if !foundGap {
		t.Fatalf("missing gap not reported: %+v", findings)
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
	found := false
	for _, finding := range findings {
		if finding.Code == FindingManifestEntryMissing {
			found = true
		}
	}
	if !found {
		t.Fatalf("an artifact missing from the manifest must be reported: %+v", findings)
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
func TestCheckSetReportsAnOperationWithoutBasis(t *testing.T) {
	set := negativeFixtureSet(t)
	for i := range set.Artifacts {
		if set.Artifacts[i].Kind != string(KindOperationSurface) {
			continue
		}
		payload, err := decodeOperationSurface(set.Artifacts[i].Content)
		if err != nil {
			t.Fatal(err)
		}
		if len(payload.Operations) == 0 {
			t.Fatal("the fixture must derive operations")
		}
		payload.Operations[0].Basis = ""
		payload.Operations[1].Autonomy = "whatever"
		encoded, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		set.Artifacts[i].Content = string(encoded) + "\n"
	}

	findings := CheckSet(set)
	if len(findings) < 2 {
		t.Fatalf("a missing basis and an unknown autonomy must both be reported: %+v", findings)
	}
	incomplete := 0
	for _, finding := range findings {
		switch finding.Code {
		case FindingOperationIncomplete:
			incomplete++
		case FindingArtifactTampered:
			// The modified operation content no longer matches its digest.
		default:
			t.Fatalf("unexpected finding: %+v", finding)
		}
	}
	if incomplete < 2 {
		t.Fatalf("expected two operation findings, got: %+v", findings)
	}
}
func TestCheckSetReportsADuplicateArtifact(t *testing.T) {
	set := negativeFixtureSet(t)
	set.Artifacts = append(set.Artifacts, set.Artifacts[0])

	findings := CheckSet(set)
	found := false
	for _, finding := range findings {
		if finding.Code == FindingDuplicateArtifact {
			found = true
		}
	}
	if !found {
		t.Fatalf("a duplicated artifact must be reported: %+v", findings)
	}
}

func TestCheckSetReportsAStaleManifestEntry(t *testing.T) {
	set := negativeFixtureSet(t)
	set.Manifest.Entries = append(set.Manifest.Entries, ManifestEntry{
		Kind: "ghost", Rule: "ghost", Version: "v1", Fingerprint: "sha256:x", Digest: "sha256:ghost",
	})

	findings := CheckSet(set)
	found := false
	for _, finding := range findings {
		if finding.Code == FindingManifestEntryStale {
			found = true
		}
	}
	if !found {
		t.Fatalf("a manifest entry without an artifact must be reported: %+v", findings)
	}
}
func TestCheckSetReportsATamperedManifest(t *testing.T) {
	set := negativeFixtureSet(t)
	set.Manifest.Entries = append(set.Manifest.Entries, ManifestEntry{
		Kind: "ghost", Rule: "ghost", Version: "v1", Fingerprint: "sha256:x", Digest: "sha256:ghost",
	})

	findings := CheckSet(set)
	found := false
	for _, finding := range findings {
		if finding.Code == FindingManifestTampered {
			found = true
		}
	}
	if !found {
		t.Fatalf("a manifest whose entries changed must be reported: %+v", findings)
	}
}
