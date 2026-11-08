# `matched`と`bound`を同じ1回の試行で観測する（§4.11の同時実測）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.11が残作業とした「`planCrossCheck=matched`と`planArtifactBinding=bound`の同時実測」を行った記録です。

**L2-bは依然として成立しません。** `l2bClaimed`は`false`のままです。本記録が増やすのは、**計画の一致と計画とartifactの結合が、同じ1回の`tei self-build`で同時に成立すること**です。

## なぜ「同時」が要るか

`planCrossCheck`と`planArtifactBinding`は別の欄であり、別々の試行で確かめても「同じ試行で両方が成り立つ」ことは言えません。片方だけが成り立つ組み合わせ（計画は一致したがartifactへ束ねられない、逆に束ねられたが計画が一致しない）は、**読み違えの余地**を残します。同時に観測することで、その余地を狭めます。

## 実行条件

| 項目 | 値 |
| --- | --- |
| 作業領域 | 隔離worktree（`origin/main` = `dff9f0cd0`） |
| OS / Go | WSL2 Ubuntu 22.04、`go1.27.3 linux/amd64` |
| `tei` 実行物 | `sha256:79e85bcfbb0e32a8be67f6c39d033f4750d3b6ee45e0b09a4d5cc36c829c9bda` |
| 解決child | `tools/reference-build-authoring-intent`、`sha256:abb0f12b33f2b6e5af68820210a3ab3386a08b0e4a0baeb15bd47b09e6244acd` |
| builder child | `tools/reference-plugin-package-build-child`、`sha256:b3646a6aab02e675fd7595ee7dadea40d4786d602a6799d3edc4e5725ba677fd` |
| 宣言 | `outputKind: plugin-package`のprobe宣言（`plugin tei.plugin.both_probe v1.0.0`） |
| カタログ | `tools/reference-build-authoring-intent/testdata/catalog.json` |
| 解決計画fingerprint | `2e51544f8daad6d60a0c5b30b85e71cd084242fda83be1b1bbc226fff43262d5` |
| state | 試行ごとに独立（`/tmp/both/state`） |

宣言は固定fixture（`testdata/intent.json`、`outputKind: runtime-artifact`）ではなく、**`outputKind: plugin-package`のprobe宣言**を使いました。builder childが配布物を作るためです。

## 観測結果

| 欄 | 値 |
| --- | --- |
| `ok` | `true` |
| `qualified` | **`false`** |
| `planReceipt.state` | `resolved` |
| `planReceipt.planMatchesExpected` | `true` |
| `planRecompute.state` / `matched` | `matched` / `true` |
| **`planCrossCheck.state` / `matched`** | **`matched` / `true`** |
| `planCrossCheck.l2bClaimed` | **`false`** |
| **`planArtifactBinding.state` / `matched`** | **`bound` / `true`** |
| `planArtifactBinding.planFingerprint` | `2e51544f…`（解決計画と同一） |
| `planArtifactBinding.artifactDigest` | `sha256:c2164a594e91edde3f2f451767ad07509d0286a56b94e5491a8e10254af5117b` |
| `planArtifactBinding.artifactDigestOnDisk` | **同じ値**（ディスクのバイトと一致） |
| `planArtifactBinding.lockFingerprint` | `e8c3c549b75317bb22773ae65ede60c46284c82d953cd7c6dae076f9403d3d3e` |
| `planArtifactBinding.l2bClaimed` | **`false`** |
| `intentBinding.intentByteCheckPerformed` | `false` |
| `childRun.artifactCrossCheckMatched` | `true` |

`artifactCrossCheckMatched`、`planArtifactBinding.artifactDigest`、`artifactDigestOnDisk`はすべて同じ値です。**childの申告・lockの記載・ディスクの実体が三者一致しています。**

## この同時観測が言えること

- 同じ1回の試行で、計画の一致（`planCrossCheck=matched`）、receiptの内部整合（`planRecompute=matched`）、計画とartifactの検証済み結合（`planArtifactBinding=bound`）が**同時に成立した**
- `bound`のartifact digestは、childの申告とディスクの実体が一致した

## この同時観測が言えないこと

- **L2-b（解釈の成立）。** `l2bClaimed`は三つの欄すべてで`false`である
  - builder childが読んだのは`-plan`のファイルであり、**entryが保持した宣言そのものではない**。この試行でも`intentByteCheckPerformed=false`（childが宣言byteを申告しない）
  - (c)が本来問う「**その計画がそのartifactを生んだか**」は`bound`では出ない。lockに載る二つの値はどちらもbuilder childの自己申告であり、entryが独立に確かめたのは「digestがディスクと合う」「lockが`internal/plugin`の検証を通る」ことだけである（§4.11）
- 意味の受理・Runtime Authority・供給のTrust・Seed Trust
- Windows、別Go版

## 次に詰めること

§4.11の末尾が挙げた**builder自身が`BuildArtifactLock`を作って返す経路**である。今はentryが二つの自己申告からlockを組むため、builderが申告を偽れば`bound`は成立してしまう。builderが自分でlockを`Validate`して返せば、少なくとも**そのプロセス内での整合**はbuilder側の責任で証明される。

## 再現に使ったprobe宣言

固定fixture（`testdata/intent.json`）は`outputKind: runtime-artifact`で、P-1のchildが配布物を作れない。この記録では次の2ファイルを用意した（カタログは`tools/reference-build-authoring-intent/testdata/catalog.json`をそのまま使用）。

```json
{
  "formatVersion": 1,
  "subject": {"kind": "plugin", "id": "tei.plugin.both_probe", "version": "1.0.0", "definitionSource": "definition"},
  "sources": [{"kind": "plugin-definition", "id": "definition", "ref": "plugin.tei"}],
  "builder": {"contract": "tei.plugin-build", "compatibleMajor": 1},
  "recipe": {"kind": "go-package", "entrypoint": "plugin.tei"},
  "outputKind": "plugin-package",
  "target": "host/linux/amd64",
  "profile": "release"
}
```

```text
plugin tei.plugin.both_probe v1.0.0 {
    runtime builtin
    provides deployment-executor runtime.both_probe {
        action application-start
    }
}
```

手順は3段である。

1. 解決childで宣言を計画へ解決し、receiptの`plan`を取り出して`plan.json`へ書く
2. `tei self-build`を`-intent-resolver`付きで起動し、builder childには`-child-arg`で`-plan`と`-plugin`を渡す
3. 出力の`planCrossCheck`と`planArtifactBinding`を読む

`planCrossCheck`と`planArtifactBinding`は別の欄なので、**片方だけが成り立つ組み合わせ**（計画は一致したがartifactへ束ねられない、逆に束ねられたが計画が一致しない）を同時観測で排除できる。

## 検証

- `go build ./cmd/tei`：PASS
- `go build ./tools/reference-build-authoring-intent`：PASS
- `go build ./tools/reference-plugin-package-build-child`：PASS
- 上記1回の実行：`ok=true`、`planCrossCheck=matched`、`planArtifactBinding=bound`、exit 0
- `go test ./tools/repository-plan-check -count=1`：PASS

**NOT RUN**：二回目の独立試行、Windows、別Go版、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.8・§4.11
- 計画とartifactの結合：[`TEI_GO_SELF_BUILD_PLAN_ARTIFACT_BINDING_2026-11-09.md`](TEI_GO_SELF_BUILD_PLAN_ARTIFACT_BINDING_2026-11-09.md)
- 計画突合の実測：[`TEI_GO_SELF_BUILD_PLAN_MATCHED_2026-11-09.md`](TEI_GO_SELF_BUILD_PLAN_MATCHED_2026-11-09.md)
- 台帳：[`TEI_SELF_BUILD_P0_EVIDENCE_LEDGER_2026-11-08.md`](TEI_SELF_BUILD_P0_EVIDENCE_LEDGER_2026-11-08.md)
