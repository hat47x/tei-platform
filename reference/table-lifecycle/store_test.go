package tablelifecycle

import (
 "context"
 "errors"
 "sync"
 "testing"
)

var allowAll=AuthorizeFunc(func(context.Context,string,string)(bool,error){return true,nil})
func fixture(t *testing.T)(*Store,Submit){
 t.Helper()
 s,err:=NewStore(20,Order{ID:"o1",State:"pending",Version:7,Units:3})
 if err!=nil{t.Fatal(err)}
 return s,Submit{OperationID:"op1",SubjectID:"alice",OrderID:"o1",ExpectedVersion:7,Units:3}
}
func assertNoChange(t *testing.T,s *Store){
 t.Helper()
 x,_:=s.Inspect("o1")
 if x.Order.State!="pending"||x.Order.Version!=7||x.Available!=20||x.Reserved!=0||len(x.History)!=0||len(x.Notices)!=0{t.Fatalf("unexpected effects: %+v",x)}
}
func TestSubmitCommitOwnerActionsAndOutbox(t *testing.T){
 s,c:=fixture(t)
 r,err:=s.Execute(context.Background(),c,allowAll)
 if err!=nil||r.Outcome!=Submitted||!r.Committed{t.Fatalf("result %+v err %v",r,err)}
 phases:=[]Phase{PhaseValidate,PhaseAuthorize,PhaseRead,PhaseBefore,PhaseConsequence,PhaseCommit,PhasePostCommit}
 if len(phases)!=len(r.Trace){t.Fatalf("trace %+v",r.Trace)}
 for i,p:=range phases{if r.Trace[i]!=p{t.Fatalf("phase %d: %v != %v",i,r.Trace[i],p)}}
 x,_:=s.Inspect("o1")
 if x.Order.State!=Submitted||x.Order.Version!=8||x.Available!=17||x.Reserved!=3||len(x.History)!=1||len(x.Notices)!=1{t.Fatalf("invalid commit %+v",x)}
 sends:=0
 for i:=0;i<2;i++{
  ok,err:=s.SendPending(context.Background(),c.OperationID,func(_ context.Context,n Notice)error{
   if n.OperationID!="op1"{t.Errorf("unexpected notice %+v",n)}
   sends++;return nil
  })
  if err!=nil||ok!=(i==0){t.Fatalf("delivery %d ok=%v err=%v",i,ok,err)}
 }
 if sends!=1{t.Fatalf("sent %d times",sends)}
 again,err:=s.Execute(context.Background(),c,allowAll)
 if err!=nil||again.Outcome!=Submitted{t.Fatalf("idempotent retry %+v %v",again,err)}
 x,_=s.Inspect("o1")
 if x.Available!=17||len(x.History)!=1{t.Fatalf("duplicate consequence %+v",x)}
}
func TestDenialInputAndStaleOldValue(t *testing.T){
 s,c:=fixture(t)
 deny:=AuthorizeFunc(func(_ context.Context,subject,order string)(bool,error){
  if subject!="alice"||order!="o1"{t.Fatal("wrong auth scope")};return false,nil
 })
 r,err:=s.Execute(context.Background(),c,deny)
 if err!=nil||r.Outcome!=Denied||len(r.Trace)!=2{t.Fatalf("denied %+v %v",r,err)}
 assertNoChange(t,s)
 c.Units=0
 if _,err:=s.Execute(context.Background(),c,allowAll);err==nil{t.Fatal("invalid units accepted")}
 assertNoChange(t,s)
 c.Units=3;c.ExpectedVersion=6
 r,err=s.Execute(context.Background(),c,allowAll)
 if err!=nil||r.Outcome!=NotSubmitted{t.Fatalf("stale old %+v %v",r,err)}
 assertNoChange(t,s)
 c.ExpectedVersion=7;c.OrderID="unknown"
 r,err=s.Execute(context.Background(),c,allowAll)
 if err!=nil||r.Outcome!=NotSubmitted{t.Fatalf("missing row %+v %v",r,err)}
 assertNoChange(t,s)
}
func TestRollbackCancellationAndProviderOutage(t *testing.T){
 s,c:=fixture(t)
 s.failBeforeCommit=true
 if _,err:=s.Execute(context.Background(),c,allowAll);err==nil{t.Fatal("pre-commit failure lost")}
 assertNoChange(t,s)
 s.failBeforeCommit=false
 ctx,abort:=context.WithCancel(context.Background());abort()
 if _,err:=s.Execute(ctx,c,allowAll);!errors.Is(err,context.Canceled){t.Fatalf("cancel %v",err)}
 assertNoChange(t,s)
 broken:=AuthorizeFunc(func(context.Context,string,string)(bool,error){return false,errors.New("store down")})
 if _,err:=s.Execute(context.Background(),c,broken);err==nil{t.Fatal("provider outage treated as denial")}
 assertNoChange(t,s)
}
func TestLostAcknowledgementReceiptAndCollision(t *testing.T){
 s,c:=fixture(t);s.loseAcknowledgement=true
 result,err:=s.Execute(context.Background(),c,allowAll)
 if err!=nil||result.Outcome!=Unknown||result.Committed{t.Fatalf("uncertain %+v %v",result,err)}
 x,_:=s.Inspect("o1");if x.Available!=17{t.Fatal("commit unexpectedly rolled back")}
 got,found,err:=s.Lookup(context.Background(),c,allowAll)
 if err!=nil||!found||got.Outcome!=Submitted||!got.Committed{t.Fatalf("lookup %+v %v %v",got,found,err)}
 different:=c;different.Units++
 if _,err:=s.Execute(context.Background(),different,allowAll);err==nil{t.Fatal("collision accepted")}
 deny:=AuthorizeFunc(func(context.Context,string,string)(bool,error){return false,nil})
 _,found,err=s.Lookup(context.Background(),c,deny)
 if err!=nil||found{t.Fatalf("receipt exposed: %v %v",found,err)}
 again,err:=s.Execute(context.Background(),c,allowAll)
 if err!=nil||again.Outcome!=Submitted||!again.Committed{t.Fatalf("retry %+v %v",again,err)}
 x,_=s.Inspect("o1");if len(x.History)!=1{t.Fatal("double consequence")}
}
func TestConcurrentCAS(t *testing.T){
 s,c:=fixture(t)
 var wg sync.WaitGroup
 results:=make(chan string,2)
 for i:=0;i<2;i++{
  wg.Add(1)
  go func(i int){
   defer wg.Done()
   op:=c;if i==1{op.OperationID="op2"}
   r,e:=s.Execute(context.Background(),op,allowAll)
   if e!=nil{results<-"error"}else{results<-r.Outcome}
  }(i)
 }
 wg.Wait();close(results)
 good,bad:=0,0
 for outcome:=range results{if outcome==Submitted{good++}else if outcome==NotSubmitted{bad++}else{t.Fatalf("unexpected %s",outcome)}}
 if good!=1||bad!=1{t.Fatalf("committed %d conflicts %d",good,bad)}
 x,_:=s.Inspect("o1")
 if x.Available!=17||len(x.History)!=1||len(x.Notices)!=1{t.Fatalf("double effects %+v",x)}
}
func TestBusinessCancelDeletesReservationAndRollsBack(t *testing.T){
 s,c:=fixture(t)
 if _,err:=s.Execute(context.Background(),c,allowAll);err!=nil{t.Fatal(err)}
 op:=Cancel{OperationID:"op-cancel",SubjectID:"alice",OrderID:"o1",ExpectedVersion:8}
 s.failBeforeCommit=true
 if _,err:=s.ExecuteCancel(context.Background(),op,allowAll);err==nil{t.Fatal("cancel should roll back")}
 x,_:=s.Inspect("o1")
 if x.Available!=17||x.Reserved!=3||x.Order.State!=Submitted||len(x.History)!=1{t.Fatalf("rollback %+v",x)}
 s.failBeforeCommit=false
 r,err:=s.ExecuteCancel(context.Background(),op,allowAll)
 if err!=nil||r.Outcome!="cancelled"||!r.Committed{t.Fatalf("cancel %+v %v",r,err)}
 x,_=s.Inspect("o1")
 if x.Order.State!="cancelled"||x.Order.Version!=9||x.Available!=20||x.Reserved!=0||len(x.History)!=2||len(x.Notices)!=2{t.Fatalf("cancel effects %+v",x)}
 r,err=s.ExecuteCancel(context.Background(),op,allowAll)
 if err!=nil||r.Outcome!="cancelled"{t.Fatalf("cancel retry %+v %v",r,err)}
 x,_=s.Inspect("o1");if x.Available!=20||len(x.History)!=2{t.Fatalf("double cancellation %+v",x)}
 if _,err:=s.ExecuteCancel(context.Background(),Cancel{OperationID:"op1",SubjectID:"alice",OrderID:"o1",ExpectedVersion:9},allowAll);err==nil{t.Fatal("cross-action operation ID reuse")}
}
func TestSenderErrorAndReentrancy(t *testing.T){
 s,c:=fixture(t)
 if _,err:=s.Execute(context.Background(),c,allowAll);err!=nil{t.Fatal(err)}
 sent:=0
 _,err:=s.SendPending(context.Background(),c.OperationID,func(context.Context,Notice)error{sent++;return errors.New("ambiguous remote timeout")})
 if err==nil{t.Fatal("remote failure was ignored")}
 entered,release:=make(chan struct{}),make(chan struct{})
 done:=make(chan error,1)
 go func(){
  _,err:=s.SendPending(context.Background(),c.OperationID,func(context.Context,Notice)error{sent++;close(entered);<-release;return nil})
  done<-err
 }()
 <-entered
 if _,ok:=s.Inspect("o1");!ok{t.Fatal("sender callback blocks store lock")}
 if _,err:=s.SendPending(context.Background(),c.OperationID,func(context.Context,Notice)error{t.Fatal("duplicate delivery");return nil});err==nil{t.Fatal("reentrant delivery accepted")}
 close(release)
 if err:=<-done;err!=nil{t.Fatal(err)}
 didSend,err:=s.SendPending(context.Background(),c.OperationID,func(context.Context,Notice)error{t.Fatal("already delivered");return nil})
 if err!=nil||didSend||sent!=2{t.Fatalf("retry %v %v calls=%d",didSend,err,sent)}
 // A remote timeout may have sent the first notice; receiver MUST deduplicate.
}
