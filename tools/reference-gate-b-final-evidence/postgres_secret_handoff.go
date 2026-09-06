package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// loadPostgresDSNFilePathFromEnvironmentは、final runner launch environmentから
// DSN本文を受理せず、secret file pathだけを取得する。
func loadPostgresDSNFilePathFromEnvironment() (string, error) {
	return loadPostgresDSNFilePathFromEntries(os.Environ())
}

// loadPostgresDSNFilePathFromEntriesは、大小文字違いを含むdirect DSN / duplicate file keyを
// secret fileへ触る前にfail closedで拒否するpure seamである。
func loadPostgresDSNFilePathFromEntries(environment []string) (string, error) {
	var filePaths []string
	for _, entry := range environment {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		if strings.EqualFold(key, postgresDSNEnv) {
			return "", fmt.Errorf("Gate B最終EvidenceではPostgreSQL DSN本文をlaunch environmentへ設定しません。TEI_TEST_POSTGRES_DSN_FILEでsecret fileを指定してください")
		}
		if strings.EqualFold(key, postgresDSNFileEnv) {
			filePaths = append(filePaths, value)
		}
	}
	if len(filePaths) == 0 || filePaths[0] == "" {
		return "", fmt.Errorf("TEI_TEST_POSTGRES_DSN_FILEが未設定です。#308用DSNを一行だけ保存したRepository外のsecret fileを指定してください")
	}
	if len(filePaths) != 1 {
		return "", fmt.Errorf("TEI_TEST_POSTGRES_DSN_FILEが大小文字違いを含め複数設定されています。曖昧なsecret file入力は受理しません")
	}
	return filePaths[0], nil
}

// loadPostgresDSNFilePathは既存pure test用seamである。
func loadPostgresDSNFilePath(lookup secretLookupEnvFunc) (string, error) {
	if _, exists := lookup(postgresDSNEnv); exists {
		return "", fmt.Errorf("Gate B最終EvidenceではPostgreSQL DSN本文をlaunch environmentへ設定しません。TEI_TEST_POSTGRES_DSN_FILEでsecret fileを指定してください")
	}
	path, exists := lookup(postgresDSNFileEnv)
	if !exists || path == "" {
		return "", fmt.Errorf("TEI_TEST_POSTGRES_DSN_FILEが未設定です。#308用DSNを一行だけ保存したRepository外のsecret fileを指定してください")
	}
	return path, nil
}

// preparePostgresDSNHandoffは、stable single-openで確定済みのDSNを
// Repository外のowner-only temporary fileへ複製する。
// #308のgo run compileにはこのpathだけを渡し、DSN本文をenvironmentへ載せない。
func preparePostgresDSNHandoff(postgresDSN, repositoryRoot string) (string, func(), error) {
	if postgresDSN == "" {
		return "", nil, fmt.Errorf("PostgreSQL DSN handoffへ空のsecretを渡せません")
	}

	dir, err := os.MkdirTemp("", "tei-gate-b-postgres-secret-")
	if err != nil {
		return "", nil, fmt.Errorf("PostgreSQL DSN handoff directoryを作成できません: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(dir) }

	absDir, err := filepath.Abs(dir)
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("PostgreSQL DSN handoff directoryを絶対pathへ解決できません: %w", err)
	}
	resolvedDir, err := filepath.EvalSymlinks(absDir)
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("PostgreSQL DSN handoff directoryを解決できません: %w", err)
	}
	if pathWithinRepository(repositoryRoot, resolvedDir) {
		cleanup()
		return "", nil, fmt.Errorf("PostgreSQL DSN handoff directoryがRepository内に作成されたため受理しません")
	}
	if err := os.Chmod(resolvedDir, 0o700); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("PostgreSQL DSN handoff directoryをowner-onlyにできません: %w", err)
	}

	path := filepath.Join(resolvedDir, "dsn")
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("PostgreSQL DSN handoff fileを作成できません: %w", err)
	}
	closed := false
	defer func() {
		if !closed {
			_ = file.Close()
		}
	}()
	closeBeforeCleanup := func() {
		if !closed {
			_ = file.Close()
			closed = true
		}
		cleanup()
	}

	if _, err := file.WriteString(postgresDSN); err != nil {
		closeBeforeCleanup()
		return "", nil, fmt.Errorf("PostgreSQL DSN handoff fileへ書き込めません: %w", err)
	}
	if err := file.Sync(); err != nil {
		closeBeforeCleanup()
		return "", nil, fmt.Errorf("PostgreSQL DSN handoff fileを同期できません: %w", err)
	}
	if err := file.Close(); err != nil {
		closed = true
		cleanup()
		return "", nil, fmt.Errorf("PostgreSQL DSN handoff fileをcloseできません: %w", err)
	}
	closed = true

	return path, cleanup, nil
}

func runLaneWithPostgresDSNFile(spec laneSpec, candidateRevision, postgresDSNFilePath string) (laneEvidence, error) {
	return runLaneWithPostgresDSNFileAndArtifactWriter(spec, candidateRevision, postgresDSNFilePath, nil)
}

func runLaneWithPostgresDSNFileAndArtifactWriter(
	spec laneSpec,
	candidateRevision,
	postgresDSNFilePath string,
	artifactWriter laneArtifactWriterFunc,
) (laneEvidence, error) {
	return runLaneWithEnvironmentAndArtifactWriter(
		spec,
		candidateRevision,
		environmentForLaneWithPostgresDSNFile(spec, os.Environ(), postgresDSNFilePath),
		artifactWriter,
	)
}

func environmentForLaneWithPostgresDSNFile(spec laneSpec, base []string, postgresDSNFilePath string) []string {
	out := environmentWithoutPostgresSecrets(base)
	if spec.Name == "real-postgres-ephemeral-fixture" {
		out = append(out, postgresDSNFileEnv+"="+postgresDSNFilePath)
	}
	return out
}
