package validation_test

import (
	"testing"

	"tei/internal/application"
	"tei/internal/database"
	"tei/internal/ir"
	"tei/internal/validation"
)

func TestDuplicateApplicationDatasourceAliasIsError(t *testing.T) {
	w := ir.World{
		Applications: []application.Application{
			{
				ID: "order_entry",
				Databases: []application.DatabaseRequirement{
					{Alias: "sales", TargetDatabase: "sales_core", Access: database.AccessRead},
					{Alias: "sales", TargetDatabase: "sales_archive", Access: database.AccessRead},
				},
			},
		},
		Databases: []database.Database{
			{ID: "sales_core", Kind: "relational", Ownership: database.OwnershipManagedHere},
			{ID: "sales_archive", Kind: "relational", Ownership: database.OwnershipManagedHere},
		},
	}

	for _, diagnostic := range validation.Validate(w) {
		if diagnostic.Code == "DUPLICATE_DATASOURCE_ALIAS" && diagnostic.Severity == validation.Error {
			return
		}
	}
	t.Fatal("duplicate datasource alias was not rejected by Workspace Validation")
}
