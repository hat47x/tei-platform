package applicationview_test

import (
	"strconv"
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/applicationview"
	"tei/internal/ir"
)

// pageBreakWorldは、header-detailの改ページ宣言だけを変えた世界を返す。
// 明細のプレースホルダ行数は宣言に依存しないため、宣言を変えても
// 行数は変わらない。これが「行を落とさない」分割の前提になる。
func pageBreakWorld(pageBreak *application.PageBreak) ir.World {
	world := headerDetailWorld()
	world.Applications[0].Patterns[0].PageBreak = pageBreak
	return world
}

func renderHeaderDetailPrint(t *testing.T, pageBreak *application.PageBreak) string {
	t.Helper()
	view, err := applicationview.BuildHeaderDetail(pageBreakWorld(pageBreak), "order_entry", "order_entry.order_lines")
	if err != nil {
		t.Fatal(err)
	}
	html, err := applicationview.RenderHeaderDetailPrintHTML(view)
	if err != nil {
		t.Fatal(err)
	}
	return html
}

func detailRowCount(html string) int {
	return strings.Count(html, "<tr><td></td><td></td><td></td></tr>")
}

func pageBreakName(pageBreak *application.PageBreak) string {
	if pageBreak == nil {
		return "none"
	}
	if pageBreak.RepeatHeader {
		return "rows " + strconv.Itoa(pageBreak.Rows) + " repeat-header"
	}
	return "rows " + strconv.Itoa(pageBreak.Rows)
}

// TestHeaderDetailPrintRepeatsHeaderOnEveryPageは、repeat-headerを宣言した
// 分割で、各面が自分の見出し行を持つことを確かめる。以前は宣言が受理・
// 投影されながら、表が1つしか出ないため見出しも1回しか出なかった。
func TestHeaderDetailPrintRepeatsHeaderOnEveryPage(t *testing.T) {
	html := renderHeaderDetailPrint(t, &application.PageBreak{Rows: 3, RepeatHeader: true})
	tables := strings.Count(html, `<table class="detail"`)
	theads := strings.Count(html, "<thead>")
	if tables != 2 || theads != 2 {
		t.Fatalf("tables=%d theads=%d, want 2 and 2: %s", tables, theads, html)
	}
	if !strings.Contains(html, `<p>3行ごとに面を分ける、各面で見出し繰返し</p>`) {
		t.Fatalf("print html lost the page-break note: %s", html)
	}
}

// TestHeaderDetailPrintSplitsWithoutRepeatingTheHeaderは、分割しても
// repeat-headerを宣言していない場合は2面目に見出しを描かないことを確かめる。
// 宣言していない繰り返しが見た目に出れば、宣言と描画が食い違う。
func TestHeaderDetailPrintSplitsWithoutRepeatingTheHeader(t *testing.T) {
	html := renderHeaderDetailPrint(t, &application.PageBreak{Rows: 3})
	tables := strings.Count(html, `<table class="detail"`)
	theads := strings.Count(html, "<thead>")
	if tables != 2 || theads != 1 {
		t.Fatalf("tables=%d theads=%d, want 2 and 1: %s", tables, theads, html)
	}
	if strings.Contains(html, "各面で見出し繰返し") {
		t.Fatalf("print html claims a repeated header that was not declared: %s", html)
	}
}

// TestHeaderDetailPrintKeepsOneTableWhenFitsOnePageは、明細が1面に収まる
// 宣言では表を分けないことを確かめる。分けない1つの表は、分割の有無を
// 読み手が区別できる基準になる。プレースホルダは5行なので、5行/ページの
// 宣言は境界である（ちょうどの大きさは分けない）。
func TestHeaderDetailPrintKeepsOneTableWhenFitsOnePage(t *testing.T) {
	html := renderHeaderDetailPrint(t, &application.PageBreak{Rows: 5, RepeatHeader: true})
	tables := strings.Count(html, `<table class="detail"`)
	theads := strings.Count(html, "<thead>")
	if tables != 1 || theads != 1 {
		t.Fatalf("tables=%d theads=%d, want 1 and 1: %s", tables, theads, html)
	}
	if got := detailRowCount(html); got != 5 {
		t.Fatalf("detail rows=%d, want 5: %s", got, html)
	}
}

// TestHeaderDetailPrintWithoutPageBreakは、page-breakを宣言しない場合に
// 分割しないことを確かめる。宣言が無いのに分ければ、宣言していない
// 改ページを投影が作ることになる。
func TestHeaderDetailPrintWithoutPageBreak(t *testing.T) {
	html := renderHeaderDetailPrint(t, nil)
	tables := strings.Count(html, `<table class="detail"`)
	theads := strings.Count(html, "<thead>")
	if tables != 1 || theads != 1 {
		t.Fatalf("tables=%d theads=%d, want 1 and 1: %s", tables, theads, html)
	}
	if strings.Contains(html, "行ごとに面を分ける") {
		t.Fatalf("print html invents a page-break that was not declared: %s", html)
	}
}

// TestHeaderDetailPrintDoesNotDropRowsは、分割のどの宣言でも明細行が
// 落ちないことを確かめる。分割は「どこで切るか」だけを変え、行の集合は
// 変えない。以前の失敗は、分割の単位を宣言の行数に合わせたため、
// 明細が1面分しか無く、2面目が空になったことだった。
func TestHeaderDetailPrintDoesNotDropRows(t *testing.T) {
	for _, pageBreak := range []*application.PageBreak{
		nil,
		{Rows: 1},
		{Rows: 1, RepeatHeader: true},
		{Rows: 2, RepeatHeader: true},
		{Rows: 3, RepeatHeader: true},
		{Rows: 5},
		{Rows: 6, RepeatHeader: true},
	} {
		html := renderHeaderDetailPrint(t, pageBreak)
		if got := detailRowCount(html); got != 5 {
			t.Fatalf("page-break %s: detail rows=%d, want 5: %s", pageBreakName(pageBreak), got, html)
		}
	}
}

// TestHeaderDetailDetailPagesNumbersPagesは、面の番号が1から連続し、
// 各部分列がDetailRowsの続きであることを確かめる。番号と行の対応が
// 崩れれば、どの面にどの行が載るかを読み手が追えない。
func TestHeaderDetailDetailPagesNumbersPages(t *testing.T) {
	world := pageBreakWorld(&application.PageBreak{Rows: 2, RepeatHeader: true})
	view, err := applicationview.BuildHeaderDetail(world, "order_entry", "order_entry.order_lines")
	if err != nil {
		t.Fatal(err)
	}
	pages := view.DetailPages()
	if len(pages) != 3 {
		t.Fatalf("pages=%d, want 3: %+v", len(pages), pages)
	}
	want := 0
	for i, page := range pages {
		if page.Number != i+1 {
			t.Fatalf("page number=%d at %d", page.Number, i)
		}
		if !page.RepeatHeader {
			t.Fatalf("page %d lost the repeated header: %+v", page.Number, page)
		}
		for _, row := range page.Rows {
			if row != want {
				t.Fatalf("page %d carries row %d, want %d", page.Number, row, want)
			}
			want++
		}
	}
	if want != len(view.DetailRows) {
		t.Fatalf("pages carry %d rows, want %d", want, len(view.DetailRows))
	}
}
