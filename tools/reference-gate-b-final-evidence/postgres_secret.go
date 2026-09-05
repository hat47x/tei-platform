package main

import (
	"fmt"
	"os"
	"strings"
)

const postgresDSNFileEnv = "TEI_TEST_POSTGRES_DSN_FILE"

// loadPostgresDSNFilePathFromEnvironmentは、final runner自身も#308自身もcompile段階へDSN本文を
// 持ち込まないため、launch environmentではsecret file pathだけを受け取る。
// file内容はfinal runnerでは読まず、#308 runtimeだけが起動後に読む。
func loadPostgresDSNFilePathFromEnvironment() (string, error) {
	return loadPostgresDSNFilePath(os.Environ())
}

func loadPostgresDSNFilePath(environment []string) (string, error) {
	var paths []string
	for _, entry := range environment {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		if strings.EqualFold(key, postgresDSNEnv) {
			return "", fmt.Errorf("Gate B最終EvidenceではPostgreSQL DSN本文をlaunch environmentへ設定しません。TEI_TEST_POSTGRES_DSN_FILEでsecret fileを指定してください")
		}
		if strings.EqualFold(key, postgresDSNFileEnv) {
			paths = append(paths, value)
		}
	}

	if len(paths) == 0 || paths[0] == "" {
		return "", fmt.Errorf("TEI_TEST_POSTGRES_DSN_FILEが未設定です。#308用DSNを一行だけ保存したRepository外のsecret fileを指定してください")
	}
	if len(paths) != 1 {
		return "", fmt.Errorf("TEI_TEST_POSTGRES_DSN_FILEが大小文字違いを含め複数設定されています。曖昧なsecret file入力は受理しません")
	}
	return paths[0], nil
}
