package main

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func lookupFrom(values map[string]string) lookupEnvFunc {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}

func neverReadFile(string) ([]byte, error) {
	return nil, errors.New("読み取り不要のはずです")
}

func TestLoadAdminDSNRequiresExactlyOneInput(t *testing.T) {
	if _, err := loadAdminDSN(lookupFrom(nil), neverReadFile); err == nil {
		t.Fatal("入力がないのに受理しています")
	}
	both := map[string]string{adminDSNEnv: "oracle://u:p@localhost/s", adminDSNFileEnv: "/secret"}
	if _, err := loadAdminDSN(lookupFrom(both), neverReadFile); err == nil || !strings.Contains(err.Error(), "同時") {
		t.Fatalf("direct入力とfile入力の同時指定を拒否していません: %v", err)
	}
	if _, err := loadAdminDSN(lookupFrom(map[string]string{adminDSNEnv: ""}), neverReadFile); err == nil {
		t.Fatal("空のdirect入力を受理しています")
	}
	got, err := loadAdminDSN(lookupFrom(map[string]string{adminDSNEnv: "oracle://u:p@localhost/s"}), neverReadFile)
	if err != nil || got != "oracle://u:p@localhost/s" {
		t.Fatalf("direct入力を取り込めません: %q %v", got, err)
	}
}

func TestLoadAdminDSNFileNormalizationAndErrorsDoNotLeakPath(t *testing.T) {
	read := func(content string) readFileFunc {
		return func(string) ([]byte, error) { return []byte(content), nil }
	}
	env := lookupFrom(map[string]string{adminDSNFileEnv: "/private/dsn"})

	for _, content := range []string{"dsn\n", "dsn\r\n", "dsn"} {
		got, err := loadAdminDSN(env, read(content))
		if err != nil || got != "dsn" {
			t.Fatalf("末尾改行を正規化できません: %q %v", got, err)
		}
	}
	for _, bad := range []string{"", "\n", "a\nb", "a\x00b"} {
		if _, err := loadAdminDSN(env, read(bad)); err == nil {
			t.Fatalf("不正なsecret file内容%qを受理しています", bad)
		}
	}
	_, err := loadAdminDSN(env, func(string) ([]byte, error) { return nil, errors.New("open /private/dsn: denied") })
	if err == nil || strings.Contains(err.Error(), "/private/dsn") {
		t.Fatalf("読み取りerrorがsecret file pathを含んでいます: %v", err)
	}
}

func TestParseAdminDSNAcceptsOnlyLoopbackPlainTCP(t *testing.T) {
	target, err := parseAdminDSN("oracle://SYSTEM:s3cr%40t@127.0.0.1:1521/FREEPDB1")
	if err != nil {
		t.Fatalf("loopbackのDSNを解析できません: %v", err)
	}
	if target.Host != "127.0.0.1" || target.Port != 1521 || target.Service != "FREEPDB1" || target.User != "SYSTEM" || target.Password != "s3cr@t" {
		t.Fatalf("解析結果が想定と異なります: host=%s port=%d service=%s user=%s", target.Host, target.Port, target.Service, target.User)
	}
	for _, ok := range []string{"oracle://u:p@localhost/s", "oracle://u:p@[::1]:1522/s"} {
		if _, err := parseAdminDSN(ok); err != nil {
			t.Fatalf("loopback DSN %qを拒否しています: %v", ok, err)
		}
	}

	rejected := []string{
		"oracle://u:topsecret@db.example.com:1521/s",     // 非loopback
		"oracle://u:topsecret@10.0.0.5:1521/s",           // 非loopback
		"oracle://u:topsecret@localhost:1521/s?ssl=true", // TLS等のoption
		"postgres://u:topsecret@localhost/s",             // 別scheme
		"oracle://u@localhost/s",                         // password無し
		"oracle://u:topsecret@localhost/",                // service無し
		"oracle://u:topsecret@localhost:99999/s",         // port不正
	}
	for _, dsn := range rejected {
		_, err := parseAdminDSN(dsn)
		if err == nil {
			t.Fatalf("DSN %qを受理しています", dsn)
		}
		if strings.Contains(err.Error(), "topsecret") {
			t.Fatalf("解析errorがpasswordを含んでいます: %v", err)
		}
	}
}

func TestClearAdminDSNEnvironmentRemovesBothInputs(t *testing.T) {
	t.Setenv(adminDSNEnv, "x")
	t.Setenv(adminDSNFileEnv, "y")
	if err := clearAdminDSNEnvironment(); err != nil {
		t.Fatal(err)
	}
	if _, ok := os.LookupEnv(adminDSNEnv); ok {
		t.Fatal("direct入力の環境変数が残っています")
	}
	if _, ok := os.LookupEnv(adminDSNFileEnv); ok {
		t.Fatal("secret file入力の環境変数が残っています")
	}
}
