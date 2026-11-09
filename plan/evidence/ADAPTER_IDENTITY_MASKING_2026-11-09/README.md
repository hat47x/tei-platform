# 観測アダプタが、provider の同一性を上書きしていた（2026-11-09）

- 対象: [`internal/plugin/runtime/adapters.go`](../../../internal/plugin/runtime/adapters.go)、[観測Routerの同一性検査](../../../internal/deployment/observation/provider.go)、[C-2 の参照](../../../tools/reference-service-carrier-connect/README.md)
- 位置づけ: **製品コードの不具合修正と実行記録**。
- 計測した版: `raw/head.txt`（`origin/main` `08094b8a` の作業ツリー＋このPRの変更）。

前の調査で、観測・証拠のアダプタが**どこからも組み立てられていない**ことを確認しました。そのコードを読むと、**観測側に不具合**がありました。

## 1. 何が問題だったか

```go
// 修正前
out.ProviderID = a.CapabilityID   // 無条件に上書き
```

一方、`observation.Router.Observe` は次の順で動きます。

```go
if f.ProviderID == "" { f.ProviderID = id }   // 空なら補う
...
if f.ProviderID != id || ... {                // そのうえで比較する
    return Fragment{}, fmt.Errorf("... mismatched fragment identity")
}
```

**アダプタが無条件に上書きするため、provider が別の identity を名乗っても、Router の比較には必ず一致した値が渡ります。** つまり **Router の同一性検査が到達不能**になっていました。

## 2. 何を変えたか

**空のときだけ補う**ようにしました（証拠側と同じ形）。

```go
func completeObservedFragment(out observation.Fragment, req observation.Request, providerID core.CanonicalID) observation.Fragment {
    if out.ProviderID == "" { out.ProviderID = providerID }
    if out.Environment == "" { out.Environment = req.Environment }
    if out.Kind == "" { out.Kind = req.Kind }
    return out
}
```

`Environment` と `Kind` も、以前は触っていませんでしたが、**空のときだけ**補うようにしました。証拠側も同じ形の関数に切り出しました（挙動は変えていません）。

## 3. 観測

### 修正後の試験

```text
--- PASS: TestAnEmptyObservedIdentityIsFilled
--- PASS: TestAForeignObservedIdentityIsNotOverwritten
--- PASS: TestAForeignObservedEnvironmentIsNotOverwritten
--- PASS: TestAForeignEvidenceActionIsNotOverwritten
--- PASS: TestAnEmptyEvidenceActionIsFilled
```

既存の `internal/plugin/runtime` のテスト（同一性検査を含む）もすべて通ります。

### 修正前の挙動を再現した結果

```text
reverted adapters.go to the old unconditional overwrite
--- FAIL: TestAForeignObservedIdentityIsNotOverwritten (0.00s)
    adapters_identity_test.go:47: a foreign provider id was overwritten with "db.observer"
FAIL
restored adapters.go
```

**修正前のコードでは、回帰試験が失敗します。** 試験がこの不具合を捕まえることを確認しました。

## 4. この観測が示すこと

- **製品コードに、Router の検査を無効化する不具合がありました。** アダプタを組み立てる経路が今は0件なので**実害は出ていません**が、組み立てれば「別の provider が答えても一致と判定される」状態でした。
- **証拠側は正しい形でした**（`ActionID == ""` のときだけ補う）。観測側だけが非対称でした。
- **CON-12b の意味が、この経路では成立していませんでした。** CON-12b は「選ばれていない Provider を実行しない」ですが、その手前の「provider が名乗る identity を偽装できない」が、アダプタ経由では破れていました。

## 5. この記録が示さないこと

- **アダプタを製品で使うべきか**。未使用のままです（設計判断）。
- **実 Provider の応答**。試験は合成の `Fragment` を渡すだけです。
- **他のアダプタ**（`DeploymentExecutor`）の同種の不具合。まだ調べていません。
- CON-12c/12d への影響。

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
| `run.sh` | `254f99346b80e0c5d39734a5e64c4d45481fd00e3857be86344ab890614b8450` | 2200 |
| `raw/head.txt` | `37639cbd0f652b2cdc4339611baaca707a1af5f2ae0ad59b88d6195e044828b6` | 73 |
| `raw/pre-status.txt` | `57208d99f1b22d1448f77166b3fec04f589b68fd0e212e963ce180f57eaa0cc1` | 160 |
| `raw/tests.txt` | `6b052218cca26b12b738596bcf5f77d6fc46a7b434991423356cb16d0a284805` | 4945 |
| `raw/tests-summary.txt` | `0a9e2fadd200fa772f1d67449b81ab36bee5a035eff4762f9ece0a77fc662d39` | 2986 |
| `raw/old-behaviour.txt` | `4a0f4828b764d9ddf8828db3dfc481950af3eee93e84578a1f6ad109f69d6da3` | 281 |
