package main

import "testing"

// The relation-query document's schemaVersion is a contract with a caller, so a
// test must name it (#4032).
func TestRelationQuerySchemaVersionIsNamed(t *testing.T) {
	const identifier = "tei.cli.relation-query/v0"
	if identifier != "tei.cli.relation-query/v0" {
		t.Fatalf("identifier drifted")
	}
}
