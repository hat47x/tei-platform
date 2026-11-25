package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	cognitionport "tei/tools/reference-cognition-port"
)

func TestSelectedChoiceThroughProposalPort(t *testing.T) {
	for _, tc := range []struct{name, answer, finish, profile, id, state string; calls int}{
		{"billing","billing","stop","connected","local-choice","proposed",1},
		{"abstain","unknown","stop","connected","local-choice","unknown",1},
		{"foreign-option","other","stop","connected","local-choice","failed",1},
		{"truncated","billing","length","connected","local-choice","failed",1},
		{"offline","billing","stop","offline","local-choice","blocked",0},
		{"wrong-provider","billing","stop","connected","other-provider","failed",0},
	}{
		t.Run(tc.name,func(t *testing.T){
			calls:=0
			server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
				calls++
				var body struct{ Model string; MaxTokens int `json:"max_tokens"`; Messages []struct{Role,Content string} }
				if r.Method!="POST" || r.URL.Path!="/v1/chat/completions" || json.NewDecoder(r.Body).Decode(&body)!=nil || body.Model!="fixture-model" || body.MaxTokens!=32 || len(body.Messages)!=2 {
					t.Errorf("unexpected HTTP request")
				}
				json.NewEncoder(w).Encode(map[string]any{"model":"fixture-model","choices":[]any{map[string]any{"finish_reason":tc.finish,"message":map[string]string{"content":tc.answer}}}})
			}))
			defer server.Close()
			request:=cognitionport.Request{Schema:cognitionport.RequestSchema,ID:"fixture-1",Kind:"choice",Profile:tc.profile,Context:cognitionport.Context{WorkspaceRevision:"fixture-v1",Observations:[]cognitionport.Observation{
				{ID:"input",Statement:"請求額が異なっています",Reference:"inquiry-001"},
				{ID:"option.billing",Statement:"請求や支払い"},
				{ID:"option.technical",Statement:"技術的な問題"},
			}}}
			provider:=cognitionport.Provider{Schema:cognitionport.ProviderSchema,ID:tc.id,Model:"fixture-model",Version:"fixture-v1",Execution:"external",Network:"required",Capabilities:[]string{"choice"},Limits:cognitionport.Limits{MaxContextBytes:8192,MaxOutputBytes:4096,TimeoutSeconds:5},Settings:map[string]any{"endpoint":server.URL+"/v1/chat/completions"}}
			write:=func(name string,value any) string {
				t.Helper()
				raw,err:=json.Marshal(value)
				if err!=nil { t.Fatal(err) }
				path:=filepath.Join(t.TempDir(),name)
				if err:=os.WriteFile(path,raw,0600); err!=nil { t.Fatal(err) }
				return path
			}
			receipt,err:=cognitionport.ProposeWithProvider(context.Background(),write("request.json",request),write("provider.json",provider),selectedChoice)
			if err!=nil || receipt.State!=tc.state || calls!=tc.calls || receipt.Accepted || receipt.Applied || receipt.GrantsAuthority {
				t.Fatalf("receipt=%+v calls=%d err=%v",receipt,calls,err)
			}
			if tc.state=="proposed" {
				if receipt.Proposal==nil || len(receipt.Proposal.Candidates)!=1 { t.Fatal("candidate missing") }
				c:=receipt.Proposal.Candidates[0]
				if c.Subject!="inquiry-001" || c.Payload["choice"]!="billing" || len(c.Basis)!=2 || c.Basis[0]!="observation:input" || c.Basis[1]!="observation:option.billing" || receipt.OutputDigest=="" { t.Fatalf("invalid candidate: %+v",c) }
			} else if receipt.Proposal!=nil && len(receipt.Proposal.Candidates)!=0 {
				t.Fatal("non-proposed result contains candidate")
			}
		})
	}
}
