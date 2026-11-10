# 配備実行アダプタは、Binding を照合できない（2026-11-09）

- 対象: [`internal/plugin/runtime/adapters.go`](../../../internal/plugin/runtime/adapters.go)、[`routing.Router.Dispatch`](../../../internal/deployment/routing/executor.go)、[C-3 の参照](../../../tools/reference-executor-id-resolution/README.md)
- 位置づけ: **読み取り専用の調査**。何も変更しません。
- 計測した版: `raw/head.txt`（`origin/main` `e63d4c8f`）。

前のラウンドで、`Dispatch` が検証しない契約を固定しました。この記録は、**その検証に到達できない経路が残っている**ことを示します。

## 1. アダプタが持っている情報

```text
type DeploymentExecutor struct {
    Plan         EnvironmentPlan
    Registry     Registry
    CapabilityID core.CanonicalID
}

type EnvironmentPlan struct {
    Environment ..., PluginLockFingerprint ..., Plugins, Capabilities, ServiceBindings
}
```

**アダプタは `Environment` を持っていません。** `Plan` は環境の ID を持ちますが、**deployment-executor の Binding を持ちません**。

## 2. アダプタが確かめているのは「種別」だけ

```text
func (a DeploymentExecutor) Supports(c executor.Command) bool {
    cr, ok := a.Plan.Capability(plugin.CapabilityDeploymentExecutor, a.CapabilityID)
    ...
    return cr.Capability.SupportsAction(deployment.ActionKind(c.Kind))
}
```

**「この能力がこの action を支えるか」しか見ていません。** 「環境の Binding がこの能力を選んでいるか」は見ていません。

- `RouteCommand` の呼出しは**0件**（`routing` はコメントにしか出てきません）
- `ValidateResult` の呼出しも**0件**

## 3. 4つの構築箇所は、どれも経路を照合していない

```text
tools/reference-deployment-attempt/main.go:        RouteCommand=0 ValidateResult=0 Dispatch=0
tools/reference-plugin-build-supply/stage4.go:     RouteCommand=0 ValidateResult=0 Dispatch=0
tools/reference-plugin-self-build/main.go:         RouteCommand=0 ValidateResult=0 Dispatch=0
examples/process-plugin/main.go:                   RouteCommand=0 ValidateResult=0 Dispatch=0
```

**4箇所すべてが `Dispatch` を経由せず、`adapter.Execute` を直接呼んでいます。**

## 4. この観測が示すこと

- **C-3 が示した「直書きの `CapabilityID` で固定の能力を直叩きする」は、いまもそのままです。** アダプタは Binding を照合**できません**（情報を持っていない）。
- **`Dispatch` の検証を強化しても、この4経路には届きません。** 前ラウンドで固定した `Dispatch` の契約は、**`Dispatch` を通る経路にしか効きません**。
- **同じ `CapabilityID` を宣言した環境では差が出ませんが、Binding が別の executor を選んでいる環境では、アダプタは宣言を無視して直書きの能力を呼びます。**

## 5. この記録が示さないこと

- **アダプタに `Environment` を渡すべきか**。設計判断です（`Plan` に Binding を載せる案、呼出し側で `RouteCommand` を先に呼ぶ案など、複数の形がありえます）。
- **実害の有無**。4箇所は参照実装と example で、対象の環境では直書きの ID が Binding と一致している可能性があります（未確認）。
- **`ObservationProvider`／`EvidenceProvider` アダプタの同種の可否**。どちらも `Plan` と `CapabilityID` だけを持ち、構築0件です。
- 製品 CLI へ繋ぐべきか（C-1〜C-4 は未決）。

## 6. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（構造・検査内容・4箇所の照合状況） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域（0バイト＝クリーン） |
| `raw/anchors.txt` | 上の1〜3節の出力 |

## 7. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `322058427d2d68ba3d6b1f112a4673e309c22f7937dbd4fd2878954ab8ac1685` | 2396 |
| `raw/head.txt` | `1e885c408261ad22f01aa80a7cc175257dd18b8463e1519a59367fb3327e3dbd` | 73 |
| `raw/pre-status.txt` | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` | 0 |
| `raw/anchors.txt` | `0efe036fad1cfb507bf4eb0696b538f6b342a025baf401ed1ce0e5fc3f12ae83` | 1785 |
