package targetprofile

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeployReplaceInstallsAndKeepsPreviousArtifact(t *testing.T) {
	_, _, _, candidates := selectionPaths(t)
	root := t.TempDir()
	profilePath := filepath.Join(repoRoot(t), "tools", "reference-target-profile", "testdata", "desktop-replace.json")
	first, err := Deploy(context.Background(), DeployOptions{
		ProfilePath: profilePath, CandidatesDir: candidates, RootDir: root,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !first.Passed || first.ArtifactDigest == "" || first.PreviousDigest != "" || first.HistoryPath != "" {
		t.Fatalf("first replace deployment failed: %+v", first)
	}

	profile, err := Load(profilePath)
	if err != nil {
		t.Fatal(err)
	}
	profile.ID = "desktop-replace-alt"
	profile.GeneratorID = "alternative"
	changed, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	altProfile := filepath.Join(t.TempDir(), "desktop-replace-alt.json")
	if err := os.WriteFile(altProfile, append(changed, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	second, err := Deploy(context.Background(), DeployOptions{
		ProfilePath: altProfile, CandidatesDir: candidates, RootDir: root,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !second.Passed || second.ArtifactDigest == "" || second.PreviousDigest == "" || second.HistoryPath == "" {
		t.Fatalf("second replace deployment failed: %+v", second)
	}
	if second.ArtifactDigest == first.ArtifactDigest {
		t.Fatal("replacement did not change the artifact digest")
	}
	if _, err := os.Stat(second.HistoryPath); err != nil {
		t.Fatalf("previous artifact was not retained: %v", err)
	}
}

func TestDeployInPlaceUpdatesTarget(t *testing.T) {
	_, _, _, candidates := selectionPaths(t)
	root := t.TempDir()
	profilePath := filepath.Join(repoRoot(t), "tools", "reference-target-profile", "testdata", "desktop-replace.json")
	profile, err := Load(profilePath)
	if err != nil {
		t.Fatal(err)
	}
	profile.ID = "desktop-in-place"
	profile.Update = UpdateInPlace
	changed, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	inPlace := filepath.Join(t.TempDir(), "desktop-in-place.json")
	if err := os.WriteFile(inPlace, append(changed, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		receipt, err := Deploy(context.Background(), DeployOptions{
			ProfilePath: inPlace, CandidatesDir: candidates, RootDir: root,
		})
		if err != nil {
			t.Fatal(err)
		}
		if !receipt.Passed || receipt.ArtifactDigest == "" {
			t.Fatalf("in-place deployment %d failed: %+v", i, receipt)
		}
		if i == 1 && (receipt.PreviousDigest == "" || receipt.HistoryPath == "") {
			t.Fatalf("in-place deployment did not retain the previous artifact: %+v", receipt)
		}
	}
}

func TestDeployRejectsUnsupportedUpdateOrStorage(t *testing.T) {
	_, _, _, candidates := selectionPaths(t)
	for _, name := range []string{"server", "embedded"} {
		profilePath := filepath.Join(repoRoot(t), "tools", "reference-target-profile", "testdata", name+".json")
		receipt, err := Deploy(context.Background(), DeployOptions{
			ProfilePath: profilePath, CandidatesDir: candidates, RootDir: t.TempDir(),
		})
		if err != nil {
			t.Fatal(err)
		}
		if receipt.Passed || receipt.Reason == "" {
			t.Fatalf("%s profile was deployed by the reference: %+v", name, receipt)
		}
		if !strings.Contains(receipt.Reason, "storage") && !strings.Contains(receipt.Reason, "update") {
			t.Fatalf("%s failure reason is unclear: %q", name, receipt.Reason)
		}
	}
}
