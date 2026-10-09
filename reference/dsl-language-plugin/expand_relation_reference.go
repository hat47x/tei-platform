package languagereference

import (
 "fmt"
 "tei/internal/core"
 "tei/internal/semantic"
)

// ExpandRelationReference validates the semantic preconditions of a single
// relation expansion. It does not execute a join or authorize an extension.
// Exact pinned operator contracts remain the authority for composition.
func ExpandRelationReference(ns semantic.Namespace, sourceID, relationID core.CanonicalID, call PinnedCall, catalog []OperatorContract) (string,error) {
 if err:=ValidateRelationKeys(ns,sourceID,relationID);err!=nil{return "",err}
 var source *semantic.Data
 for i:=range ns.Data {
  if ns.Data[i].ID==sourceID {source=&ns.Data[i];break}
 }
 var relation *semantic.Relation
 for i:=range source.Relations {
  if source.Relations[i].ID==relationID {relation=&source.Relations[i];break}
 }
 if len(relation.Keys)==0 {return "",fmt.Errorf("relation %q has no executable key mapping",relationID)}
 if call.Contract.ID!="std.op.expand" {return "",fmt.Errorf("expected std.op.expand contract")}
 input:="rows/"+string(sourceID)
 expectedOutput:="rows/"+string(relation.Target)
 if call.Contract.InputType!=input || call.Contract.OutputType!=expectedOutput {
  return "",fmt.Errorf("expand contract does not match semantic relation endpoints")
 }
 return ComposePinned(input,[]PinnedCall{call},catalog)
}
