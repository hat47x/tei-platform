package crossview

import (
 "reflect"
 "testing"
)

func TestEvaluateRequiredChecks(t *testing.T) {
 cases := []struct{
  name string
  checks []Check
  want string
 }{
  {"read-only applicable", []Check{{Subject:"function:customer-reference",Relation:"read-contract",Owner:"validation",Evidence:"source@rev1",Status:Applicable,Required:true}}, "eligible-for-proposal"},
  {"edit incompatible", []Check{{Subject:"datasource:customer",Relation:"write-authority",Owner:"datasource",Evidence:"source@rev1",Status:Incompatible,Required:true}}, "blocked-incompatible"},
  {"missing business owner", []Check{{Subject:"business:customer-maintenance",Relation:"responsibility",Status:Unresolved,Required:true}}, "blocked-unresolved"},
  {"stale wins", []Check{{Subject:"datasource:customer",Relation:"write-authority",Owner:"datasource",Evidence:"source@rev1",Status:Incompatible,Required:true},{Subject:"business:procedure",Relation:"responsibility",Owner:"business",Evidence:"source@rev0",Status:Stale,Required:true}}, "blocked-stale"},
  {"missing provenance", []Check{{Subject:"function:reference",Relation:"read-contract",Owner:"validation",Status:Applicable,Required:true}}, "blocked-unresolved"},
  {"unknown status", []Check{{Subject:"function:reference",Relation:"read-contract",Owner:"validation",Evidence:"source@rev1",Status:"other",Required:true}}, "blocked-unresolved"},
  {"optional does not block", []Check{{Subject:"function:reference",Relation:"read-contract",Owner:"validation",Evidence:"source@rev1",Status:Applicable,Required:true},{Subject:"business:optional",Relation:"context",Status:Unresolved,Required:false}}, "eligible-for-proposal"},
 }
 for _, tc := range cases {
  t.Run(tc.name,func(t *testing.T) {
   got:=Evaluate(tc.checks)
   if got.Outcome!=tc.want {t.Fatalf("outcome=%q want %q",got.Outcome,tc.want)}
  })
 }
}

func TestEvaluateDeterministicAndNoMutation(t *testing.T) {
 a:=Check{Subject:"z",Relation:"dependency",Owner:"validation",Evidence:"r1",Status:Applicable,Required:true}
 b:=Check{Subject:"a",Relation:"dependency",Owner:"validation",Evidence:"r1",Status:Applicable,Required:true}
 input:=[]Check{a,b}
 first:=Evaluate(input)
 second:=Evaluate([]Check{b,a})
 if !reflect.DeepEqual(first,second) {t.Fatalf("order-dependent result: %#v %#v",first,second)}
 if !reflect.DeepEqual(input,[]Check{a,b}) {t.Fatal("mutated caller input")}
}
