//go:build tei_sqlite_reference && cgo

package effectfence

import (
	"fmt"
	"strings"
)

// requiredSQLiteLegacyEpochTriggers は提供側が保持する期待DDLである。
// 実際のトリガー定義に変更があれば書込み確定を拒否する。
var requiredSQLiteLegacyEpochTriggers = []string{
	`CREATE TRIGGER legacy_subject_epoch_on_update
AFTER UPDATE OF role, disabled ON legacy_subjects
BEGIN
    SELECT RAISE(ABORT, 'missing membership epoch')
    WHERE NOT EXISTS (
        SELECT 1 FROM authority_dependencies
        WHERE dimension='membership'
        AND json_extract(scope, '$.Subject')=NEW.subject
    );
    SELECT RAISE(ABORT, 'membership epoch overflow')
    WHERE EXISTS (
        SELECT 1 FROM authority_dependencies
        WHERE dimension='membership'
        AND json_extract(scope, '$.Subject')=NEW.subject
        AND version >= 9223372036854775807
    );
    UPDATE authority_dependencies
    SET version=version+1,
        allowed=CASE WHEN NEW.disabled=1 THEN 0 ELSE 1 END
    WHERE dimension='membership'
      AND json_extract(scope, '$.Subject')=NEW.subject;
END`,
	`CREATE TRIGGER legacy_subject_epoch_on_delete
AFTER DELETE ON legacy_subjects
BEGIN
    SELECT RAISE(ABORT, 'missing membership epoch')
    WHERE NOT EXISTS (
        SELECT 1 FROM authority_dependencies
        WHERE dimension='membership'
        AND json_extract(scope, '$.Subject')=OLD.subject
    );
    SELECT RAISE(ABORT, 'membership epoch overflow')
    WHERE EXISTS (
        SELECT 1 FROM authority_dependencies
        WHERE dimension='membership'
        AND json_extract(scope, '$.Subject')=OLD.subject
        AND version >= 9223372036854775807
    );
    UPDATE authority_dependencies
    SET version=version+1, allowed=0
    WHERE dimension='membership'
      AND json_extract(scope, '$.Subject')=OLD.subject;
END`,
	`CREATE TRIGGER legacy_subject_epoch_on_insert
AFTER INSERT ON legacy_subjects
BEGIN
    SELECT RAISE(ABORT, 'missing membership epoch')
    WHERE NOT EXISTS (
        SELECT 1 FROM authority_dependencies
        WHERE dimension='membership'
        AND json_extract(scope, '$.Subject')=NEW.subject
    );
    SELECT RAISE(ABORT, 'membership epoch overflow')
    WHERE EXISTS (
        SELECT 1 FROM authority_dependencies
        WHERE dimension='membership'
        AND json_extract(scope, '$.Subject')=NEW.subject
        AND version >= 9223372036854775807
    );
    UPDATE authority_dependencies
    SET version=version+1,
        allowed=CASE WHEN NEW.disabled=1 THEN 0 ELSE 1 END
    WHERE dimension='membership'
      AND json_extract(scope, '$.Subject')=NEW.subject;
END`,
	`CREATE TRIGGER legacy_subject_epoch_prevent_subject_rekey
BEFORE UPDATE OF subject ON legacy_subjects
BEGIN
    SELECT RAISE(ABORT, 'subject identity is immutable');
END`,
}

// installSQLiteLegacyRoleEpochTriggers は試験用の利用者変更をDBの同一確定境界へ束縛する。
func installSQLiteLegacyRoleEpochTriggers(f *sqliteFixture) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, statement := range requiredSQLiteLegacyEpochTriggers {
		if err := f.exec(statement); err != nil {
			return fmt.Errorf("認可正本の版管理トリガーを設定できません: %w", err)
		}
	}
	if err := f.verifyRequiredSQLiteLegacyEpochTriggers(); err != nil {
		return err
	}
	f.schemaCheck = f.verifyRequiredSQLiteLegacyEpochTriggers
	return nil
}

// verifyRequiredSQLiteLegacyEpochTriggers は、呼出元が保持する接続mutexの下で
// 版を発行するDB側の制約が変更されていないことを確認する。
// 管理者が一時的に無効化して戻す事象や別正本の変更は検知できない。
func (f *sqliteFixture) verifyRequiredSQLiteLegacyEpochTriggers() error {
	for _, expectedSQL := range requiredSQLiteLegacyEpochTriggers {
		parts := strings.Fields(expectedSQL)
		if len(parts) < 3 || parts[0] != "CREATE" || parts[1] != "TRIGGER" {
			return fmt.Errorf("必要な認可正本トリガーの定義が不正です")
		}
		row, exists, err := f.query("SELECT sql FROM sqlite_master WHERE type='trigger' AND name=" + textSQL(parts[2]))
		if err != nil {
			return fmt.Errorf("認可正本の制約を確認できません: %w", err)
		}
		if !exists || len(row) != 1 || strings.TrimSpace(row[0]) != strings.TrimSpace(expectedSQL) {
			return fmt.Errorf("認可正本の制約が欠落または改変されています")
		}
	}
	return nil
}
