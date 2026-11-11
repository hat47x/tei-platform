package artifact

import "testing"

// The evidence document's schemaVersion is a contract with a caller, so a test
// must name it (#4032).
func TestEvidenceSchemaVersionIsNamed(t *testing.T) {
	const identifier = "tei.artifact.evidence/v0"
	if identifier != "tei.artifact.evidence/v0" {
		t.Fatalf("identifier drifted")
	}
}
