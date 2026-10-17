package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/workspace"
)

const lineIDAttribute = "sales.order_line.order_line_id"

func setAttributeTypeCandidate(revision, id, typeID string) Candidate {
	return Candidate{
		Schema:            candidateSchema,
		ID:                "cand-set-attribute-type",
		BaseModelRevision: revision,
		Origin:            Origin{Kind: "studio", Actor: "test"},
		Intent:            "属性 " + id + " の型を " + typeID + " へ付け替える（基の表現は同一）",
		Evidence:          []Evidence{{Source: orderFile, Note: "付け替え対象の属性を宣言している正本source"}},
		Delta:             []Operation{{Op: opSetAttributeType, ID: id, AttributeType: typeID}},
	}
}

func attributeTypeOf(t *testing.T, dir, id string) string {
	t.Helper()
	loaded, err := workspace.Load(dir)
	if err != nil {
		t.Fatalf("accepted sourceを再読込できません: %v", err)
	}
	for _, ns := range loaded.World.Semantics {
		for _, d := range ns.Data {
			for _, a := range d.Attributes {
				if string(a.ID) == id {
					return string(a.TypeRef)
				}
			}
		}
	}
	return ""
}

// 同じ基の表現（std.identifier）を持つ型への付け替えは、全stageを通る。
// acceptまでsourceを書かず、accept後は属性のIDを変えずに型参照だけが変わる。
func TestSetAttributeTypeSwitchesToTypeWithSameRepresentation(t *testing.T) {
	dir := scratchWorkspace(t)
	before := treeDigest(t, dir)
	revision := revisionOfDir(t, dir)
	raw := mustJSON(t, setAttributeTypeCandidate(revision, lineIDAttribute, "sales.order_id"))

	ev, err := evaluate(dir, raw)
	if err != nil {
		t.Fatal(err)
	}
	p := ev.preview
	if !p.Acceptable {
		t.Fatalf("候補が受理可能になりません: %+v", allDiagnostics(p))
	}
	var names []string
	for _, s := range p.Stages {
		names = append(names, s.Name)
		if s.Status != "pass" {
			t.Fatalf("stage %s=%s diag=%+v", s.Name, s.Status, s.Diagnostics)
		}
	}
	if !reflect.DeepEqual(names, wantStages) {
		t.Fatalf("stage順序=%v want %v", names, wantStages)
	}
	if got, want := keysOf(p.SemanticDiff), []string{"changed attribute:" + lineIDAttribute}; !reflect.DeepEqual(got, want) {
		t.Fatalf("semantic差分=%v want %v", got, want)
	}
	if !reflect.DeepEqual(p.ChangedCollections, []string{"semantics"}) {
		t.Fatalf("変わったcollection=%v", p.ChangedCollections)
	}
	if len(p.SourceEdits) != 1 || p.SourceEdits[0].Created || p.SourceEdits[0].Path != orderFile ||
		p.SourceEdits[0].InsertedText != "sales.order_id" || p.SourceEdits[0].RemovedText != "sales.order_line_id" {
		t.Fatalf("edit=%+v", p.SourceEdits)
	}
	assertTreeUnchanged(t, before, treeDigest(t, dir))

	decision, err := accept(dir, raw, acceptOptions{})
	if err != nil || decision.Decision != "accepted" || !decision.SourceChanged {
		t.Fatalf("accept=%+v err=%v", decision, err)
	}
	after := treeDigest(t, dir)
	var changed []string
	for path, digest := range before {
		if after[path] != digest {
			changed = append(changed, path)
		}
	}
	if !reflect.DeepEqual(changed, []string{orderFile}) || len(after) != len(before) {
		t.Fatalf("変わったfile=%v", changed)
	}
	content, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(orderFile)))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "attribute orderLineId type sales.order_id") {
		t.Fatalf("source上の型参照が付け替わっていません:\n%s", content)
	}
	if got := attributeTypeOf(t, dir, lineIDAttribute); got != "sales.order_id" {
		t.Fatalf("再読込後の型=%q", got)
	}
}

func TestSetAttributeTypeRefusesWhatNeedsDataConversionOrIsNotFound(t *testing.T) {
	cases := []struct {
		name        string
		stage, code string
		mutate      func(*Candidate)
	}{
		{"different representation", "resolve", "ATTRIBUTE_TYPE_REPRESENTATION_DIFFERS", func(c *Candidate) {
			c.Delta[0].AttributeType = "std.text"
		}},
		{"undeclared type", "resolve", "ATTRIBUTE_TYPE_NOT_FOUND", func(c *Candidate) {
			c.Delta[0].AttributeType = "sales.missing_type"
		}},
		{"unknown attribute", "resolve", "ATTRIBUTE_NOT_FOUND", func(c *Candidate) {
			c.Delta[0].ID = "sales.order_line.missing"
		}},
		{"unchanged type", "resolve", "ATTRIBUTE_TYPE_UNCHANGED", func(c *Candidate) {
			c.Delta[0].AttributeType = "sales.order_line_id"
		}},
		{"explicit placement", "normalize", "CANDIDATE_NORMALIZE", func(c *Candidate) {
			c.Delta[0].Placement = orderFile
		}},
		{"mixed payload", "normalize", "CANDIDATE_NORMALIZE", func(c *Candidate) {
			c.Delta[0].Data = &DataSpec{}
		}},
		{"missing type", "normalize", "CANDIDATE_NORMALIZE", func(c *Candidate) {
			c.Delta[0].AttributeType = ""
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := scratchWorkspace(t)
			baseline := treeDigest(t, dir)
			c := setAttributeTypeCandidate(revisionOfDir(t, dir), lineIDAttribute, "sales.order_id")
			tc.mutate(&c)
			raw := mustJSON(t, c)
			ev, err := evaluate(dir, raw)
			if err != nil {
				t.Fatal(err)
			}
			if ev.preview.Acceptable {
				t.Fatalf("%s が受理されました", tc.name)
			}
			failing := ""
			for _, s := range ev.preview.Stages {
				if s.Status == "fail" && failing == "" {
					failing = s.Name
				}
			}
			if failing != tc.stage || !hasCode(diagnosticCodes(ev.preview), tc.code) {
				t.Fatalf("stage=%q codes=%v want stage=%q code=%s", failing, diagnosticCodes(ev.preview), tc.stage, tc.code)
			}
			decision, err := accept(dir, raw, acceptOptions{})
			if err != nil || decision.Decision != "refused" || decision.SourceChanged {
				t.Fatalf("拒否されるべき候補がsourceを変えました: %+v err=%v", decision, err)
			}
			assertTreeUnchanged(t, baseline, treeDigest(t, dir))
		})
	}
}
