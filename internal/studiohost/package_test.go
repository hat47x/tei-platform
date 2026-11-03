package studiohost

import (
	"os"
	"path/filepath"
	"testing"
)

func writePackageFixture(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func packageFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writePackageFixture(t, root, "tei.workspace.yaml", "workspace: studio_package\nformatVersion: 1\nsources:\n  - semantic\n  - applications\n")
	writePackageFixture(t, root, "semantic/sales/order.tei", `namespace sales

data Order {
    attribute orderId type std.identifier
}
`)
	writePackageFixture(t, root, "applications/order-entry/application.tei", `application order_entry {
    uses semantic sales
}
`)
	writePackageFixture(t, root, "applications/order-entry/patterns/order-record.tei", `pattern order_entry.order_record record {
    data sales.order
}
`)
	writePackageFixture(t, root, "applications/order-entry/patterns/recent-orders.tei", `pattern order_entry.recent_orders browse {
    data sales.order
    query order_entry.recent_orders
}
`)
	writePackageFixture(t, root, "applications/order-entry/packages/list-to-record.tei", `package order_entry.list_to_record ^1 {
    application order_entry
    member browse order_entry.recent_orders
    member record order_entry.order_record
    requires semantic sales.order
    projection screen
}
`)
	return root
}

func TestViewPackageReturnsMembers(t *testing.T) {
	host, err := NewReadOnly([]string{packageFixture(t)})
	if err != nil {
		t.Fatal(err)
	}
	response := host.ViewPackage("workspace-1", "order_entry", "order_entry.list_to_record")
	if !response.OK || response.Failure != nil {
		t.Fatalf("response=%+v", response)
	}
	if response.Value.Version != "1" || len(response.Value.Members) != 2 || response.Value.Projection != "screen" {
		t.Fatalf("package=%+v", response.Value)
	}
}
