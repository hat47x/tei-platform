package access_test

import (
	"testing"

	"tei/internal/access"
	"tei/internal/permission"
)

func TestParseGrantsReadsTheDocumentShape(t *testing.T) {
	data := []byte(`{"grants":[{"kind":"filesystem","access":"write","scope":"/var/reports"},{"kind":"network","access":"connect","scope":"db:5432"}]}`)
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
	if _, err := access.ParseGrants([]byte(`{"grants":[{"kind":"cpu","access":"read","scope":"x"}]}`)); err == nil {
		t.Fatal("an unknown kind must be rejected")
	}
}

func TestParseGrantsRejectsAnEmptyScope(t *testing.T) {
	if _, err := access.ParseGrants([]byte(`{"grants":[{"kind":"filesystem","access":"write","scope":""}]}`)); err == nil {
		t.Fatal("an empty scope must be rejected")
	}
}

func TestParseGrantsAcceptsAnEmptyList(t *testing.T) {
	grants, err := access.ParseGrants([]byte(`{"grants":[]}`))
	if err != nil || len(grants) != 0 {
		t.Fatalf("an empty list is valid and covers nothing: %+v err=%v", grants, err)
	}
}
