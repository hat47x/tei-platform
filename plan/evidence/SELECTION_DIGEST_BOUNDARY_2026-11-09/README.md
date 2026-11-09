# route の Selection が Binding と食い違うとき、Provider は呼ばれない（CON-12c）（2026-11-09）

- 対象: [識別子を持つ層の調査](../BINDING_IDENTITY_LAYERS_2026-11-09/README.md)、[digest 変更時のピン留め](../DIGEST_CHANGE_REUSE_REFERENCE_2026-11-09/README.md)、[C-2 の参照](../../../tools/reference-service-carrier-connect/README.md)
- 位置づけ: **参照実装への試験追加と実行記録**。製品へ繋ぐべきかは示しません。
- 計測した版: `raw/head.txt`（`origin/main` `8492e296` の作業ツリー＋このPRの追加）。

前の記録は、識別子を持つ層が `generation.Binding` と `runtime.ServiceProviderSelection` の2つだと示し、`generation.Binding` の側で CON-12c を確かめました。この記録は、**残る `ServiceProviderSelection` の側**を確かめます。

## 1. 何を確かめたか

`servicehost` は、route を引いた後・Provider を呼ぶ前に、**route の Selection をピン留めした Binding と照合**します。

```go
if route.Binding != lease.Binding ||
    route.Selection.ContractID != lease.Binding.ContractID ||
    route.Selection.ContractVersion != lease.Binding.ContractVersion ||
    route.Selection.ArtifactDigest != lease.Binding.ArtifactDigest {
    return ..., ErrServiceCarrierGenerationMismatch
}
```

この比較は**実装に存在していましたが、試験がありませんでした。**

## 2. 観測

```text
--- PASS: TestARouteSelectionWithAnotherDigestStopsBeforeTheProvider
--- PASS: TestARouteSelectionWithAnotherContractVersionStopsBeforeTheProvider
--- PASS: TestAMatchingSelectionStillRuns
```

| 場合 | Provider 呼出し | ran | unverified | refused |
| --- | --- | --- | --- | --- |
| selection の digest が違う | **0回** | 1 | 1 | 0 |
| selection の contract version が違う | **0回** | 1 | 1 | 0 |
| selection が一致 | 1回 | 1 | 0 | 0 |

## 3. この観測が示すこと

- **CON-12c は `ServiceProviderSelection` の層でも満たされています。** digest が食い違えば、**Provider は一度も呼ばれません**。
- **一致する場合は通ります**（対照）。上の検査が「全部拒否している」わけではありません。
- **食い違いは「拒否」ではなく `ran=1` / `unverified=1` です。** Host guard と lease 取得は通っており、**carrier には入っている**ためです。Provider に届かなかったことは `unverifiedCount` が表します。**この区別は前ラウンドで入れた `carrierInvoked` の意味と一致します。**

## 4. 作業中に自分の誤りを1つ訂正した

最初の試験は「食い違いは拒否（`refused=1`）になる」と期待して書きましたが、**実行すると `ran=1` でした**。コードを読むと `servicehost` は carrier に入った後に照合しており、`ran=1` が正しい挙動です。**期待の側を直しました**（`refused=1` → `ran=1`/`unverified=1`）。

## 5. この記録が示さないこと

- **実 Artifact の digest**。使う値は構造的に妥当な `sha256:` 文字列です。
- **Grant の epoch**。`ServiceProviderSelection` は `Scope` を持ちますが epoch は持ちません。
- **実 Provider の実行**。Process は記録用です。
- **`Scope` の食い違い**。digest と contract version だけを試しました。
- 製品 CLI へ繋ぐべきか（C-1〜C-4 は未決）。

## 6. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（3つの場合の試験） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域 |
| `raw/tests.txt`・`raw/tests-summary.txt` | 試験の出力と要約 |

## 7. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `2a7ec36071dde5fda47321df3fa26219a81f231ef86b2051574e4bf212761cdb` | 1435 |
| `raw/head.txt` | `06f5880721eb46671d57b0d2e9b56e9021a9a61cda754ae0fb39a6e966a72ea0` | 73 |
| `raw/pre-status.txt` | `fbe96092b00db8047ec29e5e95a3a58f6c66fb7cca6c6f6646b5690aaa11182e` | 183 |
| `raw/tests.txt` | `5518cea6998ed35ff14285ecedbdf2faa688bc65bcaf0cf59847a28c5bbfe7cd` | 560 |
| `raw/tests-summary.txt` | `34a7abac00685bcc074ec6d68469dafb9b078d9e1b11e7596773a2c331b223b0` | 458 |
