# TEI Planning

このディレクトリには、TEI（鼎／テイ）の上位価値とProduct Requirementsを、実際の着手順、並走関係、設計判断、調査課題へつなぐ文書を置きます。

`plan/`はProduct Valueを定義する場所ではありません。何を守るかは`product/`、何を満たすかは`requirements/`、何から進めるかは`plan/`で扱います。

## 文書

- [`MASTER_PLAN.md`](MASTER_PLAN.md): 開発全体の作業軸と依存関係
- [`MILESTONES.md`](MILESTONES.md): 期限を置かないCapability MilestoneとVersion Horizon
- [`PRODUCT_REQUIREMENT_GAP_AUDIT.md`](PRODUCT_REQUIREMENT_GAP_AUDIT.md): M0.32時点のProduct Requirementsと実装Evidence / Gapの監査スナップショット
- [`V1_REFERENCE_APPLICATION_PLAN.md`](V1_REFERENCE_APPLICATION_PLAN.md): `order_entry`を使ったv1 Application実証の主線とEvidence Gate
- [`APPLICATION_PATTERN_V1_DESIGN.md`](APPLICATION_PATTERN_V1_DESIGN.md): Application Pattern Stageで成立した設計・実装基線
- [`GATE_A_READINESS_AUDIT.md`](GATE_A_READINESS_AUDIT.md): Driverへ進む前に必要なApplication上の意味の現在地
- [`LEGACY_AUTH_V1_DESIGN.md`](LEGACY_AUTH_V1_DESIGN.md): 既存認証仕様を保持してAuthentication / Authorization Capabilityへ接続する設計
- [`CAPABILITY_OUTCOME_V1_DESIGN.md`](CAPABILITY_OUTCOME_V1_DESIGN.md): CapabilityのBusiness ResultとExecution Errorを分離する設計
- [`adr/`](adr/): 長く残す設計判断
- [`issue/`](issue/): 調査、実装、文書整備などの作業単位

`PRODUCT_REQUIREMENT_GAP_AUDIT.md`はM0.32時点のスナップショットとして残します。そこにある「Application Pattern未着手」等の評価を過去へ遡って書き換えません。Application Pattern完了後の現在地は、このREADME、`V1_REFERENCE_APPLICATION_PLAN.md`、`GATE_A_READINESS_AUDIT.md`から確認します。

## 現在のv1主線

現在は、既存の`order_entry`をReference Applicationとして育てています。

```text
Stage 0  Product Baseline / Gap Audit      完了
              ↓
Stage 1  Application Pattern               完了
              ↓
Gate A   readiness監査                     進行中 / Blocked
              │
              ├─ #68   既存認証Compatibility
              ├─ #77   Capability Outcome
              ├─ #125  業務Validation
              └─ #126  Capability Binding
              ↓
Stage 2  Application Driver #69            Gate A待ち
              ↓
Stage 3  第一Web Target #70
              ↓
Stage 4  Editor / AI支援
              ↓
Stage 5  第二TargetでTarget-neutral性を検証
```

個々の機能を実装しただけでは、次の段階へ進みません。

## Application Pattern Stage — 完了

v1 Reference Applicationでは、候補としていた5種類のApplication Patternがすべて実Referenceまで成立しました。

```text
browse        data + query
record        data
reference     relation + datasource
command       capability
header-detail relation
```

同じ`order_entry`で5種類が共存し、Canonical World、Workspace Validation、Source Index、Reference Conformanceから追跡できます。

重要なのは、Patternが既存資産の意味を書き直す場所ではなく、Applicationがどの既存identityをどの意図で利用するかを表す層として成立したことです。

たとえば、次は参照先の正本へ残します。

- QueryのDatasource
- Relationのsource / target / cardinality
- Capabilityのinput / output
- Datasourceのread / read-write access
- Persistence Mapping
- Database Foreign Key
- Target固有UI

5種類がReferenceで成立したことは、5分類をTEI全体の永久的な標準語彙として確定したことを意味しません。Driverと複数TargetからのEvidenceを受けて、分類や共有範囲を見直せます。

## Gate A — 現在の主作業

Application Pattern Stageの完了とGate Aの完了は分けて扱います。

Gate Aでは、Driver #69がWeb画面やDriver専用設定から業務意味を補わず、Repository上のCanonicalな定義、Contract、Bindingだけから必要な情報を解決できることを確認します。

現在の詳細な判定は[`GATE_A_READINESS_AUDIT.md`](GATE_A_READINESS_AUDIT.md)を正とします。

### Readyな領域

- browse対象
- record対象
- reference対象RelationとDatasource
- header-detail対象Relation
- command対象Capability
- Pattern identity / Source Index
- `sales` read-write / `master` readのDatasource境界
- `sales.order.submit`の既存単一output
- Command / Observation分離の基本境界

### Blockedな領域

- #68: Authentication / Authorization Reference
- #77: Gate Aに必要なnamed Outcome / Execution Errorの実装Evidence
- #125: 業務Validationの正本と最小Reference
- #126: `sales.order.submit`のCapability Binding identity

このため、Application Driver #69は現在Blockedです。

## Gate Aで守ること

Gate Aを「関連Issueがすべてcloseしたか」だけでは判定しません。

Driverへ渡すために必要な意味がReference / Conformanceから再現できるかを基準にします。

たとえば#77には第一Web TargetへのOutcome Projectionも含まれますが、それはGate A以後へDeferredできます。一方、Authenticationの`denied`をBusiness Resultとして扱うためのnamed Outcome自体はGate Aに必要です。

逆に、IssueがcloseしていてもDriverが次を暗黙に補う必要があるならGate Aは未達です。

- 業務ValidationをDriver専用Goコードで実装する
- `sales.order.submit`のimplementationをfile pathやGo symbolから推測する
- Authorization ruleをDriver設定へ書き直す
- named OutcomeをRuntime errorへ畳む

## Gate Aの作業単位

### 既存認証Compatibility — `ISSUE-000003` / #68

既存システム固有の認証状態をTEI標準User schemaへ先に作り替えず、Authentication / Authorization Capabilityへ接続します。

Gate Aでは少なくとも、authenticated subject、Capability identity、action / resource、legacy Mapping、secret解決境界をDriverへ渡せる必要があります。

### Capability Outcome — `ISSUE-000009` / #77

ADR-0020に従い、Business ResultとExecution Errorを分離します。

既存`sales.order.submit`の単一`output`は維持しながら、Authenticationで必要な`authenticated` / `denied`等のnamed OutcomeをReference化します。

### 業務Validation — `ISSUE-000014` / #125

Patternの構文・参照整合性Validationとは別に、Applicationが利用する業務ruleの正本とidentityを実証します。

Driver専用Goコード、Web form validator、Database constraintのいずれか一つだけを業務ruleの正本にはしません。

### Capability Binding — `ISSUE-000015` / #126

`command` Patternは`sales.order.submit`というCapability identityだけを保持します。

Driverがfile pathやGo symbolを推測せず、明示的なBindingからimplementationを解決できるReferenceを作ります。

## Application Driver

[`ISSUE-000006`](issue/000006-application-driver-vertical-slice.md) / GitHub Issue #69は、Gate A通過までBlockedです。

Driverはproduction Runtimeの簡易版ではなく、Application上の意味をUIなしで実行・検証する環境とします。

Gate A通過後の最小Scenarioは次です。

```text
browse
  ↓
record
  ↓
reference
  ↓
header-detail
  ↓
business validation
  ↓
authorization
  ↓
command
  ↓
Business Result / Execution Error
  ↓
必要に応じてObservation
```

## ADR

ADRには、実装方法が変わっても残る設計判断を記録します。

ファイル名は4桁の連番を使います。

```text
plan/adr/0016-example.md
```

mainでは`docs/adr/0001`〜`0015`までが成立しています。このため、`plan/adr/`で新しく起票するADRは既存番号と衝突しない0016以降を使います。

現在の上位・横断的な判断は次です。

- ADR-0016: 文書体系と権威関係
- ADR-0017: Application Patternの初期所有境界
- ADR-0018: Application Patternの正本とfile pathからのidentity分離
- ADR-0019: Application利用者のAuthentication / AuthorizationをSemantic Capabilityとして扱う境界
- ADR-0020: CapabilityのBusiness ResultをSemantic契約に置き、Execution ErrorをInvocation境界へ残す判断

既存ADRは参照を壊さないよう、番号を維持したまま段階的に移行します。新旧の場所に同じ番号の異なる判断を置きません。

## Issue文書

Repository内に長く残したい作業定義は、6桁の連番で`plan/issue/`へ置きます。

```text
plan/issue/000001-example.md
```

GitHub Issueの`#123`とRepository内のIssue IDは別物です。

- Repository内Issue: 長く残す作業定義、背景、制約、完了条件
- GitHub Issue: 実際の進行、会話、PRとの接続

GitHub Issueだけで十分な小さい作業まで機械的に文書化する必要はありません。一方、Gateや設計境界に影響する作業はRepository側にも残します。

## 計画の更新方法

計画は日付を消化するためではなく、Product Valueに対する不確実性を減らすために更新します。

各作業では、少なくとも次を確認します。

1. どのProduct Requirementに関係するか
2. 何がまだ分かっていないか
3. 実験・設計・実装のどれで確かめるか
4. 他のworkstreamと並走できるか
5. 先に固定すると後戻りしにくい判断があるか
6. 完了時に何をEvidenceとして残すか

実装しやすい領域だけが先行し続けないよう、MilestoneではSemantic、Application、Database、Editor、AI、Target、Runtimeの進み方を横断して確認します。

## 自然な日本語の確認

計画文書、ADR、Issueは、内容を整理した後に必ず全文を読み直し、意味を変えず自然な日本語へ整えます。

機械的な箇条書きだけで背景や判断理由が失われている場合は文章として補います。逆に、長い文章の中で条件や完了基準が埋もれる場合は箇条書きへ分けます。
