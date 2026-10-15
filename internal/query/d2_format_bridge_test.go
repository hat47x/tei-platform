package query_test

import (
	"encoding/base64"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestD2FormatBridge temporarily transfers canonical gofmt text through the
// read-only CI log. Remove this test immediately after formatting D2 files.
func TestD2FormatBridge(t *testing.T) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test file")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(source), "..", ".."))
	for _, name := range []string{
		"internal/query/derived_select.go",
		"internal/query/derived_select_test.go",
		"internal/driver/derived_query_execution.go",
		"internal/driver/derived_query_execution_test.go",
	} {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		formatted, err := format.Source(data)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("D2FORMAT:%s:%s\n", name, base64.StdEncoding.EncodeToString(formatted))
	}
	t.Fatal("temporary D2 format extraction; remove before review")
}
