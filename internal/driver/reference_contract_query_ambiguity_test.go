package driver_test

import (
	"strings"
	"testing"

	"tei/internal/database"
	"tei/internal/driver"
)

func TestResolveReferenceContractQueryRejectsDuplicateTargetDatabase(t *testing.T) {
	world := referenceContractQueryWorld()
	world.Databases = append(world.Databases, world.Databases[0])

	_, err := driver.NewResolver(world).ResolveReferenceContractQuery("order_entry", "order_entry.customer_reference")
	if err == nil {
		t.Fatal("同じCanonical IDのDatabaseが複数ある状態で最初の一件を選んでいます")
	}
	if !strings.Contains(err.Error(), "Database master_dataが複数定義されています") {
		t.Fatalf("Database identityの曖昧さを判別できないerrorです: %v", err)
	}
}

func TestResolveReferenceContractQueryRejectsDuplicateContract(t *testing.T) {
	world := referenceContractQueryWorld()
	world.Databases[0].Contracts = append(world.Databases[0].Contracts, world.Databases[0].Contracts[0])

	_, err := driver.NewResolver(world).ResolveReferenceContractQuery("order_entry", "order_entry.customer_reference")
	if err == nil {
		t.Fatal("同じCanonical IDのContractが複数ある状態で最初の一件を選んでいます")
	}
	if !strings.Contains(err.Error(), "Contract master_data.customerが複数定義されています") {
		t.Fatalf("Contract identityの曖昧さを判別できないerrorです: %v", err)
	}
}

func TestResolveReferenceContractQueryRejectsDuplicateQueryName(t *testing.T) {
	world := referenceContractQueryWorld()
	world.Databases[0].Contracts[0].Queries = append(
		world.Databases[0].Contracts[0].Queries,
		database.QueryContract{Name: "find_by_id"},
	)

	_, err := driver.NewResolver(world).ResolveReferenceContractQuery("order_entry", "order_entry.customer_reference")
	if err == nil {
		t.Fatal("同じ名前のContract Queryが複数ある状態で最初の一件を選んでいます")
	}
	if !strings.Contains(err.Error(), "Query find_by_idが複数定義されています") {
		t.Fatalf("Contract Query identityの曖昧さを判別できないerrorです: %v", err)
	}
}
