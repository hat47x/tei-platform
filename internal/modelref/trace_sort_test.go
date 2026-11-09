package modelref

import (
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"testing"

	"tei/internal/subject"
)

// traceRowKeyReferenceは、速度のための書き換え前の定義そのもの。
// 実装が変わっても、キーの文字列と並び順がこの定義と一致し続けることを固定する。
func traceRowKeyReference(row TraceRow) string {
	to := row.Edge.To
	fields := []string{
		row.Edge.From.Scope, row.Edge.From.Subject.Kind, row.Edge.From.Subject.ID,
		to.Scope, to.Subject.Kind, to.Subject.ID,
		to.Version, row.Edge.Relation, row.Status, row.Issue, evidenceKey(row.Edge.Evidence),
	}
	out := ""
	for _, field := range fields {
		out += fmt.Sprintf("%d:%s", len(field), field)
	}
	return out
}

func traceSortTestRows(seed int64, count int) []TraceRow {
	random := rand.New(rand.NewSource(seed))
	// 区切り文字・長さ接頭辞に見える文字列・多バイト文字・空文字を意図して混ぜる。
	pieces := []string{"", "a", "b", "1", "10", "3:abc", "a:b", "a-b", "x/y", "名前", "\x00", "::", "0:", "z"}
	pick := func() string {
		var b strings.Builder
		for n := random.Intn(3); n >= 0; n-- {
			b.WriteString(pieces[random.Intn(len(pieces))])
		}
		return b.String()
	}
	authorities := []string{"accepted", "proposal", "source-observation", "derived", ""}
	rows := make([]TraceRow, count)
	for i := range rows {
		evidence := make([]Evidence, random.Intn(4))
		for j := range evidence {
			evidence[j] = Evidence{
				Authority:   authorities[random.Intn(len(authorities))],
				Context:     pick(),
				SourceToken: pick(),
			}
		}
		rows[i] = TraceRow{
			Status: []string{"resolved", "unresolved"}[random.Intn(2)],
			Issue:  []string{"", "invalid-source", "ambiguous-source", "invalid-edge-type"}[random.Intn(4)],
			Edge: Edge{
				From:     Key{Scope: pick(), Subject: subject.Subject{Kind: pick(), ID: pick()}},
				To:       Ref{Scope: pick(), Subject: subject.Subject{Kind: pick(), ID: pick()}, Version: pick()},
				Relation: pick(),
				Evidence: evidence,
			},
		}
	}
	// 同じキーになる行（重複するedge）を混ぜる。
	for i := 0; i+7 < count; i += 7 {
		rows[i+7] = rows[i]
	}
	return rows
}

func TestTraceRowKeyMatchesLengthPrefixedDefinition(t *testing.T) {
	for _, row := range traceSortTestRows(20261017, 4000) {
		if got, want := traceRowKey(row), traceRowKeyReference(row); got != want {
			t.Fatalf("key changed:\n got %q\nwant %q", got, want)
		}
	}
	if got, want := traceRowKey(TraceRow{}), traceRowKeyReference(TraceRow{}); got != want {
		t.Fatalf("empty row key: got %q want %q", got, want)
	}
}

// 区切り文字を含むIDでも、別々の行のキーが衝突しない（長さ接頭辞の意味が保たれる）。
func TestTraceRowKeyKeepsFieldBoundariesForDelimiterBearingIDs(t *testing.T) {
	left := TraceRow{Edge: Edge{
		From: Key{Scope: "a", Subject: subject.Subject{Kind: "b:c", ID: "d"}},
	}}
	right := TraceRow{Edge: Edge{
		From: Key{Scope: "a", Subject: subject.Subject{Kind: "b", ID: "c:d"}},
	}}
	if traceRowKey(left) == traceRowKey(right) {
		t.Fatal("rows with different field boundaries produced the same key")
	}
}

func TestSortTraceRowsMatchesComparatorThatBuildsKeysPerComparison(t *testing.T) {
	for _, count := range []int{0, 1, 2, 3, 12, 13, 50, 2000} {
		t.Run(fmt.Sprintf("n=%d", count), func(t *testing.T) {
			rows := traceSortTestRows(int64(count)+1, count)
			want := append([]TraceRow(nil), rows...)
			sort.Slice(want, func(i, j int) bool {
				return traceRowKeyReference(want[i]) < traceRowKeyReference(want[j])
			})

			got := append([]TraceRow(nil), rows...)
			sortTraceRows(got)
			if !reflect.DeepEqual(got, want) {
				t.Fatal("sorted rows differ from the per-comparison definition")
			}
			for i := 1; i < len(got); i++ {
				if traceRowKey(got[i-1]) > traceRowKey(got[i]) {
					t.Fatalf("rows are not ordered by key at %d", i)
				}
			}
		})
	}
}

func TestSortTraceRowsKeepsRowsAndKeysAligned(t *testing.T) {
	rows := traceSortTestRows(7, 500)
	before := map[string]int{}
	for _, row := range rows {
		before[traceRowKeyReference(row)]++
	}
	sortTraceRows(rows)
	after := map[string]int{}
	for _, row := range rows {
		after[traceRowKeyReference(row)]++
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("sorting lost, duplicated or changed rows")
	}
}
