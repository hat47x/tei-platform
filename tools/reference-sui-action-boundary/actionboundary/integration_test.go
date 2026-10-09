package actionboundary

import (
 "context"
 "errors"
 "testing"

 "tei/internal/application"
 "tei/internal/core"
 "tei/internal/driver"
 "tei/internal/security/legacyauthorization"
 "tei/internal/semantic"
)

// TestNativeActionPath joins the previously isolated command, authorization,
// and atomic document boundaries. All dependencies are still reference fakes;
// this does not demonstrate a browser event or SUI DocumentV1 integration.
func TestNativeActionPath(t *testing.T) {
 command:=driver.PatternResolution{
  Application:application.Application{ID:core.CanonicalID("sui")},
  Pattern:application.Pattern{ID:core.CanonicalID("sui.move"),Kind:application.PatternCommand},
  Capability:&semantic.Capability{ID:core.CanonicalID("sui.move_capability")},
 }
 authCapability:=semantic.Capability{
  ID:legacyauthorization.AuthorizationCapability,
  Outcomes:[]semantic.Outcome{{Name:"allowed",Payload:"auth.grant"},{Name:"denied",Payload:"auth.denial"}},
 }
 newHost:=func(outcome core.LocalName)(Host,*JSONDocumentStore){
  store:=NewJSONDocumentStore("r1",[]byte(documentFixture))
  host:=Host{
   CommandGate:&CanonicalCommandGate{Resolver:fakePatternResolver{resolution:command}},
   Authorizer:CapabilityAuthorizer{
    Subjects:fixedSubject{id:"trusted-subject"},
    Capability:authCapability,
    Adapter:authorizationAdapter{outcome:outcome},
   },
   Bindings:[]Binding{{ApplicationID:"sui",ActionID:"sui.move",Store:store}},
  }
  return host,store
 }
 intent:=Intent{
  ApplicationID:"sui",DocumentID:"doc",ActionID:"sui.move",
  ExpectedRevision:"r1",Payload:MovePayload{CardID:"a",X:12,Y:34},
 }
 t.Run("authorized commit then stale rejection",func(t *testing.T){
  host,store:=newHost("allowed")
  rev,err:=host.Commit(context.Background(),intent)
  if err!=nil||rev!="r2" {t.Fatalf("revision=%s err=%v",rev,err)}
  _,doc,writes:=store.Snapshot()
  if writes!=1 {t.Fatalf("writes=%d",writes)}
  if len(doc)==0 {t.Fatal("document missing")}
  _,err=host.Commit(context.Background(),intent)
  if !errors.Is(err,ErrConflict) {t.Fatalf("expected conflict, got %v",err)}
  _,_,writes=store.Snapshot()
  if writes!=1 {t.Fatalf("stale attempt wrote: %d",writes)}
 })
 t.Run("denied never invokes store",func(t *testing.T){
  host,store:=newHost("denied")
  _,err:=host.Commit(context.Background(),intent)
  if !errors.Is(err,ErrDenied) {t.Fatalf("expected denial, got %v",err)}
  rev,doc,writes:=store.Snapshot()
  if rev!="r1"||writes!=0||string(doc)!=documentFixture {t.Fatal("denied request mutated document")}
 })
 t.Run("other application never invokes store",func(t *testing.T){
  host,store:=newHost("allowed")
  foreign:=intent;foreign.ApplicationID="other"
  _,err:=host.Commit(context.Background(),foreign)
  if !errors.Is(err,ErrDenied) {t.Fatalf("expected denial, got %v",err)}
  _,_,writes:=store.Snapshot()
  if writes!=0 {t.Fatalf("cross-application request wrote: %d",writes)}
 })
}
