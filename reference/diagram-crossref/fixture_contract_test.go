package diagramcrossref

// This audit checks AI-drafted synthetic JSON fixtures only. It does NOT
// execute TEI's parser, semantic model, projector, CLI, Studio, or Provider.
// Passing does not qualify expected-v0.json as an independent oracle.
import (
 "encoding/json"
 "fmt"
 "os"
 "reflect"
 "sort"
 "strings"
 "testing"
)
type fixture struct {
 SourceRevision string `json:"sourceRevision"`
 Entities []struct{ID,Label string} `json:"entities"`
 Relations []struct{ID,From,To,Kind string} `json:"relations"`
 Functions []struct{
  ID string `json:"id"`
  DeclaredCrudComplete bool `json:"declaredCrudComplete"`
  Uses []struct{Data string `json:"data"`; Operations []string `json:"operations"`} `json:"uses"`
 } `json:"functions"`
 BusinessActions []struct{ID,Kind string; Function *string `json:"function"`} `json:"businessActions"`
 NegativeVariants []struct{ID string} `json:"negativeVariants"`
}
type expectation struct {
 SourceRevision string `json:"sourceRevision"`
 ExpectedGraph struct {
  EntityCount int `json:"entityCount"`
  RelationCount int `json:"relationCount"`
  ParallelEdgePair []string `json:"parallelEdgePair"`
  DuplicateDisplayLabels []struct{Label string; DistinctIDs []string `json:"distinctIds"`} `json:"duplicateDisplayLabels"`
 } `json:"expectedGraph"`
 ExpectedFunctionDeclaredCrudMatrix struct{
  Cells map[string]map[string]string `json:"cells"`
 } `json:"expectedFunctionDeclaredCrudMatrix"`
 KnownMutantRejections []struct{ID,Expected string} `json:"knownMutantRejections"`
}
func fixtureProblems(f fixture, want expectation) []string {
 var problems []string
 add:=func(s string){problems=append(problems,s)}
 if f.SourceRevision==""||f.SourceRevision!=want.SourceRevision {add("source revision mismatch")}
 if len(f.Entities)!=want.ExpectedGraph.EntityCount {add("entity count mismatch")}
 if len(f.Relations)!=want.ExpectedGraph.RelationCount {add("relation count mismatch")}
 entities:=make(map[string]string,len(f.Entities))
 for _,e:=range f.Entities {
  if e.ID=="" {add("empty entity identity")}
  if _,ok:=entities[e.ID];ok {add("duplicate entity identity: "+e.ID)}
  entities[e.ID]=e.Label
 }
 edges:=make(map[string][2]string,len(f.Relations))
 for _,edge:=range f.Relations {
  if edge.ID=="" {add("empty edge identity")}
  if _,ok:=edges[edge.ID];ok {add("duplicate edge identity: "+edge.ID)}
  edges[edge.ID]=[2]string{edge.From,edge.To}
  if _,ok:=entities[edge.From];!ok {add("unresolved source endpoint: "+edge.ID)}
  if _,ok:=entities[edge.To];!ok {add("unresolved target endpoint: "+edge.ID)}
 }
 pair:=want.ExpectedGraph.ParallelEdgePair
 if len(pair)!=2 {add("parallel-edge pair missing")} else {
  a,okA:=edges[pair[0]];b,okB:=edges[pair[1]]
  if !okA||!okB||a!=b||pair[0]==pair[1] {add("parallel edges conflated or lost")}
 }
 for _,c:=range want.ExpectedGraph.DuplicateDisplayLabels {
  if len(c.DistinctIDs)<2 {add("duplicate-label oracle not discriminating")}
  seen:=map[string]bool{}
  for _,id:=range c.DistinctIDs {
   if seen[id]||entities[id]!=c.Label {add("duplicate-label identity lost: "+id)}
   seen[id]=true
  }
 }
 functions:=map[string]bool{}
 for _,fn:=range f.Functions {
  if fn.ID==""||functions[fn.ID] {add("missing/duplicate function identity: "+fn.ID)}
  functions[fn.ID]=true
  row,ok:=want.ExpectedFunctionDeclaredCrudMatrix.Cells[fn.ID]
  if !ok {add("missing CRUD expected row: "+fn.ID);continue}
  declared:=map[string]string{}
  for _,u:=range fn.Uses {
   if _,ok:=entities[u.Data];!ok {add("unknown Function Data: "+u.Data)}
   if _,ok:=declared[u.Data];ok {add("duplicate Function Data entry: "+fn.ID+"/"+u.Data)}
   operations:=append([]string(nil),u.Operations...)
   sort.Strings(operations)
   for _,op:=range operations {
    if op!="C"&&op!="R"&&op!="U"&&op!="D" {add("unknown CRUD marker: "+op)}
   }
   declared[u.Data]=strings.Join(operations,"")
  }
  for id:=range entities {
   expected:="unknown-not-enumerated"
   if fn.DeclaredCrudComplete {
    expected="none"
    if op,ok:=declared[id];ok {expected=op}
   }
   if row[id]!=expected {add(fmt.Sprintf("CRUD cell %s/%s expected %q got %q",fn.ID,id,expected,row[id]))}
  }
  if len(row)!=len(entities) {add("CRUD matrix extra/missing columns: "+fn.ID)}
 }
 if len(functions)!=len(want.ExpectedFunctionDeclaredCrudMatrix.Cells) {add("CRUD matrix row count mismatch")}
 actionIDs:=map[string]bool{}
 for _,a:=range f.BusinessActions {
  if actionIDs[a.ID] {add("duplicate Business Action identity: "+a.ID)}
  actionIDs[a.ID]=true
  if a.Kind=="manual"&&a.Function!=nil {add("manual Action forced onto Function")}
  if a.Function!=nil&&!functions[*a.Function] {add("Business Action unknown Function")}
 }
 mutants:=map[string]bool{}
 for _,m:=range f.NegativeVariants {
  if mutants[m.ID] {add("duplicate known mutant ID: "+m.ID)}
  mutants[m.ID]=true
 }
 expectedMutants:=map[string]bool{}
 for _,m:=range want.KnownMutantRejections {
  if expectedMutants[m.ID]||m.Expected!="FAIL" {add("invalid expected mutant: "+m.ID)}
  expectedMutants[m.ID]=true
 }
 if !reflect.DeepEqual(mutants,expectedMutants) {add("mutant IDs and expected IDs differ")}
 return problems
}
func readFixtures(t *testing.T)(fixture,expectation){
 t.Helper()
 var f fixture;var e expectation
 for _,v:=range []struct{path string;dst any}{
  {"pressure-input-v0.json",&f},{"expected-v0.json",&e},
 }{
  b,err:=os.ReadFile(v.path);if err!=nil {t.Fatalf("read %s: %v",v.path,err)}
  if err:=json.Unmarshal(b,v.dst);err!=nil {t.Fatalf("decode %s: %v",v.path,err)}
 }
 return f,e
}
func TestFixtureInternalConsistencyNotProductConformance(t *testing.T){
 f,e:=readFixtures(t)
 if p:=fixtureProblems(f,e);len(p)!=0 {t.Fatalf("fixture audit failed: %s",strings.Join(p,"; "))}
}
func TestFixtureAuditRejectsFourKnownMutations(t *testing.T){
 f,e:=readFixtures(t)
 if p:=fixtureProblems(f,e);len(p)!=0 {t.Fatalf("baseline invalid: %v",p)}
 cases:=[]struct{name string;mutate func(*fixture)}{
  {"drop-parallel-edge",func(x *fixture){x.Relations=x.Relations[:len(x.Relations)-1]}},
  {"collapse-identities",func(x *fixture){x.Entities[len(x.Entities)-1].ID=x.Entities[len(x.Entities)-2].ID}},
  {"mark-incomplete-CRUD-as-complete",func(x *fixture){for i:=range x.Functions {if !x.Functions[i].DeclaredCrudComplete {x.Functions[i].DeclaredCrudComplete=true;return}}}},
  {"break-referenced-edge-target",func(x *fixture){x.Relations[0].To="no.such.subject"}},
 }
 for _,tc:=range cases {t.Run(tc.name,func(t *testing.T){
  b,err:=json.Marshal(f);if err!=nil {t.Fatal(err)}
  var x fixture;if err:=json.Unmarshal(b,&x);err!=nil {t.Fatal(err)}
  tc.mutate(&x)
  if p:=fixtureProblems(x,e);len(p)==0 {t.Fatal("synthetic mutant escaped fixture audit")}
 })}
}
