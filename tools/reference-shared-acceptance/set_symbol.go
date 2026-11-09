package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tei/internal/syntax"
)

// set-symbolは、既存semantic subjectのsymbol（source上の表示名）だけを変更する型付き操作である。
// Canonical IDは変えない。symbolから導かれるsegmentがIDの最後のsegmentと一致する場合にだけ許可する
// ため、改名しても同じsubjectのままであり、removed + addedの別identityへ分裂しない。
//
// 対象はsymbolを持つ次のsubject kindに限る。data / attribute / relation / candidate-key /
// functional-dependencyは#1804でsymbolを持つようになったidentityである。type / capability /
// validationは対象外とし、必要になった時点で同じ境界へ追加する。
func symbolSubjectKindValid(kind string) bool {
	switch kind {
	case "data", "attribute", "relation", "candidate-key", "functional-dependency":
		return true
	}
	return false
}

func lastIDSegment(id string) string {
	if i := strings.LastIndex(id, "."); i >= 0 {
		return id[i+1:]
	}
	return ""
}

// normalizeSetSymbolは、set-symbolの意味を正規化し、identifierを変える候補を拒否する。
// ここでsegmentの一致を確認することで、「symbolだけの改名」と「subject IDの変更」を混同しない。
func normalizeSetSymbol(op Operation) (Operation, error) {
	if err := onlyFields(op, "kind", "id", "symbol"); err != nil {
		return op, err
	}
	if strings.TrimSpace(op.Placement) != "" {
		return op, fmt.Errorf("%s はplacementを受け付けません（対象subjectを宣言した既存fileだけを変更します）", op.Op)
	}
	kind := strings.TrimSpace(op.Kind)
	if !symbolSubjectKindValid(kind) {
		return op, fmt.Errorf("%s のkind %q は data / attribute / relation / candidate-key / functional-dependency のいずれかである必要があります", op.Op, op.Kind)
	}
	id, err := canonical("id", op.ID)
	if err != nil {
		return op, err
	}
	symbol := strings.TrimSpace(op.Symbol)
	segment, err := segmentOf(symbol)
	if err != nil {
		return op, fmt.Errorf("symbol: %w", err)
	}
	last := lastIDSegment(string(id))
	if last == "" {
		return op, fmt.Errorf("id %s から最後のsegmentを取り出せません", id)
	}
	if segment != last {
		return op, fmt.Errorf("symbol %q はsegment %q を導き、id %s のsegment %q と異なるためidentifier（Canonical ID）を変えてしまいます。%s はIDを変えない改名だけを扱います", symbol, segment, id, last, op.Op)
	}
	return Operation{Op: op.Op, Kind: kind, ID: string(id), Symbol: symbol}, nil
}

// symbolSubjectは、kindとCanonical IDで指定した既存subjectの現在のsymbolと、
// それを所有するDataのIDを返す。type / capability / validationは扱わない。
func symbolSubject(s snapshot, kind, id string) (symbol, owner string, ok bool) {
	for _, ns := range s.loaded.World.Semantics {
		for _, d := range ns.Data {
			if kind == "data" {
				if string(d.ID) == id {
					return d.Symbol, string(d.ID), true
				}
				continue
			}
			switch kind {
			case "attribute":
				for _, a := range d.Attributes {
					if string(a.ID) == id {
						return a.Symbol, string(d.ID), true
					}
				}
			case "relation":
				for _, r := range d.Relations {
					if string(r.ID) == id {
						return r.Symbol, string(d.ID), true
					}
				}
			case "candidate-key":
				for _, k := range d.CandidateKeys {
					if string(k.ID) == id {
						return k.Symbol, string(d.ID), true
					}
				}
			case "functional-dependency":
				for _, fd := range d.FunctionalDependencies {
					if string(fd.ID) == id {
						return fd.Symbol, string(d.ID), true
					}
				}
			}
		}
	}
	return "", "", false
}

// symbolAnchorは、parsed semantic sourceの中でkind / idに対応するsymbolトークンのbyte範囲を返す。
// anchorはSource Indexと同じ`semantic-<kind>`のdefinition anchorであり、行の途中の類似名や
// 同じsymbolを持つ別subjectを対象にしない。ちょうど一件でなければ拒否する。
func symbolAnchor(content []byte, kind, id string) (syntax.Anchor, error) {
	parsed, err := syntax.ParseSemanticFileAnchored(string(content))
	if err != nil {
		return syntax.Anchor{}, err
	}
	want := "semantic-" + kind
	var found []syntax.Anchor
	for _, a := range parsed.Anchors {
		if a.Kind == want && a.ID == id {
			found = append(found, a.Anchor)
		}
	}
	if len(found) != 1 {
		return syntax.Anchor{}, fmt.Errorf("%s %s のdeclaration anchorが%d件あります", want, id, len(found))
	}
	return found[0], nil
}

// resolveSetSymbolは、対象subjectの実在・symbolが実際に変わること・所有source上の
// declaration位置を、sourceを書き換える前に確認する。
func resolveSetSymbol(s snapshot, op Operation) []Diagnostic {
	key := operationKey(op)
	fail := func(code, format string, args ...any) []Diagnostic {
		return []Diagnostic{{Code: code, Message: fmt.Sprintf(format, args...), Subject: key}}
	}
	symbol, owner, ok := symbolSubject(s, op.Kind, op.ID)
	if !ok {
		return fail("SYMBOL_SUBJECT_NOT_FOUND", "%s %s は既存のCanonical Worldにありません", op.Kind, op.ID)
	}
	if symbol == op.Symbol {
		return fail("SYMBOL_UNCHANGED", "%s %s のsymbolは既に %s です（改名ではありません）", op.Kind, op.ID, op.Symbol)
	}
	location, ok := s.loaded.Sources["semantic-data:"+owner]
	if !ok {
		return fail("SYMBOL_SOURCE_NOT_FOUND", "%s %s を宣言した正本sourceを特定できません", op.Kind, op.ID)
	}
	content, err := os.ReadFile(filepath.Join(s.root, filepath.FromSlash(location.File)))
	if err != nil {
		return fail("SYMBOL_SOURCE_UNREADABLE", "%s を読めません: %v", location.File, err)
	}
	anchor, err := symbolAnchor(content, op.Kind, op.ID)
	if err != nil {
		return fail("SYMBOL_SOURCE_ANCHOR_MISSING", "%s %s の宣言位置を %s 上で一意に特定できません: %v", op.Kind, op.ID, location.File, err)
	}
	if current := string(content[anchor.Start.Offset:anchor.End.Offset]); current != symbol {
		return fail("SYMBOL_SOURCE_MISMATCH", "%s %s のsource上のsymbol %q がCanonical Worldの %q と一致しません", op.Kind, op.ID, current, symbol)
	}
	return nil
}

// replaceSymbolTokenは、対象symbolトークン一件だけを新しいsymbolへ置き換える。
// 周囲のbyteは変更せず、置き換えた元のsymbolを返す。呼出し側は同じfileへ複数の
// set-symbolを適用するたびにcontentを渡し直し、単調に伸縮するoffsetを引きずらない。
func replaceSymbolToken(content []byte, op Operation) ([]byte, string, error) {
	anchor, err := symbolAnchor(content, op.Kind, op.ID)
	if err != nil {
		return nil, "", fmt.Errorf("%s %s のsymbolトークンを特定できません: %w", op.Kind, op.ID, err)
	}
	start, end := anchor.Start.Offset, anchor.End.Offset
	if start < 0 || end > len(content) || start >= end {
		return nil, "", fmt.Errorf("%s %s のsymbol範囲が不正です", op.Kind, op.ID)
	}
	removed := string(content[start:end])
	if removed == op.Symbol {
		return nil, "", fmt.Errorf("%s %s のsymbolは既に %s です", op.Kind, op.ID, op.Symbol)
	}
	after := make([]byte, 0, len(content)-len(removed)+len(op.Symbol))
	after = append(after, content[:start]...)
	after = append(after, op.Symbol...)
	after = append(after, content[end:]...)
	return after, removed, nil
}
