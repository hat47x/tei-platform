# ADR-0016: 上位文書と計画文書の権威関係を分離する

- 状態: 採用
- 日付: 2026-09-01

## 背景

TEIはM0.22.1以降、実行可能なvertical sliceとADR / GitHub Issueを使い、設計境界を段階的に検証してきました。その結果、Source-to-Canonical Traceability、Plugin / Runtime、actual PostgreSQL、Schema Diffなどの技術判断はかなり具体化しています。

一方、Repository全体で見ると、「TEIは何を価値として守るのか」「どのProduct Requirementを満たすために現在のIssueへ取り組むのか」「特定領域だけが深く進んでいないか」を判断する上位の基準は、README、overview、個別Issue、過去の議論へ分散していました。

また、既存ADRは`docs/adr/`、進行管理はGitHub Issueを中心としており、個々の技術判断を追うには十分でも、Product Valueから実装までの権威関係をRepositoryだけから復元するには弱い状態でした。

## 判断

TEIの文書は、次の順で権威関係を整理します。

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

`plan/MILESTONES.md`と`plan/MASTER_PLAN.md`は、着手順や並走関係を示すための文書です。上位のProduct ValueやProduct Requirementsを書き換える権威は持ちません。

### Product文書

- `product/value/`: 失ってはならないプロダクト価値
- `product/vision/`: 価値が実現された将来像
- `product/principles/`: 設計判断に使う原則

### Requirements

`requirements/`には、Product Value / Vision / Principlesを満たすためのProduct Requirementsを置きます。

Requirementsは、現在実装済みかどうかとは分離します。実装状況は`docs/status.md`などで扱います。

### Plan

`plan/`には、Milestone、Master Plan、ADR、Issueなど、要求を具体的な判断や作業へ落とす文書を置きます。

ADR番号は4桁、Repository内Issue番号は6桁とします。

## 既存ADRの扱い

このADRを作成した後もmain側では技術検証が進み、`docs/adr/`にはADR-0015まで成立しました。そのため、新しい`plan/adr/`の番号は既存履歴と衝突しないADR-0016から継続します。

既存のADR-0001〜0015は多数の文書から参照されているため、このADRの導入と同時には移動しません。リンク互換を保つ方法を別Issueで整理し、番号を変えず段階的に`plan/adr/`へ移します。

移行期間中に`docs/adr/`と`plan/adr/`が併存しても、同じADR番号に異なる判断を置くことはしません。

## 影響

### 良くなる点

- 実装都合からProduct Valueを逆算せず、上位の意図からIssueを評価できる
- 特定領域だけが深く進み続ける偏りをMilestoneから確認できる
- 別の開発者や生成AIがRepositoryだけから判断基準を復元しやすくなる
- Product Requirementsと現在の実装状況を分けて扱える
- ADRとIssueの役割が明確になる

### 増える負担

- 文書階層が増える
- ADR移行期間中は`docs/adr/`と`plan/adr/`が併存する
- 上位文書と実装の整合を継続的に監査する必要がある

この負担は、設計判断が会話や個別Issueへ散逸することを避けるために受け入れます。

## 文書品質

Product文書、計画文書、ADR、Issueは、内容と構造を整理した後、意味を変えず自然な日本語へ全文を整える工程を必須とします。

仕様語の正確さを理由に直訳調へ寄せず、逆に読みやすさのために境界や条件を曖昧にもしません。
