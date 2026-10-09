package actionboundary

import (
 "context"
 "encoding/json"
 "errors"
 "math"
 "sync"
 "testing"
)

const documentFixture = `{"cards":[{"id":"a","position":{"x":0,"y":1},"source":{"id":"s1"},"hold":true},{"id":"b","position":{"x":8,"y":9}}],"reviewer":{"id":"u1"},"edges":[{"kind":"future-edge","metadata":{"keep":true}}],"affiliations":[{"id":"f1"}]}`

func TestJSONDocumentStorePreservesUnrelatedFields(t *testing.T) {
 s:=NewJSONDocumentStore("r1",[]byte(documentFixture))
 h:=Host{Authorizer:permit(true),Bindings:[]Binding{{ApplicationID:"sui",ActionID:"sui.move",Store:s}}}
 rev,err:=h.Commit(context.Background(),Intent{ApplicationID:"sui",DocumentID:"d",ActionID:"sui.move",ExpectedRevision:"r1",Payload:MovePayload{CardID:"a",X:5,Y:6}})
 if err!=nil||rev!="r2" {t.Fatalf("rev=%s err=%v",rev,err)}
 _,got,writes:=s.Snapshot()
 if writes!=1 {t.Fatalf("writes=%d",writes)}
 var before,after map[string]json.RawMessage
 if err:=json.Unmarshal([]byte(documentFixture),&before);err!=nil{t.Fatal(err)}
 if err:=json.Unmarshal(got,&after);err!=nil{t.Fatal(err)}
 for _,key:=range []string{"reviewer","edges","affiliations"} {
  var a,b any
  _=json.Unmarshal(before[key],&a);_=json.Unmarshal(after[key],&b)
  aa,_:=json.Marshal(a);bb,_:=json.Marshal(b)
  if string(aa)!=string(bb) {t.Fatalf("%s changed: %s vs %s",key,aa,bb)}
 }
 var cards []map[string]json.RawMessage
 if err:=json.Unmarshal(after["cards"],&cards);err!=nil{t.Fatal(err)}
 if string(cards[0]["source"])!=`{"id":"s1"}`||string(cards[0]["hold"])!="true" {t.Fatal("card provenance lost")}
 if string(cards[1]["position"])!=`{"x":8,"y":9}` {t.Fatal("unrelated card moved")}
 if string(cards[0]["position"])!=`{"x":5,"y":6}` {t.Fatalf("position=%s",cards[0]["position"])}
}
func TestJSONDocumentStoreRejectsWithoutWrites(t *testing.T) {
 cases:=[]struct{name string;rev string;payload any;want error}{
  {"stale","old",MovePayload{CardID:"a",X:1,Y:2},ErrConflict},
  {"unknown card","r1",MovePayload{CardID:"missing",X:1,Y:2},ErrInvalid},
  {"missing card","r1",MovePayload{X:1,Y:2},ErrInvalid},
  {"wrong payload","r1","untyped",ErrInvalid},
 }
 for _,tc:=range cases {
  t.Run(tc.name,func(t *testing.T){
   s:=NewJSONDocumentStore("r1",[]byte(documentFixture))
   _,err:=s.Commit(context.Background(),Intent{ActionID:"sui.move",ExpectedRevision:tc.rev,Payload:tc.payload})
   if !errors.Is(err,tc.want) {t.Fatalf("err=%v want=%v",err,tc.want)}
   rev,got,writes:=s.Snapshot()
   if rev!="r1"||writes!=0||string(got)!=documentFixture {t.Fatalf("mutated on rejection: %s %d",rev,writes)}
  })
 }
}
func TestJSONDocumentStoreConcurrentRevision(t *testing.T){
 s:=NewJSONDocumentStore("r1",[]byte(documentFixture))
 var wg sync.WaitGroup
 results:=make(chan error,2)
 for i:=0;i<2;i++ {
  wg.Add(1)
  go func(){defer wg.Done();_,err:=s.Commit(context.Background(),Intent{ActionID:"sui.move",ExpectedRevision:"r1",Payload:MovePayload{CardID:"a",X:1,Y:2}});results<-err}()
 }
 wg.Wait();close(results)
 successes,conflicts:=0,0
 for err:=range results {if err==nil {successes++}else if errors.Is(err,ErrConflict){conflicts++}else{t.Fatalf("unexpected %v",err)}}
 _,_,writes:=s.Snapshot()
 if successes!=1||conflicts!=1||writes!=1 {t.Fatalf("success=%d conflict=%d writes=%d",successes,conflicts,writes)}
}

func TestJSONDocumentStoreMalformedAndNonFinite(t *testing.T) {
 cases:=[]struct{name string;document string;payload MovePayload}{
  {"null document","null",MovePayload{CardID:"a",X:1,Y:2}},
  {"null cards",`{"cards":null}`,MovePayload{CardID:"a",X:1,Y:2}},
  {"missing cards",`{"reviewer":"u1"}`,MovePayload{CardID:"a",X:1,Y:2}},
  {"non finite x",documentFixture,MovePayload{CardID:"a",X:math.NaN(),Y:2}},
  {"non finite y",documentFixture,MovePayload{CardID:"a",X:1,Y:math.Inf(1)}},
  {"duplicate card ids",`{"cards":[{"id":"a"},{"id":"a"}]}`,MovePayload{CardID:"a",X:1,Y:2}},
 }
 for _,tc:=range cases {
  t.Run(tc.name,func(t *testing.T){
   store:=NewJSONDocumentStore("r1",[]byte(tc.document))
   _,err:=store.Commit(context.Background(),Intent{ActionID:"sui.move",ExpectedRevision:"r1",Payload:tc.payload})
   if !errors.Is(err,ErrInvalid){t.Fatalf("expected invalid, got %v",err)}
   revision,document,writes:=store.Snapshot()
   if revision!="r1"||writes!=0||string(document)!=tc.document {t.Fatal("invalid input mutated state")}
  })
 }
}
