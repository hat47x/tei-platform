package main

import (
	"io"
	"strings"
	"testing"
)

func TestParseOptionsRequiresConfirmationBeforeAnything(t *testing.T) {
	for _, args := range [][]string{
		nil,
		{"-cleanup", "0123ABCD"},
		{"-workspace", "reference/workspace"},
	} {
		if _, err := parseOptions(args, io.Discard); err == nil || !strings.Contains(err.Error(), confirmFlag) {
			t.Fatalf("確認flagなしの%vが拒否されていません: %v", args, err)
		}
	}
	if _, err := parseOptions([]string{"-" + confirmFlag}, io.Discard); err != nil {
		t.Fatalf("確認flagありの既定実行が拒否されました: %v", err)
	}
}

func TestParseOptionsEvidenceOutNameAndRunID(t *testing.T) {
	confirm := "-" + confirmFlag
	if _, err := parseOptions([]string{confirm, "-evidence-out", "/tmp/evidence.json"}, io.Discard); err == nil {
		t.Fatal("検証用データ由来と分かる名前を含まない出力先が受理されました")
	}
	if _, err := parseOptions([]string{confirm, "-evidence-out", "/tmp/oracle-local-fixture-evidence.json"}, io.Discard); err != nil {
		t.Fatalf("正しい出力名が拒否されました: %v", err)
	}
	if _, err := parseOptions([]string{confirm, "-cleanup", "xyz"}, io.Discard); err == nil {
		t.Fatal("不正な実行IDが受理されました")
	}
	if got, err := parseOptions([]string{confirm, "-cleanup", "0123ABCD"}, io.Discard); err != nil || got.CleanupRun != "0123ABCD" {
		t.Fatalf("正しい実行IDが受理されません: %v", err)
	}
	if _, err := parseOptions([]string{confirm, "extra"}, io.Discard); err == nil {
		t.Fatal("余分な引数が受理されました")
	}
}
