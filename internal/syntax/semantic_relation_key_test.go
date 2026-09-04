package syntax

import "testing"

// Relation key authoringは既存の1行Relation構文を壊さず、instance traversalが必要なRelationだけを拡張する。
func TestParseSemanticRelationKeepsExistingKeylessSyntax(t *testing.T) {
	ns, err := NewParser(`namespace sales

data Order {
    relation customer to customer.customer one
}
`).ParseSemanticFile()
	if err != nil {
		t.Fatalf("既存Relation構文の解析に失敗しました: %v", err)
	}

	if len(ns.Data) != 1 || len(ns.Data[0].Relations) != 1 {
		t.Fatalf("Relationが期待どおり読み込まれていません: %#v", ns.Data)
	}
	if keys := ns.Data[0].Relations[0].Keys; len(keys) != 0 {
		t.Fatalf("key blockを持たない既存Relationへkeyを追加してはいけません: %#v", keys)
	}
}

func TestParseSemanticRelationReadsExplicitKeyMapping(t *testing.T) {
	ns, err := NewParser(`namespace sales

data Order {
    relation customer to customer.customer one
        key sales.order.customer_id -> customer.customer.customer_id
}
`).ParseSemanticFile()
	if err != nil {
		t.Fatalf("Relation key構文の解析に失敗しました: %v", err)
	}

	relation := ns.Data[0].Relations[0]
	if len(relation.Keys) != 1 {
		t.Fatalf("Relation keyが期待どおり読み込まれていません: %#v", relation.Keys)
	}
	if relation.Keys[0].SourceAttribute != "sales.order.customer_id" || relation.Keys[0].TargetAttribute != "customer.customer.customer_id" {
		t.Fatalf("Relation keyのCanonical identityが変わっています: %#v", relation.Keys[0])
	}
}

func TestParseSemanticRelationPreservesCompositeKeyOrder(t *testing.T) {
	ns, err := NewParser(`namespace sample

data Parent {
    relation children to sample.child many
        key sample.parent.tenant_id -> sample.child.tenant_id
        key sample.parent.parent_id -> sample.child.parent_id
}
`).ParseSemanticFile()
	if err != nil {
		t.Fatalf("複数Relation key構文の解析に失敗しました: %v", err)
	}

	keys := ns.Data[0].Relations[0].Keys
	if len(keys) != 2 {
		t.Fatalf("Relation key数=%d want=2: %#v", len(keys), keys)
	}
	if keys[0].SourceAttribute != "sample.parent.tenant_id" || keys[1].SourceAttribute != "sample.parent.parent_id" {
		t.Fatalf("authoring順序が保持されていません: %#v", keys)
	}
}

func TestParseSemanticRelationRejectsEmptyKeyBlock(t *testing.T) {
	_, err := NewParser(`namespace sales

data Order {
    relation customer to customer.customer one {
    }
}
`).ParseSemanticFile()
	if err == nil {
		t.Fatal("空のRelation key blockを受理してはいけません")
	}
}

func TestParseSemanticRelationRejectsKeyWithoutArrow(t *testing.T) {
	_, err := NewParser(`namespace sales

data Order {
    relation customer to customer.customer one
        key sales.order.customer_id customer.customer.customer_id
}
`).ParseSemanticFile()
	if err == nil {
		t.Fatal("矢印を欠くRelation key mappingを受理してはいけません")
	}
}
