# `matched`＋`bound`の二回独立試行での比較（§4.12の再現性）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0で、既存記録が残作業に挙げていた**「二回目の独立試行」**を実行した記録です。

**L2-bは依然として成立しません。** 本記録が増やすのは「同じ宣言と同じchildから、二回の独立試行が同じ束縛に到達する」ことです。

## 観測条件

| 項目 | 値 |
| --- | --- |
| checkout | `origin/main` = `caac97605`（§4.12統合後） |
| OS / Go | WSL2 Ubuntu 22.04、`go1.27.3 linux/amd64` |
| `tei` 実行物 | `sha256:004b0a0b03345615f795f0fbc2481b77b3a93271ab7d84331d71395ef6aa2df7` |
| 解決child | `tools/reference-build-authoring-intent` |
| builder child | `tools/reference-plugin-package-build-child`、`sha256:f600c938…` |
| 宣言 | `outputKind: plugin-package`のprobe宣言（`plugin tei.plugin.rp_probe v1.0.0`） |
| カタログ | `tools/reference-build-authoring-intent/testdata/catalog.json` |
| state / output | **試行ごとに別ディレクトリ**（`/tmp/rp/run1/*`・`/tmp/rp/run2/*`） |

2回の試行は**別プロセス**で、childのPIDも異なります（`480587` と `480611`）。

## 結果

| 欄 | 試行1 | 試行2 | 一致 |
| --- | --- | --- | --- |
| `ok` / `qualified` | `true` / `false` | `true` / `false` | **一致** |
| `planReceipt.receiptDigest` | `sha256:7c163e7d…` | 同じ | **一致** |
| `planReceipt.planFingerprint` | `b3c6cf12…` | 同じ | **一致** |
| `planRecompute.recomputedFingerprint` | `b3c6cf12…` | 同じ | **一致** |
| `planCrossCheck.state` | `matched` | `matched` | **一致** |
| `planArtifactBinding.state` | `bound` | `bound` | **一致** |
| `planArtifactBinding.lockFingerprint` | `d642fc22…` | 同じ | **一致** |
| `planArtifactBinding.childLockMatched` | `true` | `true` | **一致** |
| `planArtifactBinding.artifactDigest` | `sha256:ed9c7979…` | 同じ | **一致** |
| `artifactDigestOnDisk` | `sha256:ed9c7979…` | 同じ | **一致** |
| `childRun.invokerDigest` | `sha256:004b0a0b…` | 同じ | **一致** |
| `childRun.invoked.executableDigest` | `sha256:f600c938…` | 同じ | **一致** |
| `childRun.invoked.pid` | `480587` | `480611` | **異なる（期待どおり）** |

**束縛・指紋・成果物のdigestが、二回の独立試行で完全に一致しました。** 異なるのはPIDと、試行ごとに別にしたstate・outputのパスだけです。

`l2bClaimed`は両試行とも`false`で、`planCrossCheck`・`planArtifactBinding`のどちらも`false`のままです。

## 再現手順

`outputKind: plugin-package`の宣言が要る（固定fixtureの`runtime-artifact`ではP-1のchildが配布物を作れない）。

```json
{
  "formatVersion": 1,
  "subject": {"kind": "plugin", "id": "tei.plugin.rp_probe", "version": "1.0.0", "definitionSource": "definition"},
  "sources": [{"kind": "plugin-definition", "id": "definition", "ref": "plugin.tei"}],
  "builder": {"contract": "tei.plugin-build", "compatibleMajor": 1},
  "recipe": {"kind": "go-package", "entrypoint": "plugin.tei"},
  "outputKind": "plugin-package",
  "target": "host/linux/amd64",
  "profile": "release"
}
```

```text
plugin tei.plugin.rp_probe v1.0.0 {
    runtime builtin
    provides deployment-executor runtime.rp_probe {
        action application-start
    }
}
```

1. 解決childで計画を得て`plan.json`へ書く
2. `tei self-build`を**試行ごとに別の`-state`と`-output`**で2回実行する
3. 二つの出力の`planReceipt`・`planCrossCheck`・`planArtifactBinding`・`childRun`を比べる

## この比較が言えること・言えないこと

| 言えること | 言えないこと |
| --- | --- |
| 同じ宣言と同じchildから、**独立した二回の試行が同じ計画fingerprint・同じlock fingerprint・同じartifact digestに到達する** | L2-b（解釈の成立）。両試行とも`l2bClaimed=false` |
| 試行ごとのstate分離（SC0-03）が機能し、共有stateによる「二回と数える」偽装がない | artifactがその計画の出力であること（(c)が本来問うこと） |
| 解決childのreceipt自体が再現する（`receiptDigest`が一致） | entryが宣言から独立に計画を導いたこと（今は解決childへ委譲している） |

**再現性は「同じ手順が同じ値になる」ことの観測であり、意味の受理でもL2-bでもありません。**

## 検証

- `go build ./cmd/tei`・`go build ./tools/reference-build-authoring-intent`・`go build ./tools/reference-plugin-package-build-child`：PASS
- 二回の独立試行：両方 exit 0、`ok=true`
- `go test ./tools/repository-plan-check -count=1`：PASS

**NOT RUN**：Windows（同じprobe宣言での再観測）、別Go版、三回目以降の反復、builder側lock生成のさらに独立な経路、CI。

## 限界

- **二回の比較である。** 統計的な安定性ではなく、二回が一致したという観測。
- 二回は同じホスト・同じGo版で実行した。**別OS・別toolchainでの一致は別の観測**（OS軸は[別記録](TEI_GO_SELF_BUILD_CROSS_OS_2026-11-09.md)で計画側のみ確認済み）。
- probe宣言は`tools/`の固定fixtureではなく、この実測のために用意した（[同時実測の記録](TEI_GO_SELF_BUILD_MATCHED_AND_BOUND_2026-11-09.md)と同じ形）。
- G4は`UNQUALIFIED`のまま。
