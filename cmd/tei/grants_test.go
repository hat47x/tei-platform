package main

import (
	"encoding/json"
	"testing"

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
	raw, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(raw) == "" {
		t.Fatal("the document must be encodable")
	}
}

func TestBuildGrantsDocumentWithoutGrantsIsEmpty(t *testing.T) {
	document := buildGrantsDocument(nil, "")
	if len(document.Grants) != 0 {
		t.Fatalf("no declarations means no grants: %+v", document.Grants)
	}
}

func TestRunGrantsRequiresAWorkspace(t *testing.T) {
	if err := runGrantsCommand("", "", ""); err == nil {
		t.Fatal("a missing workspace must be reported")
	}
}
