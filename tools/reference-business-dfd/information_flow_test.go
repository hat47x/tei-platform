package main

import (
	"strings"
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
)

// Tests below address real information-flow (DFD) semantics rather than
// mistaking sequence/exception arrows for information transfers.
func TestBusinessDFDInformationFlowsAndResponsibilityRemainProposed(t *testing.T) {
	_, r, _, w := sample()
	r.BusinessRoles = []businessRole{{ID:"sales",Label:"Sales"}, {ID:"approval",Label:"Approver"}}
	r.Responsibilities = []stepResponsibility{{Step:"BP-04",Role:"approval"}, {Step:"BP-00",Role:"sales"}}
	r.ExternalParties = []externalParty{{ID:"customer",Label:"Customer"}}
	r.Stores = []informationStore{{ID:"orders",Label:"Order Data"}}
	r.InformationFlows = []informationFlow{
		{ID:"FLOW-01",From:informationEndpoint{Kind:"party",ID:"customer"},
			To:informationEndpoint{Kind:"step",ID:"BP-04"},Payload:"availability over telephone"},
		{ID:"FLOW-02",From:informationEndpoint{Kind:"step",ID:"BP-00"},
			To:informationEndpoint{Kind:"store",ID:"orders"},Payload:"draft update request"},
	}
	got, ds := buildInformationProjection(r,map[string]bool{"BP-00":true,"BP-04":true,"BP-09":true},w,"review.json")
	if len(ds)!=0 { t.Fatalf("unexpected diagnostics: %+v",ds) }
	if len(got.Roles)!=2 || len(got.Responsibilities)!=2 || len(got.ExternalParties)!=1 || len(got.Stores)!=1 || len(got.InformationFlows)!=2 {
		t.Fatalf("missing one-level DFD elements: %+v",got)
	}
	if got.Stores[0].DataStatus!="not-declared" { t.Fatal("unspecified semantic Store reference was inferred") }
	if got.Responsibilities[0].Authority!="proposal" || got.InformationFlows[0].Authority!="proposal" {
		t.Fatal("Business DFD proposal promoted to accepted authority")
	}
	if got.InformationFlows[0].From.Kind!="party" || got.InformationFlows[0].To.Kind!="step" {
		t.Fatal("typed information endpoints were lost")
	}
	var b strings.Builder
	appendInformationDOT(&b,got)
	for _,want:=range []string{"party:customer","store:orders","information / proposal:","business responsibility (proposal)"} {
		if !strings.Contains(b.String(),want) { t.Errorf("missing %q in DOT: %s",want,b.String()) }
	}
}

func TestInformationRequiresProcessAndExplicitPayload(t *testing.T) {
	_,r,_,w:=sample()
	r.ExternalParties=[]externalParty{{ID:"customer",Label:"Customer"}}
	r.Stores=[]informationStore{{ID:"orders",Label:"Orders"}}
	r.InformationFlows=[]informationFlow{{
		ID:"FLOW-01",From:informationEndpoint{Kind:"party",ID:"customer"},
		To:informationEndpoint{Kind:"store",ID:"orders"},Payload:"",
	}}
	_, ds:=buildInformationProjection(r,map[string]bool{"BP-00":true},w,"review.json")
	seen:=map[string]bool{}
	for _,d:=range ds {
		seen[d.Code]=true
		if d.Source.File!="review.json" || !strings.HasPrefix(d.Source.Pointer,"/informationFlows/0") {
			t.Errorf("untraceable information failure: %+v",d)
		}
	}
	for _,code:=range []string{"DFD_INFORMATION_PROCESS_REQUIRED","DFD_INFORMATION_PAYLOAD_MISSING"} {
		if !seen[code] { t.Errorf("missing %s: %+v",code,ds) }
	}
}

func TestInvalidInformationEndpointsAndRoleAreDiagnosed(t *testing.T) {
	_,r,_,w:=sample()
	r.BusinessRoles=[]businessRole{{ID:"sales",Label:"Sales"},{ID:"sales",Label:"Different"}}
	r.Responsibilities=[]stepResponsibility{{Step:"unknown",Role:"auth.admin"},{Step:"unknown",Role:"auth.admin"}}
	r.ExternalParties=[]externalParty{{ID:"customer",Label:"Customer"}}
	r.InformationFlows=[]informationFlow{
		{ID:"bad",From:informationEndpoint{Kind:"party",ID:"customer"},
			To:informationEndpoint{Kind:"step",ID:"unknown"},Payload:"message"},
		{ID:"bad",From:informationEndpoint{Kind:"admin",ID:"sales"},
			To:informationEndpoint{Kind:"step",ID:"BP-00"},Payload:"message"},
	}
	_,ds:=buildInformationProjection(r,map[string]bool{"BP-00":true},w,"review.json")
	seen:=map[string]bool{}
	for _,d:=range ds { seen[d.Code]=true }
	for _,code:=range []string{
		"DFD_BUSINESS_ROLE_ID_INVALID","DFD_RESPONSIBILITY_STEP_MISSING",
		"DFD_RESPONSIBILITY_ROLE_MISSING","DFD_RESPONSIBILITY_DUPLICATE",
		"DFD_INFORMATION_FLOW_ID_INVALID","DFD_INFORMATION_FROM_MISSING",
		"DFD_INFORMATION_TO_MISSING",
	} {
		if !seen[code] { t.Errorf("missing %s: %+v",code,ds) }
	}
}

func TestStoreSemanticIdentityMustResolveExactly(t *testing.T) {
	_,r,_,w:=sample()
	r.Stores=[]informationStore{{ID:"orders",Label:"Orders",SemanticData:"missing.data"}}
	info,ds:=buildInformationProjection(r,nil,w,"review.json")
	if info.Stores[0].DataStatus!="unresolved" { t.Fatal("unresolved Store semantic reference was promoted") }
	if len(ds)!=1 || ds[0].Code!="DFD_STORE_DATA_REFERENCE_NOT_EXACT" || ds[0].Source.Pointer!="/stores/0/semanticData" {
		t.Fatalf("source-aware missing semantic Data diagnostic expected, got %+v",ds)
	}
	// The isolated test exercises missing identity. The full integration test
	// uses the accepted sales.order and sales.order_line fixture identities.
}

func TestReviewHasNoInformationWithoutExplicitProposal(t *testing.T) {
	_,r,_,w:=sample()
	got,ds:=buildInformationProjection(r,map[string]bool{"BP-00":true},w,"review.json")
	if len(ds)!=0 || len(got.InformationFlows)!=0 || len(got.Responsibilities)!=0 {
		t.Fatalf("inferred DFD meaning from control-flow/Function: %+v / %+v",got,ds)
	}
}

// A small compile-time fixture check also guards the Semantic namespace type
// used by the exact-Data resolver without requiring a runtime Provider.
var _ = semantic.Namespace{}
var _ = ir.World{}
