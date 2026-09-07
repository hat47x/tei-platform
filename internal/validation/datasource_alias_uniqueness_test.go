package validation_test

import (
	"testing"

	"tei/internal/validation"
)

func TestDuplicateDatasourceAliasIsError(t *testing.T) {
	w := referenceWorld()
	w.Applications[0].Databases = append(w.Applications[0].Databases, w.Applications[0].Databases[0])

	diagnostics := validation.Validate(w)
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "DUPLICATE_DATASOURCE_ALIAS" && diagnostic.Severity == validation.Error {
			return
		}
	}
	t.Fatalf("expected DUPLICATE_DATASOURCE_ALIAS, got %#v", diagnostics)
}
