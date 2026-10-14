package legacyauth

import (
	"testing"
	"time"
)

func TestUserAccessDenialAtBoundaries(t *testing.T) {
	now := time.Date(2026, time.October, 12, 16, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		user User
		want string
	}{
		{"通常", User{}, ""},
		{"無効化", User{Disabled: true}, "account_disabled"},
		{"開始前", User{ValidFrom: now.Add(time.Nanosecond)}, "not_yet_valid"},
		{"開始時刻", User{ValidFrom: now}, ""},
		{"終了直前", User{ValidUntil: now.Add(time.Nanosecond)}, ""},
		{"終了時刻", User{ValidUntil: now}, "expired"},
		{"ロック中", User{LockedUntil: now.Add(time.Nanosecond)}, "account_locked"},
		{"ロック終了時刻", User{LockedUntil: now}, ""},
		{"終了済みと無効化", User{Disabled: true, ValidUntil: now.Add(-time.Second)}, "account_disabled"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.user.AccessDenialAt(now); got != tt.want {
				t.Fatalf("AccessDenialAt = %q, want %q", got, tt.want)
			}
		})
	}
}
