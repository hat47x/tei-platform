# AGENTS.md

TEI Repositoryの文書構成と管理ルールです。ここを文書配置の正本とします。

## フォルダ構成

```text
product/          守る価値・原則・Vision
requirements/     満たすべきProduct Requirement（RQ-001〜）
plan/             計画管理（Master Plan、Milestone、Gateの監査・契約・Runbook）
  design/         機能ごとの設計文書
    mockups/      画面モック（単体で開けるHTML。正本ではない）
  issue/          作業定義と要件詳細化（000001-*.md）
  adr/            長く残る設計判断（0001-*.md）
docs/             利用者向け文書（overview / status / architecture / naming / security）
  conformance/    意味境界のConformance記述
  adr/            旧URL互換の案内のみ。本文は plan/adr が正本
  audits/         過去の監査記録
reference/        Reference Workspaceと観測fixture
internal/         Core実装
cmd/ tools/       CLIと補助tool
examples/ bootstrap/
```

## 文書ルール

- 正本を一つにする。価値は`product/`、要求は`requirements/`、進行は`plan/issue/`、判断は`plan/adr/`、機能設計は`plan/design/`。同じ内容を複数の場所へ複製しない。
- `plan/`直下は計画管理文書だけに限定する。機能ごとの設計文書は`plan/design/`へ置く。小さい機能では新しい設計文書を作らず、Issue本文へ書く。
- ADRは4桁連番、Issueは6桁連番。ファイル名は`0001-short-description.md`形式。
- ADRの状態は `提案中` / `採用` / `置き換え済み` / `却下`。採用済みADRを書き換えず、判断が変わったら新しいADRを追加する。
- Issueの状態は `Proposed` / `Ready` / `In Progress` / `Blocked` / `Done` / `Superseded`。
- 未採用の文書は「評価待ち」と明記し、実装やReference変更の根拠にしない。
- 実装済みの機能と設計検証段階を同じ表現で書かない。実装状況は`docs/status.md`が正本。
- 各作業では、どのProduct Requirementの不確実性を減らすのかを書く。
- Canonicalな意味とEvidenceを混ぜない。syntheticな成功を実Provider・外部資産・本番運用の保証へ昇格させない。
- 文章は日本語。箇条書きだけで背景や判断理由が失われる場合は文章で補う。

コードの検証手順は[`CONTRIBUTING.md`](CONTRIBUTING.md)を参照します。
