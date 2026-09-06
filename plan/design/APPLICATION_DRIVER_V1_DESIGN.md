# Application Driver v1 設計

- 状態: 実装済みの現行基線
- 更新日: 2026-09-06
- 対象Issue: [ISSUE-000006](../issue/000006-application-driver-vertical-slice.md)
- 関連ADR: [ADR-0019](../adr/0019-authentication-capability-boundary.md) / [ADR-0020](../adr/0020-capability-business-outcome-boundary.md) / [ADR-0021](../adr/0021-header-detail-relation-traversal-realization.md) / [ADR-0023](../adr/0023-command-observation-explicit-read-boundary.md)

## この文書の役割

Application Driverの機能設計です。Canonical Worldに置かれたApplicationの意味を、UIなしで実行・観測するために、Driverがどの層を持ち、各層が何を解決し、何を推測しないかを一つの文書としてまとめます。

正本との関係は次のとおりです。

- 実装構造と技術境界の説明は[`docs/architecture.md`](../../docs/architecture.md)が現行基線です。この文書はそれを置き換えず、Driver機能の内部設計を扱います。
- 採用した設計判断はADR、作業範囲と完了条件は[ISSUE-000006](../issue/000006-application-driver-vertical-slice.md)、実装・Evidenceの現在地は[`docs/status.md`](../../docs/status.md)が正本です。
- Gate B判定と必要Evidenceは[`plan/README.md`](../README.md)と[Gate B Readiness Audit](../GATE_B_READINESS_AUDIT.md)が正本です。この文書はGate条件を新設しません。

扱うのは、Pattern解決からCapability実行、Query / Contract Query実行、Observationまでの実行境界です。Database側の物理schema lifecycle、Plugin / Process Runtimeのauthority、Target / Presentationは扱いません。

## 現在の結論

DriverはUI-lessで次の意味境界を一続きに再現できます。実行方式の選択はすべて明示された宣言から解決し、名前の一致や型の一致から補完しません。

```text
Canonical World (ir.World)
        ↓ 意味の解決だけを行う
Resolver
  ├─ Pattern解決          browse / record / reference / header-detail / command
  ├─ Capability Binding   Application + Capability
  └─ Contract Query解決   Database / Contract / Query
        ↓ 実行方式は注入された境界へ渡す
Executor境界
  ├─ BindingAdapterResolver → capabilityinvoke.Adapter
  ├─ BrowseQueryExecutor
  └─ ContractQueryExecutor（Environment composition経由）
        ↓
Business Result / Raw + Mapped result / Execution Error
        ↓ 別操作
Observation read
```

[`internal/driver`](../../internal/driver)直下の実行境界は、Database接続、SQL方言、Plugin実装、Target表現を知りません。provider固有の実行は`internal/driver/queryruntime`配下のsubpackageが担当し、実接続はさらに外側から注入します。

## Pattern解決

[`resolver.go`](../../internal/driver/resolver.go)の`ResolvePattern`は、Pattern Kindごとに必要な既存資産だけを解決します。`PatternResolution`のnil fieldは「未解決」ではなく、そのKindがその情報を要求しないことを示します。

| Kind | 解決するもの | Datasourceの正本 | 要求しないもの |
| --- | --- | --- | --- |
| `browse` | Data、Query | Queryの`datasource` | Persistence Mapping |
| `record` | Data | なし | Datasource、Query |
| `reference` | Relation（source / target Data付き）、Datasource | Patternに明示されたalias | Relation targetの物理位置 |
| `header-detail` | Relation、Realization Binding | Realization Bindingのalias | Relation keyの存在 |
| `command` | Capability | なし | 実装のfile pathやsymbol |

`browse`ではPatternとQueryのDatasourceが両方ある場合に一致を検査し、QueryのApplication所属も検査します。`reference`ではDatasourceの欠落を補完せずerrorにします。Relationのsource Dataは、ID文字列の規則ではなくSemantic定義上の所属関係から解決します。

## Capability Bindingの解決

[`binding.go`](../../internal/driver/binding.go)は、Application + Capability identityから[`realization.ResolveCustomGoBinding`](../../internal/realization/binding.go)で明示Bindingを一意に解決します。Bindingが0件でも複数件でも、候補を推測せず「実行準備を完了できない」errorとして返します。

`ResolveCommandBinding`はcommand Patternを起点に Pattern → Capability → Application-scoped Binding の順で解決します。Patternからfile pathやGo symbolを導出して実装を発見することはしません。

Bindingのidentityは`application/capability`です。新しいID体系を増やさず、Validationが重複検出に使ってきた組をそのまま安定identityにしています。

## Capability実行とBusiness Result

[`invocation.go`](../../internal/driver/invocation.go)の`InvokeCapability`は、解決済みBindingに対応するAdapterを`BindingAdapterResolver`から受け取り、[`capabilityinvoke.Invoke`](../../internal/semantic/capabilityinvoke)を通して実行します。

ADR-0020の境界をそのまま実行時に持ち込みます。

- 業務上の拒否や名前付きOutcomeは`BusinessResult`として返す
- Binding解決失敗、Adapter解決失敗、Adapter障害、Canonical契約違反はExecution Error
- `CapabilityInvocation`は実行時入力を保持しない。credential等をScenario Evidenceへ残さないため、Binding解決結果とBusiness Resultだけを返す

Authentication / AuthorizationもADR-0019に従い同じCapability経路で実行します。Driver固有の認可規則は持ちません。

## Query実行

[`query_execution.go`](../../internal/driver/query_execution.go)の`ExecuteBrowseQuery`は、解決済みQueryとlogical Datasourceをそのまま`BrowseQueryExecutor`へ渡します。

`BrowseQueryResult`は、Executorが観測した`Raw`と、Canonical Queryの明示mappingを適用した`Mapped`を分けて保持します。mappingは[`queryresult.Map`](../../internal/driver/queryresult/mapping.go)がauthoringされたresult column対応だけを適用し、列名の類似やPersistence MappingからSemantic attributeを推測しません。

Executorの障害やmapping不整合を正常な空結果へ変換しません。`Payload`にはExecutor固有の非secret Evidenceを置けますが、実行時parameter値は保持しません。

## Contract Query実行

[`contract_query_execution.go`](../../internal/driver/contract_query_execution.go)の`ContractQueryExecutor`は、Database-owned Contract Queryを実現方式へ接続する境界です。Driver CoreはSQL、API、stored procedureのいずれも知りません。

正常結果に対して二つの検査を行います。

1. **result cardinality** — `one`は1件、`zero-or-one`は0または1件、`many`は制約なし。0件をExecution Errorへ変換せず、契約違反は正常結果として受け取らない
2. **Semantic row** — rowのattributeがQuery result Dataに属さない場合は受け取らない。provider側のsupport columnや物理列がSemantic結果へ混入することを防ぐ

Executorのerrorは正常な0件へ変換せず、Execution Errorとして返します。詳細な層構造は[Database Contract Query v1 設計](DATABASE_CONTRACT_QUERY_V1_DESIGN.md)を参照します。

## Environment compositionとruntime instance identity

[`reference_contract_query_environment.go`](../../internal/driver/reference_contract_query_environment.go)は、Application の logical datasource を Environment 上の Database instance へ明示Bindingし、その instance と Contract Query target に対する executor Binding を解決してから実行します。

runtime registryのkeyは`{Environment, Instance, Capability}`です。Capability IDだけへ縮退させないのは、同じexecutor Capabilityを複数instanceで使う構成で接続先を取り違えないためです。

```text
Application datasource alias
      ↓ environment.ApplicationDatabaseBinding
Database instance
      ↓ realization.ContractQueryExecutorBinding
executor Capability
      ↓ ContractQueryExecutorRegistry {Environment, Instance, Capability}
実行可能なContractQueryExecutor
```

Environment、datasource、executor、runtime接続のいずれも暗黙選択しません。

## Observation

ADR-0023に従い、Command後の状態確認はCommand結果の派生値にしません。[`observation.go`](../../internal/driver/observation.go)の`ExecuteObservationRead`は、呼び出し側が選んだread identityと実行関数を受け取り、結果とEvidence natureを返します。

`ObservationEvidenceNature`は`synthetic` / `external-observed` / `managed-here`を区別します。Coreで列挙を閉じないのは、将来のEvidence種別をこの三つへ押し込めないためです。

read結果そのものと「期待状態に一致したか」というScenario上の評価は分離します。read失敗はCommand Business Resultへ畳みません。

## Driverが推測しないこと

この設計でもっとも重要な部分は、実装されている解決規則ではなく、意図的に実装していない補完規則です。

- Pattern KindからDatasourceやPersistence Mappingを補完しない
- Capability名やpayload field名から実装、read境界、parameterを選ばない
- Query本文（native SQL）を解析して物理table / column / 接続先を導出しない
- 列名の類似や型の一致でSemantic attributeへmappingしない
- providerが一致するCapabilityを探索してexecutorを自動選択しない
- Binding候補が複数あるときに順序や近さで選ばない
- Executorのerrorを正常な空結果へ変換しない
- Command acknowledgementをObserved stateへ変換しない

いずれも「解決できない」ことをerrorまたは診断として返し、推測で実行を継続しません。

## Scenario ReferenceとEvidenceの区別

Driver境界を一続きに接続したScenarioは[`gate_b_ui_less_scenario_test.go`](../../internal/driver/gate_b_ui_less_scenario_test.go)と[`final_gate_b_scenario_test.go`](../../internal/driver/final_gate_b_scenario_test.go)にあります。これらはtest-localなReference Scenarioであり、production契約ではありません。

Evidenceは性質ごとに分けます。取得状況そのものはこの文書へ複製せず、対応Issueと[`docs/status.md`](../../docs/status.md)を確認します。

| 種別 | 何を示すか | current stateの確認先 |
| --- | --- | --- |
| synthetic fixture | 意味境界の接続と決定論的な再現 | Repository内Scenario / `docs/status.md` |
| real-provider ephemeral fixture | 実PostgreSQL engineでの動作 | [ISSUE-000031](../issue/000031-gate-b-postgres-header-detail-evidence.md) / `docs/status.md` |
| actual external asset | 既存Oracle資産の観測 | [ISSUE-000022](../issue/000022-external-master-data-physical-query-evidence.md) / `docs/status.md` |

synthetic Scenarioの成功を、実Database、既存外部資産、本番運用の保証へ昇格させません。Gate Bの必須Evidence三系統と判定は[`plan/README.md`](../README.md)が正本です。

## 非主張範囲

- transaction境界、分散transaction、補償処理はDriverに実装していません
- retry、polling、eventual consistencyの一般化は行っていません
- 業務状態遷移（Lifecycle）はまだ意味として持ちません。設計は[業務Lifecycle v1 設計](BUSINESS_LIFECYCLE_V1_DESIGN.md)にあり、評価待ちです
- Web / Desktop等のTarget Projectionはこの層に含みません
- 5種類のPattern KindをTEI全体の永久的な標準Catalogとはみなしません

## 今後の論点

- Lifecycle評価をDriverへ接続する場合の評価順序（[ADR-0025](../adr/0025-transition-evaluation-and-conflict-result.md)、提案中）
- 複数Contract Queryをまたぐ実行の一貫性境界
- Target共有時に何をDriver結果として渡し、何をTarget固有に残すか（[Gate C Evidence契約](../GATE_C_EVIDENCE_CONTRACT.md)）
- Driver Scenarioの性能条件（RQ-024、Targetごとに定義）

## 関連文書

- [Application Pattern v1 設計](APPLICATION_PATTERN_V1_DESIGN.md) / [Capability Outcome v1 設計](CAPABILITY_OUTCOME_V1_DESIGN.md) / [既存認証Compatibility v1 設計](LEGACY_AUTH_V1_DESIGN.md)
- [Database Contract Query v1 設計](DATABASE_CONTRACT_QUERY_V1_DESIGN.md)
- [ISSUE-000006](../issue/000006-application-driver-vertical-slice.md) / [ISSUE-000018](../issue/000018-database-contract-query-execution.md) / [ISSUE-000019](../issue/000019-reference-contract-query-invocation.md) / [ISSUE-000020](../issue/000020-contract-query-runtime-instance-identity.md)
- [`docs/architecture.md`](../../docs/architecture.md) / [`docs/status.md`](../../docs/status.md)
