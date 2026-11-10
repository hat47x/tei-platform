# 他の3種の実装束縛は依然として参照グラフに現れない（2026-11-10、第43ラウンド）

- 対象: [#3983](https://github.com/hat47x/tei-platform/issues/3983)（残件として追記）、[#3989](https://github.com/hat47x/tei-platform/pull/3989)（`customGoBindings` の修正）。
- 前段: [第42ラウンド](../TEI_DOGFOODING_ROUND42_REALIZE_EDGE_2026-11-10/README.md)。
- 実行日: 2026-11-10（Asia/Tokyo）。
- 状態: **read-only の実測。残件を特定して記録。コードは変更しない。**

## 1. 目的

第42ラウンドで「`customGoBindings` を辺にする」修正を入れた。同じ層に他の束縛種が3つあるので、それらも辺になっているかを確かめる。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `c25664465`（`origin/main`） |
| CLI実行物 | `/tmp/tei44`（`c25664465` からビルド） |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 観測

### 3.1 World には4種の束縛が入っている

```bash
tei world -workspace reference/workspace
```

```text
customGoBindings                  : 4
headerDetailContractQueryBindings : 1
contractQueryExecutorBindings     : 2
contractQuerySQLRealizations      : 1
```

### 3.2 trace の関係には1種しか出ない

全 subject（scope `c`・`s`・`e`、`-direction both`）を走査した結果:

```text
type 120 / underlying 54 / owns-pattern 36 / input 36 / uses-semantic 30
relation 24 / outcome 24 / realizes 18 / uses-validation 18 / attribute 18
targets 18 / contract-query-input 12 / data 12 / query 6 / capability 6 / output 6
```

`realizes`（`customGoBindings` 由来）は18件。**他の3種に対応する関係は0件。**

### 3.3 宣言は実在する

```tei
// environments/development/contract-query-executors/customer-find-by-id.tei
contract-query-executor-binding {
    environment development
    instance master_data.dev
    database master_data
    contract master_data.customer
    query find_by_id
    executor database.query.oracle
}
```

```text
applications/order-entry/realizations/header-detail/order-lines.tei
databases/sales-core/query-realizations/order-line-find-by-order.tei
```

### 3.4 `trace` の subject-kind にも無い

```text
semantic, semantic-type, semantic-data, semantic-attribute, semantic-relation,
semantic-validation, semantic-capability, application, application-pattern, query
```

束縛に対応する kind が無い。**subject としても辺としても辿れない。**

## 4. 原因（実測）

```text
$ for k in HeaderDetailContractQueryBindings ContractQueryExecutorBindings \
           ContractQuerySQLRealizations CustomGoBindings; do
    echo "$k index.go:$(grep -c $k internal/index/index.go) \
                   projection.go:$(grep -c $k internal/modelref/derivedbridge/projection.go)"
  done

HeaderDetailContractQueryBindings  index.go:0 projection.go:0
ContractQueryExecutorBindings      index.go:0 projection.go:0
ContractQuerySQLRealizations       index.go:0 projection.go:0
CustomGoBindings                   index.go:4 projection.go:0
```

`ir.World` には4種すべての欄がある（`internal/ir/world.go:35-38`）が、**派生索引は `CustomGoBindings` しか読んでいない**。

## 5. 影響の実測

`contract-query-executor-binding` の `executor` を `database.query.oracle` から `database.query.postgres` に変えた。

| 欄 | 変化 |
| --- | --- |
| `worldFingerprint` | `c92de09f…` → `8e8c60a7…`（変化する） |
| **`graphFingerprint`** | **`48a12db2…` → `48a12db2…`（変化しない）** |
| `trace.total` | 14 → 14（変化しない） |

**実行基盤の選択が変わっても、参照グラフは何も言わない。** `#3983` の `customGoBindings` と同じ状態が、3種に残っている。

## 6. 起票の扱い

新規 Issue は作らず、[#3983](https://github.com/hat47x/tei-platform/issues/3983) に残件としてコメントした。同じ層・同じ原因であり、分割すると重複が増えるため。

## 7. 期待（案の材料）

| 束縛 | 自然な辺 | 設計判断が要る点 |
| --- | --- | --- |
| `headerDetailContractQueryBindings` | `application-pattern -header-detail-> database-contract-query` | 両端は既存ノードで足りる |
| `contractQueryExecutorBindings` | `database-contract-query -executed-by-> plugin` | `executor`（`database.query.oracle`）は plugin の capability 名。plugin ノードとの対応を決める必要がある |
| `contractQuerySQLRealizations` | `database-contract-query -sql-realized-by-> database` | `source` は SQL 文字列。辺の対象を何にするか（`database`＋`query` の組か） |

**3種とも、辺の両端を何にするかの設計判断が必要**なので、私は材料を提示するに留める。

## 8. 読み取れないこと（NOT RUN）

- 上記3種の辺の設計（両端の決定）。
- `contractQueryExecutorBindings` の `executor` が plugin のどの subject に対応するか（`provides contract-query-executor` との照合）。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 9. 位置付け

第42ラウンドの修正は `customGoBindings` の1種だけを扱った。**同じ層に同型の欠陥が3つ残っている**ことを実測で示した。

「モデルにあるが CLI から見えない」型は、1つ直しても層全体を確認しないと残る。今回は `ir.World` の束縛欄を全部数えることで、残りを網羅的に特定した。
