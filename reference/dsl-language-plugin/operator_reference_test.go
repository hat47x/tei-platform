package languagereference

import "testing"

func TestTypedOperatorComposition(t *testing.T) {
 catalog:=[]Operator{
  {ID:"std.op.where",Version:"1",InputType:"rows/order",OutputType:"rows/order",Effect:"pure"},
  {ID:"std.op.expand",Version:"1",InputType:"rows/order",OutputType:"rows/order-line",Effect:"pure"},
  {ID:"std.op.summarize",Version:"1",InputType:"rows/order-line",OutputType:"rows/total",Effect:"pure"},
 }
 calls:=[]Call{
  {OperatorID:"std.op.where",InputType:"rows/order"},
  {OperatorID:"std.op.expand",InputType:"rows/order"},
  {OperatorID:"std.op.summarize",InputType:"rows/order-line"},
 }
 got,err:=Compose("rows/order",calls,catalog)
 if err!=nil || got!="rows/total" {t.Fatalf("result %q, %v",got,err)}
}
func TestOperatorCompositionFailsClosed(t *testing.T){
 pure:=Operator{ID:"std.op.where",Version:"1",InputType:"rows/order",OutputType:"rows/order",Effect:"pure"}
 cases:=[]struct{name string; input string; calls []Call; catalog []Operator}{
  {"unknown","rows/order",[]Call{{OperatorID:"unknown",InputType:"rows/order"}},[]Operator{pure}},
  {"ambiguous","rows/order",[]Call{{OperatorID:pure.ID,InputType:"rows/order"}},[]Operator{pure,pure}},
  {"input mismatch","rows/order",[]Call{{OperatorID:pure.ID,InputType:"rows/other"}},[]Operator{pure}},
  {"effectful","rows/order",[]Call{{OperatorID:"std.op.invoke",InputType:"rows/order"}},[]Operator{{ID:"std.op.invoke",Version:"1",InputType:"rows/order",OutputType:"rows/order",Effect:"invoke"}}},
  {"incomplete","rows/order",[]Call{{OperatorID:"std.op.where",InputType:"rows/order"}},[]Operator{{ID:"std.op.where",InputType:"rows/order",OutputType:"rows/order",Effect:"pure"}}},
 }
 for _,tc:=range cases{t.Run(tc.name,func(t *testing.T){_,err:=Compose(tc.input,tc.calls,tc.catalog);if err==nil {t.Fatal("expected rejection")}})}
}
