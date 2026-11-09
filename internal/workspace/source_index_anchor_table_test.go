package workspace

import (
	"fmt"
	"strings"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/syntax"
)

// referenceExactSemanticRange は、#1177の変更前（コミットbc4b673e）の
// exactSubjectRangeのうち、意味定義Subjectの分岐をそのまま写したもの。
// Subjectごとにファイル全体を解析し、先頭の一致anchorを採る。
func referenceExactSemanticRange(content []byte, subject sourceindex.Subject) (sourceindex.Range, bool, error) {
	anchored, err := syntax.ParseSemanticFileAnchored(string(content))
	if err != nil {
		return sourceindex.Range{}, true, err
	}
	for _, item := range anchored.Anchors {
		if item.Kind == subject.Kind && item.ID == subject.ID {
			rng, err := sourceindex.RangeFromOffsets(content, item.Anchor.Start.Offset, item.Anchor.End.Offset)
			return rng, true, err
		}
	}
	return sourceindex.Range{}, true, fmt.Errorf("semantic subject %s:%s has no syntax anchor", subject.Kind, subject.ID)
}

func TestSemanticAnchorTableMatchesPerSubjectParse(t *testing.T) {
	const valid = "namespace sales\n\ndata Order {\n    attribute orderId type std.text\n    attribute total type std.text\n}\n\ndata Line {\n    attribute lineId type std.text\n}\n"
	// 同じdata・attributeを2回宣言したファイル。先頭の宣言のanchorが採られる。
	const duplicated = "namespace sales\n\ndata Order {\n    attribute orderId type std.text\n    attribute orderId type std.text\n}\n\ndata Order {\n    attribute orderId type std.text\n}\n"
	cases := []struct {
		name    string
		content string
		// wantErr は、構文エラーの失敗文言の一部。空なら構文解析は成功する。
		wantErr  string
		subjects []sourceindex.Subject
		// firstOf は、各Subjectの範囲が指すべき最初の出現位置を検査する識別子。
		firstOf map[sourceindex.Subject]string
	}{
		{
			name:    "syntax error",
			content: "namespace sales\n\ndata Order {\n    attribute\n",
			// 失敗文言そのものは旧実装の出力と比較する。空でないことだけ別に確かめる。
			wantErr: "*",
			subjects: []sourceindex.Subject{
				{Kind: "semantic-data", ID: "sales.order"},
				{Kind: "semantic-attribute", ID: "sales.order.order_id"},
			},
		},
		{
			name:    "multiple subjects in one file",
			content: valid,
			subjects: []sourceindex.Subject{
				{Kind: "semantic-attribute", ID: "sales.line.line_id"},
				{Kind: "semantic-data", ID: "sales.order"},
				{Kind: "semantic-attribute", ID: "sales.order.total"},
				{Kind: "semantic-data", ID: "sales.line"},
				{Kind: "semantic-attribute", ID: "sales.order.order_id"},
				{Kind: "semantic-data", ID: "sales.missing"}, // anchor欠落の文言も同じ
			},
		},
		{
			name:    "duplicated declarations use the first anchor",
			content: duplicated,
			subjects: []sourceindex.Subject{
				{Kind: "semantic-data", ID: "sales.order"},
				{Kind: "semantic-attribute", ID: "sales.order.order_id"},
			},
			firstOf: map[sourceindex.Subject]string{
				{Kind: "semantic-data", ID: "sales.order"}:               "Order",
				{Kind: "semantic-attribute", ID: "sales.order.order_id"}: "orderId",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			content := []byte(tc.content)
			tables := map[string]*semanticAnchorTable{}
			// 1ファイルを共有する複数Subjectを、2周して呼ぶ。2周目は保持した表を使う。
			for pass := 0; pass < 2; pass++ {
				for _, subject := range tc.subjects {
					wantRange, wantExact, wantErr := referenceExactSemanticRange(content, subject)
					gotRange, gotExact, gotErr := exactSubjectRange(content, subject, tables, "semantic/a.tei")
					if (wantErr == nil) != (gotErr == nil) || wantErr != nil && wantErr.Error() != gotErr.Error() {
						t.Fatalf("%v: 誤りが旧実装と違います: want=%v got=%v", subject, wantErr, gotErr)
					}
					if gotRange != wantRange || gotExact != wantExact {
						t.Fatalf("%v: 範囲が旧実装と違います: want=%+v/%v got=%+v/%v", subject, wantRange, wantExact, gotRange, gotExact)
					}
					if tc.wantErr != "" && gotErr == nil {
						t.Fatalf("%v: 構文エラーが握りつぶされました", subject)
					}
					if tc.wantErr == "" && tc.firstOf == nil && subject.ID != "sales.missing" && gotErr != nil {
						t.Fatalf("%v: 予期しない誤り: %v", subject, gotErr)
					}
					if token, ok := tc.firstOf[subject]; ok {
						if gotErr != nil {
							t.Fatalf("%v: 予期しない誤り: %v", subject, gotErr)
						}
						got := tc.content[gotRange.Start.Offset:gotRange.End.Offset]
						if got != token || gotRange.Start.Offset != strings.Index(tc.content, token) {
							t.Fatalf("%v: 最初の宣言を指していません: text=%q offset=%d want offset=%d",
								subject, got, gotRange.Start.Offset, strings.Index(tc.content, token))
						}
					}
				}
			}
			if tc.wantErr != "" {
				_, _, err := exactSubjectRange(content, tc.subjects[0], map[string]*semanticAnchorTable{}, "x.tei")
				if err == nil || err.Error() == "" {
					t.Fatal("構文エラーの失敗文言が空です")
				}
			}
		})
	}
}
