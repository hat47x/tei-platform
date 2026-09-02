# TEI v1 Reference Application 実証計画

## 目的

この文書は、TEI（鼎／テイ）がv1へ進むために必要なApplication側の実証を、Reference Application `order_entry`を中心に一つの流れとして整理するものです。

個別のIssueを別々に完了させることではなく、同じ業務上の意味をSemanticからApplication、Driver、Targetへ一続きに追跡できることを重視します。

期限は置きません。次へ進む条件は日付ではなく、ReferenceとConformanceから確認できるEvidenceで判断します。

## v1 Reference Application

v1では、新しい理想化されたサンプルを別に作らず、`reference/workspace/applications/order-entry`を育てます。

現在の`order_entry`には、すでに次の資産があります。

- `sales` / `customer` Semantic Namespace
- `sales.Order` / `sales.OrderLine`
- `sales.order.customer` Relation
- `sales.order.lines` 1:N Relation
- `sales.order.submit` Capability
- `sales_core`へのread-write接続
- `master_data`へのread-only接続
- Database Contract
- native SQL Query
- 5種類のApplication Pattern
- `sales.order.submit`の明示的なCapability Binding
- Source-to-Canonical Traceability
- Reference Workspaceと再生成可能なCanonical World

M0.xxで積み上げたDatabase / Provider / RuntimeのEvidenceを捨てず、Application vertical sliceへ接続します。

## v1で確認する業務シナリオ

最初の業務シナリオは、次の一続きの操作です。

```text
受注一覧を探す
    ↓
受注を開く
    ↓
顧客を参照する
    ↓
受注明細を扱う
    ↓
入力内容を検証する
    ↓
利用者の認証・認可を確認する
    ↓
受注を確定する
    ↓
業務結果と実行エラーを区別する
    ↓
必要に応じて外部状態を観測する
```

この意味をPattern、Driver、Web Targetごとに作り直しません。同じCanonicalな意味を共有し、それぞれの層では必要なProjectionやBindingだけを追加します。

## 全体の進行

```text
Stage 0  Product Baseline / Gap Audit        完了
               ↓
Stage 1  Application Pattern                 完了
               ↓
Gate A   Application上の意味をDriverへ渡せるか監査
               │
               ├─ #68   既存認証Compatibility
               ├─ #77   Capability Outcome / Execution Error
               └─ #125  業務Validation
               │
               │  #126 Capability Binding は完了
               ↓
          Gate A再監査
               ↓
Stage 2  Application Driver #69
               ↓
Gate B
               ↓
Stage 3  第一Web Target #70
               ↓
Gate C
               ↓
Stage 4  Editor / AI支援
               ↓
Gate D
               ↓
Stage 5  第二Target
```

Database / Provider側の重要な安全課題は並行して扱いますが、v1の主線はApplication vertical sliceです。

## Stage 0 — Product BaselineとGap Audit

### 状態

完了しています。

### Evidence

- Product Value
- Product Vision
- Product Principles
- Product Requirements
- Master Plan
- 期限を置かないMilestone
- Product Requirement Gap Audit

### 得た判断

Database / Provider、Source / Meaning、Runtime安全境界に比べ、Application Pattern、既存認証、Driver、Target、Editor、AI支援の実証が薄いことが明確になりました。

この偏りを受け、v1の主線をApplication vertical sliceへ移しています。

## Stage 1 — Application Pattern

関連: `ISSUE-000005` / GitHub Issue #67  
詳細: [`APPLICATION_PATTERN_V1_DESIGN.md`](APPLICATION_PATTERN_V1_DESIGN.md)

### 状態

**完了しています。**

候補としていた5種類が、同じ`order_entry`の実Referenceとして成立しました。

```tei
pattern order_entry.recent_orders browse {
    data sales.order
    query order_entry.recent_orders
}

pattern order_entry.order_record record {
    data sales.order
}

pattern order_entry.customer_reference reference {
    relation sales.order.customer
    datasource master
}

pattern order_entry.submit_order command {
    capability sales.order.submit
}

pattern order_entry.order_lines header-detail {
    relation sales.order.lines
}
```

### Stage 1で確認したこと

- PatternはApplicationが所有する意図として成立する
- Semantic / Query / Relation / Capabilityの意味をPatternへ複製しない
- Kindごとに必要なfieldだけを保持できる
- 不要なfieldを明示的に拒否できる
- ParserとWorkspace Validationの責務を分離できる
- Pattern identityをfile pathから分離できる
- Source Index / Diagnosticからauthoring sourceへ戻れる
- Reference WorldをRepository自身から再生成できる
- Semantic RelationとDatabase Foreign Keyを混同しない
- PatternをTarget固有UIへ固定しない

### Stage 1で固定していないこと

5種類をTEI全体の永久的な標準一覧とはみなしません。

Driverと複数TargetからEvidenceを得た後に、Patternの共通性、合成、共有Catalog、Presentationとの境界を改めて評価します。

## `OrderLine` Semantic substrate

`header-detail`を実証するために、Patternとは独立して`sales.OrderLine`と1:N Relationを追加しました。

```tei
type OrderLineId from std.identifier

data OrderLine {
    attribute orderLineId type sales.order_line_id
    attribute lineNumber type std.integer
    attribute itemCode type std.code
    attribute quantity type std.decimal
    attribute unitPrice type std.money
}
```

```tei
relation lines to sales.order_line many
```

この追加だけを理由にPersistence Mapping、Database Foreign Key、Database Contract、Migrationは生成していません。

Semantic上の関係、Application上の意図、物理DB上の実現を別の意味単位として扱います。

## Gate A — Application上の意味をDriverへ渡せる

関連: `ISSUE-000013` / GitHub Issue #124  
監査結果: [`GATE_A_READINESS_AUDIT.md`](GATE_A_READINESS_AUDIT.md)

### 現在の状態

**Blockedです。**

Application PatternとCapability BindingはReadyです。Capability Outcomeも、名前付きOutcomeをCanonicalに表現する基盤までは実装済みです。

一方、Driver開始前には次の3領域が残っています。

| 領域 | 現在地 | 追跡先 |
| --- | --- | --- |
| Authentication / Authorization | Reference未成立 | #68 / `ISSUE-000003` |
| named Outcome / Execution Error | Canonical基盤は実装済み。Reference / Invocation境界が未完了 | #77 / `ISSUE-000009` |
| 業務Validation | 正本とidentityが未成立 | #125 / `ISSUE-000014` |

Capability BindingはIssue #126 / PR #135で完了しました。

Driver #69は、残る意味をDriver専用Goコードや設定で補わないよう、Gate A通過まで主実装をBlockedとします。

### 現時点でReadyな情報

- browse対象: `order_entry.recent_orders`
- record対象: `order_entry.order_record`
- reference対象: `sales.order.customer` + datasource `master`
- header-detail関係: `sales.order.lines -> sales.order_line many`
- command対象: `sales.order.submit`
- Pattern identity / Source Index
- `sales` read-write / `master` readのDatasource境界
- `sales.order.submit`の既存単一`output sales.order`
- `sales.order.submit`の明示的なCapability Binding
- Binding authoring source / provenance
- Command / Observation分離の基本境界

これらをDriver専用設定へ再記述しません。

### Gate Aで答えられる必要がある問い

- このApplicationは何をbrowseするのか
- 一件のrecordとは何か
- 何をどのDatasourceからreferenceするのか
- HeaderとDetailをどのRelationで扱うのか
- どのCapabilityをcommandとして呼ぶのか
- どの業務Validationを実行するのか
- 誰がその操作を行えるのか
- どのDatasourceがread-onlyで、どこへwriteできるのか
- CapabilityをどのBindingで実現するのか
- CapabilityのBusiness ResultとExecution Errorをどう区別するのか

これらに答えるためにWeb画面やDriver専用設定を読む必要がある場合、Gate Aは未達とします。

## Gate A残条件 1 — Capability Outcome / Execution Error

関連: `ISSUE-000009` / GitHub Issue #77  
設計: [`CAPABILITY_OUTCOME_V1_DESIGN.md`](CAPABILITY_OUTCOME_V1_DESIGN.md) / ADR-0020

Authenticationでは、少なくとも次を区別する必要があります。

```text
Authenticated(subject)
Denied(reason)
ExecutionError(error)
```

credential不一致、disabled、lockは認証処理を正常に完了したBusiness Resultです。Database接続失敗やadapter異常は、認証判断そのものを完了できなかったExecution Errorです。

PR #132で、Semantic Capabilityへ名前付きOutcomeを宣言するためのCanonical基盤は実装済みです。

```tei
capability security.authenticate {
    input login std.identifier
    outcome authenticated security.authenticated_subject
    outcome denied security.authentication_denial
}
```

この段階で次を確認しています。

- Outcome local nameとpayload型を保持できる
- `outcome <name> <payload-type>`をParserから扱える
- 既存`output`と名前付きOutcomeを同時に宣言しない
- Outcome名の重複を拒否する
- payload型の参照整合性を検査できる
- 宣言順をCanonicalな意味にしない

したがって、Gate Aで残るのは「Outcomeを宣言できるか」ではなく、実際のCapability呼び出しで結果境界を保てるかです。

少なくとも次をReference / Conformanceから確認します。

- `security.authenticate`等の実Referenceでnamed Outcomeを利用できる
- `authenticated` / `denied`を正常なBusiness Resultとして扱える
- expected denialをExecution Errorへ変換しない
- Binding / adapter / Database障害をnamed Outcomeへ押し込まない
- DriverへBusiness ResultとExecution Errorの区別を渡せる

第一Web Targetでの表示方法など、Gate A以後でよい作業はDeferredします。

既存`sales.order.submit`は単一`output sales.order`で成立しているため、named Outcome導入を理由に書き換えません。

## Gate A残条件 2 — 既存認証Compatibility

関連: `ISSUE-000003` / GitHub Issue #68  
詳細: [`LEGACY_AUTH_V1_DESIGN.md`](LEGACY_AUTH_V1_DESIGN.md) / ADR-0019

既存システム固有の認証仕様を、TEI標準User schemaへ先に作り替えなくてもAuthentication / Authorization Capabilityへ接続できることを確認します。

### Referenceで保持する既存意味

少なくとも次を扱います。

- user ID
- credential reference
- disabled state
- validity period
- login failure count
- lock state
- organization / role
- system-specific rule

これらを標準化のために削ったり、Target固有コードへ再実装したりしません。

### 認証・認可の境界

AuthenticationとAuthorizationを分けます。

```text
Authentication
  誰であるかを確認する
        ↓ authenticated subject
Authorization
  そのsubjectがApplication上の操作を行えるか確認する
```

Application利用者のAuthorizationと、Plugin / Runtimeに与えるPermission Grant / Resource Leaseも別の責務です。

### Secretの扱い

password平文、秘密鍵、token等のsecret materialを次へ保存しません。

- Canonical World
- Source Index
- Durable artifact
- 通常Diagnostic
- 通常log
- Git管理するReference fixture

Referenceで必要なcredentialは、実秘密情報ではないdeterministic fixtureまたはsecret handleとして扱います。

### Gate Aへ渡すもの

- authenticated subject identity
- Authentication / Authorization Capability identity
- Application上のaction / resource
- Capability Binding / Mappingの来歴
- named Outcome / Execution Error契約
- legacy条件のうち判断に利用する意味
- secretを解決する境界

## Gate A残条件 3 — 業務Validation

関連: `ISSUE-000014` / GitHub Issue #125

Patternの構文・参照整合性を検査するValidationと、業務上のruleを評価するValidationを分けます。

PR #134で、既存Semanticだけで実証できる最初の二例を具体化しました。

### 単一属性のrule

```text
sales.order_line.quantity > 0
```

### Relation scopeを持つrule

```text
sales.order.lines
  └─ lineNumber is unique within one order
```

後者は`OrderLine`全体の一意性ではなく、一つのOrderに属するDetail集合の中での一意性です。

この二例を使い、少なくとも次の所有候補を比較します。

- Semantic rule
- Application-owned Validation
- 外部状態や処理を伴う場合のCapability / Policy等の独立単位

汎用rule engineや式言語を先に固定しません。

確認するのは次です。

- ruleの正本をどこに置くか
- Validation identityをどう追跡するか
- DriverとTargetが同じruleを再利用できるか
- Validation不成立を正常な業務結果として扱えるか
- Validation不成立とExecution Errorを区別できるか

Driver専用Goコード、Web form validator、Database CHECK constraintのいずれか一つだけを正本にはしません。

## Gate AでReadyになった項目 — Capability Binding

関連: `ISSUE-000015` / GitHub Issue #126 / PR #135

`sales.order.submit`の明示的なCustom Go BindingをReference化しました。

v1では新しい任意のBinding IDを追加せず、ApplicationとCapabilityの組をBinding identityとして扱います。

```text
order_entry / sales.order.submit
```

Driver相当の利用側は、ApplicationとCapabilityを明示してBindingを解決できます。

- 0件: Binding未定義としてerror
- 1件: 一意に解決
- 複数件: ambiguousとしてerror

file path、Go hook名、列挙順からimplementationを推測しません。

Source IndexからBinding authoring sourceへ戻ることができ、file renameでもBinding identityが変わらないことを確認しました。

同じCapabilityを異なるApplicationがBindingしても衝突せず、同じApplication + Capabilityの重複Bindingは拒否します。

この実証では第一級Binding IDは必要になりませんでした。複数implementationの同時選択やenvironment / release単位の切替が必要になった時点で再評価します。

Binding解決失敗はBusiness Resultではなく、Capabilityを実行する準備を整えられない側のerrorとして扱います。

CI #309は`verify`とPostgreSQL integrationの双方が成功しました。

## Gate A再監査

#68 / #77 / #125のEvidenceが増えたら、[`GATE_A_READINESS_AUDIT.md`](GATE_A_READINESS_AUDIT.md)を項目単位で更新します。

関連Issueが単にcloseしたかではなく、Driverへ必要な意味をRepositoryから再現できるかで判定します。

必須項目がすべてReadyになった時点で、`ISSUE-000006` / GitHub Issue #69をReady / In Progressへ移します。

## Stage 2 — Application Driver

関連: `ISSUE-000006` / GitHub Issue #69

### 現在の状態

**Gate A待ちでBlockedです。**

### 目的

UIを作る前に、Gate Aで揃えたApplication上の意味を実行し、設計の整合を確認できるようにします。

第一段階のDriverはproduction Runtimeの簡易版ではなく、設計検証環境です。

### 最小Scenario

1. `browse`で受注一覧を取得する
2. `record`で一件を扱う
3. `reference`で顧客を解決する
4. `header-detail`でOrderLineを扱う
5. 業務Validationを実行する
6. Authorizationを確認する
7. `command`で`sales.order.submit`を呼ぶ
8. Business ResultとExecution Errorを区別する
9. 必要な場合はObservationから外部状態を確認する

### Driverがしてはいけないこと

- Application ruleをDriver専用Goコードへ重複実装する
- native SQLをDriver側へコピーする
- logical datasourceを意味上ひとつへ統合する
- Target固有Presentationを必要条件にする
- Capability Bindingをfile pathやGo symbolから推測する
- Driver独自のOutcome体系を作る
- HTTP statusやGUI messageをDriver Coreの意味にする
- Driver固有状態をCanonical Worldへ逆流させる

### Gate Bへ進む条件

- ScenarioをRepositoryだけから再現できる
- Pattern / Query / Capability / Datasource / Bindingの解決結果を追跡できる
- read-only `master_data`へwriteしない
- Validation / Authorization不成立時にcommandへ進まない
- Business ResultとExecution Errorを区別できる
- CommandとObservationを必要に応じて分けられる
- 同じApplication定義を第一Targetへ渡せる

## Gate B — UIなしで業務シナリオを説明できる

Gate Bでは、画面がなくてもApplicationの主要な振る舞いを実行・検証できることを確認します。

Application上の不足をWeb固有コードで埋めないため、第一Web Targetは原則としてGate B通過後に進めます。

## Stage 3 — 第一Web Target

関連: `ISSUE-000007` / GitHub Issue #70

### 目的

Driverで確認済みのApplication上の意味をWebへProjectionし、共有できるPresentation上の意味とWeb固有実装の境界を実例から抽出します。

### 対象

- browse
- record
- reference
- header-detail
- command
- Validation feedback
- Authorizationに応じた利用可否
- Business Result
- Execution Error
- Observation

### 確認すること

- DriverとWebが同じApplication / Patternを利用する
- Web固有情報をCanonical Semanticへ逆流させない
- 標準ProjectionとCustom UIを併用できる
- Business Result / Execution Error / Observationの意味境界を保つ
- Source / Application / Targetの来歴を追跡できる

## Gate C — Editor / AIを具体化してよい

第一Targetまで成立すると、Application上の意味、Driver実行結果、Target固有差分が実資産として揃います。

この時点でEditorやAI支援が何を見せ、何を探索し、何を確認すべきかを具体化します。

## Stage 4 — Editor / AI支援

### Editor

最初から完成IDEを作りません。

少なくとも次を同じCanonical上の意味から高密度に確認できるProjectionを優先します。

- Application / Datasource
- Data / Relation
- Application Pattern
- Validation / Command
- Authorization
- Source / Diagnostic
- Driver result
- 第一Targetとの差

Editor内部状態を第二の正本にはしません。

### AI支援

最初のAI支援ではコード生成量ではなく、既存知識を利用しながら設計上の見落としを減らせるかを確認します。

- 既存Pattern / Capabilityの探索
- 既回答事項の再質問抑止
- 1:N、履歴、状態遷移、認証等の典型論点の確認提案
- AI提案と採用済みDecisionの分離
- Decision変更履歴
- 提案の無視・抑制・無効化

## Gate D — 第二Targetで検証できる

第一Web Targetで「共通」と判断した部分と「Web固有」と判断した部分を追跡できることを条件とします。

Editor / AIの完成自体はGate Dの条件にしません。

## Stage 5 — 第二Target

第二Targetは対応Target数を増やすためではなく、Target-neutral境界の圧力試験として利用します。

Desktop、API、MCP、VUI等から、Webとの差が十分に表れるTargetをその時点で選びます。

第二TargetのためにCore変更が必要になっても、それ自体を失敗とはみなしません。第一Targetの都合を誤って共通意味へ持ち上げていなかったかを検査します。

## Evidenceの残し方

各Stageでは「実装した」という記録だけで完了にしません。

少なくとも次のいずれかを残します。

- Reference source
- deterministic fixture
- Conformance test
- golden artifact
- Driver Scenario
- Target comparison
- ADR
- 未解決事項を明示した監査文書

実験結果が予想と異なる場合、その差を隠して計画へ合わせません。Product Valueに照らして意味がある差であれば、Pattern、Requirement、Milestone側を更新します。

## まだ固定しないこと

- Application Patternの永久的な標準一覧
- 第一Web Targetのframework
- Editorの最終UI framework
- 第二Targetの種類
- B2C向けの一律性能目標
- Authentication実装をDatabase / LDAP / IdP / 外部APIのどれかへ固定すること
- `std.auth.*`の標準契約
- Application単位の分散transaction一般化
- 第一級Binding IDや汎用Binding selection policy

ReferenceからEvidenceを得る前に固定しません。

## v1へ進んだと判断する条件

少なくとも次をRepositoryのEvidenceから説明できることを目安とします。

- 一つの現実的なApplicationをSemanticから第一Targetまで一続きに追跡できる
- 複数Databaseを自然に利用できる
- native SQLとCustom implementationを正本のまま利用できる
- 既存認証仕様を保持したままCapabilityへ接続できる
- Application PatternをDriverとTargetで共有できる
- UIなしでもDriverで主要シナリオを検証できる
- 第一Targetで標準ProjectionとCustom UIを併用できる
- Business ResultとExecution Errorを必要な箇所で分離できる
- CommandとObservationを必要な箇所で分離できる
- Reference / Conformanceから同じ結果を再現できる
- 何が未保証かを`docs/status.md`から判断できる

Editor / AI支援の完成、第二Target、production HAはv1の前提条件にはしません。

## 関連文書

- `product/value/TEI_PRODUCT_VALUE.md`
- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
- `plan/MASTER_PLAN.md`
- `plan/MILESTONES.md`
- `plan/PRODUCT_REQUIREMENT_GAP_AUDIT.md`
- `plan/GATE_A_READINESS_AUDIT.md`
- `plan/APPLICATION_PATTERN_V1_DESIGN.md`
- `plan/LEGACY_AUTH_V1_DESIGN.md`
- `plan/CAPABILITY_OUTCOME_V1_DESIGN.md`
- `plan/adr/0017-application-pattern-ownership.md`
- `plan/adr/0018-application-pattern-authoring-source.md`
- `plan/adr/0019-authentication-capability-boundary.md`
- `plan/adr/0020-capability-business-outcome-boundary.md`
- `plan/issue/000003-legacy-authentication-compatibility-reference.md`
- `plan/issue/000005-application-pattern-reference.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000007-first-web-target-projection.md`
- `plan/issue/000008-editor-and-ai-followup-note.md`
- `plan/issue/000009-capability-outcome-and-execution-error.md`
- `plan/issue/000013-gate-a-readiness-audit.md`
- `plan/issue/000014-business-validation-reference.md`
- `plan/issue/000015-submit-capability-binding-reference.md`
