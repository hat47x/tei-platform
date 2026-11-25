package localchoice

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLoopbackChoice(t *testing.T) {
	options:=[]Option{{"billing","請求や支払い"},{"technical","技術的な問題"}}
	for _,tc:=range []struct{name,body,want string}{
		{"valid",`{"model":"test-model","choices":[{"finish_reason":"stop","message":{"content":"billing"}}]}`,"billing"},
		{"unknown",`{"model":"test-model","choices":[{"finish_reason":"stop","message":{"content":"unknown"}}]}`,"unknown"},
		{"unexpected",`{"model":"test-model","choices":[{"finish_reason":"stop","message":{"content":"contract"}}]}`,""},
		{"wrong-model",`{"model":"other","choices":[{"finish_reason":"stop","message":{"content":"billing"}}]}`,""},
		{"incomplete",`{"model":"test-model","choices":[{"finish_reason":"length","message":{"content":"billing"}}]}`,""},
		{"tool-call",`{"model":"test-model","choices":[{"finish_reason":"stop","message":{"content":"billing","tool_calls":[{}]}}]}`,""},
	}{
		t.Run(tc.name,func(t *testing.T){
			s:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){w.Write([]byte(tc.body))}))
			defer s.Close()
			ctx,cancel:=context.WithTimeout(context.Background(),2*time.Second)
			defer cancel()
			got,err:=Choose(ctx,s.URL+"/v1/chat/completions","test-model","請求が違います",options)
			if got!=tc.want || (err==nil)!=(tc.want!="") {t.Fatalf("got=%q err=%v",got,err)}
		})
	}
	if _,err:=Choose(context.Background(),"http://192.0.2.1:8080/v1/chat/completions","test-model","input",options);err==nil {t.Fatal("non-loopback accepted")}
}

func TestNoRedirect(t *testing.T) {
	calls:=0
	target:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){calls++}))
	defer target.Close()
	s:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){http.Redirect(w,r,target.URL,http.StatusTemporaryRedirect)}))
	defer s.Close()
	ctx,cancel:=context.WithTimeout(context.Background(),time.Second)
	defer cancel()
	if _,err:=Choose(ctx,s.URL+"/v1/chat/completions","test-model","input",[]Option{{"a","A"},{"b","B"}});err==nil || calls!=0 {t.Fatalf("redirect followed: %v calls=%d",err,calls)}
}
