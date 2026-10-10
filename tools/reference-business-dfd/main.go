package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"tei/internal/application"
	"tei/internal/ir"
	"tei/internal/workspace"
)

// This command is a read-only Business DFD projection adapter. It deliberately
// does not declare a new Canonical Business Procedure or runtime Workflow.
type location struct {
	File string `json:"file"`
	Pointer string `json:"pointer"`
}
type candidate struct {
	Status string `json:"status"`
	Text string `json:"text,omitempty"`
	Label string `json:"label,omitempty"`
}
type procedure struct {
	ID string `json:"id"`
	IdentityStatus string `json:"identityStatus"`
	Purpose string `json:"purpose"`
	Trigger candidate `json:"trigger"`
	Actor candidate `json:"actor"`
}
type step struct {
	ID string `json:"id"`
	BusinessAction string `json:"businessAction"`
	Realization string `json:"realization"`
	ApplicationPatterns []string `json:"applicationPatterns,omitempty"`
	Data []string `json:"data,omitempty"`
	Relations []string `json:"relations,omitempty"`
	Validations []string `json:"validations,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
}
type fixture struct {
	SchemaVersion string `json:"schemaVersion"`
	ArtifactRole string `json:"artifactRole"`
	Canonical bool `json:"canonical"`
	Application string `json:"application"`
	Procedure procedure `json:"procedure"`
	Steps []step `json:"steps"`
}
type reviewTrigger struct {
	ID string `json:"id"`
	Kind string `json:"kind"`
	TargetStep string `json:"targetStep"`
	Label string `json:"label"`
}
type reviewFlow struct {
	From string `json:"from"`
	To string `json:"to"`
	Kind string `json:"kind"`
	Condition string `json:"condition,omitempty"`
}
type review struct {
	SchemaVersion string `json:"schemaVersion"`
	Authority string `json:"authority"`
	Procedure string `json:"procedure"`
	Triggers []reviewTrigger `json:"triggers"`
	Flows []reviewFlow `json:"flows"`
	BusinessRoles []businessRole `json:"businessRoles,omitempty"`
	ExternalParties []externalParty `json:"externalParties,omitempty"`
	Stores []informationStore `json:"stores,omitempty"`
	Responsibilities []stepResponsibility `json:"responsibilities,omitempty"`
	InformationFlows []informationFlow `json:"informationFlows,omitempty"`
}
type endpoint struct {
	Procedure string `json:"procedure"`
	Step string `json:"step"`
}
type handoff struct {
	From endpoint `json:"from"`
	To endpoint `json:"to"`
	Condition string `json:"condition"`
}
type procedureSet struct {
	SchemaVersion string `json:"schemaVersion"`
	ArtifactRole string `json:"artifactRole"`
	Procedures []struct {
		ID string `json:"id"`
		Artifact string `json:"artifact"`
	} `json:"procedures"`
	Handoffs []handoff `json:"handoffs"`
}

type diagnostic struct {
	Code string `json:"code"`
	Severity string `json:"severity"`
	Source location `json:"source"`
	Message string `json:"message"`
}
type functionRef struct {
	ID string `json:"id"`
	Status string `json:"status"`
	Data string `json:"data,omitempty"`
	Relation string `json:"relation,omitempty"`
	Capability string `json:"capability,omitempty"`
}
type stepNode struct {
	ID string `json:"id"`
	Label string `json:"label"`
	Realization string `json:"realization"`
	NoApplicationFunction bool `json:"noApplicationFunction"`
	FunctionRefs []functionRef `json:"functionRefs"`
	Source location `json:"source"`
}
type triggerNode struct {
	ID string `json:"id"`
	Kind string `json:"kind"`
	Label string `json:"label"`
	TargetStep string `json:"targetStep"`
	Authority string `json:"authority"`
	RuntimeBinding string `json:"runtimeBinding"`
	Source location `json:"source"`
}
type flowEdge struct {
	From string `json:"from"`
	To string `json:"to"`
	Kind string `json:"kind"`
	Condition string `json:"condition,omitempty"`
	Authority string `json:"authority"`
	Source location `json:"source"`
}
type handoffEdge struct {
	From endpoint `json:"from"`
	To endpoint `json:"to"`
	Condition string `json:"condition"`
	Authority string `json:"authority"`
	Source location `json:"source"`
}
type projection struct {
	SchemaVersion string `json:"schemaVersion"`
	Authority string `json:"authority"`
	CanonicalProcedureUsed bool `json:"canonicalProcedureUsed"`
	ProcedureID string `json:"procedureId"`
	ActorCandidate candidate `json:"actorCandidate"`
	TriggerCandidate candidate `json:"triggerCandidate"`
	RoleAuthorizationBinding string `json:"roleAuthorizationBinding"`
	Steps []stepNode `json:"steps"`
	Triggers []triggerNode `json:"triggers"`
	Flows []flowEdge `json:"flows"`
	Handoffs []handoffEdge `json:"handoffs"`
	Information informationProjection `json:"information"`
	UnreferencedPatterns []string `json:"unreferencedPatterns"`
	Diagnostics []diagnostic `json:"diagnostics"`
}

func decodeFile(path string, out any) error {
	file, err := os.Open(path)
	if err != nil { return err }
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil { return err }
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil { return fmt.Errorf("trailing JSON value in %s", path) }
		return err
	}
	return nil
}

func run(procedureFile, reviewFile, setFile, workspaceDir string) (projection, error) {
	var f fixture
	if err := decodeFile(procedureFile, &f); err != nil { return projection{}, fmt.Errorf("procedure: %w", err) }
	if f.SchemaVersion != "business-procedure-reference-v0" || f.ArtifactRole != "reference-authoring-fixture" || f.Canonical {
		return projection{}, fmt.Errorf("unsupported procedure source/authority; canonical=%t", f.Canonical)
	}
	var r review
	if err := decodeFile(reviewFile, &r); err != nil { return projection{}, fmt.Errorf("review: %w", err) }
	if r.SchemaVersion != "business-dfd-review-v0" || r.Authority != "proposal" {
		return projection{}, fmt.Errorf("DFD review must be an explicit proposal")
	}
	var set procedureSet
	if err := decodeFile(setFile, &set); err != nil { return projection{}, fmt.Errorf("handoff set: %w", err) }
	if set.SchemaVersion != "business-procedure-reference-set-v0" || set.ArtifactRole != "reference-authoring-set-fixture" {
		return projection{}, fmt.Errorf("unsupported handoff set")
	}
	loaded, err := workspace.Load(workspaceDir)
	if err != nil { return projection{}, fmt.Errorf("workspace: %w", err) }
	return build(f, r, set, loaded.World, procedureFile, reviewFile, setFile), nil
}

func build(f fixture, r review, set procedureSet, world ir.World, procedureFile, reviewFile, setFile string) projection {
	p := projection{
		SchemaVersion: "business-dfd-projection-v0",
		Authority: "derived-read-only",
		ProcedureID: f.Procedure.ID,
		ActorCandidate: f.Procedure.Actor,
		TriggerCandidate: f.Procedure.Trigger,
		RoleAuthorizationBinding: "not-inferred",
		Steps: []stepNode{}, Triggers: []triggerNode{}, Flows: []flowEdge{},
		Handoffs: []handoffEdge{}, Diagnostics: []diagnostic{},
		UnreferencedPatterns: []string{},
	}
	add := func(code, severity, file, pointer, msg string) {
		p.Diagnostics = append(p.Diagnostics, diagnostic{code,severity,location{file,pointer},msg})
	}
	if f.Procedure.ID == "" { add("DFD_PROCEDURE_ID_MISSING","error",procedureFile,"/procedure/id","procedure id is required") }
	if r.Procedure != f.Procedure.ID {
		add("DFD_PROCEDURE_MISMATCH","error",reviewFile,"/procedure","review does not match Procedure identity")
	}
	patterns := map[string][]application.Pattern{}
	for _, a := range world.Applications {
		for _, pattern := range a.Patterns { patterns[string(pattern.ID)] = append(patterns[string(pattern.ID)],pattern) }
	}
	ids := map[string]bool{}
	used := map[string]bool{}
	for i, s := range f.Steps {
		loc := location{procedureFile,fmt.Sprintf("/steps/%d",i)}
		if s.ID == "" || ids[s.ID] { add("DFD_STEP_ID_INVALID","error",procedureFile,loc.Pointer+"/id","step id is missing or duplicated") }
		ids[s.ID] = true
		node := stepNode{ID:s.ID,Label:s.BusinessAction,Realization:s.Realization,
			NoApplicationFunction:len(s.ApplicationPatterns)==0,
			FunctionRefs:[]functionRef{},Source:loc}
		for j, ref := range s.ApplicationPatterns {
			used[ref] = true
			resolved := patterns[ref]
			link := functionRef{ID:ref,Status:"unresolved"}
			if len(resolved)==1 {
				link.Status="resolved"
				link.Data=string(resolved[0].Data)
				link.Relation=string(resolved[0].Relation)
				link.Capability=string(resolved[0].Capability)
			} else {
				add("DFD_FUNCTION_REFERENCE_NOT_EXACT","error",procedureFile,fmt.Sprintf("/steps/%d/applicationPatterns/%d",i,j),
					fmt.Sprintf("Pattern %q exact resolution count=%d",ref,len(resolved)))
			}
			node.FunctionRefs=append(node.FunctionRefs,link)
		}
		p.Steps=append(p.Steps,node)
	}
	seenTriggers := map[string]bool{}
	for i,t := range r.Triggers {
		loc := location{reviewFile,fmt.Sprintf("/triggers/%d",i)}
		if t.ID=="" || seenTriggers[t.ID] { add("DFD_TRIGGER_ID_INVALID","error",reviewFile,loc.Pointer+"/id","trigger id missing or duplicate") }
		seenTriggers[t.ID]=true
		if t.Kind!="human" && t.Kind!="event" && t.Kind!="schedule" {
			add("DFD_TRIGGER_KIND_INVALID","error",reviewFile,loc.Pointer+"/kind","unsupported trigger kind")
		}
		if !ids[t.TargetStep] { add("DFD_TRIGGER_TARGET_MISSING","error",reviewFile,loc.Pointer+"/targetStep","target Step not found") }
		p.Triggers=append(p.Triggers,triggerNode{t.ID,t.Kind,t.Label,t.TargetStep,"proposal","unbound",loc})
	}
	for i,e := range r.Flows {
		loc := location{reviewFile,fmt.Sprintf("/flows/%d",i)}
		if !ids[e.From] || !ids[e.To] { add("DFD_FLOW_ENDPOINT_MISSING","error",reviewFile,loc.Pointer,"flow Step endpoint not found") }
		if e.Kind!="sequence" && e.Kind!="branch" && e.Kind!="exception" {
			add("DFD_FLOW_KIND_INVALID","error",reviewFile,loc.Pointer+"/kind","unsupported flow kind")
		}
		if (e.Kind=="branch" || e.Kind=="exception") && strings.TrimSpace(e.Condition)=="" {
			add("DFD_CONDITION_MISSING","error",reviewFile,loc.Pointer+"/condition","branch and exception need explicit condition")
		}
		p.Flows=append(p.Flows,flowEdge{e.From,e.To,e.Kind,e.Condition,"proposal",loc})
	}
	info, infoDiagnostics := buildInformationProjection(r, ids, world, reviewFile)
	p.Information = info
	p.Diagnostics = append(p.Diagnostics, infoDiagnostics...)
	// The set declares procedure-level endpoints. Remote Step existence is
	// not proven here: this adapter loads only one Procedure's Step source.
	procedureCounts:=map[string]int{}
	for i,member:=range set.Procedures {
		procedureCounts[member.ID]++
		if strings.TrimSpace(member.ID)=="" || strings.TrimSpace(member.Artifact)=="" || procedureCounts[member.ID]>1 {
			add("DFD_SET_MEMBER_INVALID","error",setFile,fmt.Sprintf("/procedures/%d",i),"reference set requires unique non-empty Procedure ID and artifact locator")
		}
	}
	if procedureCounts[f.Procedure.ID]!=1 {
		add("DFD_CURRENT_PROCEDURE_NOT_IN_SET","error",setFile,"/procedures","current Procedure must be listed exactly once")
	}
	for i,h := range set.Handoffs {
		if h.From.Procedure!=f.Procedure.ID && h.To.Procedure!=f.Procedure.ID { continue }
		loc := location{setFile,fmt.Sprintf("/handoffs/%d",i)}
		if procedureCounts[h.From.Procedure]!=1 {
			add("DFD_HANDOFF_FROM_PROCEDURE_NOT_EXACT","error",setFile,loc.Pointer+"/from/procedure","Handoff from Procedure not present exactly once in set")
		}
		if procedureCounts[h.To.Procedure]!=1 {
			add("DFD_HANDOFF_TO_PROCEDURE_NOT_EXACT","error",setFile,loc.Pointer+"/to/procedure","Handoff to Procedure not present exactly once in set")
		}
		if h.From.Procedure==f.Procedure.ID && !ids[h.From.Step] {
			add("DFD_HANDOFF_FROM_MISSING","error",setFile,loc.Pointer+"/from/step","handoff source Step not found")
		}
		if h.To.Procedure==f.Procedure.ID && !ids[h.To.Step] {
			add("DFD_HANDOFF_TO_MISSING","error",setFile,loc.Pointer+"/to/step","handoff target Step not found")
		}
		if strings.TrimSpace(h.Condition)=="" {
			add("DFD_HANDOFF_CONDITION_MISSING","error",setFile,loc.Pointer+"/condition","handoff requires an explicit condition")
		}
		p.Handoffs=append(p.Handoffs,handoffEdge{h.From,h.To,h.Condition,"reference-authoring-fixture",loc})
	}
	for _, a := range world.Applications {
		if string(a.ID)!=f.Application { continue }
		for _, pattern := range a.Patterns {
			if !used[string(pattern.ID)] { p.UnreferencedPatterns=append(p.UnreferencedPatterns,string(pattern.ID)) }
		}
	}
	sort.Strings(p.UnreferencedPatterns)
	sort.Slice(p.Diagnostics,func(i,j int) bool {
		a,b:=p.Diagnostics[i],p.Diagnostics[j]
		if a.Source.File!=b.Source.File { return a.Source.File<b.Source.File }
		if a.Source.Pointer!=b.Source.Pointer { return a.Source.Pointer<b.Source.Pointer }
		return a.Code<b.Code
	})
	return p
}

// DOT is a disposable layout input, not semantic Source. Dashed arrows denote
// review proposals; a Handoff is not represented as automatic execution.
func dot(p projection) string {
	var b strings.Builder
	b.WriteString("digraph business_dfd {\n  rankdir=LR;\n")
	for _,s := range p.Steps {
		shape:="box"
		if strings.Contains(s.Realization,"manual") || strings.Contains(s.Realization,"external") { shape="note" }
		fmt.Fprintf(&b,"  %s [label=%s, shape=%s];\n",strconv.Quote("step:"+s.ID),strconv.Quote(s.ID+"\\n"+s.Label),shape)
		for _, ref := range s.FunctionRefs {
			fmt.Fprintf(&b,"  %s [label=%s, shape=component];\n",strconv.Quote("function:"+ref.ID),strconv.Quote(ref.ID))
			fmt.Fprintf(&b,"  %s -> %s [style=dotted, label=\"uses\"];\n",strconv.Quote("step:"+s.ID),strconv.Quote("function:"+ref.ID))
		}
	}
	for _,t := range p.Triggers {
		fmt.Fprintf(&b,"  %s [label=%s, shape=oval];\n",strconv.Quote("trigger:"+t.ID),strconv.Quote(t.Kind+": "+t.Label))
		fmt.Fprintf(&b,"  %s -> %s [style=dashed, label=\"proposal / unbound\"];\n",strconv.Quote("trigger:"+t.ID),strconv.Quote("step:"+t.TargetStep))
	}
	for _,e := range p.Flows {
		fmt.Fprintf(&b,"  %s -> %s [style=dashed, label=%s];\n",
			strconv.Quote("step:"+e.From),strconv.Quote("step:"+e.To),strconv.Quote(e.Kind+": "+e.Condition))
	}
	appendInformationDOT(&b, p.Information)
	appendBusinessContextDOT(&b, p)
	b.WriteString("}\n")
	return b.String()
}

func main() {
	procedureFile:=flag.String("procedure","reference/business-system-design/order-entry-business-procedure.json","procedure reference fixture")
	reviewFile:=flag.String("review","reference/business-system-design/order-entry-dfd-review-proposal.json","explicit review proposal")
	setFile:=flag.String("set","reference/business-system-design/order-entry-procedure-reference-set.json","handoff source set")
	workspaceDir:=flag.String("workspace","reference/workspace","Workspace for exact Pattern resolution")
	format:=flag.String("format","json","json or dot")
	selectElement:=flag.String("select","","optional exact read-only navigation selector kind:id")
	reviewNext:=flag.String("review-next","","optional second unaccepted DFD review proposal for read-only semantic comparison")
	flag.Parse()
	p,sourceSnapshot,err:=runGuarded(*procedureFile,*reviewFile,*setFile,*workspaceDir)
	if err!=nil { fmt.Fprintln(os.Stderr,err); os.Exit(1) }
	if *reviewNext!="" {
		if *selectElement!="" || *format!="json" {
			fmt.Fprintln(os.Stderr,"review preview requires -format json and cannot be combined with -select")
			os.Exit(1)
		}
		var before, after review
		if err:=decodeFile(*reviewFile,&before);err!=nil {fmt.Fprintln(os.Stderr,err);os.Exit(1)}
		nextProjection,nextSnapshots,err:=runGuarded(*procedureFile,*reviewNext,*setFile,*workspaceDir)
		if err!=nil {fmt.Fprintln(os.Stderr,err);os.Exit(1)}
		if err:=decodeFile(*reviewNext,&after);err!=nil {fmt.Fprintln(os.Stderr,err);os.Exit(1)}
		compared,err:=previewReview(before,after,p,nextProjection,sourceSnapshot,nextSnapshots)
		if err!=nil {fmt.Fprintln(os.Stderr,err);os.Exit(1)}
		// Revalidate both source inputs after the comparison; do not silently
		// reuse source positions if another editor changed either fixture.
		if _,err:=navigateGuarded(p,"step:"+p.Steps[0].ID,sourceSnapshot);err!=nil {fmt.Fprintln(os.Stderr,err);os.Exit(1)}
		if _,err:=navigateGuarded(nextProjection,"step:"+nextProjection.Steps[0].ID,nextSnapshots);err!=nil {fmt.Fprintln(os.Stderr,err);os.Exit(1)}
		enc:=json.NewEncoder(os.Stdout);enc.SetIndent("","  ")
		if err:=enc.Encode(compared);err!=nil {fmt.Fprintln(os.Stderr,err);os.Exit(1)}
		if compared.AcceptanceStatus=="blocked-unresolved-diagnostic" {os.Exit(2)}
		return
	}
	if *selectElement!="" {
		if *format!="json" {fmt.Fprintln(os.Stderr,"navigation selection requires -format json");os.Exit(1)}
		selected,navErr:=navigateGuarded(p,*selectElement,sourceSnapshot)
		if navErr!=nil {fmt.Fprintln(os.Stderr,navErr);os.Exit(1)}
		enc:=json.NewEncoder(os.Stdout)
		enc.SetIndent("","  ")
		if encErr:=enc.Encode(selected);encErr!=nil {fmt.Fprintln(os.Stderr,encErr);os.Exit(1)}
		if selected.Status=="unresolved" {os.Exit(2)}
		return
	}
	switch *format {
	case "json":
		enc:=json.NewEncoder(os.Stdout);enc.SetIndent("","  ");err=enc.Encode(p)
	case "dot":
		_,err=fmt.Fprint(os.Stdout,dot(p))
	default:
		err=fmt.Errorf("unsupported format %q",*format)
	}
	if err!=nil { fmt.Fprintln(os.Stderr,err);os.Exit(1) }
	for _,d:=range p.Diagnostics { if d.Severity=="error" { os.Exit(2) } }
}
