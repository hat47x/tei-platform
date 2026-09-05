package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func readExternalRegularFileStable(path, repositoryRoot string) ([]byte, error) {
	if path == "" {
		return nil, fmt.Errorf("pathが空です")
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("fileをopenできません: %w", err)
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
	if !openedInfo.Mode().IsRegular() {
		return nil, fmt.Errorf("openしたfileがregular fileではありません")
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

func validateOpenedExternalFile(path, repositoryRoot string, openedInfo os.FileInfo) error {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return fmt.Errorf("pathを解決できません: %w", err)
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return fmt.Errorf("resolved pathを絶対pathへ解決できません: %w", err)
	}
	if pathWithinRepository(repositoryRoot, resolved) {
		return fmt.Errorf("Repository内のfileはsecret inputとして受理しません")
	}

	currentInfo, err := os.Stat(resolved)
	if err != nil {
		return fmt.Errorf("resolved fileをstatできません: %w", err)
	}
	if !currentInfo.Mode().IsRegular() {
		return fmt.Errorf("resolved fileがregular fileではありません")
	}
	if !os.SameFile(openedInfo, currentInfo) {
		return fmt.Errorf("secret file pathがopen済みfileとは別identityへ置き換わりました")
	}
	return nil
}
