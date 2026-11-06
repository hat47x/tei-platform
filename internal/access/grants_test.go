package access_test

import (
	"testing"

	"tei/internal/access"
	"tei/internal/core"
	"tei/internal/permission"
)

func TestParseGrantsReadsTheDocumentShape(t *testing.T) {
	data := []byte(`{"grants":[{"issuedBy":"environment:production","kind":"filesystem","access":"write","scope":"/var/reports"},{"issuedBy":"environment:production","kind":"network","access":"connect","scope":"db:5432"}]}`)
	grants, err := access.ParseGrants(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(grants) != 2 {
		t.Fatalf("expected two grants: %+v", grants)
	}
	if grants[0].Kind != permission.KindFilesystem || grants[0].Access != permission.AccessWrite || grants[0].Scope != "/var/reports" {
		t.Fatalf("unexpected first grant: %+v", grants[0])
	}
}

func TestParseGrantsRejectsAnUnknownKind(t *testing.T) {
	if _, err := access.ParseGrants([]byte(`{"grants":[{"issuedBy":"environment:prod","kind":"cpu","access":"read","scope":"x"}]}`)); err == nil {
		t.Fatal("an unknown kind must be rejected")
	}
}

func TestParseGrantsRejectsAnEmptyScope(t *testing.T) {
	if _, err := access.ParseGrants([]byte(`{"grants":[{"issuedBy":"environment:prod","kind":"filesystem","access":"write","scope":""}]}`)); err == nil {
		t.Fatal("an empty scope must be rejected")
	}
}

func TestParseGrantsAcceptsAnEmptyList(t *testing.T) {
	grants, err := access.ParseGrants([]byte(`{"grants":[]}`))
	if err != nil || len(grants) != 0 {
		t.Fatalf("an empty list is valid and covers nothing: %+v err=%v", grants, err)
	}
}

func TestEncodeDocumentRefusesAGrantWithoutAnIssuer(t *testing.T) {
	document := access.Document{Grants: []access.Entry{{Kind: "filesystem", Access: "write", Scope: "/var/reports"}}}
	if _, err := access.EncodeDocument(document); err == nil {
		t.Fatal("a grant that names no issuer must not be encoded: nothing would decide it")
	}
}

func TestEncodeDocumentRoundTripsThroughParseDocument(t *testing.T) {
	document := access.Document{Application: "order_entry", Grants: []access.Entry{{
		IssuedBy:       "environment:production",
		CapabilityKind: "deployment-executor",
		CapabilityID:   core.MustCanonicalID("migration.atlas"),
		Kind:           "filesystem",
		Access:         "write",
		Scope:          "/var/work",
	}}}
	raw, err := access.EncodeDocument(document)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	read, err := access.ParseDocument(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(read.Grants) != 1 || read.Grants[0].IssuedBy != "environment:production" ||
		read.Grants[0].CapabilityID != core.MustCanonicalID("migration.atlas") {
		t.Fatalf("the document must survive the round trip: %+v", read)
	}
}

func TestAcceptIssuedByRefusesACapabilitysOwnGrant(t *testing.T) {
	capability := core.MustCanonicalID("tei.frozen_world.lineage")
	document := access.Document{Grants: []access.Entry{{
		IssuedBy: string(capability), Kind: "filesystem", Access: "write", Scope: "/tmp/observed.json",
	}}}
	if err := access.AcceptIssuedBy(document, capability); err == nil {
		t.Fatal("a capability must not accept a grant it issued itself")
	}
}

func TestAcceptIssuedByRefusesAnUnnamedIssuer(t *testing.T) {
	capability := core.MustCanonicalID("tei.frozen_world.lineage")
	document := access.Document{Grants: []access.Entry{{Kind: "filesystem", Access: "write", Scope: "/tmp/observed.json"}}}
	if err := access.AcceptIssuedBy(document, capability); err == nil {
		t.Fatal("a grant with no issuer must be refused")
	}
}

func TestAcceptIssuedByAcceptsAThirdPartyAndCoversNothingByItself(t *testing.T) {
	capability := core.MustCanonicalID("tei.frozen_world.lineage")
	document := access.Document{Grants: []access.Entry{{
		IssuedBy: "script:build-frozen-api", Kind: "filesystem", Access: "write", Scope: "/tmp/observed.json",
	}}}
	if err := access.AcceptIssuedBy(document, capability); err != nil {
		t.Fatalf("a third-party issuance must be accepted by this check: %v", err)
	}
	// The check only judges who issued; it does not grant anything.
	if grants, err := access.ParseGrants([]byte(`{"grants":[]}`)); err != nil || len(grants) != 0 {
		t.Fatalf("an empty list covers nothing: %+v err=%v", grants, err)
	}
}
