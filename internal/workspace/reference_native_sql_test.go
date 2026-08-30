package workspace_test

import (
	"path/filepath"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

func TestReferenceQueryUsesNativeSQLAsAuthoritativeSource(t *testing.T) {
	root := repoRoot(t)
	ws := filepath.Join(root, "reference", "workspace")
	result, err := workspace.Load(ws)
	if err != nil {
		t.Fatal(err)
	}

	const queryID = "order_entry.recent_orders"
	const sqlPath = "applications/order-entry/queries/recent_orders.sql"
	const declarationPath = "applications/order-entry/queries/recent_orders.tei"
	const wantSQL = "SELECT\n            o.order_id,\n            o.customer_id,\n            o.ordered_at,\n            o.status,\n            o.total_amount\n        FROM public.orders AS o\n        WHERE o.ordered_at >= :from\n        ORDER BY o.ordered_at DESC"

	if len(result.World.Queries) != 1 {
		t.Fatalf("queries=%#v", result.World.Queries)
	}
	q := result.World.Queries[0]
	if q.ID != queryID {
		t.Fatalf("query id=%q", q.ID)
	}
	if q.Source != wantSQL {
		t.Fatalf("query SQL changed meaning or normalization\nGOT:\n%s\nWANT:\n%s", q.Source, wantSQL)
	}

	idx, err := workspace.BuildSourceIndex(ws, result)
	if err != nil {
		t.Fatal(err)
	}
	refs := idx.ReferencesFor(sourceindex.Subject{Kind: "query", ID: queryID})
	if len(refs) != 2 {
		t.Fatalf("query source references=%#v", refs)
	}
	definition := referenceByRole(t, refs, sourceindex.RoleDefinition)
	authoring := referenceByRole(t, refs, sourceindex.RoleAuthoring)
	if definition.ArtifactPath != declarationPath {
		t.Fatalf("definition path=%q", definition.ArtifactPath)
	}
	if authoring.ArtifactPath != sqlPath {
		t.Fatalf("authoring path=%q", authoring.ArtifactPath)
	}

	managed := false
	for _, file := range result.ManagedFiles {
		if file.Path == sqlPath && file.Extension == ".sql" && file.SourceRoot == "applications" {
			managed = true
			break
		}
	}
	if !managed {
		t.Fatalf("native SQL is not a managed Workspace source: %#v", result.ManagedFiles)
	}
}
