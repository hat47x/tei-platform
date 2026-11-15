package plancheck

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

// checkRulePluginPlanText guards against a Windows-encoded conflict resolution
// writing mojibake back over the Japanese plan. It checks byte validity,
// printable text, and the two stable section identities; it does not judge
// the content's business meaning or substitute for an Owner review.
func checkRulePluginPlanText(raw []byte) error {
	if !utf8.Valid(raw) {
		return fmt.Errorf("Rule-to-Plugin plan is not valid UTF-8")
	}
	for offset, r := range string(raw) {
		if (r < 0x20 && r != '\n' && r != '\r' && r != '\t') || r == 0x7f || r == utf8.RuneError {
			return fmt.Errorf("Rule-to-Plugin plan has a non-text character at byte offset %d: %U", offset, r)
		}
	}
	for _, heading := range []string{
		"# Rule Authoringから実Plugin・製品Consumerまでの第一縦断",
		"### 14.9 従来構文と選択済みRule Markdownを混ぜずに診断する",
		"### 14.10 生成Ruleを通常CLIのBuild Intent経路で実ビルドする",
	} {
		if !strings.Contains(string(raw), heading) {
			return fmt.Errorf("Rule-to-Plugin plan has lost its readable section identity %q", heading)
		}
	}
	return nil
}

func TestRulePluginVerticalPlanKeepsReadableJapanese(t *testing.T) {
	path := filepath.Join(repositoryRoot(t), "plan", "design", "PLUGIN_RULE_PRODUCT_VERTICAL_SLICE_2026-11.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := checkRulePluginPlanText(raw); err != nil {
		t.Fatal(err)
	}
}

func TestRulePluginPlanEncodingGuardRejectsCorruption(t *testing.T) {
	const readable = "# Rule Authoringから実Plugin・製品Consumerまでの第一縦断\n" +
		"### 14.9 従来構文と選択済みRule Markdownを混ぜずに診断する\n" +
		"### 14.10 生成Ruleを通常CLIのBuild Intent経路で実ビルドする\n"
	for _, tc := range []struct {
		name string
		data []byte
		want string
	}{
		{"intact", []byte(readable), ""},
		{"invalid UTF-8", append([]byte(readable), 0xff), "valid UTF-8"},
		{"control byte in a valid UTF-8 string", []byte(readable + "\x1a"), "non-text character"},
		{"mojibake title but still valid UTF-8", []byte(strings.Replace(readable, "Rule Authoringから", "Rule Authoring縺九ｉ", 1)), "readable section identity"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := checkRulePluginPlanText(tc.data)
			if tc.want == "" && err != nil {
				t.Fatalf("intact Japanese was rejected: %v", err)
			}
			if tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("unexpected verdict for %s: %v, want %q", tc.name, err, tc.want)
			}
		})
	}
}
