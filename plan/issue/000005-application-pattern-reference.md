# ISSUE-000005: `order_entry`でApplication Patternを実証する

- 状態: 進行中
- GitHub Issue: #67
- 設計・実装基線: [`APPLICATION_PATTERN_V1_DESIGN.md`](../APPLICATION_PATTERN_V1_DESIGN.md)

## 背景

Product Requirement Gap Auditでは、RQ-014「標準的なApplication Patternを再利用できること」が、Driver、Presentation、Editor、AI支援へ進む前に埋めるべき大きなGapとして残っています。

ただし、v1で最初から「標準Pattern」を確定するわけではありません。既存の`reference/workspace/applications/order-entry`をReference Applicationとして育て、次の5種類をApplicationが所有する意図の候補として順番に検証します。

- `browse`
- `record`
- `reference`
- `header-detail`
- `command`

Patternの所有境界は[ADR-0017](../adr/0017-application-pattern-ownership.md)、正本の配置とfile pathからのidentity分離は[ADR-0018](../adr/0018-application-pattern-authoring-source.md)で定めます。

## 関連するProduct Requirement

- RQ-001 業務上の意味
- RQ-003 暗黙の複数正本を作らない
- RQ-005 Application ↔ Database N:M
- RQ-011 Projection
- RQ-012 Driver
- RQ-014 Application Pattern
- RQ-017 二重管理の最小化
- RQ-021 Reference / Conformance

## 目的

Application Patternを単なる画面テンプレートとして固定せず、**同じApplication上の意図をDriverや複数Targetから利用できる単位として成立させられるか**を`order_entry`で確認します。

このIssueはv1 Reference Application実証計画のStage 1です。完了しただけで直ちにDriverへ進むのではなく、`ISSUE-000003`の既存認証Compatibilityと合わせてGate A「Application上の意味をDriverへ渡せる」を評価します。

## 現在地

Application Patternはすでに実装へ着手しています。ただし、5 Patternを`order_entry`で一続きに実証した段階ではありません。

### PR #87 — 最小Canonical modelと`browse` Parser

次を実装済みです。

- `application.Pattern`
- `Application.Patterns`
- ApplicationがPatternを子として所有する構造
- `browse`だけを有効化した`PatternKind`
- Pattern ID、Data、Query、任意のDatasource
- `browse`でData / Queryを必須にするParser
- 同じfieldの重複を拒否するParser validation

Semantic定義、Query本文、file path、Target固有PresentationはPatternへ複製していません。

### PR #88 — `patterns/*.tei` LoaderとNormalization

次を実装済みです。

- `applications/*/patterns/*.tei`の列挙
- Pattern sourceのParse
- 所属Applicationの`Application.Patterns`への統合
- Canonical ID順のNormalization
- `application-pattern:<Canonical ID>`のsource mapping

ファイル名の列挙順とCanonicalなPattern順を分離するConformanceも追加しています。

### PR #89 — Source IndexのPattern識別子range

次を実装済みです。

- `application-pattern:<Canonical ID>`からPattern宣言へ戻るSource Index経路
- Pattern ID token全体を指すsource range
- 既存のDeclaration Identifier Anchorの再利用
- Patternファイルを改名してもSubject identityを維持し、Artifact pathだけを追随させる検証

ここまでで、Pattern基盤のmodel / Parser / Loader / Normalization / Source Indexは`browse`を扱えるところまで進みました。

## まだ実証していないこと

現在の`reference/workspace/applications/order-entry`には、まだ`patterns/`がありません。

そのため、次は未完了です。

- `order_entry.recent_orders`の実Reference Pattern
- Patternが参照するData / Query / DatasourceのWorkspace全体での整合性Validation
- Pattern固有Diagnosticのend-to-end Conformance
- Patternを含む`reference/world.json`の更新と再生成一致
- 実Referenceを使ったCanonical World全体のfile rename invariance
- `record` / `reference` / `command` / `header-detail`
- 5 PatternをDriverへ渡せる一続きのApplication上の意味

PR #89で確認したrenameはSource IndexのSubject identityとArtifact pathに関するものです。これだけでCanonical World全体のrename invarianceまで実証済みとはみなしません。

## 次の優先単位

新しいPattern Kindを増やす前に、現在の`browse`基盤を実Referenceとして閉じます。

### Step 1 — Workspace参照整合性Validation

Parser単体では確認できない次をWorkspace / Validation層で検査します。

- Pattern IDがApplication内で重複していない
- 参照するDataが存在する
- 参照するQueryが存在し、対象Applicationから利用できる
- 明示したDatasource aliasがApplicationに存在する
- Pattern Kindに必要な参照が揃っている

不整合は、可能な限り元のPattern定義へ戻れるDiagnosticとして返します。

### Step 2 — `order_entry.recent_orders` browse Reference

既存の次の資産をそのまま利用します。

- Semantic Data `sales.order`
- Query `order_entry.recent_orders`
- logical datasource `sales`
- native SQL `queries/recent_orders.sql`

SQL本文やQuery定義をPatternへコピーしません。

Query自身がDatasourceを持つ場合、Pattern側にもDatasourceを明示すべきかはReferenceから判断します。同じ意味を複数箇所へ手入力し続けないことを優先します。

### Step 3 — Reference World / Diagnostic / rename invariance

実Referenceを追加したうえで、次を一続きに確認します。

```text
patterns/recent_orders.tei
        ↓
Parser / Loader
        ↓
Application.Patterns
        ↓
Normalization / Validation
        ↓
reference/world.json
        ↓
Source Index / Diagnostic
```

この段階で、Patternを含むCanonical Worldの再生成一致と、実Referenceでのfile rename invarianceまで閉じます。

## 後続Patternの実証順

`browse`がReference / Conformanceまで成立した後、既存資産だけで扱えるものを先に追加します。

1. `record`: `sales.order`をApplication上の一件として扱う
2. `reference`: `sales.order.customer` Relationから`customer.customer`を参照し、`master` Datasourceのread-only境界を維持する
3. `command`: `sales.order.submit` Capabilityへ接続する
4. `header-detail`: `OrderLine`と対応RelationをSemantic側へ追加した後に扱う

5種類を一度に有効化せず、それぞれのReferenceで必要なCanonical fieldとValidationを確認してからPattern Kindを追加します。

## `command`とCapability結果

`command` Patternは既存の`sales.order.submit` Capabilityへ接続します。

Capability結果の共通境界は[ADR-0020](../adr/0020-capability-business-outcome-boundary.md)と[`CAPABILITY_OUTCOME_V1_DESIGN.md`](../CAPABILITY_OUTCOME_V1_DESIGN.md)に従います。

ただし、`command` Patternを追加するという理由だけで、現行`sales.order.submit`の`output`をnamed Outcomeへ書き換えません。Commandを正常に実行して初めて分かる業務上の分岐がReferenceで必要な場合に、Capability Outcomeの圧力試験として扱います。

## `header-detail`と`OrderLine`

`header-detail`では、概念上次の関係を扱います。

```text
Order 1 ── N OrderLine
```

`OrderLine`はPattern基盤と同時に追加せず、`browse`、`record`、`reference`、`command`の境界を確認した後に導入します。

Semantic Relation、Application上の`header-detail`、Database Foreign Keyは別の意味単位として扱い、RelationからFKや画面構造を暗黙生成しません。

## 所有境界

ADR-0017に従い、次の三層を混同しません。

1. Semantic上の意味 — Data / Relation / Capabilityなど
2. Application上の意図 — browse / record / reference / header-detail / commandなど
3. Projection / Target — Webなどに固有の表現とInteraction

現在のPattern名や分類は永久的な標準語彙ではありません。Driver、第一Web Target、第二Targetから得られるEvidenceを使い、Applicationに残す意味、共有知識へ持ち上げる意味、Projectionへ下げる情報を見直します。

## 正本

ADR-0018に従い、Patternの正本はApplication配下の`patterns/*.tei`へ置きます。

Pattern identityは明示的なCanonical IDとし、file pathやファイル名から導出しません。

同じApplication上の意図について、DSLとEditor内部状態を暗黙の二重正本にはしません。当面のEditorは定義sourceのProjectionと編集支援を担います。

## Gate Aへ渡す情報

このIssueだけでGate Aを完了とはしませんが、Application Pattern側から少なくとも次を説明できるEvidenceを揃えます。

- 何を`browse`するか
- 一つの`record`として何を扱うか
- 何を別Datasourceから`reference`するか
- HeaderとDetailの関係
- どのValidationを実行するか
- どのCapabilityを`command`として呼ぶか
- どのDatasourceがread-onlyか
- Patternの正本とCanonical identityをどのように追跡するか

Authorizationは`ISSUE-000003`の既存認証Compatibilityから合流させます。

## Conformance

### 現在までに確認済み

- `browse` PatternをApplication modelで保持できる
- Pattern IDをfile pathから独立して保持できる
- Pattern sourceを`patterns/*.tei`からLoaderで読み込める
- Pattern順序をCanonical IDで正規化できる
- Source IndexからPattern ID tokenへ戻れる
- Source IndexのSubject identityをfile renameから独立させられる

### 次に確認すること

- Pattern参照のWorkspace整合性Validation
- Pattern固有Diagnosticのend-to-end projection
- `order_entry.recent_orders` Queryを実Referenceから参照できる
- native SQLをPattern側へ複製しない
- Patternを含むReference WorldをRepositoryだけから再生成できる
- 実Referenceのfile renameだけではCanonical Worldが変わらない

### 後続Patternで確認すること

- `reference`が`master`のread-only Bindingを維持する
- `command`が既存Capabilityへ接続できる
- `header-detail`のSemantic RelationとPersistence FKを混同しない
- Driverが利用するためにWeb固有情報を必要としない

## 非目標

- Patternの網羅的なcatalog化
- 5分類を現時点でTEI標準語彙として確定すること
- Patternごとの完成UI
- responsive designなどのTarget固有表現
- automatic UI design
- PatternをSemantic Kernelへ一括追加すること
- AIによるPattern推薦
- Pattern Catalogの共有方式を先に決めること
- Patternの継承・合成を先に実装すること
- このIssueだけでGate Aを完了させること

## 完了条件

`order_entry`の業務シナリオで5 Patternを一続きに表現でき、どこまでがApplication上の意図で、どこからがProjection / Target固有情報かを説明できることを完了条件とします。

さらに、Patternの正本、Canonical identity、Source Index、実Referenceでのfile rename invarianceをRepository内のReference / Conformanceから再現できること。その同じApplication上の意味を後続Driverが利用でき、`ISSUE-000003`のAuthorization境界と合わせてGate Aを評価できるEvidenceを残すことを求めます。

現在は、`browse`基盤のmodel / Parser / Loader / Normalization / Source Indexまでが実装済みです。Stage 1やMilestone 1の完了はまだ宣言しません。

## 関連文書

- `plan/APPLICATION_PATTERN_V1_DESIGN.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/adr/0017-application-pattern-ownership.md`
- `plan/adr/0018-application-pattern-authoring-source.md`
- `plan/adr/0020-capability-business-outcome-boundary.md`
- `plan/CAPABILITY_OUTCOME_V1_DESIGN.md`
- `plan/issue/000003-legacy-authentication-compatibility-reference.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
