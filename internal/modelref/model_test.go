package modelref_test

import (
 "encoding/json"
 "reflect"
 "testing"
 "tei/internal/modelref"
 "tei/internal/subject"
)
func k(kind,id string) modelref.Key {return modelref.Key{Scope:"orders-v1",Subject:subject.Subject{Kind:kind,ID:id}}}
func n(kind,id,version string) modelref.Node {return modelref.Node{Key:k(kind,id),Version:version,DisplayName:id}}
func r(key modelref.Key, v string) modelref.Ref{return modelref.Ref{Scope:key.Scope,Subject:key.Subject,Version:v}}
func rules() []modelref.Rule {return []modelref.Rule{
 {Relation:"reads",FromKind:"function",ToKind:"data"},
 {Relation:"renders",FromKind:"form",ToKind:"function",DisplayAffects:true},
 {Relation:"triggers",FromKind:"task",ToKind:"function",AcyclicGroup:"procedure"},
 {Relation:"precedes",FromKind:"task",ToKind:"task",AcyclicGroup:"procedure"},
 {Relation:"depends-on",FromKind:"data",ToKind:"data",AcyclicGroup:"data-dependencies"},
}}
func fixture() modelref.Graph {return modelref.Graph{Schema:"tei.modelref.v0",
 Nodes:[]modelref.Node{n("data","order-line","1"),n("function","submit-order","2"),n("task","confirm-order","1"),n("form","order-form","1")},
 Edges:[]modelref.Edge{
 {From:k("function","submit-order"),To:r(k("data","order-line"),"1"),Relation:"reads"},
 {From:k("task","confirm-order"),To:r(k("function","submit-order"),"2"),Relation:"triggers"},
 {From:k("form","order-form"),To:r(k("function","submit-order"),"2"),Relation:"renders"},
 }}}
func hasCode(ds []modelref.Diagnostic, code string) bool {for _,d:=range ds{if d.Code==code{return true}};return false}
func TestThreeViewsAndSerialization(t *testing.T) {
 g:=fixture()
 if ds:=modelref.Validate(g,rules());len(ds)!=0{t.Fatalf("valid cross-view graph: %+v",ds)}
 b,err:=json.Marshal(g);if err!=nil{t.Fatal(err)}
 var round modelref.Graph
 if err:=json.Unmarshal(b,&round);err!=nil{t.Fatal(err)}
 if !reflect.DeepEqual(g,round){t.Fatal("JSON roundtrip changed graph")}
 renamed:=fixture();renamed.Nodes[0].DisplayName="注文内訳"
 c:=modelref.Diff(g,renamed)
 if len(c)!=1||c[0].Kind!="display-renamed"||c[0].Key!=k("data","order-line"){t.Fatalf("display rename: %+v",c)}
 if impact:=modelref.Impact(g,c,rules());len(impact)!=0{t.Fatalf("a read is not a rendered label: %+v",impact)}
}
func TestNegativeReferences(t *testing.T) {
 for _,tc:=range []struct{name string;change func(*modelref.Graph);want string}{
 {"missing",func(g *modelref.Graph){g.Edges[0].To.Subject.ID="unknown"},"missing-reference"},
 {"wrong-kind",func(g *modelref.Graph){g.Edges[0].To.Subject.Kind="function"},"wrong-kind"},
 {"version",func(g *modelref.Graph){g.Edges[0].To.Version="3"},"version-mismatch"},
 {"bad-edge",func(g *modelref.Graph){g.Edges[0].Relation="writes"},"invalid-edge-type"},
 {"duplicate",func(g *modelref.Graph){g.Nodes=append(g.Nodes,g.Nodes[0])},"duplicate-node-key"},
 {"ambiguous",func(g *modelref.Graph){other:=g.Nodes[0];other.Key.Scope="returns";g.Nodes=append(g.Nodes,other);g.Edges[0].To.Scope=""},"ambiguous-reference"},
 } {
 t.Run(tc.name,func(t *testing.T){g:=fixture();tc.change(&g);if !hasCode(modelref.Validate(g,rules()),tc.want){t.Fatalf("expected %s",tc.want)}})
 }
}
func TestPluginCyclePolicy(t *testing.T) {
 g:=fixture()
 g.Nodes=append(g.Nodes,n("task","pick-order","1"))
 g.Edges=append(g.Edges,modelref.Edge{From:k("task","confirm-order"),To:r(k("task","pick-order"),"1"),Relation:"precedes"},modelref.Edge{From:k("task","pick-order"),To:r(k("task","confirm-order"),"1"),Relation:"precedes"})
 if !hasCode(modelref.Validate(g,rules()),"cycle"){t.Fatal("missing typed cycle diagnostic")}
 allow:=rules();for i:=range allow{allow[i].AcyclicGroup=""}
 if hasCode(modelref.Validate(g,allow),"cycle"){t.Fatal("unconfigured cycle prohibited")}
}
func TestChangesImpactAndLineage(t *testing.T) {
 before:=fixture();after:=fixture()
 after.Nodes[1].DisplayName="送信処理"
 impacts:=modelref.Impact(before,modelref.Diff(before,after),rules())
 if len(impacts)!=1||impacts[0].Consumer!=k("form","order-form"){t.Fatalf("render-only impact: %+v",impacts)}
 after=fixture();after.Nodes[0].Version="2"
 impacts=modelref.Impact(before,modelref.Diff(before,after),rules())
 if len(impacts)!=3{t.Fatalf("transitive impact: %+v",impacts)}
 after=fixture();after.Nodes[0].Key=k("data","order-detail")
 if !hasCode(modelref.Validate(after,rules()),"missing-reference"){t.Fatal("rekey left dangling edge undiagnosed")}
 transition:=modelref.Transition{Kind:"rekey",From:[]modelref.Key{k("data","order-line")},To:[]modelref.Key{k("data","order-detail")}}
 if got:=modelref.ValidateTransition(before,after,transition);got!=""{t.Fatal(got)}
 if got:=modelref.ValidateTransition(before,after,modelref.Transition{Kind:"rekey",To:transition.To});got!="invalid-cardinality"{t.Fatal(got)}
 changes:=modelref.Diff(before,after)
 if len(changes)!=2{t.Fatalf("rekey must be removal + addition, got %+v",changes)}
}
func TestLegacyAndAIProvenance(t *testing.T) {
 g:=fixture()
 g.Nodes[0].Evidence=[]modelref.Evidence{{Authority:"source-observation",SourceToken:"legacy:table#12",ArtifactFingerprint:"sha256:source"}}
 g.Nodes[1].Evidence=[]modelref.Evidence{{Authority:"proposal",Context:"project:orders"}}
 if ds:=modelref.Validate(g,rules());len(ds)!=0{t.Fatalf("non-accepted evidence: %+v",ds)}
 if g.Nodes[0].Key.Subject.ID!="order-line"||g.Nodes[1].Evidence[0].Authority!="proposal"{t.Fatal("authority promoted")}
 g.Nodes[0].Evidence[0].ArtifactFingerprint=""
 if !hasCode(modelref.Validate(g,rules()),"unbound-source-evidence"){t.Fatal("missing source binding accepted")}
 g.Nodes[0].Evidence=[]modelref.Evidence{{Authority:"derived",ArtifactFingerprint:"sha256:source"}}
 if !hasCode(modelref.Validate(g,rules()),"unbound-derivation-evidence"){t.Fatal("missing rule fingerprint accepted")}
}
func TestExplicitSplitAndChangedEdges(t *testing.T) {
 before:=fixture();after:=fixture()
 after.Nodes[0].Key=k("data","order-header")
 after.Nodes=append(after.Nodes,n("data","order-detail","1"))
 split:=modelref.Transition{Kind:"split",From:[]modelref.Key{k("data","order-line")},To:[]modelref.Key{k("data","order-header"),k("data","order-detail")}}
 if got:=modelref.ValidateTransition(before,after,split);got!=""{t.Fatal(got)}
 if got:=modelref.ValidateTransition(before,after,modelref.Transition{Kind:"merge",From:split.From,To:split.To});got!="invalid-cardinality"{t.Fatal(got)}
 after=fixture();after.Edges=after.Edges[1:]
 changes:=modelref.Diff(before,after)
 if len(changes)!=1||changes[0].Kind!="reference-changed"||changes[0].Key!=k("function","submit-order"){t.Fatalf("lost reference edit: %+v",changes)}
 after=fixture();after.Nodes[0].Evidence=[]modelref.Evidence{{Authority:"proposal",Context:"changed"}}
 changes=modelref.Diff(before,after)
 if len(changes)!=1||changes[0].Kind!="evidence-changed"{t.Fatalf("lost evidence change: %+v",changes)}
}
