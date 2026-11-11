# P1-B read-only 受入の機械検証記録

- **対象**: `*.tei.md` ドメイン図表サンプルの P1-B read-only 受入（B-1〜B-5）
- **設計正本**: [TEI_MD_DOMAIN_DIAGRAM_TO_BUILD_IMPLEMENTATION_GUIDE_2026-11.md](../design/TEI_MD_DOMAIN_DIAGRAM_TO_BUILD_IMPLEMENTATION_GUIDE_2026-11.md) §2.2
- **検証実装**: [`tools/reference-tei-md-p1b-check/`](../../tools/reference-tei-md-p1b-check/README.md)
- **実行日**: 2026-11-11
- **判定**: **P1-B read-only 受入は機械検証で成立**。ただし後述のとおり、これは実 Source 消費・Meaning 受理・Build 資格・Host Grant ではない。

## 1. 背景：受入条件が散文のままだった

実装導線 §2.2 は B-1〜B-5 を表で定めたが、サンプルを変更しても条件が保たれているかを誰も機械確認できなかった。条件を散文にしておくと、後続の担当者が「たぶん満たしている」と判断するしかない。そこで条件ごとに判定関数と負例テストを用意した。

## 2. 実行結果（正例）

```
$ go run ./tools/reference-tei-md-p1b-check
PASS B-1: sample is outside all 7 declared source roots
PASS B-2: frontmatter present, LF endings, 6 code fences balanced, 7221 bytes
PASS B-3: 3 subjects all declared: [customer.customer sales.order sales.order_line]
PASS B-4: raw-draft boundary stated and 4 section-level Lowering maps present
PASS B-5: frontmatter declares a synthetic draft

P1-B read-only acceptance: 5 conditions passed
```

```
$ go test -count=1 ./tools/reference-tei-md-p1b-check
ok  tei/tools/reference-tei-md-p1b-check
```

`go vet` も PASS。

## 3. 負例（チェックが失敗しうることの確認）

`TestP1BNegativeControls` は各条件について、意図的に壊した入力を与えて **FAIL になること**を確かめる。

| 条件 | 負例 | 期待 |
| --- | --- | --- |
| B-1 | サンプルを宣言 Source Root（`semantic/`）配下へ移す | FAIL |
| B-2 | 開始フェンスを1つ削除して fence 数を奇数にする | FAIL |
| B-2 | 改行を CRLF へ変える | FAIL |
| B-3 | 未定義 Subject `sales.unmodelled_thing` を混入 | FAIL |
| B-4 | Raw Draft の境界文言と `実Source消費` を削除 | FAIL |
| B-5 | `status: draft` を `status: reviewed` へ変更 | FAIL |

失敗しえないチェックの PASS は意味を持たないため、負例を常設した。

## 4. 実装上の判断（記録）

- **YAML 依存を追加しない**: ワークスペース manifest は `sources:` ブロックを行単位で読む。検査のためにモジュール依存を増やさない。
- **Mermaid を解析しない**: 図の Subject 集合は文書本文の `Subject集合` 行から読む。Mermaid 構文の解釈は実 Reader（#3030）の責務であり、本検査が代行すると第二 Reader になる。
- **Subject 名の照合は補助**: `namespace X` と `data Y` から `x.y_snake` を組み立てて名前を突き合わせる。これは**名前合わせ**であって意味の同一性判定ではない。

## 5. 成立しなかったもの（明示）

| 項目 | 状態 | 理由 |
| --- | --- | --- |
| 実 Source 消費（Loader/Reader が読んだ） | **NOT RUN** | `*.tei.md` の Reader は #3030 M2/M3 の既存 Owner。本検査はファイルの性質だけを見る |
| Meaning 受理（Owner Shared Acceptance） | **NOT ACCEPTED** | #267 の担当 |
| Build 資格（生成・ビルドへの入力になった） | **NOT RUN** | P2-C 以降の範囲 |
| Host Grant / 実 Effect | **NOT RUN** | #1174/#2516 の担当 |
| P2-C（Reader 接続・旧新比較） | **NOT RUN** | #3030 待ち |
| P3-D（複数 Consumer 一致） | **NOT RUN** | #1124/#2497/#2516 の担当 |

## 6. 段階分離（DG-04 の維持）

P1-B の PASS は **P0-A の PASS を継承しない**。P0-A は旧 `.tei` の Parse→Model→Generate→Build を独立 Expected で照合した別の Claim であり、新 `*.tei.md` の Reader 接続は含まない。P1-B の PASS も P2-C 以降へ継承しない。

## 7. CI との関係（重要）

本 PR の `repository-health` には、**main で既に失敗している項目**が含まれる。切り分けの結果:

| 失敗項目 | main での状態 | 本 PR の差分 |
| --- | --- | --- |
| `TestWorkflowsFollowGoModVersion`（`tei-studio-wails-desktop-webview-smoke.yml` の literal `go-version`） | **FAIL**（`origin/main` に literal `go-version: '1.26.11'` が存在） | 本 PR は当該 workflow を変更しない |
| `TestMarkdownTitleAppearsOnce`（`plan/design/README.md` の `# 機能設計文書` 重複） | **FAIL**（`origin/main` に同タイトルが2回） | 本 PR は当該行を追加しない |

本 PR が**自分で導入した**失敗は次の2種類で、いずれも修正済みである。

- `TestMarkdownRelativeLinksResolve`: 設計文書と実装 PR に分かれた P0-A 記録への相対リンクが宙に浮いていた → 相対リンクを PR 参照へ置換
- `TestToolsReadmeMentionsEveryEntry`: 新しい `tools/reference-tei-md-p1b-check/` を `tools/README.md` に登録していなかった → 登録

したがって `repository-health` の残る FAIL は main 由来であり、本 PR の変更範囲に閉じない。
