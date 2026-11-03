package artifact

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/database/persistence"
	"tei/internal/semantic"
)

func tableDefinitionInput() TableDefinitionInput {
	return TableDefinitionInput{
		InputRevision: "sha256:input",
		Mapping: persistence.Mapping{
			ID:           "sales_core.order",
			Database:     "sales_core",
			SemanticData: "sales.order",
			Table:        "public.orders",
			PrimaryKey:   []core.CanonicalID{"sales.order.order_id"},
			Columns: []persistence.ColumnMapping{
				{Attribute: "sales.order.order_id", Column: "order_id", Storage: persistence.StorageInt64, Nullability: persistence.Required},
				{Attribute: "sales.order.status", Column: "status", Storage: persistence.StorageVarchar, Nullability: persistence.Required},
			},
			Indexes: []persistence.IndexMapping{
				{Name: "orders_status_idx", Attributes: []core.CanonicalID{"sales.order.status"}},
			},
		},
		Data: semantic.Data{
			ID: "sales.order",
			Attributes: []semantic.Attribute{
				{ID: "sales.order.order_id", Symbol: "orderId", TypeRef: "std.identifier"},
				{ID: "sales.order.status", Symbol: "status", TypeRef: "std.code"},
			},
		},
	}
}

func TestRenderTableDefinitionIsDeterministic(t *testing.T) {
	first, err := RenderTableDefinition(tableDefinitionInput())
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	second, err := RenderTableDefinition(tableDefinitionInput())
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if first.Content != second.Content || first.Digest != second.Digest {
		t.Fatalf("the same input must render the same bytes:\n%s\n%s", first.Content, second.Content)
	}
	if err := first.Verify(); err != nil {
		t.Fatalf("rendered artifact must verify: %v", err)
	}
	if len(first.Unresolved) != 0 {
		t.Fatalf("a complete mapping must have no unresolved entries: %+v", first.Unresolved)
	}

	want := []string{
		"table public.orders",
		"data sales.order",
		"column order_id sales.order.order_id int64 required pk",
		"column status sales.order.status varchar required",
		"primary-key sales.order.order_id",
		"index orders_status_idx sales.order.status",
	}
	for _, line := range want {
		if !strings.Contains(first.Content, line+"\n") {
			t.Fatalf("content lost %q:\n%s", line, first.Content)
		}
	}
	if first.Kind != string(KindTableDefinition) || first.Rule.ID != TableDefinitionRuleID || first.Generator != TableDefinitionGenerator {
		t.Fatalf("artifact identity is incomplete: %+v", first)
	}
}

func TestRenderTableDefinitionReportsMissingColumnAndKey(t *testing.T) {
	input := tableDefinitionInput()
	input.Mapping.PrimaryKey = nil
	input.Data.Attributes = append(input.Data.Attributes, semantic.Attribute{
		ID: "sales.order.customer_id", Symbol: "customerId", TypeRef: "std.identifier",
	})

	rendered, err := RenderTableDefinition(input)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if !strings.Contains(rendered.Content, "primary-key (none)\n") {
		t.Fatalf("content must show the missing key:\n%s", rendered.Content)
	}
	if len(rendered.Unresolved) != 2 {
		t.Fatalf("expected two unresolved entries: %+v", rendered.Unresolved)
	}
	subjects := []string{rendered.Unresolved[0].Subject, rendered.Unresolved[1].Subject}
	if !contains(subjects, "sales.order.customer_id") || !contains(subjects, "sales.order") {
		t.Fatalf("unexpected unresolved subjects: %+v", rendered.Unresolved)
	}
	for _, unresolved := range rendered.Unresolved {
		if unresolved.Detail == "" || unresolved.Reason == "" {
			t.Fatalf("unresolved entries must explain themselves: %+v", unresolved)
		}
	}
}

func TestRenderTableDefinitionChangesWithInputRevision(t *testing.T) {
	first, err := RenderTableDefinition(tableDefinitionInput())
	if err != nil {
		t.Fatal(err)
	}
	input := tableDefinitionInput()
	input.InputRevision = "sha256:other"
	second, err := RenderTableDefinition(input)
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest == second.Digest {
		t.Fatal("a different input revision must produce a different digest")
	}
	if first.Content != second.Content {
		t.Fatal("the content itself does not depend on the revision; only the identity does")
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
func TestRenderTableDefinitionShowsForeignKeys(t *testing.T) {
	input := tableDefinitionInput()
	input.Mapping.ForeignKeys = []persistence.ForeignKeyMapping{{
		Name:                 "orders_customer_fk",
		Attributes:           []core.CanonicalID{"sales.order.customer_id"},
		ReferencedMapping:    "sales_core.customer",
		ReferencedAttributes: []core.CanonicalID{"sales.customer.customer_id"},
	}}

	rendered, err := RenderTableDefinitionWith(input, func(id core.CanonicalID) (string, bool) {
		if id == "sales_core.customer" {
			return "public.customers", true
		}
		return "", false
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	want := "foreign-key orders_customer_fk sales.order.customer_id -> public.customers(sales.customer.customer_id)\n"
	if !strings.Contains(rendered.Content, want) {
		t.Fatalf("content lost the foreign key:\n%s", rendered.Content)
	}
	if err := rendered.Verify(); err != nil {
		t.Fatalf("artifact must verify: %v", err)
	}
}
func TestRenderTableDefinitionShowsUniqueConstraints(t *testing.T) {
	input := tableDefinitionInput()
	input.Mapping.UniqueConstraints = []persistence.UniqueConstraintMapping{{
		Name:       "orders_status_key",
		Attributes: []core.CanonicalID{"sales.order.status"},
	}}

	rendered, err := RenderTableDefinition(input)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(rendered.Content, "unique orders_status_key sales.order.status\n") {
		t.Fatalf("content lost the unique constraint:\n%s", rendered.Content)
	}
}
