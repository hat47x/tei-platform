package derivationref

import (
 "errors"
 "testing"
)

func TestMultiply(t *testing.T){
 cases:=[]struct{name string;a,b Input;want string;state InputState;err error}{
  {"amount",Input{Present,"2"},Input{Present,"300"},"600",Present,nil},
  {"zero",Input{Present,"0"},Input{Present,"300"},"0",Present,nil},
  {"mixed denominator",Input{Present,"0.2"},Input{Present,"0.5"},"0.1",Present,nil},
  {"fraction",Input{Present,"1.25"},Input{Present,"0.4"},"0.5",Present,nil},
  {"null",Input{Null,""},Input{Present,"300"},"",Null,nil},
  {"missing",Input{Missing,""},Input{Present,"300"},"","",ErrInput},
  {"absent relation",Input{AbsentRelation,""},Input{Present,"300"},"","",ErrInput},
  {"multiple relations",Input{MultipleRelations,""},Input{Present,"300"},"","",ErrInput},
  {"invalid",Input{Present,"abc"},Input{Present,"300"},"","",ErrOperator},
 }
 for _,tc:=range cases {t.Run(tc.name,func(t *testing.T){
  got,err:=Multiply(tc.a,tc.b)
  if !errors.Is(err,tc.err){t.Fatalf("error %v, want %v",err,tc.err)}
  if tc.err==nil&&(got.State!=tc.state||got.Decimal!=tc.want){t.Fatalf("got %#v, want state %q decimal %q",got,tc.state,tc.want)}
 })}
}

func TestValidateGraph(t *testing.T){
 defs:=[]Definition{{ID:"a",Pure:true},{ID:"b",Pure:true}}
 if err:=ValidateGraph(defs,map[string][]string{"a":{"b"}});err!=nil{t.Fatal(err)}
 if err:=ValidateGraph(defs,map[string][]string{"a":{"b"},"b":{"a"}});!errors.Is(err,ErrCycle){t.Fatalf("cycle: %v",err)}
 if err:=ValidateGraph([]Definition{{ID:"a",Pure:false}},nil);!errors.Is(err,ErrImpure){t.Fatalf("impure: %v",err)}
 if err:=ValidateGraph(defs,map[string][]string{"a":{"unknown"}});err==nil{t.Fatal("unknown dependency accepted")}
}

func TestBindingsDoNotMutateDefinition(t *testing.T){
 rule:=Definition{ID:"line-amount",Inputs:[]string{"quantity","unit-price"},Operator:"multiply",Pure:true}
 bindings:=[]Binding{{RuleID:rule.ID,TargetID:"line.amount",Scope:"semantic"},{RuleID:rule.ID,TargetID:"query.total",Scope:"query"}}
 if bindings[0].RuleID!=bindings[1].RuleID||bindings[0].TargetID==bindings[1].TargetID{t.Fatal("rule identity and binding target were conflated")}
}
