package validation_test

import (
	"testing"

	"tei/internal/application"
	"tei/internal/database"
	"tei/internal/validation"
)

func TestDuplicateDatasourceAliasIsReportedWithoutUnknownAliasCascade(t *testing.T) {
	w := referenceWorld()
	w.Applications[0].Databases = append(w.Applications[0].Databases, w.Applications[0].Databases[0])

	diagnostics := validation.Validate(w)
	foundDuplicate := false
	for _, diagnostic := range diagnostics {
		switch diagnostic.Code {
		case "DUPLICATE_DATASOURCE_ALIAS":
			if diagnostic.Severity == validation.Error {
				foundDuplicate = true
			}
		case "UNKNOWN_DATASOURCE_ALIAS", "UNKNOWN_BINDING_DATASOURCE_ALIAS":
			t.Fatalf("duplicate datasource aliasをunknown aliasとして重複診断しました: %#v", diagnostics)
		}
	}
	if !foundDuplicate {
		t.Fatalf("expected DUPLICATE_DATASOURCE_ALIAS, got %#v", diagnostics)
	}
}

func TestDuplicateDatasourceAliasDoesNotLeakFirstMatchAccessIntoReferenceValidation(t *testing.T) {
	w := referenceWorld()
	w.Applications[0].Patterns = []application.Pattern{
		{
			ID:         cid("order_entry.customer_reference"),
			Kind:       application.PatternReference,
			Relation:   cid("sales.order.customer"),
			Datasource: local("master"),
		},
	}

	masterRead := w.Applications[0].Databases[1]
	w.Applications[0].Databases[1].Access = database.AccessWrite
	w.Applications[0].Databases = append(w.Applications[0].Databases, masterRead)

	diagnostics := validation.Validate(w)
	foundDuplicate := false
	for _, diagnostic := range diagnostics {
		switch diagnostic.Code {
		case "DUPLICATE_DATASOURCE_ALIAS":
			if diagnostic.Severity == validation.Error {
				foundDuplicate = true
			}
		case "APPLICATION_PATTERN_DATASOURCE_NOT_READABLE":
			t.Fatalf("duplicate datasource aliasの先頭Accessをauthorityとして利用しました: %#v", diagnostics)
		}
	}
	if !foundDuplicate {
		t.Fatalf("expected DUPLICATE_DATASOURCE_ALIAS, got %#v", diagnostics)
	}
}
