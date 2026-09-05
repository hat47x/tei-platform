package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func readExternalRegularFileStable(path, repositoryRoot string) ([]byte, error) {
	resolvedPath, expectedInfo, err := resolveExternalRegularFile(path, repositoryRoot)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("validated fileをopenできません: %w", err)
	}
	closed := false
	defer func() {
		if !closed {
			_ = file.Close()
		}
	}()

	openedInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("openしたfileをstatできません: %w", err)
	}
	if !openedInfo.Mode().IsRegular() || !os.SameFile(expectedInfo, openedInfo) {
		return nil, fmt.Errorf("secret fileが事前確認したregular file identityから置き換わりました")
	}
	if err := validateOpenedExternalFile(path, repositoryRoot, openedInfo); err != nil {
		return nil, err
	}

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("openしたfileを読めません: %w", err)
	}

	if err := validateOpenedExternalFile(path, repositoryRoot, openedInfo); err != nil {
		return nil, err
	}
	if err := file.Close(); err != nil {
		return nil, fmt.Errorf("openしたfileをcloseできません: %w", err)
	}
	closed = true
	return content, nil
}

func resolveExternalRegularFile(path, repositoryRoot string) (string, os.FileInfo, error) {
	if path == "" {
		return "", nil, fmt.Errorf("pathが空です")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", nil, fmt.Errorf("pathを解決できません: %w", err)
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return "", nil, fmt.Errorf("resolved pathを絶対pathへ解決できません: %w", err)
	}
	if pathWithinRepository(repositoryRoot, resolved) {
		return "", nil, fmt.Errorf("Repository内のfileはsecret inputとして受理しません")
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", nil, fmt.Errorf("resolved fileをstatできません: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", nil, fmt.Errorf("resolved fileがregular fileではありません")
	}
	return resolved, info, nil
}

func validateOpenedExternalFile(path, repositoryRoot string, openedInfo os.FileInfo) error {
	_, currentInfo, err := resolveExternalRegularFile(path, repositoryRoot)
	if err != nil {
		return err
	}
	if !os.SameFile(openedInfo, currentInfo) {
		return fmt.Errorf("secret file pathがopen済みfileとは別identityへ置き換わりました")
	}
	return nil
}
