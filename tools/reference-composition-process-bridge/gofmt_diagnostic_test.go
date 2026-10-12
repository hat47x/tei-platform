package compositionprocessbridge

import (
	"go/format"
	"os"
	"strings"
	"testing"
)

// TEMPORARY: reports the exact gofmt delta on the connected CI runner.
// Removed once the production test file has been normalized.
func TestReferenceFormattingDiagnostic(t *testing.T) {
	source, err := os.ReadFile("web_bridge_test.go")
	if err != nil {
		t.Fatal(err)
	}
	expected, err := format.Source(source)
	if err != nil {
		t.Fatal(err)
	}
	left, right := strings.Split(string(source), "\n"), strings.Split(string(expected), "\n")
	if string(source) == string(expected) {
		return
	}
	t.Logf("file lines=%d formatted lines=%d", len(left), len(right))
	for i := 0; i < len(left) && i < len(right); i++ {
		if left[i] != right[i] {
			t.Logf("format line %d: present %q; expected %q", i+1, left[i], right[i])
		}
	}
	t.Fatal("temporary formatting probe detected mismatch")
}
