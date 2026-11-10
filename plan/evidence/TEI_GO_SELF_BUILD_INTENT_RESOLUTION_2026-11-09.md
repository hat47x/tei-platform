# `tei self-build`のIntent解決（§4.7 I-2）とL2の切り分け（§4.8）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.7で既定案にした**I-2（`cmd/tei`が参照実装をchildとして起動する）**を実装した記録です。

**L2は依然として成立していません。** この記録が増やすのは「宣言を渡し、解決childのreceiptを受け取った」という**L2-a（受け渡し）**の材料です。**L2-b（解釈）**は`l2Claimed: false`のままです。

## 実装

`tei self-build`に3つのflagを足しました。

| flag | 意味 |
| --- | --- |
| `-intent-resolver` | 宣言を解決する**exactな実行ファイル**。childとして起動する |
| `-intent-resolver-arg` | 解決childへ渡す追加引数（JSON配列） |
| `-expected-plan-fingerprint` | 呼び手が別経路で宣言した計画fingerprint。一致しなければ停止 |

`-intent-resolver`を指定すると、`-intent-file`の宣言を渡して解決childを起動し、その**stdoutのJSON receipt**を読んで`planReceipt`欄へ記録します。

### 拒否条件

| 条件 | コード | exit |
| --- | --- | --- |
| 解決childが非0終了 | `SELF_BUILD_INTENT_RESOLUTION_FAILED` | 2 |
| receiptがJSONでない／`planFingerprint`が無い | 同上 | 2 |
| receiptが`authority: derived-cache`を名乗らない | 同上 | 2 |
| `-expected-plan-fingerprint`と一致しない | `SELF_BUILD_PLAN_MISMATCH` | 2 |

いずれも**builder childを起動する前**に止まります。

## 設計上の判断

1. **`cmd/tei`は宣言を解釈しない。** §4.7の実測どおり、`tools/reference-build-authoring-intent`は`package main`でimportできず、`tei/internal`も`cmd/`から届かない。よって解決は**subprocessへ委譲**し、entryはreceiptを読むだけにした。二重実装（I-3）を作らない。
2. **未知のauthorityは証拠として扱わない。** `derived-cache`を名乗らないreceiptは「解決済み計画」と読まない。形式を知らない出力を成功として扱わないためである。
3. **宣言をbuilderのargvへ注入しない。** 最初の実装は`-intent <path>`をchildへ足していたが、実測で`tools/reference-go-build-with-supply`が`flag provided but not defined: -intent`で落ちた。**entryはchildの引数を発明しない**方針へ直し、byte突合は呼び手が`-child-arg`で報告引数を渡す形にした（渡さない場合は「なぜ検査しなかったか」を記録する）。
4. **L2を二層に分けたまま記録する。** `planReceipt`は`l2Claimed: false`と`bindingNote`（「entryは同じ宣言を渡し、宣言済みのplan fingerprintと比べただけ。builder childが同じ読みをしたことは証明できない」）を必ず含む。

## 実測（WSL2 Ubuntu 22.04 / Go 1.27.3 linux/amd64）

解決childは`tools/reference-build-authoring-intent`（`testdata/{intent.json,catalog.json,src}`）、builder childは`tools/reference-go-build-with-supply`（対象`./tools/reference-build-authoring-intent`）。

| 実行 | 結果 |
| --- | --- |
| 1. 解決childとbuilderを起動、宣言期待なし | `ok=true`、`state=resolved`、`authority=derived-cache`、`planFingerprint=38c999fb…`、`planMatchesExpected=false`（期待を宣言していない）、`l2Claimed=false`、`crossCheck=true` |
| 2. 上のplanFingerprintを`-expected-plan-fingerprint`で宣言 | `ok=true`、`planMatchesExpected=true`、`l2Claimed=false`、`crossCheck=true` |
| 3. 一致しない期待を宣言 | `SELF_BUILD_PLAN_MISMATCH`、**`childRun`は無し**（builderは起動していない） |
| 4. 解決childの`-source-root`を誤る | `SELF_BUILD_INTENT_RESOLUTION_FAILED`、`resolver exited with 1`（source path解決の失敗がそのまま出る） |

`resolverDigest`は`sha256:0d41d4a83899ac6dca14f8693d3d5da9c4337e37c66d5be8341927cf2928cb76`、receipt digestは`sha256:37df7ecc…`。

## 検証

- `gofmt -l cmd/tei`：差分なし
- `go vet ./cmd/tei`：PASS
- `go test ./cmd/tei -count=1 -timeout 900s`：PASS（243秒、既存の外形・宣言・供給試験を含む）

## 限界（L2を主張しない理由）

- **L2-bは未成立。** entryが受け取ったのは**解決childの**receiptであり、**builder child**が同じ宣言から同じ計画を導いた証明にはならない。両者は別プロセスで、builder側の読みは`-child-arg`で渡した報告引数に依存する。
- **宣言と計画の突合は呼び手の宣言値に依存する。** `-expected-plan-fingerprint`はentryの外から来る。entryは一致を確認するだけで、その値が正しい出所かは検証しない。
- 供給のTrust・署名、`go.sum`のTrust根拠、Seed Trustは未。
- catalogへ載せていないため`tei commands`からは見えません（D-3）。G4は`UNQUALIFIED`のまま。
