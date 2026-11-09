package composition

import (
	"reflect"
	"testing"
)

func TestReconciliationDependencyFirstStartConsumerFirstRetire(t *testing.T) {
	r := NewRegistry()
	p1, err := r.Apply(0, []Provider{
		provider("data-v1", "tenant-A", storage),
		provider("consumer", "tenant-A", app, required(storage)),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	p2, err := r.Apply(1, []Provider{provider("data-v2", "tenant-A", storage)}, []string{"data-v1"})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanReconciliation(p1, p2)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(plan.StopOrder, []string{"consumer", "data-v1"}) {
		t.Fatalf("retirement order %v", plan.StopOrder)
	}
	if !reflect.DeepEqual(plan.StartOrder, []string{"data-v2", "consumer"}) {
		t.Fatalf("activation order %v", plan.StartOrder)
	}
	if len(plan.Retained) != 0 {
		t.Fatalf("stale retained services %v", plan.Retained)
	}
	p3, err := r.Apply(2, nil, []string{"data-v2"})
	if err != nil {
		t.Fatal(err)
	}
	plan, err = PlanReconciliation(p2, p3)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(plan.StopOrder, []string{"consumer", "data-v2"}) || len(plan.StartOrder) != 0 {
		t.Fatalf("coeffect loss plan: %+v", plan)
	}
}

func TestReconciliationRetainsUnaffectedServiceAndRejectsStale(t *testing.T) {
	r := NewRegistry()
	p1, err := r.Apply(0, []Provider{provider("unrelated", "tenant-B", storage)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	p2, err := r.Apply(1, []Provider{provider("other", "tenant-A", app)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanReconciliation(p1, p2)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(plan.Retained, []string{"unrelated"}) || !reflect.DeepEqual(plan.StartOrder, []string{"other"}) || len(plan.StopOrder) != 0 {
		t.Fatalf("unexpected delta %+v", plan)
	}
	if _, err = PlanReconciliation(p1, p1); err == nil {
		t.Fatal("stale snapshots accepted")
	}
}
