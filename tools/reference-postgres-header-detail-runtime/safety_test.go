package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateDedicatedFixtureDatabaseAcceptsNamedEmptyDatabase(t *testing.T) {
	if err := validateDedicatedFixtureDatabase("tei_gate_b_evidence", 0, 0); err != nil {
		t.Fatalf("専用空Databaseを拒否しました: %v", err)
	}
}

func TestValidateDedicatedFixtureDatabaseRejectsDefaultPostgresDatabase(t *testing.T) {
	err := validateDedicatedFixtureDatabase("postgres", 0, 0)
	if err == nil || !strings.Contains(err.Error(), "既定Database") {
		t.Fatalf("postgres既定DatabaseをEvidence fixture先として受け入れています: %v", err)
	}
}

func TestValidateDedicatedFixtureDatabaseRejectsExistingUserRelations(t *testing.T) {
	err := validateDedicatedFixtureDatabase("tei_gate_b_evidence", 1, 0)
	if err == nil || !strings.Contains(err.Error(), "既存user relation") {
		t.Fatalf("既存user relationのあるDatabaseを空DBとして受け入れています: %v", err)
	}
}

func TestValidateDedicatedFixtureDatabaseRejectsEventTriggers(t *testing.T) {
	err := validateDedicatedFixtureDatabase("tei_gate_b_evidence", 0, 1)
	if err == nil || !strings.Contains(err.Error(), "Event Trigger") {
		t.Fatalf("Event TriggerのあるDatabaseをDDL fixture先として受け入れています: %v", err)
	}
}

func TestValidateDedicatedFixtureDatabaseErrorsDoNotExposeDatabaseIdentity(t *testing.T) {
	cases := []struct {
		name          string
		databaseName  string
		userRelations int
		eventTriggers int
	}{
		{name: "default database", databaseName: "postgres", userRelations: 0, eventTriggers: 0},
		{name: "existing relation", databaseName: "sensitive_evidence_database_name", userRelations: 1, eventTriggers: 0},
		{name: "event trigger", databaseName: "sensitive_evidence_database_name", userRelations: 0, eventTriggers: 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateDedicatedFixtureDatabase(tc.databaseName, tc.userRelations, tc.eventTriggers)
			if err == nil {
				t.Fatal("安全条件違反を受理しています")
			}
			if strings.Contains(err.Error(), tc.databaseName) {
				t.Fatalf("安全判定errorへactual Database identityを露出しています: %v", err)
			}
		})
	}
}

func TestValidateDedicatedFixtureDatabaseRejectsInvalidCounts(t *testing.T) {
	cases := []struct {
		name          string
		userRelations int
		eventTriggers int
	}{
		{name: "negative user relation count", userRelations: -1, eventTriggers: 0},
		{name: "negative event trigger count", userRelations: 0, eventTriggers: -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateDedicatedFixtureDatabase("tei_gate_b_evidence", tc.userRelations, tc.eventTriggers); err == nil {
				t.Fatalf("不正な安全判定件数を受け入れています: userRelations=%d eventTriggers=%d", tc.userRelations, tc.eventTriggers)
			}
		})
	}
}

func TestValidateDedicatedFixtureDatabaseRejectsEmptyIdentity(t *testing.T) {
	err := validateDedicatedFixtureDatabase("   ", 0, 0)
	if err == nil || !strings.Contains(err.Error(), "identityが空") {
		t.Fatalf("identity不明のDatabaseを受け入れています: %v", err)
	}
}

func TestValidateFixtureDatabaseRestoredRequiresZeroUserRelationsAndEventTriggers(t *testing.T) {
	if err := validateFixtureDatabaseRestored(0, 0); err != nil {
		t.Fatalf("user relation / Event Triggerが0件へ復元したDatabaseを拒否しました: %v", err)
	}

	cases := []struct {
		name          string
		userRelations int
		eventTriggers int
	}{
		{name: "negative user relation count", userRelations: -1, eventTriggers: 0},
		{name: "user relation remains", userRelations: 1, eventTriggers: 0},
		{name: "negative event trigger count", userRelations: 0, eventTriggers: -1},
		{name: "event trigger remains", userRelations: 0, eventTriggers: 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateFixtureDatabaseRestored(tc.userRelations, tc.eventTriggers); err == nil {
				t.Fatalf("終了状態を復元済みとして受け入れています: userRelations=%d eventTriggers=%d", tc.userRelations, tc.eventTriggers)
			}
		})
	}
}

func TestFixtureDatabaseSafetyEvidencePersistsAssertionsWithoutDatabaseIdentity(t *testing.T) {
	got := fixtureDatabaseSafetyEvidence{
		GuardPassed:                     true,
		DatabaseIdentityVerified:        true,
		DatabaseIdentityPersisted:       false,
		NonDefaultDatabaseVerified:      true,
		UserRelationsBeforeFixture:      0,
		EventTriggersBeforeFixture:      0,
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
		`"eventTriggersBeforeFixture":0`,
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
		EventTriggersAfterCleanup:       0,
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
		`"eventTriggersAfterCleanup":0`,
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
