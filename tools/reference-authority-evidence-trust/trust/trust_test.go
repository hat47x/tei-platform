package trust

import (
	"strings"
	"testing"
)

func testFingerprint(hex string) string {
	return "sha256:" + strings.Repeat(hex, 64)
}

func testIssuer() Issuer {
	return Issuer{ID: "tei.reference.publisher", Version: "1.0.0"}
}

func testStatement() Statement {
	return Statement{
		AuthorityClass:       "invocation-shape-issuer",
		Scope:                "shape:tei.reference.request/v0",
		Issuer:               testIssuer(),
		StatementFingerprint: testFingerprint("a"),
	}
}

func testGrant(
	t *testing.T,
	authorityClass string,
	scope string,
	issuer Issuer,
	policy string,
	active bool,
) Grant {
	t.Helper()
	grant, err := NewGrant(
		authorityClass,
		scope,
		issuer,
		testFingerprint(policy),
		active,
	)
	if err != nil {
		t.Fatal(err)
	}
	return grant
}

func TestVerifyRequiresExplicitCurrentTrustGrant(t *testing.T) {
	statement := testStatement()
	if _, err := Verify(statement, nil); err == nil ||
		!strings.Contains(err.Error(), "no current trust grant") {
		t.Fatalf("self-asserted issuer was accepted without trust grant: %v", err)
	}
}

func TestVerifyAcceptsExactActiveGrant(t *testing.T) {
	statement := testStatement()
	grant := testGrant(
		t,
		statement.AuthorityClass,
		statement.Scope,
		statement.Issuer,
		"b",
		true,
	)
	qualification, err := Verify(statement, []Grant{grant})
	if err != nil {
		t.Fatal(err)
	}
	if qualification.Statement != statement ||
		qualification.GrantFingerprint != grant.GrantFingerprint ||
		qualification.PolicyFingerprint != grant.PolicyFingerprint ||
		!validSHA256(qualification.QualificationFingerprint) {
		t.Fatalf("qualification=%+v", qualification)
	}
}

func TestVerifyRejectsWrongAuthorityClassAndScope(t *testing.T) {
	statement := testStatement()
	for _, grant := range []Grant{
		testGrant(
			t,
			"semantic-namespace-owner",
			statement.Scope,
			statement.Issuer,
			"b",
			true,
		),
		testGrant(
			t,
			statement.AuthorityClass,
			"shape:other/v0",
			statement.Issuer,
			"b",
			true,
		),
	} {
		if _, err := Verify(statement, []Grant{grant}); err == nil {
			t.Fatalf("wrong trust scope/class was accepted: %+v", grant)
		}
	}
}

func TestVerifyRejectsWrongIssuer(t *testing.T) {
	statement := testStatement()
	grant := testGrant(
		t,
		statement.AuthorityClass,
		statement.Scope,
		Issuer{ID: "tei.reference.other", Version: "1.0.0"},
		"b",
		true,
	)
	if _, err := Verify(statement, []Grant{grant}); err == nil {
		t.Fatal("trust grant for another issuer was accepted")
	}
}

func TestVerifyRejectsRevokedGrantWithoutChangingStatementIdentity(t *testing.T) {
	statement := testStatement()
	grant := testGrant(
		t,
		statement.AuthorityClass,
		statement.Scope,
		statement.Issuer,
		"b",
		false,
	)
	if _, err := Verify(statement, []Grant{grant}); err == nil ||
		!strings.Contains(err.Error(), "revoked") {
		t.Fatalf("revoked trust grant was accepted: %v", err)
	}
	if statement.StatementFingerprint != testFingerprint("a") {
		t.Fatal("current trust status changed historical statement identity")
	}
}

func TestTrustPolicyRefreshChangesQualificationNotStatement(t *testing.T) {
	statement := testStatement()
	left := testGrant(
		t,
		statement.AuthorityClass,
		statement.Scope,
		statement.Issuer,
		"b",
		true,
	)
	right := testGrant(
		t,
		statement.AuthorityClass,
		statement.Scope,
		statement.Issuer,
		"c",
		true,
	)
	leftQualification, err := Verify(statement, []Grant{left})
	if err != nil {
		t.Fatal(err)
	}
	rightQualification, err := Verify(statement, []Grant{right})
	if err != nil {
		t.Fatal(err)
	}
	if leftQualification.Statement != rightQualification.Statement {
		t.Fatal("trust policy refresh changed authority statement identity")
	}
	if leftQualification.QualificationFingerprint ==
		rightQualification.QualificationFingerprint {
		t.Fatal("trust policy refresh was not observable in qualification identity")
	}
}

func TestVerifyRejectsAmbiguousMatchingGrants(t *testing.T) {
	statement := testStatement()
	left := testGrant(
		t,
		statement.AuthorityClass,
		statement.Scope,
		statement.Issuer,
		"b",
		true,
	)
	right := testGrant(
		t,
		statement.AuthorityClass,
		statement.Scope,
		statement.Issuer,
		"c",
		true,
	)
	if _, err := Verify(statement, []Grant{left, right}); err == nil ||
		!strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("ambiguous current trust was accepted: %v", err)
	}
}

func TestUnrelatedGrantDoesNotAffectExactQualification(t *testing.T) {
	statement := testStatement()
	exact := testGrant(
		t,
		statement.AuthorityClass,
		statement.Scope,
		statement.Issuer,
		"b",
		true,
	)
	unrelated := testGrant(
		t,
		"semantic-namespace-owner",
		"namespace:std.validation",
		Issuer{ID: "tei.reference.semantic.publisher", Version: "1.0.0"},
		"c",
		true,
	)
	qualification, err := Verify(statement, []Grant{unrelated, exact})
	if err != nil {
		t.Fatal(err)
	}
	if qualification.GrantFingerprint != exact.GrantFingerprint {
		t.Fatalf("unrelated grant changed qualification: %+v", qualification)
	}
}

func TestTamperedGrantFingerprintFailsClosed(t *testing.T) {
	statement := testStatement()
	grant := testGrant(
		t,
		statement.AuthorityClass,
		statement.Scope,
		statement.Issuer,
		"b",
		true,
	)
	grant.PolicyFingerprint = testFingerprint("d")
	if _, err := Verify(statement, []Grant{grant}); err == nil ||
		!strings.Contains(err.Error(), "invalid or stale") {
		t.Fatalf("tampered trust grant was accepted: %v", err)
	}
}

func TestVerifyRejectsMalformedStatementFingerprint(t *testing.T) {
	statement := testStatement()
	statement.StatementFingerprint = "not-a-fingerprint"
	grant := testGrant(
		t,
		statement.AuthorityClass,
		statement.Scope,
		statement.Issuer,
		"b",
		true,
	)
	if _, err := Verify(statement, []Grant{grant}); err == nil {
		t.Fatal("malformed authority statement was accepted")
	}
}
