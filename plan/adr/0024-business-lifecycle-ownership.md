# ADR-0024: 業務Lifecycleの正本をSemanticへ置き、状態identityと物理コードを分離する

- 状態: 提案中
- 日付: 2026-09-05
- 関連Issue: [ISSUE-000038](../issue/000038-business-lifecycle-semantics.md)（作業範囲と実装順）
- 関連設計: [業務Lifecycle v1 設計](../design/BUSINESS_LIFECYCLE_V1_DESIGN.md)（評価待ち）

この判断は評価待ちです。採用が決まるまで、実装やReference変更の根拠にしません。

## 背景

第一Reference `order_entry`では、受注に`sales.order.status`という属性があり、submit Scenarioが状態を`pending`から`submitted`へ変えます。しかし「どの状態が正当か」「どの操作でどこへ進めるか」はRepository上のどこにも書かれていません。その知識はDriver testのsynthetic adapter内のGo実装だけが持っています。

状態遷移は業務システムの中心にある意味です。受注確定、承認、取消のように、同じ規則をDriver、Web Target、将来の第二Targetが共有する必要があります。この知識がGo実装や画面イベントの中にあると、変更箇所も適用条件も追跡できません。

一方で、汎用のWorkflow engineやStateMachine専用のCore primitiveを先に導入すると、第一Referenceだけからは確認できない共通性を固定することになります。現行のSemantic modelは`Type` / `Data` / `Relation` / `Capability` / `Validation`という小さな集合で、`Validation`も`positive`と`relation-unique`の二種類しかありません。この小ささは意図的な設計です。

## 判断

業務Lifecycleを、**Semantic namespaceが所有する新しい要素**として導入します。

```text
Semantic Namespace
├─ Type
├─ Data
├─ Capability
├─ Validation
└─ Lifecycle          // 追加
    ├─ Data / Attribute   対象と状態属性
    ├─ State              状態のidentity
    └─ Transition         from / to / capability / requires
```

このADRで確定するのは所有境界です。具体的なauthoring構文、診断code、Driver APIは[業務Lifecycle v1 設計](../design/BUSINESS_LIFECYCLE_V1_DESIGN.md)に置きます。

### 状態のidentityはCanonical IDで表す

状態は`"pending"`という文字列ではなく`sales.order.status_lifecycle.pending`のようなCanonical IDで識別します。

物理DBの`"pending"`、legacy systemの`"P"`、外部APIの`"PENDING"`はいずれも実現側の表現です。Canonicalへ物理コードを混ぜません。

### 状態コードのMappingはRealization側へ置く

CanonicalなstateとAdapterが読み書きする文字列の対応は、明示的なRealization Bindingとして持ちます。identityは既存の`BindingIdentity`と同じ考え方で`application/lifecycle`とします。

型やvalue名の類似からMappingを推測しません。

### 遷移条件は既存Validationの明示参照に限る

`requires`には既存の`semantic.Validation`のCanonical IDだけを書けます。新しい条件式言語をSemanticへ導入しません。「遷移元がどの状態か」はLifecycle構造そのものが表すため、条件としては書きません。

### Authorizationを複製しない

「誰がこの遷移を実行してよいか」はLifecycleへ書きません。ADR-0019の`security.authorize`が既に所有しています。

### Applicationは第二の遷移正本を持たない

Application Patternへ遷移を書きません。DriverはCommand PatternのCapability identityからtransitionを解決します。そのために「一つのCapabilityを参照するtransitionは高々一つ」という制約をWorkspace Validationで守ります。

## 採用しない案

### Application Pattern kindとして`lifecycle`を追加する

ADR-0017のとおり、Application Patternは各Applicationが所有する適用意図です。状態遷移の規則は業務そのものの意味であり、Applicationごとに別々の正本を持つべきものではありません。Pattern kindにすると、同じ受注の遷移規則が利用Applicationの数だけ増えます。

「どの遷移をどの操作から起こすか」というApplication側の選択は、既存のcommand Patternが持つCapability参照で十分に表せます。

### 既存の`Validation`を拡張して遷移条件を表す

`validationeval.Evaluate`の入力境界は`Number(attribute)`と`RelationKeys(relation, attribute)`だけです。状態の比較を扱うには評価器の境界そのものを広げる必要があります。

また、Validationは「対象の内容が妥当か」を表す関心であり、「どの状態からどこへ進めるか」とは別です。同じkind体系へ混ぜると、両者の評価タイミングと失敗の意味が曖昧になります。

### `OrderStatus`型を列挙型へ変更する

型は表現、許容値集合はLifecycleの所有物として分けます。`std.code`を列挙型にすると、同じ型を使う無関係な属性へ受注固有の値集合が波及します。

### StateMachine専用のCore primitiveと汎用式言語を先に導入する

第一Referenceの一遷移からは、階層状態、並行状態、時間による遷移、補償の共通性を確認できません。実際に必要になった時点で、Referenceを根拠に拡張します。

### 状態コードをPersistence Mappingへ書く

`persist sales_core.order`は`sales.order.status`を`status varchar`へ対応付けています。値の対応もここへ書く案は自然に見えます。

しかしv1では採りません。同じSemanticを別Applicationが異なる物理コードで利用する場合や、Databaseを経由しないAdapterが遷移を実現する場合を塞ぐためです。

ただしこの判断は暫定です。Database Contract側でも同じMappingが必要になった場合、Persistenceが正本になる可能性を残します。

## 影響

### 良くなる点

- 状態と遷移の変更箇所が一つになり、Driver / Web / 将来のTargetが同じ規則を共有できる
- 状態遷移の規則がSource Indexを通してauthoring sourceへ戻れる
- 異なる状態コードを持つ既存資産へ、意味を変えずに接続できる
- 認可規則の正本が増えない

### 増える負担

- Semantic model、Parser、anchor、Workspace Validation、Normalizationへ新しい要素が加わる
- Reference Workspaceと`world.json` goldenの更新が必要になる
- 状態コードMappingという明示的なauthoringが一つ増える

Mappingの明示は、TEIが物理表現を推測しないために必要なコストとして受け入れます。

## 未決の論点

- 状態コードMappingの最終的な所有層（Application / Database / 双方）
- 一つのDataが複数の状態属性を持つ場合の表現
- 初期状態と終了状態を宣言する必要があるか
- 遷移履歴をどこへ保持するか

これらは第二のReferenceか実業務の要求が現れた時点で判断します。

## 検証

ISSUE-000038で、第一Referenceに対して次を確認します。

1. 状態と遷移をauthoringでき、load → Canonical → Source Index → Diagnosticを追跡できる
2. file移動でidentityと意味が変わらない
3. 異なる状態コードのMappingでも同じ意味を保てる
4. Application PatternとAuthorizationを変更せずに遷移を接続できる

## 関連

- ADR-0017: Application Patternの所有境界
- ADR-0018: Application Patternの正本とfile pathからのidentity分離
- ADR-0019: Authentication / Authorizationの境界
- ADR-0025: 遷移の評価順序と成立しなかった遷移の結果契約
- [`requirements/TEI_PRODUCT_REQUIREMENTS.md`](../../requirements/TEI_PRODUCT_REQUIREMENTS.md) RQ-001 / RQ-003 / RQ-007 / RQ-014 / RQ-019
