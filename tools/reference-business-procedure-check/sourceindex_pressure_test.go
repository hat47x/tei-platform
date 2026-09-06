package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/sourceindex"
)

func TestSourceIndexArtifactFingerprintSurvivesPureFileMove(t *testing.T) {
	root := repoPath()
	fixturePath := filepath.Join(root, "reference", "business-system-design", "order-entry-follow-up-procedure.json")
	raw, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}

	before, err := sourceindex.NewArtifact(
		"reference/business-system-design/order-entry-follow-up-procedure.json",
		raw,
	)
	if err != nil {
		t.Fatal(err)
	}
	after, err := sourceindex.NewArtifact(
		"reference/business-system-design/renamed-follow-up-procedure.json",
		raw,
	)
	if err != nil {
		t.Fatal(err)
	}

	if before.Path == after.Path {
		t.Fatal("test setupがartifact pathを変更できていません")
	}
	if before.Fingerprint != after.Fingerprint {
		t.Fatalf("内容非変更のfile moveでSource Index fingerprintが変化しました: before=%s after=%s", before.Fingerprint, after.Fingerprint)
	}
}

func TestSourceIndexArtifactFingerprintIsContentIdentityNotSemanticIdentity(t *testing.T) {
	root := repoPath()
	fixturePath := filepath.Join(root, "reference", "business-system-design", "order-entry-follow-up-procedure.json")
	raw, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}

	formattedDifferently := append(bytes.Clone(raw), '\n')
	beforeFixture, err := decodeFixture(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	afterFixture, err := decodeFixture(bytes.NewReader(formattedDifferently))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(beforeFixture, afterFixture) {
		t.Fatalf("whitespace-only changeでfixture meaningが変化しました: before=%+v after=%+v", beforeFixture, afterFixture)
	}

	before, err := sourceindex.NewArtifact(
		"reference/business-system-design/order-entry-follow-up-procedure.json",
		raw,
	)
	if err != nil {
		t.Fatal(err)
	}
	after, err := sourceindex.NewArtifact(
		"reference/business-system-design/order-entry-follow-up-procedure.json",
		formattedDifferently,
	)
	if err != nil {
		t.Fatal(err)
	}

	if before.Fingerprint == after.Fingerprint {
		t.Fatal("contentが変化したのにSource Index fingerprintが同一です")
	}
}

func TestCanonicalPatternSubjectRemainsPathIndependentAcrossProcedureArtifacts(t *testing.T) {
	subject := sourceindex.Subject{Kind: "application-pattern", ID: "order_entry.order_record"}
	index := sourceindex.Index{References: []sourceindex.Reference{
		{
			Subject:      subject,
			ArtifactPath: "reference/business-system-design/order-entry-business-procedure.json",
			Role:         sourceindex.RoleDerived,
		},
		{
			Subject:      subject,
			ArtifactPath: "reference/business-system-design/order-entry-follow-up-procedure.json",
			Role:         sourceindex.RoleDerived,
		},
	}}

	refs := index.ReferencesFor(subject)
	if len(refs) != 2 {
		t.Fatalf("同じpath-independent Pattern subjectへの2つのProcedure artifact referenceを取得できません: %+v", refs)
	}
	if refs[0].ArtifactPath == refs[1].ArtifactPath {
		t.Fatalf("異なるProcedure artifactを区別できません: %+v", refs)
	}
}
