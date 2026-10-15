// Package tablelifecycle is a synthetic, executable transaction reference for #1129.
// It is neither a Canonical lifecycle model nor legacy X-TEA runtime evidence.
package tablelifecycle

import (
 "context"
 "errors"
 "fmt"
 "sync"
)

// Phase describes observation stages, not a public Script hook API.
type Phase string
const (
 PhaseValidate Phase = "validate"
 PhaseAuthorize Phase = "authorize"
 PhaseRead Phase = "read-current"
 PhaseBefore Phase = "before-write"
 PhaseConsequence Phase = "transaction-consequence"
 PhaseCommit Phase = "commit"
 PhasePostCommit Phase = "post-commit-outbox"
)
const (
 Submitted = "submitted"
 NotSubmitted = "not_submitted"
 Denied = "denied"
 Unknown = "unknown"
)

type Submit struct {
 OperationID string
 SubjectID string
 OrderID string
 ExpectedVersion uint64
 Units int
}
type Cancel struct {
 OperationID string
 SubjectID string
 OrderID string
 ExpectedVersion uint64
}

// Authorizer is supplied by the existing authorization capability; this
// fixture makes no claim about atomic authorization revocation at commit.
type Authorizer interface {
 Allow(context.Context,string,string)(bool,error)
}
type AuthorizeFunc func(context.Context,string,string)(bool,error)
func(f AuthorizeFunc)Allow(ctx context.Context,subject,order string)(bool,error){
 return f(ctx,subject,order)
}
type Result struct {
 Outcome string
 Committed bool
 Trace []Phase
}
type Order struct {
 ID string
 State string
 Version uint64
 Units int
}
type Notice struct {
 OperationID string
 OrderID string
 Kind string
}
type Snapshot struct {
 Order Order
 Available int
 Reserved int
 History []string
 Notices []Notice
}
type receipt struct {
 Request any // either comparable Submit or Cancel
 Result Result
}
type state struct {
 orders map[string]Order
 available int
 reservations map[string]int
 history map[string][]string
 receipts map[string]receipt
 pending map[string]Notice
 delivered map[string]bool
}
func(s state)copy()state{
 c:=state{orders:make(map[string]Order,len(s.orders)),available:s.available,
 reservations:make(map[string]int,len(s.reservations)),
 history:make(map[string][]string,len(s.history)),
 receipts:make(map[string]receipt,len(s.receipts)),
 pending:make(map[string]Notice,len(s.pending)),
 delivered:make(map[string]bool,len(s.delivered))}
 for k,v:=range s.orders{c.orders[k]=v}
 for k,v:=range s.reservations{c.reservations[k]=v}
 for k,v:=range s.history{c.history[k]=append([]string(nil),v...)}
 for k,v:=range s.receipts{c.receipts[k]=v}
 for k,v:=range s.pending{c.pending[k]=v}
 for k,v:=range s.delivered{c.delivered[k]=v}
 return c
}

// Store is a single-process fixture. Staged writes are committed under the
// same mutex that validates the old version. It is not durable across restarts.
type Store struct {
 mu sync.Mutex
 state state
 failBeforeCommit bool
 loseAcknowledgement bool
 delivering map[string]bool
}
func NewStore(available int,orders ...Order)(*Store,error){
 if available<0{return nil,errors.New("negative inventory")}
 s:=state{orders:map[string]Order{},available:available,reservations:map[string]int{},
 history:map[string][]string{},receipts:map[string]receipt{},pending:map[string]Notice{},delivered:map[string]bool{}}
 for _,o:=range orders{
  if o.ID==""||o.Units<=0||o.State==""{return nil,errors.New("invalid initial order")}
  if _,exists:=s.orders[o.ID];exists{return nil,errors.New("duplicate initial order")}
  s.orders[o.ID]=o
 }
 return &Store{state:s,delivering:map[string]bool{}},nil
}
func validate(c Submit)error{
 if c.OperationID==""||c.SubjectID==""||c.OrderID==""||c.Units<=0{return errors.New("invalid submit identity or units")}
 return nil
}
func cloneResult(r Result)Result{
 r.Trace=append([]Phase(nil),r.Trace...)
 return r
}

// Execute performs validation, authorization, protected read, state/version
// check and owner-side reservation and history updates in a staged transaction.
func(s *Store)Execute(ctx context.Context,c Submit,auth Authorizer)(Result,error){
 trace:=[]Phase{PhaseValidate}
 if err:=validate(c);err!=nil{return Result{},err}
 if auth==nil{return Result{},errors.New("authorization capability missing")}
 if err:=ctx.Err();err!=nil{return Result{},err}
 trace=append(trace,PhaseAuthorize)
 allowed,err:=auth.Allow(ctx,c.SubjectID,c.OrderID)
 if err!=nil{return Result{},fmt.Errorf("authorization unavailable: %w",err)}
 if !allowed{return Result{Outcome:Denied,Trace:trace},nil}
 s.mu.Lock();defer s.mu.Unlock()
 if err:=ctx.Err();err!=nil{return Result{},err}
 trace=append(trace,PhaseRead)
 if r,ok:=s.state.receipts[c.OperationID];ok{
  if r.Request!=c{return Result{},errors.New("idempotency identity conflicts with a different request")}
  return cloneResult(r.Result),nil
 }
 current,ok:=s.state.orders[c.OrderID]
 if !ok||current.Version!=c.ExpectedVersion||current.State!="pending"||current.Units!=c.Units||s.state.available<c.Units{
  return Result{Outcome:NotSubmitted,Trace:trace},nil
 }
 trace=append(trace,PhaseBefore)
 staged:=s.state.copy()
 order:=staged.orders[c.OrderID]
 order.State=Submitted
 order.Version++
 staged.orders[c.OrderID]=order
 trace=append(trace,PhaseConsequence)
 if err:=applyReservation(&staged,c);err!=nil{return Result{},err}
 if s.failBeforeCommit{return Result{},errors.New("injected pre-commit transaction failure")}
 if err:=ctx.Err();err!=nil{return Result{},err}
 trace=append(trace,PhaseCommit,PhasePostCommit)
 confirmed:=Result{Outcome:Submitted,Committed:true,Trace:append([]Phase(nil),trace...)}
 staged.receipts[c.OperationID]=receipt{Request:c,Result:confirmed}
 staged.pending[c.OperationID]=Notice{OperationID:c.OperationID,OrderID:c.OrderID,Kind:"order.submitted"}
 s.state=staged // synthetic atomic commit, including outbox
 if s.loseAcknowledgement{return Result{Outcome:Unknown,Trace:trace},nil}
 return confirmed,nil
}
// applyReservation is the inventory owner's action; unrelated entity changes
// cannot be smuggled into a pure derived field or into a public raw Script.
func applyReservation(staged *state,c Submit)error{
 if staged.reservations[c.OrderID]!=0{return errors.New("reservation already exists")}
 if staged.available<c.Units{return errors.New("insufficient inventory at consequence")}
 staged.available-=c.Units
 staged.reservations[c.OrderID]=c.Units
 staged.history[c.OrderID]=append(staged.history[c.OrderID],"submitted:"+c.OperationID)
 return nil
}

// Lookup checks authorization again before exposing an operation receipt.
// Absence of a receipt is not proof that a remote external effect did not happen.
func(s *Store)Lookup(ctx context.Context,c Submit,auth Authorizer)(Result,bool,error){
 if err:=validate(c);err!=nil{return Result{},false,err}
 if auth==nil{return Result{},false,errors.New("authorization capability missing")}
 allowed,err:=auth.Allow(ctx,c.SubjectID,c.OrderID)
 if err!=nil{return Result{},false,err}
 if !allowed{return Result{Outcome:Denied},false,nil}
 s.mu.Lock();defer s.mu.Unlock()
 r,ok:=s.state.receipts[c.OperationID]
 if !ok{return Result{},false,nil}
 if r.Request!=c{return Result{},false,errors.New("receipt identity conflicts with request")}
 return cloneResult(r.Result),true,nil
}

// SendPending calls transport only after commit and outside the store mutex.
// An uncertain remote response preserves the notice. Retry MUST use receiver-
// side de-duplication keyed by Notice.OperationID; exactly-once is not claimed.
func(s *Store)SendPending(ctx context.Context,operationID string,sender func(context.Context,Notice)error)(bool,error){
 if operationID==""||sender==nil{return false,errors.New("invalid delivery request")}
 s.mu.Lock()
 n,ok:=s.state.pending[operationID]
 if !ok||s.state.delivered[operationID]{s.mu.Unlock();return false,nil}
 if s.delivering[operationID]{s.mu.Unlock();return false,errors.New("delivery already in progress")}
 if err:=ctx.Err();err!=nil{s.mu.Unlock();return false,err}
 s.delivering[operationID]=true
 s.mu.Unlock()
 err:=sender(ctx,n)
 s.mu.Lock()
 delete(s.delivering,operationID)
 if err==nil{s.state.delivered[operationID]=true}
 s.mu.Unlock()
 if err!=nil{return false,err}
 return true,nil
}

// ExecuteCancel is a distinct business cancellation, not context cancellation.
// It atomically removes reservation, refunds inventory and appends history.
func(s *Store)ExecuteCancel(ctx context.Context,c Cancel,auth Authorizer)(Result,error){
 trace:=[]Phase{PhaseValidate}
 if c.OperationID==""||c.SubjectID==""||c.OrderID==""{return Result{},errors.New("invalid cancel identity")}
 if auth==nil{return Result{},errors.New("authorization capability missing")}
 if err:=ctx.Err();err!=nil{return Result{},err}
 trace=append(trace,PhaseAuthorize)
 allowed,err:=auth.Allow(ctx,c.SubjectID,c.OrderID)
 if err!=nil{return Result{},fmt.Errorf("authorization unavailable: %w",err)}
 if !allowed{return Result{Outcome:Denied,Trace:trace},nil}
 s.mu.Lock();defer s.mu.Unlock()
 if err:=ctx.Err();err!=nil{return Result{},err}
 trace=append(trace,PhaseRead)
 if r,ok:=s.state.receipts[c.OperationID];ok{
  if r.Request!=c{return Result{},errors.New("idempotency identity conflicts with a different request")}
  return cloneResult(r.Result),nil
 }
 current,ok:=s.state.orders[c.OrderID]
 units:=s.state.reservations[c.OrderID]
 if !ok||current.State!=Submitted||current.Version!=c.ExpectedVersion||units<=0{
  return Result{Outcome:NotSubmitted,Trace:trace},nil
 }
 trace=append(trace,PhaseBefore)
 staged:=s.state.copy()
 order:=staged.orders[c.OrderID]
 order.State="cancelled"
 order.Version++
 staged.orders[c.OrderID]=order
 trace=append(trace,PhaseConsequence)
 staged.available+=units
 delete(staged.reservations,c.OrderID)
 staged.history[c.OrderID]=append(staged.history[c.OrderID],"cancelled:"+c.OperationID)
 if s.failBeforeCommit{return Result{},errors.New("injected pre-commit transaction failure")}
 if err:=ctx.Err();err!=nil{return Result{},err}
 trace=append(trace,PhaseCommit,PhasePostCommit)
 result:=Result{Outcome:"cancelled",Committed:true,Trace:append([]Phase(nil),trace...)}
 staged.receipts[c.OperationID]=receipt{Request:c,Result:result}
 staged.pending[c.OperationID]=Notice{OperationID:c.OperationID,OrderID:c.OrderID,Kind:"order.cancelled"}
 s.state=staged
 if s.loseAcknowledgement{return Result{Outcome:Unknown,Trace:trace},nil}
 return result,nil
}

// Inspect is a synthetic test-only state observer, not an authorized API.
func(s *Store)Inspect(orderID string)(Snapshot,bool){
 s.mu.Lock();defer s.mu.Unlock()
 o,ok:=s.state.orders[orderID]
 if !ok{return Snapshot{},false}
 snap:=Snapshot{Order:o,Available:s.state.available,Reserved:s.state.reservations[orderID],
 History:append([]string(nil),s.state.history[orderID]...)}
 for _,n:=range s.state.pending{if n.OrderID==orderID{snap.Notices=append(snap.Notices,n)}}
 return snap,true
}
