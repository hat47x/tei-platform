# `model search` の受入条件6件を実測し、#5228 を閉じる（2026-11-18、第104ラウンド）

- 解消: [#5228](https://github.com/hat47x/tei-platform/issues/5228) を**受入条件の実測**で close
- 前段: [第103ラウンド](../TEI_DOGFOODING_ROUND103_CATALOG_SCOPE_2026-11-18/README.md)
- 状態: **起票の本文は古い。実装は受入条件を満たしていた。**

## 1. 目的

`#4178`（第101）・`#4169`（第102）・`#4128`（第103）と同じ方法を、**最後に残った起票** `#5228` に適用した。**本文の「検索が無い」を再現し、受入条件6件を1件ずつ測った。**

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `c64070d4f`（`origin/main`） |
| CLI実行物 | `sha256:f2c9080ccda780fda93a49c36cbaf7cca5d466651d07a9d818e8bfc3528c51a4` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.26.11 linux/amd64` |

## 3. 本文の主張は既に成立しない

```text
$ tei commands -> 30コマンド
  search/locate-like: ['model locate', 'model search']
```

本文の「`tei commands` のうち検索を表すものが1つも無い」は**現在の挙動ではない**。実装は `0402bc2f4`（`feat(cli): model search を足し、語から操作対象へ辿れるようにする (#5228)`）以降に入っている。

## 4. 受入条件6件を1件ずつ測った

| # | 条件 | 実測 |
| --- | --- | --- |
| 1 | `tei commands` に載り、広告flag＝受理flag | **PASS**（`tei.cli.model-search/v0`、`-kind`/`-limit`/`-offset`/`-term`/`-workspace` が一致） |
| 2 | `artifactPath` と `line` を返し、行が実際に語を含む | **PASS**（`{"subject":{"kind":"application","id":"order_entry"},"artifactPath":"applications/order-entry/application.tei","line":10,"text":"        target sales_core"}`） |
| 3 | 一致0件が終了コード0 | **PASS**（`exit=0`、`ok:true`、`total:0`） |
| 4 | `-kind` が結果を絞る | **PASS**（`order` 全体→`semantic-data` で11件、`kindCounts` が1種のみ） |
| 5 | `snapshotFingerprint` が他3コマンドと同じ値 | **PASS**（search・overview・locate・inspect が同じ `sha256:a424b52a…`） |
| 6 | 既存の catalog 走査を再利用 | コードの性質。**実測対象ではない**（下記 NOT RUN） |

## 5. 本文の例は「見つからない」が、その理由は応答が言う

本文の例 `sales_core.order_legacy` は**`releases/` にしか無い**。`model search -term order_legacy` は `total: 0` を返す。

**これは欠陥ではない。** 応答の `coverage.notSearchableKinds` が列挙する:

```text
['migration', 'application-release', 'database-release', 'solution-release',
 'database', 'environment', 'persistence-mapping']
```

`coverage.notDerived` も4件挙げている。**「この語は存在しない」と「この種は走査していない」を区別できる**（`#5485` の `661a5da90` がこの欄を足した）。

**私は最初、トップレベルの `note` を探して「説明が無い」と誤読した** — 実際は `coverage` の下にあった。**同じ過ちを第103ラウンドでもしている**（`sectionScope` の存在を見落としかけた）。

## 6. 気づいた非同値（起票しない）

`snapshotFingerprint` を持つのは search・overview・locate・inspect の4つで、**`model catalog` だけ持たない**。`InspectionSnapshotFingerprint(catalog.WorldFingerprint, catalog.SourceIndexFingerprint)` は catalog から計算できるので、**足せる**。

**しかし起票しない。** catalog の応答は既に `worldFingerprint` と `sourceIndexFingerprint` を**両方**持っており、**呼び出し側は同じ値を作れる**。**「他のコマンドと同じ欄が無い」こと自体は欠陥ではない**（受入条件5も search についての条件であり、catalog を対象にしていない）。

## 7. NOT RUN

- 受入条件6（catalog 走査の再利用）は**コードの性質**であり、CLI からは観測できない。**未検証**として残す。
- 実DB・実Provider・実Host、HTTP・Desktop。

## 8. 位置付け

**「起票の本文を信じて未解決扱いを続けるのを、4回連続でやめた」回**である。`#4178`・`#4169`・`#4128`・`#5228` の4件は**いずれも本文が古く、実測すると状態が違った**。**起票は状態ではなく、測り直すべき仮説**だという結論が4回続いた。
