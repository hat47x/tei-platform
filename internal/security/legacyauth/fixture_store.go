package legacyauth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// FixtureStoreは、既存schemaを模したGit管理可能な非secret fixtureを読むReference実装である。
// production用の認証Database adapterではなく、legacy fieldを標準User schemaへ変換せず
// Authentication adapterへ渡せることを検証するために使う。
type FixtureStore struct {
	sourcePath string
	users      map[string]User
}

type fixtureDocument struct {
	Schema string        `json:"schema"`
	Users  []fixtureUser `json:"users"`
}

type fixtureUser struct {
	UserID            string `json:"user_id"`
	SubjectID         string `json:"subject_id"`
	CredentialRef     string `json:"credential_ref"`
	Disabled          bool   `json:"disabled"`
	ValidFrom         string `json:"valid_from"`
	ValidUntil        string `json:"valid_until"`
	LoginFailureCount int    `json:"login_failure_count"`
	LockedUntil       string `json:"locked_until"`
	OrganizationCode  string `json:"organization_code"`
	RoleCode          string `json:"role_code"`
}

// LoadFixtureStoreは、legacy field名を保ったJSON fixtureを読み込む。
// fixtureにcredential値やpassword hashを置く必要はなく、credential_refだけを扱う。
func LoadFixtureStore(path string) (*FixtureStore, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("legacy authentication fixtureを読み込めません: %w", err)
	}
	var doc fixtureDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("legacy authentication fixtureを解析できません: %w", err)
	}
	if doc.Schema == "" {
		return nil, fmt.Errorf("legacy authentication fixtureにschema識別子がありません")
	}

	store := &FixtureStore{sourcePath: path, users: map[string]User{}}
	for _, src := range doc.Users {
		if src.UserID == "" {
			return nil, fmt.Errorf("legacy authentication fixtureにuser_idのないrecordがあります")
		}
		if _, exists := store.users[src.UserID]; exists {
			return nil, fmt.Errorf("legacy authentication fixtureにuser_id %sが重複しています", src.UserID)
		}
		validFrom, err := parseFixtureTime(src.ValidFrom, "valid_from", src.UserID)
		if err != nil {
			return nil, err
		}
		validUntil, err := parseFixtureTime(src.ValidUntil, "valid_until", src.UserID)
		if err != nil {
			return nil, err
		}
		lockedUntil, err := parseFixtureTime(src.LockedUntil, "locked_until", src.UserID)
		if err != nil {
			return nil, err
		}
		store.users[src.UserID] = User{
			LoginID:           src.UserID,
			SubjectID:         src.SubjectID,
			CredentialRef:     src.CredentialRef,
			Disabled:          src.Disabled,
			ValidFrom:         validFrom,
			ValidUntil:        validUntil,
			LoginFailureCount: src.LoginFailureCount,
			LockedUntil:       lockedUntil,
			OrganizationCode:  src.OrganizationCode,
			RoleCode:          src.RoleCode,
		}
	}
	return store, nil
}

func (s *FixtureStore) FindByLogin(_ context.Context, login string) (User, bool, error) {
	if s == nil {
		return User{}, false, fmt.Errorf("legacy authentication fixture storeがnilです")
	}
	user, ok := s.users[login]
	return user, ok, nil
}

// SourcePathは、Reference Storeがどのlegacy fixtureを正本として読んだかを返す。
// secret値ではなく、mappingの来歴をConformanceから追跡するための情報である。
func (s *FixtureStore) SourcePath() string {
	if s == nil {
		return ""
	}
	return s.sourcePath
}

func parseFixtureTime(value, field, userID string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("legacy user %sの%sをRFC3339時刻として解析できません: %w", userID, field, err)
	}
	return parsed, nil
}
