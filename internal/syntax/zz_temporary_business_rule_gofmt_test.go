package syntax

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestZZTemporaryBusinessRuleGofmtDiagnostic(t *testing.T) {
	for _, name := range []string{
		"semantic_business_rules.go",
		filepath.Join("..", "validation", "semantic_business_rules_test.go"),
	} {
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		expected, err := format.Source(raw)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Equal(raw, expected) {
			continue
		}
		have := strings.Split(string(raw), "\n")
		want := strings.Split(string(expected), "\n")
		for i := 0; i < len(have) && i < len(want); i++ {
			if have[i] != want[i] {
				t.Error(fmt.Sprintf("gofmt %s line %d: have=%q want=%q", name, i+1, have[i], want[i]))
			}
		}
	}
}
