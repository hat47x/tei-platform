# `tei self-build` の L2-a を現行 main で取り直す（独立再観測）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.7・§4.8について、[既存の記録](TEI_GO_SELF_BUILD_INTENT_RESOLUTION_2026-11-09.md)が報告した値を、**別の作業領域・別のbuilder childで**取り直した記録です。

**L2は成立していません。** 本記録が増やすのは**L2-a（受け渡し）**の再現性であり、**L2-b（解釈）**は`l2Claimed: false`のままです。

## 既存記録との違い

| 項目 | 既存記録（`fef1f7e0e`） | 本記録（現行 main） |
| --- | --- | --- |
| 対象main | `fef1f7e0e` 時点 | `93a856401`（+ `bec7f6cb9` の重複宣言修正後） |
| builder child | `tools/reference-go-build-with-supply`（実Go Build） | **自作の最小probe**（artifactを書いて自分のdigestを申告するだけ） |
| 確かめたこと | resolver receiptの読み取りと拒否条件 | **同じ宣言から同じ`planFingerprint`が出るか、entryが何を記録するか** |

builder childを差し替えたのは、**entryの記録がbuilderの種類に依存しないこと**を確かめるためです。実Go Buildを走らせると「Buildが成功した」ことが混ざり、entryの記録だけを見分けにくくなります。

## 実行条件

| 項目 | 値 |
| --- | --- |
| 作業領域 | 隔離worktree（`origin/main` = `93a856401`、`cmd/tei` の重複宣言修正後） |
| OS / Go | WSL2 Ubuntu 22.04、`go1.27.3 linux/amd64` |
| `tei` 実行物 | `/tmp/l2r-tei`、`sha256:96b16603…` |
| 解決child | `tools/reference-build-authoring-intent` のビルド、`sha256:abb0f12b33f2b6e5af68820210a3ab3386a08b0e4a0baeb15bd47b09e6244acd` |
| builder child | 自作probe（`-artifact-out`へ`probe-artifact`を書き、そのsha256を申告） |
| 宣言 | `tools/reference-build-authoring-intent/testdata/intent.json`、`sha256:6660187445ea7ee63f0aef64c7c31ff7ae95d1b68ed3d4b0c8eb16e2a3cc9775` |
| 期待plan fingerprint | `38c999fbc35263d55fb4cb7b26e43b97c3303c651e3e65ca655e3f2c6aff8169`（既存記録と同じ値） |
| state | 試行ごとに独立（`/tmp/l2probe/state`） |

## 観測結果

| 欄 | 値 |
| --- | --- |
| `ok` | `true` |
| `qualified` | **`false`** |
| `intentDeclaration.digest` | `sha256:6660187445ea7ee63f0aef64c7c31ff7ae95d1b68ed3d4b0c8eb16e2a3cc9775`（entryが保持したbyte） |
| `intentDeclaration.resolved` | `false`（entryは解釈しない） |
| `intentBinding.intentBytesDigest` | 同じ`sha256:66601874…`（byte束縛の対象） |
| `intentBinding.intentByteCheckPerformed` | `false`（builder childは宣言byteを申告しない） |
| `intentBinding.l2bClaimed` | **`false`** |
| `planReceipt.state` | `resolved` |
| `planReceipt.authority` | `derived-cache` |
| `planReceipt.intentFingerprint` | `c22275d8f1e4c806a91aa0835a433b3bca0b01d60b5c40652ab024cd71ebeab8` |
| `planReceipt.planFingerprint` | `38c999fbc35263d55fb4cb7b26e43b97c3303c651e3e65ca655e3f2c6aff8169` |
| `planReceipt.expectedPlanFingerprint` | 同上（呼び手の宣言値） |
| `planReceipt.planMatchesExpected` | `true` |
| `planReceipt.l2Claimed` | **`false`** |
| `planReceipt.receiptDigest` | `sha256:37df7ecca389c9582ab5024c3b9a59f6a89acb64679e03da5e5e1be90fae3f33` |
| `childRun.artifactCrossCheckMatched` | `true` |
| `childRun.childArtifactDigest` | `sha256:9336bfa21a14640f0aba0adbf8ba82bc086630351f89b5c9edf6d80e68e35381` |

**`planFingerprint` と `receiptDigest` は既存記録と同じ値です。** builder childを差し替えても、宣言とカタログとSourceが同じなら同じ計画が出ることを、別の作業領域で確かめました。

**`l2Claimed` と `qualified` は両方`false`のままです。** entryは「同じ宣言を渡し、宣言済みのplan fingerprintと比べた」ことしか言っておらず、builder childが同じ読みをした証明にはならないためです。

## 途中で見つけた実装上の注意

最初の試行ではbuilder childを`-artifact`で起動してしまい、`artifact cross-check could not read the artifact` で `ok=false` になりました（`childArgs`は`-artifact-out`を渡す必要がある）。これはentryの欠陥ではなく**呼び手の引数誤り**ですが、`childRun`に失敗理由が残るため、**記録から原因を追える**ことを確認できました。

## 検証

- `go build ./cmd/tei`：PASS（重複宣言の修正後）
- `go build ./tools/reference-build-authoring-intent`：PASS
- 上記1回の実行：`ok=true`、exit 0

## 限界

- **L2-bは未成立。** entryが受け取ったのは解決childのreceiptであり、builder childの読みではない
- **`intentByteCheckPerformed=false`。** builder childが宣言byteを申告しないため、SC0-13の突合は実施されていない
- **`expected-plan-fingerprint`はentryの外から来る。** entryは一致を確認するだけで、その値の出所を検証しない
- 供給のTrust・署名、`go.sum`のTrust根拠、Seed Trustは未確認
- Windows実行、別Go版は未確認

## 参照

- 既存の記録：[`TEI_GO_SELF_BUILD_INTENT_RESOLUTION_2026-11-09.md`](TEI_GO_SELF_BUILD_INTENT_RESOLUTION_2026-11-09.md)
- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.7・§4.8
- 台帳：[`TEI_SELF_BUILD_P0_EVIDENCE_LEDGER_2026-11-08.md`](TEI_SELF_BUILD_P0_EVIDENCE_LEDGER_2026-11-08.md)
