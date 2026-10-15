package studiohost

import (
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/operations"
)

type catalogProbe struct {
	calls int
	lastNamespace string
	value operations.ModelCatalog
	err error
}
func (p *catalogProbe) Catalog(namespace string) (operations.ModelCatalog,error) {
	p.calls++
	p.lastNamespace=namespace
	return p.value,p.err
}
type validationProbe struct {
	calls int
	value operations.WorkspaceValidation
	err error
}
func(p *validationProbe) Validate()(operations.WorkspaceValidation,error){
	p.calls++
	return p.value,p.err
}

func TestInjectedOperationsOnlyReceiveAllowlistedResolvedRoot(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	base,err:=NewReadOnly([]string{root})
	if err!=nil{t.Fatal(err)}
	trustedRoot,denied:=base.workspace("workspace-1")
	if denied!=nil{t.Fatal(denied)}
	cat := &catalogProbe{value: operations.ModelCatalog{Effect:"injected-only"}}
	val := &validationProbe{}
	factoryCalls:=0
	h,err:=NewReadOnlyWithFactory([]string{root},func(path string)(RootServices,error){
		factoryCalls++
		if path!=trustedRoot {t.Fatalf("untrusted factory root: %q",path)}
		return RootServices{Catalog:cat,Validation:val},nil
	})
	if err!=nil{t.Fatal(err)}
	for _,id:=range []string{"","workspace-2","../reference/workspace",trustedRoot," workspace-1"}{
		if got:=h.CatalogWorkspace(id,""); got.OK || got.Failure==nil || got.Failure.Code!="WORKSPACE_NOT_ALLOWED" {
			t.Fatalf("catalog unexpected permission for %q: %+v",id,got)
		}
		if got:=h.ValidateWorkspace(id);got.OK || got.Failure==nil || got.Failure.Code!="WORKSPACE_NOT_ALLOWED"{
			t.Fatalf("validate unexpected permission for %q: %+v",id,got)
		}
	}
	if factoryCalls!=0 {t.Fatal("factory ran before allowlist checked")}
	catalog:=h.CatalogWorkspace("workspace-1","sales")
	if !catalog.OK || catalog.Value==nil || catalog.Value.Effect!="injected-only" || cat.calls!=1 || cat.lastNamespace!="sales"{
		t.Fatalf("catalog bypassed injected service: %+v calls=%d",catalog,cat.calls)
	}
	validation:=h.ValidateWorkspace("workspace-1")
	if !validation.OK || validation.Value==nil || val.calls!=1 || factoryCalls!=2 {
		t.Fatalf("validate bypassed injected service: %+v calls=%d",validation,val.calls)
	}
	before:=h.ListWorkspaces()
	if !reflect.DeepEqual(before,base.ListWorkspaces()) {t.Fatal("list operation changed")}
	subject:=base.CatalogWorkspace("workspace-1","").Value.Entries[0].Subject
	got:=h.InspectSubject("workspace-1",subject.Kind,subject.ID)
	want:=base.InspectSubject("workspace-1",subject.Kind,subject.ID)
	if !reflect.DeepEqual(got,want) {t.Fatal("InspectSubject operation changed")}
	if factoryCalls!=2 {t.Fatal("inspect must not resolve generated services")}
}

func TestInjectedFailureEnvelopeDoesNotExposePrivateRoot(t *testing.T){
	root:=t.TempDir()
	private:=filepath.Join(root,"private-token")
	h,err:=NewReadOnlyWithFactory([]string{root},func(_ string)(RootServices,error){
		return RootServices{},errors.New("private error: "+private)
	})
	if err!=nil {t.Fatal(err)}
	for _,failure:=range []*Failure{
		h.CatalogWorkspace("workspace-1","").Failure,
		h.ValidateWorkspace("workspace-1").Failure,
	}{
		if failure==nil || failure.Code!="OPERATION_FAILED" || strings.Contains(failure.Message,private) {
			t.Fatalf("generated service details leaked: %+v",failure)
		}
	}
	h2,err:=NewReadOnlyWithFactory([]string{root},func(_ string)(RootServices,error){return RootServices{},nil})
	if err!=nil{t.Fatal(err)}
	if x:=h2.ValidateWorkspace("workspace-1"); x.OK || x.Failure==nil || x.Failure.Code!="OPERATION_FAILED"{
		t.Fatalf("incomplete generated service accepted: %+v",x)
	}
	if x:=h2.CatalogWorkspace("workspace-1",""); x.OK || x.Failure==nil || x.Failure.Code!="OPERATION_FAILED"{
		t.Fatalf("incomplete generated service accepted: %+v",x)
	}
}
