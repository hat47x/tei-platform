package capabilitytransport

import (
	"fmt"
	"go/format"
	"os"
	"strings"
	"testing"
)

// Temporary: report exact formatter differences. Remove before merging.
func TestTemporaryQuoteRuntimeGofmtDiff(t *testing.T) {
	src, err := os.ReadFile("runtime_authorized_quote_process_test.go")
	if err != nil { t.Fatal(err) }
	dst, err := format.Source(src)
	if err != nil { t.Fatal(err) }
	a, b := strings.Split(string(src), "\n"), strings.Split(string(dst), "\n")
	var differences []string
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] { differences = append(differences, fmt.Sprintf("L%d: BEFORE: %q AFTER: %q", i+1, a[i], b[i])) }
	}
	if len(differences) != 0 { t.Fatal(strings.Join(differences, "\n")) }
}
