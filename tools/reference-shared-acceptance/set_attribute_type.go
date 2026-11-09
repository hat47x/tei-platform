package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tei/internal/semantic"
	"tei/internal/syntax"
)

// set-attribute-typeは、既存の属性が指す型を、同じ基の表現を持つ別の型へ付け替える。
// 属性のCanonical IDは変えない。基の表現が異なる付け替えは、既存データの変換を伴うため
// 受け付けない（意味の変更に伴う既存データの移行設計の「縮小」「再構成」に当たる）。

const opSetAttributeType = "set-attribute-type"

// normalizeSetAttributeTypeは、set-attribute-typeの意味を正規化する。
// 型は完全なCanonical IDで指定する。ソース上の型参照も常に完全なCanonical IDで書かれる。
func normalizeSetAttributeType(op Operation) (Operation, error) {
	if err := onlyFields(op, "id", "attributeType"); err != nil {
		return op, err
	}
	if strings.TrimSpace(op.Placement) != "" {
		return op, fmt.Errorf("%s はplacementを受け付けません（対象属性を宣言した既存fileだけを変更します）", op.Op)
	}
	id, err := canonical("id", op.ID)
	if err != nil {
		return op, err
	}
	typeID, err := canonical("attributeType", op.AttributeType)
	if err != nil {
		return op, err
	}
	return Operation{Op: op.Op, ID: string(id), AttributeType: string(typeID)}, nil
}

// attributeOf は属性とその所有Dataを返す。
func attributeOf(s snapshot, id string) (attr semantic.Attribute, owner string, ok bool) {
	for _, ns := range s.loaded.World.Semantics {
		for _, d := range ns.Data {
			for _, a := range d.Attributes {
				if string(a.ID) == id {
					return a, string(d.ID), true
				}
			}
		}
	}
	return semantic.Attribute{}, "", false
}

// representationRootは、型を基の表現（別名の連鎖の終点）まで辿る。
// 循環や未定義の型は解決できないものとして扱う。
func representationRoot(s snapshot, typeID string) (string, bool) {
	types := map[string]semantic.Type{}
	for _, ns := range s.loaded.World.Semantics {
		for _, t := range ns.Types {
			types[string(t.ID)] = t
		}
	}
	seen := map[string]bool{}
	current := typeID
	for {
		if seen[current] {
			return "", false
		}
		seen[current] = true
		t, ok := types[current]
		if !ok {
			// 標準ライブラリのprimitive型など、Worldに宣言されない型は終点として扱う。
			return current, strings.HasPrefix(current, "std.")
		}
		if t.Kind == semantic.TypePrimitive || string(t.Underlying) == "" {
			return current, true
		}
		current = string(t.Underlying)
	}
}

func typeDeclared(s snapshot, typeID string) bool {
	if strings.HasPrefix(typeID, "std.") {
		return true
	}
	for _, ns := range s.loaded.World.Semantics {
		for _, t := range ns.Types {
			if string(t.ID) == typeID {
				return true
			}
		}
	}
	return false
}

func attributeTypeAnchor(content []byte, id string) (syntax.Anchor, error) {
	parsed, err := syntax.ParseSemanticFileAnchored(string(content))
	if err != nil {
		return syntax.Anchor{}, err
	}
	var found []syntax.Anchor
	for _, a := range parsed.Anchors {
		if a.Kind == "semantic-attribute-type" && a.ID == id {
			found = append(found, a.Anchor)
		}
	}
	if len(found) != 1 {
		return syntax.Anchor{}, fmt.Errorf("型参照のanchorがちょうど一件ではありません（%d件）", len(found))
	}
	return found[0], nil
}

func resolveSetAttributeType(s snapshot, op Operation) []Diagnostic {
	key := operationKey(op)
	fail := func(code, format string, args ...any) []Diagnostic {
		return []Diagnostic{{Code: code, Message: fmt.Sprintf(format, args...), Subject: key}}
	}
	attr, owner, ok := attributeOf(s, op.ID)
	if !ok {
		return fail("ATTRIBUTE_NOT_FOUND", "属性 %s は既存のCanonical Worldにありません", op.ID)
	}
	if string(attr.TypeRef) == op.AttributeType {
		return fail("ATTRIBUTE_TYPE_UNCHANGED", "属性 %s の型は既に %s です", op.ID, op.AttributeType)
	}
	if !typeDeclared(s, op.AttributeType) {
		return fail("ATTRIBUTE_TYPE_NOT_FOUND", "型 %s はCanonical Worldに宣言されていません", op.AttributeType)
	}
	from, okFrom := representationRoot(s, string(attr.TypeRef))
	to, okTo := representationRoot(s, op.AttributeType)
	if !okFrom || !okTo {
		return fail("ATTRIBUTE_TYPE_REPRESENTATION_UNRESOLVED", "型 %s または %s の基の表現を解決できません（循環または未定義）", attr.TypeRef, op.AttributeType)
	}
	if from != to {
		return fail("ATTRIBUTE_TYPE_REPRESENTATION_DIFFERS",
			"型 %s の基の表現 %s は、%s の基の表現 %s と異なります。既存データの変換が要るため、この演算では扱いません", attr.TypeRef, from, op.AttributeType, to)
	}
	location, ok := s.loaded.Sources["semantic-data:"+owner]
	if !ok {
		return fail("ATTRIBUTE_SOURCE_NOT_FOUND", "属性 %s を宣言した正本sourceを特定できません", op.ID)
	}
	content, err := os.ReadFile(filepath.Join(s.root, filepath.FromSlash(location.File)))
	if err != nil {
		return fail("ATTRIBUTE_SOURCE_UNREADABLE", "%s を読めません: %v", location.File, err)
	}
	anchor, err := attributeTypeAnchor(content, op.ID)
	if err != nil {
		return fail("ATTRIBUTE_SOURCE_ANCHOR_MISSING", "属性 %s の型参照を %s 上で一意に特定できません: %v", op.ID, location.File, err)
	}
	if current := string(content[anchor.Start.Offset:anchor.End.Offset]); current != string(attr.TypeRef) {
		return fail("ATTRIBUTE_SOURCE_MISMATCH", "属性 %s のsource上の型 %q がCanonical Worldの %q と一致しません", op.ID, current, attr.TypeRef)
	}
	return nil
}

// replaceAttributeTypeTokenは、対象属性の型参照トークン一件だけを置換する。
func replaceAttributeTypeToken(content []byte, op Operation) ([]byte, string, error) {
	anchor, err := attributeTypeAnchor(content, op.ID)
	if err != nil {
		return nil, "", fmt.Errorf("属性 %s の型参照を特定できません: %w", op.ID, err)
	}
	start, end := anchor.Start.Offset, anchor.End.Offset
	if start < 0 || end > len(content) || start >= end {
		return nil, "", fmt.Errorf("属性 %s の型参照の範囲が不正です", op.ID)
	}
	removed := string(content[start:end])
	if removed == op.AttributeType {
		return nil, "", fmt.Errorf("属性 %s の型は既に %s です", op.ID, op.AttributeType)
	}
	after := make([]byte, 0, len(content)-len(removed)+len(op.AttributeType))
	after = append(after, content[:start]...)
	after = append(after, op.AttributeType...)
	after = append(after, content[end:]...)
	return after, removed, nil
}
