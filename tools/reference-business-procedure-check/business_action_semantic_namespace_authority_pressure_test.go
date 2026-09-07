package main

import (
	"path/filepath"
	"testing"

	"tei/internal/semantic"
	"tei/internal/workspace"
)

func TestSemanticNamespaceLensUsesExactOwnerInsteadOfDataIDPrefix(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}

	changed := loaded.World
	changed.Semantics = append([]semantic.Namespace(nil), loaded.World.Semantics...)
	changedOwner := false
	for i := range changed.Semantics {
		if string(changed.Semantics[i].ID) == "sales" {
			changed.Semantics[i].ID = "commercial"
			changedOwner = true
		}
	}
	if !changedOwner {
		t.Fatal("test setup: sales namespaceがありません")
	}

	namespaceID, ok := findSemanticNamespaceForDataExactlyOnceProbe(changed, "sales.order")
	if !ok {
		t.Fatal("sales.order owner Namespaceをexactに解決できません")
	}
	if namespaceID != "commercial" {
		t.Fatalf("Data ID prefixからNamespaceを推測しました: got=%s want=commercial", namespaceID)
	}
}

func TestSemanticNamespaceLensFailsClosedWhenDataIdentityHasMultipleOwners(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}

	changed := loaded.World
	changed.Semantics = append([]semantic.Namespace(nil), loaded.World.Semantics...)

	var duplicate semantic.Data
	foundData := false
	for _, namespace := range loaded.World.Semantics {
		for _, data := range namespace.Data {
			if string(data.ID) == "sales.order" {
				duplicate = data
				foundData = true
			}
		}
	}
	if !foundData {
		t.Fatal("test setup: sales.orderがありません")
	}

	addedDuplicate := false
	for i := range changed.Semantics {
		if string(changed.Semantics[i].ID) == "customer" {
			changed.Semantics[i].Data = append([]semantic.Data(nil), changed.Semantics[i].Data...)
			changed.Semantics[i].Data = append(changed.Semantics[i].Data, duplicate)
			addedDuplicate = true
		}
	}
	if !addedDuplicate {
		t.Fatal("test setup: customer namespaceがありません")
	}

	if namespaceID, ok := findSemanticNamespaceForDataExactlyOnceProbe(changed, "sales.order"); ok {
		t.Fatalf("duplicate Data identityを一つのNamespaceへ暗黙解決しました: %s", namespaceID)
	}
}
