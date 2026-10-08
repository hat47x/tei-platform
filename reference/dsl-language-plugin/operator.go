package languagereference

import (
 "errors"
 "fmt"
)

var (
 ErrOperatorMissing = errors.New("operator contract is unavailable")
 ErrOperatorAmbiguous = errors.New("operator contract is ambiguous")
 ErrOperatorInvalid = errors.New("operator contract is incomplete")
 ErrEffectEscalation = errors.New("operator composition escalates effects")
)

// OperatorContract is an E2 reference-only semantic declaration.
// Effect is deliberately restricted to pure: side effects require existing
// Capability authorization, not a new DSL execution shortcut.
type OperatorContract struct {
 ID string
 Version string
 Fingerprint string
 Effect string
 InputType string
 OutputType string
}

// ResolveOperator never chooses a newer version or the first registered item.
func ResolveOperator(pin OperatorContract, catalog []OperatorContract) (OperatorContract,error) {
 if pin.ID=="" || pin.Version=="" || pin.Fingerprint=="" || pin.InputType=="" || pin.OutputType=="" || pin.Effect!="pure" {
  return OperatorContract{},ErrOperatorInvalid
 }
 matches:=0
 var found OperatorContract
 for _,c:=range catalog {
  if c.ID==pin.ID && c.Version==pin.Version && c.Fingerprint==pin.Fingerprint {
   matches++
   found=c
  }
 }
 if matches==0 {return OperatorContract{},fmt.Errorf("%w: %s@%s",ErrOperatorMissing,pin.ID,pin.Version)}
 if matches>1 {return OperatorContract{},fmt.Errorf("%w: %s@%s",ErrOperatorAmbiguous,pin.ID,pin.Version)}
 if found!=pin {return OperatorContract{},fmt.Errorf("%w: incompatible signature for %s",ErrOperatorInvalid,pin.ID)}
 return found,nil
}

// ComposePure verifies that the output type of each step is exactly the input
// type of the next step. This is not a general type system or execution plan.
func ComposePure(steps []OperatorContract) (string,error) {
 if len(steps)==0 {return "",ErrOperatorInvalid}
 for i,s:=range steps {
  if s.Effect!="pure" {return "",fmt.Errorf("%w: %s",ErrEffectEscalation,s.ID)}
  if s.ID=="" || s.InputType=="" || s.OutputType=="" || s.Version=="" || s.Fingerprint=="" {return "",ErrOperatorInvalid}
  if i>0 && steps[i-1].OutputType!=s.InputType {
   return "",fmt.Errorf("%w: %s -> %s type mismatch",ErrOperatorInvalid,steps[i-1].ID,s.ID)
  }
 }
 return steps[len(steps)-1].OutputType,nil
}
