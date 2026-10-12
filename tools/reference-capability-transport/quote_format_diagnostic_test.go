package capabilitytransport

import (
	"encoding/json"
	"go/format"
	"os"
	"testing"
)

// Temporary exact-source diagnostic. Remove before merging.
func TestQuoteGofmtDiagnostic(t *testing.T) {
	src, err := os.ReadFile("authorized_quote_preflight_test.go")
	if err != nil {
		t.Fatal(err)
	}
	formatted, err := format.Source(src)
	if err != nil {
		t.Fatal(err)
	}
	line, err := json.Marshal(struct {
		Body string `json:"body"`
	}{Body: string(formatted)})
	if err != nil {
		t.Fatal(err)
	}
	t.Fatalf("TEI_GOFMT_JSON:%s", line)
}
