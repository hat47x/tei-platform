package main

import (
	"database/sql"
	"strings"
	"testing"
)

func TestNewFixtureOwnershipMarkerHasConstrainedFormat(t *testing.T) {
	marker, err := newFixtureOwnershipMarker()
	if err != nil {
		t.Fatal(err)
	}
	if err := validateFixtureOwnershipMarker(marker); err != nil {
		t.Fatalf("generated markerを検証できません: %v", err)
	}
	if !strings.HasPrefix(marker, fixtureOwnershipPrefix) {
		t.Fatalf("marker=%q", marker)
	}
}

func TestValidateFixtureOwnershipMarkerRejectsMalformedValues(t *testing.T) {
	for _, marker := range []string{
		"",
		"other:0123456789abcdef0123456789abcdef",
		fixtureOwnershipPrefix + "short",
		fixtureOwnershipPrefix + "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
	} {
		if err := validateFixtureOwnershipMarker(marker); err == nil {
			t.Fatalf("不正markerを受理しました: %q", marker)
		}
	}
}

func TestVerifyObservedFixtureOwnershipRequiresExactMarker(t *testing.T) {
	const expected = fixtureOwnershipPrefix + "0123456789abcdef0123456789abcdef"

	if err := verifyObservedFixtureOwnership(sql.NullString{String: expected, Valid: true}, expected); err != nil {
		t.Fatalf("一致markerを拒否しました: %v", err)
	}
	if err := verifyObservedFixtureOwnership(sql.NullString{}, expected); err == nil {
		t.Fatal("未観測markerを受理しました")
	}
	if err := verifyObservedFixtureOwnership(sql.NullString{String: fixtureOwnershipPrefix + "fedcba9876543210fedcba9876543210", Valid: true}, expected); err == nil {
		t.Fatal("別markerを受理しました")
	}
}

func TestFixtureOwnershipCommentSQLAcceptsOnlyGeneratedMarkerShape(t *testing.T) {
	const marker = fixtureOwnershipPrefix + "0123456789abcdef0123456789abcdef"
	query, err := fixtureOwnershipCommentSQL(marker)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(query, marker) || !strings.HasPrefix(query, "COMMENT ON TABLE public.order_lines") {
		t.Fatalf("comment SQL=%q", query)
	}

	if _, err := fixtureOwnershipCommentSQL("x'; DROP TABLE public.order_lines; --"); err == nil {
		t.Fatal("任意文字列をCOMMENT SQLへ受理しました")
	}
}

func TestOwnershipCleanupSQLLocksBeforeMarkerReadAndDrop(t *testing.T) {
	if !strings.Contains(fixtureLockSQL, "ACCESS EXCLUSIVE") {
		t.Fatalf("fixtureLockSQL=%q", fixtureLockSQL)
	}
	if !strings.Contains(fixtureMarkerQuery, "obj_description") {
		t.Fatalf("fixtureMarkerQuery=%q", fixtureMarkerQuery)
	}
	if strings.TrimSpace(fixtureDropSQL) != "DROP TABLE public.order_lines;" {
		t.Fatalf("fixtureDropSQL=%q", fixtureDropSQL)
	}
}
