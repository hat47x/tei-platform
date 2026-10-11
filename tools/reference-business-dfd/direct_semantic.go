package main

import (
	"fmt"

	"tei/internal/ir"
)

// Direct semantic references belong to Procedure authoring, not to an
// Application Pattern. In particular a manual Step may directly refer to
// Data or Validation without acquiring a fake executable Function.
type directMeaningRef struct {
	Kind string `json:"kind"`
	ID string `json:"id"`
	Status string `json:"status"`
	Authority string `json:"authority"`
	Source location `json:"source"`
}

func explicitMeaningRefs(s step, world ir.World, procedureFile string, stepIndex int) ([]directMeaningRef, []diagnostic) {
	refs:=[]directMeaningRef{}
	diagnostics:=[]diagnostic{}
	counts:=map[string]map[string]int{
		"semantic-data":{},
		"semantic-relation":{},
		"semantic-validation":{},
		"semantic-capability":{},
	}
	for _,namespace:=range world.Semantics {
		for _,data:=range namespace.Data {
			counts["semantic-data"][string(data.ID)]++
			for _,relation:=range data.Relations {
				counts["semantic-relation"][string(relation.ID)]++
			}
		}
		for _,v:=range namespace.Validations {
			counts["semantic-validation"][string(v.ID)]++
		}
		for _,c:=range namespace.Capabilities {
			counts["semantic-capability"][string(c.ID)]++
		}
	}
	appendRefs:=func(kind,field string,values []string) {
		for j,id:=range values {
			pointer:=fmt.Sprintf("/steps/%d/%s/%d",stepIndex,field,j)
			loc:=location{File:procedureFile,Pointer:pointer}
			status:="resolved-id-not-accepted"
			if counts[kind][id]!=1 {
				status="unresolved"
				diagnostics=append(diagnostics,diagnostic{
					Code:"DFD_DIRECT_SEMANTIC_REF_NOT_EXACT",
					Severity:"error",
					Source:loc,
					Message:fmt.Sprintf("%s direct reference %q exact count=%d",kind,id,counts[kind][id]),
				})
			}
			refs=append(refs,directMeaningRef{Kind:kind,ID:id,Status:status,
				Authority:"procedure-reference-only",Source:loc})
		}
	}
	appendRefs("semantic-data","data",s.Data)
	appendRefs("semantic-relation","relations",s.Relations)
	appendRefs("semantic-validation","validations",s.Validations)
	appendRefs("semantic-capability","capabilities",s.Capabilities)
	return refs,diagnostics
}
