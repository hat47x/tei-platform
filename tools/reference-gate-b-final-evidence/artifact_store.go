package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type laneArtifactStore struct {
	dir  string
	auto bool
}

func inspectRepositoryRoot(output commandOutputFunc) (string, error) {
	root, err := output("git", "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("Repository rootを確認できません: %w", err)
	}
	root = strings.TrimSpace(root)
	if root == "" {
		return "", fmt.Errorf("Repository rootが空です")
	}

	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("Repository rootを解決できません: %w", err)
	}
	resolvedRoot, err = filepath.Abs(resolvedRoot)
	if err != nil {
		return "", fmt.Errorf("Repository rootを絶対pathへ解決できません: %w", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("current working directoryを確認できません: %w", err)
	}
	resolvedCWD, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		return "", fmt.Errorf("current working directoryを解決できません: %w", err)
	}
	resolvedCWD, err = filepath.Abs(resolvedCWD)
	if err != nil {
		return "", fmt.Errorf("current working directoryを絶対pathへ解決できません: %w", err)
	}

	rootInfo, err := os.Stat(resolvedRoot)
	if err != nil {
		return "", fmt.Errorf("Repository rootをstatできません: %w", err)
	}
	cwdInfo, err := os.Stat(resolvedCWD)
	if err != nil {
		return "", fmt.Errorf("current working directoryをstatできません: %w", err)
	}
	if !os.SameFile(rootInfo, cwdInfo) {
		return "", fmt.Errorf("Gate B最終EvidenceはRepository rootから実行してください")
	}
	return filepath.Clean(resolvedRoot), nil
}

func prepareLaneArtifactStore(requestedDir, repositoryRoot string) (laneArtifactStore, error) {
	if requestedDir == "" {
		base, err := filepath.EvalSymlinks(os.TempDir())
		if err != nil {
			return laneArtifactStore{}, fmt.Errorf("temporary directoryを解決できません: %w", err)
		}
		base, err = filepath.Abs(base)
		if err != nil {
			return laneArtifactStore{}, fmt.Errorf("temporary directoryを絶対pathへ解決できません: %w", err)
		}
		if pathWithinRepository(repositoryRoot, base) {
			return laneArtifactStore{}, fmt.Errorf("OS temporary directoryがRepository内にあるためchild Evidenceを安全に保持できません。-artifact-dirでRepository外のdirectoryを指定してください")
		}
		dir, err := os.MkdirTemp(base, "tei-gate-b-final-evidence-")
		if err != nil {
			return laneArtifactStore{}, fmt.Errorf("child Evidence用temporary directoryを作成できません: %w", err)
		}
		if err := os.Chmod(dir, 0o700); err != nil {
			_ = os.Remove(dir)
			return laneArtifactStore{}, fmt.Errorf("child Evidence directoryをowner-onlyへ制限できません: %w", err)
		}
		return laneArtifactStore{dir: dir, auto: true}, nil
	}

	abs, err := filepath.Abs(requestedDir)
	if err != nil {
		return laneArtifactStore{}, fmt.Errorf("-artifact-dirを絶対pathへ解決できません: %w", err)
	}
	parent := filepath.Dir(abs)
	resolvedParent, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return laneArtifactStore{}, fmt.Errorf("-artifact-dirのparent directoryを解決できません: %w", err)
	}
	resolvedParent, err = filepath.Abs(resolvedParent)
	if err != nil {
		return laneArtifactStore{}, fmt.Errorf("-artifact-dirのparent directoryを絶対pathへ解決できません: %w", err)
	}
	resolvedDir := filepath.Join(resolvedParent, filepath.Base(abs))
	if pathWithinRepository(repositoryRoot, resolvedDir) {
		return laneArtifactStore{}, fmt.Errorf("child Evidence directoryをRepository内へ作成しません。-artifact-dirはRepository外を指定してください")
	}
	if err := os.Mkdir(resolvedDir, 0o700); err != nil {
		return laneArtifactStore{}, fmt.Errorf("child Evidence directoryを新規作成できません: %w", err)
	}
	return laneArtifactStore{dir: resolvedDir}, nil
}

func (s laneArtifactStore) writeLane(spec laneSpec, content []byte) (string, error) {
	if s.dir == "" {
		return "", fmt.Errorf("child Evidence artifact storeが未初期化です")
	}
	if len(content) == 0 {
		return "", fmt.Errorf("%s child Evidence stdoutが空です", spec.Name)
	}

	filename := spec.Name + ".json"
	path := filepath.Join(s.dir, filename)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", fmt.Errorf("%s child Evidence artifactを新規作成できません: %w", spec.Name, err)
	}
	keep := false
	defer func() {
		_ = file.Close()
		if !keep {
			_ = os.Remove(path)
		}
	}()

	if _, err := file.Write(content); err != nil {
		return "", fmt.Errorf("%s child Evidence artifactを書き込めません: %w", spec.Name, err)
	}
	if err := file.Sync(); err != nil {
		return "", fmt.Errorf("%s child Evidence artifactをsyncできません: %w", spec.Name, err)
	}
	if err := file.Close(); err != nil {
		return "", fmt.Errorf("%s child Evidence artifactをcloseできません: %w", spec.Name, err)
	}
	keep = true
	return filename, nil
}

func ensureExternalExistingRegularFile(path, repositoryRoot string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("pathが空です")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("pathを絶対pathへ解決できません: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("pathを解決できません: %w", err)
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("resolved pathを絶対pathへ解決できません: %w", err)
	}
	if pathWithinRepository(repositoryRoot, resolved) {
		return "", fmt.Errorf("Repository内のfileはsecret inputとして受理しません")
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("fileをstatできません: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("regular fileではありません")
	}
	return resolved, nil
}

func pathWithinRepository(repositoryRoot, target string) bool {
	root := filepath.Clean(repositoryRoot)
	target = filepath.Clean(target)
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func allChildArtifactsRetained(lanes []laneEvidence) bool {
	if len(lanes) != 3 {
		return false
	}
	for _, lane := range lanes {
		if !lane.Passed || lane.ArtifactFile == "" || lane.EvidenceArtifactFingerprint == "" {
			return false
		}
	}
	return true
}
