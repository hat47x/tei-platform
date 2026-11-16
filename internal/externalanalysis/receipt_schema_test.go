package externalanalysis

import (
	"testing"
	"time"
)

// The receipt's schemaVersion is a contract with whoever reads the document, so a
// test must name it (#4032). Receiving bytes is the only way the identifier is
// produced, so the test goes through Receive rather than quoting the constant.
func TestReceiveNamesTheReceiptSchemaVersion(t *testing.T) {
	receipt := Receive([]byte(`{"evidence_id":"ev_1"}`), Tool{Name: "rea"}, time.Unix(0, 0))
	if receipt.SchemaVersion != "tei.externalanalysis.receipt/v0" {
		t.Fatalf("schemaVersion = %q, want tei.externalanalysis.receipt/v0", receipt.SchemaVersion)
	}
	// A missing input still has to carry the identifier: a reader branches on the
	// version before it looks at the state.
	missing := Receive(nil, Tool{Name: "rea"}, time.Unix(0, 0))
	if missing.SchemaVersion != receipt.SchemaVersion {
		t.Errorf("empty input schemaVersion = %q, want %q", missing.SchemaVersion, receipt.SchemaVersion)
	}
}
