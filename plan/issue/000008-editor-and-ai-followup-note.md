# ISSUE-000008: Editor / AI支援の着手条件を整理する

- Status: Blocked

## 背景

Product Requirements Gap Auditでは、EditorとAI支援が大きな未実証領域であることを確認しました。

一方で、EditorやAIを先に作ると、まだ実証されていないApplication PatternやDriverの意味をUI都合で固定する危険があります。

このIssueは実装Issueではなく、**いつEditor / AI支援へ着手してよいか**を明示するための待機条件を記録します。

## 関連するProduct Requirement

- RQ-010 Editor
- RQ-011 Projection
- RQ-015 AI knowledge-first workflow
- RQ-016 AI suggestion / decision UX
- RQ-017 二重管理の最小化

## 着手条件

少なくとも次が成立するまで、Editor / AIの詳細interfaceを固定しません。

1. `ISSUE-000005`でApplication Patternのownershipが確認される
2. `ISSUE-000006`でDriverが同じApplication meaningを実行できる
3. `ISSUE-000007`で第一TargetへProjectionしたときの共通意味とTarget固有差が見える

## 着手時に最初に扱う範囲

条件が揃った後も、完成IDEや自律AI agentから始めません。

最初のEditorでは、v1 `order_entry`について次を高密度に確認できるProjectionを対象とします。

- Application / Datasource
- Data / Relation
- Pattern
- Validation / Command
- Source / Diagnostic
- Driver result

AI支援では、その同じ資産を使って次を最小対象とします。

- 既存Patternの探索
- 既回答事項の再質問抑止
- 不足している典型確認事項の提示
- 提案と採用済みDecisionの分離
- Decisionの変更履歴

## 非目標

- 条件成立前のUI framework選定
- chat-first IDEの固定
- AIによるauthoritative sourceの無確認自動編集
- Application meaningとEditor内部状態の二重正本化

## 完了条件

`ISSUE-000005`〜`000007`の成果を受け、Editor / AI支援へ着手するための意味境界が揃った時点で、本Issueを具体的な実装Issueへ分解します。
