package main

import (
	"testing"

	"tei/internal/access"
	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/permission"
)

func TestBuildGrantsDocumentCollectsAndOrdersGrants(t *testing.T) {
	declared := []environment.Environment{{
		ID: "prod",
		PermissionGrants: []environment.PermissionGrant{
			{CapabilityKind: "deployment-executor", CapabilityID: core.MustCanonicalID("migration.atlas"), Permission: permission.Spec{Kind: permission.KindNetwork, Access: permission.AccessConnect, Scope: "db:5432"}},
			{CapabilityKind: "deployment-executor", CapabilityID: core.MustCanonicalID("migration.atlas"), Permission: permission.Spec{Kind: permission.KindFilesystem, Access: permission.AccessRead, Scope: "/var/work"}},
		},
	}}
	document := buildGrantsDocument(declared, "order_entry")
	if document.Application != "order_entry" || len(document.Grants) != 2 {
		t.Fatalf("unexpected document: %+v", document)
	}
	if document.Grants[0].Kind != "filesystem" || document.Grants[1].Kind != "network" {
		t.Fatalf("grants must be ordered by kind then scope: %+v", document.Grants)
	}
	if document.Grants[0].IssuedBy != "environment:prod" {
		t.Fatalf("a declared grant must name the environment that declared it: %+v", document.Grants[0])
	}
}

func TestBuildGrantsDocumentWithoutGrantsIsEmpty(t *testing.T) {
	document := buildGrantsDocument(nil, "")
	if len(document.Grants) != 0 {
		t.Fatalf("no declarations means no grants: %+v", document.Grants)
	}
	raw, err := access.EncodeDocument(document)
	if err != nil {
		t.Fatalf("an empty document must be encodable: %v", err)
	}
	parsed, err := access.ParseDocument(raw)
	if err != nil || len(parsed.Grants) != 0 {
		t.Fatalf("an empty document must round trip as empty: %+v err=%v", parsed, err)
	}
}

func TestBuildGrantsDocumentIsReadableByTheAdapters(t *testing.T) {
	declared := []environment.Environment{{
		ID: "production",
		PermissionGrants: []environment.PermissionGrant{{
			CapabilityKind: "deployment-executor",
			CapabilityID:   core.MustCanonicalID("migration.atlas"),
			Permission:     permission.Spec{Kind: permission.KindFilesystem, Access: permission.AccessReadWrite, Scope: "/var/work"},
		}},
	}}
	raw, err := access.EncodeDocument(buildGrantsDocument(declared, "order_entry"))
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	grants, err := access.ParseGrants(raw)
	if err != nil {
		t.Fatalf("the writer and the reader must share one format: %v", err)
	}
	if len(grants) != 1 || grants[0].Scope != "/var/work" {
		t.Fatalf("unexpected grants: %+v", grants)
	}
}

func TestRunGrantsRequiresAWorkspace(t *testing.T) {
	if err := runGrantsCommand("", "", ""); err == nil {
		t.Fatal("a missing workspace must be reported")
	}
}
