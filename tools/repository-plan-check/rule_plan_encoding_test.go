package plancheck

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"unicode"
	"unicode/utf8"
)

// A concurrent merge once turned the entire Japanese Rule plan into a
// syntactically valid but unreadable UTF-8 mojibake document. Inspect actual
// Markdown bytes, not just the presence of the file or its ASCII headings.
func readableMarkdownBytes(source []byte) error {
	if !utf8.Valid(source) {
		return fmt.Errorf("invalid UTF-8")
	}
	line := 1
	for _, r := range string(source) {
		if r == '\n' {
			line++
			continue
		}
		if r == utf8.RuneError {
			return fmt.Errorf("replacement rune at line %d", line)
		}
		if unicode.IsControl(r) && r != '\t' && r != '\r' {
			return fmt.Errorf("unexpected control rune %U at line %d", r, line)
		}
	}
	return nil
}

func TestRuleVerticalSlicePlanHasReadableUTF8(t *testing.T) {
	path := filepath.Join("..", "..", "plan", "design", "PLUGIN_RULE_PRODUCT_VERTICAL_SLICE_2026-11.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := readableMarkdownBytes(data); err != nil {
		t.Fatalf("Rule vertical-slice design was damaged by a merge or encoding conversion: %v", err)
	}
}

func TestReadableMarkdownBytesRejectsMergeEncodingDamage(t *testing.T) {
	for _, tc := range []struct {
		name  string
		data  []byte
		valid bool
	}{
		{"ordinary-Japanese", []byte("# Ruleから生成Goまで\n説明を読む。\n"), true},
		{"missing-UTF8-sequence", []byte{0xe3, 0x81}, false},
		{"replacement-character", []byte("# 壊れた\ufffd文書\n"), false},
		{"embedded-SUB", []byte("# 文字化け\x1a本文\n"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := readableMarkdownBytes(tc.data)
			if tc.valid && err != nil || !tc.valid && err == nil {
				t.Fatalf("valid=%t, err=%v", tc.valid, err)
			}
		})
	}
}
