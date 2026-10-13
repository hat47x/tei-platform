package capabilitytransport

import (
    "encoding/base64"
    "go/format"
    "os"
    "testing"
)

func TestTemporaryQuoteSemanticGofmtDiagnostic(t *testing.T) {
    data, err := os.ReadFile("quote_semantic_vectors_test.go")
    if err != nil { t.Fatal(err) }
    formatted, err := format.Source(data)
    if err != nil { t.Fatal(err) }
    t.Fatalf("QUOTE_GOFMT_BASE64:%s", base64.StdEncoding.EncodeToString(formatted))
}
