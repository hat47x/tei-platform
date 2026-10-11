package disclosureref

import (
 "encoding/json"
 "errors"
 "reflect"
 "strings"
 "testing"
)

type fake func(Selector, Binding, Manifest) (Decision,error)
func (f fake) Decide(s Selector,b Binding,m Manifest)(Decision,error){return f(s,b,m)}
func base()(Request,Raw,Decision){
 s:=Selector{Operation:"trace",Kind:"semantic-data",ID:"sales.order",GraphScope:"graph-A",Direction:"both",Mode:"summary"}
 b:=Binding{Principal:"alice",Caller:"local-cli",Workspace:"w1",Tenant:"t1",Channel:"console",Destination:"terminal:A",Purpose:"authoring"}
 q:=Request{Selector:s,Binding:b,Now:100,Requested:[]string{"subject","canonical","source","related","unresolved","count"}}
 r:=Raw{Kind:s.Kind,ID:s.ID,Canonical:json.RawMessage(`{"name":"sales.order"}`),
  Sources:[]string{"secret/source.tei#sha256:AAA"},Related:[]string{"secret/neighbor.tei"},
  Unresolved:[]string{"secret-unresolved"},Total:4,DataSnapshot:"private-snapshot",SourceProvider:"workspace-file"}
 d:=Decision{Allowed:true,Current:true,Bound:b,Selector:s,AuthorizedSnapshot:r.DataSnapshot,
  PolicyRevision:"p1",DecisionRef:"host-receipt",ReusePartition:"partition-A",ExpiresAt:200,
  Sections:[]string{"subject","canonical","source","related","unresolved","count"}}
 return q,r,d
}
func run(q Request,r Raw,d Decision)(Public,Audit,*Receipt){
 return Project(q,r,fake(func(Selector,Binding,Manifest)(Decision,error){return d,nil}))
}
func marshal(t *testing.T,p Public)string{
 t.Helper()
 b,err:=json.Marshal(p);if err!=nil{t.Fatal(err)}
 return string(b)
}

func TestAllowedViewsAcrossSurfaces(t *testing.T){
 q,r,d:=base()
 for _,channel:=range []string{"console","web","desktop","llm"}{
  q.Binding.Channel=channel;q.Binding.Destination="approved:"+channel;d.Bound=q.Binding
  p,a,rec:=run(q,r,d)
  if p.Status!="available"||p.Completeness!="complete"||p.Total==nil||*p.Total!=4||
   a.Outcome!="allow"||rec==nil{t.Fatalf("%s: %+v %+v",channel,p,a)}
 }
}
func TestDeniedAndMissingHaveIdenticalPublicOutput(t *testing.T){
 q,r,d:=base();d.Allowed=false
 p,_,_:=run(q,r,d)
 missing:=r;missing.ID="unknown"
 absent,_,_:=run(q,missing,d)
 if !reflect.DeepEqual(p,absent)||strings.Contains(marshal(t,p),"sales"){t.Fatalf("existence leak %s",marshal(t,p))}
}
func TestPartialNoninterferenceAndExistenceSensitive(t *testing.T){
 q,r,d:=base()
 q.Requested=[]string{"subject","canonical","source","related","unresolved"}
 d.Sections=[]string{"subject","canonical"};d.PartialAllowed=true
 p,_,_:=run(q,r,d)
 if p.Status!="available"||p.Completeness!="partial"{t.Fatalf("unexpected partial: %+v",p)}
 r2:=r;r2.Sources=[]string{"another/hidden"};r2.Related=[]string{"different/neighbor"};r2.Unresolved=[]string{"else"};r2.Total=555
 p2,_,_:=run(q,r2,d)
 if marshal(t,p)!=marshal(t,p2){t.Fatal("hidden values influenced public view")}
 for _,secret:=range []string{"secret","p1","private-snapshot","partition-A"}{
  if strings.Contains(marshal(t,p),secret){t.Fatalf("leaked %q",secret)}
 }
 d.ExistenceSensitive=true
 forbidden,_,_:=run(q,r,d)
 if forbidden.Status!="unavailable"{t.Fatal("existence sensitive leak")}
}
func TestPartialGlobalCountCannotBeVisibleCount(t *testing.T){
 q,r,d:=base();d.Sections=[]string{"subject","count"};d.PartialAllowed=true
 p,_,_:=run(q,r,d);if p.Status!="unavailable"||p.Total!=nil{t.Fatalf("count leaked: %+v",p)}
}
func TestFailClosedCallerScopeDestinationPolicy(t *testing.T){
 cases:=map[string]func(*Request,*Decision){
  "missing-principal":func(q *Request,d *Decision){q.Binding.Principal=""},
  "cross-tenant":func(q *Request,d *Decision){q.Binding.Tenant="t2"},
  "cross-workspace":func(q *Request,d *Decision){q.Binding.Workspace="w2"},
  "caller-changed":func(q *Request,d *Decision){q.Binding.Caller="agent"},
  "graph-scope-forgery":func(q *Request,d *Decision){q.Selector.GraphScope="other"},
  "llm-egress":func(q *Request,d *Decision){q.Binding.Channel="llm";q.Binding.Destination="new-provider/acct"},
  "purpose-changed":func(q *Request,d *Decision){q.Binding.Purpose="training"},
  "no-revision":func(q *Request,d *Decision){d.PolicyRevision=""},
  "expired":func(q *Request,d *Decision){q.Now=250},
  "revoked":func(q *Request,d *Decision){d.Current=false},
  "no-decision-receipt":func(q *Request,d *Decision){d.DecisionRef=""},
  "no-partition":func(q *Request,d *Decision){d.ReusePartition=""},
 }
 for name,change:=range cases{t.Run(name,func(t *testing.T){
  q,r,d:=base();change(&q,&d)
  p,_,rec:=run(q,r,d);if p.Status!="unavailable"||rec!=nil{t.Fatalf("fail-open: %+v",p)}
 })}
}
func TestContinuationReauthorizesPolicyAndDestination(t *testing.T){
 q,r,d:=base();_,_,receipt:=run(q,r,d);if receipt==nil{t.Fatal("no prior receipt")}
 q.Previous=receipt
 cases:=map[string]func(*Request,*Raw,*Decision){
  "policy-revision":func(q *Request,r *Raw,d *Decision){d.PolicyRevision="p2"},
  "reuse-partition":func(q *Request,r *Raw,d *Decision){d.ReusePartition="partition-B"},
  "data-snapshot":func(q *Request,r *Raw,d *Decision){r.DataSnapshot="changed";d.AuthorizedSnapshot=r.DataSnapshot},
  "egress":func(q *Request,r *Raw,d *Decision){q.Binding.Destination="llm/other";d.Bound=q.Binding},
 }
 for name,change:=range cases{t.Run(name,func(t *testing.T){
  q2,r2,d2:=q,r,d;change(&q2,&r2,&d2)
  p,_,next:=run(q2,r2,d2);if p.Status!="unavailable"||next!=nil{t.Fatalf("stale page allowed: %+v",p)}
 })}
 q.Selector.Mode="full";d.Selector=q.Selector
 p,_,next:=run(q,r,d)
 if p.Status!="available"||next==nil{t.Fatal("same subject summary->full should reauthorize successfully")}
 q.Selector.Direction="incoming";d.Selector=q.Selector
 p,_,_=run(q,r,d);if p.Status!="unavailable"{t.Fatal("different direction reused continuation")}
}
func TestManifestCoverageAndProviderErrorOpaque(t *testing.T){
 q,r,d:=base()
 observed:=false
 a:=fake(func(s Selector,b Binding,m Manifest)(Decision,error){
  observed=true
  if s!=q.Selector||b!=q.Binding||m.SourceProvider!=r.SourceProvider||
   m.CanonicalSubject!="semantic-data:sales.order"||!reflect.DeepEqual(m.Sources,r.Sources)||
   !reflect.DeepEqual(m.Related,r.Related)||!reflect.DeepEqual(m.Unresolved,r.Unresolved){
   t.Fatalf("missing resource manifest: %+v",m)
  }
  return d,nil
 })
 p,_,_:=Project(q,r,a);if !observed||p.Status!="available"{t.Fatal("manifest/decision failed")}
 r.DataSnapshot="changed"
 p,_,_=Project(q,r,a);if p.Status!="unavailable"{t.Fatal("old decision reused")}
 failing:=fake(func(Selector,Binding,Manifest)(Decision,error){
  return Decision{},errors.New("private source path /secret/authorizer")
 })
 p,_,_=Project(q,r,failing)
 if p.Status!="unavailable"||strings.Contains(marshal(t,p),"secret"){t.Fatal("raw provider error leaked")}
}
func TestDisclosureSelectionDoesNotGrantAuthority(t *testing.T){
 q,r,d:=base();q.Requested=[]string{"subject","source"};d.Sections=[]string{"subject"}
 p,_,_:=run(q,r,d)
 if p.Status!="unavailable"{t.Fatal("selective Knowledge Disclosure treated as authorization")}
}
