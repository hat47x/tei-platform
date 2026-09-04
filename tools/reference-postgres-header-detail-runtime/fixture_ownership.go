package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
)

const fixtureOwnershipPrefix = "tei:reference-postgres-header-detail-runtime:"

const fixtureCreateSQL = `
CREATE TABLE public.order_lines (
    order_line_id bigint PRIMARY KEY,
    order_id bigint NOT NULL,
    line_number bigint NOT NULL,
    item_code varchar(64) NOT NULL,
    quantity numeric(12, 2) NOT NULL,
    unit_price numeric(12, 2) NOT NULL
);
`

const fixtureLockSQL = `LOCK TABLE public.order_lines IN ACCESS EXCLUSIVE MODE;`
const fixtureMarkerQuery = `SELECT pg_catalog.obj_description('public.order_lines'::regclass, 'pg_class')`
const fixtureDropSQL = `DROP TABLE public.order_lines;`

func newFixtureOwnershipMarker() (string, error) {
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("fixture ownership markerを生成できません: %w", err)
	}
	return fixtureOwnershipPrefix + hex.EncodeToString(random[:]), nil
}

func validateFixtureOwnershipMarker(marker string) error {
	if !strings.HasPrefix(marker, fixtureOwnershipPrefix) {
		return fmt.Errorf("fixture ownership markerのprefixが不正です")
	}
	suffix := strings.TrimPrefix(marker, fixtureOwnershipPrefix)
	if len(suffix) != 32 {
		return fmt.Errorf("fixture ownership markerの長さが不正です")
	}
	if _, err := hex.DecodeString(suffix); err != nil {
		return fmt.Errorf("fixture ownership markerのrandom部が16進数ではありません")
	}
	return nil
}

func fixtureOwnershipCommentSQL(marker string) (string, error) {
	if err := validateFixtureOwnershipMarker(marker); err != nil {
		return "", err
	}
	// markerは固定prefix + lowercase hexだけで構成するため、SQL literalへ安全に埋め込める。
	return fmt.Sprintf("COMMENT ON TABLE public.order_lines IS '%s';", marker), nil
}

func createOwnedFixtureTable(ctx context.Context, db *sql.DB) (string, error) {
	marker, err := newFixtureOwnershipMarker()
	if err != nil {
		return "", err
	}
	commentSQL, err := fixtureOwnershipCommentSQL(marker)
	if err != nil {
		return "", err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("header-detail fixture作成transactionを開始できません: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, fixtureCreateSQL); err != nil {
		return "", fmt.Errorf("header-detail用fixture tableを作成できません: %w", err)
	}
	if _, err := tx.ExecContext(ctx, commentSQL); err != nil {
		return "", fmt.Errorf("header-detail fixtureへownership markerを設定できません: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("header-detail fixture作成transactionを確定できません: %w", err)
	}
	return marker, nil
}

func removeOwnedFixtureTable(ctx context.Context, db *sql.DB, expectedMarker string) error {
	if err := validateFixtureOwnershipMarker(expectedMarker); err != nil {
		return fmt.Errorf("安全のためfixtureを削除しません: 期待ownership markerが不正です: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("fixture cleanup transactionを開始できません: %w", err)
	}
	defer tx.Rollback()

	// objectをlockしてからmarkerを照合する。lock取得前に別objectへ置き換わっていれば
	// marker mismatchで停止し、lock取得後はDROPまで同名objectの置換を許さない。
	if _, err := tx.ExecContext(ctx, fixtureLockSQL); err != nil {
		return fmt.Errorf("安全のためfixtureを削除しません: ownership確認用lockを取得できません: %w", err)
	}

	var observed sql.NullString
	if err := tx.QueryRowContext(ctx, fixtureMarkerQuery).Scan(&observed); err != nil {
		return fmt.Errorf("安全のためfixtureを削除しません: DB側ownership markerを確認できません: %w", err)
	}
	if !observed.Valid || observed.String != expectedMarker {
		return fmt.Errorf("安全のためfixtureを削除しません: DB側ownership markerがtool作成時のmarkerと一致しません")
	}

	if _, err := tx.ExecContext(ctx, fixtureDropSQL); err != nil {
		return fmt.Errorf("ownership確認済みheader-detail fixture tableを削除できません: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("fixture cleanup transactionを確定できません: %w", err)
	}
	return nil
}
