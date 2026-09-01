# ISSUE-000005: `order_entry`でApplication Patternを実証する

- 状態: 進行中
- GitHub Issue: #67
- 設計・実装基線: [`APPLICATION_PATTERN_V1_DESIGN.md`](../APPLICATION_PATTERN_V1_DESIGN.md)

## 背景

Product Requirement Gap Auditでは、Application PatternがDriver、Target、Editor、AI支援へ進む前に埋めるべき大きなGapとして残りました。

v1では、既存の`reference/workspace/applications/order-entry`をReference Applicationとして育て、次の5種類をApplicationが所有する意図の候補として段階的に検証します。

- `browse`
- `record`
- `reference`
- `command`
- `header-detail`

この5分類を現時点でTEI全体の永久的な標準語彙とはしません。具体的なReferenceから必要性と責務を確認し、DriverやTargetからのEvidenceも使って見直します。

Patternの所有境界は[ADR-0017](../adr/0017-application-pattern-ownership.md)、正本の配置とfile pathからのidentity分離は[ADR-0018](../adr/0018-application-pattern-authoring-source.md)に従います。

## 目的

Application Patternを単なる画面テンプレートとして固定せず、**同じApplication上の意図をDriverや複数Targetから利用できる単位として成立させられるか**を`order_entry`で確認します。

このIssueはv1 Reference Application実証計画のStage 1です。完了しただけで直ちにDriverへ進むのではなく、`ISSUE-000003`の既存認証Compatibilityと合わせてGate A「Application上の意味をDriverへ渡せる」を評価します。

## 現在地

`browse`は、基盤実装から実Referenceまで次のところへ進んでいます。

### PR #87〜#89

- 最小Canonical modelと`browse` Parser
- `patterns/*.tei` Loader
- Canonical ID順Normalization
- `application-pattern:<Canonical ID>`のSource Index
- Pattern ID tokenへのsource range
- 小さいfixtureでのfile path / Pattern identity分離

### PR #95

- Pattern Canonical ID重複の検出
- Data / Query / Datasourceの参照整合性Validation
- QueryのApplication所有関係の検査
- Pattern側DatasourceとQuery側Datasourceの矛盾検出
- Query側Datasourceだけで成立できることの確認

### PR #99

- `ValidateApplicationPatterns`によるPattern固有Validationの外部composition
- DiagnosticのPattern Subject保持
- Subjectから既存Source Indexを通じてPattern ID tokenへ戻るDiagnostic projection

### PR #101

- `order_entry.recent_orders`実Reference Pattern
- `sales.order`と既存native SQL Queryのidentity参照
- Queryが持つDatasource `sales`を利用し、Pattern側で重複記述しないことの確認
- Patternを含む`reference/world.json`のGolden更新
- 実Reference PatternのSubjectから実sourceへ戻るSource Index確認

ここまでで、`browse`についてmodel / Parser / Loader / Normalization / Source Index / Pattern固有Validation / Diagnostic projection / 実Reference / Golden Worldまで成立しています。

## `browse`で残っている二つのEvidence

### `ISSUE-000010` — 共通Validation統合

[`ISSUE-000010`](000010-application-pattern-validation-integration.md) / GitHub Issue #98で、Pattern固有Validationを通常の`validation.Validate`へ統合します。

CLI、Editor、Driver等の利用側がApplication Patternだけを特別扱いして追加Validationを呼ばなくても、同じWorkspace validation契約を利用できる状態を目指します。

### `ISSUE-000011` — 実Referenceのrename invariance

[`ISSUE-000011`](000011-order-entry-browse-reference.md) / GitHub Issue #100では、PR #101で追加した実Referenceのfile nameを変えた場合も意味が変わらないことを確認します。

- Pattern Canonical IDが変わらない
- normalized Canonical Worldが変わらない
- Source IndexのSubject identityが変わらない
- Source Artifact pathだけが新しいpathへ追随する
- 古いpathが残らない

この二つの実装順は固定しません。両方が揃った時点で、`browse`を通常のWorkspace validationから実Reference、Golden World、Source Indexまで一続きに扱える状態になります。

## `browse`後の実証順

1. `record`: `sales.order`をApplication上の一件として扱う
2. `reference`: `sales.order.customer`から`customer.customer`を参照し、`master` Datasourceのread-only境界を維持する
3. `command`: `sales.order.submit` Capabilityへ接続する
4. `header-detail`: `OrderLine`と対応RelationをSemantic側へ追加した後に扱う

5種類を一度に有効化せず、各Referenceで必要なCanonical fieldとValidationを確認してから進めます。

## 所有境界

ADR-0017に従い、次を混同しません。

1. Semantic上の意味
2. Application上の利用意図
3. Projection / Target固有の表現とInteraction

現在のPattern名や分類は永久的な標準語彙ではありません。Driver、第一Web Target、第二Targetから得られるEvidenceを使い、Applicationに残す意味、共有知識へ持ち上げる意味、Projectionへ下げる情報を見直します。

## 正本

Patternの正本はADR-0018に従い、Application配下の`patterns/*.tei`へ置きます。

Pattern identityは明示的なCanonical IDとし、file pathやfile nameから導出しません。

同じApplication上の意図について、DSLとEditor内部状態を暗黙の二重正本にはしません。

## Gate Aへ渡す情報

Application Pattern側から少なくとも次を説明できるEvidenceを揃えます。

- 何を`browse`するか
- 一つの`record`として何を扱うか
- 何を別Datasourceから`reference`するか
- HeaderとDetailをどのようなApplication上の単位として扱うか
- どのValidationを実行するか
- どのCapabilityを`command`として呼ぶか
- どのDatasourceがread-onlyか
- Patternの正本とCanonical identityをどのように追跡するか

Authorizationは`ISSUE-000003`の既存認証Compatibilityから合流させます。

## 完了条件

`order_entry`の業務シナリオで5種類のPatternを一続きに表現し、Application上の意図とProjection / Target固有情報の境界を説明できることを完了条件とします。

さらに、Patternの正本、Canonical identity、Source Index、実Referenceでのfile rename invarianceをRepository内のReference / Conformanceから再現でき、その同じApplication上の意味を後続Driverが利用できる必要があります。

現時点ではStage 1、Gate A、Milestone 1の完了は宣言しません。

## 関連文書

- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/APPLICATION_PATTERN_V1_DESIGN.md`
- `plan/adr/0017-application-pattern-ownership.md`
- `plan/adr/0018-application-pattern-authoring-source.md`
- `plan/issue/000003-legacy-authentication-compatibility-reference.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000010-application-pattern-validation-integration.md`
- `plan/issue/000011-order-entry-browse-reference.md`

関連RQ: RQ-001, RQ-003, RQ-005, RQ-011, RQ-012, RQ-014, RQ-017, RQ-021
