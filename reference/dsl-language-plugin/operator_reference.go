package languagereference

import (
 "errors"
 "fmt"
)

// Operator is a reference-only semantic signature. It is not an accepted
// extension descriptor and grants no execution authority.
type Operator struct {
 ID string
 Version string
 InputType string
 OutputType string
 Effect string
}

// Call is a typed, syntax-independent expression candidate.
type Call struct {
 OperatorID string
 InputType string
 Args []string
}

// ResolveCall refuses unknown, ambiguous, effectful or incompatible calls.
func ResolveCall(call Call, catalog []Operator) (Operator, error) {
 var found Operator
 matches := 0
 for _, op := range catalog {
  if op.ID == call.OperatorID {found=op; matches++}
 }
 if matches!=1 {return Operator{},fmt.Errorf("operator %q: expected exactly one contract, got %d",call.OperatorID,matches)}
 if found.Version=="" || found.InputType=="" || found.OutputType=="" {return Operator{},errors.New("incomplete operator contract")}
 if found.Effect!="pure" {return Operator{},fmt.Errorf("operator %q requires explicit capability boundary",call.OperatorID)}
 if call.InputType!=found.InputType {return Operator{},fmt.Errorf("operator %q: expected %s, got %s",call.OperatorID,found.InputType,call.InputType)}
 return found,nil
}

// Compose checks a chain without invoking a provider or reading any data.
func Compose(inputType string, calls []Call, catalog []Operator) (string,error) {
 current:=inputType
 for i,call:=range calls {
  if call.InputType!=current {return "",fmt.Errorf("step %d: pipeline expects %s, got %s",i,current,call.InputType)}
  op,err:=ResolveCall(call,catalog)
  if err!=nil {return "",fmt.Errorf("step %d: %w",i,err)}
  current=op.OutputType
 }
 return current,nil
}
