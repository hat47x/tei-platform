package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReferenceLocalNavigationDoesNotInventAcceptedIdentity(t *testing.T) {
	f,r,s,w:=sample()
	p:=build(f,r,s,w,"procedure.json","review.json","set.json")
	selected,err:=navigate(p,"step:BP-04")
	if err!=nil { t.Fatal(err) }
	if selected.Status!="reference-only" || selected.Editable ||
		selected.Subject.Kind!="step" || selected.Subject.ID!="BP-04" ||
		selected.Subject.Scope!="reference-procedure:test.order-handling" ||
		selected.Subject.Authority!="reference-only" {
		t.Fatalf("reference-local Step was promoted to accepted Subject: %+v",selected)
	}
	if selected.Source==nil || selected.Source.Pointer!="/steps/1" ||
		selected.AcceptanceTargetStatus!="unresolved-use-shared-acceptance" ||
		selected.WorldRevisionStatus!="unverified-no-model-revision-binding" {
		t.Fatalf("Source pointer or freshness promoted to Source Acceptance: %+v",selected)
	}
	if len(selected.Links)!=3 {
		t.Fatalf("expected only three explicitly proposed control links, got %+v",selected.Links)
	}
	for _,link:=range selected.Links {
		if link.Subject.Scope!="reference-procedure:test.order-handling" || link.Subject.Authority!="reference-only" ||
			link.Status!="reference-only-no-accepted-source" {
			t.Fatalf("invented accepted Step relation: %+v",link)
		}
	}
}

func TestNavigationFromBusinessStepToExactFunctionData(t *testing.T) {
	f,r,s,w:=sample()
	p:=build(f,r,s,w,"procedure.json","review.json","set.json")
	n,err:=navigate(p,"step:BP-00")
	if err!=nil {t.Fatal(err)}
	foundPattern, foundData:=false,false
	for _,link:=range n.Links {
		if link.Subject.Kind=="application-pattern" && link.Subject.ID=="order_entry.order_record" {
			foundPattern=true
			if link.Status!="resolved-id-not-accepted" || link.Subject.Scope!="unverified-current-world" {
				t.Fatalf("Function identity was promoted: %+v",link)
			}
		}
		if link.Subject.Kind=="semantic-data" {foundData=true}
	}
	if !foundPattern || foundData {
		t.Fatalf("only exact declared references are allowed: %+v",n.Links)
	}
	pattern,err:=navigate(p,"pattern:order_entry.order_record")
	if err!=nil {t.Fatal(err)}
	if pattern.Source!=nil || pattern.Editable || pattern.Subject.Kind!="application-pattern" {
		t.Fatalf("source Index or Source edit location inferred from Pattern ID: %+v",pattern)
	}
	if len(pattern.Links)!=1 || pattern.Links[0].Subject.ID!="BP-00" {
		t.Fatalf("reverse Pattern->Step ref missing: %+v",pattern.Links)
	}
}

func TestNavigationOfProposedBusinessRoleAndDataStore(t *testing.T) {
	f,r,s,w:=sample()
	r.BusinessRoles=[]businessRole{{ID:"sales",Label:"Sales"}}
	r.Responsibilities=[]stepResponsibility{{Step:"BP-04",Role:"sales"}}
	r.Stores=[]informationStore{{ID:"order",Label:"Orders",SemanticData:"missing.semantic"}}
	p:=build(f,r,s,w,"procedure.json","review.json","set.json")
	// An invalid semantic Data reference invalidates the entire projection;
	// navigation must not present a usable partial/accepted Source locator.
	n,err:=navigate(p,"business-role:sales")
	if err!=nil {t.Fatal(err)}
	if n.Status!="unresolved" || n.Source!=nil || len(n.Links)!=0 {
		t.Fatalf("diagnostic-bearing projection was navigable as if valid: %+v",n)
	}
	r.Stores[0].SemanticData=""
	p=build(f,r,s,w,"procedure.json","review.json","set.json")
	role,err:=navigate(p,"business-role:sales")
	if err!=nil {t.Fatal(err)}
	if len(role.Links)!=1 || role.Links[0].Subject.ID!="BP-04" || role.Source.Pointer!="/businessRoles/0" {
		t.Fatalf("proposed business role reference lost: %+v",role)
	}
	store,err:=navigate(p,"store:order")
	if err!=nil {t.Fatal(err)}
	if store.Source==nil || store.Source.Pointer!="/stores/0" || len(store.Links)!=0 {
		t.Fatalf("Store without semantic Data should not acquire an inferred binding: %+v",store)
	}
}

func TestUnresolvedOrNonexistentElementFailsClosed(t *testing.T) {
	f,r,s,w:=sample()
	p:=build(f,r,s,w,"procedure.json","review.json","set.json")
	for _,q:=range []string{"step:BP-MISSING","pattern:order_entry.missing","trigger:UNKNOWN","party:customer"} {
		n,err:=navigate(p,q)
		if err!=nil {t.Fatal(err)}
		if n.Status!="unresolved" || n.Source!=nil || n.Editable || len(n.Links)!=0 {
			t.Fatalf("invalid selector %q was not unresolved: %+v",q,n)
		}
	}
	for _,q:=range []string{"step","step:","not-supported:abc","step: BP-00","step:BP-00 "} {
		if _,err:=navigate(p,q);err==nil && (q=="step" || q=="step:" || strings.HasPrefix(q,"not-supported:")) {
			t.Errorf("invalid selector %q accepted",q)
		}
	}
}

func TestReverseDataNavigationDoesNotClaimAcceptedBusinessDependency(t *testing.T) {
	const root="../../reference/business-system-design/dfd-order-approval/"
	p,err:=run(root+"order-approval-procedure.json",root+"review-proposal.json",
		root+"procedure-set.json","../../reference/workspace")
	if err!=nil{t.Fatal(err)}
	n,err:=navigate(p,"semantic-data:sales.order")
	if err!=nil{t.Fatal(err)}
	if n.Subject.Scope!="unverified-current-world" || n.Subject.Authority!="identity-match-only" ||
		n.Editable || n.Source!=nil || n.WorldRevisionStatus!="unverified-no-model-revision-binding" {
		t.Fatalf("semantic Data navigation got fake Source authority: %+v",n)
	}
	if n.Status!="reference-only" {t.Fatalf("Data reverse link missing: %+v",n)}
	seenStep,seenStore:=false,false
	for _,link:=range n.Links {
		if link.Subject.Kind=="step" && link.Subject.ID=="APP-00" {seenStep=true}
		if link.Subject.Kind=="store" && link.Subject.ID=="orders" {seenStore=true}
		if link.Subject.Authority!="reference-only" {t.Fatalf("reverse Data navigation promoted authority: %+v",link)}
	}
	if !seenStep || !seenStore {t.Fatalf("expected proposed Step and Store cross-references: %+v",n.Links)}
}

func TestReverseDirectValidationNavigationRemainsReferenceLocal(t *testing.T) {
	p,err:=run(
		"../../reference/business-system-design/order-entry-business-procedure.json",
		"../../reference/business-system-design/order-entry-dfd-review-proposal.json",
		"../../reference/business-system-design/order-entry-procedure-reference-set.json",
		"../../reference/workspace")
	if err!=nil{t.Fatal(err)}
	n,err:=navigate(p,"semantic-validation:sales.order_line.quantity_positive")
	if err!=nil{t.Fatal(err)}
	found:=false
	for _,link:=range n.Links {
		if link.Subject.ID=="BP-05" && link.Relation=="explicitly-referenced-by-step" {
			found=true
		}
	}
	if !found || n.Status!="reference-only" || n.Editable || n.Source!=nil {
		t.Fatalf("Validation owner Step cannot be navigated as a reference-only link: %+v",n)
	}
}

func TestNavigationSourceFingerprintIsBoundToExactFixtureBytes(t *testing.T) {
	f,r,s,w:=sample()
	p:=build(f,r,s,w,"procedure.json","review.json","set.json")
	path:=filepath.Join(t.TempDir(),"business.json")
	if err:=os.WriteFile(path,[]byte("alpha"),0600);err!=nil{t.Fatal(err)}
	a,err:=navigate(p,"step:BP-04",path)
	if err!=nil{t.Fatal(err)}
	if len(a.SourceSnapshots)!=1 || !strings.HasPrefix(a.SourceSnapshots[0].SHA256,"sha256:") {
		t.Fatalf("source fingerprint missing: %+v",a.SourceSnapshots)
	}
	if err:=os.WriteFile(path,[]byte("beta"),0600);err!=nil{t.Fatal(err)}
	b,err:=navigate(p,"step:BP-04",path)
	if err!=nil{t.Fatal(err)}
	if a.SourceSnapshots[0].SHA256==b.SourceSnapshots[0].SHA256 ||
		b.WorldRevisionStatus!="unverified-no-model-revision-binding" {
		t.Fatal("changed fixture bytes or unverified World revision was silently accepted")
	}
}

func TestDemoNavigationKeepsHandoffDistinctFromExecutableFlow(t *testing.T) {
	const root="../../reference/business-system-design/dfd-order-approval/"
	p,err:=run(root+"order-approval-procedure.json",root+"review-proposal.json",
		root+"procedure-set.json","../../reference/workspace")
	if err!=nil{t.Fatal(err)}
	for _,d:=range p.Diagnostics {if d.Severity=="error" {t.Fatalf("invalid demo: %+v",d)}}
	info,err:=navigate(p,"step:APP-50")
	if err!=nil{t.Fatal(err)}
	var handoff *navigationLink
	for i,link:=range info.Links {
		if link.Relation=="conditional-handoff-to" {handoff=&info.Links[i]}
	}
	if handoff==nil || handoff.Subject.Scope!="reference-procedure:order-entry.warehouse-preparation" ||
		handoff.Subject.ID!="WH-00" || handoff.Status!="reference-set-member-only-remote-step-unverified" {
		t.Fatalf("cross-Procedure Handoff incorrectly accepted/executed: %+v",info.Links)
	}
}
