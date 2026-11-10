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

## 3. 案3: artifact の digest を比べる —— payload と envelope を区別する

`tei artifacts` が作る operation-surface の artifact は digest を持ちますが、**payload に World の識別子が入っていません**。

```text
== candidate 2: the artifact payload (what gets digested) ==
  SchemaVersion, Application, Rule, Operations, Gaps, Summary
== world-identity fields in the payload: 0
```

ただし、**payloadに識別子がないことからdigestにも入力の版が含まれないとは言えません**。`Artifact.fingerprint`はKind、**InputRevision**、Rule、Generator、Content、Unresolvedを含めます。`RenderWorkspace`は`sourceindex.ModelRevision(index)`を`RenderOperationSurface`へ渡します。同じContentでもInputRevisionが変わればdigestは変わります。初回調査は`Digest = fingerprint(artifact)`という呼出しだけを見て、その中身と呼出元を確認していませんでした。

したがってdigestは入力revisionも束縛するenvelopeの指紋です。ただしRule等の変更でも変わるため、意味のdriftだけを表す専用値ではありません。また、保存された`operation-surface.json`はContentだけであり、そのファイル単体のSHA-256と`Artifact.Digest`は異なります。ModelRevisionの網羅性と比較方針は別途検証が必要です。

## 4. 比較の材料は、どの案でも「既にある」とは言えない

| 案 | いま運べるか |
| --- | --- |
| 再導出して比べる | **運べます**（`WorldFingerprint`・`ModelRevision` が surface にある） |
| 保存済みを読み戻す | **運べません**（読み戻し0件・フラグ0件。両方の新設が要る） |
| artifact の digest | **入力revisionを含むenvelopeの指紋を運べます**。Content単体のhashとは異なり、意味以外の変更でも変わります |

## 5. この記録が示すこと

- **D-4の候補値が置かれる場所を確認しました。** 再導出側に既存の識別子はありますが、比較する以前の期待値とその供給経路は別途必要です。保存済み操作面の比較は現CLIに未実装です。読戻しと期待値の指定を別々のフラグにする必要があるとは限りません。artifactのdigestも入力revisionを含みますが、意味だけのdriftと同一視できません。
- **D-4 の既定（有効／明示指定）と独立に、経路そのものが未決です。** 有効にしても、**どの値を渡すかが決まらなければ照合は動きません。**

## 6. この記録が示さないこと

- **どれを採るべきか。** 未決のままです。
- **`WorldFingerprint` が drift を漏らさないこと。** 属性の追加で変わることは別の記録で確認していますが、**すべての意味変更を検出することは示していません**。
- **`ModelRevision` の定義**と、それが何を保証するか。
- 実配備、実 Provider。

## 7. 生データ

`raw/anchors.txt`と以下のhashは初回調査の記録として維持します。これはコード断片の静的な抽出であり、比較経路の実行試験ではありません。上記§3・§5は2026-11-11のレビューで、`internal/artifact/artifact.go`の`fingerprint`と`workspace.go`のrevision受渡しを確認して訂正しました。

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
