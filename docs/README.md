# TEI ドキュメント案内

この文書は、TEIの設計、実装、開発状況を知るための案内です。製品要求、設計判断、受入条件、検証結果はここで新たに定義せず、それぞれの正本を案内します。

全体像は[TEIの考え方](overview.md)、技術上の境界は[全体構成](architecture.md)、業務設計の境界は[業務システム設計](business-system-design.md)で説明しています。正式な要求は[製品要求](../requirements/TEI_PRODUCT_REQUIREMENTS.md)、開発の主な計画は[TEIの全体計画](../plan/MASTER_PLAN.md)を参照してください。

最初からすべての文書を読む必要はありません。知りたい内容に合わせて、読む文書を選んでください。リポジトリ全体の案内は[README](../README.md)にあります。

## 知りたいことから探す

| 知りたいこと | 最初に読む文書 | 次に読む文書 |
| --- | --- | --- |
| TEIが何を目指すか | [はじめてTEIを知る方へ](for-readers.md) | [TEIの考え方](overview.md) |
| 技術面の全体像 | [全体構成の読み方](architecture-reading-guide.md) | [全体構成](architecture.md) |
| 実装と検証の状況 | [全体構成](architecture.md) | [準拠確認と参照用資料](../reference/README.md) |
| 業務システム設計上の境界 | [業務システム設計の読み方](business-system-design-reading-guide.md) | [業務システム設計](business-system-design.md) |
| AIによる認知支援と設計案のまとめ方 | [設計案の統合と認知支援の読み方](design-synthesis-reading-guide.md) | [TEIの考え方](overview.md) |
| 製品の正式な要求 | [製品要求の読み方](../requirements/README.md) | [TEI製品要求](../requirements/TEI_PRODUCT_REQUIREMENTS.md) |
| 要求と検証根拠のつながり | [上位の受け入れシナリオとの追跡関係](../plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md) | [TEIの計画](../plan/README.md) |
| 現在どこまで動くか | [現在の実装状況](status.md) | [TEIの計画](../plan/README.md) |
| 実際に動かす | [TEIを動かす](getting-started.md) | [現在の実装状況](status.md) |
| 安全性と権限の境界 | [安全性](security.md) | [全体構成の読み方](architecture-reading-guide.md) |
| 用語と命名 | [TEIの名称と共通用語](naming.md) | [TEIの考え方](overview.md) |
| 文書の執筆と確認 | [TEI文書品質ガイド](documentation-quality.md) | [`AGENTS.md`](../AGENTS.md) |

## 文書の役割

TEIでは、説明、要求、設計判断、計画、検証根拠を一つの長い文書にまとめず、それぞれの役割に応じて管理します。

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

`docs/`には、利用者や運用者が現在のTEIを理解して使うための説明と、実装に対応する構成・準拠確認を置きます。採用前の案、複数の作業にまたがる設計、実施計画、受入条件は`plan/`で管理します。

| 文書群 | 主な役割 | 正本性 |
| --- | --- | --- |
| `product/` | 何を価値として守るか | 価値、将来像、原則の正本 |
| `requirements/` | 何を満たすか | 製品要求の正本 |
| `plan/` | 受け入れシナリオ、判定段階、設計判断、課題、機能設計、検証根拠の取得計画 | 計画、決定、判定段階、機能設計それぞれの正本 |
| [`tools/README.md`](../tools/README.md) | 開発用ツールの一覧 | 実行方法と用途の入口 |
| `docs/architecture.md` | 実装構造と技術上の境界を説明する | 現在の構成を説明する基準。要求の第二の正本ではない |
| `docs/business-system-design.md` | データ、機能、業務をまたぐ設計上の境界を説明する | 業務システム設計の現在の基準文書 |
| `docs/status.md` | 現在の実装と検証根拠の状態を示す | 更新される現状記録 |
| `reference/` / `tools/` | 設計上の仮説を実例と確認用のツールで検証する | 参照実装と実行可能な検証根拠 |
| [`docs/conformance/README.md`](conformance/README.md) | 実装に結び付いた準拠条件の確認 | 特定の実装版・検査範囲について、コードと検査への対応を説明 |
| `docs/audits/` | 特定時点の監査結果を保管する | 監査記録。全体構成の正本ではない |

`architecture-reading-guide.md`、`business-system-design-reading-guide.md`、`design-synthesis-reading-guide.md`、`requirements/README.md`は、正本の代わりとなるものではありません。背景や失敗例、要求同士のつながりを理解するための案内です。

設計判断と採用前の設計案は[`plan/`](../plan/README.md)で管理します。概念の原語やコード上の識別子を示す場合も、本文の説明には自然な日本語を使います。

## おすすめの読み順

### 1. 初めてTEIを読む

1. [はじめてTEIを知る方へ](for-readers.md)
2. [TEIの考え方](overview.md)
3. [現在の実装状況](status.md)

最初に、TEIが意味・実装・実行・観測を分ける理由を確かめます。型の詳細や判定段階の手順は、その後で読んでください。

### 2. 全体構成を読む

1. [TEIの考え方](overview.md)
2. [全体構成の読み方](architecture-reading-guide.md)
3. [全体構成](architecture.md)
4. [業務システム設計の読み方](business-system-design-reading-guide.md)
5. [業務システム設計](business-system-design.md)
6. 必要な設計判断記録、準拠説明、参照実装

まず`architecture-reading-guide.md`で、技術上の境界が防ごうとする問題を理解してから、`architecture.md`の詳細へ進みます。業務を担う主体、手作業、引き継ぎ、実際の状態などが分かりにくい場合は、`business-system-design-reading-guide.md`で具体的な業務例を確認してから正本を読んでください。

`architecture.md`は主に技術上の境界を、`business-system-design.md`は主に業務設計上の境界を扱います。同じTEIを異なる視点から説明する文書です。

### 3. 要求から実証まで追う

1. [製品要求の読み方](../requirements/README.md)
2. [TEIの製品要求](../requirements/TEI_PRODUCT_REQUIREMENTS.md)
3. [業務システム設計 要求解釈基線](../requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md)
4. [上位の受け入れシナリオとの追跡関係](../plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md)
5. [TEIの計画](../plan/README.md)
6. 対応する計画上の課題、参照実装、検証根拠

要求だけを読んで実装が成立したとは判断せず、受け入れシナリオ、設計判断、参照実装、検証根拠をたどって、何が確認できたかを確かめます。

### 4. AIによる認知支援と設計案の統合を読む

1. [TEIの考え方](overview.md)で、TEI全体におけるAIの位置を確認します。
2. [設計案の統合と認知支援の読み方](design-synthesis-reading-guide.md)で、製品の目標と認知支援の考え方を確認します。
3. 実装済みの範囲と未検証の範囲は[現在の実装状況](status.md)で確かめます。

大規模言語モデルを使わない認知支援やモデル生成の検討事項は、実装が確定した内容ではありません。正式な要求は[製品要求](../requirements/TEI_PRODUCT_REQUIREMENTS.md)を参照してください。

### 5. 現在の判定段階を確認する

1. [現在の実装状況](status.md)
2. [TEIの計画](../plan/README.md)
3. 対応する準備状況の判定記録、検証根拠の条件、実行手順

`status.md`は現在の状況を確認するための入口です。判定の正式な記録は`plan/`にあります。

### 6. 文書を書く・レビューする

1. [TEI文書品質ガイド](documentation-quality.md)
2. [`AGENTS.md`](../AGENTS.md)
3. 対象文書の上位にある正本、参照実装、検証根拠

TEIは広い技術領域と業務知識を扱うため、実装担当者だけが理解できる文書では不十分です。背景、前提、用語、失敗例、判断理由を説明し、担当者ほど詳しくない読者も目的と設計上の境界を理解し、確認に参加できる状態を目指します。

## 設計文書を読むときの区別

### 「設計として決めた」と「実装で確認した」を分ける

構成や要求に境界が記されているだけでは、実際の製品で検証根拠を得たことにはなりません。

```text
構成や要求の記述
  ≠ 実装済みであること
  ≠ 個別の試験に合格したこと
  ≠ 外部環境での検証根拠があること
  ≠ 製品に関する主張を満たしたこと
```

参照実装が動いたという理由だけで、その実装に固有の構造を正本のスキーマへ取り込みません。

### 「現在地」と「長く残す判断」を分ける

変化する実装状況や判定段階は、`status.md`と準備状況の判定記録で確認します。

長く残す構成上の判断は、全体構成の文書や設計判断記録に置きます。過去の作業経緯を現在の基準文書に積み上げず、Gitの履歴、完了した課題、監査記録に残します。

### 「説明」と「正本」を分ける

`for-readers.md`、`overview.md`、各案内文書、`requirements/README.md`は理解のための説明です。読みやすく言い換える場合も、製品要求や構成上の境界を独自に追加・変更しません。

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
