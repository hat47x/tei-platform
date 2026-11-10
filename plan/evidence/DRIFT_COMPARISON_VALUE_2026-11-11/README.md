# 照合値はどこから渡せるか（2026-11-11）

- 対象: [`internal/operations/surface.go`](../../../internal/operations/surface.go)、[`internal/artifact/surface.go`](../../../internal/artifact/surface.go)、[`cmd/tei/cli_operations_artifacts.go`](../../../cmd/tei/cli_operations_artifacts.go)、[操作面の判断材料](../../workstreams/OPERATION_SURFACE_DECISION_BRIEF_2026-11.md)の D-4
- 位置づけ: **読み取り専用の調査**。何も変更しません。
- 計測した版: `raw/head.txt`（`origin/main` `4b82484e`）。

判断材料は D-4（drift 照合の既定）の「決めないと止まること」を **「照合値の受け渡し経路（フラグ／保存済み操作面／再導出）が決まらない」** と書いています。**その3案が、いまのコードで実際に何を運べるか**を確かめました。

## 1. 案1: 再導出して比べる —— 比較に使える識別子がある

`OperationSurface` は次の2つを持ちます。

```text
WorldFingerprint string   `json:"worldFingerprint"`
ModelRevision    string   `json:"modelRevision"`
```

**どちらも `SurfaceWorkspace` が埋めます**（`DeriveOperationSurface` は埋めません）。同じ意味から同じ指紋が出るので、**再導出して `WorldFingerprint` を比べる**ことはできます。

## 2. 案2: 保存済みの操作面を読み戻して比べる —— できない

`tei operations` は**書き出すだけで、読み戻しません**。

```text
== can the CLI read a saved surface back for comparison? ==
  os.ReadFile in the operations command: 0
== any drift-comparison flag on the operations command? ==
  such flags: 0
```

**読み戻す経路も、照合用のフラグもありません。** この案を採るなら、**両方を新設する必要があります。**

## 3. 案3: artifact の digest を比べる —— 意味の識別子は入っていない

`tei artifacts` が作る operation-surface の artifact は digest を持ちますが、**payload に World の識別子が入っていません**。

```text
== candidate 2: the artifact payload (what gets digested) ==
  SchemaVersion, Application, Rule, Operations, Gaps, Summary
== world-identity fields in the payload: 0
```

**digest は「操作の一覧と gap と summary」の指紋であって、意味の版の指紋ではありません。** 同じ操作一覧を出す別の意味の版でも、digest は同じになりえます。

## 4. 比較の材料は、どの案でも「既にある」とは言えない

| 案 | いま運べるか |
| --- | --- |
| 再導出して比べる | **運べます**（`WorldFingerprint`・`ModelRevision` が surface にある） |
| 保存済みを読み戻す | **運べません**（読み戻し0件・フラグ0件。両方の新設が要る） |
| artifact の digest | **運べますが意味の版を表しません**（payload に World 識別子が0） |

## 5. この記録が示すこと

- **D-4 の「照合値の受け渡し経路」について、材料が揃いました。** 再導出は**既存の2つの識別子**で足ります。保存済みの読み戻しは**新しい口が2つ**要ります。digest は**別のものを測っている**ので、意味の drift には使えません。
- **D-4 の既定（有効／明示指定）と独立に、経路そのものが未決です。** 有効にしても、**どの値を渡すかが決まらなければ照合は動きません。**

## 6. この記録が示さないこと

- **どれを採るべきか。** 未決のままです。
- **`WorldFingerprint` が drift を漏らさないこと。** 属性の追加で変わることは別の記録で確認していますが、**すべての意味変更を検出することは示していません**。
- **`ModelRevision` の定義**と、それが何を保証するか。
- 実配備、実 Provider。

## 7. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（3案の実測） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域（0バイト＝クリーン） |
| `raw/anchors.txt` | 上の1〜3節の出力 |

## 8. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `5fcad1c34ffe9ba2c4bb5e1e444c797b0d03bc7f28fb89de7537a564e2add606` | 2461 |
| `raw/head.txt` | `c4248c41fdd8c4e57aeb30e3e0d926936583dce8b9628456033992c855747a8e` | 73 |
| `raw/pre-status.txt` | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` | 0 |
| `raw/anchors.txt` | `4b98aae058afabb76726fc2ca69d3cd5c435d411bfc6fadaba057c78f48cbf3e` | 1805 |
