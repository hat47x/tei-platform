package main

import (
	"reflect"
	"testing"
)

func previewSnapshots() ([]navigationSnapshot,[]navigationSnapshot) {
	base:=[]navigationSnapshot{
		{File:"procedure.json",SHA256:"sha256:stable-procedure"},
		{File:"review.json",SHA256:"sha256:old-review"},
		{File:"set.json",SHA256:"sha256:stable-set"},
	}
	next:=[]navigationSnapshot{
		{File:"procedure.json",SHA256:"sha256:stable-procedure"},
		{File:"next.json",SHA256:"sha256:new-review"},
		{File:"set.json",SHA256:"sha256:stable-set"},
	}
	return base,next
}

func TestReviewPreviewIsNotAcceptedSourceMutation(t *testing.T) {
	f,before,s,w:=sample()
	after:=before
	after.Triggers=append([]reviewTrigger{},before.Triggers...)
	after.Triggers[0].Label="new human review reason"
	after.InformationFlows=[]informationFlow{{
		ID:"INF-1",From:informationEndpoint{Kind:"party",ID:"customer"},
		To:informationEndpoint{Kind:"step",ID:"BP-04"},Payload:"requested delivery date",
	}}
	after.ExternalParties=[]externalParty{{ID:"customer",Label:"Customer"}}
	p:=build(f,before,s,w,"procedure.json","review.json","set.json")
	q:=build(f,after,s,w,"procedure.json","next.json","set.json")
	base,next:=previewSnapshots()
	out,err:=previewReview(before,after,p,q,base,next)
	if err!=nil{t.Fatal(err)}
	if out.AcceptanceStatus!="not-accepted-not-a-write-candidate" || out.ReadyForAcceptance ||
		len(out.SourceWrites)!=0 || out.WorldRevisionStatus!="unverified-no-model-revision-binding" {
		t.Fatalf("unaccepted DFD proposal was treated as Source Acceptance: %+v",out)
	}
	if out.ProcedureSourceSHA256!=base[0].SHA256 ||
		out.HandoffSourceSHA256!=base[2].SHA256 ||
		out.BaseReviewSHA256!=base[1].SHA256 || out.CandidateReviewSHA256!=next[1].SHA256 {
		t.Fatalf("source review fingerprints lost: %+v",out)
	}
	if len(out.Changes)!=3 {
		t.Fatalf("expected role/trigger/info proposal edits only, got %+v",out.Changes)
	}
	var ids []string
	for _,c:=range out.Changes{ids=append(ids,c.Category+":"+c.Action)}
	want:=[]string{"external-party:proposed-add","information-flow:proposed-add","trigger:proposed-update"}
	if !reflect.DeepEqual(ids,want) {
		t.Fatalf("unexpected or nondeterministic preview rows: got=%v want=%v",ids,want)
	}
}

func TestReviewPreviewRejectsProcedureOrHandoffRevisionDrift(t *testing.T) {
	f,before,s,w:=sample()
	p:=build(f,before,s,w,"procedure.json","review.json","set.json")
	base,next:=previewSnapshots()
	next[0].SHA256="sha256:another-procedure"
	if _,err:=previewReview(before,before,p,p,base,next);err==nil {
		t.Fatal("changed Procedure authoring source was compared as same meaning")
	}
	next[0]=base[0]
	next[2].SHA256="sha256:another-set"
	if _,err:=previewReview(before,before,p,p,base,next);err==nil {
		t.Fatal("changed Handoff set was compared as same meaning")
	}
}

func TestReviewPreviewReportsCandidateDiagnosticsAndDoesNotClaimReadiness(t *testing.T) {
	f,before,s,w:=sample()
	after:=before
	after.Triggers=append([]reviewTrigger{},before.Triggers...)
	after.Triggers[0].TargetStep="unknown"
	p:=build(f,before,s,w,"procedure.json","review.json","set.json")
	q:=build(f,after,s,w,"procedure.json","next.json","set.json")
	base,next:=previewSnapshots()
	got,err:=previewReview(before,after,p,q,base,next)
	if err!=nil{t.Fatal(err)}
	if got.AcceptanceStatus!="blocked-unresolved-diagnostic" || got.ReadyForAcceptance || len(got.SourceWrites)!=0 {
		t.Fatalf("invalid review was promoted to ready: %+v",got)
	}
	found:=false
	for _,d:=range got.Diagnostics {if d.Code=="DFD_TRIGGER_TARGET_MISSING" {found=true}}
	if !found {t.Fatalf("missing source diagnostic for candidate: %+v",got.Diagnostics)}
}

func TestReviewPreviewDoesNotInferLogicalRenameOrOrderSemantics(t *testing.T) {
	f,before,s,w:=sample()
	after:=before
	after.Triggers=append([]reviewTrigger{},before.Triggers...)
	after.Triggers[0].ID="T-REKEY"
	after.Flows=append([]reviewFlow{},before.Flows...)
	after.Flows[0].Condition="explanatory note only"
	p:=build(f,before,s,w,"procedure.json","review.json","set.json")
	q:=build(f,after,s,w,"procedure.json","next.json","set.json")
	base,next:=previewSnapshots()
	got,err:=previewReview(before,after,p,q,base,next)
	if err!=nil{t.Fatal(err)}
	count:=map[string]int{}
	for _,c:=range got.Changes {count[c.Category+"/"+c.Action]++}
	if count["trigger/proposed-add"]!=1 || count["trigger/proposed-remove"]!=1 ||
		count["control-transition/proposed-remove"]!=1 || count["control-transition/proposed-add"]!=1 {
		t.Fatalf("implicit trigger logical rename or control flow identity was inferred: %+v",got.Changes)
	}
}

func TestReviewPreviewRejectsChangedProcedureIdentity(t *testing.T) {
	f,before,s,w:=sample()
	after:=before
	after.Procedure="another.procedure"
	p:=build(f,before,s,w,"procedure.json","review.json","set.json")
	q:=build(f,after,s,w,"procedure.json","next.json","set.json")
	base,next:=previewSnapshots()
	if _,err:=previewReview(before,after,p,q,base,next);err==nil {
		t.Fatal("cross-Procedure review was accepted as an in-place update")
	}
}
