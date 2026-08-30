# ADR（Architecture Decision Record）

このディレクトリには、TEIの設計に長く影響する判断を記録します。

Issueは「何を行うか」を管理する場所、ADRは「なぜその方針を採ったか」を残す場所です。実装や調査が終わってIssueが閉じても、設計判断の背景は後から参照できるようにします。

## 運用方法

- 新しい設計判断には、連番のADRを追加します。
- 採用済みのADRを後から都合よく書き換えるのではなく、判断が変わった場合は新しいADRを追加し、旧ADRを「置き換え済み」とします。
- 単なる作業手順、軽微なリファクタリング、調査メモはADRにしません。Issueまたは通常の文書で扱います。
- 生成AIが並行して作業する場合も、既存ADRと矛盾する変更を独断で進めません。変更が必要なら、まず論点をIssueへ戻します。
- ADRは形式を整えることより、背景、判断、影響が自然な日本語で読めることを優先します。

## 状態

主に次の状態を使います。

- `提案中`: まだ採用を決めていない
- `採用`: 現在の設計判断として有効
- `置き換え済み`: 後続ADRによって判断が更新された
- `却下`: 検討したが採用しなかった

## 一覧

- [ADR-0001: 設計判断と実行作業をADRとIssueに分ける](0001-adr-and-issue-workflow.md)
- [ADR-0002: M0.22.1を最初の公開設計チェックポイントとする](0002-m0221-public-checkpoint.md)
- [ADR-0003: Reference Workspaceのgolden出力と観測fixtureをRepositoryに保持する](0003-reference-fixtures.md)
- [ADR-0004: Source-to-Canonical TraceabilityをCanonical Worldから分離する](0004-source-to-canonical-traceability.md)
- [ADR-0005: Supervisor authorityとruntime fencing authorityを分離する](0005-supervisor-and-runtime-fencing-authority.md)
- [ADR-0006: process commandをexecutableとargvへ分離する](0006-process-command-and-argv.md)
- [ADR-0007: actual database driverの組み立てをProvider Coreから分離する](0007-database-driver-assembly.md)

## 関連する進行管理

M0.22.1公開後の初期分担はGitHub Issue #1で完了しています。M0.23のSource-to-Canonical TraceabilityはIssue #14〜#19で完了しました。Runtime takeover時のauthority境界はIssue #9とADR-0005で整理済みです。M0.24の実process Plugin runtimeはIssue #31で完了し、M0.25のprocess command / argv境界はIssue #33とADR-0006で完了しました。M0.26のactual PostgreSQL integrationはIssue #35とADR-0007で追跡します。
