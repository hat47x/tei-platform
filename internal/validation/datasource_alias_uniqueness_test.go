package validation_test

import (
	"testing"

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
