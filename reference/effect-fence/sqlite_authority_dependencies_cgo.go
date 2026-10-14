//go:build tei_sqlite_reference && cgo

package effectfence

import (
	"errors"
	"fmt"
	"math"
	"strconv"
)

// authorityDependencyState は同一SQLiteファイル内の模擬認可根拠の版を表す。
// 実システムのrole/subject linkを読み取るProviderではない。
type authorityDependencyState struct {
	version uint64
	allowed bool
}

var authorityDimensions = []string{"subject-link", "membership", "policy"}

func validAuthorityDimension(d string) bool {
	for _, known := range authorityDimensions {
		if d == known {
			return true
		}
	}
	return false
}

// RequireAuthorityDependencies は試験用SQLite接続で全3種類の版照合を必須にする。
// 一度有効にすると無効化できず、欠落した版条件を安全側に拒否する。
func (f *sqliteFixture) RequireAuthorityDependencies() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requireEpochs = true
}

// SetAuthorityDependency は権限根拠の改版／失効を同一DB内で確定する試験用操作。
// versionは必ず増加する。外部認可正本との同期は表さない。
func (f *sqliteFixture) SetAuthorityDependency(s Scope, dimension string, version uint64, allowed bool) error {
	if !s.valid() || !validAuthorityDimension(dimension) || version == 0 || version > math.MaxInt64 {
		return errors.New("認可根拠の範囲・種類・版が不正です")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	_, err := f.transaction(func() (Result, error) {
		where := "scope=" + scopeSQL(s) + " AND dimension=" + textSQL(dimension)
		row, exists, err := f.query("SELECT version FROM authority_dependencies WHERE " + where)
		if err != nil {
			return Result{}, err
		}
		if exists {
			prior, err := parseVersion(row[0])
			if err != nil {
				return Result{}, err
			}
			if version <= prior {
				return Result{}, errors.New("認可根拠の版は単調増加させてください")
			}
		}
		v := 0
		if allowed {
			v = 1
		}
		stmt := fmt.Sprintf("INSERT INTO authority_dependencies(scope,dimension,version,allowed) VALUES(%s,%s,%d,%d) ON CONFLICT(scope,dimension) DO UPDATE SET version=excluded.version,allowed=excluded.allowed", scopeSQL(s), textSQL(dimension), version, v)
		return Result{}, f.exec(stmt)
	})
	return err
}

// readAuthorityDependencies は呼出元が保持するSQLite connection mutexの下で使用。
// これだけでは失効・保存の原子性は成立しない。Commit内で再照合が必要。
func (f *sqliteFixture) readAuthorityDependencies(s Scope) (map[string]authorityDependencyState, bool, error) {
	found := make(map[string]authorityDependencyState, len(authorityDimensions))
	for _, dim := range authorityDimensions {
		q := "SELECT version,allowed FROM authority_dependencies WHERE scope=" + scopeSQL(s) + " AND dimension=" + textSQL(dim)
		row, exists, err := f.query(q)
		if err != nil {
			return nil, false, err
		}
		if !exists {
			return nil, false, nil
		}
		version, err := strconv.ParseUint(row[0], 10, 63)
		if err != nil || version == 0 {
			return nil, false, fmt.Errorf("認可根拠の版が不正です (%s)", dim)
		}
		if row[1] != "1" {
			return nil, false, nil
		}
		found[dim] = authorityDependencyState{version: version, allowed: true}
	}
	return found, true, nil
}

// authorityDependenciesMatch はBEGIN IMMEDIATEからCOMMITまでの確定境界内で呼ぶ。
// 失効や版更新のほか、欠落した根拠と予期しない値も拒否する。
func (f *sqliteFixture) authorityDependenciesMatch(s Scope, snapshot map[string]authorityDependencyState) (bool, error) {
	if len(snapshot) != len(authorityDimensions) {
		return false, nil
	}
	current, active, err := f.readAuthorityDependencies(s)
	if err != nil || !active {
		return false, err
	}
	for _, dim := range authorityDimensions {
		saved, ok := snapshot[dim]
		if !ok || saved != current[dim] {
			return false, nil
		}
	}
	return true, nil
}
