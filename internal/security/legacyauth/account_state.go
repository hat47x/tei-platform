package legacyauth

import "time"

// AccessDenialAt は、認証後の処理でも再評価すべき利用者状態を調べる。
// 空文字列は利用可能を表す。役割・所属・対象別の許可はここでは判定しない。
// 判定時点の利用者レコードが正しいこと、失効と書込みの原子性は保証しない。
func (u User) AccessDenialAt(now time.Time) string {
	if u.Disabled {
		return "account_disabled"
	}
	if !u.ValidFrom.IsZero() && now.Before(u.ValidFrom) {
		return "not_yet_valid"
	}
	if !u.ValidUntil.IsZero() && !now.Before(u.ValidUntil) {
		return "expired"
	}
	if !u.LockedUntil.IsZero() && now.Before(u.LockedUntil) {
		return "account_locked"
	}
	return ""
}
