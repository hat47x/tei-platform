# ADR（Architecture Decision Record）

このディレクトリは、TEIの設計に長く影響する判断を記録する正規の保存場所です。

Issueは「何を行うか」を管理する場所、ADRは「なぜその方針を採ったか」を残す場所です。実装や調査が終わってIssueが閉じても、設計判断の背景を後から追えるようにします。

## 運用方法

- 新しい設計判断には4桁の連番を付けます。
- 採用済みのADRを後から都合よく書き換えず、判断が変わった場合は新しいADRを追加し、旧ADRを「置き換え済み」とします。
- 単なる作業手順、軽微なリファクタリング、調査メモはADRにせず、Issueまたは通常の文書で扱います。
- 生成AIが並行して作業する場合も、既存ADRと矛盾する変更を独断で進めません。変更が必要なら、まず論点をIssueへ戻します。
- ADRは形式を整えることより、背景、判断、影響が自然な日本語で読めることを優先します。

## 状態

主に次の状態を使います。

- `提案中`: まだ採用を決めていない
- `採用`: 現在の設計判断として有効
- `置き換え済み`: 後続ADRによって判断が更新された
- `却下`: 検討したが採用しなかった

## 判断履歴

- [ADR-0001: 設計判断と実行作業をADRとIssueに分ける](0001-adr-and-issue-workflow.md)
- [ADR-0002: M0.22.1を最初の公開設計チェックポイントとする](0002-m0221-public-checkpoint.md)
- [ADR-0003: Reference Workspaceのgolden出力と観測fixtureをRepositoryに保持する](0003-reference-fixtures.md)
- [ADR-0004: Source-to-Canonical TraceabilityをCanonical Worldから分離する](0004-source-to-canonical-traceability.md)
- [ADR-0005: Supervisor authorityとruntime fencing authorityを分離する](0005-supervisor-and-runtime-fencing-authority.md)
- [ADR-0006: process commandをexecutableとargvへ分離する](0006-process-command-and-argv.md)
- [ADR-0007: actual database driverの組み立てをProvider Coreから分離する](0007-database-driver-assembly.md)
- [ADR-0008: Desired Foreign KeyをPersistence Mappingで明示し、構造で差分比較する](0008-persistence-foreign-key-identity.md)
- [ADR-0009: UNIQUE constraintとunique indexを別のphysical intentとして扱う](0009-unique-constraint-and-unique-index.md)
- [ADR-0010: Advanced Indexを構造的intentとactual lifecycle stateに分ける](0010-advanced-index-semantics.md)
- [ADR-0011: Advanced Index authoringをDatabase-owned Persistence intentとして記述する](0011-advanced-index-authoring.md)
- [ADR-0012: ProviderExpressionのauthor textとprovider-normalized representationを分離する](0012-provider-expression-representation.md)
- [ADR-0013: Desired Physical realizationのprovider policyをruntime Provider SPIから分離する](0013-offline-physical-realization-policy.md)
- [ADR-0014: プロバイダー固有のIndex識別子は、根拠を持って判定できる範囲だけを正規化する](0014-provider-index-identifier-comparison.md)
- [ADR-0015: マイグレーションSQLの文法責務をデータベースプロバイダー側に置く](0015-provider-owned-migration-sql-rendering.md)
- [ADR-0016: 上位文書と計画文書の権威関係を分離する](0016-document-authority-and-planning-structure.md)
- [ADR-0017: Application PatternをApplicationが所有する意味として検証する](0017-application-pattern-ownership.md)
- [ADR-0018: Application Patternの正本をApplication配下の独立した定義ファイルに置く](0018-application-pattern-authoring-source.md)
- [ADR-0019: Application利用者の認証・認可をSemantic Capabilityとして扱う](0019-authentication-capability-boundary.md)
- [ADR-0020: Capabilityの業務Outcomeと実行エラーを分離する](0020-capability-business-outcome-boundary.md)
- [ADR-0021: keylessなheader-detail Relation traversalはPattern外のrealizationで実現する](0021-header-detail-relation-traversal-realization.md)
- [ADR-0022: AI ProposalとCanonical受入境界を分離する](0022-ai-proposal-canonical-acceptance-boundary.md)
- [ADR-0023: Command後のObservationは明示的なread境界として分離する](0023-command-observation-explicit-read-boundary.md)
- [ADR-0024: 業務Lifecycleの正本をSemanticへ置き、状態identityと物理コードを分離する](0024-business-lifecycle-ownership.md)（提案中）
- [ADR-0025: 遷移の評価順序を固定し、成立しなかった遷移を業務結果として返す](0025-transition-evaluation-and-conflict-result.md)（提案中）
- [ADR-0026: Partition childをprovider-owned realizationとしてgeneric Schema Driftから分離する](0026-partition-child-provider-realization-boundary.md)
- [ADR-0027: Partition treeを伴うPostgreSQL DropTableをprovider execution preconditionでfail closedする](0027-postgres-partition-drop-execution-precondition.md)
- [ADR-0028: TEI成果物を人間・AI Builder・AI Business Actorの共通知識面として扱う](0028-human-ai-builder-agent-shared-knowledge-plane.md)
- [ADR-0029: Applicationが評価するValidationの選択を、Canonicalな`uses validation`宣言から導く](0029-application-validation-selection-from-canonical-declaration.md)

## `docs/adr/`について

ADR-0001〜0015は、以前`docs/adr/`を正本としていました。旧URLや過去文書からの参照を壊さないため、現在も同じパスに互換案内を残しています。

設計判断の本文はこの`plan/adr/`だけを正本とし、`docs/adr/`側へ本文を複製しません。新しいADRも`plan/adr/`にのみ追加します。
