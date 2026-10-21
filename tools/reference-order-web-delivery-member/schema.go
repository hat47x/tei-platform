package orderwebmember

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Database roles of the two units. Each unit connects only as its own role,
// so the DB enforces the connection bulkhead and the member column grants.
const (
	MemberRole = "owdm_member"
	StaffRole  = "owdm_staff"
)

// RoleLimits are the DB-side connection limits and the member statement
// timeout.
type RoleLimits struct {
	MemberConnections int
	StaffConnections  int
	MemberStatementMS int
}

// DefaultRoleLimits caps the member role across all member unit replicas at
// 16 connections (one unit uses its pool of 4 and one LISTEN connection; the
// tests run up to three replicas at once). The staff role has its own cap.
// The member statement timeout is 10 s: PostgreSQL measures it on the wall
// clock, which jumped forward by several seconds under WSL2 and cancelled a
// 2 s statement that was not slow.
func DefaultRoleLimits() RoleLimits {
	return RoleLimits{MemberConnections: 16, StaffConnections: 10, MemberStatementMS: 10000}
}

// Seed principals and accounts used by the tests.
const (
	MemberA        = "M-A"     // owns account 1
	MemberB        = "M-B"     // owns account 2
	MemberCorp     = "M-CORP"  // sees accounts 3 and 4
	MemberRevoked  = "M-REV"   // owns account 5; revoked in P-OPS-02
	MemberCorp2    = "M-CORP2" // sees accounts 6 and 7
	MemberCorp3    = "M-CORP3" // sees accounts 8 and 9 (negative control)
	StaffMain      = "S-1"
	AccountA       = 1
	AccountB       = 2
	AccountCorp    = 3
	AccountCorp2   = 4
	StaffAccount   = 299 // pending orders confirmed by staff in the load test
	LoadMembers    = 48  // M-L01.. own accounts 101..
	ReorderMembers = 20  // M-R01.. own accounts 10..
	ItemMain       = "ITEM-A"
	ItemHot        = "ITEM-HOT"
	PendingOrders  = 4000
	// Sentinels of internal fields. They must never appear in a member response.
	SentinelCost   = "7777777.77"
	SentinelCredit = "CREDIT-HOLD-Z9"
	SentinelMemo   = "INTERNAL-MEMO-"
)

// TokenOf is the synthetic session token of a principal. Sessions live in the
// DB (owdm.session) so that the units hold no session state.
func TokenOf(principal string) string { return "tok-" + principal }

func LoadMemberID(i int) string    { return fmt.Sprintf("M-L%02d", i) }
func ReorderMemberID(i int) string { return fmt.Sprintf("M-R%02d", i) }

// ResetSchema recreates schema owdm, the two roles and their grants, and the
// seed data. The member grants are generated from the member projection:
// SELECT on exactly the projected columns plus the row-scope columns.
func ResetSchema(ctx context.Context, admin *pgx.Conn, member Projection, limits RoleLimits) error {
	var db string
	if err := admin.QueryRow(ctx, `SELECT current_database()`).Scan(&db); err != nil {
		return err
	}
	if db != "tei_owdm" {
		return fmt.Errorf("refusing to reset schema in database %q (want tei_owdm)", db)
	}
	if member.Kind != KindMember || !member.VerifyDigest() {
		return fmt.Errorf("member projection missing or not verified")
	}
	cols := map[string]bool{"account_id": true, "created_seq": true, "order_id": true}
	for _, f := range member.Fields {
		cols[f.Column] = true
	}
	var selectCols []string
	for c := range cols {
		selectCols = append(selectCols, c)
	}
	sort.Strings(selectCols)
	var writeCols []string
	for _, w := range member.Writable {
		c, _ := member.Column(w)
		writeCols = append(writeCols, c)
	}
	writeCols = append(writeCols, "version")
	sort.Strings(writeCols)

	stmts := []string{
		`DROP SCHEMA IF EXISTS owdm CASCADE`,
		roleSQL(MemberRole, limits.MemberConnections),
		roleSQL(StaffRole, limits.StaffConnections),
		fmt.Sprintf(`ALTER ROLE %s SET statement_timeout = '%dms'`, MemberRole, limits.MemberStatementMS),
		`CREATE SCHEMA owdm`,
		`CREATE TABLE owdm.clock (id int PRIMARY KEY CHECK (id = 1), seq bigint NOT NULL)`,
		`INSERT INTO owdm.clock VALUES (1, 1)`,
		`CREATE TABLE owdm.principal (
			principal_id text PRIMARY KEY,
			kind text NOT NULL CHECK (kind IN ('member','staff')),
			active boolean NOT NULL,
			authz_version bigint NOT NULL)`,
		`CREATE TABLE owdm.principal_scope (
			principal_id text NOT NULL REFERENCES owdm.principal,
			account_id bigint NOT NULL,
			PRIMARY KEY (principal_id, account_id))`,
		`CREATE TABLE owdm.session (
			token text PRIMARY KEY,
			principal_id text NOT NULL REFERENCES owdm.principal)`,
		`CREATE TABLE owdm.item (
			item_code text PRIMARY KEY,
			stock bigint NOT NULL CHECK (stock >= 0),
			version bigint NOT NULL)`,
		`CREATE TABLE owdm.item_version_log (
			item_code text NOT NULL,
			version bigint NOT NULL,
			stock bigint NOT NULL,
			PRIMARY KEY (item_code, version))`,
		`CREATE SEQUENCE owdm.order_id_seq START 10000000`,
		`CREATE TABLE owdm.orders (
			order_id bigint PRIMARY KEY,
			account_id bigint NOT NULL,
			ordered_at timestamptz NOT NULL,
			status text NOT NULL CHECK (status IN ('pending','confirmed')),
			item_code text NOT NULL,
			quantity bigint NOT NULL,
			total_amount numeric(14,2) NOT NULL,
			delivery_note text,
			version bigint NOT NULL,
			created_seq bigint NOT NULL,
			cost numeric(14,2),
			credit_class text,
			staff_memo text,
			confirmed_by text,
			confirmed_at timestamptz)`,
		`CREATE INDEX orders_account_idx ON owdm.orders (account_id, ordered_at DESC)`,
		`CREATE TABLE owdm.confirm_audit (
			order_id bigint NOT NULL,
			staff_id text NOT NULL,
			version bigint NOT NULL,
			at timestamptz NOT NULL DEFAULT clock_timestamp())`,
		// Seed.
		`INSERT INTO owdm.item VALUES ('ITEM-A', 100000, 1), ('ITEM-HOT', 10, 1)`,
		`INSERT INTO owdm.item_version_log SELECT item_code, version, stock FROM owdm.item`,
	}
	principals := []struct {
		id, kind string
		accounts []int
	}{
		{MemberA, KindMember, []int{AccountA}},
		{MemberB, KindMember, []int{AccountB}},
		{MemberCorp, KindMember, []int{AccountCorp, AccountCorp2}},
		{MemberRevoked, KindMember, []int{5}},
		{MemberCorp2, KindMember, []int{6, 7}},
		{MemberCorp3, KindMember, []int{8, 9}},
		{StaffMain, KindStaff, nil},
	}
	for i := 1; i <= LoadMembers; i++ {
		principals = append(principals, struct {
			id, kind string
			accounts []int
		}{LoadMemberID(i), KindMember, []int{100 + i}})
	}
	for i := 1; i <= ReorderMembers; i++ {
		principals = append(principals, struct {
			id, kind string
			accounts []int
		}{ReorderMemberID(i), KindMember, []int{9 + i}})
	}
	for _, p := range principals {
		stmts = append(stmts,
			fmt.Sprintf(`INSERT INTO owdm.principal VALUES ('%s', '%s', true, 1)`, p.id, p.kind),
			fmt.Sprintf(`INSERT INTO owdm.session VALUES ('%s', '%s')`, TokenOf(p.id), p.id))
		for _, a := range p.accounts {
			stmts = append(stmts, fmt.Sprintf(`INSERT INTO owdm.principal_scope VALUES ('%s', %d)`, p.id, a))
		}
	}
	// 20 confirmed orders for every member account; internal fields carry
	// sentinels that a member response must never contain.
	stmts = append(stmts, fmt.Sprintf(`INSERT INTO owdm.orders
		SELECT a * 1000 + n, a, timestamptz '2026-09-01 09:00+09' + (n || ' hours')::interval,
		       'confirmed', 'ITEM-A', 1 + n %% 3, 1000 + n * 10, NULL, 1, 1,
		       %s, '%s', '%s' || (a * 1000 + n), 'S-1', timestamptz '2026-09-02 09:00+09'
		FROM generate_series(1, 200) a, generate_series(1, 20) n`, SentinelCost, SentinelCredit, SentinelMemo),
		fmt.Sprintf(`INSERT INTO owdm.orders
		SELECT 9000000 + n, %d, timestamptz '2026-10-01 09:00+09' + (n || ' seconds')::interval,
		       'pending', 'ITEM-A', 1, 500, NULL, 1, 1, %s, '%s', '%s' || (9000000 + n), NULL, NULL
		FROM generate_series(1, %d) n`, StaffAccount, SentinelCost, SentinelCredit, SentinelMemo, PendingOrders),
		// Grants. Member: column-level, derived from the member projection.
		fmt.Sprintf(`GRANT USAGE ON SCHEMA owdm TO %s, %s`, MemberRole, StaffRole),
		fmt.Sprintf(`GRANT SELECT ON owdm.clock, owdm.principal, owdm.principal_scope, owdm.session, owdm.item, owdm.item_version_log TO %s`, MemberRole),
		fmt.Sprintf(`GRANT SELECT (%s) ON owdm.orders TO %s`, strings.Join(selectCols, ", "), MemberRole),
		fmt.Sprintf(`GRANT UPDATE (%s) ON owdm.orders TO %s`, strings.Join(writeCols, ", "), MemberRole),
		fmt.Sprintf(`GRANT INSERT (order_id, account_id, ordered_at, status, item_code, quantity, total_amount, version, created_seq) ON owdm.orders TO %s`, MemberRole),
		fmt.Sprintf(`GRANT UPDATE (stock, version) ON owdm.item TO %s`, MemberRole),
		fmt.Sprintf(`GRANT INSERT ON owdm.item_version_log TO %s`, MemberRole),
		fmt.Sprintf(`GRANT UPDATE (seq) ON owdm.clock TO %s`, MemberRole),
		fmt.Sprintf(`GRANT USAGE ON SEQUENCE owdm.order_id_seq TO %s`, MemberRole),
		fmt.Sprintf(`GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA owdm TO %s`, StaffRole),
		fmt.Sprintf(`GRANT USAGE ON ALL SEQUENCES IN SCHEMA owdm TO %s`, StaffRole),
		// The first order of every reorder member is one unit of ITEM-HOT.
		fmt.Sprintf(`UPDATE owdm.orders SET item_code = '%s', quantity = 1 WHERE order_id IN (SELECT a * 1000 + 1 FROM generate_series(10, %d) a)`, ItemHot, 9+ReorderMembers),
		`ANALYZE`,
	)
	for _, s := range stmts {
		if _, err := admin.Exec(ctx, s); err != nil {
			return fmt.Errorf("%s: %w", firstLine(s), err)
		}
	}
	return nil
}

func roleSQL(role string, limit int) string {
	return fmt.Sprintf(`DO $$ BEGIN
		IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '%[1]s') THEN CREATE ROLE %[1]s LOGIN; END IF;
		ALTER ROLE %[1]s CONNECTION LIMIT %[2]d;
	END $$`, role, limit)
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i > 0 {
		return s[:i]
	}
	return s
}
