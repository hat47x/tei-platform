package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func (s laneArtifactStore) verifyRetainedLanes(lanes []laneEvidence) error {
	if s.dir == "" {
		return fmt.Errorf("child Evidence artifact storeが未初期化です")
	}
	if !allChildArtifactsRetained(lanes) {
		return fmt.Errorf("3 child Evidence artifactのmetadataが揃っていません")
	}

	expectedFiles := map[string]string{
		"deterministic-repository":           "deterministic-repository.json",
		"generated-artifact-reproducibility": "generated-artifact-reproducibility.json",
		"real-postgres-ephemeral-fixture":    "real-postgres-ephemeral-fixture.json",
	}
	seen := make(map[string]struct{}, len(expectedFiles))
	for _, lane := range lanes {
		expectedFile, ok := expectedFiles[lane.Name]
		if !ok {
			return fmt.Errorf("未知のchild Evidence lane=%qです", lane.Name)
		}
		if _, duplicate := seen[lane.Name]; duplicate {
			return fmt.Errorf("child Evidence lane=%qが重複しています", lane.Name)
		}
		seen[lane.Name] = struct{}{}
		if lane.ArtifactFile != expectedFile {
			return fmt.Errorf("%s artifactFile=%q want=%q", lane.Name, lane.ArtifactFile, expectedFile)
		}
		if err := verifyRetainedLaneFile(s.dir, lane); err != nil {
			return err
		}
	}
	if len(seen) != len(expectedFiles) {
		return fmt.Errorf("3 child Evidence laneをfinal保持確認できません")
	}
	return nil
}

func verifyRetainedLaneFile(dir string, lane laneEvidence) error {
	path := filepath.Join(dir, lane.ArtifactFile)
	before, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("%s child Evidence artifactをfinal確認できません: %w", lane.Name, err)
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return fmt.Errorf("%s child Evidence artifactはfinal確認時にregular fileではありません", lane.Name)
	}

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("%s child Evidence artifactをfinal確認用にopenできません: %w", lane.Name, err)
	}
	opened, statErr := file.Stat()
	if statErr != nil {
		_ = file.Close()
		return fmt.Errorf("%s child Evidence artifactをopen後statできません: %w", lane.Name, statErr)
	}
	if !opened.Mode().IsRegular() || !os.SameFile(before, opened) {
		_ = file.Close()
		return fmt.Errorf("%s child Evidence artifactがfinal確認中に置換されました", lane.Name)
	}

	content, readErr := io.ReadAll(file)
	closeErr := file.Close()
	if readErr != nil {
		return fmt.Errorf("%s child Evidence artifactをfinal確認用に読めません: %w", lane.Name, readErr)
	}
	if closeErr != nil {
		return fmt.Errorf("%s child Evidence artifactのfinal確認closeに失敗しました: %w", lane.Name, closeErr)
	}

	after, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("%s child Evidence artifactをfinal読込後に再確認できません: %w", lane.Name, err)
	}
	if after.Mode()&os.ModeSymlink != 0 || !after.Mode().IsRegular() || !os.SameFile(opened, after) {
		return fmt.Errorf("%s child Evidence artifactがfinal読込中に置換されました", lane.Name)
	}

	fingerprint := evidenceArtifactFingerprint(content)
	if fingerprint != lane.EvidenceArtifactFingerprint {
		return fmt.Errorf("%s child Evidence artifactのfinal SHA-256がsame-run fingerprintと一致しません", lane.Name)
	}
	return nil
}
