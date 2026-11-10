# 空回りする検査を1箇所に集約して可視化した（2026-11-09、第36ラウンド）

- 対象: [#3931](https://github.com/hat47x/tei-platform/issues/3931)（第35ラウンドで起票、Open のまま）。
- 前段: 第33（walk が0ファイル）・34（正規表現が0一致）・35（`plan/issue` が0件）ラウンド。
- 実行日: 2026-11-09（Asia/Tokyo）。
- 状態: **残りの検査を全数調査し、空回りを1つの試験で可視化。**

## 1. 目的

第35ラウンドで「`numberedFiles` の利用者2件を見たに留まる。他の検査が同じ穴を持つかは未調査」と書いた。これを閉じる。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `a9d98e5fe`（`origin/main`） |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 観測: 各検査が実際に読む件数

計測用の試験を一時的に置いて、全スコープの件数を出した。

```text
markdown files visited: 951
issue numbered:          0
adr numbered:           47
requirementIDs:         36

plan            15 entries
plan/adr        48
plan/issue       1   ← README.md のみ
plan/evidence  290
plan/workstreams 30
tools          249

go.mod                    exists (826 bytes)
cmd/tei-studio/go.mod     exists (498 bytes)
THIRD_PARTY_NOTICES.md    exists (18984 bytes)
```

**空だったのは `plan/issue` だけ。** 第35ラウンドで塞いだ箇所と一致する。`forEachMarkdown` 系（951件）・ADR（47件）・要件（36件）・go.mod 系はすべて実体があった。

つまり**第35ラウンドの懸念は「他に無い」で閉じる。**

## 4. ただし1箇所、私の第35ラウンドの修正に穴があった

第35ラウンドで追加した `TestIssueNumbersAreLocalDocumentsAbsent` は `numberedFiles(...) == 0` を見るので正しい。しかし今回書いた統合ガードの初版は `os.ReadDir` の**件数**を見ており、`plan/issue` は `README.md` を1件持つため「空でない」と判定していた。**除外（exempt）の分岐が一度も実行されない**状態だった。

`-v` の出力に除外の行が出ないことで気づいた。

```text
（初版の出力: Markdown 951 / Requirement 36 のみ。除外の行が無い）
```

修正後:

```text
plan/issue yields 0 documents by design: Issue records live in GitHub Issues;
the directory keeps only a README (#3931)
Markdown files examined by the document checks: 951
Product Requirement ids examined: 36
```

**ディレクトリの件数ではなく、検査が読む文書の件数を測る**ようにした。

## 5. 追加した試験

`tools/repository-plan-check/scope_nonempty_test.go` の `TestEveryCheckScopeIsNonEmpty`。

| 対象 | 測り方 |
| --- | --- |
| `plan/adr`（digits=4）・`plan/issue`（digits=6） | 番号付き文書の件数 |
| `plan/evidence`・`plan/workstreams`・`tools` | ディレクトリの件数 |
| Markdown 全体 | `walkMarkdown` の訪問数が0でないこと |
| Product Requirement | `requirementIDs` が0件でないこと |

0件かつ除外指定がある場合だけ `t.Logf` で理由を出して通す。除外指定が無ければ `t.Errorf`。

### 5.1 変異試験

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | `plan/adr` を退避（番号付きスコープを空に） | FAIL |
| M2 | `plan/workstreams` を空に | FAIL `plan/workstreams yields 0 documents; the checks over it would pass without examining anything` |
| M3 | `plan/issue` に番号付き文書を1件置く | FAIL `plan/issue now holds 1 numbered documents; restore a uniqueness check for them` |

3つとも意図どおり落ちた。M3 は第35ラウンドで追加した「不在の明示」が効いていることの確認でもある。

## 6. 何が変わったか

- 空回りの**検出が1箇所に集約**された。検査を追加する人は、対象が0件ならここで落ちる。
- 除外（`plan/issue`）は**理由つきで明示**され、`-v` に出る。「空だが意図的」と「空で壊れている」が区別できる。
- 件数（951・36）が常にログに出るので、walk が縮んだことに気づける。

## 7. 読み取れないこと（NOT RUN）

- 各検査の**内部**の網羅性（例: 951件の Markdown のうち何件が実際にリンクを持つか）。今回は「入力が0でない」ことだけを保証した。
- `TestCheckReportsEachKindOfDisagreement` など、`t.TempDir()` に作った fixture を使う試験の妥当性（fixture は常に非空なので対象外）。
- GitHub Issues の状態が6値に従っているか（#3931 の残件）。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 8. 位置付け

4ラウンド続けて「検査が静かに無効になる」型に取り組んだ。

| ラウンド | 形 | 今回の位置付け |
| --- | --- | --- |
| 33 | walk の起点が誤り、0ファイル | 個別に修正済み |
| 34 | 正規表現が0一致 | 個別に修正済み |
| 35 | `plan/issue` が0件 | 個別に修正済み |
| 36 | — | **同型をまとめて検出する試験を追加** |

個別の修正だけでは、次に別の検査が空回りしたときに同じ手順（気づくまで分からない）を繰り返す。今回は検出を1箇所に置いた。

**ただし「入力が0件でない」ことしか見ていない。** 入力が十分かを保証するものではない。
