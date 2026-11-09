# C-2を試験の外で組み立て、C-1と同じ土台で比べる（2026-11-09）

- 対象: [判断材料: 実行の側をどこで宣言へ繋ぐか](../../workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)の **C-2**、[C-1とC-2は、同じ問いの2案か](../CONNECT_POINT_C1_C2_2026-11-09/README.md)、[宣言のBindingから実行までを、参照実装で繋いでみる](../BINDING_TO_EXECUTION_REFERENCE_2026-11-09/README.md)
- 位置づけ: **参照実装と実行記録**。製品へ繋ぐべきかは示しません。
- 計測した版: `raw/head.txt`（`origin/main` `cf1c473c` の作業ツリー）。

前の記録は、**C-2（`servicehost`）は非試験の呼出し元が0件**であることを示しました。この記録は、**C-1 と同じ形の参照として C-2 を組み立て、動かします。**

## 1. 何を作ったか

`tools/reference-service-carrier-connect`（新規・読み取り専用）。

Host が渡す入力から、既存の2つの関数を順に呼んで Invoker を組み立てます。

1. `servicehost.BindHostApplicationEnvironmentRoute`（Application/Environment を固定した RouteSource）
2. `composition.NewSelectedCoordinator`（Host の世代ガード）
3. `servicehost.NewSelectedServiceCarrierInvoker`（世代をピン留めした Invoker）

**参照は Runtime registry も Workspace も組み立てません。** Process・Verifier・ServiceAcceptance は付属 CLI が参照用の固定実装として渡し、**外部効果はありません**。

## 2. 観測

```text
== route-present ==
  ran: 1  refused: 0
    request=reference-request generation=1 pinned=True

== route-missing ==
  ran: 0  refused: 1
    request=reference-request generation=1 pinned=False
    error=ROUTE_UNAVAILABLE: the Host has no current route for this generation

== inputs-missing ==
  ran: 0  refused: 1
    request=reference-request generation=0 pinned=False
    error=CONNECT_INPUT_INVALID: a current route source, a registry and a guard ...
```

- **Route があれば、ピン留めした世代（`generation=1`）で要求が運ばれます**（`pinned=True`）。
- **Route が無い場合は、Invoker を呼ぶ前に拒否**します（`pinned=False`、終了コードは0で JSON を返す）。
- **Host が何も渡さない場合も拒否**し、推測で組み立てません。

## 3. この観測が示すこと

- **C-2 は成立します。** 既存の3関数だけで、試験の外でも Invoker を組み立てられます。C-1 と同じ「既存の部品を順に呼ぶ」形です。
- **C-2 の要求は C-1 より多い**ことが、組み立ての段数として現れます（C-1 は2段、C-2 は3段＋Host入力5種）。
- **Route の不在は、Invoker を呼ぶ前に検出できます。** 前の記録が「C-2 は `Coordinator.Acquire` で世代を固定する」と示した通り、**固定の単位が Route と結びついている**ためです。
- **両方を選ぶことは経路の重複ではありません。** C-1 は配備の動作、C-2 は Service の要求で、対象が違います（前の記録の結論と一致）。

## 4. この記録が示さないこと

- **製品 CLI へ繋ぐべきか**。判断材料の C-1〜C-4 は未決です。
- **実配備・実 Service の成立**。Process は記録用、Verifier は digest 一致だけを返し、Acceptance は固定の参照を返します。
- **C-1 と C-2 のどちらが優れているか**。段数と入力の数の違いは示しましたが、優劣は示していません。
- **並行実行**。1要求ずつ順に実行します。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（3つの場合を実行し、要約する） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域（新規ツールと`aw/`のみ） |
| `raw/route-present.json` | Route がある場合 |
| `raw/route-missing.json` | Route が無い場合 |
| `raw/inputs-missing.json` | Host が何も渡さない場合 |
| `raw/route-missing.exit.txt` | 2番目の終了コード |
| `raw/summary.txt` | 上の表を作る元の一覧 |

## 6. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `f47fd2a046eb7e3229c17dd8d431ef99f3839a805cf780be99c2b7281d279df8` | 2574 |
| `raw/head.txt` | `8d1976fc6cd08b2f92ddc231d037f90d8de77d4fe6d6be0468542e507ca3fe77` | 73 |
| `raw/pre-status.txt` | `2e0ed4e644e71883ef612082c6822885fcfa0a0bb558bfa4e61fdddf3bdf0a11` | 372 |
| `raw/route-present.json` | `239d02e46107dd81efa26aab17b571cefe0d726ca68919b47d637fe6169a985f` | 887 |
| `raw/route-missing.json` | `9ced14b304ffd2a47a67e03ae227d604da03b16247d0228d0a30831654c4f5ae` | 930 |
| `raw/inputs-missing.json` | `c9c78c1fd6a6b4ef703343e8cf6139fb6100b83e4b92dfa1ea58de0d2c3a98dd` | 922 |
| `raw/route-missing.exit.txt` | `0663d324ed53f0c1e846590d8d4969bf546d7d9ed8782e578bbb9e43f6fda2da` | 21 |
| `raw/summary.txt` | `7a2da5e928f736b10b35884b2919f55cf2c76107dcd2702bf1d4fa8d53d761db` | 535 |

規則の指紋は `sha256:b0b27eb27b498aece0fcfdb39df3228422c47643dd242ff405107fb51b9533f9` です。

`raw/inputs-missing.json` は `generation=0` を記録します。これは「Binding が渡されていない」ことを表す値で、実在の世代番号ではありません。
