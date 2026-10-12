# 設計判断記録（ADR）

ここには、TEIの設計に長く影響する判断を正式な記録として保存します。

Issueでは「何を行うか」を管理し、ADRでは「なぜその方針を採ったか」を記録します。実装や調査の完了によってIssueが閉じられても、判断に至った背景を後から確認できるようにします。

## 運用方法

- 新しい設計判断には4桁の連番を付けます。
- 採用済みのADRは、後から都合よく書き換えません。判断が変わった場合は新しいADRを追加し、以前のADRを「置き換え済み」とします。
- 作業手順、軽微なコード整理、調査メモはADRとせず、Issueや通常の文書で扱います。
- 複数の生成AIが並行して作業する場合も、既存のADRに反する変更を独断で進めません。判断の変更が必要なら、まずIssueで論点を扱います。
- ADRでは形式を整えることより、判断の背景・内容・影響を自然な日本語で理解できることを優先します。

## 状態

主に次の状態を使います。

- `提案中`：まだ採用を決めていない
- `採用`：現在の設計判断として有効
- `置き換え済み`：後続のADRによって判断が更新された
- `却下`：検討したが採用しなかった

## 判断履歴

- [ADR-0001: 設計判断と実行作業をADRとIssueに分ける](0001-adr-and-issue-workflow.md)
- ADR-0002: M0.22.1を最初の公開設計チェックポイントとする
- [ADR-0003: 参照作業領域の基準出力と観測用の試験データをリポジトリに保存する](0003-reference-fixtures.md)
- [ADR-0004: 情報源から意味の正本への追跡関係を正本の全体モデルと分ける](0004-source-to-canonical-traceability.md)
- [ADR-0005: 監督側の権限と実行基盤で古い担当を排除する権限を分ける](0005-supervisor-and-runtime-fencing-authority.md)
- [ADR-0006: プロセスの実行命令を実行ファイルと引数列に分ける](0006-process-command-and-argv.md)
- [ADR-0007: 実データベースのドライバーの組立てを提供実装の中核から分ける](0007-database-driver-assembly.md)
- [ADR-0008: 期待する外部キーを永続化への対応付けで明示し、構造から差分を比較する](0008-persistence-foreign-key-identity.md)
- [ADR-0009: UNIQUE制約と一意索引を別々の物理設計上の意図として扱う](0009-unique-constraint-and-unique-index.md)
- [ADR-0010: 高度な索引の構造上の意図と実際の運用状態を分ける](0010-advanced-index-semantics.md)
- [ADR-0011: 高度な索引の定義をデータベースが管理する永続化上の意図として記述する](0011-advanced-index-authoring.md)
- [ADR-0012: ProviderExpressionの記述原文と提供実装が正規化した表現を分ける](0012-provider-expression-representation.md)
- [ADR-0013: 期待する物理的な実現方式の提供元別方針を実行時のProvider SPIと分ける](0013-offline-physical-realization-policy.md)
- [ADR-0014: 提供実装固有の索引識別子は、根拠に基づいて判断できる範囲だけを正規化する](0014-provider-index-identifier-comparison.md)
- [ADR-0015: マイグレーションSQLの文法責務をデータベースプロバイダー側に置く](0015-provider-owned-migration-sql-rendering.md)
- [ADR-0016: 上位文書と計画文書の権威関係を分離する](0016-document-authority-and-planning-structure.md)
- [ADR-0017: アプリケーションの構成パターンを、そのアプリケーションが管理する意味として検証する](0017-application-pattern-ownership.md)
- [ADR-0018: アプリケーションの構成パターンの正本を、そのアプリケーション配下の独立した定義ファイルに置く](0018-application-pattern-authoring-source.md)
- [ADR-0019: アプリケーション利用者の認証・認可を意味上の機能として扱う](0019-authentication-capability-boundary.md)
- [ADR-0020: 機能の業務上の結果と実行エラーを分ける](0020-capability-business-outcome-boundary.md)
- [ADR-0021: キーを使わない見出し・明細間の関連をたどる処理は構成パターン外の実現方式で扱う](0021-header-detail-relation-traversal-realization.md)
- [ADR-0022: AIによる変更提案と意味の正本の受理境界を分ける](0022-ai-proposal-canonical-acceptance-boundary.md)
- [ADR-0023: 命令後の観測を独立した明示的な読取りの境界として扱う](0023-command-observation-explicit-read-boundary.md)
- [ADR-0024: 業務上の状態遷移の正本を意味の定義に置き、状態の識別子と物理コードを分ける](0024-business-lifecycle-ownership.md)（提案中）
- [ADR-0025: 遷移の評価順序を固定し、成立しなかった遷移を業務結果として返す](0025-transition-evaluation-and-conflict-result.md)（提案中）
- [ADR-0026: 分割テーブルの子表を提供実装が管理する実現方式とし、共通のスキーマ差分検査から分ける](0026-partition-child-provider-realization-boundary.md)
- [ADR-0027: 分割テーブルを伴うPostgreSQLのDropTableは提供実装の実行前条件を満たさなければ拒否する](0027-postgres-partition-drop-execution-precondition.md)
- [ADR-0028: TEIの成果物を人間・構築に携わるAI・業務を扱うAIの共通知識基盤として扱う](0028-human-ai-builder-agent-shared-knowledge-plane.md)
- [ADR-0029: アプリケーションが評価する検証規則を、正本の`uses validation`宣言から選ぶ](0029-application-validation-selection-from-canonical-declaration.md)
- [ADR-0030: 生成AIが扱う情報を文脈付きの認識状態として保持する](0030-context-scoped-epistemic-knowledge.md)
- [ADR-0032: Cordis型の構成中核をTEIの新しいプラグイン基盤とする](0032-composition-kernel-replacement.md)（目標方針として採用。本番向け実装は移行中）
- [ADR-0033: コアとプラグインをサブプロジェクトとして管理する](0033-core-and-plugin-subprojects.md)（目標構成として採用。ソース移動は未実施）
