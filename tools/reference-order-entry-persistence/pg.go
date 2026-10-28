package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"tei/tools/internal/postgresdb"
)

// 安全guardと所有印は、reference-data-first-persistenceの表非依存部分の考え方を
// このツール専用の接頭辞で再実装したもの。共通化は別途行う。

const (
	ordersTable              = "public.orders"
	orderLinesTable          = "public.order_lines"
	ownershipPrefix          = "tei:reference-order-entry-persistence:"
	postgresDSNEnv           = "TEI_TEST_POSTGRES_DSN"
	postgresDSNFileEnv       = "TEI_TEST_POSTGRES_DSN_FILE"
	fixtureOrderID     int64 = 1001
	fixtureLineCount         = 2
	fixtureTotalAmount       = "300.00"
)

// fixtureSeedSQLは、submit前のpendingの注文と明細を入れる。値は検証用の合成データで、
// 実業務データではない。列名はPersistence Mappingから導いたDDLの列名と一致する。
var fixtureSeedSQL = []string{
	`INSERT INTO public.orders (order_id, customer_id, ordered_at, status, total_amount) VALUES (1001, 2001, '2026-09-18T08:30:00Z', 'pending', 300.00)`,
	`INSERT INTO public.order_lines (order_line_id, order_id, line_number, item_code, quantity, unit_price) VALUES (1, 1001, 1, 'ITEM-A', 2, 100.00), (2, 1001, 2, 'ITEM-B', 1, 100.00)`,
}

const userRelationCountQuery = `
SELECT count(*)
FROM pg_catalog.pg_class AS c
JOIN pg_catalog.pg_namespace AS n ON n.oid = c.relnamespace
WHERE n.nspname NOT IN ('pg_catalog', 'information_schema')
  AND n.nspname NOT LIKE 'pg_toast%'
  AND n.nspname NOT LIKE 'pg_temp_%'
  AND c.relkind IN ('r', 'p', 'v', 'm', 'f', 'S')
`

const eventTriggerCountQuery = `SELECT count(*) FROM pg_catalog.pg_event_trigger`
const markerQuery = `SELECT pg_catalog.obj_description('public.orders'::regclass, 'pg_class')`

// ---------------------------------------------------------------------------
// DSN入力
// ---------------------------------------------------------------------------

type dsnLookupEnvFunc func(string) (string, bool)
type dsnReadFileFunc func(string) ([]byte, error)

func loadPostgresDSNInput(lookupEnv dsnLookupEnvFunc, readFile dsnReadFileFunc) (string, error) {
	direct, directSet := lookupEnv(postgresDSNEnv)
	path, fileSet := lookupEnv(postgresDSNFileEnv)
	if directSet && fileSet {
		return "", fmt.Errorf("PostgreSQL DSNの直接入力とsecret file入力が同時に設定されています。どちらか一方だけを指定してください")
	}
	if directSet {
		if direct == "" {
			return "", fmt.Errorf("%sが空です", postgresDSNEnv)
		}
		return direct, nil
	}
	if !fileSet || path == "" {
		return "", fmt.Errorf("%sまたは%sのどちらか一方を指定してください", postgresDSNEnv, postgresDSNFileEnv)
	}
	content, err := readFile(path)
	if err != nil {
		return "", fmt.Errorf("PostgreSQL DSN secret fileを読み取れません")
	}
	if bytes.IndexByte(content, 0) >= 0 {
		return "", fmt.Errorf("PostgreSQL DSN secret fileにNULが含まれているため受理しません")
	}
	content = bytes.TrimSuffix(bytes.TrimSuffix(content, []byte("\n")), []byte("\r"))
	if len(content) == 0 {
		return "", fmt.Errorf("PostgreSQL DSN secret fileが空です")
	}
	if bytes.ContainsAny(content, "\r\n") {
		return "", fmt.Errorf("PostgreSQL DSN secret fileは一行でなければなりません")
	}
	return string(content), nil
}

// clearPostgresDSNEnvironmentは、読み取り後にDSNを環境から消す。子processへ渡さない。
func clearPostgresDSNEnvironment() error {
	if err := os.Unsetenv(postgresDSNEnv); err != nil {
		return fmt.Errorf("PostgreSQL DSN環境変数を消去できません")
	}
	if err := os.Unsetenv(postgresDSNFileEnv); err != nil {
		return fmt.Errorf("PostgreSQL DSN secret file環境変数を消去できません")
	}
	return nil
}

// ---------------------------------------------------------------------------
// 安全guardと所有印
// ---------------------------------------------------------------------------

func validateDedicatedFixtureDatabase(databaseName string, userRelations, eventTriggers int) error {
	name := strings.TrimSpace(databaseName)
	if name == "" {
		return fmt.Errorf("安全のためツールを停止します: 接続先Database identityが空です")
	}
	switch name {
	case "postgres", "template0", "template1":
		return fmt.Errorf("安全のためツールを停止します: PostgreSQL既定Databaseは実行先にできません。専用の空Databaseを指定してください")
	}
	if userRelations != 0 {
		return fmt.Errorf("安全のためツールを停止します: 接続先Databaseには既存user relationが%d件あります。専用の空Databaseを指定してください", userRelations)
	}
	if eventTriggers != 0 {
		return fmt.Errorf("安全のためツールを停止します: 接続先DatabaseにはEvent Triggerが%d件あります。専用の空Databaseを指定してください", eventTriggers)
	}
	return nil
}

func newOwnershipMarker() (string, error) {
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("ownership markerを生成できません")
	}
	return ownershipPrefix + hex.EncodeToString(random[:]), nil
}

// validateOwnershipMarkerは、固定prefixと小文字16進32桁だけを受理する。
// この形のmarkerだけをSQL literalへ埋め込む。
func validateOwnershipMarker(marker string) error {
	if !strings.HasPrefix(marker, ownershipPrefix) {
		return fmt.Errorf("ownership markerのprefixが不正です")
	}
	suffix := strings.TrimPrefix(marker, ownershipPrefix)
	if len(suffix) != 32 || suffix != strings.ToLower(suffix) {
		return fmt.Errorf("ownership markerの形式が不正です")
	}
	if _, err := hex.DecodeString(suffix); err != nil {
		return fmt.Errorf("ownership markerのrandom部が16進数ではありません")
	}
	return nil
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := postgresdb.OpenSQL(dsn)
	if err != nil {
		return nil, err
	}
	// 一つの接続だけを使わせ、pg_backend_pidをprocess内の識別として固定する。
	db.SetMaxOpenConns(1)
	return db, nil
}

func countOne(ctx context.Context, db *sql.DB, query, what string) (int, error) {
	var n int
	if err := db.QueryRowContext(ctx, query).Scan(&n); err != nil {
		return 0, fmt.Errorf("%sを確認できません", what)
	}
	return n, nil
}

func relationAbsent(ctx context.Context, db *sql.DB, relation string) (bool, error) {
	var object sql.NullString
	if err := db.QueryRowContext(ctx, `SELECT to_regclass($1)::text`, relation).Scan(&object); err != nil {
		return false, fmt.Errorf("fixture objectの有無を確認できません")
	}
	return !object.Valid || strings.TrimSpace(object.String) == "", nil
}

func backendPID(ctx context.Context, db *sql.DB) (int64, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	var pid int64
	if err := conn.QueryRowContext(ctx, `SELECT pg_backend_pid()`).Scan(&pid); err != nil {
		return 0, err
	}
	return pid, nil
}

// ensureDedicatedEmptyDatabaseは、専用の空DBにだけfixtureを作る。既存のuser relation、
// Event Trigger、既存のfixture objectがあれば停止する。
func ensureDedicatedEmptyDatabase(ctx context.Context, db *sql.DB) (safetyEvidence, error) {
	var result safetyEvidence
	var databaseName string
	if err := db.QueryRowContext(ctx, `SELECT current_database()`).Scan(&databaseName); err != nil {
		return result, fmt.Errorf("接続先Database identityを確認できません")
	}
	userRelations, err := countOne(ctx, db, userRelationCountQuery, "既存user relation")
	if err != nil {
		return result, err
	}
	eventTriggers, err := countOne(ctx, db, eventTriggerCountQuery, "Event Trigger")
	if err != nil {
		return result, err
	}
	if err := validateDedicatedFixtureDatabase(databaseName, userRelations, eventTriggers); err != nil {
		return result, err
	}
	result.NonDefaultDatabaseVerified = true
	result.UserRelationsBeforeFixture = userRelations
	result.EventTriggersBeforeFixture = eventTriggers
	for _, relation := range []string{ordersTable, orderLinesTable} {
		absent, err := relationAbsent(ctx, db, relation)
		if err != nil {
			return result, err
		}
		if !absent {
			return result, fmt.Errorf("安全のためツールを停止します: %sが既に存在します", relation)
		}
	}
	result.FixtureObjectsAbsentBeforeCreate = true
	result.GuardPassed = true
	return result, nil
}

// observeOwnershipMarkerは、DB側のmarkerを読む。write/readは、このツールがprepareで作った
// fixtureにだけ接続していることを、marker接頭辞の確認で保証する。
func observeOwnershipMarker(ctx context.Context, db *sql.DB) (string, error) {
	var observed sql.NullString
	if err := db.QueryRowContext(ctx, markerQuery).Scan(&observed); err != nil {
		return "", fmt.Errorf("安全のため停止します: DB側ownership markerを確認できません")
	}
	if !observed.Valid || observed.String == "" {
		return "", fmt.Errorf("安全のため停止します: DB側ownership markerがありません")
	}
	if err := validateOwnershipMarker(observed.String); err != nil {
		return "", fmt.Errorf("安全のため停止します: DB側ownership markerがこのツールのものではありません")
	}
	return observed.String, nil
}

// ---------------------------------------------------------------------------
// fixtureの作成・更新・読取・削除
// ---------------------------------------------------------------------------

// createFixtureは、Persistence Mappingから導いたDDLを一つのtransactionで適用し、
// ownership markerとseedを入れる。markerは呼び出し側が保持してcleanupへ渡す。
func createFixture(ctx context.Context, db *sql.DB, ddl []string) (string, error) {
	marker, err := newOwnershipMarker()
	if err != nil {
		return "", err
	}
	if err := validateOwnershipMarker(marker); err != nil {
		return "", err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("fixture作成transactionを開始できません")
	}
	defer tx.Rollback()
	for _, statement := range ddl {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return "", fmt.Errorf("DDLを適用できません: %w", err)
		}
	}
	// markerは固定prefixと小文字16進数だけなので、SQL literalへ埋め込める。
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("COMMENT ON TABLE public.orders IS '%s'", marker)); err != nil {
		return "", fmt.Errorf("ownership markerを設定できません")
	}
	for _, statement := range fixtureSeedSQL {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return "", fmt.Errorf("seedを入れられません: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("fixture作成transactionを確定できません")
	}
	return marker, nil
}

// removeFixtureは、DB側のmarkerが期待と一致する場合だけfixtureを削除する。
func removeFixture(ctx context.Context, db *sql.DB, expectedMarker string) error {
	if err := validateOwnershipMarker(expectedMarker); err != nil {
		return fmt.Errorf("安全のためfixtureを削除しません: 期待ownership markerが不正です")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("fixture cleanup transactionを開始できません")
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `LOCK TABLE public.orders, public.order_lines IN ACCESS EXCLUSIVE MODE`); err != nil {
		return fmt.Errorf("安全のためfixtureを削除しません: ownership確認用lockを取得できません")
	}
	var observed sql.NullString
	if err := tx.QueryRowContext(ctx, markerQuery).Scan(&observed); err != nil {
		return fmt.Errorf("安全のためfixtureを削除しません: DB側ownership markerを確認できません")
	}
	if !observed.Valid || observed.String != expectedMarker {
		return fmt.Errorf("安全のためfixtureを削除しません: DB側ownership markerがprepare時のmarkerと一致しません")
	}
	if _, err := tx.ExecContext(ctx, `DROP TABLE public.order_lines, public.orders`); err != nil {
		return fmt.Errorf("ownership確認済みfixtureを削除できません")
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("fixture cleanup transactionを確定できません")
	}
	return nil
}

func verifyDatabaseRestored(ctx context.Context, db *sql.DB) (cleanupEvidence, error) {
	var result cleanupEvidence
	relations, err := countOne(ctx, db, userRelationCountQuery, "終了時のuser relation")
	if err != nil {
		return result, err
	}
	triggers, err := countOne(ctx, db, eventTriggerCountQuery, "終了時のEvent Trigger")
	if err != nil {
		return result, err
	}
	result.UserRelationsAfterCleanup = relations
	result.EventTriggersAfterCleanup = triggers
	if relations != 0 || triggers != 0 {
		return result, fmt.Errorf("終了時にuser relation=%d, Event Trigger=%dが残っています", relations, triggers)
	}
	result.Verified = true
	return result, nil
}

// pgStoreは、submitの唯一のDB更新を実PostgreSQLへ行う。
type pgStore struct{ db *sql.DB }

var _ orderStore = pgStore{}

func (s pgStore) SubmitOrder(ctx context.Context, orderID int64) (int64, error) {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	var pid int64
	if err := conn.QueryRowContext(ctx, `SELECT pg_backend_pid()`).Scan(&pid); err != nil {
		return 0, err
	}
	result, err := conn.ExecContext(ctx,
		`UPDATE public.orders SET status = 'submitted' WHERE order_id = $1 AND status = 'pending'`, orderID)
	if err != nil {
		return pid, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return pid, err
	}
	if affected != 1 {
		return pid, fmt.Errorf("pendingの注文を1件更新できません: %d件", affected)
	}
	return pid, nil
}

// orderObservationは、Commandとは別のprocess・別の接続が読んだ注文の状態である。
type orderObservation struct {
	Status      string
	TotalAmount string
	LineCount   int
	BackendPID  int64
}

func readOrder(ctx context.Context, db *sql.DB, orderID int64) (orderObservation, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return orderObservation{}, err
	}
	defer conn.Close()
	var out orderObservation
	if err := conn.QueryRowContext(ctx, `SELECT pg_backend_pid()`).Scan(&out.BackendPID); err != nil {
		return orderObservation{}, err
	}
	err = conn.QueryRowContext(ctx, `
SELECT o.status, o.total_amount::text,
       (SELECT count(*) FROM public.order_lines l WHERE l.order_id = o.order_id)
FROM public.orders o WHERE o.order_id = $1`, orderID).Scan(&out.Status, &out.TotalAmount, &out.LineCount)
	if err != nil {
		return orderObservation{}, err
	}
	return out, nil
}
