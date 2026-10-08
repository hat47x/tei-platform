package languagereference

import (
 "testing"
 "tei/internal/core"
 "tei/internal/semantic"
)

func TestSemanticDataBinding(t *testing.T) {
 ns:=semantic.Namespace{ID:core.CanonicalID("sales"),Data:[]semantic.Data{{ID:core.CanonicalID("sales.order"),Symbol:"Order"}}}
 catalog:=[]Operator{{ID:"std.op.where",Version:"1",InputType:"rows/sales.order",OutputType:"rows/sales.order",Effect:"pure"}}
 calls:=[]Call{{OperatorID:"std.op.where",InputType:"rows/sales.order"}}
 got,err:=BindSemanticInput(ns,"sales.order",catalog,calls)
 if err!=nil || got!="rows/sales.order" {t.Fatalf("got %q: %v",got,err)}
}
func TestSemanticBindingRejectsMissingAndDuplicateData(t *testing.T) {
 item:=semantic.Data{ID:core.CanonicalID("sales.order")}
 catalog:=[]Operator{{ID:"std.op.where",Version:"1",InputType:"rows/sales.order",OutputType:"rows/sales.order",Effect:"pure"}}
 calls:=[]Call{{OperatorID:"std.op.where",InputType:"rows/sales.order"}}
 for _,ns:=range []semantic.Namespace{
  {ID:core.CanonicalID("sales")},
  {ID:core.CanonicalID("sales"),Data:[]semantic.Data{item,item}},
 } {
  if _,err:=BindSemanticInput(ns,"sales.order",catalog,calls);err==nil {t.Fatalf("invalid semantic input accepted: %#v",ns)}
 }
}
