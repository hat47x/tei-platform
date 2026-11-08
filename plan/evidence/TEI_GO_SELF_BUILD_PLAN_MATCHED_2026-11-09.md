# `tei self-build` の計画突合を実測で`matched`にする（§4.10 P-1）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.10が次の一歩とした**P-1**を実装し、**実プロセスで`planCrossCheck.state=matched`を観測**した記録です。

**L2-bは依然として成立しません。** 本記録が増やすのは「計画を消費するchildが、解決childと同じplan fingerprintを名乗った」ことです。

## なぜ新しいchildが必要だったか

[§4.10.1](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)の走査で、`tei self-build`のchild文法（`-artifact-out`・`-state`・stdoutへ1件のJSON）を満たすtoolは`reference-go-build-with-supply`だけで、これが`planFingerprint`を名乗らないことが分かっていました。計画を名乗るtoolはchild文法を持たない。この非対称が`unperformed`の実質的な原因でした。

そこで**child文法を満たしつつ計画を消費するchild**を新設しました。

| 項目 | 値 |
| --- | --- |
| 新設child | [`tools/reference-plugin-package-build-child`](../../tools/reference-plugin-package-build-child/README.md) |
| 消費する型 | `internal/pluginのBuildPlan`（`Validate`と`BuildPlanFingerprint`を通す） |
| artifact | `plugin.MaterializePluginPackage`の出力（Plugin配布物） |

**P-2（既存childへ欄を足す）を採らなかった理由は変わりません。** 計画を解釈しないchildに欄を足すと`unperformed`が`matched`に変わるだけで、主張の強さは増えません。このchildは実際に計画を**入力として消費**します。

## 実行条件

| 項目 | 値 |
| --- | --- |
| 作業領域 | 隔離worktree（`origin/main` = `db1d60c82`） |
| OS / Go | WSL2 Ubuntu 22.04、`go1.27.3 linux/amd64` |
| `tei` 実行物 | `sha256:b068a69a1318fb72bee3657f1637c418c410f2071dbca695dbe26908e6363b9b` |
| 解決child | `tools/reference-build-authoring-intent`、`sha256:abb0f12b33f2b6e5af68820210a3ab3386a08b0e4a0baeb15bd47b09e6244acd` |
| builder child | 新設child、`sha256:b3646a6aab02e675fd7595ee7dadea40d4786d602a6799d3edc4e5725ba677fd` |
| 宣言 | `outputKind: plugin-package`のprobe宣言（`plugin tei.plugin.p1_probe v1.0.0`） |
| 解決計画fingerprint | `e007b5bfac71025695a90b3cb9bf36659dbcb18dbc7efe2807a80a808a061cb4` |
| state | 試行ごとに独立（`/tmp/p1/state`） |

宣言は`tools/reference-build-authoring-intent/testdata`の固定fixtureをそのまま使わず、**`outputKind: plugin-package`のprobe宣言を用意**しました。新設childが配布物を作るためです。カタログは同じ固定fixtureを使っています。

## 観測結果

| 欄 | 値 |
| --- | --- |
| `ok` | `true` |
| `qualified` | **`false`** |
| `planReceipt.state` | `resolved` |
| `planReceipt.planFingerprint` | `e007b5bfac71025695a90b3cb9bf36659dbcb18dbc7efe2807a80a808a061cb4` |
| `planReceipt.planMatchesExpected` | `true` |
| **`planCrossCheck.state`** | **`matched`** |
| `planCrossCheck.matched` | `true` |
| `planCrossCheck.resolverPlanFingerprint` | `e007b5bfac…` |
| `planCrossCheck.builderReportedPlanFingerprint` | `e007b5bfac…`（**childが実際に消費した計画の照合値**） |
| `planCrossCheck.l2bClaimed` | **`false`** |
| `childRun.artifactCrossCheckMatched` | `true` |
| `childRun.childArtifactDigest` | `sha256:6a01e53120f907d92fb164e69c8178cce4d98cdfe8b6f34843e3106a4ce870f0` |

**artifact照合と計画突合が同じ1回の試行で両方観測できました。** §4.10.2でP-3（childを2つに増やす）を避けた理由（1 child・1 artifactの契約維持）を、実際に満たしています。新設childがartifactを書く主体であり、かつ計画を名乗るためです。

## この`matched`が言えること・言えないこと

言える。

- 解決childが導いた計画と、builder childが**実際に読み込んで`Validate`と`BuildPlanFingerprint`を通した**計画の照合値が一致した
- その計画から作った配布物のbyte digestが、childの申告とディスクの実体で一致した

言えない。

- **L2-b（解釈の成立）**。`l2bClaimed`は`false`のままである。理由は次の二点
  - builder childが読んだのは`-plan`のファイルであり、**entryが保持した宣言そのものではない**。宣言と計画の結び付けは`intentBinding`の範囲で、この試行では`intentByteCheckPerformed=false`（childが宣言byteを申告しないため）
  - **(c) artifactがその計画の出力であること**は`matched`では出ない。`plugin.LockedBuild`（`PlanFingerprint`と`ArtifactDigest`を束ねる型）のような**別の観測**が要る
- 意味の受理・Runtime Authority・供給のTrust・Seed Trust

## 次に詰めること

§4.10.3が挙げた**(c)** である。計画とartifactを束ねる観測（`plugin.LockedBuild`相当）を`tei self-build`の出力に載せ、`planCrossCheck`とは別の欄として記録する。`matched`を`(c)`の代わりに読ませないためである。

## 検証

- `gofmt -l tools/reference-plugin-package-build-child`：差分なし
- `go vet ./tools/reference-plugin-package-build-child`：PASS
- `go test ./tools/reference-plugin-package-build-child -count=1`：PASS（5件。正常系・不正plan・定義不一致・文法）
- 上記1回の実行：`ok=true`、`planCrossCheck.state=matched`、exit 0
- `go test ./cmd/tei -count=1 -run 'SelfBuild'`：PASS
- `go test ./tools/repository-plan-check -count=1`：PASS

**NOT RUN**：Windows、別Go版、`(c)`の観測、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.9・§4.10
- 新設child：[`tools/reference-plugin-package-build-child`](../../tools/reference-plugin-package-build-child/README.md)
- 既存の計画突合：[`TEI_GO_SELF_BUILD_PLAN_CROSS_CHECK_2026-11-09.md`](TEI_GO_SELF_BUILD_PLAN_CROSS_CHECK_2026-11-09.md)、[`TEI_GO_SELF_BUILD_PLAN_CROSS_CHECK_ACTUAL_2026-11-09.md`](TEI_GO_SELF_BUILD_PLAN_CROSS_CHECK_ACTUAL_2026-11-09.md)
- 台帳：[`TEI_SELF_BUILD_P0_EVIDENCE_LEDGER_2026-11-08.md`](TEI_SELF_BUILD_P0_EVIDENCE_LEDGER_2026-11-08.md)
