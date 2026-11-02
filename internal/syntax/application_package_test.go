package syntax_test

import (
	"testing"

	"tei/internal/syntax"
)

func TestParseApplicationPackage(t *testing.T) {
	src := `package order_entry.list_to_record ^1 {
    application order_entry
    member browse order_entry.recent_orders
    member record order_entry.order_record
    requires semantic sales.order
    requires relation sales.order.customer
    projection screen
    conformance list_to_record
}`
	got, err := syntax.NewParser(src).ParseApplicationPackage()
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "order_entry.list_to_record" || got.Version != "1" {
		t.Fatalf("package=%+v", got)
	}
	if len(got.Members) != 2 || got.Members[0].Kind != "browse" || got.Members[1].Pattern != "order_entry.order_record" {
		t.Fatalf("members=%+v", got.Members)
	}
	if len(got.Requires) != 2 || got.Requires[0].Kind != "semantic" || got.Requires[1].ID != "sales.order.customer" {
		t.Fatalf("requires=%+v", got.Requires)
	}
	if got.Projection != "screen" || got.Conformance != "list_to_record" {
		t.Fatalf("package=%+v", got)
	}
}

func TestParseApplicationPackageRequiresMember(t *testing.T) {
	src := `package order_entry.empty ^1 {
    application order_entry
}`
	if _, err := syntax.NewParser(src).ParseApplicationPackage(); err == nil {
		t.Fatal("package without members was accepted")
	}
}

func TestParseApplicationPackageBinding(t *testing.T) {
	src := `application order_entry {
    uses semantic sales
    binds package order_entry.list_to_record ^1
}`
	got, err := syntax.NewParser(src).ParseApplication()
	if err != nil {
		t.Fatal(err)
	}
	if len(got.PackageBindings) != 1 || got.PackageBindings[0].ID != "order_entry.list_to_record" || got.PackageBindings[0].Version != "1" {
		t.Fatalf("bindings=%+v", got.PackageBindings)
	}
}
