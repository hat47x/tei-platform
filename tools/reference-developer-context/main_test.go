package main

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "net/http/httptest"
    "os"
    "path/filepath"
    "reflect"
    "strings"
    "testing"
)

func fixture(t *testing.T) string {
    t.Helper()
    root:=t.TempDir()
    write:=func(path,body string){
        t.Helper()
        name:=filepath.Join(root,filepath.FromSlash(path))
        if err:=os.MkdirAll(filepath.Dir(name),0755);err!=nil {t.Fatal(err)}
        if err:=os.WriteFile(name,[]byte(body),0600);err!=nil {t.Fatal(err)}
    }
    write("internal/crossview/qualification.go","package crossview\nfunc Evaluate() {}\n")
    write("internal/crossview/qualification_test.go","package crossview\n// Evaluate tests\n")
    write("internal/crossview/revision_test.go","package crossview\n// qualification crossview review\n")
    write("docs/design.md","# Design\nReview internal/crossview/qualification.go\n")
    write("plan/adr/0001.md","# Irrelevant\n")
    return root
}

func paths(candidates []Candidate) []string {
    out:=make([]string,0,len(candidates))
    for _,c:=range candidates {out=append(out,c.Path)}
    return out
}

func TestDiscoverHasRealGroundedCandidates(t *testing.T) {
    r,err:=Discover(fixture(t),"internal/crossview/qualification.go")
    if err!=nil {t.Fatal(err)}
    want:=[]string{"internal/crossview/qualification_test.go","internal/crossview/revision_test.go","docs/design.md"}
    if !reflect.DeepEqual(paths(r.Candidates),want){t.Fatalf("paths: %v",paths(r.Candidates))}
    if !r.ReadOnly || r.Accepted || r.GrantsAuthority {t.Fatalf("unexpected authority: %+v",r)}
    if r.TargetSHA256=="" || r.Evaluation.State!="baseline" {t.Fatalf("missing source receipt: %+v",r)}
    for i,c:=range r.Candidates {
        if c.BaselineRank!=i+1 || !strings.HasPrefix(c.SHA256,"sha256:") || c.Basis=="" {
            t.Fatalf("ungrounded candidate: %+v",c)
        }
    }
}

func TestDiscoverRejectsUnsafePaths(t *testing.T) {
    root:=fixture(t)
    for _,path:=range []string{"../secrets.go","/etc/passwd","internal/../crossview/qualification.go","internal\\crossview\\qualification.go","internal/crossview/qualification_test.go","missing.go"} {
        if _,err:=Discover(root,path);err==nil {t.Fatalf("accepted unsafe path %q",path)}
    }
    outside:=t.TempDir()
    if err:=os.WriteFile(filepath.Join(outside,"secret.go"),[]byte("package secret"),0600);err!=nil {t.Fatal(err)}
    if err:=os.Symlink(filepath.Join(outside,"secret.go"),filepath.Join(root,"internal/crossview/link.go"));err!=nil {
        t.Skipf("symlink not permitted: %v",err)
    }
    if _,err:=Discover(root,"internal/crossview/link.go");err==nil {t.Fatal("followed out-of-root symlink")}
}

func TestDecisionModelOnlyReranks(t *testing.T) {
    report,err:=Discover(fixture(t),"internal/crossview/qualification.go")
    if err!=nil {t.Fatal(err)}
    original:=paths(report.Candidates)
    server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
        if r.URL.Path!="/v1/systemone" || r.Method!="POST" {t.Errorf("wrong endpoint: %s %s",r.Method,r.URL.Path)}
        var input struct {
            Model string `json:"model"`
            State map[string]any `json:"state"`
            Questions map[string]struct{Type string `json:"type"`} `json:"questions"`
        }
        if err:=json.NewDecoder(r.Body).Decode(&input);err!=nil {t.Error(err)}
        if input.Model!="local-decision-model" || len(input.Questions)!=3 || input.Questions["c0"].Type!="choice" {
            t.Errorf("typed question missing: %+v",input)
        }
        w.Header().Set("Content-Type","application/json")
        fmt.Fprint(w,`{"answers":{
            "c0":{"type":"choice","choice":"not_relevant","probabilities":{"relevant":0.1,"uncertain":0.2,"not_relevant":0.7}},
            "c1":{"type":"choice","choice":"relevant","probabilities":{"relevant":0.8,"uncertain":0.1,"not_relevant":0.1}},
            "c2":{"type":"choice","choice":"uncertain","probabilities":{"relevant":0.2,"uncertain":0.7,"not_relevant":0.1}}
        }}`)
    }))
    defer server.Close()
    Rerank(context.Background(),&report,server.URL+"/v1/systemone","local-decision-model")
    if report.Evaluation.State!="model-ranked" {t.Fatalf("model failed: %+v",report.Evaluation)}
    if report.Candidates[0].Path!=original[1] || report.Candidates[0].BaselineRank!=2 {
        t.Fatalf("model did not reorder only: %+v",report.Candidates)
    }
    seen:=map[string]bool{}
    for _,c:=range report.Candidates {seen[c.Path]=true}
    if len(seen)!=len(original) {t.Fatalf("dropped grounded candidate: %+v",report.Candidates)}
    for _,p:=range original {if !seen[p] {t.Errorf("candidate lost: %s",p)}}
}

func TestModelFailureKeepsBaseline(t *testing.T) {
    for _,mode:=range []string{"server-error","bad-json","remote"} {
        t.Run(mode,func(t *testing.T){
            report,err:=Discover(fixture(t),"internal/crossview/qualification.go")
            if err!=nil {t.Fatal(err)}
            before:=paths(report.Candidates)
            server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
                if mode=="server-error" {http.Error(w,"unavailable",503)} else {fmt.Fprint(w,`{"answers":{"c0":{"choice":"relevant"}}}`)}
            }))
            defer server.Close()
            endpoint:=server.URL+"/v1/systemone"
            if mode=="remote" {endpoint="https://api.example.com/v1/systemone"}
            Rerank(context.Background(),&report,endpoint,"local-model")
            if report.Evaluation.State!="model-failed" || report.Evaluation.Diagnostic=="" {
                t.Fatalf("failure hidden: %+v",report.Evaluation)
            }
            if !reflect.DeepEqual(before,paths(report.Candidates)) {
                t.Fatalf("model error mutated baseline: %+v",report.Candidates)
            }
        })
    }
}

func TestNoCandidateAndOversizedInput(t *testing.T) {
    root:=t.TempDir()
    if err:=os.WriteFile(filepath.Join(root,"lonely.go"),[]byte("package example"),0600);err!=nil {t.Fatal(err)}
    report,err:=Discover(root,"lonely.go")
    if err!=nil {t.Fatal(err)}
    Rerank(context.Background(),&report,"http://127.0.0.1:8123/v1/systemone","model")
    if report.Evaluation.State!="no-candidates" {t.Fatalf("unexpected state: %+v",report.Evaluation)}
    if err:=os.WriteFile(filepath.Join(root,"large.go"),make([]byte,maxFileBytes+1),0600);err!=nil {t.Fatal(err)}
    if _,err:=Discover(root,"large.go");err==nil {t.Fatal("oversized source allowed")}
}
