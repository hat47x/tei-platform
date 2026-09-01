# TEI Planning

このディレクトリには、TEI（鼎／テイ）の上位価値とProduct Requirementsを、実際の着手順、並走関係、設計判断、調査課題へつなぐ文書を置きます。

`plan/`はProduct Valueを定義する場所ではありません。何を守るかは`product/`、何を満たすかは`requirements/`、何から進めるかは`plan/`で扱います。

## 文書

- [`MASTER_PLAN.md`](MASTER_PLAN.md): 開発全体の作業軸と依存関係
- [`MILESTONES.md`](MILESTONES.md): 期限を置かないCapability MilestoneとVersion Horizon
- [`PRODUCT_REQUIREMENT_GAP_AUDIT.md`](PRODUCT_REQUIREMENT_GAP_AUDIT.md): Product Requirementsと現行実装のEvidence / Gap
- [`V1_REFERENCE_APPLICATION_PLAN.md`](V1_REFERENCE_APPLICATION_PLAN.md): `order_entry`を使ったv1 Application実証の主線とEvidence Gate
- [`APPLICATION_PATTERN_V1_DESIGN.md`](APPLICATION_PATTERN_V1_DESIGN.md): Application Patternの設計・実装基線と実Referenceから得た判断
- [`LEGACY_AUTH_V1_DESIGN.md`](LEGACY_AUTH_V1_DESIGN.md): 既存認証仕様を保持してAuthentication / Authorization Capabilityへ接続する実装前設計
- [`CAPABILITY_OUTCOME_V1_DESIGN.md`](CAPABILITY_OUTCOME_V1_DESIGN.md): Capabilityの業務OutcomeとExecution Errorを分離するv1実装前設計
- [`adr/`](adr/): 長く残す設計判断
- [`issue/`](issue/): 調査、実装、文書整備などの作業単位

## 現在のv1主線

現在は、既存の`order_entry`をReference Applicationとして育てます。

```text
Application Pattern ────────┐
      完了                  │
        │             既存認証Compatibility
        │                    │
        └────── Gate A ──────┘
                  ↓
          Application Driver
                  ↓
            第一Web Target
                  ↓
             Editor / AI支援
                  ↓
      第二Targetで境界を圧力試験
```

Application Pattern側のStage 1実証は完了しました。

`order_entry`では、候補としていた次の5種類が実Referenceまで成立しています。

```text
browse
record
reference
command
header-detail
```

これは5分類をTEI全体の永久的な標準語彙として確定したという意味ではありません。具体的な業務Applicationで、Target固有UIやPersistenceから切り離したApplication上の意図として成立することを確認した段階です。

詳細は[`APPLICATION_PATTERN_V1_DESIGN.md`](APPLICATION_PATTERN_V1_DESIGN.md)と[`ISSUE-000005`](issue/000005-application-pattern-reference.md)を参照します。

`header-detail`の前提となる`sales.OrderLine`と`Order 1:N OrderLine`は[`ISSUE-000011`](issue/000011-order-line-semantic-reference.md)でSemanticとして独立して追加し、[`ISSUE-000012`](issue/000012-header-detail-application-pattern.md)でRelation-onlyのApplication Patternとして実証しました。

Semantic Relationを追加しても、Persistence Mapping、Database Foreign Key、Database Contract、Migrationを暗黙生成していません。Application PatternもWebの画面テンプレートへ固定していません。

現在の次の主線はGate Aです。

Gate AではApplication PatternのEvidenceに加えて、既存認証Compatibility、Validation、Capability Binding、Business Result / Execution Error境界を合流させ、DriverがApplication上の意味をUI固有設定やlegacy schemaの物理構造から推測せず受け取れることを確認します。

既存認証Compatibilityは[`LEGACY_AUTH_V1_DESIGN.md`](LEGACY_AUTH_V1_DESIGN.md) / `ISSUE-000003` / GitHub Issue #68で進めます。

Capabilityの業務上の結果と実行エラーの区別は、[`CAPABILITY_OUTCOME_V1_DESIGN.md`](CAPABILITY_OUTCOME_V1_DESIGN.md)と[`ISSUE-000009`](issue/000009-capability-outcome-and-execution-error.md)を共通境界として利用します。

Gate AをEvidenceで通過した後、`ISSUE-000006` / GitHub Issue #69のApplication Driverへ進みます。

個々の機能が実装されたことだけでは、次の段階へ進みません。

[`V1_REFERENCE_APPLICATION_PLAN.md`](V1_REFERENCE_APPLICATION_PLAN.md)では、Application上の意味をDriverへ渡せること、UIなしで主要シナリオを再現できること、第一Web Targetから共通意味とWeb固有差を分離できることをEvidence Gateとして定めています。

このGateによって、WebやEditorの実装都合から上位の意味モデルを先回りして固定することを避けます。

## ADR

ADRには、実装方法が変わっても残る設計判断を記録します。

ファイル名は4桁の連番を使います。

```text
plan/adr/0016-example.md
```

mainでは`docs/adr/0001`〜`0015`までが既に成立しています。このため、`plan/adr/`で新しく起票するADRは、既存番号と衝突しない0016以降を使います。

既存ADRは多くの文書から参照されているため、一度に移動してリンクを壊しません。番号を維持したまま段階的に`plan/adr/`へ移し、新旧の場所に同じ番号の異なる判断を置かないことを原則とします。

現在の上位・横断的な判断は、次の順で置いています。

- ADR-0016: 文書体系と権威関係
- ADR-0017: Application Patternの初期所有境界
- ADR-0018: Application Patternの正本とfile pathからのidentity分離
- ADR-0019: Application利用者のAuthentication / AuthorizationをSemantic Capabilityとして扱う境界
- ADR-0020: Capabilityの業務OutcomeをSemantic契約に置き、Execution ErrorをInvocation境界へ残す判断

## Issue文書

Repository内に長く残したい作業定義は、6桁の連番で`plan/issue/`へ置きます。

```text
plan/issue/000001-example.md
```

GitHub Issueの`#123`とRepository内のIssue IDは別物です。

- Repository内Issue: 長く残す作業定義、背景、制約、完了条件
- GitHub Issue: 実際の進行、会話、PRとの接続

必要な場合はRepository内IssueからGitHub Issue番号を参照します。GitHub Issueだけで十分な小さな作業まで、機械的に文書化する必要はありません。

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

機械的な箇条書きだけで背景や判断理由が失われている場合は文章として補います。逆に、長い文章の中で条件や完了基準が埋もれる場合は、箇条書きへ分けます。
