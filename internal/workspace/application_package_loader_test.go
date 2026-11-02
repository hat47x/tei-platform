package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"tei/internal/workspace"
)

func writePackageWorkspaceFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadApplicationPackages(t *testing.T) {
	root := t.TempDir()
	writePackageWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: package_loader\nsources:\n  - semantic\n  - applications\n")
	writePackageWorkspaceFile(t, root, "semantic/sales/order.tei", `namespace sales

data Order {
    attribute orderId type std.identifier
}
`)
	writePackageWorkspaceFile(t, root, "applications/order-entry/application.tei", `application order_entry {
    uses semantic sales
}
`)
	writePackageWorkspaceFile(t, root, "applications/order-entry/patterns/recent-orders.tei", `pattern order_entry.recent_orders browse {
    data sales.order
    query order_entry.recent_orders
}
`)
	writePackageWorkspaceFile(t, root, "applications/order-entry/patterns/order-record.tei", `pattern order_entry.order_record record {
    data sales.order
}
`)
	writePackageWorkspaceFile(t, root, "applications/order-entry/packages/list-to-record.tei", `package order_entry.list_to_record ^1 {
    application order_entry
    member browse order_entry.recent_orders
    member record order_entry.order_record
    requires semantic sales.order
    projection screen
}
`)

	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.World.Applications) != 1 || len(loaded.World.Applications[0].Packages) != 1 {
		t.Fatalf("applications=%+v", loaded.World.Applications)
	}
	pkg := loaded.World.Applications[0].Packages[0]
	if pkg.ID != "order_entry.list_to_record" || pkg.Version != "1" || len(pkg.Members) != 2 {
		t.Fatalf("package=%+v", pkg)
	}
}
