package interfaceadapter

import (
	"context"
	"net"
	"net/http/httptest"
	"reflect"
	"testing"

	"tei/internal/application"
	"tei/internal/command"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func testWorld() ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{{
			ID: "sales",
			Data: []semantic.Data{{
				ID: "sales.order",
				Attributes: []semantic.Attribute{
					{ID: "sales.order.order_id", TypeRef: "std.identifier"},
				},
			}},
			Capabilities: []semantic.Capability{{
				ID:       "sales.submit",
				Symbol:   "submit",
				Inputs:   []semantic.Parameter{{Name: "order", Type: "std.identifier"}},
				Outcomes: []semantic.Outcome{{Name: "submitted", Payload: "sales.order"}},
				Effect:   &semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: "sales.order"},
			}},
		}},
		Applications: []application.Application{{
			ID:           "order_entry",
			SemanticUses: []core.CanonicalID{"sales"},
			Patterns: []application.Pattern{{
				ID:   "order_entry.order_record",
				Kind: application.PatternRecord,
				Data: "sales.order",
				Interaction: &application.Interaction{
					Items: []application.InteractionItem{{Attribute: "sales.order.order_id"}},
					Commands: []application.InteractionCommand{{
						Capability: "sales.submit",
						Inputs:     []application.CommandInputBinding{{Name: "order", Attribute: "sales.order.order_id"}},
					}},
				},
			}},
		}},
	}
}

const testRevision = "rev-1"

func testRequest(revision string) Request {
	return Request{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
		Capability:  "sales.submit",
		Inputs:      map[string]string{"order": "ORD-1"},
		Revision:    revision,
	}
}

func testCore() Core {
	return NewCore(testWorld(), testRevision)
}

func newTestHTTPServer(t *testing.T) (*HTTPServer, *httptest.Server) {
	t.Helper()
	server, err := NewHTTPServer(HTTPOptions{
		Core:       testCore(),
		Path:       "/commands/invoke",
		ActionID:   "sales.submit",
		ProviderID: "fixture",
		Issuer:     "fixture:issuer",
		Tokens: map[string]HTTPIdentity{
			"editor-token": {Subject: "user-1", Role: "editor"},
			"viewer-token": {Subject: "user-2", Role: "viewer"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)
	return server, httpServer
}

func newIPC(t *testing.T, principal Principal) (IPCAdapter, *IPCServer) {
	t.Helper()
	server := &IPCServer{Core: testCore(), Principal: principal}
	clientConn, serverConn := net.Pipe()
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = server.ServeConn(serverConn)
	}()
	t.Cleanup(func() {
		_ = clientConn.Close()
		<-done
	})
	return IPCAdapter{Conn: clientConn}, server
}

// #1158 condition 1 across the chosen transports: the same typed request and
// the same verified principal produce the same business result.
func TestAdaptersAgreeOnTheSameOperation(t *testing.T) {
	ctx := context.Background()
	principal := Principal{Subject: "user-1", Role: "editor"}
	request := testRequest(testRevision)

	inProcess := InProcessAdapter{Core: testCore()}.Execute(ctx, principal, request)

	_, httpServer := newTestHTTPServer(t)
	web := HTTPAdapter{URL: httpServer.URL + "/commands/invoke", Token: "editor-token", CSRF: "matching"}
	overHTTP, err := web.Execute(ctx, request)
	if err != nil {
		t.Fatal(err)
	}

	ipc, _ := newIPC(t, principal)
	overIPC, err := ipc.Execute(request)
	if err != nil {
		t.Fatal(err)
	}

	want := Result{Outcome: "submitted", Revision: testRevision}
	for name, got := range map[string]Result{"in-process": inProcess, "http": overHTTP, "ipc": overIPC} {
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("%s result=%+v, want %+v", name, got, want)
		}
	}
}

// #1158 condition 2: the Web adapter rejects unauthenticated, unauthorized and
// CSRF-less requests before the operation is invoked.
func TestHTTPRejectsBeforeInvoke(t *testing.T) {
	cases := []struct {
		name  string
		token string
		csrf  string
		code  string
	}{
		{name: "unauthenticated", csrf: "matching", code: "UNAUTHENTICATED"},
		{name: "unauthorized role", token: "viewer-token", csrf: "matching", code: "FORBIDDEN"},
		{name: "csrf missing", token: "editor-token", csrf: "wrong", code: "FORBIDDEN"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			server, httpServer := newTestHTTPServer(t)
			adapter := HTTPAdapter{
				URL: httpServer.URL + "/commands/invoke", Token: testCase.token, CSRF: testCase.csrf,
			}
			result, err := adapter.Execute(context.Background(), testRequest(testRevision))
			if err != nil {
				t.Fatal(err)
			}
			if result.Failure == nil || result.Failure.Code != testCase.code {
				t.Fatalf("result=%+v, want failure %s", result, testCase.code)
			}
			if server.Invocations() != 0 {
				t.Fatalf("the operation ran %d time(s) despite the rejection", server.Invocations())
			}
		})
	}
}

func TestHTTPRejectsStaleRevision(t *testing.T) {
	server, httpServer := newTestHTTPServer(t)
	adapter := HTTPAdapter{URL: httpServer.URL + "/commands/invoke", Token: "editor-token", CSRF: "matching"}
	result, err := adapter.Execute(context.Background(), testRequest("rev-0"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Failure == nil || result.Failure.Code != "STALE_REVISION" {
		t.Fatalf("result=%+v, want STALE_REVISION", result)
	}
	if server.Invocations() != 1 {
		t.Fatalf("invocations=%d, want 1 (the revision check is inside the operation)", server.Invocations())
	}
}

func TestIPCRejectsMissingPrincipal(t *testing.T) {
	ipc, server := newIPC(t, Principal{})
	result, err := ipc.Execute(testRequest(testRevision))
	if err != nil {
		t.Fatal(err)
	}
	if result.Failure == nil || result.Failure.Code != "UNAUTHENTICATED" {
		t.Fatalf("result=%+v, want UNAUTHENTICATED", result)
	}
	if server.Invocations() != 1 {
		t.Fatalf("invocations=%d, want 1", server.Invocations())
	}
}

func TestIPCRejectsStaleRevision(t *testing.T) {
	ipc, _ := newIPC(t, Principal{Subject: "user-1", Role: "editor"})
	result, err := ipc.Execute(testRequest("rev-0"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Failure == nil || result.Failure.Code != "STALE_REVISION" {
		t.Fatalf("result=%+v, want STALE_REVISION", result)
	}
}

func TestInProcessRejectsMissingPrincipal(t *testing.T) {
	result := InProcessAdapter{Core: testCore()}.Execute(context.Background(), Principal{}, testRequest(testRevision))
	if result.Failure == nil || result.Failure.Code != "UNAUTHENTICATED" {
		t.Fatalf("result=%+v, want UNAUTHENTICATED", result)
	}
}

// Condition 4 support: the in-process adapter uses the same Command boundary as
// the CLI's reference execution path. The CLI process itself is covered by
// cmd/tei's cli_inprocess_parity_test.go; this harness does not run the binary.
func TestInProcessMatchesTheCLIPath(t *testing.T) {
	world := testWorld()
	invoker := command.Invoker{
		Authorizer: command.ReferenceAuthorizer{},
		World:      world,
		Executor:   command.ReferenceExecutor{World: world},
		UnitOfWork: &command.MemoryUnitOfWork{},
	}
	result, err := invoker.Invoke(context.Background(), command.Request{
		Application: "order_entry",
		Pattern:     "order_entry.order_record",
		Capability:  "sales.submit",
		Inputs:      map[string]string{"order": "ORD-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	adapted := InProcessAdapter{Core: NewCore(world, testRevision)}.Execute(
		context.Background(), Principal{Subject: "user-1", Role: "editor"}, testRequest(testRevision))
	if result.Outcome == nil || adapted.Outcome != string(result.Outcome.Name) {
		t.Fatalf("CLI path outcome=%v, adapter=%+v", result.Outcome, adapted)
	}
}
