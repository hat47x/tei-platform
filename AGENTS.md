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
- 文章は日本語を基本とし、Repository上の正式名称・identifier・一般的な技術用語は必要に応じて英語を併記する。英語の語を並べるだけで意味を説明したことにしない。
- 詳細な方式やfieldから書き始めず、原則として「背景 → 防ぎたい失敗 → 守る原則 → 必要能力 → 具体例 → 設計詳細 → 未解決 / 非主張範囲」の順で読めるようにする。
- 箇条書きだけで背景や判断理由が失われる場合は文章で補う。Requirement、ADR、Security境界等では「なぜ必要か」「満たさないと何が起きるか」「何をまだ保証しないか」まで説明する。
- TEIの要求技術・業務知識が高いことを、説明不足の理由にしない。主担当より一段浅い知識の読者でも目的・境界・判断理由をレビューできることを目標にする。
- 専門語・TEI固有語は、読者が意味を推測しなくて済むよう、初出または近い入口で短い日本語説明を添える。
- 抽象境界を説明するときは、可能な範囲でReferenceに沿った具体例、失敗例、因果を示す図を使う。ただし例をRequirementやCanonical meaningへ昇格させない。
- 文書の入口では、役割、正本との関係、対象範囲、現在の状態、非主張範囲を必要に応じて明示する。
- 詳細な執筆・レビュー指針は[`docs/documentation-quality.md`](docs/documentation-quality.md)を参照する。

## 開発フロー

- 単独開発のため、branch / PR / 人的reviewの手続きは置かない。`main`へ直接commitする。
- remote branchを作らない。pushの前に`git pull --rebase origin main`し、mainへforce pushしない。
- 詳細は[`CONTRIBUTING.md`](CONTRIBUTING.md)を参照する。

コードの検証手順も[`CONTRIBUTING.md`](CONTRIBUTING.md)を参照します。
