package main

import (
	"bytes"
	"fmt"
	"os"
)

const postgresDSNFileEnv = "TEI_TEST_POSTGRES_DSN_FILE"

type secretLookupEnvFunc func(string) (string, bool)
type readSecretFileFunc func(string) ([]byte, error)
type externalPostgresSecretLoaderFunc func(string) (string, error)

// loadPostgresDSNFromEnvironmentは、既存pure test互換のproduction wrapperである。
// Repository外pathの強制が必要なfinal runner本体では
// loadPostgresDSNFromEnvironmentOutsideRepositoryを利用する。
func loadPostgresDSNFromEnvironment() (string, error) {
	return loadPostgresDSNSecret(os.LookupEnv, os.ReadFile)
}

// loadPostgresDSNFromEnvironmentOutsideRepositoryは、final runner自身のcompile段階へDSN本文を持ち込まず、
// launch environmentにはRepository外のsecret file pathだけを受け取る。
// secret fileへ触る前にRepository sourceがcleanであることを確認し、その後だけ
// Repository外の同一regular file identityからDSN本文をsingle-openで読み取る。
func loadPostgresDSNFromEnvironmentOutsideRepository(repositoryRoot string) (string, error) {
	return loadPostgresDSNAfterCleanRepositoryPreflight(
		repositoryRoot,
		inspectRepositoryStart,
		loadPostgresDSNFromEnvironmentOutsideRepositoryAfterSourcePreflight,
	)
}

// loadPostgresDSNAfterCleanRepositoryPreflightは、dirty / source観測不能なcheckoutで
// secret fileをopen/readしないためのpure test seamである。
func loadPostgresDSNAfterCleanRepositoryPreflight(
	repositoryRoot string,
	inspect inspectStartFunc,
	load externalPostgresSecretLoaderFunc,
) (string, error) {
	source, err := inspect()
	if err != nil {
		return "", fmt.Errorf("PostgreSQL DSN secret fileを読む前にclean Repository sourceを確認できません")
	}
	if !source.WorkingTreeClean {
		return "", fmt.Errorf("PostgreSQL DSN secret fileを読む前にworking treeをcleanにしてください")
	}
	return load(repositoryRoot)
}

// loadPostgresDSNFromEnvironmentOutsideRepositoryAfterSourcePreflightは、
// clean-source preflight後のsecret-file boundaryだけを検証するpure seamである。
func loadPostgresDSNFromEnvironmentOutsideRepositoryAfterSourcePreflight(repositoryRoot string) (string, error) {
	if _, exists := os.LookupEnv(postgresDSNEnv); exists {
		return "", fmt.Errorf("Gate B最終EvidenceではPostgreSQL DSN本文をlaunch environmentへ設定しません。TEI_TEST_POSTGRES_DSN_FILEでsecret fileを指定してください")
	}

	path, exists := os.LookupEnv(postgresDSNFileEnv)
	if !exists || path == "" {
		return "", fmt.Errorf("TEI_TEST_POSTGRES_DSN_FILEが未設定です。#308用DSNを一行だけ保存したRepository外のsecret fileを指定してください")
	}
	content, err := readExternalRegularFileStable(path, repositoryRoot)
	if err != nil {
		// underlying errorはlocal pathやsymlink先を含み得るため通常診断へ複製しない。
		return "", fmt.Errorf("PostgreSQL DSN secret fileはRepository外にある安定したexisting regular fileとして読み取れなければなりません")
	}
	return normalizePostgresDSNSecret(content)
}

// loadPostgresDSNSecretはpure test seamである。
// direct DSN環境変数が存在するrunは、値が空でも正規経路から外れているためfail closedで拒否する。
func loadPostgresDSNSecret(lookupEnv secretLookupEnvFunc, readFile readSecretFileFunc) (string, error) {
	if _, exists := lookupEnv(postgresDSNEnv); exists {
		return "", fmt.Errorf("Gate B最終EvidenceではPostgreSQL DSN本文をlaunch environmentへ設定しません。TEI_TEST_POSTGRES_DSN_FILEでsecret fileを指定してください")
	}

	path, exists := lookupEnv(postgresDSNFileEnv)
	if !exists || path == "" {
		return "", fmt.Errorf("TEI_TEST_POSTGRES_DSN_FILEが未設定です。#308用DSNを一行だけ保存したRepository外のsecret fileを指定してください")
	}

	content, err := readFile(path)
	if err != nil {
		// underlying errorはlocal pathを含み得るため通常診断へ複製しない。
		return "", fmt.Errorf("PostgreSQL DSN secret fileを読み取れません")
	}
	return normalizePostgresDSNSecret(content)
}

func normalizePostgresDSNSecret(content []byte) (string, error) {
	if bytes.IndexByte(content, 0) >= 0 {
		return "", fmt.Errorf("PostgreSQL DSN secret fileにNULが含まれているため受理しません")
	}

	// editorやprintf等が付ける末尾の単一LF / CRLFだけは許容する。
	// それ以外のCR/LFは複数行secretとして拒否する。
	switch {
	case bytes.HasSuffix(content, []byte("\r\n")):
		content = content[:len(content)-2]
	case bytes.HasSuffix(content, []byte("\n")):
		content = content[:len(content)-1]
	}

	if len(content) == 0 {
		return "", fmt.Errorf("PostgreSQL DSN secret fileが空です")
	}
	if bytes.ContainsAny(content, "\r\n") {
		return "", fmt.Errorf("PostgreSQL DSN secret fileは一行でなければなりません")
	}
	return string(content), nil
}
