package decisions

import (
	"context"
	"testing"

	"tei/internal/execution"
	"tei/internal/permission"
	"tei/internal/plugin"
)

func permissionCatalog() plugin.Catalog {
	return plugin.Catalog{Plugins: []plugin.Definition{{
		ID:      "tei.decisions",
		Version: "1.0.0",
		Provides: []plugin.Capability{{
			ID:          "sales.close",
			Permissions: []permission.Spec{{Kind: permission.KindFilesystem, Access: permission.AccessWrite, Scope: "probe-table"}},
		}},
	}}}
}

func TestAMissingGrantBlocksBeforeTheWrite(t *testing.T) {
	conn, closeConn := probe(t)
	defer closeConn()

	calls := 0
	preconditions := execution.CombinePreconditions(
		execution.PermissionPreconditions(execution.RequirementsFromCatalog(permissionCatalog()), nil),
	)
	_, results, err := execution.RunPlan(context.Background(), probePlan(), []string{"2026-10"}, &memoryJournal{}, updateStep(t, conn, &calls), preconditions)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if calls != 0 {
		t.Fatalf("an ungranted capability must stop before the write: calls=%d", calls)
	}
	if len(results) != 1 || results[0].Status != "blocked" {
		t.Fatalf("the missing grant must block the run: %+v", results)
	}
	if got := status(t, conn); got != "new" {
		t.Fatalf("a blocked run must not change the row: %q", got)
	}
}

func TestACoveringGrantLetsTheWriteThrough(t *testing.T) {
	conn, closeConn := probe(t)
	defer closeConn()

	calls := 0
	granted := []permission.Spec{{Kind: permission.KindFilesystem, Access: permission.AccessReadWrite, Scope: "probe-table"}}
	preconditions := execution.CombinePreconditions(
		execution.PermissionPreconditions(execution.RequirementsFromCatalog(permissionCatalog()), granted),
	)
	_, results, err := execution.RunPlan(context.Background(), probePlan(), []string{"2026-10"}, &memoryJournal{}, updateStep(t, conn, &calls), preconditions)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(results) != 1 || results[0].Status != "resolved" {
		t.Fatalf("a covering grant must let the step run: %+v", results)
	}
	if got := status(t, conn); got != "closed" {
		t.Fatalf("the row must change: %q", got)
	}
}
