package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/ir"
)

func sample() (fixture, review, procedureSet, ir.World) {
	f:=fixture{
		SchemaVersion:"business-procedure-reference-v0",
		ArtifactRole:"reference-authoring-fixture",
		Procedure:procedure{
			ID:"test.order-handling", IdentityStatus:"reference-local-stable",
			Actor:candidate{Status:"candidate",Label:"sales operator"},
			Trigger:candidate{Status:"candidate",Text:"need to review an order"},
		},
		Application:"order_entry",
		Steps:[]step{
			{ID:"BP-00",BusinessAction:"open order",Realization:"application-use",ApplicationPatterns:[]string{"order_entry.order_record"}},
			{ID:"BP-04",BusinessAction:"call customer by phone",Realization:"manual-unresolved"},
			{ID:"BP-09",BusinessAction:"ask external warehouse",Realization:"manual-external-unresolved"},
		},
	}
	r:=review{SchemaVersion:"business-dfd-review-v0",Authority:"proposal",Procedure:f.Procedure.ID,
		Triggers:[]reviewTrigger{
			{ID:"T-H",Kind:"human",TargetStep:"BP-00",Label:"start"},
			{ID:"T-E",Kind:"event",TargetStep:"BP-00",Label:"notification"},
			{ID:"T-S",Kind:"schedule",TargetStep:"BP-00",Label:"review time"},
		},
		Flows:[]reviewFlow{
			{From:"BP-00",To:"BP-04",Kind:"sequence"},
			{From:"BP-04",To:"BP-09",Kind:"branch",Condition:"warehouse is required"},
			{From:"BP-04",To:"BP-00",Kind:"exception",Condition:"customer rejects"},
		},
	}
	s:=procedureSet{Handoffs:[]handoff{
		{From:endpoint{Procedure:f.Procedure.ID,Step:"BP-09"},To:endpoint{Procedure:"warehouse.receive",Step:"WH-00"},Condition:"accepted"},
	}}
	w:=ir.World{Applications:[]application.Application{{ID:"order_entry",Patterns:[]application.Pattern{
		{ID:"order_entry.order_record",Kind:application.PatternRecord},
		{ID:"order_entry.submit_order",Kind:application.PatternCommand},
	}}}}
	return f,r,s,w
}

func codes(p projection) map[string]bool {
	out:=map[string]bool{}
	for _,d:=range p.Diagnostics { out[d.Code]=true }
	return out
}

func TestManualExternalAndThreeTriggerKindsAreNotFunctionsOrAuthority(t *testing.T) {
	f,r,s,w:=sample()
	before:=f
	got:=build(f,r,s,w,"procedure.json","review.json","set.json")
	if len(got.Diagnostics)!=0 { t.Fatalf("unexpected diagnostics: %+v",got.Diagnostics) }
	if !reflect.DeepEqual(f,before) { t.Fatal("view builder mutated Procedure authoring input") }
	if !got.Steps[1].NoApplicationFunction || !got.Steps[2].NoApplicationFunction {
		t.Fatal("manual/external Step must remain even without Application Pattern")
	}
	if got.Steps[0].NoApplicationFunction || got.Steps[0].FunctionRefs[0].Status!="resolved" {
		t.Fatal("existing Application Pattern must resolve exactly")
	}
	if got.RoleAuthorizationBinding!="not-inferred" || got.ActorCandidate.Label!="sales operator" {
		t.Fatal("Business Actor was treated as an authorization identity")
	}
	if len(got.Triggers)!=3 { t.Fatal("expected human/event/schedule candidates") }
	for i,kind:=range []string{"human","event","schedule"} {
		if got.Triggers[i].Kind!=kind || got.Triggers[i].RuntimeBinding!="unbound" || got.Triggers[i].Authority!="proposal" {
			t.Fatalf("trigger %d was promoted to runtime authority: %+v",i,got.Triggers[i])
		}
	}
	if got.Flows[2].Kind!="exception" || got.Flows[2].Authority!="proposal" { t.Fatal("exception lost proposal authority") }
	if len(got.Handoffs)!=1 || got.Handoffs[0].From.Step!="BP-09" { t.Fatal("existing cross-procedure Handoff lost") }
	if got.Steps[1].Source.File!="procedure.json" || got.Steps[1].Source.Pointer!="/steps/1" {
		t.Fatalf("Source locator lost: %+v",got.Steps[1].Source)
	}
	if len(got.UnreferencedPatterns)!=1 || got.UnreferencedPatterns[0]!="order_entry.submit_order" {
		t.Fatal("procedure-scope unreferenced Pattern inventory missing")
	}
	d:=dot(got)
	if !strings.Contains(d,"style=dashed") || !strings.Contains(d,"shape=note") || !strings.Contains(d,"style=dotted") {
		t.Fatalf("DOT lost proposal/manual/Function distinction: %s",d)
	}
}

func TestMissingFunctionIsSourceAware(t *testing.T) {
	f,r,s,w:=sample()
	f.Steps[0].ApplicationPatterns=[]string{"order_entry.missing"}
	got:=build(f,r,s,w,"procedure.json","review.json","set.json")
	if !codes(got)["DFD_FUNCTION_REFERENCE_NOT_EXACT"] { t.Fatal("missing function must be a diagnostic") }
	if got.Steps[0].FunctionRefs[0].Status!="unresolved" { t.Fatal("unresolved must not be shown as resolved") }
	d:=got.Diagnostics[0]
	if d.Source.File!="procedure.json" || d.Source.Pointer!="/steps/0/applicationPatterns/0" {
		t.Fatalf("source-aware failure missing: %+v",d)
	}
}

func TestBadTriggerFlowAndHandoffFailAsDiagnostics(t *testing.T) {
	f,r,s,w:=sample()
	r.Triggers[0].Kind="authorization-role"
	r.Triggers[1].TargetStep="missing"
	r.Flows[1].Condition=""
	r.Flows[2].To="missing"
	s.Handoffs[0].From.Step="missing"
	got:=build(f,r,s,w,"procedure.json","review.json","set.json")
	for _,want:=range []string{"DFD_TRIGGER_KIND_INVALID","DFD_TRIGGER_TARGET_MISSING","DFD_CONDITION_MISSING","DFD_FLOW_ENDPOINT_MISSING","DFD_HANDOFF_FROM_MISSING"} {
		if !codes(got)[want] { t.Errorf("missing diagnostic %s: %+v",want,got.Diagnostics) }
	}
}

func TestProposalRoutesAreNotInferredFromStepOrder(t *testing.T) {
	f,r,s,w:=sample()
	r.Flows=nil
	got:=build(f,r,s,w,"procedure.json","review.json","set.json")
	if len(got.Flows)!=0 { t.Fatal("source array order silently became accepted control flow") }
	if len(got.Steps)!=3 { t.Fatal("Steps disappeared without proposed routes") }
}

func TestStrictInputRejectsUnknownFieldsAndTrailingValues(t *testing.T) {
	path:=filepath.Join(t.TempDir(),"data.json")
	for _,content:=range []string{
		`{"schemaVersion":"business-dfd-review-v0","authority":"proposal","procedure":"x","extra":true}`,
		`{"schemaVersion":"business-dfd-review-v0","authority":"proposal","procedure":"x"}{}`,
	} {
		if err:=os.WriteFile(path,[]byte(content),0600);err!=nil{t.Fatal(err)}
		var r review
		if err:=decodeFile(path,&r);err==nil{t.Fatalf("invalid input accepted: %s",content)}
	}
}

func TestReferenceProjectionIntegration(t *testing.T) {
	p,err:=run(
		"../../reference/business-system-design/order-entry-business-procedure.json",
		"../../reference/business-system-design/order-entry-dfd-review-proposal.json",
		"../../reference/business-system-design/order-entry-procedure-reference-set.json",
		"../../reference/workspace",
	)
	if err!=nil { t.Fatal(err) }
	if len(p.Diagnostics)!=0 { t.Fatalf("reference projection returned diagnostics: %+v",p.Diagnostics) }
	if p.ProcedureID!="order-entry.order-handling" || len(p.Steps)!=10 || len(p.Triggers)!=3 || len(p.Handoffs)!=1 {
		t.Fatalf("unexpected reference projection coverage: steps=%d triggers=%d handoffs=%d",len(p.Steps),len(p.Triggers),len(p.Handoffs))
	}
	if !p.Steps[4].NoApplicationFunction || !p.Steps[9].NoApplicationFunction {
		t.Fatal("baseline manual Step lost")
	}
	if len(p.Information.Roles)!=2 || len(p.Information.ExternalParties)!=2 ||
		len(p.Information.Stores)!=2 || len(p.Information.Responsibilities)!=3 ||
		len(p.Information.InformationFlows)!=4 {
		t.Fatalf("DFD proposal missing typed information semantics: %+v",p.Information)
	}
	for _, store:=range p.Information.Stores {
		if store.DataStatus!="resolved" {
			t.Fatalf("proposal Store does not resolve an existing semantic Data identity: %+v",store)
		}
	}
	if !strings.Contains(dot(p),"information / proposal:") {
		t.Fatal("DOT lacks explicit proposed information flow arrows")
	}
}
