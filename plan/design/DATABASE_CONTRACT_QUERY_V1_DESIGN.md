# Database Contract Query v1 設計

- 状態: 実装済みの現行基線
- 更新日: 2026-09-06
- 対象Issue: [ISSUE-000018](../issue/000018-database-contract-query-execution.md) / [ISSUE-000019](../issue/000019-reference-contract-query-invocation.md) / [ISSUE-000020](../issue/000020-contract-query-runtime-instance-identity.md) / [ISSUE-000021](../issue/000021-oracle-contract-query-sql-runner.md)
- 関連ADR: [ADR-0007](../adr/0007-database-driver-assembly.md) / [ADR-0021](../adr/0021-header-detail-relation-traversal-realization.md)

## この文書の役割

Databaseが外部へ提供するContract Queryについて、意味契約、native SQLによる実現、Environment上のexecutor選択、runtime実行までの層を一つの文書としてまとめます。

正本との関係は次のとおりです。

- Database lifecycle全体とProvider境界の説明は[`docs/architecture.md`](../../docs/architecture.md)が現行基線です。この文書はContract Queryの内部設計だけを扱います。
- 実現方式をProvider側へ置く判断は[ADR-0007](../adr/0007-database-driver-assembly.md)、header-detail traversalの実現境界は[ADR-0021](../adr/0021-header-detail-relation-traversal-realization.md)が正本です。
- 作業範囲と完了条件は各Issue、実装とEvidenceの現在地は[`docs/status.md`](../../docs/status.md)が正本です。

Application側からの利用（reference / header-detail Patternの実行）は[Application Driver v1 設計](APPLICATION_DRIVER_V1_DESIGN.md)と重複しない範囲で触れます。物理schemaのDesired / Actual / Diffは扱いません。

## 現在の結論

Contract Queryは四つの層に分かれ、各層が別のidentityを持ちます。上位層は下位層の実現方式を知らず、下位層は上位層の業務意味を知りません。

```text
① 意味契約         Database Contract Query signature
     database.QueryContract { Name, Inputs, Result, ResultCardinality }
                ↓ 同じ target identity で対応付ける
② 実現            Contract Query SQL Realization（provider固有）
     realization.ContractQuerySQLRealization { Target, Provider, Source, Inputs, ResultColumns }
                ↓ Environment / instance ごとに選択する
③ 実行者の選択     Contract Query Executor Binding
     realization.ContractQueryExecutorBinding { Environment, Instance, Target, Executor }
                ↓ 注入された接続で実行する
④ runtime         sql.Executor + provider Runner（PostgreSQL / Oracle）
```

層をまたぐ共通identityは`ContractQueryTarget`（Database / Contract / Query）です。file pathやprovider名を含めないため、realization sourceの移動や実装方式の変更でidentityが変わりません。

## ① 意味契約

[`internal/database/model.go`](../../internal/database/model.go)の`QueryContract`は、「何を受け、どのSemantic Dataをどの件数契約で返すか」だけを保持します。SQLやAPIの実現方式を含めません。

```tei
contract sales_core.order_line v1.0 {
    read sales.order_line
    query find_by_order {
        input order_id type sales.order_id
        result sales.order_line many
    }
}
```

`ResultCardinality`は`one` / `zero-or-one` / `many`です。これは正常に完了したときのinstance数の契約であり、Execution Errorとは別の意味です。0件は障害ではなく、契約に照らして判定します。

signature未定義の既存Contractは`omitempty`により従来のCanonical JSONを維持します。

## ② native SQLによる実現

[`contract_query_sql.go`](../../internal/realization/contract_query_sql.go)の`ContractQuerySQLRealization`は、provider固有のSQL本文、物理parameter名、raw result列名を保持します。identityは`<database>/<contract>/<query>/sql/<provider>`です。

```tei
contract-query-sql-realization {
    database sales_core
    contract sales_core.order_line
    query find_by_order
    sql postgres file "order-line-find-by-order.sql"
    input order_id -> order_id
    result {
        column order_line_id -> sales.order_line.order_line_id
        column line_number  -> sales.order_line.line_number
        ...
    }
}
```

`input order_id -> order_id`のように名前が同じ場合でも対応を省略しません。名前の一致から暗黙に接続することを防ぐためです。

`ValidateContract`は、realizationとsignatureの対応が明示情報だけで一意に決まることを検査します。SQL本文を解析してparameterやresult列を推測しません。native SQLはauthoritative sourceのまま保持し、Canonicalの第二正本にしません。

## ③ Environment上のexecutor選択

[`contract_query_binding.go`](../../internal/realization/contract_query_binding.go)の`ContractQueryExecutorBinding`は、Environment、Database instance、Contract Query targetを一つのexecutor Capabilityへ結び付けます。

```tei
contract-query-executor-binding {
    environment development
    instance sales_core.dev
    database sales_core
    contract sales_core.order_line
    query find_by_order
    executor database.query.postgres
}
```

`ResolveContractQueryExecutor`は完全一致するBindingを一つだけ選びます。次を明示的に拒否します。

- Bindingが存在しない、または複数存在する
- Database definitionとtarget Databaseが一致しない
- executor CapabilityのkindやproviderがDatabase instanceと一致しない

providerが一致するCapabilityを探索して自動選択することはありません。identityは`<environment>/<instance>/<target>`とし、executor自体は選択結果なのでidentityへ含めません。同じ対象へ複数executorを宣言した場合を競合として検出できるようにするためです。

## ④ runtime

[`queryruntime/sql`](../../internal/driver/queryruntime/sql)の`Executor`は、targetからprovider向けSQL realizationを解決し、Contract inputを物理parameterへ写して`Runner`へ渡します。`Runner`は`database/sql`接続で実行するだけの最小境界です。

| Provider | Runner | 実行境界 |
| --- | --- | --- |
| PostgreSQL | [`queryruntime/postgres`](../../internal/driver/queryruntime/postgres) | `:name`形式のphysical parameterを`$1, $2, ...`へ明示束縛。0 rowは正常結果 |
| Oracle | [`queryruntime/oracle`](../../internal/driver/queryruntime/oracle) | raw result列に加えてdriverが観測したcolumn metadataを保持 |

Runnerは、Semantic attribute、Contract input、table / column、接続先Databaseのいずれも推測しません。実行時parameter値はEvidenceへ保持しません。

ADR-0007のとおり、実driverの組み立てはCore外の責務です。`go.mod`が持つDatabase driverはPostgreSQL（`github.com/lib/pq`）だけで、実接続は[`tools/`](../../tools)配下のEvidence runnerが`sql.Open`で組み立てます。Oracle driverの選定と統合は[ISSUE-000033](../issue/000033-gate-b-oracle-driver-integration.md)で継続しています。

## Application側からの呼び出し

Application PatternはContract Queryのidentityだけを参照し、SQLやprovider名を持ちません。

| 呼び出し元 | Bindingの所在 | inputの向き |
| --- | --- | --- |
| `reference` Pattern | Pattern本文の`contract-query` | Relation **target側** attribute → Query input |
| `header-detail` Pattern | Application配下のrealization binding | source instanceの **source側** attribute → Query input |

header-detailでBindingをPattern外へ置くのはADR-0021の判断です。Relation target側に親identity attributeが存在することを前提にせず、Pattern自体はRelation-onlyのApplication intentとして維持します。

Driver側の実行と結果検査（cardinality、Semantic row）は[Application Driver v1 設計](APPLICATION_DRIVER_V1_DESIGN.md)を参照します。

## 各層が推測しないこと

- Query名や物理parameter名から入力の意味を推測しない
- SQL本文を解析してinput / result列を導出しない
- provider一致だけでexecutorを選ばない
- 同じCapability IDを共有する複数instanceを一つのruntime接続へ縮退させない
- result Dataに属さないattribute（provider側のsupport column等）をSemantic結果として受け取らない
- 0件をExecution Errorへ、Execution Errorを0件へ変換しない

## Validationと診断

`ValidateContract`に加えて、[`internal/validation`](../../internal/validation)がWorkspace loadで次を検査します。

- Contract Query signatureとSQL realizationの対応（[`contract_query_sql_realization.go`](../../internal/validation/contract_query_sql_realization.go)）
- signature自体の整合（[`database_contract_query_signature.go`](../../internal/validation/database_contract_query_signature.go)）
- reference Patternからのinvocation整合（[`reference_contract_query_invocation.go`](../../internal/validation/reference_contract_query_invocation.go)、[`reference_contract_query_cross.go`](../../internal/validation/reference_contract_query_cross.go)）
- header-detail Bindingの整合（[`header_detail_contract_query_binding.go`](../../internal/validation/header_detail_contract_query_binding.go)）

診断はSource Indexを通してauthoring sourceへ戻れます。

## Evidenceの種類と確認先

取得状況そのものはこの設計文書へ複製せず、対応Issueと[`docs/status.md`](../../docs/status.md)を確認します。

| Evidence | 何を示すか | current stateの確認先 |
| --- | --- | --- |
| Repository内Conformance（Go test） | 契約・realization・executor選択・runtime変換の決定論的な再現 | Repository内test / `docs/status.md` |
| real PostgreSQL ephemeral fixture | 実engineでのheader-detail実行と前後の安全性 | [ISSUE-000031](../issue/000031-gate-b-postgres-header-detail-evidence.md) / `docs/status.md` |
| actual Oracle | 既存外部資産に対するread、driver統合、read-only強制 | [ISSUE-000022](../issue/000022-external-master-data-physical-query-evidence.md) / [ISSUE-000033](../issue/000033-gate-b-oracle-driver-integration.md) / [ISSUE-000034](../issue/000034-gate-b-oracle-read-only-evidence.md) / `docs/status.md` |

Contract Queryの機械検証は`reference_contract_query_sql_conformance_test.go`等のRepository testを正とします。説明文書のcurrent構成は`docs/conformance/`を確認します。

## 非主張範囲

- Contract Queryにwrite操作やtransaction境界は含みません
- 複数Contract Queryをまたぐ一貫性やretryは扱いません
- 実driverでのOracle動作はこの設計文書自体からは主張せず、actual Evidenceの現在地は対応Issueと`docs/status.md`を確認します
- Reference固有のSQLやcolumn名をCanonical schemaへ昇格させません
- PostgreSQLを深いReference Providerとして使っていることは、provider中立性の証明ではありません

## 今後の論点

- 同一providerに複数realization variantが必要になった場合のidentity拡張（現在はtarget + providerで一意）
- Contract Query用のConformance説明文書を`docs/conformance/`へ置くかどうか
- write / commandを持つContractの契約表現
- Oracle driver統合後のcolumn metadata観測とSemantic mappingの受け入れ境界

## 関連文書

- [Application Driver v1 設計](APPLICATION_DRIVER_V1_DESIGN.md)
- [Application Pattern v1 設計](APPLICATION_PATTERN_V1_DESIGN.md)
- [ADR-0007](../adr/0007-database-driver-assembly.md) / [ADR-0021](../adr/0021-header-detail-relation-traversal-realization.md)
- [ISSUE-000023](../issue/000023-header-detail-relation-traversal-boundary.md) / [ISSUE-000024](../issue/000024-keyless-relation-physical-storage.md)
- [`docs/architecture.md`](../../docs/architecture.md) / [`docs/status.md`](../../docs/status.md)
