package submit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"tei/internal/security/effectcommit"
	"tei/internal/security/legacyauthorization"
)

// These tests are DB-free and synthetic. They prove the mapping and request
// building of this package only.

func grant() legacyauthorization.AuthorizationGrant {
	return legacyauthorization.AuthorizationGrant{
		SubjectID: "subj", Action: "execute", Resource: "cap.res",
		Epochs: &legacyauthorization.AuthorityEpochs{Source: "src", SubjectLink: 1, Membership: 1, Policy: 1},
	}
}

var spec = Spec{Application: "app", Capability: "cap.res"}

type fakeRunner struct {
	execResult effectcommit.Result
	execErr    error
	lookResult effectcommit.Result
	lookErr    error
	executes   int
	lookups    int
	lastReq    effectcommit.Request
}

func (f *fakeRunner) Execute(_ context.Context, r effectcommit.Request, _ legacyauthorization.AuthorizationGrant) (effectcommit.Result, error) {
	f.executes++
	f.lastReq = r
	return f.execResult, f.execErr
}
func (f *fakeRunner) Lookup(_ context.Context, r effectcommit.Request, _ legacyauthorization.AuthorizationGrant) (effectcommit.Result, error) {
	f.lookups++
	f.lastReq = r
	return f.lookResult, f.lookErr
}

func TestClassifyTable(t *testing.T) {
	const secret = "postgres://u:p@h/db driver detail"
	unc := &effectcommit.CommitUncertainError{Cause: errors.New(secret)}
	for _, tc := range []struct {
		name   string
		result effectcommit.Result
		err    error
		want   Disposition
	}{
		{"committed", effectcommit.Result{Status: effectcommit.StatusCommitted}, nil, Committed},
		{"replayed", effectcommit.Result{Status: effectcommit.StatusReplayed}, nil, Replayed},
		{"denied", effectcommit.Result{Status: effectcommit.StatusDenied}, nil, NotCommitted},
		{"conflict", effectcommit.Result{Status: effectcommit.StatusConflict}, nil, NotCommitted},
		{"not-recorded-from-execute", effectcommit.Result{Status: effectcommit.StatusNotRecorded}, nil, Failed},
		{"unknown-status", effectcommit.Result{Status: "weird"}, nil, Failed},
		{"empty-status", effectcommit.Result{}, nil, Failed},
		{"uncertain", effectcommit.Result{}, unc, Uncertain},
		{"uncertain-wrapped", effectcommit.Result{}, fmt.Errorf("outer: %w", unc), Uncertain},
		{"uncertain-joined", effectcommit.Result{}, errors.Join(errors.New("x"), unc), Uncertain},
		{"plain-error", effectcommit.Result{Status: effectcommit.StatusCommitted}, errors.New(secret), Failed},
		{"invalid-request", effectcommit.Result{}, effectcommit.ErrInvalidRequest, Failed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := Classify(tc.result, tc.err)
			if out.Disposition != tc.want {
				t.Fatalf("got %q want %q", out.Disposition, tc.want)
			}
			if s := fmt.Sprintf("%+v", out); strings.Contains(s, "postgres://") || strings.Contains(s, "driver") {
				t.Fatalf("outcome leaks error text: %s", s)
			}
		})
	}
}

func TestReplayedIsNotCollapsedAndStatusIsKept(t *testing.T) {
	r := Classify(effectcommit.Result{Status: effectcommit.StatusReplayed, Receipt: effectcommit.Receipt{OperationID: "op"}}, nil)
	n := Classify(effectcommit.Result{Status: effectcommit.StatusDenied}, nil)
	if r.Disposition == NotCommitted || r.Disposition != Replayed || r.OperationID != "op" {
		t.Fatalf("replayed=%+v", r)
	}
	if n.Disposition != NotCommitted || n.Status != effectcommit.StatusDenied {
		t.Fatalf("denied=%+v", n)
	}
	c := Classify(effectcommit.Result{Status: effectcommit.StatusConflict}, nil)
	if c.Status != effectcommit.StatusConflict || c.Status == n.Status {
		t.Fatalf("conflict=%+v", c)
	}
}

func TestClassifyLookupTable(t *testing.T) {
	unc := &effectcommit.CommitUncertainError{Cause: errors.New("x")}
	for _, tc := range []struct {
		name   string
		result effectcommit.Result
		err    error
		want   Disposition
	}{
		{"replayed", effectcommit.Result{Status: effectcommit.StatusReplayed, Receipt: effectcommit.Receipt{OperationID: "op"}}, nil, Replayed},
		{"not-recorded", effectcommit.Result{Status: effectcommit.StatusNotRecorded}, nil, NotRecorded},
		{"denied-after-revoke", effectcommit.Result{Status: effectcommit.StatusDenied}, nil, NotCommitted},
		{"conflict", effectcommit.Result{Status: effectcommit.StatusConflict}, nil, NotCommitted},
		{"committed-from-lookup", effectcommit.Result{Status: effectcommit.StatusCommitted}, nil, Failed},
		{"raw-commit-error", effectcommit.Result{}, errors.New("commit failed"), Failed},
		{"uncertain-type-from-lookup", effectcommit.Result{}, unc, Failed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClassifyLookup(tc.result, tc.err).Disposition; got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
	if d := ClassifyLookup(effectcommit.Result{Status: effectcommit.StatusDenied}, nil); d.Disposition == Replayed {
		t.Fatal("denied must not read as a stale replayed")
	}
}

func TestRequestFromGrantTakesScopeOnlyFromGrant(t *testing.T) {
	g := grant()
	req, err := RequestFromGrant(spec, g, Intent{OperationID: "op-1", ExpectedVersion: 3, Payload: []byte(`{"a": 1}`)})
	if err != nil {
		t.Fatal(err)
	}
	want := effectcommit.Scope{Application: "app", Capability: "cap.res", Subject: "subj", Action: "execute", Resource: "cap.res"}
	if req.Scope != want || req.OperationID != "op-1" || req.ExpectedVersion != 3 {
		t.Fatalf("req=%+v", req)
	}
}

func TestRequestFromGrantRefusesEmptyIDs(t *testing.T) {
	for name, tc := range map[string]struct {
		s  Spec
		in Intent
	}{
		"application": {Spec{Capability: "c"}, Intent{OperationID: "o"}},
		"capability":  {Spec{Application: "a"}, Intent{OperationID: "o"}},
		"operation":   {Spec{Application: "a", Capability: "c"}, Intent{}},
	} {
		t.Run(name, func(t *testing.T) {
			fr := &fakeRunner{}
			c := Caller{Runner: fr, Spec: tc.s}
			if _, err := c.Execute(context.Background(), grant(), tc.in); !errors.Is(err, ErrInvalidIntent) {
				t.Fatalf("execute err=%v", err)
			}
			if _, err := c.Lookup(context.Background(), grant(), tc.in); !errors.Is(err, ErrInvalidIntent) {
				t.Fatalf("lookup err=%v", err)
			}
			if fr.executes != 0 || fr.lookups != 0 {
				t.Fatal("runner called")
			}
		})
	}
}

func TestPayloadPassesThroughUnmodifiedAndHashIsStable(t *testing.T) {
	// Deliberately non-canonical JSON: a re-marshal would change the bytes.
	payload := []byte("{ \"b\":2,  \"a\":1 }")
	orig := append([]byte(nil), payload...)
	fr := &fakeRunner{execResult: effectcommit.Result{Status: effectcommit.StatusCommitted}}
	c := Caller{Runner: fr, Spec: spec}
	in := Intent{OperationID: "op", Payload: payload}
	if _, err := c.Execute(context.Background(), grant(), in); err != nil {
		t.Fatal(err)
	}
	first := fr.lastReq
	if !bytes.Equal(first.Payload, orig) {
		t.Fatalf("payload changed: %q", first.Payload)
	}
	// Mutating the caller slice afterwards must not alter the built request.
	payload[0] = 'X'
	if !bytes.Equal(first.Payload, orig) {
		t.Fatal("request aliases the caller payload")
	}
	payload[0] = '{'
	if _, err := c.Execute(context.Background(), grant(), in); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(fr.lastReq.Payload, first.Payload) || fr.lastReq.Scope != first.Scope {
		t.Fatal("retry builds a different request")
	}
	if _, err := c.Lookup(context.Background(), grant(), in); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(fr.lastReq.Payload, first.Payload) || fr.lastReq.OperationID != first.OperationID {
		t.Fatal("lookup builds a different request than execute")
	}
}

func TestGrantSourceFailsClosed(t *testing.T) {
	const secret = "postgres://leak driver"
	fr := &fakeRunner{}
	for name, src := range map[string]GrantSource{
		"nil": nil,
		"error": func(context.Context) (legacyauthorization.AuthorizationGrant, error) {
			return legacyauthorization.AuthorizationGrant{}, errors.New(secret)
		},
	} {
		t.Run(name, func(t *testing.T) {
			c := Caller{Runner: fr, Grants: src, Spec: spec}
			g, err := c.Grant(context.Background())
			if !errors.Is(err, ErrGrantUnavailable) || strings.Contains(err.Error(), "postgres") || g.Epochs != nil || g.SubjectID != "" {
				t.Fatalf("g=%+v err=%v", g, err)
			}
			if fr.executes != 0 {
				t.Fatal("execute reached")
			}
		})
	}
}

// countingProvider/Tx drive the real effectcommit.Runner.
type countingProvider struct {
	begins int
	tx     *countingTx
}
type countingTx struct{ checks int }

func (p *countingProvider) Begin(context.Context) (effectcommit.Transaction, error) {
	p.begins++
	return p.tx, nil
}
func (t *countingTx) CheckAuthority(context.Context, effectcommit.Scope, legacyauthorization.AuthorityEpochs) (bool, error) {
	t.checks++
	return true, nil
}
func (t *countingTx) FindReceipt(context.Context, string) (effectcommit.Receipt, bool, error) {
	return effectcommit.Receipt{}, false, nil
}
func (t *countingTx) WriteResource(context.Context, effectcommit.Request) (uint64, error) {
	return 1, nil
}
func (t *countingTx) SaveReceipt(context.Context, effectcommit.Receipt) error { return nil }
func (t *countingTx) Commit(context.Context) error                            { return nil }
func (t *countingTx) Rollback() error                                         { return nil }

func TestRealRunnerRefusesScopeDisagreeingWithGrantBeforeProvider(t *testing.T) {
	p := &countingProvider{tx: &countingTx{}}
	runner := effectcommit.Runner{Provider: p}
	req, err := RequestFromGrant(spec, grant(), Intent{OperationID: "op"})
	if err != nil {
		t.Fatal(err)
	}
	req.Scope.Subject = "someone-else" // hand-forged scope that the builder never produces
	res, err := runner.Execute(context.Background(), req, grant())
	if !errors.Is(err, effectcommit.ErrInvalidRequest) || p.begins != 0 || p.tx.checks != 0 {
		t.Fatalf("res=%+v err=%v begins=%d checks=%d", res, err, p.begins, p.tx.checks)
	}
	if out := Classify(res, err); out.Disposition != Failed {
		t.Fatalf("out=%+v", out)
	}
	if _, err := runner.Lookup(context.Background(), req, grant()); !errors.Is(err, effectcommit.ErrInvalidRequest) || p.begins != 0 {
		t.Fatalf("lookup err=%v begins=%d", err, p.begins)
	}
}

func TestNilEpochsGrantNeverChecksAuthority(t *testing.T) {
	p := &countingProvider{tx: &countingTx{}}
	g := grant()
	g.Epochs = nil
	c := Caller{Runner: effectcommit.Runner{Provider: p}, Spec: spec}
	out, err := c.Execute(context.Background(), g, Intent{OperationID: "op"})
	if err != nil || out.Disposition != Failed || p.tx.checks != 0 || p.begins != 0 {
		t.Fatalf("out=%+v err=%v begins=%d checks=%d", out, err, p.begins, p.tx.checks)
	}
}

func TestRealRunnerHappyPathThroughCaller(t *testing.T) {
	p := &countingProvider{tx: &countingTx{}}
	c := Caller{Runner: effectcommit.Runner{Provider: p}, Spec: spec,
		Grants: func(context.Context) (legacyauthorization.AuthorizationGrant, error) { return grant(), nil }}
	g, err := c.Grant(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.Execute(context.Background(), g, Intent{OperationID: "op", Payload: []byte("{}")})
	if err != nil || out.Disposition != Committed || out.OperationID != "op" || p.tx.checks != 1 {
		t.Fatalf("out=%+v err=%v", out, err)
	}
}

// TestImportGuard keeps this package a leaf: stdlib, effectcommit and
// legacyauthorization only, so it cannot pull pgx, the IR, or any target.
func TestImportGuard(t *testing.T) {
	allowed := map[string]bool{
		"tei/internal/security/effectcommit":        true,
		"tei/internal/security/legacyauthorization": true,
	}
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("glob: %v %v", files, err)
	}
	checked := 0
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		checked++
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, src, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range f.Imports {
			path, _ := strconv.Unquote(imp.Path.Value)
			if allowed[path] {
				continue
			}
			if first := strings.SplitN(path, "/", 2)[0]; strings.Contains(first, ".") || strings.HasPrefix(path, "tei/") {
				t.Errorf("%s imports %q, which is outside stdlib/effectcommit/legacyauthorization", name, path)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no non-test file checked")
	}
}

func TestValidateIntentUnsetKeepsBehaviour(t *testing.T) {
	r := &fakeRunner{execResult: effectcommit.Result{Status: effectcommit.StatusCommitted}}
	out, err := Caller{Runner: r, Spec: spec}.Execute(context.Background(), grant(), Intent{OperationID: "op/1", Payload: []byte("x")})
	if err != nil || out.Disposition != Committed || r.executes != 1 {
		t.Fatalf("out=%+v err=%v executes=%d", out, err, r.executes)
	}
}

func TestValidateIntentRejectionNeverReachesRunnerAndLeaksNothing(t *testing.T) {
	const secret = "secret-payload-body"
	var seen Intent
	sp := spec
	sp.ValidateIntent = func(in Intent) error {
		seen = in
		in.Payload[0] = 'X' // a hook must not be able to alter the real payload
		return errors.New("bad " + string(in.Payload))
	}
	r := &fakeRunner{}
	in := Intent{OperationID: "op/1", Payload: []byte(secret)}
	c := Caller{Runner: r, Spec: sp}

	out, err := c.Execute(context.Background(), grant(), in)
	if !errors.Is(err, ErrIntentRejected) || out.Disposition != NotCommitted || out.OperationID != "op/1" {
		t.Fatalf("execute out=%+v err=%v", out, err)
	}
	out, err = c.Lookup(context.Background(), grant(), in)
	if !errors.Is(err, ErrIntentRejected) || out.Disposition != NotCommitted {
		t.Fatalf("lookup out=%+v err=%v", out, err)
	}
	if r.executes != 0 || r.lookups != 0 {
		t.Fatalf("Runner reached: %d/%d", r.executes, r.lookups)
	}
	if string(in.Payload) != secret || string(seen.Payload) == "" {
		t.Fatal("caller payload was altered")
	}
	for _, s := range []string{err.Error(), fmt.Sprintf("%+v", out)} {
		if strings.Contains(s, "secret") || strings.Contains(s, "bad") {
			t.Fatalf("leak: %s", s)
		}
	}
	if _, err := RequestFromGrant(sp, grant(), in); !errors.Is(err, ErrIntentRejected) {
		t.Fatalf("RequestFromGrant err=%v", err)
	}
}

func TestValidateIntentAcceptingHookRunsRunner(t *testing.T) {
	sp := spec
	sp.ValidateIntent = func(Intent) error { return nil }
	r := &fakeRunner{execResult: effectcommit.Result{Status: effectcommit.StatusDenied}}
	out, err := Caller{Runner: r, Spec: sp}.Execute(context.Background(), grant(), Intent{OperationID: "op/1"})
	if err != nil || out.Disposition != NotCommitted || out.Status != effectcommit.StatusDenied || r.executes != 1 {
		t.Fatalf("out=%+v err=%v executes=%d", out, err, r.executes)
	}
}

func TestEmptyIDsStillWinOverHook(t *testing.T) {
	sp := spec
	sp.ValidateIntent = func(Intent) error { return nil }
	if _, err := RequestFromGrant(sp, grant(), Intent{}); !errors.Is(err, ErrInvalidIntent) {
		t.Fatalf("err=%v", err)
	}
}

func TestValidatedBytesAreTheExecutedBytes(t *testing.T) {
	sp := spec
	sp.ValidateIntent = func(in Intent) error {
		in.Payload[0] = 'V' // the hook's view is what must execute
		return nil
	}
	r := &fakeRunner{execResult: effectcommit.Result{Status: effectcommit.StatusCommitted}}
	orig := []byte("abc")
	if _, err := (Caller{Runner: r, Spec: sp}).Execute(context.Background(), grant(), Intent{OperationID: "op/1", Payload: orig}); err != nil {
		t.Fatal(err)
	}
	if string(r.lastReq.Payload) != "Vbc" {
		t.Fatalf("executed %q, checked bytes were %q", r.lastReq.Payload, "Vbc")
	}
	if string(orig) != "abc" {
		t.Fatalf("caller slice altered: %q", orig)
	}
}

func TestValidateIntentPanicIsRejection(t *testing.T) {
	sp := spec
	sp.ValidateIntent = func(in Intent) error { panic("boom " + string(in.Payload)) }
	r := &fakeRunner{}
	c := Caller{Runner: r, Spec: sp}
	in := Intent{OperationID: "op/1", Payload: []byte("secret-payload")}
	out, err := c.Execute(context.Background(), grant(), in)
	if !errors.Is(err, ErrIntentRejected) || out.Disposition != NotCommitted {
		t.Fatalf("out=%+v err=%v", out, err)
	}
	if _, err := c.Lookup(context.Background(), grant(), in); !errors.Is(err, ErrIntentRejected) {
		t.Fatalf("lookup err=%v", err)
	}
	if r.executes != 0 || r.lookups != 0 {
		t.Fatalf("Runner reached")
	}
	if s := fmt.Sprintf("%v %+v", err, out); strings.Contains(s, "boom") || strings.Contains(s, "secret") {
		t.Fatalf("leak: %s", s)
	}
}
