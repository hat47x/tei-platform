package targetprofile

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	if !first.Passed || !first.Installed || first.ArtifactDigest == "" || first.PreviousDigest != "" || first.HistoryPath != "" {
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

func TestRollbackToPreservesVerifiedCurrentHistory(t *testing.T) {
	for _, state := range []string{"missing", "valid", "corrupt", "directory"} {
		t.Run(state, func(t *testing.T) {
			root := t.TempDir()
			history := filepath.Join(root, "history")
			if err := os.Mkdir(history, 0o755); err != nil {
				t.Fatal(err)
			}
			current, target := []byte("current artifact"), []byte("retired artifact")
			targetDigest := digestBytes(target)
			targetHistory := filepath.Join(history, strings.TrimPrefix(targetDigest, "sha256:"))
			currentHistory := filepath.Join(history, strings.TrimPrefix(digestBytes(current), "sha256:"))
			currentPath := filepath.Join(root, "artifact")
			for path, contents := range map[string][]byte{currentPath: current, targetHistory: target} {
				if err := os.WriteFile(path, contents, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			switch state {
			case "valid", "corrupt":
				contents := current
				if state == "corrupt" {
					contents = []byte("damaged history")
				}
				if err := os.WriteFile(currentHistory, contents, 0o600); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.Mkdir(currentHistory, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			receipt, err := RollbackTo(root, targetDigest)
			if err != nil {
				t.Fatal(err)
			}
			want := target
			if state == "corrupt" || state == "directory" {
				want = current
				if receipt.Passed || !strings.Contains(receipt.Reason, "current history artifact") {
					t.Fatalf("unusable current history was accepted: %+v", receipt)
				}
				if _, err := os.Stat(filepath.Join(root, ".artifact-rollback")); !os.IsNotExist(err) {
					t.Fatalf("rollback staging started before history validation: %v", err)
				}
			} else {
				if !receipt.Passed || receipt.PreviousDigest != digestBytes(current) {
					t.Fatalf("valid rollback failed: %+v", receipt)
				}
				installed, err := os.ReadFile(currentPath)
				if err != nil || string(installed) != string(target) {
					t.Fatalf("target artifact not installed: %q, %v", installed, err)
				}
				saved, err := os.ReadFile(currentHistory)
				if err != nil || string(saved) != string(current) {
					t.Fatalf("current artifact not retained: %q, %v", saved, err)
				}
				// The retained artifact must support the reverse rollback too.
				reverse, err := RollbackTo(root, receipt.PreviousDigest)
				if err != nil || !reverse.Passed || reverse.PreviousDigest != targetDigest {
					t.Fatalf("reverse rollback failed: %+v, %v", reverse, err)
				}
				want = current
			}
			installed, err := os.ReadFile(currentPath)
			if err != nil || string(installed) != string(want) {
				t.Fatalf("current artifact changed unexpectedly: %q, %v", installed, err)
			}
			storedTarget, err := os.ReadFile(targetHistory)
			if err != nil || string(storedTarget) != string(target) {
				t.Fatalf("target history changed: %q, %v", storedTarget, err)
			}
			if state == "corrupt" {
				stored, err := os.ReadFile(currentHistory)
				if err != nil || string(stored) != "damaged history" {
					t.Fatalf("damaged evidence was overwritten: %q, %v", stored, err)
				}
			}
		})
	}
}

func TestDeployNoneBuildsWithoutInstalling(t *testing.T) {
	_, _, _, candidates := selectionPaths(t)
	profile, err := Load(filepath.Join(repoRoot(t), "tools", "reference-target-profile", "testdata", "desktop-replace.json"))
	if err != nil {
		t.Fatal(err)
	}
	profile.ID = "desktop-none"
	profile.Storage = StorageNone
	changed, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	profilePath := filepath.Join(t.TempDir(), "desktop-none.json")
	if err := os.WriteFile(profilePath, append(changed, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "unused-root")
	receipt, err := Deploy(context.Background(), DeployOptions{
		ProfilePath: profilePath, CandidatesDir: candidates, RootDir: root,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Passed || receipt.Installed || receipt.Storage != StorageNone || receipt.ArtifactDigest == "" {
		t.Fatalf("storage=none deployment failed: %+v", receipt)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("storage=none created the root: %v", err)
	}
}

func TestDeploySQLiteAndRollback(t *testing.T) {
	_, _, _, candidates := selectionPaths(t)
	profile, err := Load(filepath.Join(repoRoot(t), "tools", "reference-target-profile", "testdata", "desktop-replace.json"))
	if err != nil {
		t.Fatal(err)
	}
	profile.Storage = StorageSQLite
	changed, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	profilePath := filepath.Join(t.TempDir(), "sqlite.json")
	if err := os.WriteFile(profilePath, append(changed, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	first, err := Deploy(context.Background(), DeployOptions{
		ProfilePath: profilePath, CandidatesDir: candidates, RootDir: root,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !first.Passed || !first.Installed || first.DatabasePath == "" || first.ArtifactDigest == "" {
		t.Fatalf("sqlite deployment failed: %+v", first)
	}
	if _, err := os.Stat(first.DatabasePath); err != nil {
		t.Fatalf("sqlite database is missing: %v", err)
	}

	alternativeProfile := profile
	alternativeProfile.ID = "sqlite-alt"
	alternativeProfile.GeneratorID = "alternative"
	changed, err = json.MarshalIndent(alternativeProfile, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(profilePath, append(changed, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	second, err := Deploy(context.Background(), DeployOptions{
		ProfilePath: profilePath, CandidatesDir: candidates, RootDir: root,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !second.Passed || second.PreviousDigest != first.ArtifactDigest {
		t.Fatalf("second sqlite deployment failed: %+v", second)
	}

	rolledBack, err := RollbackTo(root, first.ArtifactDigest)
	if err != nil {
		t.Fatal(err)
	}
	if !rolledBack.Passed || rolledBack.ArtifactDigest != first.ArtifactDigest {
		t.Fatalf("sqlite rollback-to failed: %+v", rolledBack)
	}
	swapped, err := Rollback(root)
	if err != nil {
		t.Fatal(err)
	}
	if !swapped.Passed || swapped.ArtifactDigest != second.ArtifactDigest {
		t.Fatalf("sqlite slot rollback failed: %+v", swapped)
	}

	history, err := History(root)
	if err != nil {
		t.Fatal(err)
	}
	if !history.Passed || len(history.Entries) != 2 {
		t.Fatalf("sqlite history did not verify: %+v", history)
	}

	thirdProfile := profile
	thirdProfile.ID = "sqlite-third"
	thirdProfile.GeneratorID = "long-running-template"
	changed, err = json.MarshalIndent(thirdProfile, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(profilePath, append(changed, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Deploy(context.Background(), DeployOptions{
		ProfilePath: profilePath, CandidatesDir: candidates, RootDir: root,
	}); err != nil {
		t.Fatal(err)
	}
	dryRun, err := PruneHistory(root, 2, true)
	if err != nil {
		t.Fatal(err)
	}
	if !dryRun.Passed || len(dryRun.Kept) != 2 || len(dryRun.WouldDelete) != 1 {
		t.Fatalf("sqlite prune dry-run failed: %+v", dryRun)
	}
	pruned, err := PruneHistory(root, 2, false)
	if err != nil {
		t.Fatal(err)
	}
	if !pruned.Passed || len(pruned.Deleted) != 1 || pruned.AuditPath == "" {
		t.Fatalf("sqlite prune failed: %+v", pruned)
	}
	history, err = History(root)
	if err != nil {
		t.Fatal(err)
	}
	if !history.Passed || len(history.Entries) != 2 {
		t.Fatalf("sqlite history after prune is wrong: %+v", history)
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
	profile, err := Load(filepath.Join(repoRoot(t), "tools", "reference-target-profile", "testdata", "embedded.json"))
	if err != nil {
		t.Fatal(err)
	}
	profile.Storage = StorageRemote
	changed, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	profilePath := filepath.Join(t.TempDir(), "remote.json")
	if err := os.WriteFile(profilePath, append(changed, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	receipt, err := Deploy(context.Background(), DeployOptions{
		ProfilePath: profilePath, CandidatesDir: candidates, RootDir: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Passed || !strings.Contains(receipt.Reason, "storage") {
		t.Fatalf("remote profile was deployed by the reference: %+v", receipt)
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

func TestDeployPreservesExistingCurrentHistory(t *testing.T) {
	_, _, _, candidates := selectionPaths(t)
	base := filepath.Join(repoRoot(t), "tools/reference-target-profile/testdata/desktop-replace.json")
	for _, mode := range []string{UpdateReplace, UpdateInPlace} {
		for _, state := range []string{"valid", "corrupt"} {
			t.Run(mode+"/"+state, func(t *testing.T) {
				root := t.TempDir()
				current := []byte("current artifact")
				currentPath := filepath.Join(root, "artifact")
				if err := os.WriteFile(currentPath, current, 0o755); err != nil {
					t.Fatal(err)
				}
				saved := "current artifact"
				if state == "corrupt" {
					saved = "damaged history"
				}
				history := filepath.Join(root, "history", strings.TrimPrefix(digestBytes(current), "sha256:"))
				if err := os.MkdirAll(filepath.Dir(history), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(history, []byte(saved), 0o600); err != nil {
					t.Fatal(err)
				}
				oldTime := time.Unix(100, 0)
				if err := os.Chtimes(history, oldTime, oldTime); err != nil {
					t.Fatal(err)
				}
				profile, err := Load(base)
				if err != nil {
					t.Fatal(err)
				}
				profile.Update = mode
				raw, err := json.Marshal(profile)
				if err != nil {
					t.Fatal(err)
				}
				profilePath := filepath.Join(t.TempDir(), "profile.json")
				if err := os.WriteFile(profilePath, raw, 0o600); err != nil {
					t.Fatal(err)
				}
				receipt, err := Deploy(context.Background(), DeployOptions{ProfilePath: profilePath, CandidatesDir: candidates, RootDir: root})
				if err != nil {
					t.Fatal(err)
				}
				if state == "valid" {
					if !receipt.Passed || !receipt.Installed {
						t.Fatalf("valid retained history blocked deploy: %+v", receipt)
					}
				} else {
					if receipt.Passed || receipt.Installed || !strings.Contains(receipt.Reason, "history artifact") {
						t.Fatalf("corrupt history was overwritten: %+v", receipt)
					}
					now, err := os.ReadFile(currentPath)
					if err != nil || string(now) != string(current) {
						t.Fatalf("rejected deploy changed current: %q %v", now, err)
					}
				}
				now, err := os.ReadFile(history)
				if err != nil || string(now) != saved {
					t.Fatalf("retained history overwritten: %q %v", now, err)
				}
				info, err := os.Stat(history)
				if err != nil || !info.ModTime().Equal(oldTime) {
					t.Fatalf("history retention time changed: %v %v", info, err)
				}
				entries, err := os.ReadDir(root)
				if err != nil {
					t.Fatal(err)
				}
				for _, entry := range entries {
					if entry.Name() == ".update-lock" || strings.HasPrefix(entry.Name(), ".staging-") {
						t.Fatalf("deploy left temporary state: %s", entry.Name())
					}
				}
			})
		}
	}
}
