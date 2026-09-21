package changeanalysis

import (
	"reflect"
	"strings"
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
)

func addConceptualRelation(t *testing.T) (before, after ir.World) {
	t.Helper()
	before = loadWorld(t, referenceWorkspaceCopy(t)).World
	after = loadWorld(t, referenceWorkspaceCopy(t)).World
	for i := range after.Semantics {
		for d := range after.Semantics[i].Data {
			if after.Semantics[i].Data[d].ID == "sales.order_line" {
				after.Semantics[i].Data[d].Relations = append(after.Semantics[i].Data[d].Relations, semantic.Relation{
					ID: "sales.order_line.order", Symbol: "order", Target: "sales.order", Cardinality: semantic.CardinalityOne,
				})
			}
		}
	}
	return before, after
}

func viaSet(edges []ImpactEdge) map[string]bool {
	out := map[string]bool{}
	for _, e := range edges {
		out[e.From+" -> "+e.To+" via "+e.Via] = true
	}
	return out
}

func unresolvedByCode(unresolved []Unresolved) map[string]string {
	out := map[string]string{}
	for _, u := range unresolved {
		out[u.Code] = u.Message
	}
	return out
}

func TestAnAddedRelationBetweenPersistedDataLeavesItsStorageUndecided(t *testing.T) {
	before, after := addConceptualRelation(t)
	diff, err := Diff(before, after)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := keys(diff), []string{"added relation:sales.order_line.order"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("diff=%v want %v", got, want)
	}
	edges, unresolved := Impact(after, diff)
	got := viaSet(edges)
	for _, want := range []string{
		"relation:sales.order_line.order -> data:sales.order_line via semantic.data.relation",
		"relation:sales.order_line.order -> data:sales.order via semantic.relation.target",
		"relation:sales.order_line.order -> persistence-mapping:sales_core.order_line via persistence.mapping.realizes-endpoint",
		"relation:sales.order_line.order -> persistence-mapping:sales_core.order via persistence.mapping.realizes-endpoint",
	} {
		if !got[want] {
			t.Fatalf("辺 %q がありません: %v", want, edges)
		}
	}
	if len(edges) != 4 {
		t.Fatalf("宣言された参照以外の辺が混じっています（%d本）: %v", len(edges), edges)
	}
	byCode := unresolvedByCode(unresolved)
	message, ok := byCode["RELATION_STORAGE_NOT_AUTHORED"]
	if !ok || !strings.Contains(message, "sales_core.order") || !strings.Contains(message, "sales_core.order_line") || !strings.Contains(message, "sales.order_line.order") {
		t.Fatalf("保存方法が未決定であることが、対象のMappingとともに示されていません: %v", byCode)
	}
}

func TestARelationWithRelationStorageIsNotLeftUndecided(t *testing.T) {
	world := loadWorld(t, referenceWorkspaceCopy(t)).World
	edges, unresolved := Impact(world, []SubjectChange{{Kind: "relation", ID: "sales.order.lines", Change: "changed"}})
	if !viaSet(edges)["relation:sales.order.lines -> persistence-mapping:sales_core.order_line via persistence.relation-storage"] {
		t.Fatalf("relation storageを定義したMappingへの辺がありません: %v", edges)
	}
	if _, undecided := unresolvedByCode(unresolved)["RELATION_STORAGE_NOT_AUTHORED"]; undecided {
		t.Fatalf("relation storageが定義済みなのに未決定として残っています: %v", unresolved)
	}
}

func TestARelationWhoseKeyColumnsArePersistedIsDerivableFromThem(t *testing.T) {
	world := loadWorld(t, referenceWorkspaceCopy(t)).World
	edges, unresolved := Impact(world, []SubjectChange{{Kind: "relation", ID: "sales.order.customer", Change: "changed"}})
	if !viaSet(edges)["relation:sales.order.customer -> persistence-mapping:sales_core.order via persistence.relation-key-columns"] {
		t.Fatalf("keyの列から保存を導ける辺がありません: %v", edges)
	}
	if _, undecided := unresolvedByCode(unresolved)["RELATION_STORAGE_NOT_AUTHORED"]; undecided {
		t.Fatalf("keyのsource Attributeが列として実現されているのに未決定として残っています: %v", unresolved)
	}
}

func TestARelationBetweenUnpersistedDataHasNoPersistenceImpact(t *testing.T) {
	before, after := worldWithSecondApplication(t)
	diff, err := Diff(before, after)
	if err != nil {
		t.Fatal(err)
	}
	edges, unresolved := Impact(after, diff)
	for _, e := range edges {
		if strings.HasPrefix(e.To, "persistence-mapping:") {
			t.Fatalf("persistされていないDataのRelationからPersistence Mappingへの辺があります: %+v", e)
		}
	}
	if _, undecided := unresolvedByCode(unresolved)["RELATION_STORAGE_NOT_AUTHORED"]; undecided {
		t.Fatalf("persistされていないDataのRelationを保存未決定として扱っています: %v", unresolved)
	}
}

func TestAKeyOnlyDerivesStorageWhenItsSourceAttributeIsAColumnOfTheSourceMapping(t *testing.T) {
	world := loadWorld(t, referenceWorkspaceCopy(t)).World
	// keyのsource Attributeが、source側Mappingの列として実現されていない場合は導けない。
	for i := range world.Semantics {
		for d := range world.Semantics[i].Data {
			if world.Semantics[i].Data[d].ID != "sales.order_line" {
				continue
			}
			world.Semantics[i].Data[d].Relations = append(world.Semantics[i].Data[d].Relations, semantic.Relation{
				ID: "sales.order_line.order", Symbol: "order", Target: "sales.order", Cardinality: semantic.CardinalityOne,
				Keys: []semantic.RelationKey{{SourceAttribute: "sales.order_line.quantity", TargetAttribute: "sales.order.order_id"}},
			})
		}
	}
	edges, unresolved := Impact(world, []SubjectChange{{Kind: "relation", ID: "sales.order_line.order", Change: "added"}})
	// quantityはsales_core.order_lineの列なので導ける。
	if !viaSet(edges)["relation:sales.order_line.order -> persistence-mapping:sales_core.order_line via persistence.relation-key-columns"] {
		t.Fatalf("keyの列から保存を導ける辺がありません: %v", edges)
	}
	if _, undecided := unresolvedByCode(unresolved)["RELATION_STORAGE_NOT_AUTHORED"]; undecided {
		t.Fatalf("導けるのに未決定として残っています: %v", unresolved)
	}

	// 列として実現されていないAttributeをkeyのsourceにすると導けない。
	for i := range world.Semantics {
		for d := range world.Semantics[i].Data {
			for r := range world.Semantics[i].Data[d].Relations {
				if world.Semantics[i].Data[d].Relations[r].ID == "sales.order_line.order" {
					world.Semantics[i].Data[d].Relations[r].Keys = []semantic.RelationKey{{SourceAttribute: "sales.order_line.not_a_column", TargetAttribute: "sales.order.order_id"}}
				}
			}
		}
	}
	_, unresolved = Impact(world, []SubjectChange{{Kind: "relation", ID: "sales.order_line.order", Change: "added"}})
	if _, undecided := unresolvedByCode(unresolved)["RELATION_STORAGE_NOT_AUTHORED"]; !undecided {
		t.Fatalf("列として実現されていないkeyからも保存を導けるとしています: %v", unresolved)
	}
}
