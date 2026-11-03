//go:build !tei_repair_only

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"tei/internal/security/effectcommit"
	effectpostgres "tei/internal/security/effectcommit/postgres"
	"tei/internal/security/effectcommit/submit"
	"tei/internal/security/legacyauthorization"
)

// AT2 of issue #1174: result type, DB state and audit (Evidence) per failure
// cell, through both consumers of the one effectcommit.Runner (the Web store
// and the test-only CLI driver). Real PostgreSQL, TEST-ONLY DDL, SYNTHETIC
// grant; gated by TEI_TEST_POSTGRES_DSN (a skip is not evidence).
//
// Each cell is run once per consumer on a FRESH fixture. For every cell the
// test asserts: the neutral Disposition the Runner answer classifies to, the
// user-facing label of that consumer, the DB end state (order rows, receipt
// count, receipt version sum), exactly one Evidence with the expected
// Status/ErrorClass (zero when the Runner is never reached), and that no
// driver/secret text reaches the returned error, the CLI output or Evidence.
//
// Not claimed: this is a reference result on a disposable PostgreSQL, not
// proof of production atomicity; the grant is not a security.authorize result;
// the Disposition set is an unapproved A2 proposal.

const atSecret = "postgres://svc:at2-secret-pw@internal-db:5432/x"

var atLeakMarkers = []string{"postgres://", "secret", "at2-secret-pw", "internal-db", "pgx", "SQLSTATE", "tei_test_only", "closed pool", "refused"}

func atAssertNoLeak(t *testing.T, what, text string) {
	t.Helper()
	low := strings.ToLower(text)
	for _, m := range atLeakMarkers {
		if strings.Contains(low, strings.ToLower(m)) {
			t.Fatalf("%s leaks %q: %q", what, m, text)
		}
	}
}

type atSink struct{ got []effectcommit.Evidence }

func (s *atSink) Record(e effectcommit.Evidence) { s.got = append(s.got, e) }

// atRecRunner records the Runner answer so the neutral Disposition can be
// asserted even for the Web store, whose own mapping collapses Replayed.
type atRecRunner struct {
	inner  effectcommit.Runner
	calls  int
	result effectcommit.Result
	err    error
}

func (r *atRecRunner) Execute(ctx context.Context, q effectcommit.Request, g legacyauthorization.AuthorizationGrant) (effectcommit.Result, error) {
	r.calls++
	r.result, r.err = r.inner.Execute(ctx, q, g)
	return r.result, r.err
}
func (r *atRecRunner) Lookup(ctx context.Context, q effectcommit.Request, g legacyauthorization.AuthorizationGrant) (effectcommit.Result, error) {
	return r.inner.Lookup(ctx, q, g)
}

// atFaultProvider injects Execution Errors at Begin or at CheckAuthority.
type atFaultProvider struct {
	inner    effectcommit.Provider
	beginErr error
	checkErr error
}

type atFaultTx struct {
	effectcommit.Transaction
	checkErr error
}

func (p atFaultProvider) Begin(ctx context.Context) (effectcommit.Transaction, error) {
	if p.beginErr != nil {
		return nil, p.beginErr
	}
	tx, err := p.inner.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return atFaultTx{tx, p.checkErr}, nil
}

func (t atFaultTx) CheckAuthority(ctx context.Context, s effectcommit.Scope, e legacyauthorization.AuthorityEpochs) (bool, error) {
	if t.checkErr != nil {
		return false, t.checkErr
	}
	return t.Transaction.CheckAuthority(ctx, s, e)
}

type atRun struct {
	label    string
	text     string // returned error text (web) or CLI output (cli)
	rec      *atRecRunner
	evidence []effectcommit.Evidence
}

// Distinct epoch values so a Provider that permutes the epoch arguments cannot
// pass by coincidence (every other fixture seeds 1,1,1).
const (
	atEpochSubjectLink = 3
	atEpochMembership  = 5
	atEpochPolicy      = 7
)

const atSeedEpochsSQL = `UPDATE ` + runnerTestSchema + `.authority SET subject_link=3, membership=5, policy=7`

func atBaseGrant(kind string) legacyauthorization.AuthorizationGrant {
	g := syntheticSubmitGrant()
	if kind == "cli" {
		g = cliSyntheticGrant()
	}
	src := ""
	if g.Epochs != nil {
		src = g.Epochs.Source
	}
	g.Epochs = &legacyauthorization.AuthorityEpochs{Source: src,
		SubjectLink: atEpochSubjectLink, Membership: atEpochMembership, Policy: atEpochPolicy}
	return g
}

// atSubmit runs one submit for order id through the named consumer, with the
// given Provider and grant source, and returns what was observed.
func atSubmit(t *testing.T, env *runnerPGEnv, kind string, prov effectcommit.Provider,
	grants func(context.Context) (legacyauthorization.AuthorizationGrant, error), id int64) atRun {
	t.Helper()
	sink := &atSink{}
	rec := &atRecRunner{inner: effectcommit.Runner{Provider: prov,
		Audit: effectcommit.AuditConfig{Sink: sink, Provenance: effectcommit.ProvenanceRealPG}}}
	run := atRun{rec: rec}
	switch kind {
	case "web":
		store, err := newRunnerOrderSubmitStore(rec,
			func(c context.Context, _ int64) (legacyauthorization.AuthorizationGrant, error) { return grants(c) },
			nil, &statusReader{})
		if err != nil {
			t.Fatal(err)
		}
		switch err := store.Submit(context.Background(), id); {
		case err == nil:
			run.label = "committed"
		case errors.Is(err, errOrderNotSubmittable):
			run.label, run.text = "refused", err.Error()
		case errors.Is(err, errOrderSubmitCommitUnconfirmed):
			run.label, run.text = "unconfirmed", err.Error()
		default:
			run.label, run.text = "failed", err.Error()
		}
	case "cli":
		if _, err := env.db.Exec(cliAuthoritySQL); err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		code := cliOrderRun(context.Background(), submit.Caller{Runner: rec, Grants: grants, Spec: cliSpec},
			[]string{"submit-order", itoa(id)}, &out)
		run.text = out.String()
		switch {
		case code == cliExitOK && strings.HasPrefix(run.text, "submitted order"):
			run.label = "committed"
		case code == cliExitOK && strings.HasPrefix(run.text, "already submitted order"):
			run.label = "already"
		case code == cliExitNotCommitted:
			run.label = "refused"
		case code == cliExitUncertain:
			run.label = "unconfirmed"
		default:
			run.label = "failed"
		}
	default:
		t.Fatalf("unknown consumer %q", kind)
	}
	run.evidence = sink.got
	return run
}

func atGrants(kind string, mutate func(*legacyauthorization.AuthorizationGrant)) func(context.Context) (legacyauthorization.AuthorizationGrant, error) {
	return func(context.Context) (legacyauthorization.AuthorizationGrant, error) {
		g := atBaseGrant(kind)
		if mutate != nil {
			mutate(&g)
		}
		return g, nil
	}
}

func atClosedPoolProvider(t *testing.T, env *runnerPGEnv) effectcommit.Provider {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), env.dsn)
	if err != nil {
		t.Fatal(err)
	}
	p, err := effectpostgres.NewWithTrustedOwnerRoutine(pool, runnerTestSchema)
	if err != nil {
		pool.Close()
		t.Fatal(err)
	}
	pool.Close() // every later Begin fails: a broken Store
	return p
}

type atCell struct {
	name string
	// setup: DB mutation, run after the fixture exists and before the call.
	sql string
	// prep: run before the call (e.g. an earlier clean commit); may be nil.
	prep func(t *testing.T, env *runnerPGEnv, kind string)
	// mutate: change the grant the consumer hands to the Runner.
	mutate func(*legacyauthorization.AuthorizationGrant)
	// provider wraps the real Provider (fault injection); nil = real Provider.
	provider func(t *testing.T, env *runnerPGEnv, real effectcommit.Provider) effectcommit.Provider
	// grantErr: the grant source itself fails (Runner is never reached).
	grantErr bool

	wantDisp   submit.Disposition // "" = Runner never reached
	wantStatus effectcommit.Status
	wantClass  effectcommit.ErrorClass
	web, cli   string
	want       pgSnapshot
}

var atDraft = pgSnapshot{"draft", "draft", 0, 0}
var atSubmitted = pgSnapshot{"submitted", "draft", 1, 1}

func atCells() []atCell {
	epoch := func(col string) string {
		return `UPDATE ` + runnerTestSchema + `.authority SET ` + col + `=` + col + `+1`
	}
	return []atCell{
		{name: "allowed", wantDisp: submit.Committed, wantStatus: effectcommit.StatusCommitted,
			web: "committed", cli: "committed", want: atSubmitted},
		{name: "replayed-second-submit",
			prep: func(t *testing.T, env *runnerPGEnv, kind string) {
				first := atSubmit(t, env, kind, env.provider(t), atGrants(kind, nil), 1001)
				if first.label != "committed" {
					t.Fatalf("prep submit=%q", first.label)
				}
			},
			wantDisp: submit.Replayed, wantStatus: effectcommit.StatusReplayed,
			web: "refused", cli: "already", want: atSubmitted},
		{name: "subject-mismatch", mutate: func(g *legacyauthorization.AuthorizationGrant) { g.SubjectID = "someone-else" },
			wantDisp: submit.NotCommitted, wantStatus: effectcommit.StatusDenied, web: "refused", cli: "refused", want: atDraft},
		{name: "resource-mismatch", mutate: func(g *legacyauthorization.AuthorizationGrant) { g.Resource += ".other" },
			wantDisp: submit.NotCommitted, wantStatus: effectcommit.StatusDenied, web: "refused", cli: "refused", want: atDraft},
		{name: "grant-source-mismatch", mutate: func(g *legacyauthorization.AuthorizationGrant) {
			e := *g.Epochs
			e.Source += "-other"
			g.Epochs = &e
		}, wantDisp: submit.NotCommitted, wantStatus: effectcommit.StatusDenied, web: "refused", cli: "refused", want: atDraft},
		{name: "grant-subject-link-epoch-mismatch", mutate: func(g *legacyauthorization.AuthorizationGrant) {
			e := *g.Epochs
			e.SubjectLink++
			g.Epochs = &e
		}, wantDisp: submit.NotCommitted, wantStatus: effectcommit.StatusDenied, web: "refused", cli: "refused", want: atDraft},
		{name: "grant-membership-epoch-mismatch", mutate: func(g *legacyauthorization.AuthorizationGrant) {
			e := *g.Epochs
			e.Membership++
			g.Epochs = &e
		}, wantDisp: submit.NotCommitted, wantStatus: effectcommit.StatusDenied, web: "refused", cli: "refused", want: atDraft},
		{name: "grant-policy-epoch-mismatch", mutate: func(g *legacyauthorization.AuthorizationGrant) {
			e := *g.Epochs
			e.Policy++
			g.Epochs = &e
		}, wantDisp: submit.NotCommitted, wantStatus: effectcommit.StatusDenied, web: "refused", cli: "refused", want: atDraft},
		{name: "role-revoked", sql: `UPDATE ` + runnerTestSchema + `.authority SET enabled=FALSE`,
			wantDisp: submit.NotCommitted, wantStatus: effectcommit.StatusDenied, web: "refused", cli: "refused", want: atDraft},
		{name: "membership-epoch-bump", sql: epoch("membership"),
			wantDisp: submit.NotCommitted, wantStatus: effectcommit.StatusDenied, web: "refused", cli: "refused", want: atDraft},
		{name: "subject-link-revoked", sql: epoch("subject_link"),
			wantDisp: submit.NotCommitted, wantStatus: effectcommit.StatusDenied, web: "refused", cli: "refused", want: atDraft},
		{name: "authority-epoch-bump", sql: epoch("policy"),
			wantDisp: submit.NotCommitted, wantStatus: effectcommit.StatusDenied, web: "refused", cli: "refused", want: atDraft},
		{name: "version-mismatch-order-not-draft", sql: `UPDATE public.orders SET status='cancelled' WHERE order_id=1001`,
			wantDisp: submit.NotCommitted, wantStatus: effectcommit.StatusDenied, web: "refused", cli: "refused",
			want: pgSnapshot{"cancelled", "draft", 0, 0}},
		{name: "store-fault-closed-pool",
			provider: func(t *testing.T, env *runnerPGEnv, _ effectcommit.Provider) effectcommit.Provider {
				return atClosedPoolProvider(t, env)
			},
			wantDisp: submit.Failed, wantClass: effectcommit.ErrorClassExecution, web: "failed", cli: "failed", want: atDraft},
		{name: "store-fault-begin-error-with-secret",
			provider: func(_ *testing.T, _ *runnerPGEnv, real effectcommit.Provider) effectcommit.Provider {
				return atFaultProvider{inner: real, beginErr: errors.New(atSecret + ": connection refused")}
			},
			wantDisp: submit.Failed, wantClass: effectcommit.ErrorClassExecution, web: "failed", cli: "failed", want: atDraft},
		{name: "authorization-provider-fault",
			provider: func(_ *testing.T, _ *runnerPGEnv, real effectcommit.Provider) effectcommit.Provider {
				return atFaultProvider{inner: real, checkErr: errors.New(atSecret + ": authority lookup failed")}
			},
			wantDisp: submit.Failed, wantClass: effectcommit.ErrorClassExecution, web: "failed", cli: "failed", want: atDraft},
		{name: "commit-ack-lost-uncertain",
			provider: func(_ *testing.T, _ *runnerPGEnv, real effectcommit.Provider) effectcommit.Provider {
				return ackLossProvider{real}
			},
			wantDisp: submit.Uncertain, wantClass: effectcommit.ErrorClassCommitUncertain, web: "unconfirmed", cli: "unconfirmed",
			want: atSubmitted},
		{name: "grant-source-fault-runner-not-reached", grantErr: true,
			wantDisp: "", web: "failed", cli: "failed", want: atDraft},
	}
}

func TestRunnerOrderSubmitRealPostgresAT2ResultTypeAndAuditAcrossFailureCells(t *testing.T) {
	for _, cell := range atCells() {
		for _, cb := range consumerBuilders {
			kind := cb.name
			t.Run(cell.name+"/"+kind, func(t *testing.T) {
				env := newRunnerPGEnv(t)
				if kind == "cli" {
					// The CLI authority row must exist BEFORE the cell's authority change.
					if _, err := env.db.Exec(cliAuthoritySQL); err != nil {
						t.Fatal(err)
					}
				}
				if _, err := env.db.Exec(atSeedEpochsSQL); err != nil {
					t.Fatal(err)
				}
				if cell.prep != nil {
					cell.prep(t, env, kind)
				}
				if cell.sql != "" {
					if _, err := env.db.Exec(cell.sql); err != nil {
						t.Fatal(err)
					}
				}
				var prov effectcommit.Provider = env.provider(t)
				if cell.provider != nil {
					prov = cell.provider(t, env, prov)
				}
				grants := atGrants(kind, cell.mutate)
				if cell.grantErr {
					grants = func(context.Context) (legacyauthorization.AuthorizationGrant, error) {
						return legacyauthorization.AuthorizationGrant{}, errors.New(atSecret + ": grant lookup failed")
					}
				}
				run := atSubmit(t, env, kind, prov, grants, 1001)

				wantLabel := cell.web
				if kind == "cli" {
					wantLabel = cell.cli
				}
				if run.label != wantLabel {
					t.Fatalf("label=%q want %q (text=%q)", run.label, wantLabel, run.text)
				}
				if cell.wantDisp == "" {
					if run.rec.calls != 0 || len(run.evidence) != 0 {
						t.Fatalf("Runner reached: calls=%d evidence=%d", run.rec.calls, len(run.evidence))
					}
				} else {
					if run.rec.calls != 1 {
						t.Fatalf("runner calls=%d", run.rec.calls)
					}
					if d := submit.Classify(run.rec.result, run.rec.err).Disposition; d != cell.wantDisp {
						t.Fatalf("disposition=%q want %q", d, cell.wantDisp)
					}
					if len(run.evidence) != 1 {
						t.Fatalf("evidence count=%d", len(run.evidence))
					}
					ev := run.evidence[0]
					if ev.Operation != "execute" || ev.Status != cell.wantStatus || ev.ErrorClass != cell.wantClass ||
						ev.Provenance != effectcommit.ProvenanceRealPG {
						t.Fatalf("evidence=%+v", ev)
					}
					atAssertNoLeak(t, "evidence", fmt.Sprintf("%+v", ev))
				}
				atAssertNoLeak(t, "returned text", run.text)
				if got := env.snapshot(t); got != cell.want {
					t.Fatalf("snapshot=%+v want %+v", got, cell.want)
				}
			})
		}
	}
}

// A stale ExpectedVersion is a declared Conflict (NotCommitted), not a
// failure, and writes nothing. Neither consumer builds a non-zero version, so
// this cell is driven at the Runner level through the same Provider and Caller.
func TestRunnerOrderSubmitRealPostgresAT2StaleExpectedVersionIsConflictNotFailure(t *testing.T) {
	env := newRunnerPGEnv(t)
	sink := &atSink{}
	caller := submit.Caller{
		Runner: effectcommit.Runner{Provider: env.provider(t),
			Audit: effectcommit.AuditConfig{Sink: sink, Provenance: effectcommit.ProvenanceRealPG}},
		Grants: atGrants("web", nil), Spec: submit.Spec{Application: runnerSubmitApplication, Capability: runnerSubmitCapability},
	}
	grant := syntheticSubmitGrant()
	intent := submit.Intent{OperationID: "order-submit/1001", ExpectedVersion: 1,
		Payload: []byte(`{"orderId":1001,"to":"submitted"}`)}
	out, err := caller.Execute(context.Background(), grant, intent)
	if err != nil || out.Disposition != submit.NotCommitted || out.Status != effectcommit.StatusConflict {
		t.Fatalf("out=%+v err=%v", out, err)
	}
	if len(sink.got) != 1 || sink.got[0].Status != effectcommit.StatusConflict || sink.got[0].ErrorClass != effectcommit.ErrorClassNone {
		t.Fatalf("evidence=%+v", sink.got)
	}
	if got := env.snapshot(t); got != atDraft {
		t.Fatalf("snapshot=%+v", got)
	}
}
