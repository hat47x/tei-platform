package transform

import "testing"

func TestSnapshotInputsDeepCopiesCallerValuesOnce(t *testing.T) {
	nested := Value{
		Kind: ValueObject,
		Object: []Field{{
			Name: "items",
			Value: Value{
				Kind: ValueArray,
				Array: []Value{{Kind: ValueString, String: "before"}},
			},
		}},
	}
	inputs := map[string]Value{"source": nested}
	snapshot := snapshotInputs(inputs)

	inputs["source"].Object[0].Value.Array[0].String = "caller-changed"
	got := snapshot["source"].Object[0].Value.Array[0].String
	if got != "before" {
		t.Fatalf("snapshot aliases caller input: got %q", got)
	}
}

func TestEvaluatorViewsShareOnlyInvocationOwnedNestedValues(t *testing.T) {
	caller := Value{
		Kind: ValueObject,
		Object: []Field{{
			Name: "items",
			Value: Value{
				Kind: ValueArray,
				Array: []Value{{Kind: ValueString, String: "before"}},
			},
		}},
	}
	snapshot := snapshotInputs(map[string]Value{"source": caller})["source"]
	selected, ok := fieldView(snapshot, "items")
	if !ok {
		t.Fatal("items field not found")
	}

	selected.Array[0].String = "evaluator-owned-change"
	if got := snapshot.Object[0].Value.Array[0].String; got != "evaluator-owned-change" {
		t.Fatalf("field view unexpectedly deep-copied nested value: got %q", got)
	}
	if got := caller.Object[0].Value.Array[0].String; got != "before" {
		t.Fatalf("evaluator-owned view leaked mutation to caller: got %q", got)
	}
}

func TestReplaceFieldOwnedCopiesObjectContainerOnly(t *testing.T) {
	source := Value{
		Kind: ValueObject,
		Object: []Field{
			{Name: "replace", Value: String("old")},
			{
				Name: "keep",
				Value: Value{
					Kind: ValueArray,
					Array: []Value{{Kind: ValueString, String: "nested"}},
				},
			},
		},
	}
	out, replaced, err := replaceFieldOwned(source, "replace", String("new"))
	if err != nil {
		t.Fatal(err)
	}
	if !replaced || out.Object[0].Value.String != "new" || source.Object[0].Value.String != "old" {
		t.Fatalf("replace result: source=%#v out=%#v", source, out)
	}

	out.Object[1].Name = "renamed"
	if source.Object[1].Name != "keep" {
		t.Fatal("replace-field reused mutable object container")
	}

	out.Object[1].Value.Array[0].String = "shared"
	if source.Object[1].Value.Array[0].String != "shared" {
		t.Fatal("replace-field unexpectedly deep-copied unchanged nested value")
	}
}

func TestOmitFieldOwnedCopiesObjectContainerOnly(t *testing.T) {
	source := Value{
		Kind: ValueObject,
		Object: []Field{
			{Name: "drop", Value: String("x")},
			{
				Name: "keep",
				Value: Value{
					Kind: ValueArray,
					Array: []Value{{Kind: ValueString, String: "nested"}},
				},
			},
		},
	}
	out, err := omitFieldOwned(source, "drop")
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Object) != 1 || out.Object[0].Name != "keep" {
		t.Fatalf("omit result: %#v", out)
	}

	out.Object[0].Name = "renamed"
	if source.Object[1].Name != "keep" {
		t.Fatal("omit-field reused mutable object container")
	}

	out.Object[0].Value.Array[0].String = "shared"
	if source.Object[1].Value.Array[0].String != "shared" {
		t.Fatal("omit-field unexpectedly deep-copied unchanged nested value")
	}
}
