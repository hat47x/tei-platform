package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tei/internal/sourceindex"
	"tei/internal/syntax"
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

// renderRelationは、data blockの中へ入れるRelation宣言（1〜数行、各行は改行で終わる）を返す。
func renderRelation(r AddRelationSpec, newline string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "    relation %s to %s %s%s", r.Symbol, r.Target, r.Cardinality, newline)
	for _, k := range r.Keys {
		fmt.Fprintf(&b, "        key %s -> %s%s", k.SourceAttribute, k.TargetAttribute, newline)
	}
	return b.String()
}

// insertIntoDataBlockは、既存のdata blockの閉じ括弧の行の直前へ、宣言を挿入する。
// 閉じ括弧が行頭（空白だけの後）にないlayoutは、行の途中へ手探りで挿入せず、未対応として拒否する。
func insertIntoDataBlock(content []byte, dataSymbol string, relation AddRelationSpec) ([]byte, string, error) {
	src := string(content)
	lx := syntax.NewLexer(src)
	var previous, beforePrevious syntax.Token
	closing := -1
	inBlock := false
	for closing < 0 {
		t, err := lx.Next()
		if err != nil {
			return nil, "", err
		}
		if t.Kind == syntax.TokenEOF {
			break
		}
		switch {
		case !inBlock && t.Kind == syntax.TokenLBrace && previous.Kind == syntax.TokenIdent && previous.Text == dataSymbol && beforePrevious.Text == "data":
			inBlock = true
		case inBlock && t.Kind == syntax.TokenRBrace:
			closing = t.Start.Offset
		}
		beforePrevious, previous = previous, t
	}
	if closing < 0 {
		return nil, "", fmt.Errorf("data %s のblockをsource上で特定できません", dataSymbol)
	}
	lineStart := strings.LastIndex(src[:closing], "\n") + 1
	if strings.TrimLeft(src[lineStart:closing], " \t") != "" {
		return nil, "", fmt.Errorf("data %s の閉じ括弧が行頭にないため、挿入位置を決められません", dataSymbol)
	}
	newline := "\n"
	if strings.Contains(src, "\r\n") {
		newline = "\r\n"
	}
	text := renderRelation(relation, newline)
	return []byte(src[:lineStart] + text + src[lineStart:]), text, nil
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

// planEditsは、deltaをauthoritative sourceへのtext追記・挿入・新規file作成へ決定論的に変換する。
// 既存fileへの変更（add-relationのdata block挿入 → add-validationの末尾追記）は、fileごとにこの順で適用する。
func planEdits(s snapshot, ops []Operation) ([]SourceEdit, []Diagnostic) {
	fingerprints := s.fingerprints()
	existing := map[string][]Operation{}
	var edits []SourceEdit
	var diagnostics []Diagnostic

	for _, op := range ops {
		key := operationKey(op)
		if content, isNew := renderNewFile(op); isNew {
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

		var path string
		switch op.Op {
		case opAddRelation:
			location, ok := s.loaded.Sources["semantic-data:"+op.AddRelation.Data]
			if !ok {
				diagnostics = append(diagnostics, Diagnostic{Code: "PLACEMENT_UNRESOLVED", Message: fmt.Sprintf("data %s を宣言したsource fileを特定できません", op.AddRelation.Data), Subject: key})
				continue
			}
			path = location.File
		default:
			path = op.Placement
			if path == "" {
				var err error
				path, err = defaultPlacement(s, op)
				if err != nil {
					diagnostics = append(diagnostics, Diagnostic{Code: "PLACEMENT_UNRESOLVED", Message: err.Error(), Subject: key})
					continue
				}
			}
		}
		path, err := sourceindex.NormalizeWorkspacePath(path)
		if err != nil {
			diagnostics = append(diagnostics, Diagnostic{Code: "PLACEMENT_UNRESOLVED", Message: err.Error(), Subject: key})
			continue
		}
		if _, ok := fingerprints[path]; !ok {
			diagnostics = append(diagnostics, Diagnostic{
				Code:    "PLACEMENT_NOT_MANAGED_SOURCE",
				Message: fmt.Sprintf("placement %s は管理対象sourceではありません", path),
				Subject: key,
			})
			continue
		}
		existing[path] = append(existing[path], op)
	}
	if len(diagnostics) > 0 {
		return nil, diagnostics
	}

	for path, group := range existing {
		before, err := os.ReadFile(filepath.Join(s.root, filepath.FromSlash(path)))
		if err != nil {
			return nil, []Diagnostic{{Code: "SOURCE_UNREADABLE", Message: err.Error(), Subject: "file:" + path}}
		}
		content := before
		var inserted strings.Builder
		for _, op := range group {
			if op.Op != opAddRelation {
				continue
			}
			symbol, ok := dataSymbol(s, op.AddRelation.Data)
			if !ok {
				return nil, []Diagnostic{{Code: "DATA_NOT_FOUND", Message: fmt.Sprintf("data %s は存在しません", op.AddRelation.Data), Subject: operationKey(op)}}
			}
			var text string
			content, text, err = insertIntoDataBlock(content, symbol, *op.AddRelation)
			if err != nil {
				return nil, []Diagnostic{{Code: "DATA_BLOCK_INSERTION_FAILED", Message: err.Error(), Subject: operationKey(op)}}
			}
			inserted.WriteString(text)
		}
		appendBase := strings.TrimRight(string(content), "\r\n\t ")
		for _, op := range group {
			if op.Op == opAddValidation {
				content = appendDeclaration(content, renderValidation(op))
			}
		}
		if len(strings.TrimRight(string(content), "\r\n\t ")) != len(appendBase) {
			inserted.WriteString(string(content)[len(appendBase):])
		}
		edits = append(edits, SourceEdit{
			Path:              path,
			Layer:             layerSemantic,
			BeforeFingerprint: sourceindex.Fingerprint(before),
			AfterFingerprint:  sourceindex.Fingerprint(content),
			InsertedText:      inserted.String(),
			before:            before,
			after:             content,
		})
	}
	sort.Slice(edits, func(i, j int) bool { return edits[i].Path < edits[j].Path })
	return edits, nil
}

func dataSymbol(s snapshot, id string) (string, bool) {
	for _, ns := range s.loaded.World.Semantics {
		for _, d := range ns.Data {
			if string(d.ID) == id {
				return d.Symbol, true
			}
		}
	}
	return "", false
}
