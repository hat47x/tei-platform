# 標準CRUDの負例（`standard-crud`）

- 状態: **合成の負例**。製品機能ではなく、操作面の導出が返す「できない理由」を実際の宣言で固定するための作業領域です。
- 設計: [標準CRUDの導出](../../plan/design/DERIVED_CRUD_OPERATION_DESIGN_2026-11.md) 6章 W6、[操作面の導出](../../plan/design/OPERATION_SURFACE_DESIGN_2026-11.md) 6章

## 何を固定するか

正常に読める作業領域でも、限られた宣言からは導出できない操作があります。この負例は、その「できない理由」が毎回同じ形で出ることを確かめます。宣言を足せば解消する種類の不足と、製品側の宣言が未整備なために解消できない不足を混ぜています。

| 宣言 | 期待する結果 | 何を示すか |
| --- | --- | --- |
| `sales.note` に主キーも候補キーも無い | `read`・`create` が `missing-semantics` | 識別子が無いと一件参照も登録も導けない |
| `sales.order` は識別キーがある | `update` が `concurrency: expected-state` で導出される | 後勝ちを避ける方法を、宣言を足さずに既定で持つ（仮説H1） |
| `sales.order` に `effect update` の能力はあるが、ライフサイクルの宣言が無い | `retire` が `missing-input` | 状態遷移の意図はあるが、製品側にライフサイクルの宣言がまだ無い |
| `sales_archive` を `access write` で使う | `list` が `conflict`（`create`・`update` は導出する） | 書込み専用の経路から読めない |

## 宣言の構成

sales.order.cancel は applications/ledger/realizations/cancel-order.tei で実現済みです。束縛（binding）の解決と、実行へ進むための前提になります。

```text
reference/standard-crud/workspace/
├─ tei.workspace.yaml
├─ semantic/sales/order.tei          # Order・Note・effect update の能力
├─ applications/ledger/application.tei  # read-write の sales、write の archive
└─ databases/
   ├─ sales-core/                    # managed-here、正本側の保存
   │  ├─ database.tei
   │  └─ persistence/{order,note}.tei
   └─ sales-archive/                 # write 専用の別名が指す保存先
      ├─ database.tei
      └─ persistence/order-archive.tei
```

## 実行

```sh
go run ./cmd/tei validate -workspace reference/standard-crud/workspace
go test ./internal/operations -run StandardCRDFixture -count=1 -v
```

`validate` は `ok` を返します。**未解決は検証エラーではありません。** 意味としては読めるが、操作として導出できないことを表します。

## 期待する操作面

```text
operations
  ledger.list.sales.note
  ledger.list.sales.order
  ledger.read.sales.order
  ledger.create.sales.order
  ledger.update.sales.order
gaps
  sales.note/create/missing-semantics
  sales.note/read/missing-semantics
  sales.order/list/conflict
  sales.order/retire/missing-input
```

## 非目標と限界

- この負例は、実データベース接続・実行・画面を含みません。操作面の導出だけを対象にします。
- 未解決は「製品が未完成である」ことの証明ではありません。宣言を足せば解消するものと、製品側の宣言（同時更新・ライフサイクル）が決まっていないために解消できないものを区別して読んでください。
- 参照用作業領域（`reference/workspace`）は変更しません。この負例は独立した作業領域です。
