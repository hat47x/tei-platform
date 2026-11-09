package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
)

// errDSNMissing means the caller has not supplied a database to check. It is not
// a failure of the check: nothing was attempted.
var errDSNMissing = errors.New("no PostgreSQL DSN was supplied")

// runDBCheck reports whether the database named by the DSN can be used for the
// real-database verifications. It creates and drops a temporary table, which is
// scoped to the session and leaves no trace.
func runDBCheck(ctx context.Context, dsn string, out io.Writer) error {
	if strings.TrimSpace(dsn) == "" {
		return errDSNMissing
	}
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer func() { _ = conn.Close(ctx) }()

	var version string
	if err := conn.QueryRow(ctx, "SELECT version()").Scan(&version); err != nil {
		return fmt.Errorf("read the server version: %w", err)
	}
	fmt.Fprintf(out, "接続: 成功\nサーバー: %s\n", firstLine(version))

	if _, err := conn.Exec(ctx, "CREATE TEMPORARY TABLE tei_dbcheck_probe (id integer)"); err != nil {
		return fmt.Errorf("create a temporary table: %w", err)
	}
	if _, err := conn.Exec(ctx, "DROP TABLE tei_dbcheck_probe"); err != nil {
		return fmt.Errorf("drop the temporary table: %w", err)
	}
	fmt.Fprintln(out, "一時表の作成: 成功")

	fmt.Fprintln(out, "\n実データベースの確認は次で実行します。")
	fmt.Fprintln(out, "  TEI_TEST_POSTGRES_DSN='<同じ接続文字列>' go test ./tools/reference-expected-state-concurrency -count=1 -v")
	return nil
}

// firstLine keeps the version banner on one line.
func firstLine(value string) string {
	if index := strings.IndexByte(value, '\n'); index >= 0 {
		return value[:index]
	}
	return value
}

// dbCheckDSN reads the DSN from the flag or the environment the tests use.
func dbCheckDSN(flagValue string) string {
	if strings.TrimSpace(flagValue) != "" {
		return flagValue
	}
	return os.Getenv("TEI_TEST_POSTGRES_DSN")
}
