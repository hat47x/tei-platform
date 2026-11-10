# `tei self-build`が解決childの計画指紋を自前で再計算する

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.8の項目2を、**自己申告の記録から検算へ**進めた記録です。

**L2-bは依然として成立しません。** 増えたのは「resolverの算術が合っている」ことの確認であり、**builder childがその計画を建てた証拠ではありません**。

## 何を足したか

解決childのreceiptには`plan`（`plugin.BuildPlan`）が入っている。entryはそれを使って`plugin.BuildPlanFingerprint`を**自分で計算し**、receiptが名乗る`planFingerprint`と比べます。

| `state` | 意味 |
| --- | --- |
| `matched` | 再計算値が名乗りと一致 |
| `mismatched` | 一致しない。`SELF_BUILD_INTENT_RESOLUTION_FAILED`で**builder起動前に停止** |
| `unperformed` | `plan`が無い／`planFingerprint`が無い／`plan`が`BuildPlan`として妥当でない。理由を`unperformedReason`へ |

`unperformed`は**一致として扱いません**。`mismatched`は事前条件の失敗として扱います（resolverの算術はentryが依存している前提だから）。

## 設計上の判断

1. **指紋を複写せず、検算する。** これまではreceiptの`planFingerprint`をそのまま記録していた。receiptが`plan`を運んでいる以上、entryは同じ関数（`internal/plugin`）で再計算できる。**外部から来た値を、検算せずに証拠として扱わない**。
2. **`internal/plugin`の利用は既存経路と同じ。** `cmd/tei`は既に`plugin-lock`系で`tei/internal/plugin`をimportしている（`cli_plugin_lock_input.go`・`cli_result.go`）。新しい依存を足していない。
3. **`mismatched`は停止、`unperformed`は継続。** 算術の不一致はentryの前提を壊す。一方、計画を運ばないreceipt（古い形式や別実装）は「検算できなかった」として記録し、build自体は止めない。
4. **`L2Claimed`は常に`false`。** `planRecomputeBoundary`は「resolverの算術とreceiptの内部整合性を見ただけで、builderがその計画を建てたことではない」と述べる。

## 実測（WSL2 Ubuntu 22.04 / Go 1.27.3 linux/amd64）

対象は`reference/workspace`、解決childは`tools/reference-build-authoring-intent`（`testdata/{intent.json,catalog.json,src}`）、builderは`tools/reference-go-build-with-supply`。

| 欄 | 値 |
| --- | --- |
| `planReceipt.planFingerprint`（名乗り） | `38c999fbc35263d55fb4cb7b26e43b97c3303c651e3e65ca655e3f2c6aff8169` |
| `planRecompute.recomputedFingerprint`（再計算） | 同じ |
| `planRecompute.state` / `matched` | `matched` / `true` |
| `planRecompute.l2bClaimed` | **`false`** |
| `planCrossCheck.state` | `unperformed`（builderが計画を報告しない） |
| `qualified` | **`false`** |

### 退け方を確かめた入力（単体試験）

| 入力 | 結果 |
| --- | --- |
| 本物のresolver receipt | `matched`。`plugin.BuildPlanFingerprint`の再計算値が名乗りと一致 |
| `planFingerprint`だけ改竄したreceipt | `mismatched`、`failureCause`に記録 |
| `plan`が無いreceipt | `unperformed`（`plan`を運ばない） |
| `planFingerprint`が無いreceipt | `unperformed` |
| 妥当でない`plan` | `unperformed`（`Validate`の失敗理由つき） |
| 宣言そのものが無い | 再計算の記録なし（resolver未起動） |

## 検証

- `gofmt -l cmd/tei`：差分なし
- `go vet ./cmd/tei`：PASS
- `go test ./cmd/tei -count=1 -timeout 900s`：PASS（447秒）
- `go test ./cmd/tei -run TestPlanRecompute`：6件PASS

## 限界（L2-bを主張しない理由）

- **builder childの計画申告経路は依然として未了。** 実プロセスの`planCrossCheck`は`unperformed`のままで、`matched`は単体試験のfixtureでしか観測していない。
- **再計算はresolverのreceiptについてだけ。** builderが同じ計画を建てたことも、artifactがその計画の出力であることも示さない。
- 供給のTrust・署名、`go.sum`のTrust根拠、Seed Trustは未。catalogへ載せていない（D-3）。G4は`UNQUALIFIED`のまま。
