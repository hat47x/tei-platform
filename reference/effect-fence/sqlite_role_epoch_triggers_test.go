//go:build tei_sqlite_reference && cgo

package effectfence

import "fmt"

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
	f.requireSchemaGuard = true
	return nil
}
