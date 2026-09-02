# ISSUE-000006: Application Driverの最小vertical sliceを実装する

- 状態: In Progress
- GitHub Issue: #69
- 前段: `ISSUE-000013` / GitHub Issue #124（Gate A Ready）
- 次のGate: Gate B「UIなしで主要Scenarioを再現できる」
- Query実行の追跡Issue: `ISSUE-000016` / GitHub Issue #167

## 背景

Product Requirement Gap Auditでは、RQ-012「Driverを設計検証用の実行・シミュレーション環境として利用できること」が未着手でした。

現行RepositoryにはDeployment executorやprocess Plugin等の実行基盤がありますが、それらはApplication設計を開発途中に動かし、意味の不足や矛盾を見つけるDriverとは役割が異なります。

Gate A監査 #124により、Driverへ渡すApplication上の意味はReadyになりました。現在は、その意味をDriver専用コードへ複製せず、小さなvertical sliceを一つずつmainへ積み上げています。

## Driverの役割

第一段階のDriverはproduction Runtimeの簡易版ではありません。

設計途中のApplicationについて、UIがなくても主要な業務Scenarioを一続きに実行し、次のような問題を早い段階で見つけるための設計検証環境です。

- Patternが参照する意味を一意に解決できない
- logical DatasourceやBindingが不足している
- 業務Validationが成立しない
- Authentication / Authorizationで後続処理へ進めない
- CapabilityのBusiness ResultとExecution Errorが混同されている
- native SQLや既存実装とCanonicalな意味の間に対応関係が不足している

```text
Application定義
       ↓
Pattern / Datasource / Bindingの解決
       ↓
Driver Scenario
       ↓
Authentication / Query / Load / Edit / Validate / Authorize / Command
       ↓
Business Result / Execution Error
       ↓
Observed Result / Evidence
```

Driverは不足する意味を埋める場所ではありません。実行中にGapを発見した場合は、意味を所有すべきApplication / Semantic / Query / Binding等の上位層へ戻します。

## 実装済みのvertical slice

### 1. Application PatternのCanonical解決 — PR #161

`internal/driver.Resolver`を追加しました。

Application IDとPattern IDを起点に、Pattern Kindごとに既存Canonical modelを解決します。

- `browse`: Semantic Data / Query / Queryが保持するlogical Datasource
- `record`: Semantic Data
- `reference`: Relation / source・target Data / Patternに明示されたDatasource
- `header-detail`: `many` Relation / source・target Data
- `command`: Semantic Capability

次は推測しません。

- `record`の保存先Datasource
- `header-detail`のPersistence Mapping
- `command`のBinding実行
- file pathやGo symbolからの意味発見

### 2. Capability Bindingの明示解決 — PR #162

Application + Capability identityから既存の`CustomGoBinding`を一意に解決する入口をDriverへ接続しました。

Referenceでは次を同じ原則で扱います。

```text
order_entry / sales.order.submit
order_entry / security.authenticate
order_entry / security.authorize
```

Bindingが0件または複数件なら、列挙順等で一件を選ばず実行準備上のerrorにします。

### 3. Binding経由のCapability実行 — PR #164

解決済みBindingに対応する`capabilityinvoke.Adapter`を受け取り、既存のInvocation境界からCapabilityを実行できるようにしました。

Driver本体は`legacyauth`等の具体実装を知りません。

Authentication Referenceで次を確認しています。

- `authenticated(subject)`はBusiness Result
- credential不一致の`denied`もBusiness Result
- Adapter解決失敗やAdapter障害はExecution Error
- Capability契約にないOutcomeもExecution Error
- 実行時credentialをDriverの実行結果へ保持しない

### 4. AuthenticationからAuthorizationへの接続 — PR #165

PR #164で作った共通実行入口をそのまま再利用し、Driver本体へSecurity専用コードを追加せずAuthorizationまで接続しました。

```text
security.authenticate
        ↓ authenticated subject
security.authorize
        ↓
allowed / denied
```

Referenceでは次を確認しています。

- `ORDER_EDITOR`は`browse / order_entry.recent_orders`を許可される
- `ORDER_VIEWER`の`edit / order_entry.order_record`は`denied`になる
- 認可拒否をExecution Errorへ変換しない
- Driverはlegacy organization / roleを直接読まない

### 5. browse Queryの実行境界 — PR #166

`browse` Patternから解決したCanonical Queryとlogical Datasourceを、そのまま`BrowseQueryExecutor`へ渡す最小境界を追加しました。

```text
order_entry.recent_orders
        ↓
Query order_entry.recent_orders
        ↓ datasource sales
Application database sales
        ↓
sales_core / read-write
        ↓
BrowseQueryExecutor
```

次を確認しています。

- native SQLをDriverへ複製しない
- SQL本文から物理Databaseやtableを推測しない
- Workspace LoaderがCanonicalに正規化したQuery SourceをDriverがさらに加工しない
- Query parameterをExecutorへ渡せる
- 実行時parameterをDriverの実行結果へ保持しない
- Executor障害を正常な空結果へ丸めない
- browse以外のPatternをQueryとして実行しない

初回CI #397では、テストが外部SQLファイルとCanonical Query Sourceのbyte完全一致を仮定していたため失敗しました。既存Loaderの仕様を読み直し、ファイル外縁の空白を除去してCanonicalへ正規化することが既存仕様であると確認したうえでテストを修正しました。最終CI #398は全成功しています。

## 現在の主要Gap

### native Query Runtime — ISSUE-000016 / #167

第五縦切りの結果、実PostgreSQLへ進む前に二つの意味を確立する必要があると分かりました。

#### named parameter binding

Reference SQLは`:from`のようなnamed parameterを使います。

これをPostgreSQL固有placeholderへ変換する処理をDriver Coreへ簡易実装すると、文字列、comment、cast等を誤認してSQLの意味を壊す可能性があります。

Provider / Query Executor側の責務として、安全なparameter bindingを成立させます。

#### Query result mapping

native SQLが返す物理列とSemantic Dataのattributeを対応付けるCanonicalな正本は、まだありません。

この状態でDriverが列名から推測してSemantic Dataを作ることはしません。まずraw resultを観測し、`record / reference / header-detail`へ本当に必要な意味を確認したうえで、mappingの所有層を決めます。

詳細は`ISSUE-000016`を正とします。

## Gate Aから受け取っている意味

### Pattern / Data access

`order_entry`では次を解決できます。

- `browse`: `order_entry.recent_orders`
- `record`: `order_entry.order_record`
- `reference`: `sales.order.customer` + datasource `master`
- `header-detail`: `sales.order.lines -> sales.order_line many`
- `command`: `order_entry.submit_order -> sales.order.submit`
- `sales`: `sales_core` / read-write
- `master`: `master_data` / read

### 業務Validation

既存のCanonical ruleとEvaluatorを利用します。

```text
sales.order_line.quantity_positive
sales.order.lines.line_number_unique
```

Validation不成立は`NotSatisfied`、評価自体を完了できない状態はerrorです。

### Capability結果境界

ADR-0020 / Issue #77を正とします。

```text
Business Result
  ├─ Output value
  └─ Named Outcome(name, payload)

Execution Error
  └─ Invocation自体を完了できなかった状態
```

Authenticationの`denied`、Authorizationの`denied`、Validationの`NotSatisfied`はDriver自体の障害ではありません。

## 最小Scenario

`order_entry`で最終的に次を一続きに再現します。

1. Authenticationを実行してsubjectを得る
2. `browse`で受注一覧を取得する
3. `record`で一件を扱う
4. `reference`で顧客を`master_data`から解決する
5. `header-detail`でOrderLineを扱う
6. Canonical業務Validationを評価する
7. Authorizationを評価する
8. `command`で`sales.order.submit`を呼び出す
9. Business ResultとExecution Errorを区別する
10. 必要な場合はObservationから外部状態を確認する

この順序を汎用Workflow Engineへ一般化することは、第一段階の目的ではありません。

## Database境界

`sales_core`と`master_data`を意味上ひとつへ統合しません。

```text
sales  -> sales_core  -> read-write
master -> master_data -> read
```

`master_data`へwriteしないことをScenarioのConformanceとして確認します。

接続資格情報やConnection PoolはDriver CoreのCanonicalな意味ではありません。Reference composition / Provider側から実行環境へ与えます。

## SecretとEvidence

再現性を理由にpassword、token、credential hash、接続資格情報を通常Evidenceへ保存しません。

Evidenceへ残せる候補は次です。

- secretではない入力識別情報
- resolved Pattern
- resolved logical Datasource
- resolved Capability Binding
- Query identityとnative sourceの来歴
- Authentication / Authorization / Validationの結果
- Business Result
- Execution Errorの有無
- command acknowledgement
- observed result

Scenario EvidenceはCanonicalの正本ではありません。

## CommandとObservation

Capability invocationの結果と、外部世界で実際に成立した状態を分けます。

`sales.order.submit`では少なくとも次の三つを区別します。

1. Capability呼び出しを開始・完了できたか
2. CapabilityがどのBusiness Resultを返したか
3. 外部世界で期待した状態が実際に成立したか

必要な場合は3をObservationから確認します。

## 次の作業

1. `ISSUE-000016`でPostgreSQL Query Executorの境界を成立させる
2. `recent_orders`を実PostgreSQLで実行する
3. raw resultからresult mappingに必要な意味を観測する
4. mappingの正本を適切な上位層へ追加する
5. `record / reference / header-detail`をScenarioへ接続する
6. Canonical Validationを既存Evaluatorへ接続する
7. Authorization不成立時にcommandへ進まないことを確認する
8. `sales.order.submit`を既存Bindingから呼び出す
9. Scenario EvidenceをRepositoryから再現可能な形へ整える
10. Gate Bを監査する

## Gate Bへ進む条件

Driverのコードが存在するだけではGate B Readyとはみなしません。

少なくとも次をReference Scenarioから確認します。

- 同じApplication定義からScenarioを再現できる
- Pattern / Query / Relation / Capability / Datasource / Bindingの解決を追跡できる
- native SQLの正本をDriverへ複製しない
- logical datasource aliasを保持する
- `master_data`へwriteしない
- legacy authentication schemaへDriverが直接依存しない
- Validation不成立時にcommandへ進まない
- Authorization `denied`時にcommandへ進まない
- expectedなnamed OutcomeをDriver障害として扱わない
- Execution Errorを業務Outcomeへ押し込まない
- command acknowledgementとobserved resultを混同しない
- secretを通常Evidenceへ混入させない
- Driver固有状態をCanonical Worldへ逆流させない
- 同じApplication上の意味を第一Web Targetへ渡せる
- Reference ScenarioをRepositoryだけから再現できる

## 非目標

- production-grade Workflow Runtime
- distributed execution / production HA
- 完成済みWeb / Desktop UI
- arbitrary user scripting environment
- full transaction coordinator
- external orchestratorの内製
- 汎用ORM
- Driver独自のOutcome体系
- Driver独自のValidation / Authentication / Authorization rule
- Runtime上の汎用implementation選択機構
- DB固有placeholder処理をDriver Coreへ固定すること
- 未定義のQuery result mappingをDriverが推測すること

## 関連

- `plan/GATE_A_READINESS_AUDIT.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/issue/000016-native-query-runtime-boundary.md`
- GitHub Issue #67 / #68 / #69 / #77 / #124 / #125 / #126 / #167
- GitHub PR #161 / #162 / #164 / #165 / #166
- ADR-0019 / ADR-0020
- RQ-005, RQ-008, RQ-009, RQ-011, RQ-012, RQ-014, RQ-017, RQ-018, RQ-021
