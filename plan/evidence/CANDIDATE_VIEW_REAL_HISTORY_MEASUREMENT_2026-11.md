# 候補ビューの実変更履歴での負担測定（2026-11）

- 状態: **読み取り専用の測定。製品コード・受理条件は変更していない。**
- 目的: `reference/workspace` の実際の変更履歴を使い、候補ビューの件数と修正負担を測る。合成シナリオだけでなく、実際に受理された変更で過剰・過小報告が追加作業を生むかを見る。
- 関連: [#2619](https://github.com/hat47x/tei-platform/issues/2619)、[候補ビューの負担比較](../../plan/design/ROLE_BOOTSTRAP_AND_SELF_DESCRIPTION_2026-11.md)

## 1. 方法

1. 対象コミットの親と本人から `reference/workspace` を `git archive` で一時領域へ展開する。
2. 履歴上の manifest に `formatVersion: 1` を補う。当時の manifest には形式版がなく、現行 loader が読めないための機械的な読み込み正規化であり、意味や対象ファイルは変えない。
3. 現行の `tei validate` と `compare-application-membership` を実行する。
4. 候補件数と、`reference/workspace` の変更ファイル・行数を記録する。`.tei/decisions/*.json` は受理記録として生成されたもので、手作業の修正負担とは分けて数える。

## 2. 測定結果

| 変更 | 手作業のSource変更 | validate | contract | gate | appRelease | dbRelease | 観測 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `55dec4480` header-detail Contract Query traversal | 3ファイル、+17行 | 前後とも成功 | 1 | 0 | 0 | 0 | `sales_core.order_line` の追加を、Application要求とheader-detail Bindingの2 consumer付きで報告。新しい3ビューは0 |
| `1619de2cc` order_entryのValidation選択 | 1ファイル、+2行（決定JSON 375行は別） | 前後とも成功 | 0 | 0 | 0 | 0 | 候補0件。合成の変更履歴に対して偽陽性なし |
| `fc40ac4b0` line_number_positiveの受理 | 2ファイル、+5行（決定JSON 405行は別） | 前後とも成功 | 0 | 0 | 0 | 0 | 候補0件。Applicationとsemanticの変更は意味差分・影響に現れる |
| `126f752cd` customer Contract Queryのtyped signature | 1ファイル、+4/−1行 | 変更後は成功。変更前は現行規則で `REFERENCE_CONTRACT_QUERY_SIGNATURE_REQUIRED` | 1 | 0 | 0 | 0 | `-include-unvalidated-candidates` で実行。`master_data.customer` の変更を10 consumer entry（前後各5）で報告。新しい3ビューは0、終了1 |

`126f752cd` の変更前が現行規則で不正なのは、この変更自体が後から必須になったtyped signatureを追加するためである。履歴Snapshotを現在の検証規則で読む場合、当時は存在しなかったエラーが出ることを記録する。

## 3. 解釈

- 新しい3ビュー（`unreferencedUnsatisfiableGateCandidates`、`applicationReleaseContractCandidates`、`databaseReleaseContractCandidates`）は、対象とした実変更4件で偽陽性を出さなかった。
- `contractCandidates` は実際に変更されたContractだけを報告した。`55dec4480` の追加Contractは2 consumer、`126f752cd` の変更Contractは前後各5 consumerとして現れた。
- 手作業の修正負担は1〜3ファイル、2〜17行で、候補の処理が追加のSource編集を要求した例はなかった。決定JSONの375〜405行は受理記録であり、手作業の修正とは別に扱う。
- 利用できる履歴には、Contractのmajor、Database Release、Application Release要求のmajorを変更する実コミットがなかった。そのため、major不一致を扱う3ビューの実変更での修正負担は測れていない。この範囲は合成Workspaceの試験（既存の負担比較）に限られる。

## 4. 再現手順（概要）

```sh
# 例: 55dec4480 の親と本人を展開
git archive 55dec4480^ reference/workspace | tar -x -C /tmp/before
git archive 55dec4480  reference/workspace | tar -x -C /tmp/after
# 各 manifest に formatVersion: 1 を補ってから
go run ./cmd/tei validate -workspace /tmp/after/reference/workspace
go run ./tools/reference-shared-acceptance compare-application-membership \
  -before /tmp/before/reference/workspace -after /tmp/after/reference/workspace
```

`126f752cd` は変更前が現行規則で不正なため、`-include-unvalidated-candidates` を付ける。候補は未検証のまま表示され、終了コードは1のままである。

## 5. 限界

- `reference/workspace` の履歴だけを対象とした。他リポジトリ・実利用者の変更履歴ではない。
- 現在の検証規則を過去のSnapshotへ適用している。当時の受理判断を再現したものではない。
- 実Provider、本番権限、データ移行、実行時Deployment、実利用者の負担は未検証である。
- major変更の実履歴がないため、major不一致候補の実負担は未測定である。
