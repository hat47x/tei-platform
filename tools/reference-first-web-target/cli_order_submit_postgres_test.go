//go:build !tei_repair_only

package main

import (
	"bytes"
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"tei/internal/security/effectcommit"
	effectpostgres "tei/internal/security/effectcommit/postgres"
	"tei/internal/security/effectcommit/submit"
	"tei/internal/security/legacyauthorization"
)

// Real PostgreSQL tests that run the SAME scenarios through two consumers of the
// one effectcommit.Runner: the Web store (runnerOrderSubmitStore) and the
// test-only in-process CLI driver (cliOrderRun, built on submit.Caller). They
// share the DDL and fixture of runner_order_submit_postgres_test.go (TEST-ONLY
// DDL, synthetic grant) and are gated by TEI_TEST_POSTGRES_DSN. Without it they
// SKIP, and a skip is not evidence. The test names start with
// TestRunnerOrderSubmitRealPostgres so that the existing CI step (-run
// '^TestRunnerOrderSubmit', fails on SKIP) would also select them; this has not
// been run in CI. They prove nothing about production atomicity, and the grant
// is not a security.authorize result.

const cliAuthoritySQL = `INSERT INTO ` + runnerTestSchema + `.authority VALUES
 ('` + cliApplication + `','` + cliCapability + `','legacy-order-editor','execute','` + cliCapability + `','test-authority',1,1,1,TRUE) ON CONFLICT DO NOTHING`

func cliSyntheticGrant() legacyauthorization.AuthorizationGrant {
	g := syntheticSubmitGrant()
	g.Resource = cliCapability
	return g
}

func (e *runnerPGEnv) provider(t *testing.T) effectpostgres.Provider {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), e.dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	p, err := effectpostgres.NewWithTrustedOwnerRoutine(pool, runnerTestSchema)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// snapshot is everything a scenario must leave unchanged or changed exactly once.
type pgSnapshot struct {
	s1001, s1002 string
	receipts     int
	versionSum   int
}

func (e *runnerPGEnv) snapshot(t *testing.T) pgSnapshot {
	t.Helper()
	s := pgSnapshot{s1001: orderStatus(t, e.db, 1001), s1002: orderStatus(t, e.db, 1002), receipts: e.receipts(t)}
	if err := e.db.QueryRow(`SELECT coalesce(sum(resource_version),0) FROM ` + runnerTestSchema + `.receipts`).Scan(&s.versionSum); err != nil {
		t.Fatal(err)
	}
	return s
}

// A submitConsumer reduces one consumer to coarse labels:
// committed, already, refused, unconfirmed, failed, norecord.
type submitConsumer struct {
	name     string
	opID     string // OperationID this consumer uses for order 1001
	repeated string // label of a repeated submit (the consumers differ on purpose)
	submit   func(id int64) string
	lookup   func(id int64) (label, operation string)
}

func lookupLabel(o submit.Outcome) string {
	switch o.Disposition {
	case submit.Replayed:
		return "already"
	case submit.NotRecorded:
		return "norecord"
	case submit.NotCommitted:
		return "refused"
	default:
		return "failed"
	}
}

func webConsumer(t *testing.T, e *runnerPGEnv, lossy bool) submitConsumer {
	t.Helper()
	clean := e.store(t)
	writer := clean
	if lossy {
		var err error
		writer, err = newRunnerOrderSubmitStore(effectcommit.Runner{Provider: ackLossProvider{e.provider(t)}},
			func(context.Context, int64) (legacyauthorization.AuthorizationGrant, error) {
				return syntheticSubmitGrant(), nil
			}, nil, &statusReader{})
		if err != nil {
			t.Fatal(err)
		}
	}
	return submitConsumer{
		name: "web", opID: "order-submit/1001", repeated: "refused",
		submit: func(id int64) string {
			switch err := writer.Submit(context.Background(), id); {
			case err == nil:
				return "committed"
			case errors.Is(err, errOrderNotSubmittable):
				return "refused"
			case errors.Is(err, errOrderSubmitCommitUnconfirmed):
				return "unconfirmed"
			default:
				return "failed"
			}
		},
		lookup: func(id int64) (string, string) {
			res, err := clean.LookupSubmit(context.Background(), id)
			o := submit.ClassifyLookup(res, err)
			return lookupLabel(o), o.OperationID
		},
	}
}

func cliConsumer(t *testing.T, e *runnerPGEnv, lossy bool) submitConsumer {
	t.Helper()
	if _, err := e.db.Exec(cliAuthoritySQL); err != nil {
		t.Fatal(err)
	}
	grants := func(context.Context) (legacyauthorization.AuthorizationGrant, error) { return cliSyntheticGrant(), nil }
	cleanRunner := effectcommit.Runner{Provider: e.provider(t)}
	writeRunner := submit.Runner(cleanRunner)
	if lossy {
		writeRunner = effectcommit.Runner{Provider: ackLossProvider{e.provider(t)}}
	}
	write := submit.Caller{Runner: writeRunner, Grants: grants, Spec: cliSpec}
	read := submit.Caller{Runner: cleanRunner, Grants: grants, Spec: cliSpec}
	return submitConsumer{
		name: "cli", opID: "cli-order-submit/1001", repeated: "already",
		submit: func(id int64) string {
			var out bytes.Buffer
			code := cliOrderRun(context.Background(), write, []string{"submit-order", itoa(id)}, &out)
			line := out.String()
			switch {
			case code == cliExitOK && strings.HasPrefix(line, "submitted order"):
				return "committed"
			case code == cliExitOK && strings.HasPrefix(line, "already submitted order"):
				return "already"
			case code == cliExitNotCommitted:
				return "refused"
			case code == cliExitUncertain:
				if strings.Contains(line, "postgres://") || strings.Contains(line, "lost-ack-secret") {
					return "leak"
				}
				return "unconfirmed"
			default:
				return "failed"
			}
		},
		lookup: func(id int64) (string, string) {
			var out bytes.Buffer
			code := cliOrderRun(context.Background(), read, []string{"lookup-order", itoa(id)}, &out)
			line := out.String()
			op := ""
			if i := strings.Index(line, "operation="); i >= 0 {
				op = strings.TrimSpace(line[i+len("operation="):])
			}
			switch code {
			case cliExitOK:
				return "already", op
			case cliExitNotRecorded:
				return "norecord", ""
			case cliExitNotCommitted:
				return "refused", ""
			default:
				return "failed", ""
			}
		},
	}
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

var consumerBuilders = []struct {
	name  string
	build func(*testing.T, *runnerPGEnv, bool) submitConsumer
}{{"web", webConsumer}, {"cli", cliConsumer}}

func TestRunnerOrderSubmitRealPostgresSharedConstraintHappyPathAndRepeat(t *testing.T) {
	for _, cb := range consumerBuilders {
		t.Run(cb.name, func(t *testing.T) {
			env := newRunnerPGEnv(t)
			c := cb.build(t, env, false)
			if got := c.submit(1001); got != "committed" {
				t.Fatalf("submit=%q", got)
			}
			after := env.snapshot(t)
			if after != (pgSnapshot{"submitted", "draft", 1, 1}) {
				t.Fatalf("snapshot=%+v", after)
			}
			// A repeated submit is the one place the consumers differ on purpose.
			if got := c.submit(1001); got != c.repeated {
				t.Fatalf("repeat=%q want %q", got, c.repeated)
			}
			if env.snapshot(t) != after {
				t.Fatal("repeated submit changed state")
			}
		})
	}
}

func TestRunnerOrderSubmitRealPostgresSharedConstraintEpochBumpAndRevoke(t *testing.T) {
	for name, stmt := range map[string]string{
		"epoch-bump": `UPDATE ` + runnerTestSchema + `.authority SET policy=policy+1`,
		"revoked":    `UPDATE ` + runnerTestSchema + `.authority SET enabled=FALSE`,
	} {
		for _, cb := range consumerBuilders {
			t.Run(name+"/"+cb.name, func(t *testing.T) {
				env := newRunnerPGEnv(t)
				c := cb.build(t, env, false)
				if _, err := env.db.Exec(stmt); err != nil {
					t.Fatal(err)
				}
				if got := c.submit(1001); got != "refused" {
					t.Fatalf("%s submit=%q", c.name, got)
				}
				if got, _ := c.lookup(1001); got != "refused" {
					t.Fatalf("%s lookup=%q", c.name, got)
				}
				if env.snapshot(t) != (pgSnapshot{"draft", "draft", 0, 0}) {
					t.Fatalf("%s snapshot=%+v", c.name, env.snapshot(t))
				}
			})
		}
	}
}

func TestRunnerOrderSubmitRealPostgresSharedConstraintAckLossThenLookup(t *testing.T) {
	for _, cb := range consumerBuilders {
		t.Run(cb.name, func(t *testing.T) {
			env := newRunnerPGEnv(t)
			lossy := cb.build(t, env, true)
			if got := lossy.submit(1001); got != "unconfirmed" {
				t.Fatalf("submit=%q (leak or wrong class)", got)
			}
			// The write really committed.
			committed := env.snapshot(t)
			if committed != (pgSnapshot{"submitted", "draft", 1, 1}) {
				t.Fatalf("snapshot=%+v", committed)
			}
			label, op := lossy.lookup(1001)
			if label != "already" || op != lossy.opID {
				t.Fatalf("lookup=%q operation=%q want %q", label, op, lossy.opID)
			}
			if miss, _ := lossy.lookup(1002); miss != "norecord" {
				t.Fatalf("1002 lookup=%q", miss)
			}
			// Lookup is read-only; a repeated submit through a clean consumer of the
			// same kind does not produce a second effect either.
			if env.snapshot(t) != committed {
				t.Fatal("lookup changed state")
			}
			clean := cb.build(t, env, false)
			if got := clean.submit(1001); got != clean.repeated {
				t.Fatalf("repeat=%q want %q", got, clean.repeated)
			}
			if env.snapshot(t) != committed {
				t.Fatal("repeated submit produced a second effect")
			}
		})
	}
}
