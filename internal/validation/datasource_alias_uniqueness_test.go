package validation_test

import (
	"testing"

	"tei/internal/application"
	"tei/internal/database"
	"tei/internal/validation"
)

func TestDuplicateDatasourceAliasIsReportedWithoutDerivedBindingCascade(t *testing.T) {
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
		case "DATABASE_BINDING_TARGET_MISMATCH":
			t.Fatalf("duplicate datasource aliasの非authoritative lookup値からbinding target mismatchを派生しました: %#v", diagnostics)
		}
	}
	if !foundDuplicate {
		t.Fatalf("expected DUPLICATE_DATASOURCE_ALIAS, got %#v", diagnostics)
	}
}

func TestDuplicateDatasourceAliasWithDifferentTargetsDoesNotDeriveBindingTargetMismatch(t *testing.T) {
	w := referenceWorld()
	duplicate := w.Applications[0].Databases[0]
	duplicate.TargetDatabase = cid("master_data")
	duplicate.RequiredContracts = nil
	w.Applications[0].Databases = append(w.Applications[0].Databases, duplicate)

	diagnostics := validation.Validate(w)
	foundDuplicate := false
	for _, diagnostic := range diagnostics {
		switch diagnostic.Code {
		case "DUPLICATE_DATASOURCE_ALIAS":
			if diagnostic.Severity == validation.Error {
				foundDuplicate = true
			}
		case "UNKNOWN_DATASOURCE_ALIAS", "UNKNOWN_BINDING_DATASOURCE_ALIAS":
			t.Fatalf("異なるTargetDatabaseを持つduplicate datasource aliasをunknown aliasとして診断しました: %#v", diagnostics)
		case "DATABASE_BINDING_TARGET_MISMATCH":
			t.Fatalf("異なるTargetDatabaseを持つduplicate datasource aliasからbinding target mismatchを派生しました: %#v", diagnostics)
		}
	}
	if !foundDuplicate {
		t.Fatalf("expected DUPLICATE_DATASOURCE_ALIAS, got %#v", diagnostics)
	}
}

func TestDuplicateDatasourceAliasStillReportsIndependentMissingInstance(t *testing.T) {
	w := referenceWorld()
	w.Applications[0].Databases = append(w.Applications[0].Databases, w.Applications[0].Databases[0])
	w.Environments[0].Bindings[0].Instance = cid("missing.instance")

	diagnostics := validation.Validate(w)
	foundDuplicate := false
	foundMissingInstance := false
	for _, diagnostic := range diagnostics {
		switch diagnostic.Code {
		case "DUPLICATE_DATASOURCE_ALIAS":
			if diagnostic.Severity == validation.Error {
				foundDuplicate = true
			}
		case "UNKNOWN_DATABASE_INSTANCE":
			foundMissingInstance = true
		case "UNKNOWN_DATASOURCE_ALIAS", "UNKNOWN_BINDING_DATASOURCE_ALIAS":
			t.Fatalf("duplicate datasource aliasをunknown aliasとして重複診断しました: %#v", diagnostics)
		case "DATABASE_BINDING_TARGET_MISMATCH":
			t.Fatalf("missing Instanceより先にduplicate datasource aliasの非authoritative targetを利用しました: %#v", diagnostics)
		}
	}
	if !foundDuplicate {
		t.Fatalf("expected DUPLICATE_DATASOURCE_ALIAS, got %#v", diagnostics)
	}
	if !foundMissingInstance {
		t.Fatalf("expected UNKNOWN_DATABASE_INSTANCE, got %#v", diagnostics)
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
