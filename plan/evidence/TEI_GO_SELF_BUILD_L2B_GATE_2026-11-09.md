# L2-bのClaimを回帰試験にする（§4.24）

§4.23でL2-bを五条件の述語にしました。**述語が正しくても、実際の試行で五条件が揃わなければClaimは立ちません。** 本記録は、**リポジトリ内の参照catalogと参照builder childで五条件が揃うこと**を試験に固定したことを記録します。

## 固定したもの

`cmd/tei`の試験が、実際に動く実行物を組み立てて**一度の`self-build`を走らせます**。

| 手順 | 内容 |
| --- | --- |
| 1 | `cmd/tei`・`tools/reference-plugin-package-build-child`・`tools/reference-build-authoring-intent`を`go build` |
| 2 | ワークスペース・宣言・source root・参照catalog（`testdata/catalog.json`）を一時ディレクトリに置く |
| 3 | 参照実装を**直接**起動して、builder childが消費するplanファイルを作る（**entryの経路ではない**） |
| 4 | `tei self-build`を**in-process＋`-intent`＋`-rebuild-check`**で走らせる |
| 5 | `l2b.claimed=true`・`l2b.unmet`空・`qualified=false`・`ok=true`・artifactが存在することを確かめる |

**参照実装はplanファイルを作るためだけに使います。** entryは自分で計画を導く（それが条件1）ので、この試験は「entryが解釈した」ことを参照実装に依存せずに確かめています。

## 二つの試験

| 試験 | 確かめること |
| --- | --- |
| `TestSelfBuildEstablishesL2BEndToEnd` | 五条件が揃い、`l2b.claimed=true`・`unmet`空・`qualified=false`・exit 0・artifact存在 |
| `TestSelfBuildWithoutTheRebuildCheckNamesTheMissingCondition` | `-rebuild-check`を外すと`claimed=false`になり、`unmet`が**再現の欠如を名指しする** |

## 観測結果

| 項目 | 値 |
| --- | --- |
| 作業領域 | 隔離worktree（`origin/main` = `8dd12cdfc`） |
| OS / Go | WSL2 Ubuntu 22.04、`go1.27.3 linux/amd64` |

| 試験 | 結果 | 所要 |
| --- | --- | --- |
| `TestSelfBuildEstablishesL2BEndToEnd` | **PASS** | 5.44秒 |
| `TestSelfBuildWithoutTheRebuildCheckNamesTheMissingCondition` | **PASS** | 4.57秒 |

`go test ./cmd/tei -count=1 -run 'EstablishesL2B|WithoutTheRebuildCheck'` が `ok` で終わりました。**SKIPではありません**（Goツールチェーンが無い環境では明示的にskipします）。

## この試験が保証すること

- 五条件が**リポジトリ内の材料だけで**同時に成立する
- どれか一つが壊れたら、**名指しされた条件**として落ちる（`l2b.unmet`を失敗メッセージに出す）
- `go test ./...`に含まれるため、**`tools/repository-health-check.sh`の経路で自動的に回る**

## この試験が保証しないこと

- **別ホスト・別Go版**での成立（§4.21の範囲）
- **catalogが正しいこと**（参照catalogは参照実装用のデータである）
- 受理・Runtime Authority・Trust
- **既定の試行**でClaimが立つこと。既定はin-processでも`-rebuild-check`でもない（§4.23末尾）

## 落ちたときの読み方

`l2b.unmet`の文字列がそのまま失敗メッセージに入ります。どの条件が欠けたかが試験の出力だけで分かります。

## 作業環境について

この作業中、使用していたworktreeが**外部の後片付けで削除**され、未コミットの作業が一度失われました。**同じ内容を作り直して復旧**しています。復旧後の検証結果を上に記しています。

## 検証

- `gofmt -l cmd/tei`：差分なし／`go vet ./cmd/tei`：PASS
- 上記2件の試験：PASS（10.2秒）
- `go test ./tools/repository-plan-check -count=1 -timeout 1800s`：PASS
- `doc-reference-probe` 未解決参照0件／文字化け検査 該当なし

**NOT RUN**：Windows、別Go版、全体`go test ./cmd/tei`（CIに委ねる）、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.24
- 述語：[同 §4.23](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
- 述語の実測：[`TEI_GO_SELF_BUILD_L2B_CLAIM_2026-11-09.md`](TEI_GO_SELF_BUILD_L2B_CLAIM_2026-11-09.md)
