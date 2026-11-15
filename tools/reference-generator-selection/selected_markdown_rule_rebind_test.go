package generatorselection

import (
    "bytes"
    "crypto/sha256"
    "encoding/json"
    "fmt"
    "math/big"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "testing"

    "tei/internal/core"
    "tei/internal/semantic"
    "tei/internal/semantic/validationeval"
    "tei/internal/sourceindex"
    "tei/internal/validation"
    "tei/internal/workspace"
)

// The Source-interface read is observable, but it is not physical Provider I/O.
type markdownRuleReadProbe struct {
    expected core.CanonicalID
    calls []core.CanonicalID
}
func (p *markdownRuleReadProbe) Number(attribute core.CanonicalID) (*big.Rat,error) {
    p.calls=append(p.calls,attribute)
    if attribute!=p.expected {return nil,fmt.Errorf("unexpected numeric read: %s",attribute)}
    return big.NewRat(1,1),nil
}
func (p *markdownRuleReadProbe) RelationKeys(relation,attribute core.CanonicalID) ([]string,error) {
    return nil,fmt.Errorf("unexpected relation read: %s/%s",relation,attribute)
}

// Both workspaces have the same two numeric attributes. Only the Rule
// reference changes; the accompanying contract is adjusted explicitly.
func selectedMarkdownFixture(t *testing.T, ref string) (root string, authored []byte) {
    t.Helper()
    root=t.TempDir()
    fixture:=filepath.Join(repoRoot(t),"internal","workspace","testdata","tei-md-rule")
    const rules="semantic/sales/order-line-rules.tei.md"
    for _,rel:=range []string{"tei.workspace.yaml","semantic/sales/order.tei",rules} {
        raw,err:=os.ReadFile(filepath.Join(fixture,filepath.FromSlash(rel)))
        if err!=nil {t.Fatal(err)}
        switch rel {
        case "semantic/sales/order.tei":
            const old="attribute quantity type std.decimal"
            if bytes.Count(raw,[]byte(old))!=1 {t.Fatal("baseline Data changed")}
            raw=bytes.Replace(raw,[]byte(old),[]byte(old+"\n    attribute threshold type std.decimal"),1)
        case rules:
            if ref=="threshold" {
                const old="sales.order_line.quantity is positive"
                if bytes.Count(raw,[]byte(old))!=1 {t.Fatal("baseline Rule changed")}
                raw=bytes.Replace(raw,[]byte(old),[]byte("sales.order_line.threshold is positive"),1)
            }
            authored=raw
        }
        dst:=filepath.Join(root,filepath.FromSlash(rel))
        if err:=os.MkdirAll(filepath.Dir(dst),0700);err!=nil{t.Fatal(err)}
        if err:=os.WriteFile(dst,raw,0600);err!=nil{t.Fatal(err)}
    }
    return root,authored
}
func selectedMarkdownContract(t *testing.T,root,ref string) string {
    t.Helper()
    raw,err:=os.ReadFile(validationTestPaths(t).contract)
    if err!=nil {t.Fatal(err)}
    var c Contract
    if err:=json.Unmarshal(raw,&c);err!=nil{t.Fatal(err)}
    if c.Validation==nil || len(c.Inputs)!=1 {t.Fatal("baseline Generator contract changed")}
    c.Inputs[0].Attribute="sales.order_line."+ref
    c.Validation.Input=ref
    encoded,err:=json.Marshal(c)
    if err!=nil{t.Fatal(err)}
    dst:=filepath.Join(root,"selected-generator-contract.json")
    if err:=os.WriteFile(dst,encoded,0600);err!=nil{t.Fatal(err)}
    return dst
}
func selectedMarkdownBinaryRun(t *testing.T,binary string,input map[string]any,wantState,wantCode string){
    t.Helper()
    b,err:=json.Marshal(input)
    if err!=nil{t.Fatal(err)}
    cmd:=exec.Command(binary)
    cmd.Stdin=bytes.NewReader(b)
    output,err:=cmd.CombinedOutput()
    if err!=nil{t.Fatalf("generated binary error on %s: %v; %s",b,err,output)}
    var got map[string]string
    if err:=json.Unmarshal(output,&got);err!=nil{t.Fatalf("generated output %s: %v",output,err)}
    if got["state"]!=wantState || got["code"]!=wantCode {
        t.Fatalf("input %s yielded state/code %q/%q, want %q/%q",b,got["state"],got["code"],wantState,wantCode)
    }
}

// This is a product selected-reader -> reference Generator -> separate Go
// process test, not product Builder/Provider or Stage2 self-build evidence.
func TestSelectedMarkdownRuleRebindReachesReferenceGenerator(t *testing.T){
    const id="sales.order_line.quantity_positive"
    const path="semantic/sales/order-line-rules.tei.md"
    generator:=filepath.Join(validationTestPaths(t).generators,generatorSources["go-positive"])
    var generatedSHA [2][32]byte
    for index,ref:=range []string{"quantity","threshold"}{
        t.Run(ref,func(t *testing.T){
            root,source:=selectedMarkdownFixture(t,ref)
            load,err:=workspace.Load(root)
            if err!=nil{t.Fatalf("product Workspace Load: %v",err)}
            if len(load.AuthoringSources)!=1 || load.AuthoringSources[0].Language.LanguageID!="tei.rule-authoring" ||
                load.AuthoringSources[0].Language.ContractVersion!="v0" || load.AuthoringSources[0].Definitions!=2 {
                t.Fatalf("selected built-in Markdown Reader not observed: %+v",load.AuthoringSources)
            }
            readByLoader:=false
            for _,managed:=range load.ManagedFiles{
                if managed.Path==path {readByLoader=managed.ReadByLoader}
            }
            if !readByLoader{t.Fatal("selected Markdown was not read by product Loader")}
            wantAttribute:=core.CanonicalID("sales.order_line."+ref)
            rule,ok:=findValidation(load.World,id)
            if !ok || rule.Kind!=semantic.ValidationPositive || rule.Attribute!=wantAttribute {
                t.Fatalf("selected canonical Rule mismatch: %+v (found=%t)",rule,ok)
            }
            for _,diagnostic:=range validation.Validate(load.World){
                if diagnostic.Subject!=nil && diagnostic.Subject.ID==id && diagnostic.Severity==validation.Error {
                    t.Fatalf("normal Validator rejected selected Rule: %+v",diagnostic)
                }
            }
            sourceIndex,err:=workspace.BuildSourceIndexFromLoad(root,load)
            if err!=nil{t.Fatal(err)}
            refs:=sourceIndex.ReferencesFor(sourceindex.Subject{Kind:"semantic-validation",ID:id})
            if len(refs)!=1 || refs[0].ArtifactPath!=path || refs[0].Range.Validate(source)!=nil ||
                string(source[refs[0].Range.Start.Offset:refs[0].Range.End.Offset])!=id {
                t.Fatalf("selected Rule Source byte span mismatch: %+v",refs)
            }
            probe:=&markdownRuleReadProbe{expected:wantAttribute}
            result,err:=validationeval.Evaluate(rule,probe)
            if err!=nil || result!=validationeval.ResultSatisfied || len(probe.calls)!=1 || probe.calls[0]!=wantAttribute{
                t.Fatalf("evaluator read wrong Attribute: result=%s error=%v reads=%v",result,err,probe.calls)
            }

            contract:=selectedMarkdownContract(t,root,ref)
            out:=filepath.Join(t.TempDir(),"candidate")
            _,err=GenerateFromTransformSource(TransformGeneratorOptions{
                ID:"go-positive",WorkspaceDir:root,ContractPath:contract,
                GeneratorSourcePath:generator,OutDir:out,
            })
            if err!=nil{t.Fatalf("reference Generator did not consume selected Rule: %v",err)}
            sourceGo,err:=os.ReadFile(filepath.Join(out,"main.go"))
            if err!=nil{t.Fatal(err)}
            if !bytes.Contains(sourceGo,[]byte("positive over "+string(wantAttribute))) {
                t.Fatalf("generated Go did not consume selected Attribute: %s",sourceGo)
            }
            generatedSHA[index]=sha256.Sum256(sourceGo)
            binary:=filepath.Join(t.TempDir(),"rule")
            cmd:=exec.Command("go","build","-trimpath","-o",binary,filepath.Join(out,"main.go"))
            if output,err:=cmd.CombinedOutput();err!=nil {
                t.Fatalf("generated source failed separate-process build: %v\n%s",err,output)
            }
            selectedMarkdownBinaryRun(t,binary,map[string]any{ref:"1"},"satisfied","")
            selectedMarkdownBinaryRun(t,binary,map[string]any{ref:"0"},"not-satisfied","")
            selectedMarkdownBinaryRun(t,binary,map[string]any{ref:"-1"},"not-satisfied","")
            selectedMarkdownBinaryRun(t,binary,map[string]any{ref:nil},"unevaluable","null")
            selectedMarkdownBinaryRun(t,binary,map[string]any{},"unevaluable","missing")
            other:="threshold"
            if ref=="threshold"{other="quantity"}
            selectedMarkdownBinaryRun(t,binary,map[string]any{other:"1"},"unevaluable","missing")
        })
    }
    if generatedSHA[0]==generatedSHA[1]{t.Fatal("one-factor Rule rebind did not change generated Go bytes")}
}

// A non-numeric change is rejected before the generator produces any output.
func TestSelectedMarkdownRuleRebindRejectsIncompatibleType(t *testing.T){
    root,_:=selectedMarkdownFixture(t,"threshold")
    path:=filepath.Join(root,"semantic","sales","order.tei")
    raw,err:=os.ReadFile(path)
    if err!=nil{t.Fatal(err)}
    const old="attribute threshold type std.decimal"
    if bytes.Count(raw,[]byte(old))!=1{t.Fatal("baseline type changed")}
    raw=bytes.Replace(raw,[]byte(old),[]byte("attribute threshold type std.text"),1)
    if err:=os.WriteFile(path,raw,0600);err!=nil{t.Fatal(err)}
    out:=filepath.Join(t.TempDir(),"candidate")
    _,err=GenerateFromTransformSource(TransformGeneratorOptions{
        ID:"go-positive",WorkspaceDir:root,
        ContractPath:selectedMarkdownContract(t,root,"threshold"),
        GeneratorSourcePath:filepath.Join(validationTestPaths(t).generators,generatorSources["go-positive"]),
        OutDir:out,
    })
    if err==nil || !strings.Contains(err.Error(),"resolves to std.text"){
        t.Fatalf("non-numeric rebind was not refused: %v",err)
    }
    if _,statErr:=os.Stat(out);!os.IsNotExist(statErr){
        t.Fatalf("generator wrote output after type mismatch: %v",statErr)
    }
}
