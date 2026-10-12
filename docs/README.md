# TEI ドキュメント案内

この文書は、TEIの設計、実装、開発状況を調べるための案内です。製品要求、設計判断、受入条件、検証結果はここでは新たに定義しません。それぞれの正本はリンク先を参照してください。

全体像は[TEIの考え方](overview.md)、実装の境界は[全体構成](architecture.md)、業務モデルは[業務システム設計](business-system-design.md)で確認できます。正式な要求は[製品要求](../requirements/TEI_PRODUCT_REQUIREMENTS.md)、開発の中心となる計画は[TEIの全体計画](../plan/MASTER_PLAN.md)に記載しています。

すべての文書を順番に読む必要はありません。目的に応じて入口を選んでください。リポジトリ全体の入口は[README](../README.md)です。

## 知りたいことから探す

| 知りたいこと | 最初に読む文書 | 次に読む文書 |
| --- | --- | --- |
| TEIが何を目指すか | [はじめてTEIを知る方へ](for-readers.md) | [TEIの考え方](overview.md) |
| 技術的な全体構造 | [全体構成の読み方](architecture-reading-guide.md) | [全体構成](architecture.md) |
| 機能の実装と検証 | [全体構成](architecture.md) | [準拠確認と参照用資料](../reference/README.md) |
| 業務システム設計の境界 | [業務システム設計の読み方](business-system-design-reading-guide.md) | [業務システム設計](business-system-design.md) |
| AIによる認知支援と設計案の統合 | [設計案の統合と認知支援の読み方](design-synthesis-reading-guide.md) | [TEIの考え方](overview.md) |
| 正式な製品要求 | [製品要求の読み方](../requirements/README.md) | [TEI製品要求](../requirements/TEI_PRODUCT_REQUIREMENTS.md) |
| 要求から検証根拠までのつながり | [上位の受け入れシナリオとの追跡関係](../plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md) | [TEIの計画](../plan/README.md) |
| 現在どこまで動くか | [現在の実装状況](status.md) | [TEIの計画](../plan/README.md) |
| 実際に動かす | [TEIを動かす](getting-started.md) | [現在の実装状況](status.md) |
| 安全性と権限の境界 | [安全性](security.md) | [全体構成の読み方](architecture-reading-guide.md) |
| 用語と命名 | [TEIの名称と共通用語](naming.md) | [TEIの考え方](overview.md) |
| 文書の執筆と確認 | [TEI文書品質ガイド](documentation-quality.md) | [`AGENTS.md`](../AGENTS.md) |

## 文書の役割

TEIでは、説明、要求、設計判断、計画、検証の根拠を一つの長文にまとめず、役割ごとに管理します。

```text
製品の価値・将来像・設計原則
        ↓
製品要求
        ↓
製品の受け入れシナリオ
        ↓
全体構成・設計判断・計画上の課題
        ↓
参照実装・準拠確認
        ↓
検証根拠
        ↓
判定段階・製品についての主張
```

`docs/`には、利用者や運用者が現行のTEIを理解し使うための説明と、実装に結び付いた構成・準拠確認を置きます。採用前の案、複数作業にまたがる設計、実施計画、受入条件は`plan/`で管理します。

| 文書群 | 主な役割 | 正本性 |
| --- | --- | --- |
| `product/` | 何を価値として守るか | 価値、将来像、原則の正本 |
| `requirements/` | 何を満たすか | 製品要求の正本 |
| `plan/` | 受け入れシナリオ、判定段階、設計判断、課題、機能設計、検証根拠の取得計画 | 計画、決定、判定段階、機能設計それぞれの正本 |
| [`tools/README.md`](../tools/README.md) | 開発用ツールの一覧 | 実行方法と用途の入口 |
| `docs/architecture.md` | 実装構造と技術上の境界を説明する | 現在の構成を説明する基準。要求の第二の正本ではない |
| `docs/business-system-design.md` | データ、機能、業務をまたぐ設計上の境界を説明する | 業務システム設計の現在の基準文書 |
| `docs/status.md` | 現在の実装と検証根拠の状態を示す | 更新される現状記録 |
| `reference/` / `tools/` | 設計上の仮説を実例と確認用ツールで検証する | 参照実装と実行可能な検証根拠 |
| [`docs/conformance/README.md`](conformance/README.md) | 実装に結び付いた準拠条件の確認 | 特定の実装版・検査範囲について、コードと検査への対応を説明 |
| `docs/audits/` | 特定時点の監査結果を保管する | 監査記録。全体構成の正本ではない |

`architecture-reading-guide.md`、`business-system-design-reading-guide.md`、`design-synthesis-reading-guide.md`、`requirements/README.md`は正本を置き換えません。背景、失敗例、要求同士のつながりを知るための案内です。

設計判断と採用前の設計案は[`plan/`](../plan/README.md)で管理します。文書に英語の概念名やコード上の識別子が必要な場合も、説明文は日本語で書きます。

## おすすめの読み順

### 1. 初めてTEIを読む

1. [はじめてTEIを知る方へ](for-readers.md)
2. [TEIの考え方](overview.md)
3. [現在の実装状況](status.md)

まず、TEIが「意味・実装・実行・観測」を分ける理由を確認します。型の詳細や判定段階の手順は、その後に読み進めます。

### 2. 全体構成を読む

1. [TEIの考え方](overview.md)
2. [全体構成を読む案内](architecture-reading-guide.md)
3. [全体構成](architecture.md)
4. [業務システム設計 読み方ガイド](business-system-design-reading-guide.md)
5. [業務システム設計](business-system-design.md)
6. 必要な設計判断記録、準拠説明、参照実装

`architecture-reading-guide.md`で、技術上の境界がどのような失敗を防ぐかを把握してから、`architecture.md`の詳細へ進みます。業務主体、手作業、引継ぎ、実際の状態などが難しい場合は、`business-system-design-reading-guide.md`で一つの業務例を追ってから正本を読みます。

`architecture.md`は主に技術上の境界を、`business-system-design.md`は主に業務設計上の境界を扱います。二つの文書は、同じTEIの異なる側面を説明します。

### 3. 要求から実証まで追う

1. [製品要求の読み方](../requirements/README.md)
2. [TEIの製品要求](../requirements/TEI_PRODUCT_REQUIREMENTS.md)
3. [業務システム設計 要求解釈基線](../requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md)
4. [上位の受け入れシナリオとの追跡関係](../plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md)
5. [TEIの計画](../plan/README.md)
6. 対応する計画上の課題、参照実装、検証根拠

要求だけから実装を直接導かず、受け入れシナリオ、設計判断、参照実装、検証根拠を通して、成立が確認できた範囲を確かめます。

### 4. AIによる認知支援と設計案の統合を読む

1. [TEIの考え方](overview.md)で、TEI全体におけるAIの位置を確認します。
2. [設計案の統合と認知支援の読み方](design-synthesis-reading-guide.md)で、製品の目標と認知支援の考え方を確認します。
3. 実装済みの範囲と未検証の範囲は[現在の実装状況](status.md)で確かめます。

大規模言語モデルを使わない認知機能やモデル生成の検討項目は、実装が確定した内容ではありません。正式な要求は[製品要求](../requirements/TEI_PRODUCT_REQUIREMENTS.md)に従います。

### 5. 現在の判定段階を確認する

1. [現在の実装状況](status.md)
2. [TEIの計画](../plan/README.md)
3. 対応する準備状況の判定記録、検証根拠の条件、実行手順

`status.md`は現在の状況を知るための入口です。判定そのものの正本は`plan/`にあります。

### 6. 文書を書く・レビューする

1. [TEI文書品質ガイド](documentation-quality.md)
2. [`AGENTS.md`](../AGENTS.md)
3. 対象文書の上位にある正本、参照実装、検証根拠

TEIは広い技術領域と業務知識を扱います。実装者だけが読める文書では足りません。背景、前提、用語、失敗例、判断理由を説明し、主担当者ほど詳しくない読者も目的と設計上の境界を理解して確認に参加できる状態を目指します。

## 設計文書を読むときの区別

### 「設計として決めた」と「実装で確認した」を分ける

構成や要求に境界が書かれていても、それだけで実際の製品を検証した根拠を取得済みとは扱いません。

```text
構成や要求の記述
  ≠ 実装済みであること
  ≠ 個別の試験に合格したこと
  ≠ 外部環境での検証根拠があること
  ≠ 製品に関する主張を満たしたこと
```

参照実装が動いたことだけで、その実装固有の形を正本のスキーマへ取り込みません。

### 「現在地」と「長く残す判断」を分ける

頻繁に変わる実装状況や判定段階は`status.md`や準備状況の判定記録で追います。

長く残す構成上の判断は、全体構成の文書や設計判断記録に置きます。過去の作業経緯は現在の基準文書へ積み上げず、Gitの履歴、完了した課題、監査記録に分けます。

### 「説明」と「正本」を分ける

`for-readers.md`、`overview.md`、各案内文書、`requirements/README.md`は理解のための説明です。読みやすく言い換えても、製品要求や構成上の境界の意味を独自に増減させません。

## 設計文書の基本体裁

新しい長文の設計・分析文書を追加するときは、内容に応じて冒頭で次を明らかにします。

1. 役割。全体構成、要求の解釈、計画、監査、実験などのどれか。
2. 正本との関係。どの情報の正本で、どの情報を扱わないか。
3. 対象範囲。何を扱い、何を扱わないか。
4. 現在の結論と状態。長い経緯より先に、読者が現在の判断を確認できること。
5. 根拠。参照実装、準拠確認、検証根拠、関連する設計判断記録と課題記録。
6. 結論を出していない範囲。試験への合格や参照実装の成立から、何を一般化できないか。

さらに、内容が難しい文書ほど、「なぜ必要か」「どんな失敗を防ぐか」「具体例ではどう見えるか」を先に説明します。詳しい執筆方針は[TEI文書品質ガイド](documentation-quality.md)を参照してください。

すべての文書へ同じ固定テンプレートを強制するものではありません。文書の種類に応じて必要な項目だけを使い、形式を守ること自体を目的にしません。
