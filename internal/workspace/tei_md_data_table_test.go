package workspace

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/sourceindex"
	"tei/internal/validation"
)

// These tests read a Data item table through the product Load. The expected
// meaning is the same Data written in the native grammar and loaded by the
// native loader; positions are found with bytes.Index on the document.

const teiMDDataTableDocument = "semantic/organization/transfer-request.tei.md"

func copyTEIMDDataTableWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join("testdata", "tei-md-data-table")
	err := filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(root, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n")), 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func dataOf(result Result, id string) (semantic.Data, bool) {
	for _, ns := range result.World.Semantics {
		for _, data := range ns.Data {
			if string(data.ID) == id {
				return data, true
			}
		}
	}
	return semantic.Data{}, false
}

const transferRequestNative = `namespace organization

data TransferRequest {
    attribute transferRequestId type std.identifier display "異動申請ID" required
    attribute personId type std.identifier description "所属を変える人" display "対象者" required
    attribute toOrgUnitId type std.identifier display "異動先" required
    attribute allocation type std.decimal description "兼務のときの割合。主務だけなら1" display "配分" required
    attribute effectiveFrom type std.datetime description "この日から新しい所属になる" display "発効日" required
    attribute effectiveTo type std.datetime description "決まっていないときは値を持たない" display "終了日"
    attribute reason type std.text display "理由"
}

validation organization.transfer_request.allocation_positive positive {
    attribute organization.transfer_request.allocation
}
`

func TestTEIMDDataTableYieldsTheNativeDataMeaning(t *testing.T) {
	root := copyTEIMDDataTableWorkspace(t)
	loaded, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if diagnostics := validation.Validate(loaded.World); len(diagnostics) != 0 {
		t.Fatalf("validator rejects the table: %+v", diagnostics)
	}

	reference := t.TempDir()
	writeTEIMDTestFile(t, reference, "tei.workspace.yaml", "workspace: tei_md_data_table\nformatVersion: 1\nsources:\n  - semantic\n")
	writeTEIMDTestFile(t, reference, "semantic/organization/transfer-request.tei", transferRequestNative)
	expected, err := Load(reference)
	if err != nil {
		t.Fatal(err)
	}

	got, found := dataOf(loaded, "organization.transfer_request")
	want, wantFound := dataOf(expected, "organization.transfer_request")
	if !found || !wantFound || !reflect.DeepEqual(got, want) {
		t.Fatalf("table Data differs from the native declaration:\n got=%+v\nwant=%+v", got, want)
	}
	if !reflect.DeepEqual(validationsOf(loaded, "organization"), validationsOf(expected, "organization")) {
		t.Fatal("the rule next to the table differs from the native validation")
	}

	// The unmarked example table is documentation.
	for _, attribute := range got.Attributes {
		if strings.Contains(attribute.Symbol, "対象者") || strings.Contains(string(attribute.ID), "担当") {
			t.Fatalf("an unmarked table became a definition: %+v", attribute)
		}
	}
	wantRead := []AuthoringSourceRead{{
		Path: teiMDDataTableDocument,
		Language: AuthoringLanguageBinding{
			LanguageID: "tei.authoring", ContractVersion: "v1", ImplementationID: "tei.builtin.authoring-markdown",
		},
		Regions: 2, Definitions: 2,
	}}
	if !reflect.DeepEqual(loaded.AuthoringSources, wantRead) {
		t.Fatalf("read record=%+v, want %+v", loaded.AuthoringSources, wantRead)
	}

	// Data and attributes are findable at the cells they were written in.
	index, err := BuildSourceIndexFromLoad(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	document := readTEIMDTestFile(t, root, teiMDDataTableDocument)
	for kind, pairs := range map[string]map[string]string{
		"semantic-data":      {"organization.transfer_request": "TransferRequest in"},
		"semantic-attribute": {"organization.transfer_request.effective_to": "effectiveTo |", "organization.transfer_request.reason": "reason |"},
	} {
		for id, text := range pairs {
			if bytes.Count(document, []byte(text)) != 1 {
				t.Fatalf("fixture must write %q once", text)
			}
			refs := index.ReferencesFor(sourceindex.Subject{Kind: kind, ID: id})
			if len(refs) != 1 || refs[0].ArtifactPath != teiMDDataTableDocument ||
				refs[0].Range.Start.Offset != bytes.Index(document, []byte(text)) {
				t.Fatalf("%s %s reference=%+v", kind, id, refs)
			}
		}
	}
}

func TestTEIMDDataTableLanguageSelection(t *testing.T) {
	t.Run("the fence-only language refuses the table instead of skipping it", func(t *testing.T) {
		root := copyTEIMDDataTableWorkspace(t)
		rewriteTEIMDTestFile(t, root, "tei.workspace.yaml", "tei.authoring/v1", "tei.rule-authoring/v0")
		loaded, err := Load(root)
		if err == nil || !strings.Contains(err.Error(), "comment marks are not read") {
			_, found := dataOf(loaded, "organization.transfer_request")
			t.Fatalf("err=%v, table read=%v", err, found)
		}
	})

	t.Run("selecting both languages is refused", func(t *testing.T) {
		root := copyTEIMDDataTableWorkspace(t)
		rewriteTEIMDTestFile(t, root, "tei.workspace.yaml", "  - tei.authoring/v1\n", "  - tei.authoring/v1\n  - tei.rule-authoring/v0\n")
		if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "TEI_MD_PARSER_AMBIGUOUS") {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("a fence-only document means the same under both languages", func(t *testing.T) {
		older, err := Load(copyTEIMDRuleWorkspace(t))
		if err != nil {
			t.Fatal(err)
		}
		root := copyTEIMDRuleWorkspace(t)
		rewriteTEIMDTestFile(t, root, "tei.workspace.yaml", "tei.rule-authoring/v0", "tei.authoring/v1")
		newer, err := Load(root)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(older.World, newer.World) {
			t.Fatal("moving a fence-only Workspace to the newer language changed its World")
		}
	})
}

func TestTEIMDDataTableNegatives(t *testing.T) {
	t.Run("Data already declared natively", func(t *testing.T) {
		root := copyTEIMDDataTableWorkspace(t)
		writeTEIMDTestFile(t, root, "semantic/organization/model.tei",
			"namespace organization\n\ndata TransferRequest {\n    attribute other type std.text\n}\n")
		_, err := Load(root)
		if err == nil || !strings.Contains(err.Error(), "already declared at semantic/organization/model.tei") {
			t.Fatalf("two declarations of one Data loaded: %v", err)
		}
	})

	t.Run("Data declared by two tables", func(t *testing.T) {
		root := copyTEIMDDataTableWorkspace(t)
		writeTEIMDTestFile(t, root, "semantic/organization/zz-copy.tei.md",
			"<!-- tei: data TransferRequest in organization -->\n| attribute | type |\n| --- | --- |\n| other | std.text |\n")
		_, err := Load(root)
		if err == nil || !strings.Contains(err.Error(), "already declared at "+teiMDDataTableDocument) {
			t.Fatalf("the later table was accepted: %v", err)
		}
	})

	t.Run("a bad table stops the whole document", func(t *testing.T) {
		root := copyTEIMDDataTableWorkspace(t)
		rewriteTEIMDTestFile(t, root, teiMDDataTableDocument, "| reason | std.text |", "| reason |  |")
		loaded, err := Load(root)
		if err == nil {
			t.Fatal("a table with an empty type cell loaded")
		}
		if len(validationsOf(loaded, "organization")) != 0 {
			t.Fatal("the rule of a refused document contributed meaning")
		}
	})

	t.Run("unknown type is diagnosed at the row", func(t *testing.T) {
		root := copyTEIMDDataTableWorkspace(t)
		rewriteTEIMDTestFile(t, root, teiMDDataTableDocument, "| reason | std.text |", "| reason | std.txt |")
		loaded, err := Load(root)
		if err != nil {
			t.Fatal(err)
		}
		var codes []string
		for _, diagnostic := range validation.Validate(loaded.World) {
			codes = append(codes, diagnostic.Code)
		}
		if len(codes) != 1 || codes[0] != "UNKNOWN_SEMANTIC_ATTRIBUTE_TYPE" {
			t.Fatalf("codes=%v", codes)
		}
	})
}
