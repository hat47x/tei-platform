# P1-B read-only 受入チェック

`*.tei.md` ドメイン図表サンプルの **P1-B read-only 受入**（B-1〜B-5）を機械確認する参照ツールです。

- 設計・受入の正本: [TEI_MD_DOMAIN_DIAGRAM_TO_BUILD_IMPLEMENTATION_GUIDE_2026-11.md §2.2](../../plan/design/TEI_MD_DOMAIN_DIAGRAM_TO_BUILD_IMPLEMENTATION_GUIDE_2026-11.md)
- 対象サンプル: [reference/candidates/domain-diagram-to-build/order_entry.tei.md](../../reference/candidates/domain-diagram-to-build/order_entry.tei.md)
- P0-A 判定記録: [P0A_VERTICAL_LOWERING_EVIDENCE_2026-11.md](../../plan/gate/P0A_VERTICAL_LOWERING_EVIDENCE_2026-11.md)

## 何を確認し、何を確認しないか

確認するのは **P1-B の前提条件だけ**です。PASS は次を意味しません。

- 実 Source 消費（Loader/Reader が読んだ）
- Meaning 受理（Owner Shared Acceptance）
- Build 資格（生成・ビルドへの入力になった）
- Host grant / 実 Effect

新 `*.tei.md` Reader の実装・移行は [#3030](https://github.com/hat47x/tei-platform/pull/3030) MC の既存 Owner であり、本ツールはその正本を作りません。

## 使い方

```bash
# リポジトリルートで
go run ./tools/reference-tei-md-p1b-check
go test -count=1 ./tools/reference-tei-md-p1b-check
```

出力例:

```text
PASS B-1: sample is outside all 7 declared source roots
PASS B-2: frontmatter present, LF endings, 6 code fences balanced, 7221 bytes
PASS B-3: 3 subjects all declared: [customer.customer sales.order sales.order_line]
PASS B-4: raw-draft boundary stated and 4 section-level Lowering maps present
PASS B-5: frontmatter declares a synthetic draft

P1-B read-only acceptance: 5 conditions passed
```

## 条件

| ID | 確認内容 | 失敗しうる例（負例テストあり） |
| --- | --- | --- |
| B-1 | サンプルが `tei.workspace.yaml` の宣言 Source Root の外にある | サンプルを `semantic/` 配下へ移す |
| B-2 | Markdown として保全されている（frontmatter、LF、fence 均衡、末尾改行） | CRLF 化、fence を1つ削除 |
| B-3 | 図表が挙げる Subject 集合が semantic Source に実在する | 未定義 Subject を1つ混入 |
| B-4 | Raw Draft の境界が本文で明示され、節ごとの Lowering map がある | 境界文言の削除、Lowering map が不足 |
| B-5 | frontmatter が合成 draft であることを宣言する | `status: draft` を `status: reviewed` へ変更 |

負例テスト（`TestP1BNegativeControls`）は、各チェックが**失敗しうること**を示します。失敗できないチェックの PASS は無意味なので、負例を常設しています。

## 実装上の制約

- ワークスペース manifest は `sources:` ブロックを行単位で読む（YAML 依存を追加しない）。
- Subject 照合は `namespace X` と `data Y` 宣言から `x.y_snake` を組み立てる。これは**名前合わせの補助**であり、意味の同一性判定ではありません。
- サンプルの Subject 集合は文書本文から読む。Mermaid 構文の解析は本ツールの責務ではありません（実 Reader の担当）。

## 状態

- 作成日: 2026-11-11
- 対象 HEAD: `docs/tei-md-diagram-to-build-20261111`
- 本ツールの PASS は P1-B の前提条件のみ。P2-C（Reader 接続・旧新比較）と P3-D（複数 Consumer 一致）は **NOT RUN**。
