package standard

import (
	"testing"
)

// TestStandardToSQLRefusesANilQuery checks the refusal a caller sees when no query
// is given. The code was declared but no test named it, so the repository's coverage
// ratchet (tools/repository-plan-check) failed on it.
func TestStandardToSQLRefusesANilQuery(t *testing.T) {
	sql, diagnostics := StandardToSQL(nil)
	if sql != "" {
		t.Fatalf("a nil query produced SQL: %q", sql)
	}
	if len(diagnostics) != 1 {
		t.Fatalf("diagnostics=%+v", diagnostics)
	}
	if diagnostics[0].Code != "EMPTY_QUERY" {
		t.Fatalf("code=%s", diagnostics[0].Code)
	}
}

// TestParseSQLToStandardRefusesEmptySource checks the other declared refusal: an
// empty source is reported rather than parsed into an empty query.
func TestParseSQLToStandardRefusesEmptySource(t *testing.T) {
	query, diagnostics := ParseSQLToStandard("", "postgres")
	if query != nil {
		t.Fatalf("an empty source produced a query: %+v", query)
	}
	if len(diagnostics) != 1 {
		t.Fatalf("diagnostics=%+v", diagnostics)
	}
	if diagnostics[0].Code != "EMPTY_SQL" {
		t.Fatalf("code=%s", diagnostics[0].Code)
	}
}
