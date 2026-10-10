package trust

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const (
	GrantSchema           = "tei.reference.authority-trust-grant/v0"
	QualificationSchema   = "tei.reference.authority-trust-qualification/v0"
	Canonicalization      = "json-struct-field-order/v0"
)

type Issuer struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type Statement struct {
	AuthorityClass       string `json:"authorityClass"`
	Scope                string `json:"scope"`
	Issuer               Issuer `json:"issuer"`
	StatementFingerprint string `json:"statementFingerprint"`
}

type Grant struct {
	Schema            string `json:"schema"`
	Canonicalization  string `json:"canonicalization"`
	AuthorityClass    string `json:"authorityClass"`
	Scope             string `json:"scope"`
	Issuer            Issuer `json:"issuer"`
	PolicyFingerprint string `json:"policyFingerprint"`
	Active            bool   `json:"active"`
	GrantFingerprint  string `json:"grantFingerprint"`
}

type Qualification struct {
	Schema                   string    `json:"schema"`
	Statement                Statement `json:"statement"`
	GrantFingerprint         string    `json:"grantFingerprint"`
	PolicyFingerprint        string    `json:"policyFingerprint"`
	QualificationFingerprint string    `json:"qualificationFingerprint"`
}

func NewGrant(
	authorityClass string,
	scope string,
	issuer Issuer,
	policyFingerprint string,
	active bool,
) (Grant, error) {
	grant := Grant{
		Schema:            GrantSchema,
		Canonicalization:  Canonicalization,
		AuthorityClass:    strings.TrimSpace(authorityClass),
		Scope:             strings.TrimSpace(scope),
		Issuer: Issuer{
			ID:      strings.TrimSpace(issuer.ID),
			Version: strings.TrimSpace(issuer.Version),
		},
		PolicyFingerprint: strings.TrimSpace(policyFingerprint),
		Active:            active,
	}
	grant.GrantFingerprint = GrantFingerprint(grant)
	if err := ValidateGrant(grant); err != nil {
		return Grant{}, err
	}
	return grant, nil
}

func Verify(
	statement Statement,
	grants []Grant,
) (Qualification, error) {
	normalized, err := normalizeStatement(statement)
	if err != nil {
		return Qualification{}, err
	}

	var matching []Grant
	for i, grant := range grants {
		if err := ValidateGrant(grant); err != nil {
			return Qualification{}, fmt.Errorf("trust grant[%d]: %w", i, err)
		}
		if grant.AuthorityClass == normalized.AuthorityClass &&
			grant.Scope == normalized.Scope &&
			grant.Issuer == normalized.Issuer {
			matching = append(matching, grant)
		}
	}

	switch len(matching) {
	case 0:
		return Qualification{}, fmt.Errorf(
			"no current trust grant for issuer %s@%s authority %s scope %s",
			normalized.Issuer.ID,
			normalized.Issuer.Version,
			normalized.AuthorityClass,
			normalized.Scope,
		)
	case 1:
	default:
		return Qualification{}, fmt.Errorf(
			"trust grant is ambiguous for issuer %s@%s authority %s scope %s",
			normalized.Issuer.ID,
			normalized.Issuer.Version,
			normalized.AuthorityClass,
			normalized.Scope,
		)
	}

	grant := matching[0]
	if !grant.Active {
		return Qualification{}, fmt.Errorf(
			"trust grant is revoked for issuer %s@%s authority %s scope %s",
			normalized.Issuer.ID,
			normalized.Issuer.Version,
			normalized.AuthorityClass,
			normalized.Scope,
		)
	}

	qualification := Qualification{
		Schema:            QualificationSchema,
		Statement:         normalized,
		GrantFingerprint:  grant.GrantFingerprint,
		PolicyFingerprint: grant.PolicyFingerprint,
	}
	qualification.QualificationFingerprint = qualificationFingerprint(qualification)
	return qualification, nil
}

func ValidateGrant(grant Grant) error {
	if grant.Schema != GrantSchema {
		return fmt.Errorf("trust grant schema %q is not supported", grant.Schema)
	}
	if grant.Canonicalization != Canonicalization {
		return fmt.Errorf(
			"trust grant canonicalization %q is not supported",
			grant.Canonicalization,
		)
	}
	if strings.TrimSpace(grant.AuthorityClass) == "" ||
		grant.AuthorityClass != strings.TrimSpace(grant.AuthorityClass) {
		return fmt.Errorf("trust grant authority class is invalid")
	}
	if strings.TrimSpace(grant.Scope) == "" ||
		grant.Scope != strings.TrimSpace(grant.Scope) {
		return fmt.Errorf("trust grant scope is invalid")
	}
	if err := validateIssuer(grant.Issuer); err != nil {
		return fmt.Errorf("trust grant issuer: %w", err)
	}
	if !validSHA256(grant.PolicyFingerprint) {
		return fmt.Errorf("trust grant policy fingerprint is invalid")
	}
	if !validSHA256(grant.GrantFingerprint) ||
		GrantFingerprint(grant) != grant.GrantFingerprint {
		return fmt.Errorf("trust grant fingerprint is invalid or stale")
	}
	return nil
}

func GrantFingerprint(grant Grant) string {
	grant.GrantFingerprint = ""
	return fingerprint(grant)
}

func QualificationFingerprint(value Qualification) string {
	value.QualificationFingerprint = ""
	return fingerprint(value)
}

func normalizeStatement(statement Statement) (Statement, error) {
	statement.AuthorityClass = strings.TrimSpace(statement.AuthorityClass)
	statement.Scope = strings.TrimSpace(statement.Scope)
	statement.Issuer.ID = strings.TrimSpace(statement.Issuer.ID)
	statement.Issuer.Version = strings.TrimSpace(statement.Issuer.Version)
	statement.StatementFingerprint = strings.TrimSpace(statement.StatementFingerprint)

	if statement.AuthorityClass == "" {
		return Statement{}, fmt.Errorf("authority statement class is empty")
	}
	if statement.Scope == "" {
		return Statement{}, fmt.Errorf("authority statement scope is empty")
	}
	if err := validateIssuer(statement.Issuer); err != nil {
		return Statement{}, fmt.Errorf("authority statement issuer: %w", err)
	}
	if !validSHA256(statement.StatementFingerprint) {
		return Statement{}, fmt.Errorf("authority statement fingerprint is invalid")
	}
	return statement, nil
}

func validateIssuer(issuer Issuer) error {
	if strings.TrimSpace(issuer.ID) == "" ||
		issuer.ID != strings.TrimSpace(issuer.ID) {
		return fmt.Errorf("issuer ID is invalid")
	}
	if strings.TrimSpace(issuer.Version) == "" ||
		issuer.Version != strings.TrimSpace(issuer.Version) {
		return fmt.Errorf("issuer version is invalid")
	}
	if _, err := parseVersion(issuer.Version); err != nil {
		return err
	}
	return nil
}

func parseVersion(value string) ([3]int, error) {
	var out [3]int
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return out, fmt.Errorf("version %q must be major.minor.patch", value)
	}
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return out, fmt.Errorf("version %q contains invalid component", value)
		}
		out[i] = n
	}
	return out, nil
}

func validSHA256(value string) bool {
	if !strings.HasPrefix(value, "sha256:") {
		return false
	}
	raw := strings.TrimPrefix(value, "sha256:")
	if len(raw) != 64 {
		return false
	}
	_, err := hex.DecodeString(raw)
	return err == nil
}

func fingerprint(value any) string {
	raw, _ := json.Marshal(value)
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}
