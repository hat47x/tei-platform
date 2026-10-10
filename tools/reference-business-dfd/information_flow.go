package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"tei/internal/ir"
)

// Information exchanges and role ownership are strictly proposal-local
// annotations on the existing Procedure Step identities. They are deliberately
// not added to the normalized Business Procedure, Handoff or Canonical World.
type businessRole struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type externalParty struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type informationStore struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	SemanticData string `json:"semanticData,omitempty"`
}

type stepResponsibility struct {
	Step string `json:"step"`
	Role string `json:"role"`
}

type informationEndpoint struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type informationFlow struct {
	ID      string              `json:"id"`
	From    informationEndpoint `json:"from"`
	To      informationEndpoint `json:"to"`
	Payload string              `json:"payload"`
}

type roleNode struct {
	ID     string   `json:"id"`
	Label  string   `json:"label"`
	Source location `json:"source"`
}

type partyNode struct {
	ID     string   `json:"id"`
	Label  string   `json:"label"`
	Source location `json:"source"`
}

type storeNode struct {
	ID           string   `json:"id"`
	Label        string   `json:"label"`
	SemanticData string   `json:"semanticData,omitempty"`
	DataStatus   string   `json:"dataStatus"`
	Source       location `json:"source"`
}

type responsibilityEdge struct {
	Step      string   `json:"step"`
	Role      string   `json:"role"`
	Authority string   `json:"authority"`
	Source    location `json:"source"`
}

type informationEdge struct {
	ID        string              `json:"id"`
	From      informationEndpoint `json:"from"`
	To        informationEndpoint `json:"to"`
	Payload   string              `json:"payload"`
	Authority string              `json:"authority"`
	Source    location            `json:"source"`
}

type informationProjection struct {
	Roles            []roleNode           `json:"roles"`
	ExternalParties   []partyNode          `json:"externalParties"`
	Stores            []storeNode          `json:"stores"`
	Responsibilities  []responsibilityEdge `json:"responsibilities"`
	InformationFlows  []informationEdge    `json:"informationFlows"`
}

// Candidate information exchanges are intentionally distinct from control-flow
// arrows: a sequence/order or a Handoff does not imply a transferred payload.
func buildInformationProjection(r review, stepIDs map[string]bool, world ir.World, sourceFile string) (informationProjection, []diagnostic) {
	out := informationProjection{
		Roles: []roleNode{}, ExternalParties: []partyNode{}, Stores: []storeNode{},
		Responsibilities: []responsibilityEdge{}, InformationFlows: []informationEdge{},
	}
	diagnostics := []diagnostic{}
	add := func(code, pointer, msg string) {
		diagnostics = append(diagnostics, diagnostic{
			Code: code, Severity: "error", Source: location{sourceFile, pointer}, Message: msg,
		})
	}
	roles, parties, stores := map[string]bool{}, map[string]bool{}, map[string]bool{}
	dataCounts := map[string]int{}
	for _, ns := range world.Semantics {
		for _, d := range ns.Data {
			dataCounts[string(d.ID)]++
		}
	}

	for i, role := range r.BusinessRoles {
		pointer := fmt.Sprintf("/businessRoles/%d", i)
		if strings.TrimSpace(role.ID)=="" || roles[role.ID] {
			add("DFD_BUSINESS_ROLE_ID_INVALID", pointer+"/id", "business Role id missing or duplicated")
		}
		roles[role.ID]=true
		if strings.TrimSpace(role.Label)=="" {
			add("DFD_BUSINESS_ROLE_LABEL_MISSING", pointer+"/label", "business Role must have a label")
		}
		out.Roles=append(out.Roles, roleNode{ID:role.ID, Label:role.Label, Source:location{sourceFile,pointer}})
	}
	for i, party := range r.ExternalParties {
		pointer := fmt.Sprintf("/externalParties/%d", i)
		if strings.TrimSpace(party.ID)=="" || parties[party.ID] {
			add("DFD_PARTY_ID_INVALID", pointer+"/id", "external Party id missing or duplicated")
		}
		parties[party.ID]=true
		if strings.TrimSpace(party.Label)=="" {
			add("DFD_PARTY_LABEL_MISSING", pointer+"/label", "external Party must have a label")
		}
		out.ExternalParties=append(out.ExternalParties,partyNode{ID:party.ID,Label:party.Label,Source:location{sourceFile,pointer}})
	}
	for i, store := range r.Stores {
		pointer:=fmt.Sprintf("/stores/%d",i)
		if strings.TrimSpace(store.ID)=="" || stores[store.ID] {
			add("DFD_STORE_ID_INVALID", pointer+"/id", "Store id missing or duplicated")
		}
		stores[store.ID]=true
		if strings.TrimSpace(store.Label)=="" {
			add("DFD_STORE_LABEL_MISSING", pointer+"/label", "Store must have a label")
		}
		status:="not-declared"
		if store.SemanticData!="" {
			if dataCounts[store.SemanticData]==1 {
				status="resolved"
			} else {
				status="unresolved"
				add("DFD_STORE_DATA_REFERENCE_NOT_EXACT",pointer+"/semanticData",
					fmt.Sprintf("Semantic Data %q exact resolution count=%d",store.SemanticData,dataCounts[store.SemanticData]))
			}
		}
		out.Stores=append(out.Stores,storeNode{ID:store.ID,Label:store.Label,
			SemanticData:store.SemanticData,DataStatus:status,Source:location{sourceFile,pointer}})
	}
	assigned:=map[string]bool{}
	for i, a := range r.Responsibilities {
		pointer:=fmt.Sprintf("/responsibilities/%d",i)
		if !stepIDs[a.Step] {
			add("DFD_RESPONSIBILITY_STEP_MISSING",pointer+"/step","assigned Step not found")
		}
		if !roles[a.Role] {
			add("DFD_RESPONSIBILITY_ROLE_MISSING",pointer+"/role","business Role not found")
		}
		key:=strconv.Quote(a.Step)+":"+strconv.Quote(a.Role)
		if assigned[key] {
			add("DFD_RESPONSIBILITY_DUPLICATE",pointer,"duplicate Step / Business Role assignment")
		}
		assigned[key]=true
		out.Responsibilities=append(out.Responsibilities,responsibilityEdge{Step:a.Step,Role:a.Role,
			Authority:"proposal",Source:location{sourceFile,pointer}})
	}
	known:=func(e informationEndpoint) bool {
		switch e.Kind {
		case "step": return stepIDs[e.ID]
		case "party": return parties[e.ID]
		case "store": return stores[e.ID]
		default: return false
		}
	}
	seenFlows:=map[string]bool{}
	for i, flow := range r.InformationFlows {
		pointer:=fmt.Sprintf("/informationFlows/%d",i)
		if strings.TrimSpace(flow.ID)=="" || seenFlows[flow.ID] {
			add("DFD_INFORMATION_FLOW_ID_INVALID",pointer+"/id","information Flow id missing or duplicated")
		}
		seenFlows[flow.ID]=true
		if !known(flow.From) { add("DFD_INFORMATION_FROM_MISSING",pointer+"/from","from endpoint kind/id not found") }
		if !known(flow.To) { add("DFD_INFORMATION_TO_MISSING",pointer+"/to","to endpoint kind/id not found") }
		if flow.From==flow.To {
			add("DFD_INFORMATION_SELF_LOOP",pointer,"information Flow must connect distinct endpoints")
		}
		// A data flow diagram must have a Process boundary for all exchanges
		// between external Parties and Stores; it does not imply a direct
		// external write, bypass of security, or executable action.
		if flow.From.Kind!="step" && flow.To.Kind!="step" {
			add("DFD_INFORMATION_PROCESS_REQUIRED",pointer,"information exchange must involve at least one Business Step")
		}
		if strings.TrimSpace(flow.Payload)=="" {
			add("DFD_INFORMATION_PAYLOAD_MISSING",pointer+"/payload","an explicit information payload is required")
		}
		out.InformationFlows=append(out.InformationFlows,informationEdge{ID:flow.ID,
			From:flow.From,To:flow.To,Payload:flow.Payload,Authority:"proposal",Source:location{sourceFile,pointer}})
	}
	sort.Slice(diagnostics,func(i,j int) bool {
		if diagnostics[i].Source.Pointer!=diagnostics[j].Source.Pointer {
			return diagnostics[i].Source.Pointer<diagnostics[j].Source.Pointer
		}
		return diagnostics[i].Code<diagnostics[j].Code
	})
	return out, diagnostics
}

func appendInformationDOT(b *strings.Builder, info informationProjection) {
	for _, party := range info.ExternalParties {
		fmt.Fprintf(b,"  %s [label=%s, shape=box, style=rounded];\n",
			strconv.Quote("party:"+party.ID),strconv.Quote(party.Label))
	}
	for _, store := range info.Stores {
		fmt.Fprintf(b,"  %s [label=%s, shape=cylinder];\n",
			strconv.Quote("store:"+store.ID),strconv.Quote(store.Label))
	}
	for _, responsibility := range info.Responsibilities {
		fmt.Fprintf(b,"  %s -> %s [style=dotted, label=\"business responsibility (proposal)\"];\n",
			strconv.Quote("role:"+responsibility.Role),strconv.Quote("step:"+responsibility.Step))
	}
	for _, role := range info.Roles {
		fmt.Fprintf(b,"  %s [label=%s, shape=plaintext];\n",
			strconv.Quote("role:"+role.ID),strconv.Quote(role.Label))
	}
	for _, flow := range info.InformationFlows {
		from:=strconv.Quote(flow.From.Kind+":"+flow.From.ID)
		to:=strconv.Quote(flow.To.Kind+":"+flow.To.ID)
		fmt.Fprintf(b,"  %s -> %s [style=dashed, penwidth=2, label=%s];\n",
			from,to,strconv.Quote("information / proposal: "+flow.Payload))
	}
}
