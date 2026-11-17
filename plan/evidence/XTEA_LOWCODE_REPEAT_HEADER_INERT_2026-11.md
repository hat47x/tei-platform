# 改ページの見出し繰返しが宣言されるが効いていない（2026-11-17）

## 対象

- `internal/application/model.go` の `PageBreak{ Rows, RepeatHeader }` と `repeat-header` の構文
- `internal/applicationview/header_detail.go`（`PageBreakRef` への解決と `DetailRows` の生成）
- `internal/applicationview/print.go` の `RenderHeaderDetailPrintHTML`

## 見つけたこと

`page-break N repeat-header` は **構文として受理され、検証も通り、投影にも出る**。ところが**実行（印刷HTML）では、繰返しは文字として出るだけで、表の構造に効かない**。

`RenderHeaderDetailPrintHTML` の該当部分:

```
{{if .PageBreak}}<p>{{.PageBreak.Rows}}行/ページ{{if .PageBreak.RepeatHeader}}、見出し繰返し{{end}}</p>{{end}}
<table>
<thead><tr>...</tr></thead>
<tbody>
{{range .DetailRows}}<tr>...</tr>
{{end}}</tbody>
</table>
```

- `Rows` は**行数として現れる**: `header_detail.go` が `DetailRows` を `rows` 個生成する。ただし**それは実データではなく合成のプレースホルダ**（`0..N-1` の整数）であり、実データの明細行数に効いているわけではない（下記の訂正）。
- `RepeatHeader` は**注記だけ**: `<p>` に「見出し繰返し」と書く。`<thead>` は1回しか出ておらず、改ページの区切り（`page-break-inside`・複数の `<table>` 分割・`break-before`）も無い。

つまり「見出しを各ページに繰り返す」という宣言は、**HTML投影の時点で効いていない**。PDF 生成が未実装であることを踏まえても、**投影の中でさえ効いていない**点が問題である。

これは知識目録 D-9（定義に残るが実行されない）と同じ形である。X-TEA の `RepeatHeader` 相当（`XF390` の句の属性）は帳票エンジンが解釈していたが、TEI では**受け取った宣言の一部が描画に反映されていない**。

## なぜ問題か

1. **宣言が受理されるので、著者は効いていると考える。** 検証も投影も通るため、気づく手掛かりが無い。
2. **記録（受容対応表）では XF390 を「一部」と書いたが、その「一部」の中身に「宣言はあるが描画に効かない」が含まれる**ことを、私は明記していなかった。
3. 同じ `PageBreak` を PDF 実装が読むとき、`Rows` だけを見て `RepeatHeader` を忘れる危険がある（**今の投影がまさにその状態**）。

## 選択肢

### A. 投影で実際に改ページする

`DetailRows` を `Rows` ごとの複数の `<table>` に分け、2枚目以降の `<thead>` を繰り返す。

- **利点**: 宣言が投影で効く。PDF 実装への手本になる。
- **欠点**: 表を分割すると列幅・罫線が分かれる。HTML の見た目として正しいかは別の判断が要る。
- **コスト**: 小〜中（テンプレートの変更）。

### B. 効かないことを明示する（注記の文言を変える）

「見出し繰返し」と書く代わりに「見出し繰返し（未適用）」と書く、または警告を出す。

- **利点**: 嘘をつかない。コスト最小。
- **欠点**: 宣言が効かないまま残る。

### C. 宣言を受理しない（検証で拒否する）

`repeat-header` を書いたら「まだ適用できない」と拒否する。

- **利点**: 受理＝効く、を保てる。
- **欠点**: 正しい宣言を書けなくなる。X-TEA の意味を写せない。

## 既定案: A（投影で実際に改ページする）

理由:

1. **投影の中で閉じた修正**であり、PDF 実装を待たずに「宣言が効く」状態にできる。
2. ~~`Rows` が既に効いているので、同じ宣言の半分だけが効く非対称を残す理由が無い。~~ **この前提は誤り**（下記の訂正）。`Rows` も実データに効いていないため、投影だけの修正では足りない。
3. C（受理しない）は、意味としては正しい宣言を落とすので、受容の目的に反する。

ただし **A を実施するまでは、現状を「効いていない」と記録しておく**（これが本稿の役割）。B は A までの暫定として妥当であり、A を選ぶなら同時に不要になる。

## 覆す条件

- 印刷の版面を PDF 側（帳票エンジン）で決める設計になったら、HTML 投影で分割せず、`Rows` と `RepeatHeader` を PDF へ渡すだけにする（A をやめる）。
- 表の分割が HTML として不適切（列幅・罫線の崩れ）と判明したら、B に戻す。

## 読み取れないこと

- PDF 生成が実装されたとき、`RepeatHeader` を読む保証は無い（今は読む実装が無い）。
- X-TEA の帳票エンジンが `RepeatHeader` をどう解釈していたか（句の属性に埋まっており、静的な読解の範囲）。
- 実ブラウザでの改ページ表示（WebView2 等）は未確認。

## 位置付け

**発見の記録**であり、修正は未了。受容対応表の XF390 の「残る未確認」に、この非対称を追記する。仮承認。
## 訂正（2026-11-17）: `Rows` も実データに効いていない

本記録は当初「`Rows` は効いている」と書いた。**これは誤りだった。**

`header_detail.go` を読み直すと、`DetailRows` は**合成のプレースホルダ**である。

```go
rows := 5
if view.PageBreak != nil && view.PageBreak.Rows > 0 && view.PageBreak.Rows <= 50 {
    rows = view.PageBreak.Rows
}
for i := 0; i < rows; i++ {
    view.DetailRows = append(view.DetailRows, i)   // 0..N-1 の整数
}
```

`Rows` は「投影に置くプレースホルダの個数」を決めるだけで、**実際の明細行を読んでいない**。このビューは実データを持たないため、`repeat-header` を各ページへ反映する修正を**投影の中だけで行うことはできない**。

実際に試して失敗した: 5行を2行/ページで分割する実装を書き、試験を書いたところ、`DetailRows` が2個（`Rows` の値）しかないため**1ページ**にしかならず落ちた。そのため修正は取り消した（revert 済み）。

したがって、この2つの宣言は**どちらも実行時に効いていない**。

| 宣言 | 投影での見え方 | 実行（印刷）での効果 |
| --- | --- | --- |
| `page-break N` | プレースホルダが N 個になる | **無い**（実データを読まない） |
| `repeat-header` | `<p>` に注記が出る | **無い** |

前稿の「同じ宣言の半分だけが効く非対称」という表現も不正確だった。**両方とも効いていない。**

修正には、印刷のビューが**実際の明細行**（`driver` の `MappedResult` など）を持つことが前提になる。これは行スコープのフックと同じ供給源の問題であり、[行の宣言場所の判断](XTEA_LOWCODE_ROW_HOOK_DECLARATION_SITE_2026-11.md)と合わせて扱うべきである。

## 実装（2026-11-17、やり直し）: 投影で各ページに `<thead>` を出す

前稿の「投影の中だけでは直せない」という結論は、**半分だけ正しかった**。実データが無いことは正しい。だが、それで直せないのは `page-break` が**実データの行数**に効くことだけで、**宣言した大きさで投影の明細を分け、各面へ見出しを出す**ことは投影の中でできる。

### 前ラウンドが失敗した理由（コードで確認）

失敗の原因は投影の構造ではなく、**行数の作られ方**だった。

```go
rows := 5
if view.PageBreak != nil && view.PageBreak.Rows > 0 && view.PageBreak.Rows <= 50 {
    rows = view.PageBreak.Rows
}
```

プレースホルダの個数が `page-break` の `Rows` と等しくなるため、`Rows` 行を `Rows` 行/ページで割れば常に**1面**になる。前ラウンドはここを直さずに分割だけを書いたので、2面目が空になり落ちた。訂正節の「投影の中だけで行うことはできない」は、この1行を直さない前提での結論である。

### 選んだもの: (b) 分割の単位を「実際に描ける行数」にし、プレースホルダでも各ページに thead を出す

前ラウンドの失敗は (b) で解ける。(a)（プレースホルダを `Rows` に合わせて増やす）は採らない。見た目の行数が見た目の改ページ宣言に依存し続け、実データが来たときに何も変わらないためである。(c)（`MappedResult` → `HeaderDetailView` の新設）は**今回やらない**。理由は、それが `view -environment` の読み取り経路と印刷の経路を初めて繋ぐ変更であり、`plan/NOW.md` の道の2（Webの実行先）と同じ供給源の作業になるため。部分実装で止めるのではなく、今回は (b) として独立に閉じる。

### 変えたこと

1. `internal/applicationview/header_detail.go`
   - `detailPlaceholderRows = 5` を導入し、プレースホルダの個数を `page-break` から切り離した。`Rows` は「どう分けるか」だけを決める。
   - `HeaderDetailView.DetailPages()` を追加した。`PageBreak.Rows` ごとに `DetailPage{Number, Rows, RepeatHeader}` を作る。`Rows` 行へ収まるなら分割しない。
   - 見出しの描画は `RepeatHeader` が決める。1面目は常に描き、2面目以降は `repeat-header` があるときだけ描く。
2. `internal/applicationview/print.go`
   - テンプレートを `DetailPages` のループへ変え、面ごとに `<table class="detail" data-page="N">` を出す。
   - 2面目以降に改ページを持たせるため `.detail+.detail{break-before:page}` を CSS へ足した。
   - `<thead>` は `RepeatHeader` が真の面にだけ出す。宣言していない繰返しが見た目に出ない。
3. 試験
   - `TestHeaderDetailPrintRepeatsHeaderOnEveryPage`: 3行/ページ・`repeat-header` で表2・`<thead>`2。
   - `TestHeaderDetailPrintKeepsOneTableWithoutRepeatHeader`: `repeat-header` 無しでは、分割しても `<thead>` は1（宣言と描画を一致させる）。
   - `TestHeaderDetailPrintKeepsOneTableWhenFitsOnePage`: 5行/ページ（ちょうどの大きさ）では分割しない1表。
   - `TestHeaderDetailPrintWithoutPageBreak`: 宣言が無ければ注記も分割もしない。
   - `TestHeaderDetailPrintDoesNotDropRows`: 7通りの宣言（未宣言・1〜6行/ページ）で明細行が5行のまま落ちない。
   - `TestHeaderDetailDetailPagesNumbersPages`: 面番号が1から連続し、行が重複も欠落もしない。
   - `TestBuildHeaderDetailCarriesPageBreak`: プレースホルダ数が `Rows` ではなく5であることを固定した（前ラウンドの前提を試験で止める）。

### 状態

- **動く（投影の中で）**: `page-break N repeat-header` は、印刷HTMLで N 行ごとの複数の表になり、2面目以降も `<thead>` を描く。
- **試験だけ**: 実ブラウザ（WebView2等）での改ページ表示とPDF生成は未確認。分割はCSSの `break-before` に依存する。
- **未実装**: 実データの明細行数への反映。`DetailRows` は依然として合成のプレースホルダであり、印刷面の行数は実際のRelationの行数ではない。この解消は (c) の経路（`driver` の `MappedResult` → 印刷）で行う。

### 覆す条件

- 帳票エンジン（PDF）が版面を決める設計になったら、HTML投影の分割をやめ、`Rows` と `RepeatHeader` をそのまま渡す。
- 表の分割がHTMLの見た目として不適切（列幅・罫線の崩れ）と判明したら、分割をやめて注記だけに戻す。
## 追記2（2026-11-18）: 独立レビューの指摘

**#5428 で描画は効くようになったが、意味が変わった。** 独立レビューが次を指摘した。

- **`page-break N` の意味が「描く行数」から「分割単位」へ変わった**。プレースホルダ行は常に5。例の `page-break 3 repeat-header` は「1表3行」から「2表5行（3+2）」へ変わる。**利用者可視の出力変更**である。`application/model.go` のコメントと、印刷の注記「N行/ページ」を実装へ合わせて直した（注記は「N行ごとに面を分ける」へ）。
- **最終面は端数**なので、注記の N と実際の面の行数が一致しない（プレースホルダ前提の限界）。
- 試験名 `KeepsOneTableWithoutRepeatHeader` は**実際には2表を期待**しており名前と期待が逆だった。`SplitsWithoutRepeatingTheHeader` へ改名。
- `.detail+.detail{break-before:page}` は CSS Fragmentation として妥当。古い実装向けに `page-break-before` の併記が安全（未対応）。

## 読み取れること（更新）

- `repeat-header` を宣言すると、**分割した各面に `<thead>` が出る**。
- 宣言しない場合は分割しても2面目に `<thead>` を出さない。

## 読み取れないこと（更新）

- **実データの行数**。プレースホルダは5固定で、`Rows` は分割の単位だけを決める。実データ経路（`MappedResult` → `HeaderDetailView`）は未実装。
- 実ブラウザ（WebView2等）・PDF での改ページ。
