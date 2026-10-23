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

func TestRollbackToHistoryArtifact(t *testing.T) {
	_, _, _, candidates := selectionPaths(t)
	root := t.TempDir()
	profilePath := filepath.Join(repoRoot(t), "tools", "reference-target-profile", "testdata", "desktop-replace.json")
	first, err := Deploy(context.Background(), DeployOptions{
		ProfilePath: profilePath, CandidatesDir: candidates, RootDir: root,
	})
	if err != nil {
		t.Fatal(err)
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
	rolledBack, err := RollbackTo(root, first.ArtifactDigest)
	if err != nil {
		t.Fatal(err)
	}
	if !rolledBack.Passed || rolledBack.RolledBackTo != first.ArtifactDigest || rolledBack.PreviousDigest != second.ArtifactDigest {
		t.Fatalf("rollback to history failed: %+v", rolledBack)
	}
	installed, err := os.ReadFile(filepath.Join(root, "artifact"))
	if err != nil {
		t.Fatal(err)
	}
	if digestBytes(installed) != first.ArtifactDigest {
		t.Fatalf("installed artifact digest=%s want %s", digestBytes(installed), first.ArtifactDigest)
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

func TestDeployRejectsUnsupportedStorage(t *testing.T) {
	_, _, _, candidates := selectionPaths(t)
	profilePath := filepath.Join(repoRoot(t), "tools", "reference-target-profile", "testdata", "embedded.json")
	receipt, err := Deploy(context.Background(), DeployOptions{
		ProfilePath: profilePath, CandidatesDir: candidates, RootDir: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Passed || !strings.Contains(receipt.Reason, "storage") {
		t.Fatalf("sqlite profile was deployed by the reference: %+v", receipt)
	}
}

func TestDeployRollingSwitchesSlotsAndRollsBack(t *testing.T) {
	_, _, _, candidates := selectionPaths(t)
	root := t.TempDir()
	serverPath := filepath.Join(repoRoot(t), "tools", "reference-target-profile", "testdata", "server.json")
	first, err := Deploy(context.Background(), DeployOptions{
		ProfilePath: serverPath, CandidatesDir: candidates, RootDir: root,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !first.Passed || first.CurrentSlot != "a" || first.PreviousSlot != "" || first.RollbackAvailable {
		t.Fatalf("first rolling deployment failed: %+v", first)
	}
	profile, err := Load(serverPath)
	if err != nil {
		t.Fatal(err)
	}
	profile.ID = "server-template"
	profile.GeneratorID = "template"
	changed, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	secondProfile := filepath.Join(t.TempDir(), "server-template.json")
	if err := os.WriteFile(secondProfile, append(changed, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	second, err := Deploy(context.Background(), DeployOptions{
		ProfilePath: secondProfile, CandidatesDir: candidates, RootDir: root,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !second.Passed || second.CurrentSlot != "b" || second.PreviousSlot != "a" || !second.RollbackAvailable {
		t.Fatalf("second rolling deployment did not switch slots: %+v", second)
	}
	for _, slot := range []string{"a", "b"} {
		if _, err := os.Stat(filepath.Join(root, "slots", slot, "artifact")); err != nil {
			t.Fatalf("slot %s artifact is missing: %v", slot, err)
		}
	}
	rolledBack, err := Rollback(root)
	if err != nil {
		t.Fatal(err)
	}
	if !rolledBack.Passed || rolledBack.CurrentSlot != "a" || rolledBack.PreviousSlot != "b" {
		t.Fatalf("rollback did not restore the previous slot: %+v", rolledBack)
	}
	if rolledBack.ArtifactDigest != first.ArtifactDigest {
		t.Fatalf("rollback artifact digest=%s want %s", rolledBack.ArtifactDigest, first.ArtifactDigest)
	}
}
