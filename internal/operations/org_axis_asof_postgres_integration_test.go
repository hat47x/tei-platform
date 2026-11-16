package operations

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"tei/internal/driver/queryresult"
	postgresquery "tei/internal/driver/queryruntime/postgres"
	"tei/internal/workspace"
)

// This opt-in test runs the same native .tei Query against disposable PostgreSQL.
// It only verifies interval filtering for one row per Membership occurrence.
// Existing PK(membership_id) cannot preserve multiple known-time revisions of
// the SAME occurrence; that remains a separate, unaccepted design decision.
func TestOrgAxisAsOfActualPostgresOptIn(t *testing.T) {
	dsn := os.Getenv("TEI_ORG_AXIS_PG_DSN")
	if dsn == "" {
		t.Skip("real PostgreSQL as-of read NOT RUN: TEI_ORG_AXIS_PG_DSN unset")
	}
	const isolated = "tei_org_axis_e2e"
	if os.Getenv("TEI_ORG_AXIS_PG_CONFIRM_ISOLATED") != isolated {
		t.Fatal("isolated database confirmation absent; refusing to create fixture")
	}
	ctx, stop := context.WithTimeout(context.Background(), 60*time.Second)
	defer stop()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if err := db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	var dbName string
	if err := db.QueryRowContext(ctx, "SELECT current_database()").Scan(&dbName); err != nil || dbName != isolated {
		t.Fatalf("refusing to change database %q: %v", dbName, err)
	}
	var preexisting sql.NullString
	if err := db.QueryRowContext(ctx, "SELECT to_regclass('public.org_memberships')::text").Scan(&preexisting); err != nil {
		t.Fatal(err)
	}
	if preexisting.Valid {
		t.Fatal("refusing to modify pre-existing public.org_memberships")
	}
	ddl := "CREATE TABLE public.org_memberships (" +
		"membership_id varchar PRIMARY KEY, org_unit_id varchar NOT NULL, allocation numeric(8,2) NOT NULL, " +
		"valid_from timestamptz NOT NULL, valid_to timestamptz, known_from timestamptz NOT NULL, known_to timestamptz)"
	if _, err := db.ExecContext(ctx, ddl); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := db.ExecContext(cleanup, "DROP TABLE public.org_memberships"); err != nil {
			t.Errorf("disposable fixture cleanup failed: %v", err)
		}
	}()
	date := func(s string) any {
		if s == "" {
			return nil
		}
		tm, err := time.Parse("2006-01-02", s)
		if err != nil {
			t.Fatal(err)
		}
		return tm
	}
	// Each row has its own occurrence identity; this does NOT model historical
	// correction of A2 with one stable identity. These are literal test inputs.
	records := [][7]string{
		{"A1", "D1", "1.00", "2027-01-01", "", "2027-04-01", ""},
		{"A2", "D1", "0.60", "2027-01-01", "2027-06-01", "2027-04-01", ""},
		{"A3", "D2", "0.40", "2027-01-01", "", "2027-04-01", ""},
		{"A4", "D2", "1.00", "2027-01-01", "", "2027-07-02", ""},
		{"A5", "D1", "0.80", "2027-06-15", "2027-06-16", "2027-01-01", "2027-07-02"},
		{"A6", "D2", "1.20", "2027-07-01", "", "2027-06-01", ""},
		{"A7", "D1", "0.70", "2027-01-01", "", "2027-07-03", ""},
		{"A8", "D2", "0.50", "2027-01-01", "", "2027-01-01", "2027-07-01"},
		{"A9", "D1", "0.10", "2027-01-01", "", "2027-01-01", ""},
	}
	insert := "INSERT INTO public.org_memberships " +
		"(membership_id,org_unit_id,allocation,valid_from,valid_to,known_from,known_to) VALUES ($1,$2,$3,$4,$5,$6,$7)"
	for _, r := range records {
		if _, err := db.ExecContext(ctx, insert, r[0], r[1], r[2], date(r[3]), date(r[4]), date(r[5]), date(r[6])); err != nil {
			t.Fatalf("insert %s: %v", r[0], err)
		}
	}
	// The source mapping's primary-key is membership_id only. Keep the
	// original A2 row and attempt a later known-time correction with the
	// SAME occurrence identity. A different allocation/known_from cannot
	// be stored beside A2 under this physical key. This is a measured
	// negative case, not a proposed revision schema or a Person link.
	t.Run("same membership occurrence cannot co-store known-time revisions", func(t *testing.T) {
		_, err := db.ExecContext(ctx, insert,
			"A2", "D1", "0.90", date("2027-01-01"), nil, date("2027-07-03"), nil)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
			t.Fatalf("duplicate A2 version must fail with PostgreSQL unique_violation (23505), got: %v", err)
		}
		var total, original int
		if err := db.QueryRowContext(ctx,
			"SELECT COUNT(*), COUNT(*) FILTER (WHERE allocation = 0.60 AND known_from = DATE '2027-04-01') FROM public.org_memberships WHERE membership_id = $1",
			"A2").Scan(&total, &original); err != nil {
			t.Fatal(err)
		}
		if total != 1 || original != 1 {
			t.Fatalf("rejected revision must not alter original A2: total=%d original=%d", total, original)
		}
	})

	loaded, err := workspace.Load(filepath.Join("..", "..", "reference", "org-staged-axis"))
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.World.Queries) != 1 || len(loaded.World.Applications) != 1 {
		t.Fatal("unexpected native Query/Application cardinality")
	}
	q := loaded.World.Queries[0]
	// Reuse the exact Source/IR path qualification in this *same* actual
	// PostgreSQL read attempt. A standalone Source test and an unrelated
	// Provider success cannot be combined into one route witness.
	qualifiedUnitRoute := false
	for _, namespace := range loaded.World.Semantics {
		for _, data := range namespace.Data {
			if data.ID != "organization.membership" {
				continue
			}
			for _, relation := range data.Relations {
				if relation.ID == "organization.membership.unit" &&
					orgC1StableUnitReadPath(relation, q) {
					qualifiedUnitRoute = true
				}
			}
		}
	}
	if !qualifiedUnitRoute {
		t.Fatal("the native PostgreSQL read must use the Source-qualified Membership/OrgUnit and as-of Query interface")
	}
	ds, ok := loaded.World.Applications[0].DatabaseByAliasExact(q.Datasource)
	if !ok || ds.TargetDatabase != "org_store" {
		t.Fatal("native datasource mismatch")
	}
	executor := postgresquery.Executor{DB: db, Database: ds.TargetDatabase}
	for _, tc := range []struct {
		title, validAt, knownAt, minimum string
		want                             []string
	}{
		{"valid and known boundary", "2027-06-15", "2027-07-01", "0.40", []string{"A1", "A3", "A5"}},
		{"late known source", "2027-06-15", "2027-07-03", "0.40", []string{"A1", "A3", "A4", "A7"}},
		{"before valid end and higher allocation", "2027-05-31", "2027-07-01", "0.60", []string{"A1", "A2"}},
		{"at valid end is excluded", "2027-06-16", "2027-07-01", "0.40", []string{"A1", "A3"}},
	} {
		t.Run(tc.title, func(t *testing.T) {
			actual, err := executor.ExecuteBrowseQuery(ctx, q, ds, map[string]any{
				"valid_at": date(tc.validAt), "known_at": date(tc.knownAt), "allocation_min": tc.minimum,
			})
			if err != nil {
				t.Fatal(err)
			}
			mapped, err := queryresult.Map(q, actual.Raw)
			if err != nil {
				t.Fatal(err)
			}
			if ids := orgAxisResultIDs(mapped.Rows); !reflect.DeepEqual(ids, tc.want) {
				t.Fatalf("actual=%v expected=%v", ids, tc.want)
			}
		})
	}

	// The actual as-of Query projects Membership, OrgUnit, and allocation,
	// not the Person behind each Membership. Two *hypothetical* Person
	// assignments can therefore agree on every observed Query row while
	// making a distinct-Person claim disagree. This is an observational
	// insufficiency witness, NOT an adopted Person link or r7 identity map.
	t.Run("same as-of projection cannot establish distinct Person count", func(t *testing.T) {
		const candidate = "pg_temp.org_person_link_observation_candidate"
		if _, err := db.ExecContext(ctx,
			"CREATE TEMP TABLE org_person_link_observation_candidate (" +
				"candidate_world varchar NOT NULL, membership_id varchar NOT NULL, person_id varchar NOT NULL, " +
				"PRIMARY KEY(candidate_world, membership_id))"); err != nil {
			t.Fatal(err)
		}
		defer func() {
			cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if _, err := db.ExecContext(cleanup, "DROP TABLE " + candidate); err != nil {
				t.Errorf("drop hypothetical Person links: %v", err)
			}
		}()

		// These are alternative *unaccepted candidate worlds* over the same
		// four actual Membership IDs, not the independent org-synthetic-r7
		// fixture. No candidate Person IDs are added to the product mapping.
		for _, row := range [][3]string{
			{"all_distinct", "A1", "P1"},
			{"all_distinct", "A3", "P2"},
			{"all_distinct", "A4", "P3"},
			{"all_distinct", "A7", "P4"},
			{"shared_person", "A1", "P1"},
			{"shared_person", "A3", "P2"},
			{"shared_person", "A4", "P2"},
			{"shared_person", "A7", "P3"},
		} {
			if _, err := db.ExecContext(ctx,
				"INSERT INTO " + candidate + " (candidate_world,membership_id,person_id) VALUES ($1,$2,$3)",
				row[0], row[1], row[2]); err != nil {
				t.Fatalf("insert hypothetical identity %v: %v", row, err)
			}
		}

		wantMemberships := []string{"A1", "A3", "A4", "A7"}
		var firstProjection any
		for _, tc := range []struct {
			world       string
			wantPersons int
		}{
			{"all_distinct", 4},
			{"shared_person", 3},
		} {
			t.Run(tc.world, func(t *testing.T) {
				actual, err := executor.ExecuteBrowseQuery(ctx, q, ds, map[string]any{
					"valid_at": date("2027-06-15"), "known_at": date("2027-07-03"),
					"allocation_min": "0.40",
				})
				if err != nil {
					t.Fatal(err)
				}
				mapped, err := queryresult.Map(q, actual.Raw)
				if err != nil {
					t.Fatal(err)
				}
				if ids := orgAxisResultIDs(mapped.Rows); !reflect.DeepEqual(ids, wantMemberships) {
					t.Fatalf("actual Membership projection=%v want=%v", ids, wantMemberships)
				}
				if firstProjection == nil {
					firstProjection = mapped.Rows
				} else if !reflect.DeepEqual(firstProjection, mapped.Rows) {
					t.Fatalf("same actual Query produced different projections across candidate worlds: %v vs %v", firstProjection, mapped.Rows)
				}
				for _, row := range mapped.Rows {
					if len(row) != 3 {
						t.Fatalf("unexpected as-of result columns: %v", row)
					}
				}
				var joined, persons int
				err = db.QueryRowContext(ctx,
					"SELECT COUNT(*), COUNT(DISTINCT person_id) FROM " + candidate + " WHERE candidate_world = $1",
					tc.world).Scan(&joined, &persons)
				if err != nil {
					t.Fatal(err)
				}
				if joined != len(wantMemberships) || persons != tc.wantPersons {
					t.Fatalf("candidate %s has joined=%d distinctPersons=%d, expected=%d/%d",
						tc.world, joined, persons, len(wantMemberships), tc.wantPersons)
				}
			})
		}
	})
}

// TestOrgAxisAsOfRevisionKeyCandidateOptIn compares a possible *physical*
// identity for historical revisions without changing the product's Persistence
// Mapping or declaring the candidate as accepted business meaning.
//
// The candidate primary key (membership_id, known_from) can store two revisions
// but cannot itself reject overlapping known-time intervals. Both properties
// are measured in a transaction against disposable PostgreSQL. The transaction
// is rolled back even when a test fails.
func TestOrgAxisAsOfRevisionKeyCandidateOptIn(t *testing.T) {
	dsn := os.Getenv("TEI_ORG_AXIS_PG_DSN")
	if dsn == "" {
		t.Skip("physical revision candidate NOT RUN: TEI_ORG_AXIS_PG_DSN unset")
	}
	const isolated = "tei_org_axis_e2e"
	if os.Getenv("TEI_ORG_AXIS_PG_CONFIRM_ISOLATED") != isolated {
		t.Fatal("isolated database confirmation absent")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var dbName string
	if err := db.QueryRowContext(ctx, "SELECT current_database()").Scan(&dbName); err != nil || dbName != isolated {
		t.Fatalf("refusing revision candidate outside isolated DB %q: %v", dbName, err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `CREATE TEMP TABLE org_membership_revision_candidate (
		membership_id text NOT NULL,
		known_from timestamptz NOT NULL,
		known_to timestamptz,
		allocation numeric(8,2) NOT NULL,
		PRIMARY KEY (membership_id, known_from)
	) ON COMMIT DROP`)
	if err != nil {
		t.Fatal(err)
	}
	const insert = `INSERT INTO org_membership_revision_candidate
		(membership_id, known_from, known_to, allocation)
		VALUES ($1, $2::timestamptz, $3::timestamptz, $4)`
	if _, err := tx.ExecContext(ctx, insert, "A2", "2027-04-01", "2027-07-03", "0.60"); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, insert, "A2", "2027-07-03", nil, "0.90"); err != nil {
		t.Fatalf("candidate must be able to store two versions of A2: %v", err)
	}
	var versionCount int
	if err := tx.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM org_membership_revision_candidate WHERE membership_id = $1", "A2").Scan(&versionCount); err != nil {
		t.Fatal(err)
	}
	if versionCount != 2 {
		t.Fatalf("two revisions must remain stored: got %d", versionCount)
	}
	const asOf = `SELECT allocation::text FROM org_membership_revision_candidate
		WHERE membership_id = $1 AND known_from <= $2::timestamptz
		AND (known_to IS NULL OR $2::timestamptz < known_to)`
	for _, tc := range []struct {
		name, knownAt, allocation string
	}{
		{"previous version immediately before boundary", "2027-07-02", "0.60"},
		{"corrected version exactly at boundary", "2027-07-03", "0.90"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows, err := tx.QueryContext(ctx, asOf, "A2", tc.knownAt)
			if err != nil {
				t.Fatal(err)
			}
			defer rows.Close()
			var values []string
			for rows.Next() {
				var v string
				if err := rows.Scan(&v); err != nil {
					t.Fatal(err)
				}
				values = append(values, v)
			}
			if err := rows.Err(); err != nil {
				t.Fatal(err)
			}
			if len(values) != 1 || values[0] != tc.allocation {
				t.Fatalf("as-of %s: got %v, want [%s]", tc.knownAt, values, tc.allocation)
			}
		})
	}
	// This third row has a distinct key, but overlaps both accepted candidates.
	// A composite PK alone is not an interval-exclusion constraint.
	if _, err := tx.ExecContext(ctx, insert, "A2", "2027-06-01", "2027-08-01", "0.70"); err != nil {
		t.Fatalf("candidate key unexpectedly rejected overlapping known-time version: %v", err)
	}
	var active, distinct int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*), COUNT(DISTINCT allocation)
		FROM org_membership_revision_candidate
		WHERE membership_id = $1 AND known_from <= $2::timestamptz
		AND (known_to IS NULL OR $2::timestamptz < known_to)`,
		"A2", "2027-07-03").Scan(&active, &distinct); err != nil {
		t.Fatal(err)
	}
	if active != 2 || distinct != 2 {
		t.Fatalf("overlapping candidate should leave two conflicting versions, count=%d distinct=%d", active, distinct)
	}
}
