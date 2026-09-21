package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tei/internal/sourceindex"
)

const (
	layerSemantic    = "semantic"
	layerApplication = "application"
)

// SourceEditは、authoritative sourceへ入る変更を、適用前後のfingerprintとともに記録する。
// 既存fileへの追記（add-validation）と、新しいfileの作成（Created）がある。
type SourceEdit struct {
	Path              string `json:"path"`
	Layer             string `json:"layer"`
	Created           bool   `json:"created,omitempty"`
	BeforeFingerprint string `json:"beforeFingerprint,omitempty"`
	AfterFingerprint  string `json:"afterFingerprint"`
	InsertedText      string `json:"insertedText"`

	before []byte
	after  []byte
}

func layerOf(op Operation) string {
	switch op.Op {
	case opAddApplication, opAddReferencePattern:
		return layerApplication
	default:
		return layerSemantic
	}
}

func layerPrefix(layer string) string {
	if layer == layerApplication {
		return "applications/"
	}
	return "semantic/"
}

func renderValidation(op Operation) string {
	var b strings.Builder
	fmt.Fprintf(&b, "validation %s %s {\n", op.ID, op.Kind)
	if op.Relation != "" {
		fmt.Fprintf(&b, "    relation %s\n", op.Relation)
	}
	fmt.Fprintf(&b, "    attribute %s\n", op.Attribute)
	b.WriteString("}")
	return b.String()
}

// renderDataは新しいnamespaceのsemantic fileの本文を返す。末尾の改行は含めない。
func renderData(d DataSpec) string {
	var b strings.Builder
	fmt.Fprintf(&b, "namespace %s\n\n", d.Namespace)
	fmt.Fprintf(&b, "data %s {\n", d.Symbol)
	for _, a := range d.Attributes {
		fmt.Fprintf(&b, "    attribute %s type %s\n", a.Symbol, a.Type)
	}
	for _, r := range d.Relations {
		fmt.Fprintf(&b, "    relation %s to %s %s\n", r.Symbol, r.Target, r.Cardinality)
		for _, k := range r.Keys {
			fmt.Fprintf(&b, "        key %s -> %s\n", k.SourceAttribute, k.TargetAttribute)
		}
	}
	b.WriteString("}")
	return b.String()
}

func renderApplication(a ApplicationSpec) string {
	var b strings.Builder
	fmt.Fprintf(&b, "application %s {\n", a.ID)
	for _, u := range a.UsesSemantic {
		fmt.Fprintf(&b, "    uses semantic %s\n", u)
	}
	for _, db := range a.Databases {
		fmt.Fprintf(&b, "\n    database %s {\n", db.Alias)
		fmt.Fprintf(&b, "        target %s\n", db.Target)
		fmt.Fprintf(&b, "        access %s\n", db.Access)
		if len(db.Contracts) > 0 {
			b.WriteString("        requires {\n")
			for _, c := range db.Contracts {
				fmt.Fprintf(&b, "            contract %s ^%d\n", c.Contract, c.Major)
			}
			b.WriteString("        }\n")
		}
		b.WriteString("    }\n")
	}
	b.WriteString("}")
	return b.String()
}

func renderPattern(p PatternSpec) string {
	var b strings.Builder
	fmt.Fprintf(&b, "pattern %s reference {\n", p.ID)
	fmt.Fprintf(&b, "    relation %s\n", p.Relation)
	fmt.Fprintf(&b, "    datasource %s\n", p.Datasource)
	b.WriteString("    contract-query {\n")
	fmt.Fprintf(&b, "        contract %s\n", p.ContractQuery.Contract)
	fmt.Fprintf(&b, "        query %s\n", p.ContractQuery.Query)
	for _, in := range p.ContractQuery.Inputs {
		fmt.Fprintf(&b, "        input %s -> %s\n", in.Attribute, in.Input)
	}
	b.WriteString("    }\n")
	b.WriteString("}")
	return b.String()
}

// renderNewFileは、新しいfileの内容を返す（本文 + 末尾の改行一つ）。
// S4のProposal identityは、この本文から末尾の改行を除いたものである。
func renderNewFile(op Operation) (string, bool) {
	switch op.Op {
	case opAddData:
		return renderData(*op.Data), true
	case opAddApplication:
		return renderApplication(*op.Application), true
	case opAddReferencePattern:
		return renderPattern(*op.Pattern), true
	}
	return "", false
}

// appendDeclarationは、既存fileの改行規則（LF / CRLF）を保ったまま、末尾へ宣言を一つ追加する。
func appendDeclaration(content []byte, block string) []byte {
	newline := "\n"
	if bytes.Contains(content, []byte("\r\n")) {
		newline = "\r\n"
	}
	prefix := strings.TrimRight(string(content), "\r\n\t ")
	suffix := newline + newline + strings.ReplaceAll(block, "\n", newline) + newline
	return []byte(prefix + suffix)
}

// defaultPlacementは、同じnamespaceの既存Validationがあるfile（複数ならpath順で先頭）を選び、
// 無ければnamespaceを最初に宣言したfileを選ぶ。物理的な置き場所は意味のidentityではない。
func defaultPlacement(s snapshot, op Operation) (string, error) {
	var candidates []string
	for _, ns := range s.loaded.World.Semantics {
		if string(ns.ID) != op.Namespace {
			continue
		}
		for _, rule := range ns.Validations {
			if location, ok := s.loaded.Sources["semantic-validation:"+string(rule.ID)]; ok {
				candidates = append(candidates, location.File)
			}
		}
	}
	if len(candidates) == 0 {
		location, ok := s.loaded.Sources["semantic:"+op.Namespace]
		if !ok {
			return "", fmt.Errorf("namespace %s を宣言したsource fileを特定できません", op.Namespace)
		}
		candidates = append(candidates, location.File)
	}
	sort.Strings(candidates)
	return sourceindex.NormalizeWorkspacePath(candidates[0])
}

func fileExists(root, path string) bool {
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(path)))
	return err == nil
}

// planEditsは、deltaをauthoritative sourceへのtext追記・新規file作成へ決定論的に変換する。
func planEdits(s snapshot, ops []Operation) ([]SourceEdit, []Diagnostic) {
	fingerprints := s.fingerprints()
	appends := map[string][]Operation{}
	var edits []SourceEdit
	var diagnostics []Diagnostic

	for _, op := range ops {
		if content, isNew := renderNewFile(op); isNew {
			key := operationKey(op)
			if _, managed := fingerprints[op.Placement]; managed || fileExists(s.root, op.Placement) {
				diagnostics = append(diagnostics, Diagnostic{
					Code:    "DESTINATION_ALREADY_EXISTS",
					Message: fmt.Sprintf("作成先 %s は既に存在します", op.Placement),
					Subject: key,
				})
				continue
			}
			after := []byte(content + "\n")
			edits = append(edits, SourceEdit{
				Path:             op.Placement,
				Layer:            layerOf(op),
				Created:          true,
				AfterFingerprint: sourceindex.Fingerprint(after),
				InsertedText:     string(after),
				after:            after,
			})
			continue
		}

		path := op.Placement
		if path == "" {
			var err error
			path, err = defaultPlacement(s, op)
			if err != nil {
				diagnostics = append(diagnostics, Diagnostic{Code: "PLACEMENT_UNRESOLVED", Message: err.Error(), Subject: "validation:" + op.ID})
				continue
			}
		}
		if _, ok := fingerprints[path]; !ok {
			diagnostics = append(diagnostics, Diagnostic{
				Code:    "PLACEMENT_NOT_MANAGED_SOURCE",
				Message: fmt.Sprintf("placement %s は管理対象sourceではありません", path),
				Subject: "validation:" + op.ID,
			})
			continue
		}
		appends[path] = append(appends[path], op)
	}
	if len(diagnostics) > 0 {
		return nil, diagnostics
	}

	for path, group := range appends {
		before, err := os.ReadFile(filepath.Join(s.root, filepath.FromSlash(path)))
		if err != nil {
			return nil, []Diagnostic{{Code: "SOURCE_UNREADABLE", Message: err.Error(), Subject: "file:" + path}}
		}
		after := before
		for _, op := range group {
			after = appendDeclaration(after, renderValidation(op))
		}
		original := strings.TrimRight(string(before), "\r\n\t ")
		edits = append(edits, SourceEdit{
			Path:              path,
			Layer:             layerSemantic,
			BeforeFingerprint: sourceindex.Fingerprint(before),
			AfterFingerprint:  sourceindex.Fingerprint(after),
			InsertedText:      string(after)[len(original):],
			before:            before,
			after:             after,
		})
	}
	sort.Slice(edits, func(i, j int) bool { return edits[i].Path < edits[j].Path })
	return edits, nil
}
