# `tei self-build` の計画突合を実プロセスで3状態そろえて観測する

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.8について、[既存の記録](TEI_GO_SELF_BUILD_PLAN_CROSS_CHECK_2026-11-09.md)が「実プロセスでは`unperformed`しか観測できない」としていた制約を埋めた記録です。

**L2-bは依然として成立しません。** 本記録が増やすのは、**計画を報告するbuilder childを用意したときに、`matched`・`mismatched`・`unperformed`の三状態が実プロセスで区別して観測できる**ことです。

## 既存記録の制約と、それをどう外したか

既存記録の限界は次の一文でした。

> builderの`planFingerprint`は自己申告。`reference-go-build-with-supply`はこの欄を返さないため、実測の実プロセスでは`unperformed`になる。

`matched`と`mismatched`は単体試験のfixtureでのみ確かめられていました。本記録では**計画fingerprintを報告するbuilder probeを自作**し、実プロセスで三状態を出しました。

| 項目 | 値 |
| --- | --- |
| 作業領域 | 隔離worktree（`origin/main` = `04ea1470b`） |
| `tei` 実行物 | `/tmp/planx-tei`、`sha256:50edda87c23452e11aab38caf2cd0f307c6900d42fc9f39391ebb31574cdfc40` |
| 解決child | `tools/reference-build-authoring-intent`、`sha256:abb0f12b33f2b6e5af68820210a3ab3386a08b0e4a0baeb15bd47b09e6244acd` |
| builder child | 自作probe（artifactを書き、`-report-plan-fingerprint`の値をそのまま申告する） |
| 宣言 | `testdata/intent.json`（既存記録と同じ） |
| 解決計画fingerprint | `38c999fbc35263d55fb4cb7b26e43b97c3303c651e3e65ca655e3f2c6aff8169` |

builder probeが計画を「計算」せず**渡された値をそのまま申告する**のは意図的です。この記録が確かめるのは「entryが二つの申告を比べて三状態を区別するか」であり、builderの計画導出そのものではありません。

## 観測結果

| 場合 | `ok` | `state` | `matched` | `l2bClaimed` | 備考 |
| --- | --- | --- | --- | --- | --- |
| builderが解決計画と同じ値を申告 | `true` | `matched` | `true` | `false` | `resolver=38c999fb…` / `builder=38c999fb…` |
| builderが別の値を申告 | **`false`** | `mismatched` | `false` | `false` | `failureCause`に「builder child reported a plan fingerprint that differs…」、exit 1 |
| builderが申告しない | `true` | `unperformed` | `false` | `false` | `unperformedReason`に「did not report the plan fingerprint it built」 |

`qualified`は三場合とも`false`です。

**`mismatched`だけが`ok=false`と終了コード1になります。** 不一致は「停止しなかったが記録された発見」として扱われ、`unperformed`は「比べられなかった」であって失敗ではないため`ok`が保たれます。この区別は既存記録の設計判断1（突合は事後観測であり事前条件ではない）と一致します。

## 既存記録との関係

| 観点 | 既存記録 | 本記録 |
| --- | --- | --- |
| `matched` | 単体試験のfixtureのみ | **実プロセス** |
| `mismatched` | 単体試験のfixtureのみ | **実プロセス**（`ok=false`・exit 1を確認） |
| `unperformed` | 実プロセス（報告しないbuilder） | 実プロセスで再確認 |

## 主張しないこと

- **L2-bは未成立。** 二つの自己申告が一致したことは、builderが**このentryの保持した宣言**からその計画を導いた証明ではない
- **artifactと計画の結び付きは未観測。** artifactがその計画の出力であることは言えない
- **builder probeは計画を計算していない。** 渡された値を申告するだけであり、`reference-go-build-with-supply`のような実Buildの計画一致を示す記録ではない
- 供給のTrust・署名、`go.sum`のTrust根拠、Seed Trust、Windows、別Go版は未確認
- catalogへは載せていない（D-3）。**G4は`UNQUALIFIED`のまま**

## 検証

- `go build ./cmd/tei`：PASS
- `go build ./tools/reference-build-authoring-intent`：PASS
- 3回の実行：`matched`（exit 0）、`mismatched`（exit 1）、`unperformed`（exit 0）
- `go test ./tools/repository-plan-check -count=1`：PASS

## 参照

- 既存の記録：[`TEI_GO_SELF_BUILD_PLAN_CROSS_CHECK_2026-11-09.md`](TEI_GO_SELF_BUILD_PLAN_CROSS_CHECK_2026-11-09.md)
- L2-a再観測：[`TEI_GO_SELF_BUILD_L2A_REOBSERVATION_2026-11-09.md`](TEI_GO_SELF_BUILD_L2A_REOBSERVATION_2026-11-09.md)
- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.8
- 台帳：[`TEI_SELF_BUILD_P0_EVIDENCE_LEDGER_2026-11-08.md`](TEI_SELF_BUILD_P0_EVIDENCE_LEDGER_2026-11-08.md)
