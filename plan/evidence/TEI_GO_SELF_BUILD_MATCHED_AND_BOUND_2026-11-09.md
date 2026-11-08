# `planCrossCheck=matched`と`planArtifactBinding=bound`の同時実測（§4.11）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §7が残作業に挙げた**「`matched`と`bound`の同時実測」**を、1回の試行で観測した記録です。

**L2-bは依然として成立しません。** 本記録が増やすのは「計画突合と計画・成果物の結合が、同じ試行で同時に成立した」ことです。

## これまでとの違い

| 記録 | 観測したもの |
| --- | --- |
| [`TEI_GO_SELF_BUILD_PLAN_MATCHED_2026-11-09.md`](TEI_GO_SELF_BUILD_PLAN_MATCHED_2026-11-09.md) | P-1のchildで`planCrossCheck=matched` |
| `§4.11`（[#3638](https://github.com/hat47x/tei-platform/pull/3638)） | `planArtifactBinding`の実装。単体試験のみで、実プロセスの`bound`は未観測 |
| **本記録** | **同じ1回の試行で`matched`と`bound`の両方** |

## 実行条件

| 項目 | 値 |
| --- | --- |
| checkout | `origin/main` = `04e3089bb`（§4.11統合後） |
| OS / Go | WSL2 Ubuntu 22.04、`go1.27.3 linux/amd64` |
| 宣言 | `outputKind: plugin-package`のprobe宣言（`plugin tei.plugin.mb_probe v1.0.0`） |
| 解決child | `tools/reference-build-authoring-intent`、`sha256:90865a1a…` |
| builder child | `tools/reference-plugin-package-build-child`（P-1）、`sha256:c83c7a1d…` |
| 計画 | 解決childのreceiptから`plan`を取り出して`-plan`で渡す（`planFingerprint=6ab80253…`） |
| state | `/tmp/mb/state`（1回の試行で独立） |

`tei`実行物は`sha256:8ccddd91…`。builderのargvは`-plan <plan.json> -plugin <plugin.tei> -artifact-out <package.json> -state <state>`で、entryが`-artifact-out`と`-state`を付けたことを記録に残しています。

## 観測結果

| 欄 | 値 |
| --- | --- |
| `ok` | `true` |
| `qualified` | **`false`** |
| `planReceipt.state` | `resolved`（`authority: derived-cache`） |
| `planReceipt.receiptDigest` | `sha256:ab9f2680…` |
| `planReceipt.planFingerprint` | `6ab80253f55da1634bb9a5079813584e4938f21eb4ff387e508992aad8ba3f48` |
| `planRecompute.state` | `matched`（再計算値が名乗りと一致） |
| **`planCrossCheck.state`** | **`matched`**（`matched=true`, `l2bClaimed=false`） |
| `planCrossCheck.builderReportedPlanFingerprint` | `6ab80253…`（解決計画と同じ） |
| **`planArtifactBinding.state`** | **`bound`**（`matched=true`, `l2bClaimed=false`） |
| `planArtifactBinding.artifactDigest` | `sha256:376e2e77…` |
| `planArtifactBinding.artifactDigestOnDisk` | `sha256:376e2e77…`（**ディスク実体と一致**） |
| `planArtifactBinding.lockFingerprint` | `581c02e5…` |
| `childRun.artifactCrossCheckMatched` | `true` |
| `childRun.invoked.exitCode` | `0` |

**artifact照合・計画突合・計画と成果物の結合が、同じ1回の試行で同時に成立しました。** §4.10.2でP-3（childを2つに増やす）を避けた理由（1 child・1 artifactの契約維持）を、実際に満たしています。

## この同時成立が言えること・言えないこと

言える。

- 解決childの計画と、builder childが実際に読み込んだ計画の照合値が一致した
- その計画fingerprintと、builder childが書いたartifactのdigestを**1つの検証済み`BuildArtifactLock`として束ねられ**、lockが`internal/plugin`の検証を通った
- artifactのdigestはディスクのバイトと独立に一致した

言えない。

- **L2-b（解釈の成立）**。`l2bClaimed`は`planCrossCheck`も`planArtifactBinding`も`false`のまま。理由は次の二点
  - **`intentByteCheckPerformed=false`**。builder childは`-plan`のファイルを読んでおり、**entryが保持した宣言そのものではない**。宣言と計画の結び付けは`intentBinding`の範囲で、今回は成立していない
  - lockに載る計画fingerprintとartifact digestは、**どちらもbuilder childの自己申告**である。entryが独立に確かめたのは「digestがディスクと合う」「lockが検証を通る」だけ
- 意味の受理・Runtime Authority・供給のTrust・Seed Trust

## 次に詰めること

設計§4.11の末尾が挙げた**「builderが自分で`BuildArtifactLock`を作って返す経路」**である。lockの二つの値をbuilder自身が自分の内部状態から作れば、entryが組み立てる自己申告の束ねではなくなる。あわせて`intentBinding`の(a)（builderが読んだ宣言byteの申告）を埋める必要がある。

## 再現に使ったprobe宣言

`tools/`の固定fixtureは`outputKind: runtime-artifact`で、P-1のchildが配布物を作れない。この実測では次の2ファイルを用意した（カタログは`tools/reference-build-authoring-intent/testdata/catalog.json`をそのまま使用）。

```json
{
  "formatVersion": 1,
  "subject": {"kind": "plugin", "id": "tei.plugin.mb_probe", "version": "1.0.0", "definitionSource": "definition"},
  "sources": [{"kind": "plugin-definition", "id": "definition", "ref": "plugin.tei"}],
  "builder": {"contract": "tei.plugin-build", "compatibleMajor": 1},
  "recipe": {"kind": "go-package", "entrypoint": "plugin.tei"},
  "outputKind": "plugin-package",
  "target": "host/linux/amd64",
  "profile": "release"
}
```

```text
plugin tei.plugin.mb_probe v1.0.0 {
    runtime builtin
    provides deployment-executor runtime.mb_probe {
        action application-start
    }
}
```

手順は3段である。

1. 解決childで宣言を計画へ解決し、receiptの`plan`を取り出して`plan.json`へ書く
2. `tei self-build`を`-intent-resolver`付きで起動し、builder childには`-child-arg`で`-plan`と`-plugin`を渡す
3. 出力の`planCrossCheck`と`planArtifactBinding`を読む

## 検証

- `go build ./cmd/tei`・`go build ./tools/reference-plugin-package-build-child`：PASS
- 解決childの実行：exit 0、`planFingerprint=6ab80253…`、`outputKind=plugin-package`
- `tei self-build`の実行：exit 0、`ok=true`、`planCrossCheck=matched`、`planArtifactBinding=bound`
- `go test ./tools/repository-plan-check -count=1`：PASS

**NOT RUN**：Windows（同じprobe宣言での再観測）、二回の独立試行での比較、別Go版、builder側lock生成、CI。

## 限界

- **1回の観測である。** 二回の独立試行での比較（SB-N8の形）は別途必要。
- **probe宣言は`tools/`の固定fixtureではなく、この実測のために用意した。** `outputKind: plugin-package`の宣言が必要だったため。カタログは固定fixtureをそのまま使っている。
- `intentBinding`のbyte検査は未実施のまま（`intentByteCheckPerformed=false`）。
- G4は`UNQUALIFIED`のまま。
