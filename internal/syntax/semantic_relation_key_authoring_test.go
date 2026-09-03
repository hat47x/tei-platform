package syntax_test

import (
	"testing"

	"tei/internal/syntax"
)

func TestParseSemanticRelationKeepsOrderedKeys(t *testing.T) {
	src := `namespace sample

type TenantId from std.identifier
type ItemId from std.identifier

data Source {
    attribute tenantId type sample.tenant_id
    attribute itemId type sample.item_id
    relation target to sample.target one
        key sample.source.tenant_id -> sample.target.tenant_id
        key sample.source.item_id -> sample.target.item_id
}

data Target {
    attribute tenantId type sample.tenant_id
    attribute itemId type sample.item_id
}`

	ns, err := syntax.NewParser(src).ParseSemanticFile()
	if err != nil {
		t.Fatalf("Relation keyを含むSemantic sourceを読み取れません: %v", err)
	}
	if len(ns.Data) != 2 || len(ns.Data[0].Relations) != 1 {
		t.Fatalf("Relation構造が期待値と異なります: %#v", ns.Data)
	}

	keys := ns.Data[0].Relations[0].Keys
	if len(keys) != 2 {
		t.Fatalf("Relation key数が変わっています: %#v", keys)
	}
	if keys[0].SourceAttribute != "sample.source.tenant_id" || keys[0].TargetAttribute != "sample.target.tenant_id" {
		t.Fatalf("先頭keyのauthor順を保持できていません: %#v", keys)
	}
	if keys[1].SourceAttribute != "sample.source.item_id" || keys[1].TargetAttribute != "sample.target.item_id" {
		t.Fatalf("後続keyのauthor順を保持できていません: %#v", keys)
	}
}

func TestParseSemanticRelationKeepsLegacyRelationWithoutKeys(t *testing.T) {
	src := `namespace sample

data Source {
    relation target to sample.target one
}

data Target {
}`

	ns, err := syntax.NewParser(src).ParseSemanticFile()
	if err != nil {
		t.Fatalf("従来のRelation構文を読み取れません: %v", err)
	}
	if got := ns.Data[0].Relations[0].Keys; len(got) != 0 {
		t.Fatalf("key未定義Relationへ暗黙のkeyを追加しています: %#v", got)
	}
}
