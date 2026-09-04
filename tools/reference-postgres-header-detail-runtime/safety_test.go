package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateDedicatedFixtureDatabaseAcceptsNamedEmptyDatabase(t *testing.T) {
	if err := validateDedicatedFixtureDatabase("tei_gate_b_evidence", 0); err != nil {
		t.Fatalf("専用空Databaseを拒否しました: %v", err)
	}
}

func TestValidateDedicatedFixtureDatabaseRejectsDefaultPostgresDatabase(t *testing.T) {
	err := validateDedicatedFixtureDatabase("postgres", 0)
	if err == nil || !strings.Contains(err.Error(), "既定Database") {
		t.Fatalf("postgres既定DatabaseをEvidence fixture先として受け入れています: %v", err)
	}
}

func TestValidateDedicatedFixtureDatabaseRejectsExistingUserRelations(t *testing.T) {
	err := validateDedicatedFixtureDatabase("tei_gate_b_evidence", 1)
	if err == nil || !strings.Contains(err.Error(), "既存user relation") {
		t.Fatalf("既存user relationのあるDatabaseを空DBとして受け入れています: %v", err)
	}
}

func TestValidateDedicatedFixtureDatabaseRejectsEmptyIdentity(t *testing.T) {
	err := validateDedicatedFixtureDatabase("   ", 0)
	if err == nil || !strings.Contains(err.Error(), "identityが空") {
		t.Fatalf("identity不明のDatabaseを受け入れています: %v", err)
	}
}

func TestValidateFixtureDatabaseRestoredRequiresZeroUserRelations(t *testing.T) {
	if err := validateFixtureDatabaseRestored(0); err != nil {
		t.Fatalf("user relation 0件へ復元したDatabaseを拒否しました: %v", err)
	}

	for _, got := range []int{-1, 1, 2} {
		if err := validateFixtureDatabaseRestored(got); err == nil {
			t.Fatalf("終了時user relation=%dを復元済みとして受け入れています", got)
		}
	}
}

func TestFixtureDatabaseSafetyEvidencePersistsAssertionsWithoutDatabaseIdentity(t *testing.T) {
	got := fixtureDatabaseSafetyEvidence{
		GuardPassed:                     true,
		DatabaseIdentityVerified:        true,
		DatabaseIdentityPersisted:       false,
		NonDefaultDatabaseVerified:      true,
		UserRelationsBeforeFixture:      0,
		FixtureObjectAbsentBeforeCreate: true,
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	for _, want := range []string{
		`"guardPassed":true`,
		`"databaseIdentityVerified":true`,
		`"databaseIdentityPersisted":false`,
		`"nonDefaultDatabaseVerified":true`,
		`"userRelationsBeforeFixture":0`,
		`"fixtureObjectAbsentBeforeCreate":true`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("安全判定Evidenceに%sがありません: %s", want, text)
		}
	}
	for _, forbidden := range []string{"databaseName", "dsn", "TEI_TEST_POSTGRES_DSN", "tei_gate_b_evidence"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("安全判定Evidenceへ接続先identityまたはDSN情報を保存しています: %s", text)
		}
	}
}

func TestFixtureCleanupSafetyEvidencePersistsRestoredStateWithoutDatabaseIdentity(t *testing.T) {
	got := fixtureCleanupSafetyEvidence{
		Verified:                        true,
		UserRelationsAfterCleanup:       0,
		FixtureObjectAbsentAfterCleanup: true,
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	for _, want := range []string{
		`"verified":true`,
		`"userRelationsAfterCleanup":0`,
		`"fixtureObjectAbsentAfterCleanup":true`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("cleanup安全判定Evidenceに%sがありません: %s", want, text)
		}
	}
	for _, forbidden := range []string{"databaseName", "dsn", "TEI_TEST_POSTGRES_DSN", "tei_gate_b_evidence"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("cleanup安全判定Evidenceへ接続先identityまたはDSN情報を保存しています: %s", text)
		}
	}
}
