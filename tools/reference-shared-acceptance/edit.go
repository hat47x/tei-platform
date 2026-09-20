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

// SourceEditは、authoritative sourceへ入る変更を、適用前後のfingerprintとともに記録する。
type SourceEdit struct {
	Path              string `json:"path"`
	BeforeFingerprint string `json:"beforeFingerprint"`
	AfterFingerprint  string `json:"afterFingerprint"`
	InsertedText      string `json:"insertedText"`

	before []byte
	after  []byte
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

// planEditsは、deltaをauthoritative sourceへのtext追記へ決定論的に変換する。
func planEdits(s snapshot, ops []Operation) ([]SourceEdit, []Diagnostic) {
	fingerprints := s.fingerprints()
	byPath := map[string][]Operation{}
	var diagnostics []Diagnostic
	for _, op := range ops {
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
		byPath[path] = append(byPath[path], op)
	}
	if len(diagnostics) > 0 {
		return nil, diagnostics
	}

	paths := make([]string, 0, len(byPath))
	for path := range byPath {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	var edits []SourceEdit
	for _, path := range paths {
		before, err := os.ReadFile(filepath.Join(s.root, filepath.FromSlash(path)))
		if err != nil {
			return nil, []Diagnostic{{Code: "SOURCE_UNREADABLE", Message: err.Error(), Subject: "file:" + path}}
		}
		after := before
		for _, op := range byPath[path] {
			after = appendDeclaration(after, renderValidation(op))
		}
		original := strings.TrimRight(string(before), "\r\n\t ")
		edits = append(edits, SourceEdit{
			Path:              path,
			BeforeFingerprint: sourceindex.Fingerprint(before),
			AfterFingerprint:  sourceindex.Fingerprint(after),
			InsertedText:      string(after)[len(original):],
			before:            before,
			after:             after,
		})
	}
	return edits, nil
}
