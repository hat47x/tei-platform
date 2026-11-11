package threeelement

import (
	"encoding/json"
	"os"
	"testing"
)

func TestOrderSystemFixtureProducesExpectedCrossReferences(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	modelPath := root + "/model.json"
	if err := os.WriteFile(modelPath, []byte(fixtureBytes), 0o644); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	var model ThreeElementModel
	if err := json.Unmarshal(raw, &model); err != nil {
		t.Fatal(err)
	}
	model.ModelFingerprint = ModelFingerprint(model)

	report := ValidateModel(model)
	if !report.OK {
		t.Fatalf("validation failed: %+v", report.Diagnostics)
	}
	if len(report.UnusedData) != 0 {
		t.Fatalf("expected no unused data, got %v", report.UnusedData)
	}
	if len(report.UnreferencedFunctions) != 0 {
		t.Fatalf("expected no unreferenced functions, got %v", report.UnreferencedFunctions)
	}

	rows := GenerateCRUDTable(model)
	if len(rows) != 8 {
		t.Fatalf("expected 8 CRUD rows, got %d", len(rows))
	}

	bizFn := GenerateBusinessFunctionCrossReference(model)
	if len(bizFn) != 5 {
		t.Fatalf("expected 5 business->function mappings, got %d", len(bizFn))
	}

	fnData := GenerateFunctionDataCrossReference(model)
	if len(fnData) != 8 {
		t.Fatalf("expected 8 function->data mappings, got %d", len(fnData))
	}

	bizData := GenerateBusinessDataCrossReference(model)
	if len(bizData) != 8 {
		t.Fatalf("expected 8 business->data mappings, got %d", len(bizData))
	}

	diff := DiffModels(model, model)
	if len(diff.Added) != 0 || len(diff.Removed) != 0 || len(diff.Changed) != 0 {
		t.Fatalf("expected empty diff for identical models, got %+v", diff)
	}
}

var fixtureBytes = []byte(`{
  "schemaVersion": "tei.three-element.model/v0",
  "effect": "read-only",
  "modelId": "order-system",
  "name": "Order management system",
  "description": "Representative business system for three-element analysis",
  "dataElements": [
    {
      "id": "customer",
      "name": "Customer",
      "attributes": [
        {"id": "customerId", "name": "Customer ID", "type": "text", "key": true},
        {"id": "name", "name": "Name", "type": "text"}
      ],
      "compositeKey": ["customerId"]
    },
    {
      "id": "order",
      "name": "Order",
      "attributes": [
        {"id": "orderId", "name": "Order ID", "type": "text", "key": true},
        {"id": "customerId", "name": "Customer ID", "type": "text", "key": true},
        {"id": "amount", "name": "Amount", "type": "number"}
      ],
      "compositeKey": ["orderId", "customerId"],
      "relations": [
        {"id": "order-customer", "name": "belongs to", "targetId": "customer", "kind": "many-to-one", "directed": true}
      ]
    },
    {
      "id": "orderLine",
      "name": "Order line",
      "attributes": [
        {"id": "orderLineId", "name": "Order line ID", "type": "text", "key": true},
        {"id": "orderId", "name": "Order ID", "type": "text", "key": true},
        {"id": "productId", "name": "Product ID", "type": "text"},
        {"id": "quantity", "name": "Quantity", "type": "number"}
      ],
      "compositeKey": ["orderLineId", "orderId"],
      "relations": [
        {"id": "orderline-order", "name": "belongs to", "targetId": "order", "kind": "many-to-one", "directed": true}
      ]
    }
  ],
  "functionElements": [
    {
      "id": "registerCustomer",
      "name": "Register customer",
      "type": "create",
      "operations": [
        {"dataId": "customer", "operations": "C"}
      ]
    },
    {
      "id": "submitOrder",
      "name": "Submit order",
      "type": "transaction",
      "operations": [
        {"dataId": "order", "operations": "C"},
        {"dataId": "orderLine", "operations": "C"}
      ]
    },
    {
      "id": "listOrders",
      "name": "List orders",
      "type": "query",
      "operations": [
        {"dataId": "order", "operations": "R"},
        {"dataId": "customer", "operations": "R"}
      ]
    },
    {
      "id": "updateOrderAmount",
      "name": "Update order amount",
      "type": "update",
      "operations": [
        {"dataId": "order", "operations": "U"}
      ]
    },
    {
      "id": "cancelOrder",
      "name": "Cancel order",
      "type": "delete",
      "operations": [
        {"dataId": "order", "operations": "D"},
        {"dataId": "orderLine", "operations": "D"}
      ]
    }
  ],
  "businessElements": [
    {
      "id": "salesDesk",
      "name": "Sales desk",
      "role": "organization",
      "activities": [
        {"id": "enterCustomer", "name": "Enter customer", "functionIds": ["registerCustomer"]},
        {"id": "acceptOrder", "name": "Accept order", "functionIds": ["submitOrder"]},
        {"id": "reviewOrders", "name": "Review orders", "functionIds": ["listOrders"]},
        {"id": "adjustOrder", "name": "Adjust order", "functionIds": ["updateOrderAmount"]},
        {"id": "cancelOrder", "name": "Cancel order", "functionIds": ["cancelOrder"]}
      ]
    }
  ],
  "crossReferences": [
    {"id": "xref-1", "kind": "business-to-function", "fromKind": "business", "fromId": "salesDesk", "toKind": "function", "toId": "submitOrder"},
    {"id": "xref-2", "kind": "function-to-data", "fromKind": "function", "fromId": "submitOrder", "toKind": "data", "toId": "order"},
    {"id": "xref-3", "kind": "data-to-data", "fromKind": "data", "fromId": "order", "toKind": "data", "toId": "customer", "directed": true}
  ],
  "crudDefinitions": [
    {"id": "crud-1", "functionId": "submitOrder", "dataId": "order", "create": true, "read": true, "update": false, "delete": false, "source": "declared"},
    {"id": "crud-2", "functionId": "submitOrder", "dataId": "orderLine", "create": true, "read": true, "update": false, "delete": false, "source": "declared"},
    {"id": "crud-3", "functionId": "listOrders", "dataId": "order", "create": false, "read": true, "update": false, "delete": false, "source": "declared"},
    {"id": "crud-4", "functionId": "listOrders", "dataId": "customer", "create": false, "read": true, "update": false, "delete": false, "source": "declared"},
    {"id": "crud-5", "functionId": "updateOrderAmount", "dataId": "order", "create": false, "read": true, "update": true, "delete": false, "source": "declared"},
    {"id": "crud-6", "functionId": "cancelOrder", "dataId": "order", "create": false, "read": true, "update": false, "delete": true, "source": "declared"},
    {"id": "crud-7", "functionId": "cancelOrder", "dataId": "orderLine", "create": false, "read": true, "update": false, "delete": true, "source": "declared"}
  ]
}`)
