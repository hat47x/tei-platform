package crossview

import "testing"

func TestVerifyBase(t *testing.T) {
 world:=SourceRevision{Scope:"world-conformance",Digest:"sha256:world1"}
 business:=SourceRevision{Scope:"business-authoritative-source-set",Digest:"sha256:business1"}
 base:=EvidenceBase{World:world,Business:&business}
 cases:=[]struct{name string; evidence,current EvidenceBase; requireBusiness bool; want Status}{
  {"same",base,base,true,Applicable},
  {"world changed",base,EvidenceBase{World:SourceRevision{Scope:world.Scope,Digest:"sha256:world2"},Business:&business},true,Stale},
  {"business changed",base,EvidenceBase{World:world,Business:&SourceRevision{Scope:business.Scope,Digest:"sha256:business2"}},true,Stale},
  {"business required missing",EvidenceBase{World:world},EvidenceBase{World:world},true,Unresolved},
  {"missing world",EvidenceBase{},base,true,Unresolved},
  {"business removed",base,EvidenceBase{World:world},false,Stale},
  {"world-only",EvidenceBase{World:world},EvidenceBase{World:world},false,Applicable},
 }
 for _,tc:=range cases {t.Run(tc.name,func(t *testing.T){
  if got:=VerifyBase(tc.evidence,tc.current,tc.requireBusiness);got!=tc.want {t.Fatalf("got %q want %q",got,tc.want)}
 })}
}

func TestEvaluateWithBaseCannotBypassStale(t *testing.T) {
 required:=[]Requirement{{Subject:"pattern:p",Relation:"pattern-validation",Owner:"validation"}}
 checks:=[]Check{{Subject:"pattern:p",Relation:"pattern-validation",Owner:"validation",Evidence:"world1",Status:Applicable,Required:true}}
 e:=EvidenceBase{World:SourceRevision{Scope:"world",Digest:"one"}}
 c:=EvidenceBase{World:SourceRevision{Scope:"world",Digest:"two"}}
 if got:=EvaluateWithBase(required,checks,e,c,false).Outcome;got!="blocked-stale" {t.Fatalf("got %q",got)}
}
