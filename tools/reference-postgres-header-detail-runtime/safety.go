package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type fixtureDatabaseSafetyEvidence struct {
	GuardPassed                     bool `json:"guardPassed"`
	DatabaseIdentityVerified        bool `json:"databaseIdentityVerified"`
	DatabaseIdentityPersisted       bool `json:"databaseIdentityPersisted"`
	NonDefaultDatabaseVerified      bool `json:"nonDefaultDatabaseVerified"`
	UserRelationsBeforeFixture      int  `json:"userRelationsBeforeFixture"`
	EventTriggersBeforeFixture      int  `json:"eventTriggersBeforeFixture"`
	FixtureObjectAbsentBeforeCreate bool `json:"fixtureObjectAbsentBeforeCreate"`
}

type fixtureCleanupSafetyEvidence struct {
	Verified                        bool `json:"verified"`
	UserRelationsAfterCleanup       int  `json:"userRelationsAfterCleanup"`
	EventTriggersAfterCleanup       int  `json:"eventTriggersAfterCleanup"`
	FixtureObjectAbsentAfterCleanup bool `json:"fixtureObjectAbsentAfterCleanup"`
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

// ensureDedicatedEmptyFixtureDatabaseは、Evidence toolが既存業務Databaseへfixtureを作成しないため、
// 接続先が第一ReferenceのEvidence専用空Databaseとして扱える最小条件を確認する。
//
// Database名だけで安全性を推測せず、既定管理Databaseを拒否したうえで、system / temporary領域を除く
// user relationとEvent Triggerが一つもないことを要求する。既存objectを削除して空にする処理は行わない。
// Event Triggerはrelationが0件でもDDLへ介入できるため、fixtureのCREATE / DROPより前に明示確認する。
//
// 成功時は、接続先Database名そのものを保持せず、安全判定の非機密な結果だけを返す。
func ensureDedicatedEmptyFixtureDatabase(ctx context.Context, db *sql.DB) (fixtureDatabaseSafetyEvidence, error) {
	result := fixtureDatabaseSafetyEvidence{}

	var databaseName string
	if err := db.QueryRowContext(ctx, `SELECT current_database()`).Scan(&databaseName); err != nil {
		return result, fmt.Errorf("PostgreSQL Evidence接続先Database identityを確認できません: %w", err)
	}

	userRelations, err := countUserRelations(ctx, db)
	if err != nil {
		return result, err
	}
	eventTriggers, err := countEventTriggers(ctx, db)
	if err != nil {
		return result, err
	}

	if err := validateDedicatedFixtureDatabase(databaseName, userRelations, eventTriggers); err != nil {
		return result, err
	}
	result.DatabaseIdentityVerified = true
	result.DatabaseIdentityPersisted = false
	result.NonDefaultDatabaseVerified = true
	result.UserRelationsBeforeFixture = userRelations
	result.EventTriggersBeforeFixture = eventTriggers

	if err := ensureFixtureObjectAbsent(ctx, db); err != nil {
		return result, err
	}
	result.FixtureObjectAbsentBeforeCreate = true
	result.GuardPassed = true
	return result, nil
}

// verifyFixtureDatabaseRestoredは、Provider runtimeとmissing-table Error確認後に、
// toolが接続先を開始時と同じ「user relation 0件 / Event Trigger 0件 / fixture object不在」へ戻せたことを確認する。
// 他主体の並行変更を含め、終了時に対象objectが残っていれば成功Evidenceへ進まない。
func verifyFixtureDatabaseRestored(ctx context.Context, db *sql.DB) (fixtureCleanupSafetyEvidence, error) {
	result := fixtureCleanupSafetyEvidence{}

	userRelations, err := countUserRelations(ctx, db)
	if err != nil {
		return result, fmt.Errorf("PostgreSQL Evidence終了時の既存user relationを確認できません: %w", err)
	}
	result.UserRelationsAfterCleanup = userRelations

	eventTriggers, err := countEventTriggers(ctx, db)
	if err != nil {
		return result, fmt.Errorf("PostgreSQL Evidence終了時のEvent Triggerを確認できません: %w", err)
	}
	result.EventTriggersAfterCleanup = eventTriggers

	if err := validateFixtureDatabaseRestored(userRelations, eventTriggers); err != nil {
		return result, err
	}

	if err := ensureFixtureObjectAbsent(ctx, db); err != nil {
		return result, fmt.Errorf("PostgreSQL Evidence終了時にfixture object不在を確認できません: %w", err)
	}
	result.FixtureObjectAbsentAfterCleanup = true
	result.Verified = true
	return result, nil
}

func countUserRelations(ctx context.Context, db *sql.DB) (int, error) {
	var userRelations int
	if err := db.QueryRowContext(ctx, userRelationCountQuery).Scan(&userRelations); err != nil {
		return 0, fmt.Errorf("PostgreSQL Evidence接続先の既存user relationを確認できません: %w", err)
	}
	return userRelations, nil
}

func countEventTriggers(ctx context.Context, db *sql.DB) (int, error) {
	var eventTriggers int
	if err := db.QueryRowContext(ctx, eventTriggerCountQuery).Scan(&eventTriggers); err != nil {
		return 0, fmt.Errorf("PostgreSQL Evidence接続先のEvent Triggerを確認できません: %w", err)
	}
	return eventTriggers, nil
}

func validateDedicatedFixtureDatabase(databaseName string, userRelations, eventTriggers int) error {
	name := strings.TrimSpace(databaseName)
	if name == "" {
		return fmt.Errorf("安全のためEvidence toolを停止します: 接続先Database identityが空です")
	}
	switch name {
	case "postgres", "template0", "template1":
		return fmt.Errorf("安全のためEvidence toolを停止します: PostgreSQL既定Databaseはfixture実行先にできません。Evidence専用の空Databaseを指定してください")
	}
	if userRelations < 0 {
		return fmt.Errorf("安全のためEvidence toolを停止します: 既存user relation件数が不正です: %d", userRelations)
	}
	if userRelations != 0 {
		return fmt.Errorf("安全のためEvidence toolを停止します: 接続先Databaseには既存user relationが%d件あります。既存資産を変更せず、Evidence専用の空Databaseを指定してください", userRelations)
	}
	if eventTriggers < 0 {
		return fmt.Errorf("安全のためEvidence toolを停止します: Event Trigger件数が不正です: %d", eventTriggers)
	}
	if eventTriggers != 0 {
		return fmt.Errorf("安全のためEvidence toolを停止します: 接続先DatabaseにはEvent Triggerが%d件あります。DDLへ介入し得る既存objectを変更せず、Evidence専用の空Databaseを指定してください", eventTriggers)
	}
	return nil
}

func validateFixtureDatabaseRestored(userRelations, eventTriggers int) error {
	if userRelations < 0 {
		return fmt.Errorf("PostgreSQL Evidence終了時のuser relation件数が不正です: %d", userRelations)
	}
	if userRelations != 0 {
		return fmt.Errorf("PostgreSQL Evidence終了時にuser relationが%d件残っています。tool-owned fixtureまたは並行変更の残存を確認してください", userRelations)
	}
	if eventTriggers < 0 {
		return fmt.Errorf("PostgreSQL Evidence終了時のEvent Trigger件数が不正です: %d", eventTriggers)
	}
	if eventTriggers != 0 {
		return fmt.Errorf("PostgreSQL Evidence終了時にEvent Triggerが%d件存在します。並行変更を含むDDL介入可能なobjectの残存を確認してください", eventTriggers)
	}
	return nil
}
