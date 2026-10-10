package crossview

import (
 "testing"
 "tei/internal/application"
)

func TestPatternRequirementsExplicitOnly(t *testing.T) {
 app:=application.Application{ID:"order_entry"}
 p:=application.Pattern{ID:"customer_reference",Kind:application.PatternReference,Relation:"customer.customer",Datasource:"master_data",ContractQuery:&application.ContractQueryInvocation{Contract:"master.customer",Query:"find_by_id"}}
 req:=PatternRequirements(app,p)
 if len(req)!=4 {t.Fatalf("got %d requirements, want 4: %#v",len(req),req)}
 want:=[]string{"pattern-validation","relation-reference","datasource-declaration","contract-query-reference"}
 for i, expected:=range want {
  if req[i].Relation!=expected {t.Fatalf("requirement[%d]=%q want %q",i,req[i].Relation,expected)}
 }
 for _, r:=range req {
  if r.Relation=="write-authority" || r.Relation=="responsibility" {t.Fatalf("invented authority: %#v",r)}
 }
}

func TestPatternRequirementsNoInferredCRUD(t *testing.T) {
 app:=application.Application{ID:"orders"}
 p:=application.Pattern{ID:"save",Kind:application.PatternRecord,Data:"sales.order"}
 req:=PatternRequirements(app,p)
 if len(req)!=2 {t.Fatalf("unexpected requirements: %#v",req)}
 for _, r:=range req {
  if r.Relation=="write-authority" || r.Relation=="business-crud" {t.Fatalf("inferred write from kind: %#v",r)}
 }
}
