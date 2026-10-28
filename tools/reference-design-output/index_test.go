package designoutput

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestIndexSeparatesDocumentsFromUnqualifiedCode(t *testing.T) {
	for _, withCode := range []bool{false, true} {
		t.Run(map[bool]string{false: "documents", true: "with-code"}[withCode], func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "bundle")
			if withCode {
				if _, err := WriteWithGenerator(referenceWorkspace(t), out, "postgres", generatorContract(t), "template", "7"); err != nil {
					t.Fatal(err)
				}
			} else if _, err := Write(referenceWorkspace(t), out, "postgres"); err != nil {
				t.Fatal(err)
			}
			page := string(readTree(t, out)["index.html"])
			for _, want := range []string{"bundle全体の承認を意味しません", "受理済みWorldの宣言から導出した文書・DDL", "business-cross-view", "#1125"} {
				if !strings.Contains(page, want) {
					t.Fatalf("missing boundary %q", want)
				}
			}
			marker := "<h2>未選抜コード候補と参照用契約</h2>"
			parts := strings.Split(page, marker)
			if withCode {
				if len(parts) != 2 {
					t.Fatal("candidate section missing or duplicated")
				}
				for _, path := range []string{"generated/main.go", "generated/candidate.json", "generated/contract.json"} {
					link := `<a href="` + path + `">`
					if strings.Contains(parts[0], link) || strings.Count(parts[1], link) != 1 {
						t.Fatalf("candidate %s not separate from documents", path)
					}
				}
				if !strings.Contains(parts[0], `<a href="table-definitions.md">`) || strings.Contains(parts[1], `<a href="table-definitions.md">`) {
					t.Fatal("document misplaced")
				}
				for _, want := range []string{"build・実行・選抜していません", "reference-local", "業務意味の受理", "宣言版は生成器の現在の資格を示しません"} {
					if !strings.Contains(parts[1], want) {
						t.Fatalf("missing candidate boundary %q", want)
					}
				}
			} else if len(parts) != 1 || strings.Contains(page, "generated/") {
				t.Fatal("document-only bundle advertises code candidates")
			}
			for _, forbidden := range []string{"<script", "<form", "ZgotmplZ"} {
				if strings.Contains(page, forbidden) {
					t.Fatalf("unexpected active content or invalid link: %s", forbidden)
				}
			}
		})
	}
}

func TestIndexKeepsUnknownKindsAndEscapesUnresolvedText(t *testing.T) {
	page := string(renderIndex(Manifest{
		SourceRevision: "source-v1",
		Artifacts: []Artifact{
			{Name: "table", Kind: "table-definition-document", Path: "table.md", Digest: "sha256:doc", StructureRevision: "structure-v1", Unresolved: []string{`<script>alert("unresolved")</script>`}},
			{Name: "future artifact", Kind: "future-kind", Path: "future.json", Digest: "sha256:future"},
		},
	}))
	if !strings.Contains(page, "未解決：") || !strings.Contains(page, "&lt;script&gt;") || strings.Contains(page, "<script>") {
		t.Fatal("unresolved text absent or not escaped")
	}
	if !strings.Contains(page, "structure-v1") || !strings.Contains(page, "sha256:doc") {
		t.Fatal("artifact provenance missing")
	}
	parts := strings.Split(page, "<h2>その他の成果物（種別を確認してください）</h2>")
	if len(parts) != 2 || strings.Contains(parts[0], `href="future.json"`) || !strings.Contains(parts[1], `href="future.json"`) {
		t.Fatal("unknown kind silently classified as a document or candidate")
	}
}
