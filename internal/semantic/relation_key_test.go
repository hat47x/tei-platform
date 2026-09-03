package semantic

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRelationKeysPreserveAuthoredOrder(t *testing.T) {
	relation := Relation{
		ID:          "sample.parent.children",
		Symbol:      "children",
		Target:      "sample.child",
		Cardinality: CardinalityMany,
		Keys: []RelationKey{
			{SourceAttribute: "sample.parent.tenant_id", TargetAttribute: "sample.child.tenant_id"},
			{SourceAttribute: "sample.parent.parent_id", TargetAttribute: "sample.child.parent_id"},
		},
	}

	if len(relation.Keys) != 2 {
		t.Fatalf("keys=%#v", relation.Keys)
	}
	if relation.Keys[0].SourceAttribute != "sample.parent.tenant_id" || relation.Keys[0].TargetAttribute != "sample.child.tenant_id" {
		t.Fatalf("先頭のkey対応が変わっています: %#v", relation.Keys[0])
	}
	if relation.Keys[1].SourceAttribute != "sample.parent.parent_id" || relation.Keys[1].TargetAttribute != "sample.child.parent_id" {
		t.Fatalf("2番目のkey対応が変わっています: %#v", relation.Keys[1])
	}
}

func TestRelationWithoutKeysKeepsExistingJSONShape(t *testing.T) {
	relation := Relation{
		ID:          "sales.order.customer",
		Symbol:      "customer",
		Target:      "customer.customer",
		Cardinality: CardinalityOne,
	}

	encoded, err := json.Marshal(relation)
	if err != nil {
		t.Fatalf("RelationのJSON化に失敗しました: %v", err)
	}
	if strings.Contains(string(encoded), `"keys"`) {
		t.Fatalf("key未定義の既存Relationへkeysを出力してはいけません: %s", encoded)
	}
}

func TestRelationWithKeysIncludesSemanticAttributeIdentityInJSON(t *testing.T) {
	relation := Relation{
		ID:          "sales.order.customer",
		Symbol:      "customer",
		Target:      "customer.customer",
		Cardinality: CardinalityOne,
		Keys: []RelationKey{{
			SourceAttribute: "sales.order.customer_id",
			TargetAttribute: "customer.customer.customer_id",
		}},
	}

	encoded, err := json.Marshal(relation)
	if err != nil {
		t.Fatalf("RelationのJSON化に失敗しました: %v", err)
	}
	got := string(encoded)
	for _, want := range []string{
		`"sourceAttribute":"sales.order.customer_id"`,
		`"targetAttribute":"customer.customer.customer_id"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("Relation keyのCanonical identityがJSONへ保持されていません: got=%s want=%s", got, want)
		}
	}
}
