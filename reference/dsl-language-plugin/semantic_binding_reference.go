package languagereference

import (
 "fmt"
 "tei/internal/semantic"
)

// BindSemanticInput connects an operator pipeline to an existing semantic
// Data identity. It does not introduce a second Semantic Extension resolver.
// The caller must supply an already accepted namespace; this reference cannot
// establish acceptance itself.
func BindSemanticInput(ns semantic.Namespace, dataID string, catalog []Operator, calls []Call) (string,error) {
 if string(ns.ID)=="" {return "",fmt.Errorf("missing semantic namespace")}
 matches:=0
 for _,d:=range ns.Data {if string(d.ID)==dataID {matches++}}
 if matches!=1 {return "",fmt.Errorf("semantic data %q: expected one definition, got %d",dataID,matches)}
 inputType:="rows/"+dataID
 return Compose(inputType,calls,catalog)
}
