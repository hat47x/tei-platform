# `tei self-build`の計画値をWindowsとLinuxで再観測する

[#2831](https://github.com/hat47x/tei-platform/issues/2831) の台帳§2.4が残差に挙げていた**「別OS・別toolchainでの再観測」**のうち、OS軸を一つ埋めた記録です。

**L2-bは成立しません。** 観測したのは計画側の値がOSを跨いで一致することであり、builderが計画を解釈した証拠ではありません。

## 観測条件

| 項目 | Windows | Linux |
| --- | --- | --- |
| OS / Go | `windows/amd64`、go1.27.0 | `linux/amd64`、go1.27.3 |
| checkout | `tei-wt-xos`（main `a2d2fec00`） | 同じcheckout（WSL2経由） |
| 解決child | `tools/reference-build-authoring-intent`（その場でビルド） | 同じ |
| builder child | `tools/reference-go-build-with-supply`（対象`./tools/reference-build-authoring-intent`） | 同じ |
| 宣言 | `testdata/intent.json` | 同じ |
| 供給 | `~/go/pkg/mod/cache/download` | 同じ |

## 結果

| 欄 | Windows | Linux | 一致 |
| --- | --- | --- | --- |
| `intentBinding.intentBytesDigest` | `sha256:66601874…` | `sha256:66601874…` | **一致** |
| `planReceipt.receiptDigest` | `sha256:37df7ecc…` | `sha256:37df7ecc…` | **一致** |
| `planReceipt.planFingerprint` | `38c999fb…` | `38c999fb…` | **一致** |
| `planRecompute.recomputedFingerprint` | `38c999fb…` | `38c999fb…` | **一致** |
| `planRecompute.state` | `matched` | `matched` | **一致** |
| `planCrossCheck.state` | `unperformed` | `unperformed` | **一致** |
| `qualified` | `false` | `false` | 一致 |
| `invokerDigest`（`tei`自身） | `sha256:614796e6…` | `sha256:e82eae94…` | 不一致（OSごとのバイナリ） |
| `childRun.invoked.executableDigest` | `sha256:88dd2a94…` | `sha256:853d84d4…` | 不一致（OSごとのバイナリ） |
| `childRun.invoked.stdoutDigest` | `sha256:0c054d0c…` | `sha256:2041f801…` | 不一致（絶対pathを含む出力） |
| `childRun.artifactCrossCheckedDigest` | `sha256:43f663e4…` | `sha256:5bb9f33b…` | 不一致（OSごとのartifact） |

**計画側の値はOSを跨いで一致し、実行体とartifactのdigestは一致しない。** これは期待どおりである。`planFingerprint`は`plugin.BuildPlan`の正規化JSONのdigestであり、OS固有の値を持たない。artifactは`GOOS`/`GOARCH`ごとに別のバイナリなので一致しない。

## 何を言えるか・言えないか

| 言えること | 言えないこと |
| --- | --- |
| 同じ宣言と同じ解決childから、**OSに依存しない計画fingerprint**が出る | builderがその計画を解釈したこと（`planCrossCheck`は両OSで`unperformed`） |
| entryの再計算が**両OSで同じ値**を出し、`matched`になる | artifactがその計画の出力であること（§4.8の(c)は未観測） |
| 実行体とartifactのdigestはOSごとに異なる | 供給のTrust・署名（未検証） |

**「両OSで`matched`」は、L2-bの成立ではない。** 一致したのは計画の指紋であり、`planCrossCheck`（builderの申告との突合）は両OSとも`unperformed`のままである。これは§4.10のP-1が未了であることと整合する。

## 検証

- Windows: `go build`（`tei`・builder・resolver）PASS、`tei self-build` exit 0、`ok=true`
- Linux（WSL2）: 同じくビルドPASS、exit 0、`ok=true`
- 両OSで`gofmt`差分なし（コード変更を含まない記録のみ）

**NOT RUN**：Windows CIでの再現（本記録は手元の実測）、macOS、別toolchain（go1.26系）、P-1実装後の再観測。

## 限界

- **1回の観測である。** 二回の独立試行での比較は別途（SB-N8の形）が必要。
- 本記録はEvidenceのみで、コードも設計も変えない。
- G4は`UNQUALIFIED`のまま。
