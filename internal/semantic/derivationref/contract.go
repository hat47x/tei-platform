// Package derivationref is an experimental, isolated contract pressure test.
// It is not a Canonical model, a generic expression engine, or an accepted API.
package derivationref

import (
 "errors"
 "fmt"
 "math/big"
)

type InputState string
const (
 Present InputState = "present"
 Null InputState = "null"
 Missing InputState = "missing"
 AbsentRelation InputState = "absent-relation"
 MultipleRelations InputState = "multiple-relations"
)

type Input struct {
 State InputState
 Decimal string
}

type Definition struct {
 ID string
 Inputs []string
 Operator string
 Pure bool
}

type Binding struct {
 RuleID string
 TargetID string
 Scope string
}

var ErrInput = errors.New("derivation input unavailable")
var ErrCycle = errors.New("derivation dependency cycle")
var ErrImpure = errors.New("impure derivation")
var ErrOperator = errors.New("unsupported derivation operator")

// ValidateGraph checks references and cycles without executing any rule.
func ValidateGraph(defs []Definition, dependencies map[string][]string) error {
 byID:=make(map[string]Definition,len(defs))
 for _,d:=range defs {
  if d.ID=="" {return fmt.Errorf("empty rule id")}
  if _,exists:=byID[d.ID];exists{return fmt.Errorf("duplicate rule %q",d.ID)}
  if !d.Pure{return fmt.Errorf("%w: %s",ErrImpure,d.ID)}
  byID[d.ID]=d
 }
 state:=map[string]uint8{}
 var visit func(string)error
 visit=func(id string)error {
  if _,ok:=byID[id];!ok{return fmt.Errorf("unknown rule %q",id)}
  if state[id]==1{return fmt.Errorf("%w: %s",ErrCycle,id)}
  if state[id]==2{return nil}
  state[id]=1
  for _,dep:=range dependencies[id] {if err:=visit(dep);err!=nil{return err}}
  state[id]=2
  return nil
 }
 for id:=range byID {if err:=visit(id);err!=nil{return err}}
 return nil
}

// Multiply is deliberately narrow: exact decimal multiplication with explicit
// null propagation. Missing and relation cardinality failures are not NULL.
func Multiply(left,right Input)(Input,error){
 for _,in:=range []Input{left,right}{
  switch in.State {
  case Missing,AbsentRelation,MultipleRelations:
   return Input{},fmt.Errorf("%w: %s",ErrInput,in.State)
  case Present,Null:
  default:
   return Input{},fmt.Errorf("%w: invalid state %q",ErrInput,in.State)
  }
 }
 if left.State==Null||right.State==Null{return Input{State:Null},nil}
 a,ok:=new(big.Rat).SetString(left.Decimal)
 if !ok{return Input{},fmt.Errorf("%w: invalid decimal input",ErrOperator)}
 b,ok:=new(big.Rat).SetString(right.Decimal)
 if !ok{return Input{},fmt.Errorf("%w: invalid decimal input",ErrOperator)}
 product:=new(big.Rat).Mul(a,b)
 // A rational with a non-terminating decimal expansion needs an explicit
 // rounding policy; multiplication of finite decimals always terminates.
 denominator:=new(big.Int).Set(product.Denom())
 two,five:=big.NewInt(2),big.NewInt(5)
 rem:=new(big.Int)
 scale:=0
 for _,factor:=range []*big.Int{two,five}{
  for {
   q:=new(big.Int)
   q.QuoRem(denominator,factor,rem)
   if rem.Sign()!=0{break}
   denominator=q
   scale++
  }
 }
 if denominator.Cmp(big.NewInt(1))!=0{return Input{},fmt.Errorf("%w: nonterminating decimal",ErrOperator)}
 return Input{State:Present,Decimal:product.FloatString(scale)},nil
}
