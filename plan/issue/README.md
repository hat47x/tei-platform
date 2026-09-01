# TEI Planning Issues

このディレクトリには、TEI（鼎／テイ）の開発で長く残す価値がある作業定義を置きます。

GitHub Issueは日々の進行やPRとの接続に向いていますが、背景、制約、完了条件、上位要求との関係まで含む作業定義は、Repository側にも残しておく方が別の開発者や生成AIから再利用しやすくなります。

## 番号

Repository内Issueは6桁の連番を使います。

```text
000001
000002
000003
```

ファイル名は次の形式を基本とします。

```text
000001-short-description.md
```

GitHub Issueの番号とは別に管理します。

## 何をRepository内Issueにするか

次のような作業は、Repository内Issueとして残す価値があります。

- 複数のPRや実験にまたがる作業
- Product Requirementとの関係を長く残したい作業
- 設計判断を行う前の調査
- 複数workstreamへ影響するGap Audit
- 別の生成AIへ委ねても背景を失いたくない作業
- 完了条件をRepository上のEvidenceで確認したい作業

一方、誤字修正や小さい局所改修まで機械的にRepository内Issueへ昇格させる必要はありません。

## 推奨する構成

各Issueでは、必要に応じて次を記述します。

- 背景
- 関連するProduct Requirement
- 現在分かっていること
- まだ分からないこと
- 制約
- 作業内容
- 非目標
- 完了条件
- 関連ADR / GitHub Issue / PR

見出しを埋めること自体を目的にはしません。小さいIssueでは不要な項目を省略して構いません。

## 状態

文書冒頭に次のいずれかを記載します。

- Proposed
- Ready
- In Progress
- Blocked
- Done
- Superseded

状態は進捗率ではなく、次に何をしてよいかを示します。
