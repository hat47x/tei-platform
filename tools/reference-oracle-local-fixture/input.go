package main

import (
	"bytes"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// adminDSNEnvとadminDSNFileEnvは、使い捨てprincipalを作成できる管理接続のHost側入力である。
// どちらか一方だけを指定し、値はRepository、receipt、標準出力、標準エラーへ出さない。
const (
	adminDSNEnv     = "TEI_ORACLE_FIXTURE_ADMIN_DSN"
	adminDSNFileEnv = "TEI_ORACLE_FIXTURE_ADMIN_DSN_FILE"
)

type lookupEnvFunc func(string) (string, bool)
type readFileFunc func(string) ([]byte, error)

// loadAdminDSNは、direct入力かsecret file入力のどちらか一方だけを受け付ける。
func loadAdminDSN(lookupEnv lookupEnvFunc, readFile readFileFunc) (string, error) {
	direct, directSet := lookupEnv(adminDSNEnv)
	path, fileSet := lookupEnv(adminDSNFileEnv)

	if directSet && fileSet {
		return "", fmt.Errorf("管理DSNのdirect入力とsecret file入力が同時に設定されています。どちらか一方だけを指定してください")
	}
	if directSet {
		if direct == "" {
			return "", fmt.Errorf("%sが空です", adminDSNEnv)
		}
		return direct, nil
	}
	if !fileSet || path == "" {
		return "", fmt.Errorf("%sまたは%sのどちらか一方を指定してください", adminDSNEnv, adminDSNFileEnv)
	}
	content, err := readFile(path)
	if err != nil {
		// underlying errorはlocal secret file pathを含み得るため複製しない。
		return "", fmt.Errorf("管理DSN secret fileを読み取れません")
	}
	return normalizeSecretFile(content)
}

func normalizeSecretFile(content []byte) (string, error) {
	if bytes.IndexByte(content, 0) >= 0 {
		return "", fmt.Errorf("管理DSN secret fileにNULが含まれているため受理しません")
	}
	switch {
	case bytes.HasSuffix(content, []byte("\r\n")):
		content = content[:len(content)-2]
	case bytes.HasSuffix(content, []byte("\n")):
		content = content[:len(content)-1]
	}
	if len(content) == 0 {
		return "", fmt.Errorf("管理DSN secret fileが空です")
	}
	if bytes.ContainsAny(content, "\r\n") {
		return "", fmt.Errorf("管理DSN secret fileは一行でなければなりません")
	}
	return string(content), nil
}

// clearAdminDSNEnvironmentは、DSNをGo値へ取り込んだ後にsubprocessへcredentialやsecret file pathを継承しないために呼ぶ。
func clearAdminDSNEnvironment() error {
	for _, key := range []string{adminDSNEnv, adminDSNFileEnv} {
		if err := os.Unsetenv(key); err != nil {
			return fmt.Errorf("管理DSN用の環境変数を消去できません")
		}
	}
	return nil
}

// adminTargetは、管理DSNから取り出した接続先とcredentialである。
// Passwordを含むため、このstructをそのままJSONや診断へ出さない。
type adminTarget struct {
	Host     string
	Port     int
	Service  string
	User     string
	Password string
}

// parseAdminDSNは`oracle://user:password@host:port/service`だけを受理する。
//
// このtoolは新しいOracle userを作成するため、他者の資産へ向けた誤接続を避ける目的で、
// loopback host以外・query option（TLS等）付きのDSNを拒否する。
// 解析errorはDSN本文を含めない。
func parseAdminDSN(dsn string) (adminTarget, error) {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return adminTarget{}, fmt.Errorf("管理DSNをURLとして解析できません")
	}
	if parsed.Scheme != "oracle" {
		return adminTarget{}, fmt.Errorf("管理DSNのschemeはoracleでなければなりません")
	}
	if parsed.RawQuery != "" {
		return adminTarget{}, fmt.Errorf("管理DSNにquery optionは指定できません。local fixtureはTCP平文のloopback接続だけを扱います")
	}
	if parsed.User == nil {
		return adminTarget{}, fmt.Errorf("管理DSNにuserとpasswordが必要です")
	}
	password, hasPassword := parsed.User.Password()
	if parsed.User.Username() == "" || !hasPassword || password == "" {
		return adminTarget{}, fmt.Errorf("管理DSNにuserとpasswordが必要です")
	}
	host := parsed.Hostname()
	if !isLoopbackHost(host) {
		return adminTarget{}, fmt.Errorf("管理DSNのhostはloopbackでなければなりません。このtoolは他者のOracle資産へ接続しません")
	}
	port := 1521
	if rawPort := parsed.Port(); rawPort != "" {
		port, err = strconv.Atoi(rawPort)
		if err != nil || port <= 0 || port > 65535 {
			return adminTarget{}, fmt.Errorf("管理DSNのportが不正です")
		}
	}
	service := strings.TrimPrefix(parsed.Path, "/")
	if service == "" || strings.Contains(service, "/") {
		return adminTarget{}, fmt.Errorf("管理DSNにservice名が必要です")
	}
	return adminTarget{
		Host:     host,
		Port:     port,
		Service:  service,
		User:     parsed.User.Username(),
		Password: password,
	}, nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
