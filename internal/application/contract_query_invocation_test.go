package application

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestReferencePatternKeepsLegacyJSONWithoutContractQueryInvocation(t *testing.T) {
	pattern := Pattern{
		ID:         "order_entry.customer_reference",
		Kind:       PatternReference,
		Relation:   "sales.order.customer",
		Datasource: "master",
	}

	got, err := json.Marshal(pattern)
	if err != nil {
		t.Fatalf("reference PatternをJSONへ変換できません: %v", err)
	}
	want := "{\"id\":\"order_entry.customer_reference\",\"kind\":\"reference\",\"relation\":\"sales.order.customer\",\"datasource\":\"master\"}"
	if string(got) != want {
		t.Fatalf("Contract Query呼び出しを定義していない既存PatternのJSON shapeを変更しています:\nwant=%s\n got=%s", want, got)
	}
}

func TestReferencePatternKeepsExplicitContractQueryInvocation(t *testing.T) {
	want := Pattern{
		ID:         "order_entry.customer_reference",
		Kind:       PatternReference,
		Relation:   "sales.order.customer",
		Datasource: "master",
		ContractQuery: &ContractQueryInvocation{
			Contract: "master_data.customer",
			Query:    "find_by_id",
			Inputs: []ContractQueryInputBinding{
				{
					TargetAttribute: "customer.customer.customer_id",
					Input:           "id",
				},
			},
		},
	}

	encoded, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Contract Query呼び出しを持つreference PatternをJSONへ変換できません: %v", err)
	}
	var got Pattern
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatalf("Contract Query呼び出しを持つreference PatternをJSONから復元できません: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Contract Query呼び出しのidentityを保持できていません:\nwant=%#v\n got=%#v", want, got)
	}
}

func TestContractQueryInvocationKeepsCompositeKeyBindingOrder(t *testing.T) {
	invocation := ContractQueryInvocation{
		Contract: "master_data.customer",
		Query:    "find_by_composite_key",
		Inputs: []ContractQueryInputBinding{
			{TargetAttribute: "customer.customer.tenant_id", Input: "tenant"},
			{TargetAttribute: "customer.customer.customer_id", Input: "id"},
		},
	}

	if len(invocation.Inputs) != 2 {
		t.Fatalf("複合keyのinput binding数が変わっています: %d", len(invocation.Inputs))
	}
	if invocation.Inputs[0].TargetAttribute != "customer.customer.tenant_id" || invocation.Inputs[0].Input != "tenant" {
		t.Fatalf("先頭のinput binding順序を保持できていません: %#v", invocation.Inputs)
	}
	if invocation.Inputs[1].TargetAttribute != "customer.customer.customer_id" || invocation.Inputs[1].Input != "id" {
		t.Fatalf("後続のinput binding順序を保持できていません: %#v", invocation.Inputs)
	}
}
