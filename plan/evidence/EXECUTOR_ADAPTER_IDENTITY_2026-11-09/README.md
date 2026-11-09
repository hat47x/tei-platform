# 配備実行アダプタも、executor の同一性を上書きしていた（2026-11-09）

- 対象: [`internal/plugin/runtime/adapters.go`](../../../internal/plugin/runtime/adapters.go)、[`routing.ValidateResult`](../../../internal/deployment/routing/executor.go)、[観測アダプタの同種の不具合](../ADAPTER_IDENTITY_MASKING_2026-11-09/README.md)
- 位置づけ: **製品コードの不具合修正と実行記録**。
- 計測した版: `raw/head.txt`（`origin/main` `47667667` の作業ツリー＋このPRの変更）。

前の記録で観測アダプタの同一性上書きを直しました。**同じ形が配備実行アダプタにもありました。**

## 1. 何が問題だったか

```go
// 修正前
out.ExecutorID = string(a.CapabilityID)   // 無条件に上書き
```

一方 `routing.ValidateResult` はこのフィールドを比較します。

```go
if result.ExecutorID != string(routed.ExecutorID) {
    return fmt.Errorf("executor result for action %s came from %s, environment binds %s", ...)
}
```

**アダプタが無条件に上書きするため、executor が別の identity を名乗っても、`ValidateResult` には必ず一致した値が渡ります。** つまり **C-1 が呼ぶ検査が到達不能**でした。

## 2. 何を変えたか

**空のときだけ補う**ようにしました（観測側と同じ形）。

```go
if out.ExecutorID == "" {
    out.ExecutorID = string(a.CapabilityID)
}
```

## 3. 観測

### 修正後の試験

```text
--- PASS: TestAnEmptyExecutorIdentityIsFilled
--- PASS: TestAForeignExecutorIdentityIsNotOverwritten
--- PASS: TestAForeignExecutorIdentityIsRejectedByValidateResult
```

3つ目が要点です。**executor が `someone.else` を名乗った結果を `ValidateResult` に渡すと、正しく拒否されます。**

### 修正前の挙動を再現した結果

```text
reverted adapters.go to the old unconditional overwrite
--- FAIL: TestAForeignExecutorIdentityIsNotOverwritten (0.00s)
    a foreign executor id was overwritten with "runtime.main"
--- FAIL: TestAForeignExecutorIdentityIsRejectedByValidateResult (0.00s)
    a result from another executor was accepted
FAIL
restored adapters.go
```

**修正前は、別 executor の結果が `ValidateResult` を通っていました。** 試験がこの不具合を捕まえることを確認しました。

## 4. この観測が示すこと

- **配備実行アダプタにも、検査を無効化する不具合がありました。** 観測側と同じ形で、**上書きが検査を到達不能にしていました**。
- **C-1 の参照が `ValidateResult` を呼んでいても、アダプタ経由では意味がなかった**ことになります。C-1 の記録は `ValidateResult` の拒否を「不正な応答」として数えていましたが、**アダプタが identity を偽装していたため、その拒否は起きませんでした**。
- **アダプタは参照実装と example からのみ構築されます**（非試験で4件）。実害が出ていたかは、それらの経路が実際に動いたかに依存します。

## 5. この記録が示さないこと

- **C-1 の過去の実行記録が無効であること**。C-1 の `ValidateResult` 拒否試験は**独自のテスト用 executor を渡しており**、このアダプタを経由していません。無効になるのは「アダプタ経由の経路」だけです。
- **実配備での影響**。4つの構築箇所が実際にこの検査に依存していたかは調べていません。
- **`DeploymentExecutor` 以外の同種の不具合**。3つのアダプタ（配備実行・観測・証拠）を確認し、配備実行と観測に同じ形がありました。証拠側は元から正しい形でした。

## 6. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（修正後の試験と、修正前の再現） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域 |
| `raw/tests.txt`・`raw/tests-summary.txt` | 修正後の試験出力と要約 |
| `raw/old-behaviour.txt` | 修正前に戻したときの失敗 |

## 7. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `34f28377b9e6c0bc04aced31a37cd3e4aa8863820b26a3f9745e6a591449ff47` | 2185 |
| `raw/head.txt` | `4c91574572a38bb38e711ce8f34ed683e4274c4e975e52607bb49ed5c0e0ac91` | 73 |
| `raw/pre-status.txt` | `05ccb6723796918a1675b8859a6269f8b649f38b226e38f15735c085f5118141` | 163 |
| `raw/tests.txt` | `40e5581bd7095914dcf9eba24046a36a9f6a0ce883505a7172aeb767fad21b9a` | 401 |
| `raw/tests-summary.txt` | `b3f50ffe8aeb117f5baccfcf64716055ffc7c1eb56a51fe9f363ecd4a2eab2ba` | 396 |
| `raw/old-behaviour.txt` | `011d3b8ada2c8e73536fcf4e96eec67cd4130cf624d39d70677b4e63f5a6018b` | 453 |
