package exampleslide

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"strings"
	"testing"
)

func TestZZTemporaryRuleDraftGofmtDiagnosis(t *testing.T) {
	raw, err := os.ReadFile("model_rule_draft_test.go")
	if err != nil {
		t.Fatal(err)
	}
	formatted, err := format.Source(raw)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(raw, formatted) {
		return
	}
	actualLines := strings.Split(string(raw), "\n")
	wantLines := strings.Split(string(formatted), "\n")
	for i := 0; i < len(actualLines) && i < len(wantLines); i++ {
		if actualLines[i] != wantLines[i] {
			t.Error(fmt.Sprintf("gofmt line %d: have %q want %q", i+1, actualLines[i], wantLines[i]))
		}
	}
}
