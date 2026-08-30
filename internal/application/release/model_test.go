package release

import (
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
)

func TestReleaseSnapshotsDatabaseRequirements(t *testing.T) {
	r := Definition{Application: core.MustCanonicalID("order_entry"), Version: "3.8.0", Databases: []application.DatabaseRequirement{
		{Alias: core.LocalName("sales"), TargetDatabase: core.MustCanonicalID("sales_core"), Access: database.AccessReadWrite, RequiredContracts: []database.ContractRequirement{{ContractID: core.MustCanonicalID("sales_core.order"), CompatibleMajor: 4}}},
	}}
	app := r.AsApplication()
	// Mutating a current-application copy must not mutate the release snapshot.
	app.Databases[0].RequiredContracts[0].CompatibleMajor = 99
	if got := r.Databases[0].RequiredContracts[0].CompatibleMajor; got != 4 {
		t.Fatalf("release snapshot mutated through derived application: %d", got)
	}
}
