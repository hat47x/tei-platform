package postgresquery

// Real-PostgreSQL verification that ObserveAcquisition's lock-first protocol closes the race a
// catalog read cannot. Opt-in against its OWN empty disposable database:
//
//	TEI_ACQ_OBS_PG_DSN       owner DSN of an EMPTY database tei_acq_obs_e2e
//	TEI_ACQ_OBS_PG_CONFIRM   must be tei_acq_obs_e2e (and equal current_database())
//
// Unset means NOT RUN, not passed.

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const acqObsName = "tei_acq_obs_e2e"

const acqObsSource = `SELECT o.order_id FROM public.orders AS o WHERE o.ordered_at >= :from ORDER BY o.ordered_at DESC`

func TestObserveAcquisitionLocksBeforeObserving(t *testing.T) {
	dsn := os.Getenv("TEI_ACQ_OBS_PG_DSN")
	if dsn == "" {
		t.Skip("acquisition lock-first observation NOT RUN: TEI_ACQ_OBS_PG_DSN unset")
	}
	if os.Getenv("TEI_ACQ_OBS_PG_CONFIRM") != acqObsName {
		t.Fatal("refusing to create fixtures without explicit disposable database confirmation")
	}
	ctx := context.Background()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var name string
	var relations int
	if err := db.QueryRowContext(ctx, `SELECT current_database()`).Scan(&name); err != nil || name != acqObsName {
		t.Fatalf("refusing database %q", name)
	}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
		WHERE c.relkind IN ('r','p','v','m','S','f') AND n.nspname NOT IN ('pg_catalog','information_schema') AND n.nspname NOT LIKE 'pg_toast%'`).Scan(&relations); err != nil || relations != 0 {
		t.Fatalf("refusing a database that already has %d user relations (%v)", relations, err)
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE public.orders (order_id text PRIMARY KEY, ordered_at timestamptz NOT NULL, status text NOT NULL);
		INSERT INTO public.orders VALUES ('O-1','2026-11-07T10:00:00Z','accepted'),('O-2','2026-11-07T10:05:00Z','pending')`); err != nil {
		t.Fatal(err)
	}
	params := map[string]any{"from": time.Date(2026, 11, 7, 9, 0, 0, 0, time.UTC)}

	// Lock first: a concurrent DDL that would turn row security on must wait, then time out,
	// while the observation transaction is open; the observation and the read agree.
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		t.Fatal(err)
	}
	obs, err := ObserveAcquisition(ctx, tx, acqObsSource, params, []string{"public.orders"})
	if err != nil {
		t.Fatal(err)
	}
	if len(obs.LockedTables) != 1 || obs.LockNote != "" || obs.PlanNote != "" || obs.SystemIdentifier == "" {
		t.Fatalf("observation=%+v", obs)
	}
	ddl, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer ddl.Close()
	if _, err := ddl.ExecContext(ctx, `SET lock_timeout = '1500ms'`); err != nil {
		t.Fatal(err)
	}
	_, ddlErr := ddl.ExecContext(ctx, `ALTER TABLE public.orders ENABLE ROW LEVEL SECURITY`)
	if ddlErr == nil || !strings.Contains(ddlErr.Error(), "55P03") {
		t.Fatalf("the concurrent DDL must wait for the lock and time out, got %v", ddlErr)
	}
	var seen int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM public.orders`).Scan(&seen); err != nil || seen != 2 {
		t.Fatalf("read after observation: %d %v", seen, err)
	}
	_ = tx.Rollback()

	// A failing statement is isolated by the savepoint: the transaction stays usable and the
	// failure is reported as a plan note, not as an error that would hide the read's own.
	tx, err = db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		t.Fatal(err)
	}
	obs, err = ObserveAcquisition(ctx, tx, acqObsSource, map[string]any{"from": 42}, []string{"public.orders"})
	if err != nil || obs.PlanNote == "" || len(obs.LockedTables) != 1 {
		t.Fatalf("a wrong argument type must be a plan note: %+v %v", obs, err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM public.orders`).Scan(&seen); err != nil || seen != 2 {
		t.Fatalf("the transaction was aborted by an observation failure: %d %v", seen, err)
	}
	_ = tx.Rollback()

	// An unqualified declared relation is refused, never resolved through search_path.
	tx, err = db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	obs, err = ObserveAcquisition(ctx, tx, acqObsSource, params, []string{"orders"})
	if err != nil || obs.LockNote == "" || len(obs.LockedTables) != 0 {
		t.Fatalf("an unqualified relation must not be locked: %+v %v", obs, err)
	}
	_ = tx.Rollback()

	// Row security: the Provider's answer for the session, not the catalog flag. The DSN user
	// is a superuser, who always bypasses row security, so the session switches to an ordinary
	// role. The same table is observed as inactive for the superuser and active for the role.
	if _, err := db.ExecContext(ctx, `DROP ROLE IF EXISTS tei_obs_user; CREATE ROLE tei_obs_user NOLOGIN;
		GRANT SELECT ON public.orders TO tei_obs_user;
		ALTER TABLE public.orders ENABLE ROW LEVEL SECURITY;
		CREATE POLICY accepted_only ON public.orders FOR SELECT TO tei_obs_user USING (status = 'accepted')`); err != nil {
		t.Fatal(err)
	}
	rlsActive := func(role string) bool {
		tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback()
		if role != "" {
			if _, err := tx.ExecContext(ctx, `SET LOCAL ROLE `+role); err != nil {
				t.Fatal(err)
			}
		}
		obs, err := ObserveAcquisition(ctx, tx, acqObsSource, params, []string{"public.orders"})
		if err != nil {
			t.Fatal(err)
		}
		active := false
		for _, rel := range obs.Relations {
			if rel.Name == "public.orders" && rel.Found && rel.RelKind == "r" {
				active = rel.RowSecurityActive
			}
		}
		var visible int
		if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM public.orders`).Scan(&visible); err != nil {
			t.Fatal(err)
		}
		// The observation must agree with what the read really sees.
		if want := map[bool]int{true: 1, false: 2}[active]; visible != want {
			t.Fatalf("role %q: rowSecurityActive=%v but the read sees %d rows", role, active, visible)
		}
		return active
	}
	if rlsActive("") {
		t.Fatal("a superuser bypasses row security and must be observed as inactive")
	}
	if !rlsActive("tei_obs_user") {
		t.Fatal("an ordinary role under a policy must be observed as active")
	}
}
