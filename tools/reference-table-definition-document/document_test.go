package tabledoc

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/database/persistence"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func world() ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{{ID: "s",
			Types: []semantic.Type{{ID: "t.code", Symbol: "Code", Kind: semantic.TypeAlias}},
			Data: []semantic.Data{
				{ID: "s.order", Symbol: "Order", Attributes: []semantic.Attribute{
					{ID: "s.order.no", Symbol: "no", TypeRef: "t.code"},
					{ID: "s.order.memo", Symbol: "memo|note", TypeRef: "t.text"},
				}, CandidateKeys: []semantic.CandidateKey{{ID: "s.order.k", Symbol: "byNo",
					Attributes: []core.CanonicalID{"s.order.no"}}},
					Relations: []semantic.Relation{{ID: "s.order.cust", Symbol: "cust", Target: "s.customer",
						Cardinality: semantic.CardinalityOne}}},
				{ID: "s.line", Symbol: "Line"},
			}}},
		Persistence: []persistence.Mapping{
			{ID: "m.order", Database: "db", SemanticData: "s.order", Table: "orders",
				Columns:    []persistence.ColumnMapping{{Attribute: "s.order.no", Column: "order_no", Storage: persistence.StorageVarchar, Nullability: persistence.Required}},
				PrimaryKey: []core.CanonicalID{"s.order.no"}},
			{ID: "m.ghost", Database: "db", SemanticData: "s.ghost", Table: "ghost"},
		},
	}
}

func TestDeterministicAndOrderIndependent(t *testing.T) {
	a, err := Render(world())
	if err != nil {
		t.Fatal(err)
	}
	w := world()
	d := w.Semantics[0].Data
	d[0], d[1] = d[1], d[0]
	b, _ := Render(w)
	// The structure revision identifies the source bytes, so a reordered
	// source has a different revision; the rest of the document is equal.
	body := func(d Document) string { return strings.ReplaceAll(d.Markdown, d.StructureRevision, "") }
	if body(a) != body(b) {
		t.Fatal("output depends on input order")
	}
	c, _ := Render(world())
	if a.Digest != c.Digest || a.StructureRevision != c.StructureRevision {
		t.Fatal("render is not repeatable")
	}
}

func TestRecordsRevisionAndDerivedNotice(t *testing.T) {
	doc, _ := Render(world())
	if !strings.Contains(doc.Markdown, doc.StructureRevision) {
		t.Fatal("revision missing from document")
	}
	if !strings.Contains(doc.Markdown, "意味の正本には反映されません") {
		t.Fatal("derived notice missing")
	}
}

func TestUnresolvedAreListedNotHidden(t *testing.T) {
	doc, _ := Render(world())
	got := strings.Join(doc.Unresolved, "\n")
	for _, want := range []string{"t.text", "s.customer", "s.ghost"} {
		if !strings.Contains(got, want) {
			t.Fatalf("unresolved %q missing: %v", want, doc.Unresolved)
		}
	}
	if !strings.Contains(doc.Markdown, "## 未解決の事項") {
		t.Fatal("unresolved section missing")
	}
}

func TestTableCellsEscapeAuthoredText(t *testing.T) {
	doc, _ := Render(world())
	if !strings.Contains(doc.Markdown, `memo\|note`) {
		t.Fatal("pipe in symbol not escaped")
	}
}

func TestPhysicalAndLogicalStayDistinct(t *testing.T) {
	doc, _ := Render(world())
	md := doc.Markdown
	if !strings.Contains(md, "候補キー（業務上の識別）") || !strings.Contains(md, "主キー: `s.order.no`") {
		t.Fatal("logical and physical keys not both shown")
	}
	if !strings.Contains(md, "物理表への対応は宣言されていません") {
		t.Fatal("Data without mapping not stated")
	}
}
