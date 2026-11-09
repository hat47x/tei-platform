# 「Runtime へ直接入る3経路」は、いま何を指しているか（2026-11-09）

- 対象: [成立状況](../../../docs/status.md)の「Runtime へは `InvokeWithProvenance` を直接呼ぶ経路が3件あり、いずれも `composition.Coordinator` を使わない」、[その元の記録](../RUNTIME_UNPINNED_PATH_PURPOSES_2026-11-08/README.md)
- 位置づけ: **読み取り専用の調査**。何も変更しません。
- 計測した版: `raw/head.txt`（`origin/main` `a7f23e93`）。

成立状況は「3件」と書いていますが、**その3件が何を指すのかを現在のコードで数え直しました。**

## 1. `InvokeWithProvenance` の非試験呼出し元（5件）

```text
internal/plugin/runtime/adapters.go:35          DeploymentExecutor アダプタ
internal/plugin/runtime/model.go:595            Registry.Invoke（公開APIの実装本体）
internal/plugin/runtime/service_carrier.go:193  InvokeSelectedServiceCarrier（ピン留め済みの wrapper）
tools/reference-order-rollout-host-continuity/host.go:337   参照実装
tools/reference-plugin-capability-contract/runtime_authority_bridge.go:171  参照実装
```

**内訳が「3件の直接経路」とは一致しません。** 内部の2件（`model.go` と `service_carrier.go`）は Runtime 自身の wrapper で、外部から入る経路ではありません。

## 2. `Registry.Invoke`（ピン留めなしの公開API）を呼ぶ非試験の箇所

```text
internal/plugin/runtime/adapters.go:77   ObservationProvider アダプタ
internal/plugin/runtime/adapters.go:104  EvidenceProvider アダプタ
```

**2件で、どちらも同じファイルのアダプタです。**

## 3. そのアダプタは、どこかで組み立てられているか

```text
== ObservationProvider adapter constructed anywhere? ==
  (zero constructions, including tests)

== EvidenceProvider adapter constructed anywhere? ==
  (zero constructions, including tests)
```

**試験を含めて0件です。** 宣言されているが、誰も組み立てていません。

## 4. 対照：`DeploymentExecutor` は4件で組み立てられている

```text
examples/process-plugin/main.go:53
tools/reference-deployment-attempt/main.go:131
tools/reference-plugin-build-supply/stage4.go:117
tools/reference-plugin-self-build/main.go:764
```

いずれも `CapabilityID` を直書きしています（C-3 の記録のとおり）。

## 5. ピン留め済みの wrapper は存在し、使われている

```text
internal/plugin/runtime/service_carrier.go:107  func (r Registry) InvokeSelectedServiceCarrier(
internal/servicehost/service_carrier.go:112     これを呼ぶ（C-2 の経路）
```

## 6. この観測が示すこと

- **「3件」という数は、いまのコードでは再現できません。** `InvokeWithProvenance` の非試験呼出し元は5件で、うち2件は Runtime 内部の wrapper、2件は参照実装、残る1件が `DeploymentExecutor` アダプタです。
- **ピン留めなしの公開API `Registry.Invoke` を呼ぶのは2件で、どちらも組み立てられていないアダプタです。** つまり「製品が Coordinator を通らずに Runtime へ入っている」経路は、**いまのところ0件**です。
- **ただし、これは「安全になった」という意味ではありません。** アダプタが未使用なだけで、**使えば Coordinator を通りません**。C-3 が指摘した「直書きの CapabilityID で固定の能力を直叩きする」形は、そのまま残っています。
- **`DeploymentExecutor` の4件は、いまも参照実装と example です。** 製品コードからの構築はありません。

## 7. この記録が示さないこと

- **成立状況の「3件」をどう直すべきか**。数え方の定義（何を「経路」と呼ぶか）が変わるため、この記録は事実の提示にとどめます。
- **未使用アダプタを削除すべきか**。設計判断です。
- **アダプタを使う経路が将来増えないこと。**
- 実 Provider の実行、実配備。

## 8. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（呼出し元・構築箇所・wrapper の列挙） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域（0バイト＝クリーン） |
| `raw/anchors.txt` | 上の1〜5節の出力 |

## 9. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `c5c3b01ff60abbc37a478192f98e50aaeee26e496814715e559225acdf00c0b5` | 2659 |
| `raw/head.txt` | `df5b8f7b25f8db08acb17836817273a6a7bca7ecbea25e8ec14cc5b0a7e453b9` | 73 |
| `raw/pre-status.txt` | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` | 0 |
| `raw/anchors.txt` | `a9ea5490350b684fec16beae8766ba31f3e05658c07af00c218cd05c12b34f75` | 2398 |
