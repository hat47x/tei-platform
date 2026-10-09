package languagereference

import (
 "fmt"
 "tei/internal/semantic"
)

// BindPinnedSemanticInput verifies the data identity against an externally
// accepted Namespace, then composes only exact-pinned operator contracts.
// The rows/<id> type remains a temporary reference convention.
func BindPinnedSemanticInput(ns semantic.Namespace, dataID string, calls []PinnedCall, catalog []OperatorContract) (string,error) {
 if string(ns.ID)=="" || dataID=="" {return "",fmt.Errorf("missing semantic namespace or data identity")}
 matches:=0
 for _,d:=range ns.Data {if string(d.ID)==dataID {matches++}}
 if matches!=1 {return "",fmt.Errorf("semantic data %q: expected exactly one definition, got %d",dataID,matches)}
 return ComposePinned("rows/"+dataID,calls,catalog)
}
