package main

import (
	"bytes"
	"fmt"
	"os"
)

const postgresDSNEnv = "TEI_TEST_POSTGRES_DSN"
const postgresDSNFileEnv = "TEI_TEST_POSTGRES_DSN_FILE"

type dsnLookupEnvFunc func(string) (string, bool)
type dsnReadFileFunc func(string) ([]byte, error)

func loadPostgresDSNFromEnvironment() (string, error) {
	return loadPostgresDSNInput(os.LookupEnv, os.ReadFile)
}

func loadPostgresDSNInput(lookupEnv dsnLookupEnvFunc, readFile dsnReadFileFunc) (string, error) {
	direct, directSet := lookupEnv(postgresDSNEnv)
	path, fileSet := lookupEnv(postgresDSNFileEnv)

	if directSet && fileSet {
		return "", fmt.Errorf("PostgreSQL DSNのdirect入力とsecret file入力が同時に設定されています。どちらか一方だけを指定してください")
	}
	if directSet {
		if direct == "" {
			return "", fmt.Errorf("TEI_TEST_POSTGRES_DSNが空です")
		}
		return direct, nil
	}
	if !fileSet || path == "" {
		return "", fmt.Errorf("TEI_TEST_POSTGRES_DSNまたはTEI_TEST_POSTGRES_DSN_FILEのどちらか一方を指定してください")
	}

	content, err := readFile(path)
	if err != nil {
		// underlying errorはlocal secret file pathを含み得るため通常診断へ複製しない。
		return "", fmt.Errorf("PostgreSQL DSN secret fileを読み取れません")
	}
	return normalizePostgresDSNFile(content)
}

func normalizePostgresDSNFile(content []byte) (string, error) {
	if bytes.IndexByte(content, 0) >= 0 {
		return "", fmt.Errorf("PostgreSQL DSN secret fileにNULが含まれているため受理しません")
	}

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

// clearPostgresDSNEnvironmentはDSNをGo valueへ取り込んだ後、Repository source probeや
// その後の不要なsubprocessへcredential候補やsecret file pathを継承しないために呼ぶ。
func clearPostgresDSNEnvironment() error {
	if err := os.Unsetenv(postgresDSNEnv); err != nil {
		return fmt.Errorf("PostgreSQL DSN環境変数を消去できません: %w", err)
	}
	if err := os.Unsetenv(postgresDSNFileEnv); err != nil {
		return fmt.Errorf("PostgreSQL DSN secret file環境変数を消去できません: %w", err)
	}
	return nil
}
