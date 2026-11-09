# 自律区分は何から導かれているか（2026-11-07）

- 対象: [操作面の設計候補](../../design/OPERATION_SURFACE_DESIGN_2026-11.md)の8章（自律区分）と、15章 未決3（`unattended` に環境のサービス主体の宣言を必須とするか）
- 位置づけ: **実行して観測した記録**。区分の採否・Owner受理・Gate合否ではない。
- 計測した版: `raw/head.txt`（`origin/main` `8a5cc408` の作業ツリー）。

設計8章は `unattended` を「読み取り専用である。**または**、実行主体が環境に宣言され（サービス主体）、権限が呼出しごとの人の判断に依存せず、冪等の識別子が宣言され、人の判断を要する段階を含まない」と定義し、15章の未決3で「環境のサービス主体の宣言を必須とするか」を保留しています。

この記録は、**現行の導出がその条件をどこまで読んでいるか**を観測します。

## 1. 観測

### 1.1 操作面が出す区分

| 対象 | `unattended` | `attended` | `human-gated` |
| --- | --- | --- | --- |
| `standard-crud`（`ledger`、Persistence あり） | 3（`list`×2・`read`×1） | 2（`create`・`update`） | **0** |
| `reference`（`order_entry`） | 4（`list`×2・`read`×2） | 4（`create`×2・`update`×2） | **0** |

`unattended` が付くのは**読み取り操作だけ**で、`create`・`update` は `attended` です。`human-gated` はどちらのWorkspaceでも**1件も出ていません**。

### 1.2 宣言の置き場所の探索

非試験のGoコードで、`ServiceSubject`・`service subject` は**1件も見つかりません**。`human-gated` は [`internal/artifact/consistency.go`](../../../internal/artifact/consistency.go) の許容値一覧にだけ現れます。

`Environment`の型（[`internal/environment/model.go`](../../../internal/environment/model.go)）が持つのはプラグイン・Bundle・Feature・Database instance・Binding・Executor・Observation/Evidence Provider・権限付与・Resource binding・Lease policyで、**実行主体（サービス主体）を宣言する欄はありません**。

### 1.3 `human-gated` を出せる経路

`internal/operations/surface.go` が `Autonomy` に代入するのは `"unattended"`（297・314行）と `"attended"`（344・363行）だけで、`"human-gated"` を代入する箇所はありません。`knownAutonomy`（`artifact/consistency.go`）は3値を**許容**しますが、生成側は2値しか出しません。

## 2. この観測が示すこと

- 現行の導出は、**「読み取り専用かどうか」だけ**で `unattended` を決めています。設計8章の後半（サービス主体の宣言・冪等の識別子・人の判断段階の不在）は**読んでいません**。読む対象の宣言がモデルに存在しないためです。
- 設計8章の「判断段階が未解決なら `human-gated` に倒す（安全側）」は、**現行では実装されていません**。`human-gated` を出す経路が無く、`run-plan` の宣言でも `attended`／`unattended` の2値しか受理しません（`internal/validation/runplan.go`）。
- したがって未決3は、「必須にするかどうか」ではなく、**必須にするなら宣言の置き場所と生成規則を新設する**判断になります。現行では「読み取り専用なら無条件に `unattended`」という規則が、サービス主体の宣言を待たずに成立しています。

## 3. この記録が示さないこと

- **どちらの案を採るべきか**。15章の未決3はOwner判断です。
- **`unattended` の現在の判定が誤りであること**。読み取り専用に効果が無いという事実から `unattended` を導く規則は、設計8章の前半と一致します。
- **`human-gated` が不要であること**。区分として定義され、`knownAutonomy` が許容している以上、生成されないことは「不要」ではなく「未実装」です。
- **実DB・実Provider・権限**。合成のWorkspaceと読み取りのみです。

## 4. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（2つのWorkspaceで操作面を出し、宣言の置き場所を走査する） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域（0行） |
| `raw/standard-crud-operations.json`・`raw/reference-operations.json` | 操作面の出力 |
| `raw/declaration-scan.txt` | 自律区分に関する宣言の走査結果と、fixtureの environments 有無 |
| `raw/summary.txt` | 上の表を作る元の一覧 |

`*.err.txt`はすべて0バイトです。

## 5. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `7abb3c5a6e1d74faac1b02a04671ef2ceb85e1e60b090741bfa92ad04538515c` | 3368 |
| `raw/head.txt` | `a4c9bb80365db58c231f1de5c5678e0e880b11fe5f9d416198b145ea116b8112` | 73 |
| `raw/pre-status.txt` | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`（空） | 0 |
| `raw/standard-crud-operations.json` | `e6a8443c13d8715f2015de58d0abf25691065d8d1c8a42a0474723692622d171` | 3268 |
| `raw/reference-operations.json` | `aae8986f434b97758f4adda183ce3286e24164b879f759fc92dbffed5a832232` | 4208 |
| `raw/declaration-scan.txt` | `58b6f5feb55b7afc0c7d7bba413da57e6e27d775c50b7d2b2947f20805024479` | 2151 |
| `raw/summary.txt` | `baf0da78e1a61fc39cfa206b7d416a6e05b4001e7e393f5ac2201712fe6333b5` | 1226 |

`raw/standard-crud-operations.json`と`raw/reference-operations.json`のSHA-256は、それぞれ[構成Kernel移行Gateの実行記録](../COMPOSITION_FIRST_CONSUMER_RUN_2026-11-07/README.md)の`raw/operations.json`、[操作とPatternの突き合わせ](../OPERATION_PATTERN_CROSSWALK_REFERENCE_2026-11-07/README.md)の`raw/accepted-crosswalk.json`系列のものと同じ値です。別の版でも同じWorkspaceから同じ出力が出たことの傍証で、同一性の証明ではありません。

`raw/declaration-scan.txt`は**grepによる文字列走査**で、Goの型解決をしていません。`ServiceSubject`という名前が無いことは、同義の宣言が別名で存在しないことの証明にはなりません。
