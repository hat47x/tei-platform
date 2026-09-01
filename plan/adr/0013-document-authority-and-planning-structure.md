# ADR-0013: 上位文書と計画文書の権威関係を分離する

- Status: Accepted
- Date: 2026-09-01

## Context

TEIは、M0.22.1以降、実行可能なvertical sliceとADR / GitHub Issueを用いて設計境界を段階的に検証してきました。

この進め方により、Source-to-Canonical Traceability、Plugin / Runtime、actual PostgreSQL、Schema Diff等の具体的な設計判断はかなり精緻になりました。

一方で、Repository全体として「TEIは何を価値として守るのか」「どの要求を満たすために、このIssueへ取り組んでいるのか」「Database以外の領域と比べて何を先に確かめるべきか」という上位の基準は、README、overview、個別Issue、過去の議論へ分散していました。

また、既存ADRは`docs/adr/`へ置かれ、作業管理はGitHub Issueが中心でした。これらは技術判断を追うには有効ですが、Product Valueから実装までの権威関係を明示するには十分ではありませんでした。

## Decision

TEIの文書を次の順に整理します。

```text
product/value
  ↓
product/vision
  ↓
product/principles
  ↓
requirements
  ↓
plan/adr / plan/issue
  ↓
docs/architecture 等の詳細設計
  ↓
implementation / test / evidence
```

`plan/MILESTONES.md`と`plan/MASTER_PLAN.md`は着手順や並走関係を示しますが、上位のValueやRequirementsを書き換える権威は持ちません。

### Product文書

- `product/value/`: 失ってはならないプロダクト価値
- `product/vision/`: 価値が実現された将来像
- `product/principles/`: 設計判断に使う原則

### Requirements

`requirements/`には、Product Value / Vision / Principlesを満たすためのProduct Requirementsを置きます。

Requirementsは「現在実装済みか」と分離します。現在地は`docs/status.md`等で扱います。

### Plan

`plan/`には、Milestone、Master Plan、ADR、Issue等、要求を実際の作業へ落とす文書を置きます。

ADR番号は4桁、Repository内Issue番号は6桁とします。

### 既存ADRの扱い

既存の`docs/adr/0001`〜`0012`は、多数の文書から参照されているため、このADRで一括移動しません。

新しい上位・横断的ADRは`plan/adr/`へ置きます。既存ADRは、リンク互換を保つ移行方法を別Issueで整理し、番号を変えず段階的に移します。

## Consequences

### 良くなる点

- 実装都合からProduct Valueを逆算するのではなく、上位意図からIssueを評価できる
- 特定領域だけが深く進み続ける偏りをMilestoneから確認できる
- 別の生成AIや開発者がRepositoryだけから判断基準を復元しやすくなる
- Product Requirementsと実装状況を分離できる
- ADR / Issueの役割が明確になる

### 負担になる点

- 文書階層が増える
- 既存の`docs/adr/`と新しい`plan/adr/`が移行期間中は併存する
- 上位文書と実装の整合を継続的に監査する必要がある

この負担は、設計が会話や個別Issueだけへ散逸することを避けるために受け入れます。

## 文書品質

上位文書、計画文書、ADR、Issueは、内容と構造を整理した後に、意味を変えず自然な日本語へ全文を整える工程を必須とします。

仕様語の正確さを理由に読みにくい直訳調へ寄せず、逆に読みやすさのために境界や条件を曖昧にもしません。
