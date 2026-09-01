package workspace_test

import (
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

func TestWorkspaceLoadsAdvancedIndexAndIndexesNestedDeclaration(t *testing.T) {
	root := t.TempDir()
	writeSource(t, root, "tei.workspace.yaml", []byte("workspace: advanced_index_test\nsources:\n  - databases\n"))
	writeSource(t, root, "databases/sales-core/database.tei", []byte(`database sales_core {
    kind relational
    ownership managed-here
}
`))
	persistencePath := "databases/sales-core/persistence/customer.tei"
	content := []byte(`persist sales_core.customer {
    database sales_core
    semantic customer.customer
    table public.customers
    column customer.name -> name storage varchar required
    column customer.external_id -> external_id storage varchar required
    column customer.profile -> profile storage json optional

    index customers_external_id_idx on customer.external_id

    index customers_name_advanced_idx {
        key attribute customer.name {
            order desc
            nulls last
        }
        include customer.external_id
        predicate {
            provider postgres
            dialect postgresql
            source "profile IS NOT NULL"
        }
    }
}
`)
	writeSource(t, root, persistencePath, content)

	result, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.World.Persistence) != 1 || len(result.World.Persistence[0].Indexes) != 2 {
		t.Fatalf("persistence=%#v", result.World.Persistence)
	}
	if result.World.Persistence[0].Indexes[1].Name != "customers_name_advanced_idx" || len(result.World.Persistence[0].Indexes[1].Keys) != 1 {
		t.Fatalf("advanced index=%#v", result.World.Persistence[0].Indexes[1])
	}

	idx, err := workspace.BuildSourceIndex(root, result)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"customers_external_id_idx", "customers_name_advanced_idx"} {
		subject := sourceindex.Subject{Kind: "persistence-index", ID: "sales_core.customer/" + name}
		refs := idx.ReferencesFor(subject)
		if len(refs) != 1 {
			t.Fatalf("references for %+v=%#v", subject, refs)
		}
		if refs[0].ArtifactPath != persistencePath {
			t.Fatalf("artifact for %+v=%q", subject, refs[0].ArtifactPath)
		}
		rng := refs[0].Range
		if rng.Start.Offset == rng.End.Offset {
			t.Fatalf("index anchor remained a point: %+v", rng)
		}
		if got := string(content[rng.Start.Offset:rng.End.Offset]); got != name {
			t.Fatalf("anchor for %s=%q range=%+v", name, got, rng)
		}
	}
}
