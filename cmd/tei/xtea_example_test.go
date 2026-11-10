package main

import (
	"context"
	"path/filepath"
	"testing"

	"tei/internal/applicationview"
	"tei/internal/command"
	"tei/internal/core"
	"tei/internal/validation"
	"tei/internal/workspace"
)

func TestXTEAExampleWorkspace(t *testing.T) {
	root := filepath.Join("..", "..", "examples", "xtea-lowcode")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range validation.Validate(loaded.World) {
		if diagnostic.Severity == validation.Error {
			t.Fatalf("unexpected diagnostic: %s %s", diagnostic.Code, diagnostic.Message)
		}
	}

	view, err := applicationview.Build(loaded.World, "order_entry", "order_entry.order_record")
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Items) != 2 || len(view.Unused) != 2 || len(view.Hooks) != 1 || len(view.Commands) != 1 {
		t.Fatalf("unexpected record view: %+v", view)
	}
	// The example declares the hook's scope, so a reader sees which unit the hook
	// decides for instead of inferring it from the pattern kind.
	if view.Hooks[0].Scope != "record" {
		t.Fatalf("hook scope lost: %+v", view.Hooks[0])
	}

	plan, err := applicationview.BuildCommandPlan(loaded.World, "order_entry", "order_entry.order_record", "sales.submit_order", map[string]string{"order": "ORD-1"})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Ready || len(plan.Inputs) != 1 || plan.Period == nil {
		t.Fatalf("unexpected command plan: %+v", plan)
	}
	// The example declares a closing day, so the plan carries it and a reader sees
	// the period is narrower than a month.
	if plan.Period.ClosingDay != 20 {
		t.Fatalf("closing day lost from the plan: %+v", plan.Period)
	}

	pkg, err := applicationview.BuildPackage(loaded.World, "order_entry", "order_entry.list_to_record")
	if err != nil {
		t.Fatal(err)
	}
	if !pkg.Bound || len(pkg.Members) != 4 {
		t.Fatalf("unexpected package view: %+v", pkg)
	}
}

func TestXTEAExampleLifecycle(t *testing.T) {
	root := filepath.Join("..", "..", "examples", "xtea-lowcode")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	store := command.NewMemoryStore()
	executor := command.ReferenceDataExecutor{World: loaded.World, Store: store, KeyFields: []string{"order_id"}}
	invoke := func(pattern, capability string, inputs map[string]string) error {
		_, err := executor.Execute(context.Background(), command.Request{
			Application: "order_entry",
			Pattern:     core.CanonicalID(pattern),
			Capability:  core.CanonicalID(capability),
			Inputs:      inputs,
		})
		return err
	}
	if err := invoke("order_entry.order_create", "sales.create_order", map[string]string{
		"order_id": "ORD-1", "status": "open", "order_date": "2026-11-06",
	}); err != nil {
		t.Fatal(err)
	}
	if err := invoke("order_entry.line_create", "sales.create_line", map[string]string{
		"order_line_id": "L1", "order_id": "ORD-1", "quantity": "2",
	}); err != nil {
		t.Fatal(err)
	}
	if err := invoke("order_entry.payment_create", "sales.create_payment", map[string]string{
		"payment_id": "P1", "order_id": "ORD-1",
	}); err != nil {
		t.Fatal(err)
	}
	if err := invoke("order_entry.order_delete", "sales.delete_order", map[string]string{"order_id": "ORD-1"}); err == nil {
		t.Fatal("restricting payment did not block order delete")
	}
	if len(store.Records("sales.order")) != 1 || len(store.Records("sales.order_line")) != 1 || len(store.Records("sales.payment")) != 1 {
		t.Fatalf("restricted delete was not rolled back: orders=%d lines=%d payments=%d",
			len(store.Records("sales.order")), len(store.Records("sales.order_line")), len(store.Records("sales.payment")))
	}
	if err := invoke("order_entry.payment_delete", "sales.delete_payment", map[string]string{
		"payment_id": "P1", "order_id": "ORD-1",
	}); err != nil {
		t.Fatal(err)
	}
	if err := invoke("order_entry.order_delete", "sales.delete_order", map[string]string{"order_id": "ORD-1"}); err != nil {
		t.Fatal(err)
	}
	if len(store.Records("sales.order")) != 0 || len(store.Records("sales.order_line")) != 0 || len(store.Records("sales.payment")) != 0 {
		t.Fatalf("cascade left records: orders=%d lines=%d payments=%d",
			len(store.Records("sales.order")), len(store.Records("sales.order_line")), len(store.Records("sales.payment")))
	}

	duplicate := map[string]string{"order_id": "ORD-2", "status": "open", "order_date": "2026-11-06"}
	if err := invoke("order_entry.order_create", "sales.create_order", duplicate); err != nil {
		t.Fatal(err)
	}
	duplicate["status"] = "closed"
	if err := invoke("order_entry.order_create", "sales.create_order", duplicate); err == nil {
		t.Fatal("duplicate candidate key was accepted")
	}
}
