package main

import (
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
