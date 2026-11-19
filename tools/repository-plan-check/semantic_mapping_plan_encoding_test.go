package plancheck

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A concurrent merge once transformed a readable Japanese design document
// into valid UTF-8 mojibake with embedded control characters. Checking a
// generic UTF-8 decoder alone cannot protect the title, sections or meaning.
func semanticMappingPlanReadable(raw []byte) error {
	if err := readableMarkdownBytes(raw); err != nil {
		return fmt.Errorf("semantic structure mapping is not readable: %w", err)
	}
	body := string(raw)
	const title = "# 標準データ構造の意味的な対応付けと、構文・図解・編集の共通設計\n"
	if !strings.HasPrefix(body, title) {
		return fmt.Errorf("the Japanese semantic structure mapping title was damaged")
	}
	sections := []string{
		"## 1. ",
		"## 37. 標準データ様式から実際の処理・表現へ進む最小の一往復",
		"## 39. 共通Source契約と異なるデータ様式を、一つの実行可能な受入手順にまとめる",
		"## 40. 宣言的な記述を、実際に使える機能までつなぐ",
	}
	start := 0
	for _, heading := range sections {
		offset := strings.Index(body[start:], heading)
		if offset < 0 {
			return fmt.Errorf("semantic structure mapping lost its section %q", heading)
		}
		start += offset + len(heading)
	}
	return nil
}

func TestSemanticMappingPlanRestoredJapaneseAndSectionIdentity(t *testing.T) {
	path := filepath.Join(repositoryRoot(t), "plan", "design",
		"SEMANTIC_STRUCTURE_MAPPING_AND_DECLARATIVE_SYNTAX_VIEW_2026-11.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := semanticMappingPlanReadable(raw); err != nil {
		t.Fatal(err)
	}
}

func TestSemanticMappingPlanGuardRejectsEncodingAndMergeRegression(t *testing.T) {
	valid := []byte("# 標準データ構造の意味的な対応付けと、構文・図解・編集の共通設計\n" +
		"## 1. 意味と表現を区別する\n" +
		"## 37. 標準データ様式から実際の処理・表現へ進む最小の一往復\n" +
		"## 39. 共通Source契約と異なるデータ様式を、一つの実行可能な受入手順にまとめる\n" +
		"## 40. 宣言的な記述を、実際に使える機能までつなぐ\n")
	cases := []struct {
		name string
		raw  []byte
		ok   bool
	}{
		{"intact", valid, true},
		{"mojibake-title-in-valid-UTF8", []byte(strings.Replace(string(valid), "標準データ構造の意味的な対応付け", "讓呎ｺ悶ョ繝ｼ繧ｿ", 1)), false},
		{"embedded-control", append(append([]byte(nil), valid...), 0x1a), false},
		{"replacement-rune", []byte(string(valid) + "\ufffd"), false},
		{"missing-latest-section", []byte(strings.Replace(string(valid), "## 40. 宣言的な記述を、実際に使える機能までつなぐ\n", "", 1)), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := semanticMappingPlanReadable(tc.raw)
			if (err == nil) != tc.ok {
				t.Fatalf("accepted=%t, want %t, err=%v", err == nil, tc.ok, err)
			}
		})
	}
}
