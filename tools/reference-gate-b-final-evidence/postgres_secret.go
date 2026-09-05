package main

import (
	"fmt"
	"os"
)

const postgresDSNFileEnv = "TEI_TEST_POSTGRES_DSN_FILE"

type secretLookupEnvFunc func(string) (string, bool)

// loadPostgresDSNFilePathFromEnvironmentは、final runner自身も#308自身もcompile段階へDSN本文を
// 持ち込まないため、launch environmentではsecret file pathだけを受け取る。
// file内容はfinal runnerでは読まず、#308 runtimeだけが起動後に読む。
func loadPostgresDSNFilePathFromEnvironment() (string, error) {
	return loadPostgresDSNFilePath(os.LookupEnv)
}

func loadPostgresDSNFilePath(lookupEnv secretLookupEnvFunc) (string, error) {
	if _, exists := lookupEnv(postgresDSNEnv); exists {
		return "", fmt.Errorf("Gate B最終EvidenceではPostgreSQL DSN本文をlaunch environmentへ設定しません。TEI_TEST_POSTGRES_DSN_FILEでsecret fileを指定してください")
	}

	path, exists := lookupEnv(postgresDSNFileEnv)
	if !exists || path == "" {
		return "", fmt.Errorf("TEI_TEST_POSTGRES_DSN_FILEが未設定です。#308用DSNを一行だけ保存したRepository外のsecret fileを指定してください")
	}
	return path, nil
}
