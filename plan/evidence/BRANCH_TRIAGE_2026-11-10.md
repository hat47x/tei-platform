# 未マージブランチの棚卸し（2026-11-10）

- **位置づけ**：リポジトリ衛生の**実測記録**です。#3805（未マージのまま終了したブランチの記録）の続きとして、現在の状態を機械的に分類しました。ブランチの削除は「内容が main に取り込み済みで、かつ PR が MERGED」のものだけに限っています。
- **測定時点**：`main` `ae2771bba`、リモートブランチ92本。
- **方法**（すべて再現可能）
  1. `git ls-remote --heads origin` でブランチと SHA を取得
  2. `git merge-base --is-ancestor <sha> origin/main` で main に含まれるものを除外
  3. `git cherry -v origin/main <sha>` の `+` 行を「main が持たないコミット」として数える
  4. `gh pr list --state open` と突き合わせ、進行中のものを分離
  5. 残ったものは `git log origin/main --grep=<subject>` と main の実装で内容の有無を確認

## 結果

| 区分 | 本数 |
| --- | --- |
| リモートブランチ総数 | 92 |
| main に未マージ | 54 |
| うち内容が main に取り込み済み（削除可） | **0**（後述の6本を削除して0になった） |
| うち open PR がある（進行中） | 35 |
| うち PR が無く、未マージのコミットを持つ | **19** |

## 削除したもの（内容が main にあり、PR が MERGED）

`git cherry` が `+` を返さず、かつそのブランチを使った PR がすべて `MERGED` であることを**削除直前に再確認**したうえで削除しました。CONTRIBUTING の「統合後はブランチを削除します」に沿う作業です。

| ブランチ | PR |
| --- | --- |
| `docs/sb0c-arm64-childdigest-precision-20261110` | #4045 |
| `docs/sb0c-arm64-evidence-repro-20261110` | #4041 |
| `docs/sb0c-arm64-l2b-20261110` | #4035 |
| `docs/sb0c-child-identity-scope-20261110` | #4055 |
| `docs/sb0c-child-swap-direct-20261110` | #4063 |
| `docs/sb0c-cross-compile-superseded-20261110` | #4050 |

（`test/l2b-real-go-build-20261110` は #4090 で統合済みのうえ既に削除されていました。）

## PR が無く未マージの19本

`unique` は main が持たないコミット数です。

| ブランチ | unique | 状態（確認した根拠） |
| --- | --- | --- |
| `test/inspector-selected-source-isolation-20261109` | 16 | **未決**。Inspectorのworkspace隔離・Source版の試験群。main に対応する主題が見当たらず、内容も未確認。**最優先で判断が要る** |
| `archive/model-map-3866-before-20261110` | 7 | 名前が `archive`。マージ対象ではなく保存用と読める |
| `rebase/model-map-3866-main-20261110` | 4 | 名前が `rebase`。同上 |
| `docs/inspector-id-origin-qualified-20261110` | 3 | **未決**。InspectorのID出自の資格記録。main に同主題が見当たらない |
| `docs/sb0c-go-version-observation-20261110` | 2 | 取り込み済み。main `f83c1f4e2`（#4028）が「Go版とアーキテクチャを独立に動かし2×2で一致」を実測 |
| `docs/sb0c-strong-superseded-20261110` | 2 | 取り込み済み。main に §4.25 の注記（#3988）と既定の決着（`TEI_GO_SELF_BUILD_DEFAULTS_CLOSED_2026-11-10.md`）がある |
| `feat/inprocess-build-catalog-20261109` | 2 | 取り込み済み。main に `cmd/tei/cli_self_build_inprocess.go` がある |
| `feat/self-build-strong-flag-20261109` | 1 | 破棄と読める。`-strong` を推す §4.25 は main で superseded 扱い。main に `strength` フラグは無い |
| `diagnose/wasm-candidate-output-20261110` | 1 | 診断用。#3971 は解決済み（PR #3973・#4012） |
| `docs/sb0c-arm64-artifact-20261110` | 1 | **未決**（観測記録）。main に対応する主題が見当たらない |
| `docs/sb0c-arm64-observation-20261110` | 1 | **未決**（同上） |
| `docs/sb0c-build-time-measure-20261109` | 1 | 取り込み済み。main に `-rebuild-check` の比の実測（#2831 SB-C0 §4.28）がある |
| `feat/close-requires-enforced-20261110` | 1 | 取り込み済み。main に同じ主題のコミット（#4019）がある |
| `feat/go-builder-from-plan-20261110` | 1 | **重複**。下の `feat/l2b-from-real-go-build-20261110` と同じ主題（同一メッセージ） |
| `feat/l2b-from-real-go-build-20261110` | 1 | 取り込み済み。main に同じ主題のコミットがある（#2831 SB-C0 §4.39 段2） |
| `fix/cli-missing-argument-code-20261109` | 1 | 取り込み済み。main で必須引数不足は `INVALID_ARGUMENT`（実行して確認） |
| `fix/materialize-data-layer-nodes-20261111` | 1 | 取り込み済み。main で `unmaterialized-from` が0件（実行して確認） |
| `fix/plugin-lock-classification-20261109` | 1 | 取り込み済み。main に `INVALID_PLUGIN_LOCK` と `plugin.IsInvalidLock` がある |
| `fix/tools-readme-tei-eki-20261110` | 1 | 取り込み済み。`tools/README.md` に `reference-tei-eki-contract/` の行がある |

## 分かったこと

1. **主題の不在は内容の不在ではない。** `git log --grep` で 11本が「主題が main に無い」と出ましたが、そのうち実際に main へ入っていないのは一部だけでした。取り込みは PR タイトルを書き換えるため、主題の照合だけでは判断できません。**内容（実装・実行結果）で確かめる必要があります。**
2. **棚卸しの最中に1本が統合された。** `fix/id-rule-diagnostic-and-correction-20261110` は「main に無い」と判定した直後に PR #4091 として統合されました。並行作業では、判定と操作の間に状態が変わります。
3. **`unique=0` が最も安全な削除条件。** `git cherry` が `+` を返さないブランチは内容が main にあり、削除で失われるものがありません。今回の6本はこれに該当しました。

## 残る作業（判断が要る）

- `test/inspector-selected-source-isolation-20261109`（16コミット）の内容確認と、統合するか破棄するかの判断。
- `docs/inspector-id-origin-qualified-20261110`・`docs/sb0c-arm64-artifact-20261110`・`docs/sb0c-arm64-observation-20261110` の主題が main の同等記録で足りているかの確認。
- `archive/*`・`rebase/*` は保存目的と読めますが、保存が要るかの確認。
- open PR がある35本は進行中なので触っていません。

## 再現手順

```bash
git fetch origin --prune
git ls-remote --heads origin | while read -r sha ref; do
  branch=${ref#refs/heads/}
  [ "$branch" = main ] && continue
  if git merge-base --is-ancestor "$sha" origin/main 2>/dev/null; then continue; fi
  unique=$(git cherry origin/main "$sha" | grep -c '^+')
  printf '%-62s unique=%s\n' "$branch" "$unique"
done
gh pr list --repo hat47x/tei-platform --state open --limit 300 --json number,headRefName
```

## 関連

- #3805（未マージブランチの記録。本記録はその後続の棚卸し）
- 削除基準は [CONTRIBUTING.md](../../CONTRIBUTING.md) の「統合後はブランチを削除します」
