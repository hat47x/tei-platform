package database

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestQueryContractKeepsLegacyJSONWhenSignatureIsUndefined(t *testing.T) {
	got, err := json.Marshal(QueryContract{Name: "find_by_id"})
	if err != nil {
		t.Fatalf("QueryContractをJSONへ変換できません: %v", err)
	}
	if string(got) != `{"name":"find_by_id"}` {
		t.Fatalf("signature未定義の既存JSON shapeを変更しています: %s", got)
	}
}

func TestQueryContractKeepsTypedInputAndSemanticResult(t *testing.T) {
	want := QueryContract{
		Name: "find_by_id",
		Inputs: []QueryInput{
			{Name: "id", Type: "customer.customer_id"},
		},
		Result:            "customer.customer",
		ResultCardinality: QueryResultZeroOrOne,
	}

	encoded, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("QueryContractをJSONへ変換できません: %v", err)
	}
	var got QueryContract
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatalf("QueryContractをJSONから復元できません: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("typed signatureのidentityを保持できていません:\nwant=%#v\n got=%#v", want, got)
	}
}

func TestQueryResultCardinalityAcceptsOnlyDeclaredContracts(t *testing.T) {
	for _, cardinality := range []QueryResultCardinality{
		QueryResultOne,
		QueryResultZeroOrOne,
		QueryResultMany,
	} {
		if !cardinality.Valid() {
			t.Fatalf("定義済みresult cardinality %qを拒否しています", cardinality)
		}
	}

	if QueryResultCardinality("unknown").Valid() {
		t.Fatal("未定義のresult cardinalityを受理しています")
	}
}
