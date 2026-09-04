package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// ensureDedicatedEmptyFixtureDatabaseは、Evidence toolが既存業務Databaseへfixtureを作成しないため、
// 接続先が第一ReferenceのEvidence専用空Databaseとして扱える最小条件を確認する。
//
// Database名だけで安全性を推測せず、既定管理Databaseを拒否したうえで、system / temporary領域を除く
// user relationが一つもないことを要求する。既存relationを削除して空にする処理は行わない。
func ensureDedicatedEmptyFixtureDatabase(ctx context.Context, db *sql.DB) error {
	var databaseName string
	if err := db.QueryRowContext(ctx, `SELECT current_database()`).Scan(&databaseName); err != nil {
		return fmt.Errorf("PostgreSQL Evidence接続先Database identityを確認できません: %w", err)
	}

	var userRelations int
	if err := db.QueryRowContext(ctx, `
SELECT count(*)
FROM pg_catalog.pg_class AS c
JOIN pg_catalog.pg_namespace AS n ON n.oid = c.relnamespace
WHERE n.nspname NOT IN ('pg_catalog', 'information_schema')
  AND n.nspname NOT LIKE 'pg_toast%'
  AND n.nspname NOT LIKE 'pg_temp_%'
  AND c.relkind IN ('r', 'p', 'v', 'm', 'f', 'S')
`).Scan(&userRelations); err != nil {
		return fmt.Errorf("PostgreSQL Evidence接続先の既存user relationを確認できません: %w", err)
	}

	if err := validateDedicatedFixtureDatabase(databaseName, userRelations); err != nil {
		return err
	}
	if err := ensureFixtureObjectAbsent(ctx, db); err != nil {
		return err
	}
	return nil
}

func validateDedicatedFixtureDatabase(databaseName string, userRelations int) error {
	name := strings.TrimSpace(databaseName)
	if name == "" {
		return fmt.Errorf("安全のためEvidence toolを停止します: 接続先Database identityが空です")
	}
	switch name {
	case "postgres", "template0", "template1":
		return fmt.Errorf("安全のためEvidence toolを停止します: PostgreSQL既定Database %qはfixture実行先にできません。Evidence専用の空Databaseを指定してください", name)
	}
	if userRelations < 0 {
		return fmt.Errorf("安全のためEvidence toolを停止します: 既存user relation件数が不正です: %d", userRelations)
	}
	if userRelations != 0 {
		return fmt.Errorf("安全のためEvidence toolを停止します: 接続先Database %qには既存user relationが%d件あります。既存資産を変更せず、Evidence専用の空Databaseを指定してください", name, userRelations)
	}
	return nil
}
