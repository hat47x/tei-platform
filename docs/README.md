# TEI ドキュメント案内

この文書は、TEIの設計・実装・現在地を読むための案内板です。新しいProduct Requirement、Architecture判断、Gate条件、Evidence結論はここでは定義しません。各項目の意味上の正本は、それぞれのリンク先にあります。

全体像、業務モデル、実装境界、全26要求、未決事項、X-TEA原典の検討結果は、[想定仕様ガイド（HTML）](specification/index.html)でまとめて見られます。[期限なしロードマップ](../plan/product-roadmap.html)もあります。どちらも既存の正本を変更する文書ではありません。

全文書を順番に読む必要はありません。知りたいことに応じて入口を選んでください。

## クイックナビゲーション

| 知りたいこと | 最初に読む文書 | 次に読む文書 |
| --- | --- | --- |
| TEIが何を目指すか | [はじめてTEIを知る方へ](for-readers.md) | [TEIの考え方](overview.md) |
| 技術的な全体構造 | [Architecture 読み方ガイド](architecture-reading-guide.md) | [Architecture](architecture.md) |
| 機能ごとの内部設計 | [機能設計文書](../plan/design/README.md) | 対応するADR / Plan Issue |
| 業務システム設計としての境界 | [業務システム設計 読み方ガイド](business-system-design-reading-guide.md) | [Business-System Design](business-system-design.md) |
| AI / local cognition / Design Synthesis | [Design Synthesis / 認知支援 読み方ガイド](design-synthesis-reading-guide.md) | [Design Synthesis Loop Requirement Draft](../plan/DESIGN_SYNTHESIS_LOOP_REQUIREMENT_DRAFT_2026-09.md) |
| 正式なProduct Requirement | [Product Requirements 読み方ガイド](../requirements/README.md) | [TEI Product Requirements](../requirements/TEI_PRODUCT_REQUIREMENTS.md) |
| RequirementからEvidenceまでのつながり | [Upstream Acceptance Scenario Traceability](../plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md) | [TEI Planning](../plan/README.md) |
| 現在どこまで成立しているか | [現在の実装状況](status.md) | [TEI Planning](../plan/README.md) |
| 実際に動かしたい | [Getting Started](getting-started.md) | [現在の実装状況](status.md) |
| Security / Authority境界 | [Security](security.md) | [Architecture 読み方ガイド](architecture-reading-guide.md) |
| 用語・命名を確認したい | [Naming](naming.md) | [TEIの考え方](overview.md) |
| 文書を書きたい・レビューしたい | [TEI 文書品質ガイド](documentation-quality.md) | [`AGENTS.md`](../AGENTS.md) |

## 文書の役割

TEIでは、説明、設計判断、要求、計画、Evidenceを一つの長文へ集約しません。

```text
Product Value / Vision / Principles
        ↓
Product Requirements
        ↓
Product Acceptance Scenario
        ↓
Architecture / ADR / Plan Issue
        ↓
Reference / Conformance
        ↓
Evidence
        ↓
Gate / Product claim
```

一方、`docs/`はこの構造を理解・利用するための説明と技術文書を置く場所です。

| 文書群 | 主な役割 | 正本性 |
| --- | --- | --- |
| `product/` | 何を価値として守るか | Product Value / Vision / Principlesの正本 |
| `requirements/` | 何を満たすか | Product Requirementの正本 |
| `plan/` | PAS、Gate、ADR、Issue、機能設計、Evidence取得計画 | planning / decision / Gate / 機能設計の各正本 |
| `docs/architecture.md` | 実装構造と技術境界を説明する | Architectureのcurrent説明基線。Requirement第二正本ではない |
| `docs/business-system-design.md` | Data / Function / Businessをまたぐ業務設計境界を説明する | 業務システム設計のcurrent architecture baseline |
| `docs/status.md` | 現在の実装・Evidence状態を示す | 可変のcurrent snapshot |
| `reference/` / `tools/` | 設計仮説を実例・checkerで圧力試験する | Reference / executable Evidence surface |
| `docs/conformance/` | 実装と設計上の不変条件の対応を示す | 対象範囲ごとのConformance説明 |
| [`docs/audits/`](audits/README.md) | 特定時点の監査結果を残す | 監査記録。current architectureの第二正本ではない |

`architecture-reading-guide.md`、`business-system-design-reading-guide.md`、`design-synthesis-reading-guide.md`、`requirements/README.md`は、正本の内容を置き換える文書ではありません。前提知識の少ない読者が、背景・失敗例・要求群のつながりを理解してから正本やplanning文書へ進むための補助入口です。

## おすすめの読み順

### 1. 初めてTEIを読む

1. [はじめてTEIを知る方へ](for-readers.md)
2. [TEIの考え方](overview.md)
3. [現在の実装状況](status.md)

細かな型やGate手順より先に、TEIが「意味・実装・実行・観測」をなぜ分けるのかをつかみます。

### 2. Architectureを追う

1. [TEIの考え方](overview.md)
2. [Architecture 読み方ガイド](architecture-reading-guide.md)
3. [Architecture](architecture.md)
4. [業務システム設計 読み方ガイド](business-system-design-reading-guide.md)
5. [Business-System Design](business-system-design.md)
6. 必要なADR / Conformance / Reference

`architecture-reading-guide.md`で、各技術境界がどんな失敗を防ぐのかを把握してから、`architecture.md`の詳細へ進みます。Business側のactor、manual work、Handoff、actuality等が難しい場合は、`business-system-design-reading-guide.md`で一つの業務例を追ってから正本へ進みます。

`architecture.md`は主に技術境界、`business-system-design.md`は主に業務設計上の境界を扱います。両者は競合する二つのArchitectureではなく、同じTEIを異なる焦点から説明します。

### 3. Requirementから実証まで追う

1. [Product Requirements 読み方ガイド](../requirements/README.md)
2. [TEI Product Requirements](../requirements/TEI_PRODUCT_REQUIREMENTS.md)
3. [業務システム設計 要求解釈基線](../requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md)
4. [Upstream Acceptance Scenario Traceability](../plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md)
5. [TEI Planning](../plan/README.md)
6. 対応するPlan Issue / Reference / Evidence

Requirementだけから実装を直接導出せず、PAS、設計判断、Reference、Evidenceを通して成立範囲を確認します。

### 4. AI / local cognition / Design Synthesisを追う

1. [TEIの考え方](overview.md)で、TEI全体におけるAIの位置を確認します。
2. [Design Synthesis / 認知支援 読み方ガイド](design-synthesis-reading-guide.md)で、人の作業を何へ圧縮したいのかを具体例から理解します。
3. [Design Synthesis Loop Requirement Draft](../plan/DESIGN_SYNTHESIS_LOOP_REQUIREMENT_DRAFT_2026-09.md)に、上位のProduct GoalとDSR候補があります。
4. [Derived Cognitive Exploration Overview](../plan/DERIVED_COGNITIVE_EXPLORATION_OVERVIEW_2026-09.md)に、非LLM認知を検討する理由があります。
5. [Derived Cognitive Requirement Draft](../plan/DERIVED_COGNITIVE_REQUIREMENT_DRAFT_2026-09.md)に、TEIへ求める性質があります。
6. 対応するPlan Issue / Benchmark / Feature Contract

FlyHash、HDC、embedding、feature atom等の詳細から読み始めず、「既知の事実で候補空間を狭め、人は残った意味判断を行う」というProduct Goalから下へ降ります。

### 5. 現在のGateを確認する

1. [現在の実装状況](status.md)
2. [TEI Planning](../plan/README.md)
3. 対応するGate Readiness Audit / Evidence Contract / Runbook

`status.md`は現在地の入口であり、Gate判定そのものの正本は`plan/`側にあります。

### 6. 文書を書く・レビューする

1. [TEI 文書品質ガイド](documentation-quality.md)
2. [`AGENTS.md`](../AGENTS.md)
3. 対象文書の上位正本 / Reference / Evidence

TEIは扱う技術領域と業務知識が広いため、実装できる人だけが読める文書では足りません。文書側で背景、前提、用語、失敗例、設計理由を引き受け、主担当より一段浅い知識の読者でも、目的・境界・判断理由をレビューできる状態を目指します。

## 設計文書を読むときの区別

### 「設計として決めた」と「実装で確認した」を分ける

ArchitectureやRequirementに境界が書かれていても、それだけでactual Product Evidence取得済みとは扱いません。

```text
Architecture / Requirement exists
  != implementation exists
  != isolated test passes
  != actual external Evidence exists
  != Product claim satisfied
```

逆に、あるReferenceが動いたことだけで、Reference固有の形をCanonical schemaへ昇格させません。

### 「現在地」と「長く残す判断」を分ける

頻繁に変わる実装状況やGate状態は`status.md`やReadiness Auditで追います。

長く残すArchitecture判断はArchitecture文書やADRへ置きます。過去の作業経緯をcurrent baseline本文へ無制限に積み上げず、Git history、完了Issue、audit記録へ分離します。

### 「説明」と「正本」を分ける

`for-readers.md`、`overview.md`、各reading guide、`requirements/README.md`は理解のための説明・案内です。読みやすく言い換えても、Product RequirementやArchitecture boundaryの意味を独自に増減させません。

## 設計文書の基本体裁

新しい長文の設計・分析文書を追加するときは、内容に応じて冒頭で次を明らかにします。

1. 役割。Architecture、Requirement interpretation、planning、audit、experiment等のどれか。
2. 正本との関係。何の正本で、何の第二正本ではないか。
3. 対象範囲。何を扱い、何を扱わないか。
4. 現在の結論と状態。長い経緯より先に、読者が現在の判断を確認できること。
5. 根拠。Reference、Conformance、Evidence、関連ADRとIssue。
6. 非主張範囲。test PASSやReference成立から、何を一般化しないか。

さらに、内容が難しい文書ほど、「なぜ必要か」「どんな失敗を防ぐか」「具体例ではどう見えるか」を先に説明します。詳細な執筆方針は[TEI 文書品質ガイド](documentation-quality.md)を参照してください。

すべての文書へ同じ固定テンプレートを強制するものではありません。文書の種類に応じて必要な項目だけを使い、形式を守ること自体を目的にしません。

## Repository内の主な入口

- Product / planning全体: [`../README.md`](../README.md)
- Planning正本への案内: [`../plan/README.md`](../plan/README.md)
- 初心者向け: [`for-readers.md`](for-readers.md)
- 概念説明: [`overview.md`](overview.md)
- Architecture読み方: [`architecture-reading-guide.md`](architecture-reading-guide.md)
- 技術Architecture: [`architecture.md`](architecture.md)
- Product Requirements読み方: [`../requirements/README.md`](../requirements/README.md)
- Product Requirements正本: [`../requirements/TEI_PRODUCT_REQUIREMENTS.md`](../requirements/TEI_PRODUCT_REQUIREMENTS.md)
- 業務システム設計読み方: [`business-system-design-reading-guide.md`](business-system-design-reading-guide.md)
- 業務システム設計正本: [`business-system-design.md`](business-system-design.md)
- Design Synthesis / 認知支援読み方: [`design-synthesis-reading-guide.md`](design-synthesis-reading-guide.md)
- 現在地: [`status.md`](status.md)
- 実行開始: [`getting-started.md`](getting-started.md)
- Security: [`security.md`](security.md)
- Naming: [`naming.md`](naming.md)
- 文書品質: [`documentation-quality.md`](documentation-quality.md)

この案内板に現在地や設計判断そのものを複製しすぎないようにします。状態や意味が変わった場合は、まず各正本文書を更新し、この文書は読み手導線だけを追随させます。
