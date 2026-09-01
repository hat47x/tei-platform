package validation

import (
	"testing"

	"tei/internal/application"
	"tei/internal/database"
)

func TestValidateApplicationPatternsAcceptsReadWriteDatasourceForReference(t *testing.T) {
	w := validApplicationPatternWorld()
	w.Applications[0].Databases[1].Access = database.AccessReadWrite
	w.Applications[0].Patterns = []application.Pattern{{
		ID:         "order_entry.customer_reference",
		Kind:       application.PatternReference,
		Relation:   "sales.order.customer",
		Datasource: "master",
	}}

	if ds := validateApplicationPatterns(w); len(ds) != 0 {
		t.Fatalf("read-write Datasourceを使うreference Patternに不要なDiagnosticがあります: %#v", ds)
	}
}

func TestValidateRejectsWriteOnlyDatasourceForReferenceOnce(t *testing.T) {
	w := validApplicationPatternWorld()
	w.Applications[0].Databases[1].Access = database.AccessWrite
	w.Applications[0].Patterns = []application.Pattern{{
		ID:         "order_entry.customer_reference",
		Kind:       application.PatternReference,
		Relation:   "sales.order.customer",
		Datasource: "master",
	}}

	diagnostics := Validate(w)
	matches := diagnosticsByCode(diagnostics, "APPLICATION_PATTERN_DATASOURCE_NOT_READABLE")
	if len(matches) != 1 {
		t.Fatalf("write専用Datasourceに対するDiagnosticの件数=%d want=1: %#v", len(matches), diagnostics)
	}
	if matches[0].Subject == nil || matches[0].Subject.Kind != "application-pattern" || matches[0].Subject.ID != "order_entry.customer_reference" {
		t.Fatalf("Datasource読み取りDiagnosticのPattern Subjectが失われました: %#v", matches[0].Subject)
	}
}
